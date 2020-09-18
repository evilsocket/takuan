package main

import (
	"flag"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/evilsocket/islazy/log"
	"github.com/oschwald/geoip2-golang"

	"github.com/evilsocket/takuan/core"
)

var (
	conf      = (*core.Config)(nil)
	geoip     = (*geoip2.Reader)(nil)
	groupLock = sync.Mutex{}
	groups    = make(map[string][]core.Event)
)

func add(event core.Event) {
	groupLock.Lock()
	defer groupLock.Unlock()

	if events, found := groups[event.Address]; found {
		groups[event.Address] = append(events, event)
	} else {
		groups[event.Address] = []core.Event{event}
	}
}

type counter struct {
	Address string
	Count   int
}

func report() {
	groupLock.Lock()
	defer groupLock.Unlock()

	totByAddress := []counter{}

	for address, events := range groups {
		totByAddress = append(totByAddress, counter{
			Address: address,
			Count:   len(events),
		})
	}

	sort.Slice(totByAddress, func(i, j int) bool {
		return totByAddress[i].Count > totByAddress[j].Count
	})

	if conf.Report.Top > 0 {
		numTot := len(totByAddress)
		if numTot > conf.Report.Top {
			log.Info("top %d of %d", conf.Report.Top, numTot)
			totByAddress = totByAddress[:conf.Report.Top]
		}
	}

	for _, counter := range totByAddress {
		byType := make(map[string][]core.Event)
		for _, e := range groups[counter.Address] {
			typeName := fmt.Sprintf("%s/%s", e.Sensor, e.Rule)
			if evs, found := byType[typeName]; found {
				byType[typeName] = append(evs, e)
			} else {
				byType[typeName] = []core.Event{e}
			}
		}

		counters := []string{}
		for typeName, events := range byType {
			counters = append(counters, fmt.Sprintf("%s:%d", typeName, len(events)))
		}

		countryCode := "??"
		country, err := geoip.Country(net.ParseIP(counter.Address))
		if err != nil {
			log.Error("%v", err)
		} else {
			countryCode = country.Country.IsoCode
		}

		hostName := ""
		names, err := net.LookupAddr(counter.Address)
		if err == nil {
			hostName = names[0]
		}
		log.Info("%s %s %s (%d): %s", counter.Address, countryCode, hostName, counter.Count, strings.Join(counters, " "))
	}

	// reset
	groups = make(map[string][]core.Event)
}

func main() {
	var err error

	flag.Parse()

	setup()
	defer cleanup()

	conf, err = core.Load(confFile)
	if err != nil {
		log.Fatal("error loading configuration from %s: %v", confFile, err)
	}

	geoip, err = geoip2.Open(conf.GeoIP)
	if err != nil {
		log.Fatal("%+v", err)
	}
	defer geoip.Close()

	log.Info("takuan service starting ...")

	events := make(chan core.Event)
	errors := make(chan error)

	for _, sensor := range conf.Sensors {
		if sensor.Enabled {
			sensor.Start(events, errors)
		} else {
			log.Debug("sensor %s is disabled", sensor.Name)
		}
	}

	go func() {
		ticker := time.NewTicker(time.Duration(conf.Report.PeriodSecs) * time.Second)
		for _ = range ticker.C {
			report()
		}
	}()

	for {
		select {
		case event := <-events:
			// log.Debug("<%s> %s/%s: %s -> %s", event.Time, event.Sensor, event.Rule, event.Address, event.Payload)
			add(event)
		case err := <-errors:
			log.Error("error: %v", err)
		}
	}
}
