package core

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/evilsocket/islazy/log"
	"github.com/enescakir/emoji"

	"github.com/evilsocket/takuan/models"
)

type Twitter struct {
	sync.Mutex

	Enabled        bool   `yaml:"enabled"`
	ConsumerKey    string `yaml:"consumer_key"`
	ConsumerSecret string `yaml:"consumer_secret"`
	AccessKey      string `yaml:"access_key"`
	AccessSecret   string `yaml:"access_secret"`

	client *twitter.Client
}

func (t *Twitter) Init() (err error) {
	config := oauth1.NewConfig(t.ConsumerKey, t.ConsumerSecret)
	token := oauth1.NewToken(t.AccessKey, t.AccessSecret)
	// http.Client will automatically authorize Requests
	httpClient := config.Client(oauth1.NoContext, token)
	// twitter client
	t.client = twitter.NewClient(httpClient)
	return
}

type countryCounter struct {
	Country string
	Count   int
}

func (t *Twitter) OnBatch(events []models.Event, reportURL string) {
	t.Lock()
	defer t.Unlock()
	if t.Enabled {
		byCountry := make(map[string]int)
		for _, event := range events {
			if _, found := byCountry[event.CountryCode]; found {
				byCountry[event.CountryCode]++
			} else {
				byCountry[event.CountryCode] = 1
			}
		}

		// sort by number of events
		countryCounters := make([]countryCounter, 0)
		for country, count := range byCountry {
			countryCounters = append(countryCounters, countryCounter{
				Country: country,
				Count:   count,
			})
		}
		sort.Slice(countryCounters, func(i, j int) bool {
			return countryCounters[i].Count > countryCounters[j].Count
		})

		countries := make([]string, 0)
		for _, country := range countryCounters {
			code := country.Country
			if flag, err := emoji.CountryFlag(code); err == nil {
				code = string(flag)
			}
			countries = append(countries, fmt.Sprintf("%s (%d)", code, country.Count))
		}

		if len(countries) > 5 {
			countries = append(countries[:5], "...")
		}

		numEvents := len(events)
		plural := "s"
		if numEvents == 1 {
			plural = ""
		}

		tweet := fmt.Sprintf("%d new event%s from %s %s #takuan #threatreport", numEvents, plural,
			strings.Join(countries, "," +
			" "), reportURL)
		if err := t.postUpdate(tweet); err != nil {
			log.Error("error tweeting: %v", err)
		}
	}
}

func (t *Twitter) postUpdate(status string) error {
	log.Info("tweet> %s", status)
	tweet, _, err := t.client.Statuses.Update(status, nil)
	if err == nil {
		log.Debug("tweet: %+v", tweet)
	}
	return err
}
