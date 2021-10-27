package config

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"time"
)

// Config is the top-level configuration for Prometheus's config files.
type Config struct {
	GlobalConfig GlobalConfig `yaml:"global"`
	//AlertingConfig AlertingConfig  `yaml:"alerting,omitempty"`
	//RuleFiles      []string        `yaml:"rule_files,omitempty"`
	ScrapeConfigs []*ScrapeConfig `yaml:"scrape_configs,omitempty"`
}

func (c Config) String() string {
	b, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Sprintf("<error creating config string: %s>", err)
	}
	return string(b)
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultConfig
	// We want to set c to the defaults and then overwrite it with the input.
	// To make unmarshal fill the plain data struct rather than calling UnmarshalYAML
	// again, we have to hide it using a type indirection.
	type plain Config
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	// If a global block was open but empty the default global config is overwritten.
	// We have to restore it here.
	if c.GlobalConfig.isZero() {
		c.GlobalConfig = DefaultGlobalConfig
	}

	// Do global overrides and validate unique names.
	jobNames := map[string]struct{}{}
	for _, scfg := range c.ScrapeConfigs {
		if scfg == nil {
			return errors.New("empty or null scrape config section")
		}
		// First set the correct scrape interval, then check that the timeout
		// (inferred or explicit) is not greater than that.
		if scfg.ScrapeInterval == 0 {
			scfg.ScrapeInterval = c.GlobalConfig.ScrapeInterval
		}
		if scfg.ScrapeTimeout > scfg.ScrapeInterval {
			return errors.Errorf("scrape timeout greater than scrape interval for scrape config with job name %q", scfg.JobName)
		}
		if scfg.ScrapeTimeout == 0 {
			if c.GlobalConfig.ScrapeTimeout > scfg.ScrapeInterval {
				scfg.ScrapeTimeout = scfg.ScrapeInterval
			} else {
				scfg.ScrapeTimeout = c.GlobalConfig.ScrapeTimeout
			}
		}

		if _, ok := jobNames[scfg.JobName]; ok {
			return errors.Errorf("found multiple scrape configs with job name %q", scfg.JobName)
		}
		jobNames[scfg.JobName] = struct{}{}
	}
	return nil
}

// Load parses the YAML input s into a Config.
func Load(s string, logger log.Logger) (*Config, error) {
	cfg := &Config{}
	// If the entire config body is empty the UnmarshalYAML method is
	// never called. We thus have to set the DefaultConfig at the entry
	// point as well.
	*cfg = DefaultConfig

	err := yaml.UnmarshalStrict([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// LoadFile parses the given YAML file into a Config.
func LoadFile(filename string, logger log.Logger) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(string(content), logger)
	if err != nil {
		return nil, errors.Wrapf(err, "parsing YAML file %s", filename)
	}
	return cfg, nil
}

// The defaults applied before parsing the respective config sections.
var (
	// DefaultConfig is the default top-level configuration.
	DefaultConfig = Config{
		GlobalConfig: DefaultGlobalConfig,
	}

	// DefaultGlobalConfig is the default global configuration.
	DefaultGlobalConfig = GlobalConfig{
		ScrapeInterval:     time.Duration(1 * time.Minute),
		ScrapeTimeout:      time.Duration(10 * time.Second),
		EvaluationInterval: time.Duration(1 * time.Minute),
	}

	// DefaultScrapeConfig is the default scrape configuration.
	DefaultScrapeConfig = ScrapeConfig{
		// ScrapeTimeout and ScrapeInterval default to the
		// configured globals.
		MetricsPath: "/metrics",
	}
)

// GlobalConfig configures values that are used across other configuration
// objects.
type GlobalConfig struct {
	// How frequently to scrape targets by default.
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	// The default timeout when scraping targets.
	ScrapeTimeout time.Duration `yaml:"scrape_timeout,omitempty"`
	// How frequently to evaluate rules by default.
	EvaluationInterval time.Duration `yaml:"evaluation_interval,omitempty"`
}

// isZero returns true iff the global config is the zero value.
func (c *GlobalConfig) isZero() bool {
	return c.ScrapeInterval == 0 &&
		c.ScrapeTimeout == 0 &&
		c.EvaluationInterval == 0
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *GlobalConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Create a clean global config as the previous one was already populated
	// by the default due to the YAML parser behavior for empty blocks.
	gc := &GlobalConfig{}
	type plain GlobalConfig
	if err := unmarshal((*plain)(gc)); err != nil {
		return err
	}

	// First set the correct scrape interval, then check that the timeout
	// (inferred or explicit) is not greater than that.
	if gc.ScrapeInterval == 0 {
		gc.ScrapeInterval = DefaultGlobalConfig.ScrapeInterval
	}
	if gc.ScrapeTimeout > gc.ScrapeInterval {
		return errors.New("global scrape timeout greater than scrape interval")
	}
	if gc.ScrapeTimeout == 0 {
		if DefaultGlobalConfig.ScrapeTimeout > gc.ScrapeInterval {
			gc.ScrapeTimeout = gc.ScrapeInterval
		} else {
			gc.ScrapeTimeout = DefaultGlobalConfig.ScrapeTimeout
		}
	}
	if gc.EvaluationInterval == 0 {
		gc.EvaluationInterval = DefaultGlobalConfig.EvaluationInterval
	}
	*c = *gc
	return nil
}

// ScrapeConfig configures a scraping unit for Prometheus.
type ScrapeConfig struct {
	// The job name to which the job label is set by default.
	JobName string `yaml:"job_name"`
	// How frequently to scrape the targets of this scrape config.
	ScrapeInterval time.Duration `yaml:"scrape_interval,omitempty"`
	// The timeout for scraping targets of this config.
	ScrapeTimeout time.Duration `yaml:"scrape_timeout,omitempty"`
	// The HTTP resource path on which to fetch metrics from targets.
	MetricsPath string `yaml:"metrics_path,omitempty"`

	// We cannot do proper Go type embedding below as the parser will then parse
	// values arbitrarily into the overflow maps of further-down types.
	StaticConfig StaticConfig `yaml:"static_configs,omitempty"`
	// List of target relabel configurations.
	//RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// List of metric relabel configurations.
	//MetricRelabelConfigs []*relabel.Config `yaml:"metric_relabel_configs,omitempty"`
}

type StaticConfig struct {
	targets []string `yaml:"targets,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
//func (c *ScrapeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
//	*c = DefaultScrapeConfig
//	if err := discovery.UnmarshalYAMLWithInlineConfigs(c, unmarshal); err != nil {
//		return err
//	}
//	if len(c.JobName) == 0 {
//		return errors.New("job_name is empty")
//	}
//
//	return nil
//}
