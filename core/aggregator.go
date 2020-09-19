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

	"github.com/evilsocket/takuan/models"
)

type Aggregator struct {
	sync.Mutex

	EventBus chan models.Event
	StateBus chan models.SensorState
	ErrorBus chan error

	conf   *Config
	db     *gorm.DB
	geoip  *geoip2.Reader
	buffer []models.Event
}

func NewAggregator(conf *Config) *Aggregator {
	return &Aggregator{
		EventBus: make(chan models.Event),
		ErrorBus: make(chan error),
		StateBus: make(chan models.SensorState),
		conf:     conf,
		buffer:   make([]models.Event, 0),
	}
}

func (r *Aggregator) addEvent(e models.Event) {
	r.Lock()
	defer r.Unlock()
	r.buffer = append(r.buffer, e)
}

func (r *Aggregator) onNewBatch() {
	r.Lock()
	defer r.Unlock()

	num := len(r.buffer)

	if num > 0 {
		log.Debug("saving %d new events", num)

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

		r.buffer = make([]models.Event, 0)
	}
}

func (r *Aggregator) onReport() {
	var unreported []models.Event
    var reportURL string

	err := r.db.Where("reported_at = ?", nil).Find(&unreported).Error
	if err != nil {
		log.Error("error getting unreported events: %v", err)
		return
	}

	numUnreported := len(unreported)
	log.Info("%d unreported events", numUnreported)
	if numUnreported > 0 {
		if reportURL, err = r.conf.Reporter.OnBatch(unreported); err != nil {
			log.Error("%v", err)
			return
		}

		now := time.Now()
		for _, event := range unreported {
			event.ReportedAt = &now
			if err := r.db.Save(event).Error; err != nil {
				log.Error("error updating event reported field: %v", err)
			}
		}

		if reportURL != "" {
			log.Info("TODO: tweet %s", reportURL)
			// r.conf.Twitter.OnBatch(r.buffer, reportURL)
		}
	}
}

func (r *Aggregator) sensorStateByName(sensorName string) int64 {
	state := models.SensorState{}
	if err := r.db.Where("sensor_name = ?", sensorName).Take(&state).Error; err != nil {
		return 0
	}
	return state.LastPosition
}

func (r *Aggregator) updateState(state models.SensorState) {
	var existing models.SensorState

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

	err = r.db.AutoMigrate(&models.Event{}, &models.SensorState{})
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
		log.Info("flushing to database every %d seconds", r.conf.Database.PeriodSecs)
		dbTicker := time.NewTicker(time.Duration(r.conf.Database.PeriodSecs) * time.Second)
		// reportTicker := time.NewTicker(time.Duration(r.conf.Reporter.PeriodSecs) * time.Second)
		for _ = range dbTicker.C {
			r.onNewBatch()
		}
	}()

	go func() {
		log.Info("reporting every %d seconds", r.conf.Reporter.PeriodSecs)
		// warm up period for parsers to generate data
		time.Sleep(time.Duration(30) * time.Second)
		for {
			r.onReport()
			time.Sleep(time.Duration(r.conf.Reporter.PeriodSecs) * time.Second)
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
