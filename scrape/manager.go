package scrape

import (
	"github.com/anathantu/monitor/config"
	"github.com/anathantu/monitor/discovery"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"sync"
	"time"
)

// NewManager is the Manager constructor
func NewManager(logger log.Logger) *Manager {
	if logger == nil {
		logger = log.NewNopLogger()
	}
	m := &Manager{
		logger:        logger,
		scrapeConfigs: make(map[string]*config.ScrapeConfig),
		scrapePools:   make(map[string]*scrapePool),
		graceShut:     make(chan struct{}),
		triggerReload: make(chan struct{}, 1),
	}

	return m
}

// Manager maintains a set of scrape pools and manages start/stop cycles
// when receiving new target groups from the discovery manager.
type Manager struct {
	logger log.Logger
	//append    storage.Appendable
	graceShut chan struct{}

	MtxScrape     sync.Mutex // Guards the fields below.
	scrapeConfigs map[string]*config.ScrapeConfig
	scrapePools   map[string]*scrapePool
	targetSets    map[string]*discovery.TargetGroup

	triggerReload chan struct{}
}

func (m *Manager) ApplyConfig(cfg *config.Config) {
	for _, sc := range cfg.ScrapeConfigs {
		m.scrapeConfigs[sc.JobName] = sc
	}
}

// Run receives and saves target set updates and triggers the scraping loops reloading.
// Reloading happens in the background so that it doesn't block receiving targets updates.
func (m *Manager) Run(tsets <-chan map[string]*discovery.TargetGroup) error {
	go m.reloader()
	for {
		select {
		case ts := <-tsets:
			m.updateTsets(ts)

			select {
			case m.triggerReload <- struct{}{}:
			default:
			}

		case <-m.graceShut:
			return nil
		}
	}
}

// Stop cancels all running scrape pools and blocks until all have exited.
func (m *Manager) Stop() {
	m.MtxScrape.Lock()
	defer m.MtxScrape.Unlock()

	//for _, sp := range m.scrapePools {
	//	sp.stop()
	//}
	close(m.graceShut)
}

func (m *Manager) reloader() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.graceShut:
			return
		case <-ticker.C:
			select {
			case <-m.triggerReload:
				m.reload()
			case <-m.graceShut:
				return
			}
		}
	}
}

func (m *Manager) reload() {
	m.MtxScrape.Lock()
	var wg sync.WaitGroup
	for setName, groups := range m.targetSets {
		if _, ok := m.scrapePools[setName]; !ok {
			scrapeConfig, ok := m.scrapeConfigs[setName]
			if !ok {
				level.Error(m.logger).Log("msg", "error reloading target set", "err", "invalid config id:"+setName)
				continue
			}
			sp, err := newScrapePool(scrapeConfig, log.With(m.logger, "scrape_pool", setName))
			if err != nil {
				level.Error(m.logger).Log("msg", "error creating new scrape pool", "err", err, "scrape_pool", setName)
				continue
			}
			m.scrapePools[setName] = sp
		}

		wg.Add(1)
		// Run the sync in parallel as these take a while and at high load can't catch up.
		go func(sp *scrapePool, groups *discovery.TargetGroup) {
			sp.Sync(groups)
			wg.Done()
		}(m.scrapePools[setName], groups)

	}
	m.MtxScrape.Unlock()
	wg.Wait()
}

func (m *Manager) updateTsets(tsets map[string]*discovery.TargetGroup) {
	m.MtxScrape.Lock()
	m.targetSets = tsets
	m.MtxScrape.Unlock()
}
