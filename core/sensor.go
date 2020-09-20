package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/evilsocket/islazy/log"

	"github.com/evilsocket/takuan/models"
)

type Sensor struct {
	Name       string  `yaml:"name"`
	Enabled    bool    `yaml:"enabled"`
	Filename   string  `yaml:"filename"`
	PeriodSecs int     `yaml:"period"`
	Parser     *Parser `yaml:"parser"`
	Rules      []*Rule `yaml:"rules"`

	fp      *os.File
	lastPos int64
}

func (s *Sensor) Compile() error {
	if s.Enabled {
		if err := s.Parser.Compile(); err != nil {
			return err
		}

		for _, r := range s.Rules {
			if err := r.Compile(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Sensor) Start(events chan models.Event, errors chan error, states chan models.SensorState, state int64) {
	if !s.Enabled {
		return
	}

	go func() {
		log.Info("sensor %s started for file %s (from offset %d)...", s.Name, s.Filename, state)
		s.lastPos = state

		for {
			var err error

			s.fp, err = os.Open(s.Filename)
			if err != nil {
				errors <- err
				continue
			}

			// if file size < last pos, reset last pos
			if stat, err := s.fp.Stat(); err != nil {
				s.fp.Close()
				errors <- err
				continue
			} else if stat.Size() < s.lastPos {
				log.Debug("resetting last offset for %s", s.Filename)
				s.lastPos = 0
			}

			// continue from the last position
			_, err = s.fp.Seek(s.lastPos, os.SEEK_SET)
			if err != nil {
				s.fp.Close()
				errors <- err
				continue
			}

			scanner := bufio.NewScanner(s.fp)
			scanner.Split(bufio.ScanLines)

			// for each new line
			for scanner.Scan() {
				line := scanner.Text()
				if matched, tokens := s.Parser.Parse(line); matched {
					// TODO: use work queue

					// for each rule
					for _, r := range s.Rules {
						if matched, _ := r.Match(tokens); matched {
							event := models.Event{
								DetectedAt: time.Now(),
								Address:    tokens["address"],
								Payload:    line,
								Rule:       r.Name,
								Sensor:     s.Name,
							}

							event.CreatedAt, err = time.Parse(s.Parser.DatetimeFormat, tokens["datetime"])
							if err != nil {
								errors <- fmt.Errorf("could not parse datetime '%s' with format '%s': %v", tokens["datetime"], s.Parser.DatetimeFormat, err)
							}

							events <- event
							break
						}
					}
				} else {
					// log.Info("nope: %s", line)
				}
			}

			s.lastPos, _ = s.fp.Seek(0, io.SeekCurrent)
			s.fp.Close()

			states <- models.SensorState{
				SensorName:   s.Name,
				LastPosition: s.lastPos,
			}

			time.Sleep(time.Duration(s.PeriodSecs) * time.Second)
		}
	}()
}
