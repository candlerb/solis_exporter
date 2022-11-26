package main

// Restrict the range of registers which can be accessed,
// and optionally the function codes
type Rule struct {
	From      uint16  `yaml:"from"`
	To        uint16  `yaml:"to"`
	Functions []uint8 `yaml:"functions"`
	Stations  []uint8 `yaml:"station"`
}

var DEFAULT_ALLOW_STATIONS = []uint8{1}
var DEFAULT_ALLOW_FUNCTIONS = []uint8{1, 2, 3, 4}

func findUint8(s []uint8, v uint8) bool {
	for _, item := range s {
		if v == item {
			return true
		}
	}
	return false
}

func CheckRules(m *ModbusExchange, rules []Rule) bool {
	a1 := m.Base
	a2 := m.Base + m.Count - 1
	for _, rule := range rules {
		stations := rule.Stations
		if len(stations) == 0 {
			stations = DEFAULT_ALLOW_STATIONS
		}
		if !findUint8(stations, m.Station) {
			continue
		}
		lower := rule.From
		upper := rule.To
		if upper == 0 {
			upper = lower
		}
		if a1 < lower || a1 > upper || a2 < lower || a2 > upper {
			continue
		}
		fns := rule.Functions
		if len(fns) == 0 {
			fns = DEFAULT_ALLOW_FUNCTIONS
		}
		if !findUint8(fns, m.Function) {
			continue
		}
		// All conditions matched
		return true
	}
	return false
}
