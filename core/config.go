package core

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
	"github.com/evilsocket/islazy/log"
)

type Config struct {
	NodeName string    `yaml:"name"`
	Debug    bool      `yaml:"debug"`
	Database Database  `yaml:"database"`
	Reporter *Reporter `yaml:"reports"`
	Twitter  *Twitter  `yaml:"twitter"`
	Sensors  []*Sensor `yaml:"sensors"`
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

	if conf.Debug {
		log.Level = log.DEBUG
	}

	for _, sensor := range conf.Sensors {
		if err = sensor.Compile(); err != nil {
			return nil, err
		}
	}

	if conf.Reporter.Enabled {
		if err = conf.Reporter.Init(); err != nil {
			return nil, err
		}
	}

	if conf.Twitter.Enabled {
		if err = conf.Twitter.Init(); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}
