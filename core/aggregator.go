package core

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/evilsocket/islazy/log"
	"github.com/oschwald/geoip2-golang"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Aggregator struct {
	sync.Mutex

	EventBus chan Event
	ErrorBus chan error
	StateBus chan SensorState

	conf   *Config
	db     *gorm.DB
	geoip  *geoip2.Reader
	buffer []Event
}

func NewAggregator(conf *Config) *Aggregator {
	return &Aggregator{
		EventBus: make(chan Event),
		ErrorBus: make(chan error),
		StateBus: make(chan SensorState),
		conf:     conf,
		buffer:   make([]Event, 0),
	}
}

func (r *Aggregator) addEvent(e Event) {
	r.Lock()
	defer r.Unlock()
	r.buffer = append(r.buffer, e)
}

func (r *Aggregator) saveBatch() {
	r.Lock()
	defer r.Unlock()

	num := len(r.buffer)

	if num > 0 {
		log.Info("saving %d new events", num)

		started := time.Now()

		for i, event := range r.buffer {
			event.NodeName = r.conf.NodeName
			country, err := r.geoip.Country(net.ParseIP(event.Address))
			if err == nil {
				event.CountryCode = country.Country.IsoCode
				event.CountryName = country.Country.Names["en"]
			}

			/*
				SLOW

				names, err := net.LookupAddr(event.Address)
				if err == nil {
					event.Hostname = names[0]
				}
			*/

			if err := r.db.Create(&event).Error; err != nil {
				log.Error("error saving event: %v", err)
			}

			r.buffer[i] = event
		}

		log.Info("%d events saved in %s", num, time.Since(started))

		r.conf.Twitter.OnBatch(r.buffer)

		r.buffer = make([]Event, 0)
	}
}

func (r *Aggregator) sensorStateByName(sensorName string) int64 {
	state := SensorState{}
	if err := r.db.Where("sensor_name = ?", sensorName).Take(&state).Error; err != nil {
		return 0
	}
	return state.LastPosition
}

func (r *Aggregator) updateState(state SensorState) {
	var existing SensorState

	log.Debug("updating sensor state: %s -> %d", state.SensorName, state.LastPosition)

	err := r.db.Where("sensor_name = ?", state.SensorName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		log.Debug("creating state %v", state)
		err = r.db.Create(&state).Error
	} else if state.LastPosition != existing.LastPosition {
		log.Debug("updating state %v -> %v", existing, state)
		existing.LastPosition = state.LastPosition
		err = r.db.Save(&existing).Error
	}

	if err != nil {
		log.Error("error updating sensor state %v: %v", state, err)
	}
}

func (r *Aggregator) Start() (err error) {
	r.geoip, err = geoip2.Open(r.conf.Database.GeoIP)
	if err != nil {
		return err
	}

	r.db, err = gorm.Open(mysql.Open(r.conf.Database.URL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return err
	}

	log.Debug("connected to the database")

	err = r.db.AutoMigrate(&Event{}, &SensorState{})
	if err != nil {
		return fmt.Errorf("error performing database migration: %v", err)
	}

	for _, sensor := range r.conf.Sensors {
		if sensor.Enabled {
			sensor.Start(r.EventBus, r.ErrorBus, r.StateBus, r.sensorStateByName(sensor.Name))
		} else {
			log.Debug("sensor %s is disabled", sensor.Name)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Duration(r.conf.Database.PeriodSecs) * time.Second)
		for _ = range ticker.C {
			r.saveBatch()
		}
	}()

	for {
		select {
		case state := <-r.StateBus:
			r.updateState(state)

		case event := <-r.EventBus:
			r.addEvent(event)

		case err := <-r.ErrorBus:
			log.Error("%v", err)
		}
	}

	return nil
}
