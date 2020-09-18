package main

import (
	"flag"

	"github.com/evilsocket/islazy/log"
)

var (
	debug    = false
	confFile = "config.yml"
)

func init() {
	flag.BoolVar(&debug, "debug", debug, "Enable debug logs.")
	flag.StringVar(&log.Output, "log", log.Output, "Log file path or empty for standard output.")
	flag.StringVar(&confFile, "config", confFile, "Configuration file.")
}

func setup() {
	if debug {
		log.Level = log.DEBUG
	} else {
		log.Level = log.INFO
	}
	log.OnFatal = log.ExitOnFatal
}

func cleanup() {

}
