package core

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Config struct {
	GeoIP   string    `yaml:"geoip"`
	Report  Report    `yaml:"report"`
	Sensors []*Sensor `yaml:"sensors"`
}

func Load(filename string) (*Config, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	conf := Config{}

	if err = yaml.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	for _, sensor := range conf.Sensors {
		if err = sensor.Compile(); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}
