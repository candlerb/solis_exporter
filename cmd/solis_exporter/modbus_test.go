package main

import (
	"bytes"
	"testing"
)

func TestModbus04Basic(t *testing.T) {
	var n int
	m := &ModbusExchange{}
	req := []byte{0x01, 0x04, 0x80, 0xE8, 0x00, 0x01, 0x98, 0x3E}
	rep := []byte{0x01, 0x04, 0x02, 0x31, 0x05, 0x6C, 0xA3}

	// Packet corruptions
	b := make([]byte, len(req), len(req)+1)
	copy(b, req)

	n = m.ParseRequest(b[:len(b)-1])
	if n != 1 || m.Error != nil {
		t.Errorf("Unable to handle short packet: %v, %v", n, m.Error)
	}

	b[1] ^= 0x80
	n = m.ParseRequest(b)
	if m.Error == nil {
		t.Errorf("Did not detect detect invalid request with exception: %v", m.Error)
	}
	b[1] ^= 0x80

	b[len(b)-1] ^= 0xff
	n = m.ParseRequest(b)
	if m.Error != ERR_CRC_FAILED {
		t.Errorf("Did not detect CRC error: %v", m.Error)
	}
	b[len(b)-1] ^= 0xff

	b = append(b, 0xff)
	n = m.ParseRequest(b)
	if m.Error != ERR_INVALID {
		t.Errorf("Did not detect over-long packet: %v", m.Error)
	}

	// The correct request
	n = m.ParseRequest(req)
	if n != 0 || m.Error != nil {
		t.Errorf("Unable to parse valid request: %v, %v", n, m.Error)
	}

	b = make([]byte, len(rep), len(rep)+1)
	copy(b, rep)

	n = m.ParseResponse(b[:len(b)-1])
	if n != 1 || m.Error != nil {
		t.Errorf("Unable to handle short packet: %v, %v", n, m.Error)
	}

	b[len(b)-1] ^= 0xff
	n = m.ParseResponse(b)
	if m.Error != ERR_CRC_FAILED {
		t.Errorf("Did not detect CRC error: %v", m.Error)
	}
	b[len(b)-1] ^= 0xff

	b = append(b, 0xff)
	n = m.ParseResponse(b)
	if m.Error != ERR_INVALID {
		t.Errorf("Did not detect over-long packet: %v", m.Error)
	}

	n = m.ParseResponse([]byte{0x01, 0x03, 0x02, 0x00, 0x01, 0x79, 0x84})
	if m.Error != ERR_RESPONSE_MISMATCH {
		t.Errorf("Did not handle mismatched response")
	}

	// The correct response
	n = m.ParseResponse(rep)
	if n != 0 || m.Error != nil {
		t.Errorf("Unable to parse valid response")
	}

	// Check decoded values
	if !bytes.Equal(m.Request, req) {
		t.Errorf("Did not capture request: %v", m.Request)
	}
	if !bytes.Equal(m.Response, rep) {
		t.Errorf("Did not capture response: %v", m.Response)
	}
	if m.Station != 1 {
		t.Errorf("Invalid station: %02x", m.Station)
	}
	if m.Function != 4 {
		t.Errorf("Invalid function: %02x", m.Function)
	}
	if m.Base != 0x80e8 {
		t.Errorf("Invalid register base: %04x", m.Base)
	}
	if m.Count != 0x0001 {
		t.Errorf("Invalid count: %04x", m.Count)
	}
	if !bytes.Equal(m.Data, []byte{0x31, 0x05}) {
		t.Errorf("Invalid data: %v", m.Data)
	}
	if m.Exception != 0 {
		t.Errorf("Unexpected exception: %02x", m.Exception)
	}
}

func TestModbus04Exception(t *testing.T) {
	var n int
	m := &ModbusExchange{}
	req := []byte{0x01, 0x04, 0x0B, 0xB7, 0x00, 0x01, 0x83, 0xC8}
	rep := []byte{0x01, 0x84, 0x02, 0xC2, 0xC1}

	n = m.ParseRequest(req)
	if n != 0 || m.Error != nil {
		t.Errorf("Unable to parse valid request")
	}

	n = m.ParseResponse(rep)
	if n != 0 || m.Error != nil {
		t.Errorf("Unable to parse valid response")
	}

	// Check decoded values
	if m.Station != 1 {
		t.Errorf("Invalid station: %02x", m.Station)
	}
	if m.Function != 4 {
		t.Errorf("Invalid function: %02x", m.Function)
	}
	if m.Base != 0x0bb7 {
		t.Errorf("Invalid register base: %04x", m.Base)
	}
	if m.Count != 0x0001 {
		t.Errorf("Invalid count: %04x", m.Count)
	}
	if m.Data != nil {
		t.Errorf("Unexpected data: %v", m.Data)
	}
	if m.Exception != 2 {
		t.Errorf("Invalid exception: %02x", m.Exception)
	}
}

func TestModbusCRC(t *testing.T) {
	testcases := [][]byte{
		// wikipedia example
		{0x01, 0x04, 0x02, 0xFF, 0xFF, 0xB8, 0x80},
		// packets captured from inverter
		{0x01, 0x04, 0x80, 0xFE, 0x00, 0x06, 0x38, 0x38},
		{0x01, 0x84, 0x02, 0xC2, 0xC1},
		{0x01, 0x03, 0xA8, 0x01, 0x00, 0x01, 0xF5, 0xAA},
		{0x01, 0x03, 0x02, 0x00, 0x01, 0x79, 0x84},
		{0x01, 0x10, 0xA7, 0xF8, 0x00, 0x06, 0x0C, 0x00, 0x16, 0x00, 0x0B, 0x00, 0x0B, 0x00, 0x16, 0x00, 0x25, 0x00, 0x2C, 0x59, 0x2B},
		{0x01, 0x10, 0xA7, 0xF8, 0x00, 0x06, 0xE2, 0x8E},
	}
	for _, c := range testcases {
		res := ModbusCRC(c[:len(c)-2])
		if !bytes.Equal(res, c[len(c)-2:]) {
			t.Errorf("%v: got %v", c, res)
		}
	}
}
