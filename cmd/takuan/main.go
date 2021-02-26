package main

import (
	"flag"

	"github.com/evilsocket/islazy/log"

	"github.com/evilsocket/takuan/core"
)

var (
	conf       = (*core.Config)(nil)
	aggregator = (*core.Aggregator)(nil)
)

func main() {
	var err error

	flag.Parse()

	setup()
	defer cleanup()

	conf, err = core.Load(confFile)
	if err != nil {
		log.Fatal("%v", err)
	}

	aggregator = core.NewAggregator(conf)

	log.Info("takuan service starting for node <%s> ...", conf.NodeName)

	if err := aggregator.Start(geoLocate); err != nil {
		log.Fatal("%v", err)
	}
}
