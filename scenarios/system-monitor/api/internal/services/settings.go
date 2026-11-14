package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sync"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/logx"
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
	CPUThreshold                  float64         `json:"cpu_threshold"`
	CPUHighPercent                float64         `json:"cpu_high_percent"`
	CPUCriticalPercent            float64         `json:"cpu_critical_percent"`
	CPUEscalationCooldownSeconds  int             `json:"cpu_escalation_cooldown_seconds"`
	CPUEscalationDebounceTicks    int             `json:"cpu_escalation_debounce_ticks"`
	CPUSustainedWindowTicks       int             `json:"cpu_sustained_window_ticks"`
	CPUPressureThreshold          float64         `json:"cpu_pressure_threshold"`
	MemoryThreshold               float64         `json:"memory_threshold"`
	MemoryCriticalPercent         float64         `json:"memory_critical_percent"`
	TCPConnectionsWarning         int             `json:"tcp_connections_warning"`
	TCPConnectionsCritical        int             `json:"tcp_connections_critical"`
	TCPGrowthWindowSeconds        int             `json:"tcp_growth_window_seconds"`
	TCPGrowthFloor                int             `json:"tcp_growth_floor"`
	TCPGrowthMinSamples           int             `json:"tcp_growth_min_samples"`
	TCPGrowthFactor               float64         `json:"tcp_growth_factor"`
	GPUUsageWarningPercent        float64         `json:"gpu_usage_warning_percent"`
	MetricsSnapshotStaleIntervals int             `json:"metrics_snapshot_stale_intervals"`
	OrphanedAgentAgeSeconds       int             `json:"orphaned_agent_age_seconds"`
	OrphanedAgentMemoryBytes      int64           `json:"orphaned_agent_memory_bytes"`
	AutoInvestigateCriticalRules  map[string]bool `json:"auto_investigate_critical_rules"`

	// DiskThreshold is the warning band boundary: the usage percentage at
	// which disk pressure starts being recorded. It is deliberately the same
	// setting the band model calls "warning" rather than a second setting
	// meaning the same thing — two names for one boundary is how a
	// configuration surface drifts away from the code that reads it.
	DiskThreshold float64 `json:"disk_threshold"`

	// Disk-pressure escalation bands, in percent used. Bands must ascend:
	// DiskThreshold (warning) < DiskHighPercent < DiskCriticalPercent.
	// warning  — forward the pressure; storage-manager applies only safe tier.
	// high     — start bounded autonomous recovery through storage-manager.
	// critical — bounded safe/regenerable reclamation may run unattended.
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

	// AlertSustainSeconds is how long an alert condition must hold before
	// the evaluator opens an alert, and how long it must stay clear before the
	// alert resolves.
	AlertSustainSeconds int `json:"alert_sustain_seconds"`

	// Linux memory-pressure level boundaries (see collectors
	// DeriveMemoryPressureLevel). The swap-in boundaries may also raise a
	// macOS native level.
	MemoryPressureSomeAvg60ElevatedPercent     float64 `json:"memory_pressure_some_avg60_elevated_percent"`
	MemoryPressureFullAvg60CriticalPercent     float64 `json:"memory_pressure_full_avg60_critical_percent"`
	MemoryPressureSwapInElevatedBytesPerSecond float64 `json:"memory_pressure_swap_in_elevated_bytes_per_second"`
	MemoryPressureSwapInCriticalBytesPerSecond float64 `json:"memory_pressure_swap_in_critical_bytes_per_second"`

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

const settingsFileName = "system-monitor-settings.json"

// SettingsManager manages system monitor settings with thread safety
type SettingsManager struct {
	settings          Settings
	mutex             sync.RWMutex
	clock             Clock
	logger            logx.Logger
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

// WithSettingsLogger sets the structured logger used by settings lifecycle.
func WithSettingsLogger(logger logx.Logger) SettingsOption {
	return func(sm *SettingsManager) {
		if logger != nil {
			sm.logger = logger
		}
	}
}

// WithSettingsConfigStore sets the config store used by the settings manager.
func WithSettingsConfigStore(cs ConfigStore) SettingsOption {
	return func(sm *SettingsManager) { sm.stateStore = cs }
}

// Default settings (always start inactive for safety)
var defaultSettings = Settings{
	Active:                        false, // ALWAYS start inactive for safety
	MetricCollectionInterval:      20,    // 20 seconds
	AnomalyDetectionInterval:      30,    // 30 seconds
	ThresholdCheckInterval:        20,    // 20 seconds
	CooldownPeriodSeconds:         300,   // 5 minutes
	CPUThreshold:                  85.0,  // 85%
	CPUHighPercent:                92.0,
	CPUCriticalPercent:            97.0,
	CPUEscalationCooldownSeconds:  1800,
	CPUEscalationDebounceTicks:    2,
	CPUSustainedWindowTicks:       3,
	CPUPressureThreshold:          10.0,
	MemoryThreshold:               90.0, // 90%
	MemoryCriticalPercent:         95.0,
	TCPConnectionsWarning:         5000,
	TCPConnectionsCritical:        20000,
	TCPGrowthWindowSeconds:        600,
	TCPGrowthFloor:                500,
	TCPGrowthMinSamples:           5,
	TCPGrowthFactor:               2,
	GPUUsageWarningPercent:        90,
	MetricsSnapshotStaleIntervals: 10,
	OrphanedAgentAgeSeconds:       3600,
	OrphanedAgentMemoryBytes:      2 * 1024 * 1024 * 1024,
	AutoInvestigateCriticalRules: map[string]bool{
		RuleMetricsSnapshot: true,
		RuleCPUUsage:        true,
		RuleMemoryUsage:     true,
		RuleTCPConnections:  true,
		RuleGPUUsage:        true,
		RuleMemoryPressure:  true,
		RuleOrphanedAgents:  true,
	},
	DiskThreshold: 80.0, // warning band

	DiskHighPercent:               90.0,
	DiskCriticalPercent:           95.0,
	DiskEscalationCooldownSeconds: 1800, // 30 minutes
	DiskEscalationDebounceTicks:   2,
	DiskFastFillJumpPercent:       5.0,

	AlertSustainSeconds: 300,

	MemoryPressureSomeAvg60ElevatedPercent:     10,
	MemoryPressureFullAvg60CriticalPercent:     5,
	MemoryPressureSwapInElevatedBytesPerSecond: 10 * 1024 * 1024,
	MemoryPressureSwapInCriticalBytesPerSecond: 50 * 1024 * 1024,

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
	if settings.MemoryCriticalPercent <= 0 {
		settings.MemoryCriticalPercent = defaultSettings.MemoryCriticalPercent
		changed = true
	}
	if settings.MemoryCriticalPercent <= settings.MemoryThreshold {
		settings.MemoryThreshold = defaultSettings.MemoryThreshold
		settings.MemoryCriticalPercent = defaultSettings.MemoryCriticalPercent
		changed = true
	}
	if settings.TCPConnectionsWarning <= 0 {
		settings.TCPConnectionsWarning = defaultSettings.TCPConnectionsWarning
		changed = true
	}
	if settings.TCPConnectionsCritical <= settings.TCPConnectionsWarning {
		settings.TCPConnectionsWarning = defaultSettings.TCPConnectionsWarning
		settings.TCPConnectionsCritical = defaultSettings.TCPConnectionsCritical
		changed = true
	}
	if settings.TCPGrowthWindowSeconds <= 0 {
		settings.TCPGrowthWindowSeconds = defaultSettings.TCPGrowthWindowSeconds
		changed = true
	}
	if settings.TCPGrowthFloor <= 0 {
		settings.TCPGrowthFloor = defaultSettings.TCPGrowthFloor
		changed = true
	}
	if settings.TCPGrowthMinSamples < 2 {
		settings.TCPGrowthMinSamples = defaultSettings.TCPGrowthMinSamples
		changed = true
	}
	if settings.TCPGrowthFactor <= 1 {
		settings.TCPGrowthFactor = defaultSettings.TCPGrowthFactor
		changed = true
	}
	if settings.GPUUsageWarningPercent <= 0 || settings.GPUUsageWarningPercent > 100 {
		settings.GPUUsageWarningPercent = defaultSettings.GPUUsageWarningPercent
		changed = true
	}
	if settings.MetricsSnapshotStaleIntervals <= 0 {
		settings.MetricsSnapshotStaleIntervals = defaultSettings.MetricsSnapshotStaleIntervals
		changed = true
	}
	if settings.OrphanedAgentAgeSeconds <= 0 {
		settings.OrphanedAgentAgeSeconds = defaultSettings.OrphanedAgentAgeSeconds
		changed = true
	}
	if settings.OrphanedAgentMemoryBytes <= 0 {
		settings.OrphanedAgentMemoryBytes = defaultSettings.OrphanedAgentMemoryBytes
		changed = true
	}
	if sanitizeAutoInvestigateRules(&settings) {
		changed = true
	}

	if settings.DiskThreshold <= 0 {
		settings.DiskThreshold = defaultSettings.DiskThreshold
		changed = true
	}

	if sanitizeDiskBands(&settings) {
		changed = true
	}
	if sanitizeAlertSettings(&settings) {
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

func sanitizeAutoInvestigateRules(settings *Settings) bool {
	changed := false
	if settings.AutoInvestigateCriticalRules == nil {
		settings.AutoInvestigateCriticalRules = make(map[string]bool, len(defaultSettings.AutoInvestigateCriticalRules))
	}
	for ruleID, defaultEnabled := range defaultSettings.AutoInvestigateCriticalRules {
		if _, ok := settings.AutoInvestigateCriticalRules[ruleID]; !ok {
			settings.AutoInvestigateCriticalRules[ruleID] = defaultEnabled
			changed = true
		}
	}
	for ruleID := range settings.AutoInvestigateCriticalRules {
		if _, ok := defaultSettings.AutoInvestigateCriticalRules[ruleID]; !ok {
			delete(settings.AutoInvestigateCriticalRules, ruleID)
			changed = true
		}
	}
	return changed
}

func cloneSettings(settings Settings) Settings {
	if settings.AutoInvestigateCriticalRules != nil {
		rules := make(map[string]bool, len(settings.AutoInvestigateCriticalRules))
		for ruleID, enabled := range settings.AutoInvestigateCriticalRules {
			rules[ruleID] = enabled
		}
		settings.AutoInvestigateCriticalRules = rules
	}
	return settings
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

// sanitizeAlertSettings restores unset alert-evaluation settings to their
// defaults and repairs memory-pressure boundaries that do not ascend.
func sanitizeAlertSettings(settings *Settings) bool {
	changed := false
	if settings.AlertSustainSeconds <= 0 {
		settings.AlertSustainSeconds = defaultSettings.AlertSustainSeconds
		changed = true
	}
	if settings.MemoryPressureSomeAvg60ElevatedPercent <= 0 {
		settings.MemoryPressureSomeAvg60ElevatedPercent = defaultSettings.MemoryPressureSomeAvg60ElevatedPercent
		changed = true
	}
	if settings.MemoryPressureFullAvg60CriticalPercent <= 0 {
		settings.MemoryPressureFullAvg60CriticalPercent = defaultSettings.MemoryPressureFullAvg60CriticalPercent
		changed = true
	}
	if settings.MemoryPressureSwapInElevatedBytesPerSecond <= 0 {
		settings.MemoryPressureSwapInElevatedBytesPerSecond = defaultSettings.MemoryPressureSwapInElevatedBytesPerSecond
		changed = true
	}
	if settings.MemoryPressureSwapInCriticalBytesPerSecond <= 0 {
		settings.MemoryPressureSwapInCriticalBytesPerSecond = defaultSettings.MemoryPressureSwapInCriticalBytesPerSecond
		changed = true
	}
	if settings.MemoryPressureSwapInElevatedBytesPerSecond >= settings.MemoryPressureSwapInCriticalBytesPerSecond {
		settings.MemoryPressureSwapInElevatedBytesPerSecond = defaultSettings.MemoryPressureSwapInElevatedBytesPerSecond
		settings.MemoryPressureSwapInCriticalBytesPerSecond = defaultSettings.MemoryPressureSwapInCriticalBytesPerSecond
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
		logger:   logx.Default(),
	}
	for _, opt := range opts {
		opt(sm)
	}
	if sm.stateStore == nil {
		sm.stateStore = &FileConfigStore{basePath: ResolveRuntimeStateBasePath()}
	}

	// Missing settings are initialized with defaults. Invalid or unreadable
	// settings fall back in memory but are preserved for operator recovery.
	if err := sm.loadFromFile(); err != nil {
		sm.logger.Warn("could not load settings file; using defaults in memory", "error", err)
		if errors.Is(err, fs.ErrNotExist) {
			if saveErr := sm.saveToFile(); saveErr != nil {
				sm.logger.Error("could not persist default settings", "error", saveErr)
			}
		}
	}

	return sm
}

// GetSettings returns a copy of current settings (thread-safe)
func (sm *SettingsManager) GetSettings() Settings {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()
	return cloneSettings(sm.settings)
}

// UpdateSettings updates the settings and saves to file
func (sm *SettingsManager) UpdateSettings(newSettings Settings) error {
	sm.mutex.Lock()

	// Check if active status is changing
	oldActive := sm.settings.Active
	newActive := newSettings.Active

	// Update settings
	sanitized, changed := sanitizeSettings(cloneSettings(newSettings))
	sm.settings = sanitized

	// Save to file
	if err := sm.saveToFile(); err != nil {
		sm.mutex.Unlock()
		return fmt.Errorf("failed to save settings: %w", err)
	}

	activeCallback := sm.onActiveChanged
	settingsCallback := sm.onSettingsChanged
	settingsCopy := cloneSettings(sanitized)
	sm.mutex.Unlock()

	// Call callbacks synchronously outside the settings lock. This keeps their
	// effects within the admitted request that changed settings, so maintenance
	// drain does not race detached updates after the handler returns.
	if oldActive != newActive && activeCallback != nil {
		activeCallback(newActive)
	}
	if settingsCallback != nil {
		settingsCallback(settingsCopy)
	}

	if changed {
		sm.logger.Warn("invalid settings adjusted to safe defaults")
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

	oldActive := sm.settings.Active
	sm.settings.Active = active

	// Save to file
	if err := sm.saveToFile(); err != nil {
		sm.mutex.Unlock()
		return fmt.Errorf("failed to save settings: %w", err)
	}

	callback := sm.onActiveChanged
	sm.mutex.Unlock()
	if oldActive != active && callback != nil {
		callback(active)
	}

	return nil
}

// ResetSettings resets to default settings (inactive)
func (sm *SettingsManager) ResetSettings() error {
	return sm.UpdateSettings(cloneSettings(defaultSettings))
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
	data, err := sm.stateStore.ReadConfig(settingsFileName)
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
		sm.logger.Warn("invalid settings detected; reverting values to safe defaults")
		if err := sm.saveToFile(); err != nil {
			sm.logger.Error("failed to persist sanitized settings", "error", err)
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

	if err := sm.stateStore.WriteConfig(settingsFileName, data); err != nil {
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
