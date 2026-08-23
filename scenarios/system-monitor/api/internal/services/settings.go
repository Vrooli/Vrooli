package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Settings represents the system monitor settings
type Settings struct {
	// Monitor activation status
	Active bool `json:"active"`

	// Monitoring intervals (in seconds)
	MetricCollectionInterval int `json:"metric_collection_interval"`
	AnomalyDetectionInterval int `json:"anomaly_detection_interval"`
	ThresholdCheckInterval   int `json:"threshold_check_interval"`

	// Investigation settings
	CooldownPeriodSeconds int `json:"cooldown_period_seconds"`

	// System thresholds
	CPUThreshold                 float64 `json:"cpu_threshold"`
	CPUHighPercent               float64 `json:"cpu_high_percent"`
	CPUCriticalPercent           float64 `json:"cpu_critical_percent"`
	CPUEscalationCooldownSeconds int     `json:"cpu_escalation_cooldown_seconds"`
	CPUEscalationDebounceTicks   int     `json:"cpu_escalation_debounce_ticks"`
	CPUSustainedWindowTicks      int     `json:"cpu_sustained_window_ticks"`
	CPUPressureThreshold         float64 `json:"cpu_pressure_threshold"`
	MemoryThreshold              float64 `json:"memory_threshold"`

	// DiskThreshold is the warning band boundary: the usage percentage at
	// which disk pressure starts being recorded. It is deliberately the same
	// setting the band model calls "warning" rather than a second setting
	// meaning the same thing — two names for one boundary is how a
	// configuration surface drifts away from the code that reads it.
	DiskThreshold float64 `json:"disk_threshold"`

	// Disk-pressure escalation bands, in percent used. Bands must ascend:
	// DiskThreshold (warning) < DiskHighPercent < DiskCriticalPercent.
	// warning  — record the pressure, take no action.
	// high     — request a cleanup preview; still no deletion.
	// critical — safe-tier reclamation may run with no operator present.
	DiskHighPercent     float64 `json:"disk_high_percent"`
	DiskCriticalPercent float64 `json:"disk_critical_percent"`

	// DiskEscalationCooldownSeconds is the minimum gap between two records
	// for the same band. Without it a disk parked above a boundary emits one
	// record per tick, which is how alerting becomes noise an operator learns
	// to ignore.
	DiskEscalationCooldownSeconds int `json:"disk_escalation_cooldown_seconds"`

	// DiskEscalationDebounceTicks is how many consecutive observations a new
	// band needs before it takes effect, so a single noisy sample cannot
	// escalate on its own.
	DiskEscalationDebounceTicks int `json:"disk_escalation_debounce_ticks"`

	// DiskFastFillJumpPercent bounds the debounce delay. A rise of at least
	// this many percentage points in one tick escalates immediately. The
	// incident's own growth was 3-5 GB per day, but a runaway process can
	// consume 100 GB in minutes, and debounce must not stall that response.
	DiskFastFillJumpPercent float64 `json:"disk_fast_fill_jump_percent"`

	// Metrics lifecycle
	MetricsRetentionDays          int  `json:"metrics_retention_days"`
	RetentionCheckIntervalSeconds int  `json:"retention_check_interval_seconds"`
	RetentionRunOnStartup         bool `json:"retention_run_on_startup"`
	CompactAfterRetention         bool `json:"compact_after_retention"`
}

// settingsFile is the canonical on-disk representation of the settings file.
// Settings are always persisted and loaded through this wrapper shape.
type settingsFile struct {
	Version  string                 `json:"version"`
	Metadata map[string]interface{} `json:"metadata"`
	Settings Settings               `json:"settings"`
}

// SettingsManager manages system monitor settings with thread safety
type SettingsManager struct {
	settings          Settings
	mutex             sync.RWMutex
	clock             Clock
	stateStore        ConfigStore
	onActiveChanged   func(active bool)   // Callback for when active status changes
	onSettingsChanged func(next Settings) // Callback for any settings change
}

// SettingsOption configures a SettingsManager.
type SettingsOption func(*SettingsManager)

// WithSettingsClock sets the clock used by the settings manager.
func WithSettingsClock(c Clock) SettingsOption {
	return func(sm *SettingsManager) { sm.clock = c }
}

// WithSettingsConfigStore sets the config store used by the settings manager.
func WithSettingsConfigStore(cs ConfigStore) SettingsOption {
	return func(sm *SettingsManager) { sm.stateStore = cs }
}

// Default settings (always start inactive for safety)
var defaultSettings = Settings{
	Active:                       false, // ALWAYS start inactive for safety
	MetricCollectionInterval:     20,    // 20 seconds
	AnomalyDetectionInterval:     30,    // 30 seconds
	ThresholdCheckInterval:       20,    // 20 seconds
	CooldownPeriodSeconds:        300,   // 5 minutes
	CPUThreshold:                 85.0,  // 85%
	CPUHighPercent:               92.0,
	CPUCriticalPercent:           97.0,
	CPUEscalationCooldownSeconds: 1800,
	CPUEscalationDebounceTicks:   2,
	CPUSustainedWindowTicks:      3,
	CPUPressureThreshold:         10.0,
	MemoryThreshold:              90.0, // 90%
	DiskThreshold:                80.0, // warning band

	DiskHighPercent:               90.0,
	DiskCriticalPercent:           95.0,
	DiskEscalationCooldownSeconds: 1800, // 30 minutes
	DiskEscalationDebounceTicks:   2,
	DiskFastFillJumpPercent:       5.0,

	MetricsRetentionDays:          30,   // keep 30 days of metrics history
	RetentionCheckIntervalSeconds: 3600, // re-check retention hourly
	RetentionRunOnStartup:         true, // prune stale data without waiting an hour
	CompactAfterRetention:         false,
}

func sanitizeSettings(settings Settings) (Settings, bool) {
	changed := false

	if settings.MetricCollectionInterval <= 0 {
		settings.MetricCollectionInterval = defaultSettings.MetricCollectionInterval
		changed = true
	}

	if settings.AnomalyDetectionInterval <= 0 {
		settings.AnomalyDetectionInterval = defaultSettings.AnomalyDetectionInterval
		changed = true
	}

	if settings.ThresholdCheckInterval <= 0 {
		settings.ThresholdCheckInterval = defaultSettings.ThresholdCheckInterval
		changed = true
	}

	if settings.CooldownPeriodSeconds <= 0 {
		settings.CooldownPeriodSeconds = defaultSettings.CooldownPeriodSeconds
		changed = true
	}

	if settings.CPUThreshold <= 0 {
		settings.CPUThreshold = defaultSettings.CPUThreshold
		changed = true
	}
	if sanitizeCPUBands(&settings) {
		changed = true
	}

	if settings.MemoryThreshold <= 0 {
		settings.MemoryThreshold = defaultSettings.MemoryThreshold
		changed = true
	}

	if settings.DiskThreshold <= 0 {
		settings.DiskThreshold = defaultSettings.DiskThreshold
		changed = true
	}

	if sanitizeDiskBands(&settings) {
		changed = true
	}

	// A non-positive retention window means the lifecycle block is unset
	// (legacy file or fresh defaults); apply the full retention default set,
	// including enabling startup retention.
	if settings.MetricsRetentionDays <= 0 {
		settings.MetricsRetentionDays = defaultSettings.MetricsRetentionDays
		settings.RetentionRunOnStartup = defaultSettings.RetentionRunOnStartup
		changed = true
	}

	if settings.RetentionCheckIntervalSeconds <= 0 {
		settings.RetentionCheckIntervalSeconds = defaultSettings.RetentionCheckIntervalSeconds
		changed = true
	}

	return settings, changed
}

func sanitizeCPUBands(settings *Settings) bool {
	changed := false
	if settings.CPUHighPercent <= 0 {
		settings.CPUHighPercent = defaultSettings.CPUHighPercent
		changed = true
	}
	if settings.CPUCriticalPercent <= 0 {
		settings.CPUCriticalPercent = defaultSettings.CPUCriticalPercent
		changed = true
	}
	if settings.CPUEscalationCooldownSeconds <= 0 {
		settings.CPUEscalationCooldownSeconds = defaultSettings.CPUEscalationCooldownSeconds
		changed = true
	}
	if settings.CPUEscalationDebounceTicks <= 0 {
		settings.CPUEscalationDebounceTicks = defaultSettings.CPUEscalationDebounceTicks
		changed = true
	}
	if settings.CPUSustainedWindowTicks <= 0 {
		settings.CPUSustainedWindowTicks = defaultSettings.CPUSustainedWindowTicks
		changed = true
	}
	if settings.CPUPressureThreshold <= 0 {
		settings.CPUPressureThreshold = defaultSettings.CPUPressureThreshold
		changed = true
	}
	if !(settings.CPUThreshold < settings.CPUHighPercent && settings.CPUHighPercent < settings.CPUCriticalPercent) {
		settings.CPUThreshold = defaultSettings.CPUThreshold
		settings.CPUHighPercent = defaultSettings.CPUHighPercent
		settings.CPUCriticalPercent = defaultSettings.CPUCriticalPercent
		changed = true
	}
	return changed
}

// sanitizeDiskBands restores any unset escalation setting to its default and
// repairs a non-ascending band order.
//
// Order matters more than the individual values: bands that do not ascend make
// a higher band unreachable, so pressure could climb past critical while only
// ever classifying as warning. A file that gets this wrong is repaired rather
// than obeyed.
func sanitizeDiskBands(settings *Settings) bool {
	changed := false

	if settings.DiskHighPercent <= 0 {
		settings.DiskHighPercent = defaultSettings.DiskHighPercent
		changed = true
	}
	if settings.DiskCriticalPercent <= 0 {
		settings.DiskCriticalPercent = defaultSettings.DiskCriticalPercent
		changed = true
	}
	if settings.DiskEscalationCooldownSeconds <= 0 {
		settings.DiskEscalationCooldownSeconds = defaultSettings.DiskEscalationCooldownSeconds
		changed = true
	}
	if settings.DiskEscalationDebounceTicks <= 0 {
		settings.DiskEscalationDebounceTicks = defaultSettings.DiskEscalationDebounceTicks
		changed = true
	}
	if settings.DiskFastFillJumpPercent <= 0 {
		settings.DiskFastFillJumpPercent = defaultSettings.DiskFastFillJumpPercent
		changed = true
	}

	if !diskBandsAscend(*settings) {
		settings.DiskThreshold = defaultSettings.DiskThreshold
		settings.DiskHighPercent = defaultSettings.DiskHighPercent
		settings.DiskCriticalPercent = defaultSettings.DiskCriticalPercent
		changed = true
	}

	return changed
}

// diskBandsAscend reports whether the three band boundaries are strictly
// increasing.
func diskBandsAscend(s Settings) bool {
	return s.DiskThreshold < s.DiskHighPercent && s.DiskHighPercent < s.DiskCriticalPercent
}

// NewSettingsManager creates a new settings manager
func NewSettingsManager(opts ...SettingsOption) *SettingsManager {
	sm := &SettingsManager{
		settings: defaultSettings,
		clock:    RealClock{},
	}
	for _, opt := range opts {
		opt(sm)
	}
	if sm.stateStore == nil {
		sm.stateStore = &FileConfigStore{basePath: ResolveRuntimeStateBasePath()}
	}

	// Try to load existing settings, but if it fails, use defaults
	if err := sm.loadFromFile(); err != nil {
		// Log the error but continue with defaults
		fmt.Printf("Warning: Could not load settings from file (%v), using defaults\n", err)
		// Save defaults to create the config file
		if err := sm.saveToFile(); err != nil {
			fmt.Printf("Warning: Could not save default settings (%v)\n", err)
		}
	}

	return sm
}

// GetSettings returns a copy of current settings (thread-safe)
func (sm *SettingsManager) GetSettings() Settings {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.settings
}

// UpdateSettings updates the settings and saves to file
func (sm *SettingsManager) UpdateSettings(newSettings Settings) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if active status is changing
	oldActive := sm.settings.Active
	newActive := newSettings.Active

	// Update settings
	sanitized, changed := sanitizeSettings(newSettings)
	sm.settings = sanitized

	// Save to file
	if err := sm.saveToFile(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Call callbacks outside of the mutex to prevent deadlock.
	if oldActive != newActive && sm.onActiveChanged != nil {
		go sm.onActiveChanged(newActive)
	}
	if sm.onSettingsChanged != nil {
		go sm.onSettingsChanged(sanitized)
	}

	if changed {
		fmt.Println("Warning: Invalid monitoring settings values were adjusted to safe defaults")
	}

	return nil
}

// IsActive returns whether the monitor is currently active
func (sm *SettingsManager) IsActive() bool {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return sm.settings.Active
}

// SetActive updates just the active status
func (sm *SettingsManager) SetActive(active bool) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	oldActive := sm.settings.Active
	sm.settings.Active = active

	// Save to file
	if err := sm.saveToFile(); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}

	// Call callback if status changed
	if oldActive != active && sm.onActiveChanged != nil {
		// Call callback outside of mutex to prevent deadlock
		go sm.onActiveChanged(active)
	}

	return nil
}

// ResetSettings resets to default settings (inactive)
func (sm *SettingsManager) ResetSettings() error {
	return sm.UpdateSettings(defaultSettings)
}

// SetActiveChangedCallback sets the callback function for when active status changes
func (sm *SettingsManager) SetActiveChangedCallback(callback func(active bool)) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.onActiveChanged = callback
}

// SetSettingsChangedCallback sets the callback invoked with the new settings
// whenever settings are updated (including resets). The callback runs in its
// own goroutine to avoid holding the settings lock.
func (sm *SettingsManager) SetSettingsChangedCallback(callback func(next Settings)) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.onSettingsChanged = callback
}

// loadFromFile loads settings from JSON file
func (sm *SettingsManager) loadFromFile() error {
	data, err := sm.stateStore.ReadConfig("system-monitor-settings.json")
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var file settingsFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	sanitized, changed := sanitizeSettings(file.Settings)
	sm.settings = sanitized
	if changed {
		fmt.Println("Warning: Detected invalid monitoring settings; reverting to safe defaults")
		if err := sm.saveToFile(); err != nil {
			fmt.Printf("Warning: failed to persist sanitized settings: %v\n", err)
		}
	}
	return nil
}

// saveToFile saves current settings to JSON file
func (sm *SettingsManager) saveToFile() error {
	// Create config with metadata using the canonical wrapper shape.
	config := settingsFile{
		Version: "1.0.0",
		Metadata: map[string]interface{}{
			"last_modified":  sm.clock.Now().Format(time.RFC3339),
			"config_version": "1.0.0",
			"description":    "System Monitor settings including active/inactive status",
		},
		Settings: sm.settings,
	}

	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := sm.stateStore.WriteConfig("system-monitor-settings.json", data); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetMaintenanceState returns the maintenance state for external systems
func (sm *SettingsManager) GetMaintenanceState() string {
	if sm.IsActive() {
		return "active"
	}
	return "inactive"
}

// SetMaintenanceState sets the maintenance state (for external maintenance-window control)
func (sm *SettingsManager) SetMaintenanceState(state string) error {
	active := state == "active"
	return sm.SetActive(active)
}
