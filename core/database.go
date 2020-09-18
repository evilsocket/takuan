package core

type Database struct {
	URL        string `yaml:"url"`
	GeoIP      string `yaml:"geoip"`
	PeriodSecs int    `yaml:"period"`
}
