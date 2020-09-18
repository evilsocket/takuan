package core

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/evilsocket/islazy/log"
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

func (t *Twitter) OnBatch(events []Event) {
	t.Lock()
	defer t.Unlock()
	if t.Enabled {
		content := ""

		byAddress := make(map[string][]Event)
		byCountry := make(map[string]bool)
		countries := make([]string, 0)
		for _, event := range events {
			if list, found := byAddress[event.Address]; found {
				byAddress[event.Address] = append(list, event)
			} else {
				byAddress[event.Address] = []Event{event}
			}
			byCountry[event.CountryName] = true
		}

		for country := range byCountry {
			countries = append(countries, country)
		}

		if len(countries) > 5 {
			countries = append(countries[:5], "...")
		}

		// sort by number of events
		addrCounters := make([]addrCounter, 0)
		for address, addrEvents := range byAddress {
			addrCounters = append(addrCounters, addrCounter{
				Address: address,
				Count:   len(addrEvents),
			})
		}

		sort.Slice(addrCounters, func(i, j int) bool {
			return addrCounters[i].Count > addrCounters[j].Count
		})

		//for address, addrEvents := range byAddress {
		for _, c := range addrCounters {
			address := c.Address
			addrEvents := byAddress[address]

			byTypeName := make(map[string]int)
			counters := make([]string, 0)
			for _, event := range addrEvents {
				typeName := fmt.Sprintf("%s/%s", event.Sensor, event.Rule)
				if _, found := byTypeName[typeName]; found {
					byTypeName[typeName]++
				} else {
					byTypeName[typeName] = 1
				}
			}

			for typeName, count := range byTypeName {
				counters = append(counters, fmt.Sprintf("%s:%d", typeName, count))
			}

			content += fmt.Sprintf("%s from %s: %s total:%d\n",
				address,
				addrEvents[0].CountryName,
				strings.Join(counters, " "),
				c.Count)
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
	}
}

func (t *Twitter) createBin(content string) (string, error) {
	data := url.Values{
		"api_option":        {"paste"},
		"api_dev_key":       {t.PasteBinKey},
		"api_paste_private": {"1"}, // unlisted
		"api_paste_name":    {"takuan report"},
		"api_paste_code":    {content},
		// "api_paste_expire_date": {"1H"},
	}

	resp, err := http.PostForm("https://pastebin.com/api/api_post.php", data)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response %v", resp.StatusCode)
	}

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	url := string(bodyBytes)

	if !strings.HasPrefix(url, "https://pastebin.com/") {
		return "", fmt.Errorf("invalid  url response: %s", url)
	}

	return url, nil
}

func (t *Twitter) postUpdate(status string) error {
	log.Info("tweet> %s", status)
	tweet, _, err := t.client.Statuses.Update(status, nil)
	if err == nil {
		log.Debug("tweet: %+v", tweet)
	}
	return err
}
