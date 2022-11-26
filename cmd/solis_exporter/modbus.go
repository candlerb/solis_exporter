/*
This is an implementation of a minimal subset of Modbus suitable
for sniffing request and response pairs, at least with Solis inverter
*/

package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/sigurn/crc16"
)

var crcTable = crc16.MakeTable(crc16.CRC16_MODBUS)

var ERR_INVALID = fmt.Errorf("Invalid or unknown packet format")
var ERR_TIMEOUT = fmt.Errorf("Too few bytes received")
var ERR_CRC_FAILED = fmt.Errorf("CRC check failed")
var ERR_RESPONSE_MISMATCH = fmt.Errorf("Response packet does not match request")

var modbusErrorToLabel = map[error]string{
	ERR_CRC_FAILED:        "crc_failed",
	ERR_INVALID:           "decode_failed",
	ERR_RESPONSE_MISMATCH: "response_mismatch",
	ERR_TIMEOUT:           "timeout",
}

// structure representing a modbus command and response pair
type ModbusExchange struct {
	Sniffed   bool   // passively received
	Error     error  // gross transmit/receive error: rest of fields likely to be invalid
	Exception byte   // from exception responses (will be zero for no exception)
	Request   []byte // raw request, including CRC
	Response  []byte // raw response, including CRC
	Station   byte   // station ID
	Function  byte   // function code
	Base      uint16 // base register ID
	Count     uint16 // number of registers in request or response
	Data      []byte // sub-slice containing the request or response data
}

// Parse a complete or partial modbus request.  If it is incomplete,
// return the number of further bytes to be read (including CRC).
// Otherwise return 0 for a complete packet, decode and validate it.
// Set m.Error for gross protocol violations; the caller should
// assume messaging is corrupt and resynchronize.
func (m *ModbusExchange) ParseRequest(pkt []byte) int {
	m.Error = nil
	l := len(pkt)
	if l < 2 {
		return 5 - l // we will always need at least 5 bytes
	}
	if (pkt[1] & 0x80) == 0x80 { // only allowed in responses
		m.Error = ERR_RESPONSE_MISMATCH
		return 0
	}
	var exp int
	switch pkt[1] {
	case 0x02, 0x03, 0x04, 0x06:
		// STA-1 FUN-1 REG-2 CNT-2 CRC-2
		// STA-1 FUN-1 REG-2 VAL-2 CRC-2  # 06
		exp = 8
	case 0x10:
		// STA-1 FUN-1 REG-2 CNT-2 LEN-1 DATA-LEN CRC-2
		if l < 7 {
			return 9 - l
		}
		exp = int(pkt[6]) + 9
	}

	// Don't know?
	if exp == 0 {
		m.Error = ERR_INVALID
		return 0
	}
	// Incomplete packet?
	if l < exp {
		return exp - l
	}
	// Over-long packet?
	if l > exp {
		m.Error = ERR_INVALID
		return 0
	}
	// CRC check
	plen := l - 2
	if !bytes.Equal(pkt[plen:], ModbusCRC(pkt[:plen])) {
		m.Error = ERR_CRC_FAILED
		return 0
	}

	// Now we can decode it
	m.Request = pkt
	m.Station = pkt[0]
	m.Function = pkt[1]

	switch m.Function {
	case 0x02, 0x03, 0x04:
		// STA-1 FUN-1 REG-2 CNT-2 CRC-2
		m.Base = binary.BigEndian.Uint16(pkt[2:])
		m.Count = binary.BigEndian.Uint16(pkt[4:])
	case 0x06:
		// STA-1 FUN-1 REG-2 VAL-2 CRC-2
		m.Base = binary.BigEndian.Uint16(pkt[2:])
		m.Count = 1
		m.Data = pkt[4:6]
	case 0x10:
		// STA-1 FUN-1 REG-2 CNT-2 LEN-1 DATA-LEN CRC-2
		m.Base = binary.BigEndian.Uint16(pkt[2:])
		m.Count = binary.BigEndian.Uint16(pkt[4:])
		m.Data = pkt[7:plen]
		if int(pkt[6]) != len(m.Data) {
			m.Error = ERR_INVALID // should not happen
			return 0
		}
	}
	return 0
}

// Parse a complete or partial modbus response.  If it is incomplete,
// return the number of further bytes to be read (including CRC).
// Otherwise return 0 for a complete packet, decode and validate it
// in the context of the corresponding request.
// Set m.Error for gross protocol violations; the caller should
// assume messaging is corrupt and resynchronize.
func (m *ModbusExchange) ParseResponse(pkt []byte) int {
	m.Error = nil
	l := len(pkt)
	if l < 2 {
		return 5 // we will always need at least 5 bytes
	}
	var exp int
	if (pkt[1] & 0x80) == 0x80 { // fixed-size error response
		exp = 5
	} else {
		switch m.Function {
		case 0x02, 0x03, 0x04:
			// STA-1 FUN-1 LEN-1 DATA-LEN CRC-2
			if l < 3 {
				return 5
			}
			exp = int(pkt[2]) + 5
		case 0x06, 0x10:
			// STA-1 FUN-1 REG-2 VAL-2 CRC-2
			// STA-1 FUN-1 REG-2 CNT-2 CRC-2
			exp = 8
		}
	}

	// Don't know?
	if exp == 0 {
		m.Error = ERR_INVALID
		return 0
	}
	// Incomplete packet?
	if l < exp {
		return exp - l
	}
	// Over-long packet?
	if l > exp {
		m.Error = ERR_INVALID
		return 0
	}
	// CRC check
	plen := l - 2
	if !bytes.Equal(pkt[plen:], ModbusCRC(pkt[:plen])) {
		m.Error = ERR_CRC_FAILED
		return 0
	}

	// Now we can decode it
	m.Response = pkt
	if pkt[0] != m.Station {
		m.Error = ERR_RESPONSE_MISMATCH
		return 0
	}
	if pkt[1] == (m.Function | 0x80) {
		m.Exception = pkt[2] & 0x7f
		return 0
	} else if pkt[1] != m.Function {
		m.Error = ERR_RESPONSE_MISMATCH
		return 0
	}

	switch m.Function {
	case 0x02, 0x03, 0x04:
		// STA-1 FUN-1 LEN-1 DATA-LEN CRC-2
		m.Data = pkt[3:plen]
		if int(pkt[2]) != len(m.Data) {
			m.Error = ERR_INVALID // should not happen
			return 0
		}
	case 0x06:
		// STA-1 FUN-1 REG-2 VAL-2 CRC-2
		if binary.BigEndian.Uint16(pkt[2:]) != m.Base {
			m.Error = ERR_RESPONSE_MISMATCH
			return 0
		}
		m.Count = 1
		m.Data = pkt[4:6]
	case 0x10:
		// STA-1 FUN-1 REG-2 CNT-2 CRC-2
		if binary.BigEndian.Uint16(pkt[2:]) != m.Base {
			m.Error = ERR_RESPONSE_MISMATCH
			return 0
		}
		m.Count = binary.BigEndian.Uint16(pkt[4:])
	}
	return 0
}

func ModbusCRC(pkt []byte) []byte {
	crc := crc16.Checksum(pkt, crcTable)
	return []byte{byte(crc & 0xff), byte(crc >> 8)}
}
