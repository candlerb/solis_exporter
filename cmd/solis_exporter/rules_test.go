package main

import (
	"testing"
)

var testRules = []Rule{
	{From: 30001, To: 39999, Functions: []uint8{3, 4}},
	{From: 43110, Functions: []uint8{6}}, // implied: upper bound 43110
	{From: 43143, To: 43150, Functions: []uint8{16}},
}

func TestRules(t *testing.T) {
	type testRuleCase struct {
		m  *ModbusExchange
		ok bool
	}
	var testRuleCases = []testRuleCase{
		{&ModbusExchange{Base: 1234, Count: 1, Function: 4}, false},
		{&ModbusExchange{Base: 30001, Count: 1, Function: 2}, false},
		{&ModbusExchange{Base: 30001, Count: 1, Function: 3}, true},
		{&ModbusExchange{Base: 30001, Count: 1, Function: 4}, true},
		{&ModbusExchange{Base: 30001, Count: 1, Function: 5}, false},
		{&ModbusExchange{Base: 43109, Count: 1, Function: 6}, false},
		{&ModbusExchange{Base: 43110, Count: 1, Function: 6}, true},
		{&ModbusExchange{Base: 43110, Count: 1, Function: 16}, false},
		{&ModbusExchange{Base: 43111, Count: 1, Function: 6}, false},
		{&ModbusExchange{Base: 43142, Count: 2, Function: 16}, false},
		{&ModbusExchange{Base: 43143, Count: 1, Function: 6}, false},
		{&ModbusExchange{Base: 43143, Count: 1, Function: 16}, true},
		{&ModbusExchange{Base: 43143, Count: 8, Function: 16}, true},
		{&ModbusExchange{Base: 43143, Count: 9, Function: 16}, false},
		{&ModbusExchange{Base: 43147, Count: 4, Function: 16}, true},
		{&ModbusExchange{Base: 43147, Count: 5, Function: 16}, false},
		{&ModbusExchange{Base: 43150, Count: 1, Function: 16}, true},
		{&ModbusExchange{Base: 43150, Count: 2, Function: 16}, false},
	}

	for i, tc := range testRuleCases {
		tc.m.Station = 1
		res := CheckRules(tc.m, testRules)
		if tc.ok && !res {
			t.Errorf("Case %d: should be allowed", i)
		}
		if !tc.ok && res {
			t.Errorf("Case %d: should not be allowed", i)
		}
	}
}

func TestInvalidStation(t *testing.T) {
	m := &ModbusExchange{Station: 1, Base: 30001, Count: 1, Function: 4}
	if !CheckRules(m, testRules) {
		t.Fatalf("Should be allowed")
	}
	m.Station = 0
	if CheckRules(m, testRules) {
		t.Fatalf("Should not be allowed")
	}
}
