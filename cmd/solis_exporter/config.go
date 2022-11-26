package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Serial        *SerialConfig        `yaml:"serial"`
	SolisExporter *SolisExporterConfig `yaml:"solis_exporter"`
	Gateway       *GatewayConfig       `yaml:"gateway"`
}

func ReadConfigFile(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := Config{}
	dec := yaml.NewDecoder(file)
	dec.KnownFields(true)
	err = dec.Decode(&config)
	if err != nil {
		return nil, err
	}

	if config.Serial == nil && config.SolisExporter == nil && config.Gateway == nil {
		return nil, fmt.Errorf("Empty configuration!")
	}

	return &config, nil
}
