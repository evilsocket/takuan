package models

import (
	"time"
)

type Event struct {
	ID          uint       `gorm:"primary_key" json:"-"`
	CreatedAt   time.Time  `gorm:"index" json:"created_at"`
	DetectedAt  time.Time  `gorm:"index" json:"detected_at"`
	DeletedAt   *time.Time `gorm:"index" json:"-"`
	NodeName    string     `gorm:"index" json:"node_name"`
	Address     string     `gorm:"index" gorm:"size:50; not null" json:"address"`
	CountryCode string     `gorm:"index" gorm:"size:5;" json:"country_code"`
	CountryName string     `json:"country_name"`
	Sensor      string     `gorm:"index" json:"sensor"`
	Rule        string     `gorm:"index" json:"rule"`
	Payload     string     `json:"payload"`
	Reported    bool       `gorm:"index; default:false" json:"reported"`
	ReportedAt  time.Time  `gorm:"index" json:"reported_at"`
}
