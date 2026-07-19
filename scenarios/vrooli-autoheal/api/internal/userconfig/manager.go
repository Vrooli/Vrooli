package userconfig

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
)

// Manager handles loading, saving, and accessing configuration
type Manager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
	schemaPath string
	io         configIO
}

// NewManager creates a new configuration manager
// configPath is where the config file is stored (e.g., ~/.vrooli-autoheal/config.json)
// schemaPath is the path to the JSON schema file for validation
func NewManager(configPath, schemaPath string) *Manager {
	return newManagerWithIO(configPath, schemaPath, realConfigIO{})
}

func newManagerWithIO(configPath, schemaPath string, io configIO) *Manager {
	return &Manager{
		configPath: configPath,
		schemaPath: schemaPath,
		config:     DefaultConfig(),
		io:         io,
	}
}

// Load reads configuration from file, merging with defaults
// If the file doesn't exist, returns default configuration
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Start with defaults
	m.config = DefaultConfig()

	// Check if config file exists
	if _, err := m.io.Stat(m.configPath); isNotExistErr(err) {
		// No config file - use defaults (not an error)
		return nil
	}

	// Read and parse config file
	data, err := m.io.ReadFile(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	// Merge file config with defaults
	m.mergeConfig(&fileConfig)

	return nil
}

// Save writes the current configuration to file
func (m *Manager) Save() error {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := m.io.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write atomically (write to temp file, then rename)
	tempPath := m.configPath + ".tmp"
	if err := m.io.WriteFile(tempPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	if err := m.io.Rename(tempPath, m.configPath); err != nil {
		_ = m.io.Remove(tempPath) // Best-effort cleanup on rename failure
		return fmt.Errorf("failed to save config file: %w", err)
	}

	return nil
}

// Get returns a copy of the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return a copy to prevent external modification
	return m.copyConfig(m.config)
}

// Update replaces the configuration and saves to file
func (m *Manager) Update(config *Config) error {
	// Validate before updating
	result := m.Validate(config)
	if !result.Valid {
		return fmt.Errorf("invalid configuration: %v", result.Errors)
	}

	m.mu.Lock()
	m.config = m.copyConfig(config)
	m.mu.Unlock()

	return m.Save()
}

// GetCheck returns the effective configuration for a check
// This merges the user config with defaults
func (m *Manager) GetCheck(checkID string) CheckConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	defaults := GetCheckDefaults(checkID)
	result := CheckConfig{
		Enabled:         defaults.Enabled,
		AutoHeal:        defaults.AutoHeal,
		IntervalSeconds: defaults.IntervalSeconds,
		Settings: CheckSettings{
			AutoHealOn: defaults.AutoHealOn,
		},
	}
	if result.Settings.AutoHealOn == "" {
		result.Settings.AutoHealOn = "critical"
	}

	// Copy default thresholds if present
	if defaults.Thresholds != nil {
		result.Thresholds = *defaults.Thresholds
	}

	// Apply user overrides
	if check, ok := m.config.Checks[checkID]; ok {
		if check.Enabled != nil {
			result.Enabled = *check.Enabled
		}
		if check.AutoHeal != nil {
			result.AutoHeal = *check.AutoHeal
		}
		if check.IntervalSeconds != nil {
			result.IntervalSeconds = *check.IntervalSeconds
		}
		if check.Thresholds != nil {
			m.mergeThresholds(&result.Thresholds, check.Thresholds)
		}
		if check.Settings != nil {
			if check.Settings.TunnelTestURL != "" {
				result.Settings.TunnelTestURL = check.Settings.TunnelTestURL
			}
			if check.Settings.CleanPortsBeforeRestart != nil {
				result.Settings.CleanPortsBeforeRestart = check.Settings.CleanPortsBeforeRestart
			}
			if check.Settings.CaptureLogsOnFailure != nil {
				result.Settings.CaptureLogsOnFailure = check.Settings.CaptureLogsOnFailure
			}
			if check.Settings.LogLinesToCapture != nil {
				result.Settings.LogLinesToCapture = check.Settings.LogLinesToCapture
			}
			if check.Settings.AutoHealOn != "" {
				result.Settings.AutoHealOn = check.Settings.AutoHealOn
			}
		}
	}
	// Core runtime scenarios are the platform recovery floor. Their health
	// coverage cannot be disabled by stale or hand-edited user config; user
	// configuration remains additive for all non-core checks.
	if isMandatoryCoreScenarioCheck(checkID) {
		result.Enabled = true
		result.AutoHeal = true
		result.Settings.AutoHealOn = "critical"
	}

	return result
}

// CheckConfig is the resolved configuration for a check (with defaults applied)
type CheckConfig struct {
	Enabled         bool
	AutoHeal        bool
	IntervalSeconds int
	Thresholds      Thresholds
	Settings        CheckSettings
}

// IsCheckEnabled returns whether a check is enabled
func (m *Manager) IsCheckEnabled(checkID string) bool {
	return m.GetCheck(checkID).Enabled
}

// IsAutoHealEnabled returns whether auto-heal is enabled for a check
func (m *Manager) IsAutoHealEnabled(checkID string) bool {
	cfg := m.GetCheck(checkID)
	return cfg.Enabled && cfg.AutoHeal
}

// GetAutoHealOn returns the status threshold that triggers auto-heal.
// Valid values: "critical", "warning+critical".
func (m *Manager) GetAutoHealOn(checkID string) string {
	cfg := m.GetCheck(checkID)
	if cfg.Settings.AutoHealOn == "" {
		return "critical"
	}
	return cfg.Settings.AutoHealOn
}

// GetGlobal returns the global configuration
func (m *Manager) GetGlobal() GlobalConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Global
}

// GetUI returns the UI configuration
func (m *Manager) GetUI() UIConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.UI
}

// GetMonitoring returns the monitoring configuration
func (m *Manager) GetMonitoring() MonitoringConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Monitoring
}

// SetMonitoring replaces the monitoring configuration and saves
func (m *Manager) SetMonitoring(monitoring MonitoringConfig) error {
	m.mu.Lock()
	m.config.Monitoring = monitoring
	m.mu.Unlock()
	return m.Save()
}

// AddScenario adds a scenario to monitoring
func (m *Manager) AddScenario(name string, critical bool) error {
	m.mu.Lock()
	if m.config.Monitoring.Scenarios == nil {
		m.config.Monitoring.Scenarios = make(map[string]MonitoredScenario)
	}
	m.config.Monitoring.Scenarios[name] = MonitoredScenario{Critical: critical}
	m.mu.Unlock()
	return m.Save()
}

// RemoveScenario removes a scenario from monitoring
func (m *Manager) RemoveScenario(name string) error {
	m.mu.Lock()
	if m.config.Monitoring.Scenarios != nil {
		delete(m.config.Monitoring.Scenarios, name)
	}
	m.mu.Unlock()
	return m.Save()
}

// SetScenarioCritical sets the criticality of a monitored scenario
func (m *Manager) SetScenarioCritical(name string, critical bool) error {
	m.mu.Lock()
	if m.config.Monitoring.Scenarios == nil {
		m.config.Monitoring.Scenarios = make(map[string]MonitoredScenario)
	}
	m.config.Monitoring.Scenarios[name] = MonitoredScenario{Critical: critical}
	m.mu.Unlock()
	return m.Save()
}

// AddResource adds a resource to monitoring
func (m *Manager) AddResource(name string) error {
	m.mu.Lock()
	checkID := "resource-" + name
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	check := m.config.Checks[checkID]
	check.Enabled = boolPtr(true)
	m.config.Checks[checkID] = check

	exists := false
	for _, r := range m.config.Monitoring.Resources {
		if r == name {
			exists = true
			break
		}
	}
	if !exists {
		m.config.Monitoring.Resources = append(m.config.Monitoring.Resources, name)
	}
	m.mu.Unlock()
	return m.Save()
}

// RemoveResource removes a resource from monitoring
func (m *Manager) RemoveResource(name string) error {
	m.mu.Lock()
	checkID := "resource-" + name
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	check := m.config.Checks[checkID]
	check.Enabled = boolPtr(false)
	m.config.Checks[checkID] = check

	var filtered []string
	for _, r := range m.config.Monitoring.Resources {
		if r != name {
			filtered = append(filtered, r)
		}
	}
	m.config.Monitoring.Resources = filtered
	m.mu.Unlock()
	return m.Save()
}

// SetCheckEnabled enables or disables a check
func (m *Manager) SetCheckEnabled(checkID string, enabled bool) error {
	m.mu.Lock()
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	check := m.config.Checks[checkID]
	check.Enabled = boolPtr(enabled)
	m.config.Checks[checkID] = check
	m.mu.Unlock()

	return m.Save()
}

// SetCheckAutoHeal enables or disables auto-heal for a check
func (m *Manager) SetCheckAutoHeal(checkID string, autoHeal bool) error {
	m.mu.Lock()
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	check := m.config.Checks[checkID]
	check.AutoHeal = boolPtr(autoHeal)
	m.config.Checks[checkID] = check
	m.mu.Unlock()

	return m.Save()
}

// SetAllEnabled enables or disables all checks
func (m *Manager) SetAllEnabled(enabled bool) error {
	m.mu.Lock()
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	for checkID := range KnownCheckDefaults {
		check := m.config.Checks[checkID]
		check.Enabled = boolPtr(enabled)
		m.config.Checks[checkID] = check
	}
	m.mu.Unlock()

	return m.Save()
}

// SetAllAutoHeal enables or disables auto-heal for all checks
func (m *Manager) SetAllAutoHeal(autoHeal bool) error {
	m.mu.Lock()
	if m.config.Checks == nil {
		m.config.Checks = make(map[string]Check)
	}
	for checkID := range KnownCheckDefaults {
		check := m.config.Checks[checkID]
		check.AutoHeal = boolPtr(autoHeal)
		m.config.Checks[checkID] = check
	}
	m.mu.Unlock()

	return m.Save()
}

// Validate checks if a configuration is valid
func (m *Manager) Validate(config *Config) ValidationResult {
	var errors []ValidationError

	// Check version
	if config.Version != "1.0" {
		errors = append(errors, ValidationError{
			Path:    "version",
			Message: fmt.Sprintf("unsupported version '%s', expected '1.0'", config.Version),
		})
	}

	// Validate global config
	if config.Global.GracePeriodSeconds < 0 || config.Global.GracePeriodSeconds > 600 {
		errors = append(errors, ValidationError{
			Path:    "global.gracePeriodSeconds",
			Message: "must be between 0 and 600",
		})
	}
	if config.Global.TickIntervalSeconds < 10 || config.Global.TickIntervalSeconds > 3600 {
		errors = append(errors, ValidationError{
			Path:    "global.tickIntervalSeconds",
			Message: "must be between 10 and 3600",
		})
	}
	if config.Global.VerifyDelaySeconds < 5 || config.Global.VerifyDelaySeconds > 300 {
		errors = append(errors, ValidationError{
			Path:    "global.verifyDelaySeconds",
			Message: "must be between 5 and 300",
		})
	}
	if config.Global.MaxRestartAttempts < 1 || config.Global.MaxRestartAttempts > 10 {
		errors = append(errors, ValidationError{
			Path:    "global.maxRestartAttempts",
			Message: "must be between 1 and 10",
		})
	}
	if config.Global.RestartCooldownSeconds < 60 || config.Global.RestartCooldownSeconds > 3600 {
		errors = append(errors, ValidationError{
			Path:    "global.restartCooldownSeconds",
			Message: "must be between 60 and 3600",
		})
	}
	if config.Global.HistoryRetentionHours < 1 || config.Global.HistoryRetentionHours > 168 {
		errors = append(errors, ValidationError{
			Path:    "global.historyRetentionHours",
			Message: "must be between 1 and 168",
		})
	}
	if config.Global.ActionTimeoutFastSeconds < 5 || config.Global.ActionTimeoutFastSeconds > 120 {
		errors = append(errors, ValidationError{
			Path:    "global.actionTimeoutFastSeconds",
			Message: "must be between 5 and 120",
		})
	}
	if config.Global.ActionTimeoutRestartSeconds < 60 || config.Global.ActionTimeoutRestartSeconds > 1800 {
		errors = append(errors, ValidationError{
			Path:    "global.actionTimeoutRestartSeconds",
			Message: "must be between 60 and 1800",
		})
	}
	if config.Global.TimeoutRetrySeconds < 5 || config.Global.TimeoutRetrySeconds > 600 {
		errors = append(errors, ValidationError{
			Path:    "global.timeoutRetrySeconds",
			Message: "must be between 5 and 600",
		})
	}

	// Validate UI config
	if config.UI.AutoRefreshSeconds < 5 || config.UI.AutoRefreshSeconds > 300 {
		errors = append(errors, ValidationError{
			Path:    "ui.autoRefreshSeconds",
			Message: "must be between 5 and 300",
		})
	}
	validThemes := map[string]bool{"system": true, "light": true, "dark": true}
	if !validThemes[config.UI.Theme] {
		errors = append(errors, ValidationError{
			Path:    "ui.theme",
			Message: "must be 'system', 'light', or 'dark'",
		})
	}
	validTabs := map[string]bool{"dashboard": true, "trends": true, "docs": true}
	if !validTabs[config.UI.DefaultTab] {
		errors = append(errors, ValidationError{
			Path:    "ui.defaultTab",
			Message: "must be 'dashboard', 'trends', or 'docs'",
		})
	}

	// Validate check configs
	for checkID, check := range config.Checks {
		if check.IntervalSeconds != nil {
			if *check.IntervalSeconds < 10 || *check.IntervalSeconds > 86400 {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("checks.%s.intervalSeconds", checkID),
					Message: "must be between 10 and 86400",
				})
			}
		}
		if check.Thresholds != nil {
			if check.Thresholds.WarningPercent != nil {
				if *check.Thresholds.WarningPercent < 0 || *check.Thresholds.WarningPercent > 100 {
					errors = append(errors, ValidationError{
						Path:    fmt.Sprintf("checks.%s.thresholds.warningPercent", checkID),
						Message: "must be between 0 and 100",
					})
				}
			}
			if check.Thresholds.CriticalPercent != nil {
				if *check.Thresholds.CriticalPercent < 0 || *check.Thresholds.CriticalPercent > 100 {
					errors = append(errors, ValidationError{
						Path:    fmt.Sprintf("checks.%s.thresholds.criticalPercent", checkID),
						Message: "must be between 0 and 100",
					})
				}
			}
		}
		if check.Settings != nil && check.Settings.AutoHealOn != "" {
			if check.Settings.AutoHealOn != "critical" && check.Settings.AutoHealOn != "warning+critical" {
				errors = append(errors, ValidationError{
					Path:    fmt.Sprintf("checks.%s.settings.autoHealOn", checkID),
					Message: "must be 'critical' or 'warning+critical'",
				})
			}
		}
	}

	return ValidationResult{
		Valid:  len(errors) == 0,
		Errors: errors,
	}
}

// GetSchema returns the JSON schema as a string
func (m *Manager) GetSchema() (string, error) {
	data, err := m.io.ReadFile(m.schemaPath)
	if err != nil {
		return "", fmt.Errorf("failed to read schema file: %w", err)
	}
	return string(data), nil
}

// Import loads configuration from JSON data
func (m *Manager) Import(data []byte) (*Config, error) {
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	result := m.Validate(&config)
	if !result.Valid {
		return nil, fmt.Errorf("validation failed: %v", result.Errors)
	}

	return &config, nil
}

// Export returns the current configuration as JSON
func (m *Manager) Export() ([]byte, error) {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	return json.MarshalIndent(config, "", "  ")
}

// mergeConfig merges file config into the current config
func (m *Manager) mergeConfig(file *Config) {
	// Version
	if file.Version != "" {
		m.config.Version = file.Version
	}

	// Global config - only override non-zero values
	if file.Global.GracePeriodSeconds != 0 {
		m.config.Global.GracePeriodSeconds = file.Global.GracePeriodSeconds
	}
	if file.Global.TickIntervalSeconds != 0 {
		m.config.Global.TickIntervalSeconds = file.Global.TickIntervalSeconds
	}
	if file.Global.VerifyDelaySeconds != 0 {
		m.config.Global.VerifyDelaySeconds = file.Global.VerifyDelaySeconds
	}
	if file.Global.MaxRestartAttempts != 0 {
		m.config.Global.MaxRestartAttempts = file.Global.MaxRestartAttempts
	}
	if file.Global.RestartCooldownSeconds != 0 {
		m.config.Global.RestartCooldownSeconds = file.Global.RestartCooldownSeconds
	}
	if file.Global.HistoryRetentionHours != 0 {
		m.config.Global.HistoryRetentionHours = file.Global.HistoryRetentionHours
	}
	if file.Global.ActionTimeoutFastSeconds != 0 {
		m.config.Global.ActionTimeoutFastSeconds = file.Global.ActionTimeoutFastSeconds
	}
	if file.Global.ActionTimeoutRestartSeconds != 0 {
		m.config.Global.ActionTimeoutRestartSeconds = file.Global.ActionTimeoutRestartSeconds
	}
	if file.Global.TimeoutRetrySeconds != 0 {
		m.config.Global.TimeoutRetrySeconds = file.Global.TimeoutRetrySeconds
	}

	// Checks - merge each check
	if file.Checks != nil {
		if m.config.Checks == nil {
			m.config.Checks = make(map[string]Check)
		}
		for id, check := range file.Checks {
			m.config.Checks[id] = check
		}
	}

	// UI config
	if file.UI.AutoRefreshSeconds != 0 {
		m.config.UI.AutoRefreshSeconds = file.UI.AutoRefreshSeconds
	}
	if file.UI.Theme != "" {
		m.config.UI.Theme = file.UI.Theme
	}
	m.config.UI.ShowDisabledChecks = file.UI.ShowDisabledChecks
	if file.UI.DefaultTab != "" {
		m.config.UI.DefaultTab = file.UI.DefaultTab
	}

	// Monitoring config starts from defaults so new default checks are adopted
	// by older saved configs. Explicit disabled check entries still suppress
	// default resources, which preserves intentional removals.
	if file.Monitoring.Scenarios != nil || file.Monitoring.Resources != nil {
		m.config.Monitoring = m.mergeMonitoringConfig(file.Monitoring)
	}
}

func (m *Manager) mergeMonitoringConfig(file MonitoringConfig) MonitoringConfig {
	merged := m.config.Monitoring

	if file.Scenarios != nil {
		if merged.Scenarios == nil {
			merged.Scenarios = make(map[string]MonitoredScenario)
		}
		for name, cfg := range file.Scenarios {
			merged.Scenarios[name] = cfg
		}
	}

	if file.Resources != nil {
		merged.Resources = append([]string(nil), file.Resources...)
		for _, resource := range DefaultMonitoring().Resources {
			checkID := "resource-" + resource
			if check, ok := m.config.Checks[checkID]; ok && check.Enabled != nil && !*check.Enabled {
				merged.Resources = removeString(merged.Resources, resource)
				continue
			}
			if !containsString(merged.Resources, resource) {
				merged.Resources = append(merged.Resources, resource)
			}
		}
	}
	ensureMandatoryCoreScenarios(&merged)

	return merged
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func removeString(items []string, remove string) []string {
	filtered := items[:0]
	for _, item := range items {
		if item != remove {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// mergeThresholds merges user threshold overrides into defaults
func (m *Manager) mergeThresholds(dest *Thresholds, src *Thresholds) {
	if src.WarningPercent != nil {
		dest.WarningPercent = src.WarningPercent
	}
	if src.CriticalPercent != nil {
		dest.CriticalPercent = src.CriticalPercent
	}
	if src.WarningCount != nil {
		dest.WarningCount = src.WarningCount
	}
	if src.CriticalCount != nil {
		dest.CriticalCount = src.CriticalCount
	}
	if src.Partitions != nil {
		dest.Partitions = src.Partitions
	}
}

// copyConfig creates a deep copy of a config
func (m *Manager) copyConfig(src *Config) *Config {
	// Use JSON marshal/unmarshal for deep copy (simple and correct)
	data, err := json.Marshal(src)
	if err != nil {
		// Fallback to a shallow copy if serialization fails.
		copy := *src
		return &copy
	}

	var dest Config
	if err := json.Unmarshal(data, &dest); err != nil {
		// Fallback to a shallow copy if deserialization fails.
		copy := *src
		return &copy
	}
	return &dest
}

// GetConfigPath returns the path to the config file
func (m *Manager) GetConfigPath() string {
	return m.configPath
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	return defaultConfigPathWithIO(realConfigIO{})
}

func defaultConfigPathWithIO(io configIO) string {
	home, err := io.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".vrooli-autoheal", "config.json")
}
