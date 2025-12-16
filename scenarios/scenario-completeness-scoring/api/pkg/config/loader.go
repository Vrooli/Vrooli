package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// GlobalConfigFile is the filename for global configuration
	GlobalConfigFile = "scoring-config.json"
	// VrooliConfigDir is the directory name for Vrooli configuration
	VrooliConfigDir = ".vrooli"
)

// Loader manages configuration loading and saving.
// [REQ:SCS-CFG-004] Configuration persistence
type Loader struct {
	vrooliRoot string
	mu         sync.RWMutex
	global     *ScoringConfig
}

// NewLoader creates a new configuration loader.
func NewLoader(vrooliRoot string) *Loader {
	return &Loader{vrooliRoot: vrooliRoot}
}

// globalConfigPath returns the path to the global config file.
// ASSUMPTION: User home directory is available
// HARDENED: Falls back to /tmp if home dir lookup fails
func (l *Loader) globalConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, VrooliConfigDir, GlobalConfigFile)
}

// LoadGlobal loads the global configuration from disk.
// [REQ:SCS-CFG-004] Configuration persistence - load
func (l *Loader) LoadGlobal() (*ScoringConfig, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.globalConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		l.global = &cfg
		return &cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config: %w", err)
	}

	var cfg ScoringConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse global config: %w", err)
	}

	l.global = &cfg
	return &cfg, nil
}

// SaveGlobal saves the global configuration to disk.
// [REQ:SCS-CFG-004] Configuration persistence - save
func (l *Loader) SaveGlobal(cfg *ScoringConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.globalConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write global config: %w", err)
	}

	l.global = cfg
	return nil
}

// ClearCache clears the in-memory configuration cache.
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.global = nil
}

// ValidateConfig validates a configuration for correctness.
func ValidateConfig(cfg *ScoringConfig) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	for name, w := range map[string]int{
		"quality":  cfg.Weights.Quality,
		"coverage": cfg.Weights.Coverage,
		"quantity": cfg.Weights.Quantity,
		"ui":       cfg.Weights.UI,
	} {
		if w < 0 || w > 100 {
			return fmt.Errorf("%s weight must be between 0 and 100, got %d", name, w)
		}
	}

	totalWeight := cfg.Weights.TotalWeight()
	if totalWeight != 100 {
		return fmt.Errorf("weights must sum to 100, got %d", totalWeight)
	}

	anyEnabled := false
	if cfg.Components.Quality.Enabled {
		anyEnabled = true
		if !cfg.Components.Quality.RequirementPassRate && !cfg.Components.Quality.TargetPassRate && !cfg.Components.Quality.TestPassRate {
			return fmt.Errorf("quality is enabled but no quality sub-metrics are enabled")
		}
	}
	if cfg.Components.Coverage.Enabled {
		anyEnabled = true
		if !cfg.Components.Coverage.TestCoverageRatio && !cfg.Components.Coverage.RequirementDepth {
			return fmt.Errorf("coverage is enabled but no coverage sub-metrics are enabled")
		}
	}
	if cfg.Components.Quantity.Enabled {
		anyEnabled = true
		if !cfg.Components.Quantity.Requirements && !cfg.Components.Quantity.Targets && !cfg.Components.Quantity.Tests {
			return fmt.Errorf("quantity is enabled but no quantity sub-metrics are enabled")
		}
	}
	if cfg.Components.UI.Enabled {
		anyEnabled = true
		if !cfg.Components.UI.TemplateDetection && !cfg.Components.UI.ComponentComplexity && !cfg.Components.UI.APIIntegration && !cfg.Components.UI.Routing && !cfg.Components.UI.CodeVolume {
			return fmt.Errorf("ui is enabled but no ui sub-metrics are enabled")
		}
	}
	if !anyEnabled {
		return fmt.Errorf("at least one scoring dimension must be enabled")
	}

	if cfg.Penalties.Enabled {
		if !cfg.Penalties.InsufficientTestCoverage &&
			!cfg.Penalties.InvalidTestLocation &&
			!cfg.Penalties.MonolithicTestFiles &&
			!cfg.Penalties.SingleLayerValidation &&
			!cfg.Penalties.TargetMappingRatio &&
			!cfg.Penalties.SuperficialTestImplementation &&
			!cfg.Penalties.ManualValidations {
			return fmt.Errorf("penalties are enabled but no penalty types are enabled")
		}
	}

	if cfg.CircuitBreaker.Enabled {
		if cfg.CircuitBreaker.FailThreshold < 1 {
			return fmt.Errorf("fail_threshold must be at least 1")
		}
		if cfg.CircuitBreaker.RetryIntervalS < 1 {
			return fmt.Errorf("retry_interval_seconds must be at least 1")
		}
	}

	return nil
}

