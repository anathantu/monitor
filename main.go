package main

import (
	"fmt"
	"github.com/anathantu/monitor/config"
	"github.com/anathantu/monitor/scrape"
	_ "github.com/anathantu/monitor/scrape"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"os"
	"sync"
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

	//确保某些操作在高并发的场景下只执行一次，例如只加载一次配置文件、只关闭一次通道等
	// sync.Once is used to make sure we can close the channel
	//at different execution stages(SIGTERM or when the config is loaded).
	type closeOnce struct {
		C     chan struct{}
		once  sync.Once
		Close func()
	}

	// Wait until the server is ready to handle reloading.
	reloadReady := &closeOnce{
		C: make(chan struct{}),
	}
	reloadReady.Close = func() {
		reloadReady.once.Do(func() {
			close(reloadReady.C)
		})
	}

	level.Info(logger).Log("msg", fmt.Sprintf("monitor.yml  %s", cfgFile.String()))
	var (
		//ctxScrape, cancelScrape = context.WithCancel(context.Background())
		//discoveryManagerScrape  = discovery.NewManager(ctxScrape,
		//	log.With(logger, "component", "discovery manager scrape"), discovery.Name("scrape"))
		scrapeManager = scrape.NewManager(log.With(logger, "component", "scrape manager"))
	)

	var g run.Group
	{
		//initial config
		cancel := make(chan struct{})
		g.Add(
			func() error {
				//select {
				//case <-dbOpen:
				//// In case a shutdown is initiated before the dbOpen is released
				//case <-cancel:
				//	reloadReady.Close()
				//	return nil
				//}

				if err := reloadConfig(cfgFile, scrapeManager); err != nil {
					return errors.Wrapf(err, "error loading config from %q", cfgFile)
				}

				reloadReady.Close()
				level.Info(logger).Log("msg", "Server is ready to receive web requests.")
				<-cancel
				return nil
			},
			func(err error) {
				close(cancel)
			},
		)
	}
	{
		g.Add(func() error {
			<-reloadReady.C
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

func reloadConfig(cfg *config.Config, m *scrape.Manager) (err error) {
	m.ApplyConfig(cfg)
	return nil
}
