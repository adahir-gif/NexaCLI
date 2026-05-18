// Package config handles loading and parsing of nexacli.yaml configuration files.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Provider represents a supported cloud deployment provider.
type Provider string

const (
	ProviderAWS     Provider = "aws"
	ProviderGCP     Provider = "gcp"
	ProviderAzure   Provider = "azure"
	ProviderFlyIO   Provider = "fly"
	ProviderRailway Provider = "railway"
	ProviderRender  Provider = "render"
)

// Region defines a deployment region for multi-region support.
type Region struct {
	Name     string `yaml:"name"`
	Primary  bool   `yaml:"primary,omitempty"`
	Replicas int    `yaml:"replicas,omitempty"`
}

// BuildConfig holds build-time configuration.
type BuildConfig struct {
	Command    string            `yaml:"command,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty"`
	Context    string            `yaml:"context,omitempty"`
	Args       map[string]string `yaml:"args,omitempty"`
}

// HealthCheck defines the application health check configuration.
type HealthCheck struct {
	Path     string `yaml:"path,omitempty"`
	Interval int    `yaml:"interval,omitempty"` // seconds
	Timeout  int    `yaml:"timeout,omitempty"`  // seconds
	Retries  int    `yaml:"retries,omitempty"`
}

// ScalingConfig controls autoscaling behavior.
type ScalingConfig struct {
	Min    int `yaml:"min,omitempty"`
	Max    int `yaml:"max,omitempty"`
	Target int `yaml:"target,omitempty"` // CPU % target
}

// Config is the top-level nexacli.yaml configuration structure.
type Config struct {
	Version     string            `yaml:"version"`
	App         string            `yaml:"app"`
	Provider    Provider          `yaml:"provider"`
	Regions     []Region          `yaml:"regions,omitempty"`
	Build       BuildConfig       `yaml:"build,omitempty"`
	HealthCheck HealthCheck       `yaml:"healthcheck,omitempty"`
	Scaling     ScalingConfig     `yaml:"scaling,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig(appName string, provider Provider) *Config {
	return &Config{
		Version:  "1",
		App:      appName,
		Provider: provider,
		Regions: []Region{
			{Name: "us-east-1", Primary: true},
		},
		Build: BuildConfig{
			Dockerfile: "Dockerfile",
			Context:    ".",
		},
		HealthCheck: HealthCheck{
			Path:     "/health",
			Interval: 30,
			Timeout:  10,
			Retries:  3,
		},
		Scaling: ScalingConfig{
			Min:    1,
			Max:    3,
			Target: 70,
		},
	}
}

// Load reads and parses a nexacli.yaml file from the given path.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("nexacli.yaml not found in current directory; run `nexacli init` to create one")
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing nexacli.yaml: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes the config to a YAML file at the given path.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}
	header := []byte("# nexacli.yaml — NexaCLI deployment configuration\n# See https://github.com/adahir-gif/NexaCLI for full schema reference.\n\n")
	return os.WriteFile(path, append(header, data...), 0644)
}

// Validate checks that required fields are set and values are valid.
func (c *Config) Validate() error {
	if c.App == "" {
		return fmt.Errorf("config: 'app' field is required")
	}
	if c.Provider == "" {
		return fmt.Errorf("config: 'provider' field is required")
	}
	validProviders := map[Provider]bool{
		ProviderAWS:     true,
		ProviderGCP:     true,
		ProviderAzure:   true,
		ProviderFlyIO:   true,
		ProviderRailway: true,
		ProviderRender:  true,
	}
	if !validProviders[c.Provider] {
		return fmt.Errorf("config: unsupported provider %q; valid: aws, gcp, azure, fly, railway, render", c.Provider)
	}
	return nil
}
