package config

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration loaded from config.yaml.
type Config struct {
	TargetRegex     string   `yaml:"target_regex"`
	ExcludePatterns []string `yaml:"exclude_patterns"`
}

func Load(logger *slog.Logger) Config {
	var config Config
	data, err := os.ReadFile("config.yaml")
	if err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			logger.Warn("Failed to parse config.yaml", "error", err)
		}
	}
	return config
}
