package main

import (
	"encoding/hex"
	"math"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

const float64EqualityThreshold = 0.00001

func tExporter(t *testing.T) *SolisExporter {
	e, err := NewSolisExporter(&SolisExporterConfig{}, nil)
	if err != nil {
		t.Fatalf("Failed to create exporter: %v", err)
	}
	return e
}

func tPrepExchange(t *testing.T, reqs, reps string) *ModbusExchange {
	var n int
	m := &ModbusExchange{}

	req, err := hex.DecodeString(reqs)
	if err != nil {
		t.Fatalf("Invalid request: %v: %v", reqs, err)
	}
	rep, err := hex.DecodeString(reps)
	if err != nil {
		t.Fatalf("Invalid request: %v: %v", reps, err)
	}

	n = m.ParseRequest(req)
	if n != 0 || m.Error != nil {
		t.Fatalf("Unable to parse request: %d %v", n, m.Error)
	}
	n = m.ParseResponse(rep)
	if n != 0 || m.Error != nil {
		//t.Logf("CRC:%02X", ModbusCRC(rep))
		t.Fatalf("Unable to parse response: %d %v", n, m.Error)
	}
	return m
}

func tTestGauges(t *testing.T, e *SolisExporter, gauges map[uint16]float64) {
	for reg, exp := range gauges {
		v := testutil.ToFloat64(e.metrics[reg].(*handlerGauge).g)
		if math.Abs(v-exp) > float64EqualityThreshold*(math.Abs(v)+math.Abs(exp)) {
			t.Errorf("Metrid %d: got value %f, expected %f", reg, v, exp)
		}
	}
}

// Meter placement and other meter (grid) data
func TestExporter33250(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "010481E20025B9DB", "01044A0002095D00EA0000000000000000FFFFFE620000000000000000FFFFFE6200000121000000000000000000000121000001F90000000000000000000001F9FFAF1387000032880001BAD409A5")
	e.handleMessage(m)
	tTestGauges(t, e, map[uint16]float64{
		33263: -414,
	})
}

func TestExporter33000(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "010480E800299820", "01045231050032003C00013630333130353939393939393939393900000000000000000000000000000000000000000016000B000D001300240020000000000CF5000000460000016D0029001B00000CF500000000CC5F")
	e.handleMessage(m)
}

func TestExporter33049(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "01048119002409EA", "010448000E0001000F00000000000000000000000000000000000000000000000000000000000000000000003100000F3A0000095800000000001400000000FFFFFF9C0009FFF60000000A7C94")
	e.handleMessage(m)
}

func TestExporter33091(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "010481430005E9E1", "01040A0000003500EF13880003A506")
	e.handleMessage(m)
}

func TestExporter33100(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "0104814C0016982F", "01042C00000000000000002AF803E80000000000000000000000000000000000000002000000000000000000000701F38E")
	e.handleMessage(m)
}

func TestExporter33126(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "0104816600183823", "01043000134598095D00EEFFFFFE55002301EA001C00010D2209580014001400631303000602E402E400000000012E00000000B2F6")
	e.handleMessage(m)
}

func TestExporter33161(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "01048189001409D3", "010428000003E0001C00150000047A0022003300000081003000120000046D0000000000000986005E004BC806")
	e.handleMessage(m)
}

func TestExporter33243(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "010481DB0004A9CE", "0104080000000000000000240D")
	e.handleMessage(m)
}

// These are poll messages which are received and won't update the exporter,
// just test that they don't crash anything
func TestExporterPoll2900(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "01040BB7000183C8", "018402C2C1")
	if m.Exception != 2 {
		t.Fatalf("Should have got exception")
	}
	e.handleMessage(m)
}

func TestExporterPoll33000(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "010480E80001983E", "01040231056CA3")
	e.handleMessage(m)
}

func TestExporterPoll33250(t *testing.T) {
	e := tExporter(t)
	m := tPrepExchange(t, "0103A8010001F5AA", "01030200017984")
	e.handleMessage(m)
}
