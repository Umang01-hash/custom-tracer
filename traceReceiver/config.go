package traceReceiver

import (
	"fmt"
	"strconv"
)

// Config represents the receiver config settings within the collector's config.yaml
type Config struct {
	Port           string `mapstructure:"port"`
	NumberOfTraces int    `mapstructure:"number_of_traces"`
}

// Validate checks if the receiver configuration is valid
func (cfg *Config) Validate() error {
	_, err := strconv.Atoi(cfg.Port)
	if err != nil {
		return fmt.Errorf("failed to parse port : %w", err)
	}

	if cfg.NumberOfTraces < 1 {
		return fmt.Errorf("number_of_traces must be greater or equal to 1")
	}
	return nil
}
