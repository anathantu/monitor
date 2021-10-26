package scrape

import (
	"github.com/go-kit/log"
	"sync"
)

// NewManager is the Manager constructor
//func NewManager(o *Options, logger log.Logger, app storage.Appendable) *Manager {
//	if o == nil {
//		o = &Options{}
//	}
//	if logger == nil {
//		logger = log.NewNopLogger()
//	}
//	m := &Manager{
//		append:        app,
//		opts:          o,
//		logger:        logger,
//		scrapeConfigs: make(map[string]*config.ScrapeConfig),
//		scrapePools:   make(map[string]*scrapePool),
//		graceShut:     make(chan struct{}),
//		triggerReload: make(chan struct{}, 1),
//	}
//
//	return m
//}

// Manager maintains a set of scrape pools and manages start/stop cycles
// when receiving new target groups from the discovery manager.
type Manager struct {
	logger log.Logger
	//append    storage.Appendable
	//graceShut chan struct{}

	mtxScrape sync.Mutex // Guards the fields below.
	//scrapeConfigs map[string]*config.ScrapeConfig
	//scrapePools   map[string]*scrapePool
	//targetSets    map[string][]*targetgroup.Group

	triggerReload chan struct{}
}
