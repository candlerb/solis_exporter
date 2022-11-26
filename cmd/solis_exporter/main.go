package main

import (
	"flag"
	"log"
	"sync"
)

var config *Config

func main() {
	var err error
	var cf = flag.String("config", "solis_exporter.yml", "path to configuration file")
	flag.Parse()

	config, err := ReadConfigFile(*cf)
	if err != nil {
		log.Fatalf("read config: %s\n", err)
	}

	var serial *Serial
	if config.Serial != nil {
		serial, err = NewSerial(config.Serial)
		if err != nil {
			log.Fatalf("serial: %s\n", err)
		}
		if config.Serial.Dump {
			log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
		}
	}

	var exporter *SolisExporter
	if config.SolisExporter != nil {
		exporter, err = NewSolisExporter(config.SolisExporter, serial.Subscribe(5))
		if err != nil {
			log.Fatalf("solis_exporter: %s\n", err)
		}
	}

	var gateway *Gateway
	if config.Gateway != nil {
		if serial == nil {
			log.Fatalf("gateway requires serial")
		}
		gateway, err = NewGateway(config.Gateway, serial.Inject)
		if err != nil {
			log.Fatalf("gateway: %s\n", err)
		}
	}

	var wg sync.WaitGroup
	if exporter != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			exporter.Run()
		}()
	}
	if gateway != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			gateway.Run()
		}()
	}
	if serial != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			serial.Run()
		}()
	}
	wg.Wait()
}
