package core

import (
	"encoding/csv"
	"fmt"
	"github.com/go-git/go-git/v5/plumbing/object"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/evilsocket/islazy/fs"
	"github.com/evilsocket/islazy/log"
	"github.com/go-git/go-git/v5"

	"github.com/evilsocket/takuan/models"
)

type Reporter struct {
	sync.Mutex

	Enabled    bool       `yaml:"enabled"`
	Repository repository `yaml:"repository"`

	repo *git.Repository
	tree *git.Worktree
}

func (r *Reporter) Init() (err error) {
	if fs.Exists(r.Repository.Local) {
		// open local copy and pull
		if r.repo, err = git.PlainOpen(r.Repository.Local); err != nil {
			return fmt.Errorf("error while opening git repo %s: %v", r.Repository.Local, err)
		}

		r.tree, err = r.repo.Worktree()
		if err != nil {
			return fmt.Errorf("error while getting working tree for git repo %s: %v", r.Repository.Local, err)
		}

		log.Info("updating %s from %s ...", r.Repository.Local, r.Repository.Remote)

		if err = r.tree.Pull(&git.PullOptions{RemoteName: "origin"}); err != nil {
			return fmt.Errorf("error while updating git repo %s: %v", r.Repository.Local, err)
		}
	} else {
		log.Info("cloning %s to %s ...", r.Repository.Remote, r.Repository.Local)

		r.repo, err = git.PlainClone(r.Repository.Local, true, &git.CloneOptions{
			URL: r.Repository.Remote,
			Progress: os.Stdout,
		})

		if err != nil {
			return fmt.Errorf("error while cloning git repo %s to %s: %v", r.Repository.Remote, r.Repository.Local, err)
		}

		r.tree, err = r.repo.Worktree()
		if err != nil {
			return fmt.Errorf("error while getting working tree for git repo %s: %v", r.Repository.Local, err)
		}
	}

	return nil
}

func (r *Reporter) OnBatch(events []models.Event) (reportURL string, err error) {
	r.Lock()
	defer r.Unlock()

	if r.Enabled {
		byAddress := make(map[string][]models.Event)
		for _, event := range events {
			if list, found := byAddress[event.Address]; found {
				byAddress[event.Address] = append(list, event)
			} else {
				byAddress[event.Address] = []models.Event{event}
			}
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

		fileBaseName := fmt.Sprintf("report_%s.csv", time.Now().Format("2006-01-02T15:04:05-0700"))
		fileName := path.Join(r.Repository.Local, fileBaseName)

		fp, err := os.Create(fileName)
		if err != nil {
			return "", fmt.Errorf("error creating %s: %v", fileName, err)
		}

		log.Info("saving report to %s", fileName)

		writer := csv.NewWriter(fp)

		writer.Write([]string{
			"address",
			"country_code",
			"country_name",
			"total_events",
			"counters",
		})

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

			writer.Write([]string {
				address,
				addrEvents[0].CountryCode,
				addrEvents[0].CountryName,
				fmt.Sprintf("%d", c.Count),
				strings.Join(counters, "|"),
			})
		}

		writer.Flush()
		fp.Close()

		// add, commit and push
		log.Info("updating repository")

		if _, err := r.tree.Add(fileName); err != nil {
			return "", fmt.Errorf("error while updating git repo %s: %v", r.Repository.Local, err)
		}

		_, err = r.tree.Commit("new report", &git.CommitOptions{
			Author: &object.Signature{
				When:  time.Now(),
			},
		})

		if err = r.repo.Push(&git.PushOptions{}); err != nil {
			return "", fmt.Errorf("error while updating git repo %s: %v", r.Repository.Local, err)
		}

		reportURL := r.Repository.HTTP
		if !strings.HasSuffix(reportURL, "/") {
			reportURL += "/"
		}
		reportURL = fmt.Sprintf("%s%s", reportURL, fileBaseName)

		log.Info("new report saved to %s", reportURL)

		return reportURL, nil
	}

	return "", nil
}