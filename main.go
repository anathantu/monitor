package main

import (
	"fmt"
	"github.com/anathantu/monitor/config"
	_ "github.com/anathantu/monitor/scrape"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"os"
)

func main() {
	var logger log.Logger
	var cfgFile *config.Config
	var err error
	filename := "monitor.yml"
	if cfgFile, err = config.LoadFile(filename, log.NewNopLogger()); err != nil {
		level.Error(logger).Log("msg", fmt.Sprintf("Error loading config (--config.file=%s)", filename), "err", err)
		os.Exit(2)
	}

	level.Info(logger).Log("msg", fmt.Sprintf("monitor.yml  %s", cfgFile))
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
