package scrape

import (
	"context"
	"fmt"
	"github.com/anathantu/monitor/config"
	"github.com/anathantu/monitor/discovery"
	"github.com/go-kit/log"
	"net/http"
	"sync"
	"time"
)

// scrapePool manages scrapes for sets of targets.
type scrapePool struct {
	logger log.Logger

	// mtx must not be taken after targetMtx.
	mtx    sync.Mutex
	config *config.ScrapeConfig
	client *http.Client

	targetMtx sync.Mutex
	loops     []loop
}

func newScrapePool(cfg *config.ScrapeConfig, logger log.Logger) (*scrapePool, error) {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	client := &http.Client{}

	sp := &scrapePool{
		config: cfg,
		client: client,
		logger: logger,
		loops:  make([]loop, 16),
	}

	return sp, nil
}

// Sync 转换成可以直接执行的scrapeLoop
func (sp *scrapePool) Sync(tgs *discovery.TargetGroup) {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()

	sp.targetMtx.Lock()
	//转换成loop
	for _, tg := range tgs.Targets {
		sl := newScrapeLoop(nil, nil, sp.config.ScrapeInterval, sp.config.ScrapeTimeout, sp.client, tg)
		sp.loops = append(sp.loops, sl)
	}
	sp.targetMtx.Unlock()
	for _, loop := range sp.loops {
		go loop.run(nil)
	}
}

// A loop can run and be stopped again. It must not be reused after it was stopped.
type loop interface {
	run(errc chan<- error)
	setForcedError(err error)
	stop()
}

type scrapeLoop struct {
	scraper         Scraper
	l               log.Logger
	honorTimestamps bool
	forcedErr       error
	forcedErrMtx    sync.Mutex
	interval        time.Duration
	timeout         time.Duration

	parentCtx context.Context
	ctx       context.Context
	cancel    func()
	stopped   chan struct{}
}

type Scraper struct {
	client *http.Client
	url    string
}

func newScrapeLoop(ctx context.Context,
	l log.Logger,
	interval time.Duration,
	timeout time.Duration,
	client *http.Client,
	url string,
) *scrapeLoop {
	if l == nil {
		l = log.NewNopLogger()
	}
	sl := &scrapeLoop{
		stopped:   make(chan struct{}),
		l:         l,
		parentCtx: ctx,
		interval:  interval,
		timeout:   timeout,
		scraper:   Scraper{client: client, url: url},
	}
	sl.ctx, sl.cancel = context.WithCancel(ctx)

	return sl
}

func (sl *scrapeLoop) run(errc chan<- error) {
	select {
	//case <-time.After(sl.scraper.offset(sl.interval, sl.jitterSeed)):
	// Continue after a scraping offset.
	case <-time.After(sl.interval):
	case <-sl.ctx.Done():
		close(sl.stopped)
		return
	}

	var last time.Time

	ticker := time.NewTicker(sl.interval)
	defer ticker.Stop()

mainLoop:
	for {
		select {
		case <-sl.parentCtx.Done():
			close(sl.stopped)
			return
		case <-sl.ctx.Done():
			break mainLoop
		default:
		}

		scrapeTime := time.Now().Round(0)

		last = sl.scrapeAndReport(sl.interval, sl.timeout, last, scrapeTime, errc)

		select {
		case <-sl.parentCtx.Done():
			close(sl.stopped)
			return
		case <-sl.ctx.Done():
			break mainLoop
		case <-ticker.C:
		}
	}

	close(sl.stopped)
}

// scrapeAndReport performs a scrape and then appends the result to the storage
// together with reporting metrics, by using as few appenders as possible.
// In the happy scenario, a single appender is used.
// This function uses sl.parentCtx instead of sl.ctx on purpose. A scrape should
// only be cancelled on shutdown, not on reloads.
func (sl *scrapeLoop) scrapeAndReport(interval, timeout time.Duration, last, appendTime time.Time, errc chan<- error) time.Time {
	start := time.Now()
	c := sl.scraper.client
	u := sl.scraper.url
	req, _ := http.NewRequest("GET", u, nil)
	resp, err := c.Do(req)
	if err != nil {
		return start
	}
	fmt.Println(resp.Body)
	return start
}

func (sl *scrapeLoop) setForcedError(err error) {
	sl.forcedErrMtx.Lock()
	defer sl.forcedErrMtx.Unlock()
	sl.forcedErr = err
}

// Stop the scraping. May still write data and stale markers after it has
// returned. Cancel the context to stop all writes.
func (sl *scrapeLoop) stop() {
	sl.cancel()
	<-sl.stopped
}
