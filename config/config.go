package config

import (
	"net/url"
	"time"
)

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName string `yaml:"job_name"`
	// A set of query parameters with which the target is scraped.
	Params url.Values `yaml:"params,omitempty"`
	// How frequently to scrape the targets of this scrape config.
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	// The timeout for scraping targets of this config.
	ScrapeTimeout time.Duration `yaml:"scrape_timeout,omitempty"`
	// The HTTP resource path on which to fetch metrics from targets.
	MetricsPath string `yaml:"metrics_path,omitempty"`

	// We cannot do proper Go type embedding below as the parser will then parse
	// values arbitrarily into the overflow maps of further-down types.
	//ServiceDiscoveryConfigs discovery.Configs       `yaml:"-"`
	// List of target relabel configurations.
	//RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// List of metric relabel configurations.
	//MetricRelabelConfigs []*relabel.Config `yaml:"metric_relabel_configs,omitempty"`
}
