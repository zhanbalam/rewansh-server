package main

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	defaultAddress = "localhost:8080"
	defaultMaxTime = 1
)

type config struct {
	Addr      string     `yaml:"addr"`
	HttpHosts []httpHost `yaml:"http_hosts"`
	MaxTime   uint       `yaml:"maxtime"`
}

type httpHost struct {
	Url          string `yaml:"url"`
	RequestData  string `yaml:"request"`
	ResponseData string `yaml:"response"`
}

func loadConfig(path string) (*config, error) {
	c := &config{}
	c.setDefaults()

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open config file: %v", err)
	}
	defer file.Close()

	d := yaml.NewDecoder(file)
	if err := d.Decode(&c); err != nil {
		return nil, fmt.Errorf("Failed read config file: %v", err)
	}

	if err := c.validate(); err != nil {
		return nil, fmt.Errorf("Validation failed: %v", err)
	}

	return c, nil
}

func (c *config) setDefaults() {
	c.Addr = defaultAddress
	c.MaxTime = defaultMaxTime
}

func (c *config) validate() error {
	if len(c.HttpHosts) == 0 {
		return fmt.Errorf("site checking requires at least one host")
	}
	for _, h := range c.HttpHosts {
		_, err := url.Parse(h.Url)
		if err != nil {
			return fmt.Errorf("Invalid host url: %s", err)
		}
	}
	return nil
}
