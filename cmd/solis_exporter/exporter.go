package main

/*
See:
https://prometheus.io/docs/guides/go-application/
https://github.com/prometheus/client_golang
https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#hdr-A_Basic_Example
*/

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type SolisExporterConfig struct {
	Listen           string `yaml:"listen"`
	Station          byte
	GoCollector      bool `yaml:"go_collector"`
	ProcessCollector bool `yaml:"process_collector"`
}

// This interface covers handlerGauge and handlerGaugeVec
type ModbusMetricHandler interface {
	Process([]byte)                    // process slice of data
	SetRegistry(prometheus.Registerer) // where to register this handler's collector(s)
}

// A handler which updates a Gauge
type handlerGauge struct {
	g prometheus.Gauge
	f func(prometheus.Gauge, []byte)
	r prometheus.Registerer
}

func (h *handlerGauge) Process(data []byte) {
	// Register gauges on demand, so we don't get spurious zero values
	if h.r != nil {
		h.r.MustRegister(h.g)
		h.r = nil
	}
	h.f(h.g, data)
}

func (h *handlerGauge) SetRegistry(r prometheus.Registerer) {
	h.r = r
}

// A handler which updates a GaugeVec
type handlerGaugeVec struct {
	gv *prometheus.GaugeVec
	f  func(*prometheus.GaugeVec, []byte)
}

func (h *handlerGaugeVec) Process(data []byte) {
	h.f(h.gv, data)
}

func (h *handlerGaugeVec) SetRegistry(r prometheus.Registerer) {
	// Register GaugeVec up-front.  However, multiple modbus
	// metrics may share the same GaugeVec.
	err := r.Register(h.gv)
	if err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			panic(err)
		}
	}
}

// Helper functions for simple gauges
type gaugeFunc func(prometheus.Gauge, []byte)

func scaledGaugeU16(scale float64) gaugeFunc {
	return func(g prometheus.Gauge, data []byte) {
		val := binary.BigEndian.Uint16(data)
		g.Set(float64(val) * scale)
	}
}

func scaledGaugeS16(scale float64) gaugeFunc {
	return func(g prometheus.Gauge, data []byte) {
		val := int16(binary.BigEndian.Uint16(data))
		g.Set(float64(val) * scale)
	}
}

func scaledGaugeU32(scale float64) gaugeFunc {
	return func(g prometheus.Gauge, data []byte) {
		if len(data) >= 4 {
			val := binary.BigEndian.Uint32(data)
			g.Set(float64(val) * scale)
		}
	}
}

func scaledGaugeS32(scale float64) gaugeFunc {
	return func(g prometheus.Gauge, data []byte) {
		if len(data) >= 4 {
			val := int32(binary.BigEndian.Uint32(data))
			g.Set(float64(val) * scale)
		}
	}
}

var gaugeU16 = scaledGaugeU16(1.0)
var gaugeS16 = scaledGaugeS16(1.0)
var gaugeU32 = scaledGaugeU32(1.0)
var gaugeS32 = scaledGaugeS32(1.0)

// Helper functions for gauge vectors
type gaugeVecFunc func(*prometheus.GaugeVec, []byte)

func scaledGaugeVecU16(scale float64, labelValues ...string) gaugeVecFunc {
	return func(g *prometheus.GaugeVec, data []byte) {
		val := binary.BigEndian.Uint16(data)
		g.WithLabelValues(labelValues...).Set(float64(val) * scale)
	}
}

func scaledGaugeVecS16(scale float64, labelValues ...string) gaugeVecFunc {
	return func(g *prometheus.GaugeVec, data []byte) {
		val := int16(binary.BigEndian.Uint16(data))
		g.WithLabelValues(labelValues...).Set(float64(val) * scale)
	}
}

func scaledGaugeVecU32(scale float64, labelValues ...string) gaugeVecFunc {
	return func(g *prometheus.GaugeVec, data []byte) {
		if len(data) >= 4 {
			val := binary.BigEndian.Uint32(data)
			g.WithLabelValues(labelValues...).Set(float64(val) * scale)
		}
	}
}

func scaledGaugeVecS32(scale float64, labelValues ...string) gaugeVecFunc {
	return func(g *prometheus.GaugeVec, data []byte) {
		if len(data) >= 4 {
			val := int32(binary.BigEndian.Uint32(data))
			g.WithLabelValues(labelValues...).Set(float64(val) * scale)
		}
	}
}

var gaugeVecU16 = scaledGaugeVecU16(1.0)
var gaugeVecS16 = scaledGaugeVecS16(1.0)
var gaugeVecU32 = scaledGaugeVecU32(1.0)
var gaugeVecS32 = scaledGaugeVecS32(1.0)

// The overall exporter instance
type SolisExporter struct {
	config      *SolisExporterConfig
	modbus      <-chan *ModbusExchange
	reg         *prometheus.Registry
	metrics     map[uint16]ModbusMetricHandler
	messages    *prometheus.CounterVec
	errors      *prometheus.CounterVec
	lastMessage prometheus.Gauge
}

func NewSolisExporter(config *SolisExporterConfig, modbus <-chan *ModbusExchange) (*SolisExporter, error) {
	if config.Listen == "" {
		config.Listen = ":3105"
	}
	if config.Station == 0 {
		config.Station = 1
	}
	e := &SolisExporter{
		config:  config,
		modbus:  modbus,
		reg:     prometheus.NewRegistry(),
		metrics: make(map[uint16]ModbusMetricHandler),
		messages: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "solis_serial_messages_total",
				Help: "Number of packet exchanges",
			},
			[]string{"source"}),
		errors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "solis_serial_errors_total",
				Help: "Serial bus transmission or reception errors",
			},
			[]string{"error"}),
		lastMessage: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_serial_last_message_time_seconds",
			Help: "Time when last message received, in unixtime",
		}),
	}
	e.reg.MustRegister(e.messages)
	e.reg.MustRegister(e.errors)
	e.reg.MustRegister(e.lastMessage)
	// Instantiate the counters to zero
	for _, label := range []string{"sniffed", "injected"} {
		e.messages.WithLabelValues(label)
	}
	for _, label := range modbusErrorToLabel {
		e.errors.WithLabelValues(label)
	}

	// Register system metrics
	e.reg.MustRegister(collectors.NewBuildInfoCollector())
	if e.config.GoCollector {
		e.reg.MustRegister(collectors.NewGoCollector())
	}
	if e.config.ProcessCollector {
		e.reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	// Register inverter metrics parsed from modbus messages
	e.addSolisMetrics()
	return e, nil
}

func (e *SolisExporter) addHandler(regbase uint16, handler ModbusMetricHandler) {
	if _, ok := e.metrics[regbase]; ok {
		log.Fatalf("Duplicate metric registration: %d", regbase)
	}
	e.metrics[regbase] = handler
	handler.SetRegistry(e.reg)
}

func (e *SolisExporter) addSolisMetrics() {
	// Read register 33000-33040: Product information and total power generation
	e.addHandler(33000, &handlerGaugeVec{
		gv: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "solis_inverter_info",
				Help: "Static information about the inverter",
			},
			[]string{"model", "dsp_version", "lcd_version", "protocol_version", "serial"}),
		f: func(gv *prometheus.GaugeVec, data []byte) {
			if len(data) >= 40 {
				gv.Reset()
				gv.WithLabelValues(
					fmt.Sprintf("%04X", data[0:2]),
					fmt.Sprintf("%04X", data[2:4]),
					fmt.Sprintf("%04X", data[4:6]),
					fmt.Sprintf("%04X", data[6:8]),
					string(bytes.TrimRight(data[8:40], "\x00")),
				).Set(1)
			}
		},
	})

	inverter_energy := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_energy",
			Help: "Inverter total power generation and use",
		},
		[]string{"type", "period"})
	e.addHandler(33029, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU32(1, "yield", "all"),
	})
	e.addHandler(33031, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU32(1, "yield", "month"),
	})
	e.addHandler(33033, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU32(1, "yield", "month-1"),
	})
	e.addHandler(33035, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU16(0.1, "yield", "day"),
	})
	e.addHandler(33036, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU16(0.1, "yield", "day-1"),
	})
	e.addHandler(33037, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU32(1, "yield", "year"),
	})
	e.addHandler(33039, &handlerGaugeVec{
		gv: inverter_energy,
		f:  scaledGaugeVecU32(1, "yield", "year-1"),
	})

	// Read register 33049-33084: Inverter voltage and current data
	dc_voltage := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_dc_voltage",
			Help: "PV array DC voltage",
		},
		[]string{"pv"})
	dc_current := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_dc_current",
			Help: "PV array DC current",
		},
		[]string{"pv"})
	for pv := uint16(0); pv < 2; pv++ {
		pv_label := fmt.Sprintf("%d", pv+1)
		e.addHandler(33049+pv*2, &handlerGaugeVec{
			gv: dc_voltage,
			f:  scaledGaugeVecU16(0.1, pv_label),
		})
		e.addHandler(33050+pv*2, &handlerGaugeVec{
			gv: dc_current,
			f:  scaledGaugeVecU16(0.1, pv_label),
		})
	}

	e.addHandler(33057, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_dc_power",
			Help: "Total DC output power (W)",
		}),
		f: gaugeU32,
	})

	inverter_ac_voltage := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_ac_voltage",
			Help: "Inverter AC voltage",
		},
		[]string{"phase"})
	inverter_ac_current := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_ac_current",
			Help: "Inverter AC current",
		},
		[]string{"phase"})
	for i, phase := range []string{"U", "V", "W"} {
		e.addHandler(33073+uint16(i), &handlerGaugeVec{
			gv: inverter_ac_voltage,
			f:  scaledGaugeVecU16(0.1, phase),
		})
		e.addHandler(33076+uint16(i), &handlerGaugeVec{
			gv: inverter_ac_current,
			f:  scaledGaugeVecU16(0.1, phase),
		})
	}
	e.addHandler(33079, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_power_active",
			Help: "Inverter total active power (W)",
		}),
		f: gaugeS32,
	})
	e.addHandler(33081, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_power_reactive",
			Help: "Inverter total reactive power (Var)",
		}),
		f: gaugeS32,
	})
	e.addHandler(33083, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_power_apparent",
			Help: "Inverter total apparent power (VA)",
		}),
		f: gaugeS32,
	})

	// Read register 33091-33095: Working mode and temperature
	e.addHandler(33093, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_temperature",
			Help: "Inverter temperature - °C",
		}),
		f: scaledGaugeS16(0.1),
	})
	e.addHandler(33094, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_frequency",
			Help: "Inverter output frequency",
		}),
		f: scaledGaugeU16(0.01),
	})
	e.addHandler(33095, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_operating_state",
			Help: "Inverter operating state, register 33095",
		}),
		f: gaugeU16,
	})

	// Read register 33100-33121: Power and fault information
	fault := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_inverter_fault_flags",
			Help: "Fault flags, register 33116-33120",
		},
		[]string{"code"})
	for i, code := range []string{"01", "02", "03", "04", "05"} {
		e.addHandler(33116+uint16(i), &handlerGaugeVec{
			gv: fault,
			f:  scaledGaugeVecU16(1.0, code),
		})
	}
	e.addHandler(33121, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_working_status_flags",
			Help: "Working status bits, register 33121",
		}),
		f: gaugeU16,
	})

	// Read register 33126-33149: Power and battery state
	e.addHandler(33132, &handlerGauge{
		// Reflects the storage mode written to 43110
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_storage_control_flags",
			Help: "Energy storage control mode, register 33132",
		}),
		f: gaugeU16,
	})
	e.addHandler(33133, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_battery_voltage",
			Help: "Battery voltage",
		}),
		f: scaledGaugeU16(0.1),
	})
	e.addHandler(33134, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_battery_current",
			Help: "Battery current (+ = charging, - = discharging)",
		}),
		f: func(g prometheus.Gauge, data []byte) {
			if len(data) >= 4 {
				val := float64(binary.BigEndian.Uint16(data)) * 0.1
				// 0=charging, 1=discharging
				// Choose the polarity to match Solis Cloud graphs
				dir := binary.BigEndian.Uint16(data[2:])
				if dir == 1 {
					val = -val
				}
				g.Set(val)
			}
		},
	})
	e.addHandler(33137, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_backup_voltage",
			Help: "Backup output voltage",
		}),
		f: scaledGaugeU16(0.1),
	})
	e.addHandler(33138, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_backup_current",
			Help: "Backup output current",
		}),
		f: scaledGaugeU16(0.01), // Documentation appears to have wrong scale factor
	})
	e.addHandler(33139, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_battery_soc",
			Help: "Battery state of charge - percent",
		}),
		f: gaugeU16,
	})
	e.addHandler(33140, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_battery_soh",
			Help: "Battery state of health - percent",
		}),
		f: gaugeU16,
	})
	e.addHandler(33141, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_bms_battery_voltage",
			Help: "BMS Battery Voltage",
		}),
		f: scaledGaugeU16(0.01),
	})
	e.addHandler(33142, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_bms_battery_current",
			Help: "BMS Battery Current",
		}),
		f: scaledGaugeS16(0.1), // documented scale factor is wrong.  Also never goes negative?
	})
	e.addHandler(33143, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_bms_charge_limit_current",
			Help: "BMS Battery Charge Limit - Amps",
		}),
		f: scaledGaugeU16(0.1),
	})
	e.addHandler(33144, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_bms_discharge_limit_current",
			Help: "BMS Battery Discharge Limit - Amps",
		}),
		f: scaledGaugeU16(0.1),
	})
	battery_failure := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_bms_failure_flags",
			Help: "BMS battery failure information, register 33145-33146",
		},
		[]string{"code"})
	for i, code := range []string{"01", "02"} {
		e.addHandler(33145+uint16(i), &handlerGaugeVec{
			gv: battery_failure,
			f:  scaledGaugeVecU16(1.0, code),
		})
	}
	e.addHandler(33147, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_load_power",
			Help: "House load power (W)",
		}),
		f: gaugeU16,
	})
	e.addHandler(33148, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_inverter_backup_power",
			Help: "Backup load power (W)",
		}),
		f: gaugeU16,
	})
	// Doc says 33149 (battery power) is S32, but logger reads only 16 bits!

	// Read register 33161-33180: Battery charge and grid power totals
	// Note that grid import/export are lower resolution than 33283/33285
	// but do provide daily figures
	for _, item := range []struct {
		Base uint16
		Type string
	}{
		{33161, "charge"},
		{33165, "discharge"},
		{33169, "import"},
		{33173, "export"},
		{33177, "load"},
	} {
		e.addHandler(item.Base, &handlerGaugeVec{
			gv: inverter_energy,
			f:  scaledGaugeVecU32(1, item.Type, "all"),
		})
		e.addHandler(item.Base+2, &handlerGaugeVec{
			gv: inverter_energy,
			f:  scaledGaugeVecU16(0.1, item.Type, "day"),
		})
		e.addHandler(item.Base+3, &handlerGaugeVec{
			gv: inverter_energy,
			f:  scaledGaugeVecU16(0.1, item.Type, "day-1"),
		})
	}

	// Read register 33250-33286: Meter (grid) data
	meter_ac_voltage := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_grid_voltage",
			Help: "Grid AC voltage",
		},
		[]string{"phase"})
	meter_ac_current := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_grid_current",
			Help: "Grid AC current",
		},
		[]string{"phase"})
	for i, phase := range []string{"U", "V", "W"} {
		e.addHandler(33251+uint16(i)*2, &handlerGaugeVec{
			gv: meter_ac_voltage,
			f:  scaledGaugeVecU16(0.1, phase),
		})
		e.addHandler(33252+uint16(i)*2, &handlerGaugeVec{
			gv: meter_ac_current,
			f:  scaledGaugeVecU16(0.01, phase),
		})
	}
	e.addHandler(33263, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_grid_power_active",
			Help: "Grid total active power (W)",
		}),
		f: gaugeS32,
	})
	e.addHandler(33271, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_grid_power_reactive",
			Help: "Grid total reactive power (Var)",
		}),
		f: gaugeS32,
	})
	e.addHandler(33279, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_grid_power_apparent",
			Help: "Grid total apparent power (VA)",
		}),
		f: gaugeS32,
	})
	/* Not sure how to convert value of this one
	e.addHandler(33281, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_grid_power_factor",
			Help: "Grid power factor",
		}),
		f: scaledGaugeS16(0.01),
	})
	*/
	e.addHandler(33282, &handlerGauge{
		g: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "solis_grid_frequency",
			Help: "Grid frequency",
		}),
		f: scaledGaugeU16(0.01),
	})
	grid_energy := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "solis_grid_energy",
			Help: "Grid meter total power import and export",
		},
		[]string{"type"})
	e.addHandler(33283, &handlerGaugeVec{
		gv: grid_energy,
		f:  scaledGaugeVecU32(0.01, "import"),
	})
	e.addHandler(33285, &handlerGaugeVec{
		gv: grid_energy,
		f:  scaledGaugeVecU32(0.01, "export"),
	})
}

func (e *SolisExporter) handleMessage(m *ModbusExchange) {
	if m.Sniffed {
		e.messages.WithLabelValues("sniffed").Inc()
	} else {
		e.messages.WithLabelValues("injected").Inc()
	}
	if m.Error != nil {
		label := modbusErrorToLabel[m.Error]
		if label == "" {
			label = modbusErrorToLabel[ERR_INVALID]
		}
		e.errors.With(prometheus.Labels{"error": label}).Inc()
		return
	}
	e.lastMessage.SetToCurrentTime()
	if m.Exception != 0 {
		return
	}
	if m.Station != e.config.Station {
		return
	}

	switch m.Function {
	case 3, 4: // multi-register read: 'Count' is the number of (2-byte) registers in 'Data'
		limit := m.Base + m.Count
		for r := m.Base; r < limit; r++ {
			if handler, ok := e.metrics[r]; ok {
				p1 := (r - m.Base) * 2
				// sanity check: at least 2 bytes
				if p1 >= 0 && int(p1) < len(m.Data)-1 {
					handler.Process(m.Data[p1:])
				}
			}
		}
	}
}

func (e *SolisExporter) Run() {
	go func() {
		for m := range e.modbus {
			e.handleMessage(m)
		}
	}()

	http.Handle("/metrics", promhttp.HandlerFor(e.reg, promhttp.HandlerOpts{Registry: e.reg}))
	log.Printf("Starting metrics listener on %s", e.config.Listen)
	log.Fatal(http.ListenAndServe(e.config.Listen, nil))
}
