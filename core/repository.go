package core

type repository struct {
	HTTP   string `yaml:"http"`
	Remote string `yaml:"remote"`
	Local  string `yaml:"local"`
}
