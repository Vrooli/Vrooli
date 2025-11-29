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

// Loader manages configuration loading and saving
// [REQ:SCS-CFG-004] Configuration persistence
type Loader struct {
	vrooliRoot string
	mu         sync.RWMutex
	global     *ScoringConfig
	overrides  map[string]*ScenarioOverride
}

// NewLoader creates a new configuration loader
func NewLoader(vrooliRoot string) *Loader {
	return &Loader{
		vrooliRoot: vrooliRoot,
		overrides:  make(map[string]*ScenarioOverride),
	}
}

// globalConfigPath returns the path to the global config file
// ASSUMPTION: User home directory is available
// HARDENED: Falls back to /tmp if home dir lookup fails
func (l *Loader) globalConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		// Fall back to a reasonable default directory
		// This can happen in containerized environments or restricted shells
		homeDir = "/tmp"
	}
	return filepath.Join(homeDir, VrooliConfigDir, GlobalConfigFile)
}

// scenarioConfigPath returns the path to a scenario-specific config file
func (l *Loader) scenarioConfigPath(scenario string) string {
	return filepath.Join(l.vrooliRoot, "scenarios", scenario, VrooliConfigDir, GlobalConfigFile)
}

// LoadGlobal loads the global configuration from disk
// [REQ:SCS-CFG-004] Configuration persistence - load
func (l *Loader) LoadGlobal() (*ScoringConfig, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.globalConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
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

// SaveGlobal saves the global configuration to disk
// [REQ:SCS-CFG-004] Configuration persistence - save
func (l *Loader) SaveGlobal(cfg *ScoringConfig) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.globalConfigPath()

	// Ensure directory exists
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

// LoadScenarioOverride loads scenario-specific overrides
// [REQ:SCS-CFG-002] Per-scenario overrides
func (l *Loader) LoadScenarioOverride(scenario string) (*ScenarioOverride, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check cache first
	if override, ok := l.overrides[scenario]; ok {
		return override, nil
	}

	configPath := l.scenarioConfigPath(scenario)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return nil override if file doesn't exist
		return nil, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenario config: %w", err)
	}

	var override ScenarioOverride
	if err := json.Unmarshal(data, &override); err != nil {
		return nil, fmt.Errorf("failed to parse scenario config: %w", err)
	}

	override.Scenario = scenario
	l.overrides[scenario] = &override
	return &override, nil
}

// SaveScenarioOverride saves scenario-specific overrides
// [REQ:SCS-CFG-002] Per-scenario overrides
func (l *Loader) SaveScenarioOverride(override *ScenarioOverride) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.scenarioConfigPath(override.Scenario)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(override, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal override: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write scenario config: %w", err)
	}

	l.overrides[override.Scenario] = override
	return nil
}

// DeleteScenarioOverride removes scenario-specific overrides
func (l *Loader) DeleteScenarioOverride(scenario string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	configPath := l.scenarioConfigPath(scenario)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Already doesn't exist
		delete(l.overrides, scenario)
		return nil
	}

	if err := os.Remove(configPath); err != nil {
		return fmt.Errorf("failed to delete scenario config: %w", err)
	}

	delete(l.overrides, scenario)
	return nil
}

// GetEffectiveConfig returns the merged config for a scenario
// [REQ:SCS-CFG-002] Per-scenario overrides with proper precedence
func (l *Loader) GetEffectiveConfig(scenario string) (*ScoringConfig, error) {
	// Load global config
	global, err := l.LoadGlobal()
	if err != nil {
		return nil, err
	}

	// Load scenario override
	override, err := l.LoadScenarioOverride(scenario)
	if err != nil {
		return nil, err
	}

	// If no override, return global
	if override == nil || override.Overrides == nil {
		return global, nil
	}

	// Merge configs (scenario > global > defaults)
	merged := global.Merge(override.Overrides)
	return &merged, nil
}

// ListScenarioOverrides lists all scenarios with overrides
func (l *Loader) ListScenarioOverrides() ([]string, error) {
	scenariosDir := filepath.Join(l.vrooliRoot, "scenarios")

	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read scenarios directory: %w", err)
	}

	var scenarios []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		configPath := l.scenarioConfigPath(entry.Name())
		if _, err := os.Stat(configPath); err == nil {
			scenarios = append(scenarios, entry.Name())
		}
	}

	return scenarios, nil
}

// ClearCache clears the in-memory configuration cache
func (l *Loader) ClearCache() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.global = nil
	l.overrides = make(map[string]*ScenarioOverride)
}

// ValidateConfig validates a configuration for correctness
func ValidateConfig(cfg *ScoringConfig) error {
	// Validate weights sum to 100
	totalWeight := cfg.Weights.TotalWeight()
	if totalWeight != 100 {
		return fmt.Errorf("weights must sum to 100, got %d", totalWeight)
	}

	// Validate circuit breaker settings
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
