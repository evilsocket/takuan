package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/evilsocket/islazy/log"
	"github.com/teris-io/shortid"
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
	if err := s.Parser.Compile(); err != nil {
		return err
	}

	for _, r := range s.Rules {
		if err := r.Compile(); err != nil {
			return err
		}
	}

	return nil
}

func (s *Sensor) Start(events chan Event, errors chan error) {
	go func() {
		log.Info("sensor %s started for file %s ...", s.Name, s.Filename)

		for {
			var err error

			log.Debug("sensor %s running %d rules from offset %d", s.Name, len(s.Rules), s.lastPos)

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
						if matched, value := r.Match(tokens); matched {
							event := Event{
								ID:         shortid.MustGenerate(),
								DetectedAt: time.Now(),
								Address:    tokens["address"],
								LogLine:    line,
								Tokens:     tokens,
								Payload:    value,
								Rule:       r.Name,
								Sensor:     s.Name,
							}

							event.Time, err = time.Parse(s.Parser.DatetimeFormat, tokens["datetime"])
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

			time.Sleep(time.Duration(s.PeriodSecs) * time.Second)
		}
	}()
}
