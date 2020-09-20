package models

type SensorState struct {
	ID           uint   `gorm:"primary_key" json:"-"`
	NodeName     string `gorm:"index" gorm:"column:node_name"`
	SensorName   string `gorm:"index" gorm:"column:sensor_name"`
	LastPosition int64  `gorm:"index" gorm:"column:last_position"`
}
