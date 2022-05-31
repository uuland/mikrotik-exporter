package config

import (
	"io"
	"io/ioutil"
	"sync"

	"gopkg.in/routeros.v2"
	"gopkg.in/yaml.v2"
)

// Config represents the configuration for the exporter
type Config struct {
	Devices  []*Device       `yaml:"devices"`
	Features map[string]bool `yaml:"features,omitempty"`
}

// Device represents a target device
type Device struct {
	sync.Mutex
	Name     string           `yaml:"name"`
	Address  string           `yaml:"address,omitempty"`
	Srv      SrvRecord        `yaml:"srv,omitempty"`
	User     string           `yaml:"user"`
	Password string           `yaml:"password"`
	Port     string           `yaml:"port"`
	Cli      *routeros.Client `yaml:"-"`
}

type SrvRecord struct {
	Record string    `yaml:"record"`
	Dns    DnsServer `yaml:"dns,omitempty"`
}
type DnsServer struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

// Load reads YAML from reader and unmashals in Config
func Load(r io.Reader) (*Config, error) {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	c := &Config{}
	err = yaml.Unmarshal(b, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}
