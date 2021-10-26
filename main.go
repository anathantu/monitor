package main

import (
	"github.com/anathantu/monitor/config"
	"github.com/anathantu/monitor/scrape"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"os"
)

func main() {
	var logger log.Logger
	var cfgFile *config.Config
	var (
	//ctxScrape, cancelScrape = context.WithCancel(context.Background())
	//discoveryManagerScrape  = discovery.NewManager(ctxScrape,
	//	log.With(logger, "component", "discovery manager scrape"), discovery.Name("scrape"))
	//scrapeManager = scrape.NewManager(&cfg.scrape, log.With(logger, "component", "scrape manager"), fanoutStorage)
	)

	var g run.Group
	{
		g.Add(func() error {

			level.Info(logger).Log("msg", "Scrape discovery manager stopped")
			return nil
		}, func(err error) {
			level.Info(logger).Log("msg", "Stopping scrape discovery manager...")
		})
	}
	if err := g.Run(); err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "See you next time!")
}
