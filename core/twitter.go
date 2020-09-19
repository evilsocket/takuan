package core

import (
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/evilsocket/islazy/log"

	"github.com/evilsocket/takuan/models"
)

type Twitter struct {
	sync.Mutex

	Enabled        bool   `yaml:"enabled"`
	PasteBinKey    string `yaml:"pastebin_api_key"`
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

type addrCounter struct {
	Address string
	Count   int
}

func (t *Twitter) OnBatch(events []models.Event, reportURL string) {
	t.Lock()
	defer t.Unlock()
	if t.Enabled {
		/*
		content := ""

		byCountry := make(map[string]bool)
		countries := make([]string, 0)
		for _, event := range events {
			byCountry[event.CountryName] = true
		}

		for country := range byCountry {
			countries = append(countries, country)
		}

		if len(countries) > 5 {
			countries = append(countries[:5], "...")
		}

		if url, err := t.createBin(content); err != nil {
			log.Error("error creating pastebin: %v", err)
		} else {
			numEvents := len(events)
			plural := "s"
			if numEvents == 1 {
				plural = ""
			}

			tweet := fmt.Sprintf("%d new event%s from %s %s", numEvents, plural, strings.Join(countries, ", "), url)
			if err := t.postUpdate(tweet); err != nil {
				log.Error("error tweeting: %v", err)
			}
		}
		 */
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
