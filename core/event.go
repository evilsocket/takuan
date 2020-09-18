package core

import (
	"time"
)

type Event struct {
	ID         string    `json:"id"`
	DetectedAt time.Time `json:"detected_at"`
	Time       time.Time `json:"happened_at"`
	Address    string    `json:"address"`
	Country    string    `json:"country"`
	Hostname   string    `json:"hostname"`
	LogLine    string    `json:"line"`
	Tokens     Tokens    `json:"tokens"`
	Payload    string    `json:"payload"`
	Sensor     string    `json:"sensor"`
	Rule       string    `json:"rule"`
}
