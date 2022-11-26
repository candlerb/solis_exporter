package main

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"go.bug.st/serial"
)

const (
	ERROR_TIMEOUT          = 1500 * time.Millisecond // time at start or after error to wait for idle line
	BUSY_TIMEOUT           = 1500 * time.Millisecond // time after sniffed exchange when we can transmit
	RESPONSE_TIMEOUT       = 1000 * time.Millisecond // max time between request and response
	POST_TRANSMIT_TIMEOUT  = 300 * time.Millisecond  // time to wait after transmit response before next transmit
	POST_BROADCAST_TIMEOUT = 500 * time.Millisecond  // time to wait after transmitting a broadcast
)

type SerialConfig struct {
	Device string `yaml:"device"`
	Dump   bool   `yaml:"dump"`
}

type Serial struct {
	Inject      chan *InjectMessage
	config      *SerialConfig
	port        serial.Port
	subscribers []chan *ModbusExchange
	msg         atomic.Pointer[ModbusExchange] // the message exchange currently in progress
}

type InjectMessage struct {
	Modbus       *ModbusExchange // already-decoded request, including CRC
	ResponseChan chan struct{}   // exchange complete; response will have been added to Modbus
}

func NewSerial(config *SerialConfig) (*Serial, error) {
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit, // although modbus spec says TwoStopBits
	}
	port, err := serial.Open(config.Device, mode)
	if err != nil {
		return nil, fmt.Errorf("%s: device %s", config.Device, err)
	}

	s := &Serial{
		Inject: make(chan *InjectMessage),
		config: config,
		port:   port,
	}
	return s, nil
}

// Add a subscriber (WARNING: not concurrency safe, do not use while running)
func (s *Serial) Subscribe(buflen int) <-chan *ModbusExchange {
	c := make(chan *ModbusExchange, buflen)
	s.subscribers = append(s.subscribers, c)
	return c
}

func (s *Serial) publishMessage(m *ModbusExchange) {
	// Distribute to subscribers (without blocking)
	for _, sub := range s.subscribers {
		select {
		case sub <- m:
		default:
		}
	}
}

func (s *Serial) readRemainderOfPacket(m *ModbusExchange, buf []byte, nread int, isRequest bool) {
	rem := 4 // minimum packet is 5 bytes including CRC
	for rem > 0 {
		// s.port.Read can return partial results.
		// It returns n == 0 for timeout.
		for rem > 0 {
			n, err := s.port.Read(buf[nread : nread+rem])
			nread += n
			rem -= n
			if err != nil {
				m.Error = err
				return
			}
			if n == 0 {
				m.Error = ERR_TIMEOUT
				return
			}
		}
		// Parse what we have, see if we need more
		if isRequest {
			rem = m.ParseRequest(buf[0:nread])
		} else {
			rem = m.ParseResponse(buf[0:nread])
		}
		if m.Error != nil {
			return
		}
	}
}

// Serial port reader goroutine: allows use of select{} for multiplexing and timeouts
func (s *Serial) serialReader(sniffer chan *ModbusExchange, response, busy chan<- struct{}) {
	dummy := make([]byte, 256)
Error:
	for {
		// prevent transmit; discard data until line is clear
		s.msg.Store(&ModbusExchange{Sniffed: true})
		s.port.SetReadTimeout(ERROR_TIMEOUT)
		for {
			n, err := s.port.Read(dummy)
			if err != nil {
				log.Printf("!receive discard: %v", err)
				time.Sleep(1 * time.Second)
			}
			if n == 0 {
				break
			}
		}
		s.msg.Store(nil)

		for {
			reqbuf := make([]byte, 256)
			// Waiting for either a sniffed request or a response
			// to an injected command.  Wait forever for first byte.
			s.port.SetReadTimeout(serial.NoTimeout)
			n, err := s.port.Read(reqbuf[0:1])
			if n != 1 || err != nil {
				log.Printf("!request first byte: %d: %v", n, err)
				continue Error
			}

			// If this is not part of a current exchange, then it's a sniffed Request.
			// If there is an ongoing exchange, then it's a Response to injected command.
			isRequest := s.msg.CompareAndSwap(nil, &ModbusExchange{Sniffed: true})
			m := s.msg.Load()

			if isRequest {
				// Signal main loop that line is now busy, even though we
				// haven't received the full exchange; but don't block
				select {
				case busy <- struct{}{}:
				default:
				}
			}

			// With go-serial-bugst, under Linux at least, Read() returns
			// whatever data is in the buffer, or the first incoming byte
			// (it uses VMIN=1), or 0 if nothing was received within the
			// timeout period.  Other OSes may have a lower resolution.
			// Although the message is ended by 3.5 character gaps (~4ms),
			// a longer timeout is fine since we calculate exactly how
			// many bytes we want to read.
			s.port.SetReadTimeout(50 * time.Millisecond)
			s.readRemainderOfPacket(m, reqbuf, 1, isRequest)

			if isRequest {
				if m.Error != nil {
					log.Printf("!request: %v", m.Error)
					continue Error
				}
				if s.config.Dump {
					log.Printf("->%02X", m.Request)
				}
				// Wait for response, except for broadcast
				if m.Station != 0 {
					respbuf := make([]byte, 256)
					s.port.SetReadTimeout(RESPONSE_TIMEOUT)
					n, err = s.port.Read(respbuf[0:1])
					if n == 0 {
						log.Printf("!response: timeout")
						continue Error
					}
					if n != 1 || err != nil {
						log.Printf("!response first byte: %d: %v", n, err)
						continue Error
					}
					s.port.SetReadTimeout(50 * time.Millisecond)
					s.readRemainderOfPacket(m, respbuf, 1, false)
					if m.Error != nil {
						log.Printf("!response: %v", m.Error)
						continue Error
					}
					if s.config.Dump {
						log.Printf("-<%02X", m.Response)
					}
				}
				s.msg.CompareAndSwap(m, nil)
				sniffer <- m
			} else {
				if s.config.Dump {
					log.Printf("=<%02X", m.Response)
				}
				s.msg.CompareAndSwap(m, nil)
				select {
				// Notify sender that response is available.
				// If sender has timed out/gone away, don't block
				case response <- struct{}{}:
				default:
				}
			}
		}
	}
}

func (s *Serial) Run() {
	log.Print("Starting serial port handler")
	sniffer := make(chan *ModbusExchange, 1)
	response := make(chan struct{}, 1)
	busy := make(chan struct{}, 1)
	go s.serialReader(sniffer, response, busy)

	var injector <-chan *InjectMessage // don't inject while it's nil
	var timeout <-chan time.Time       // don't wait while it's nil
Busy:
	for {
		injector = nil
		timeout = time.After(BUSY_TIMEOUT)
		for {
			select {
			case <-busy:
				//log.Printf("Got busy signal")
				continue Busy
			case m := <-sniffer:
				//log.Printf("Got message")
				s.publishMessage(m)
				continue Busy
			case <-timeout:
				//log.Printf("Busy: switching to idle")
				injector = s.Inject
				timeout = nil
			case i := <-injector:
				//log.Printf("Idle: injecting message")
				m := i.Modbus
				if m == nil || m.Error != nil {
					log.Printf("Inject: invalid message")
					i.ResponseChan <- struct{}{}
					continue
				}

				if m.Station != 0 {
					// Mark as transmitting. At this point we hand over responsibility
					// for updating 'm' to the serialReader goroutine, and any message
					// it receives will be considered as the response part of 'm'.
					ok := s.msg.CompareAndSwap(nil, m)
					if !ok { // This should be very rare
						log.Printf("COLLISION: inject during receive?!")
						m.Error = ERR_TIMEOUT
						i.ResponseChan <- struct{}{}
						continue Busy
					}

					// Flush any stale buffered response
					select {
					case <-response:
					default:
					}
				}

				// Send the request
				p := 0
				for p < len(m.Request) {
					n, err := s.port.Write(m.Request[p:])
					if err != nil || n < 1 {
						log.Printf("Write: %v", err)
					}
					p += n
				}
				if s.config.Dump {
					log.Printf("=>%02X", m.Request)
				}

				if m.Station != 0 {
					// Wait for response
					select {
					case <-response:
						i.ResponseChan <- struct{}{}
						s.publishMessage(m)
						// Cannot send a follow-up message until we've waited
						injector = nil
						timeout = time.After(POST_TRANSMIT_TIMEOUT)
					case <-time.After(RESPONSE_TIMEOUT):
						log.Printf("Inject: response timeout")
						m.Error = ERR_TIMEOUT
						i.ResponseChan <- struct{}{}
						continue Busy
					}
				} else {
					i.ResponseChan <- struct{}{}
					injector = nil
					timeout = time.After(POST_BROADCAST_TIMEOUT)
				}
			}
		}
	}
}
