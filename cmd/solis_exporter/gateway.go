package main

// Modbus tcp gateway, for injecting messages to the target inverter

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

type GatewayConfig struct {
	Listen string `yaml:"listen"`
	Rules  []Rule `yaml:"rules"`
}

type Gateway struct {
	config   *GatewayConfig
	listener net.Listener
	inject   chan<- *InjectMessage
}

func NewGateway(config *GatewayConfig, inject chan<- *InjectMessage) (*Gateway, error) {
	if config.Listen == "" {
		config.Listen = "127.0.0.1:502"
	}
	listener, err := net.Listen("tcp", config.Listen)
	if err != nil {
		return nil, err
	}
	e := &Gateway{
		config:   config,
		inject:   inject,
		listener: listener,
	}
	return e, nil
}

func (g *Gateway) handleConnection(conn net.Conn) {
	defer conn.Close()
	responseChan := make(chan struct{})
	for {
		// Read 6 bytes: txID(2), protocol(2), length(2)
		header := make([]byte, 6, 6)
		n, err := conn.Read(header)
		if err == io.EOF {
			return
		}
		if err != nil || n < 6 {
			log.Printf("Read request header: %d: %v", n, err)
			return
		}
		proto := binary.BigEndian.Uint16(header[2:4])
		if proto != 0 {
			log.Printf("Proto: got %d", proto)
			return
		}
		l := binary.BigEndian.Uint16(header[4:6])
		if l < 2 || l > 256 {
			log.Printf("Len: got %d", l)
			return
		}
		request := make([]byte, l, l)
		n, err = conn.Read(request)
		if err != nil || n < int(l) {
			log.Printf("Read request body: %d: %v", n, err)
			return
		}

		// Parse and validate the request
		m := &ModbusExchange{}
		request = append(request, ModbusCRC(request)...)
		var response []byte
		rem := m.ParseRequest(request)
		if rem != 0 || m.Error != nil {
			log.Printf("Gateway: incomplete or invalid packet: %d: %v", rem, m.Error)
			response = []byte{request[0], request[1] | 0x80, 1}
			goto SendResponse
		}
		if !CheckRules(m, g.config.Rules) {
			log.Printf("Gateway: Rejected by rules: reg %d, count %d, function %d", m.Base, m.Count, m.Function)
			response = []byte{request[0], request[1] | 0x80, 2}
			goto SendResponse
		}

		// Inject it
		g.inject <- &InjectMessage{
			Modbus:       m,
			ResponseChan: responseChan,
		}
		<-responseChan
		if m.Station == 0 {
			// No response to broadcast
			continue
		}
		if m.Error != nil {
			log.Printf("Error in exchange: %v", m.Error)
			// Should we turn this into a modbus exception response?
			// Easier just to drop the connection on the floor
			return
		}
		if len(m.Response) < 5 {
			log.Printf("Too short response! %d", len(m.Response))
			return
		}
		response = m.Response[0 : len(m.Response)-2] // strip CRC

	SendResponse:
		binary.BigEndian.PutUint16(header[4:6], uint16(len(response)))
		n, err = conn.Write(header)
		if err != nil || n < 6 {
			log.Printf("Write response header: %d: %v", n, err)
			return
		}
		n, err = conn.Write(response)
		if err != nil || n < len(response) {
			log.Printf("Write response body: %d: %v", n, err)
			return
		}
	}
}

func (g *Gateway) Run() {
	log.Printf("Starting modbus TCP gateway on %s", g.config.Listen)
	for {
		conn, err := g.listener.Accept()
		if err != nil {
			log.Printf("listener.Accept: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}
		go g.handleConnection(conn)
	}
}
