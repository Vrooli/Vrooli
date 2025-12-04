// Package userconfig manages user-configurable settings for vrooli-autoheal
// Configuration is stored as JSON and validated against a JSON schema
package userconfig

// Config is the root configuration structure
type Config struct {
	Version string           `json:"version"`
	Global  GlobalConfig     `json:"global,omitempty"`
	Checks  map[string]Check `json:"checks,omitempty"`
	UI      UIConfig         `json:"ui,omitempty"`
}

// GlobalConfig contains settings that apply to all checks
type GlobalConfig struct {
	GracePeriodSeconds     int `json:"gracePeriodSeconds,omitempty"`
	TickIntervalSeconds    int `json:"tickIntervalSeconds,omitempty"`
	VerifyDelaySeconds     int `json:"verifyDelaySeconds,omitempty"`
	MaxRestartAttempts     int `json:"maxRestartAttempts,omitempty"`
	RestartCooldownSeconds int `json:"restartCooldownSeconds,omitempty"`
	HistoryRetentionHours  int `json:"historyRetentionHours,omitempty"`
}

// Check contains per-check configuration
type Check struct {
	Enabled         *bool          `json:"enabled,omitempty"`
	AutoHeal        *bool          `json:"autoHeal,omitempty"`
	IntervalSeconds *int           `json:"intervalSeconds,omitempty"`
	Thresholds      *Thresholds    `json:"thresholds,omitempty"`
	Settings        *CheckSettings `json:"settings,omitempty"`
}

// Thresholds contains check-specific threshold overrides
type Thresholds struct {
	WarningPercent  *float64 `json:"warningPercent,omitempty"`
	CriticalPercent *float64 `json:"criticalPercent,omitempty"`
	WarningCount    *int     `json:"warningCount,omitempty"`
	CriticalCount   *int     `json:"criticalCount,omitempty"`
	Partitions      []string `json:"partitions,omitempty"`
}

// CheckSettings contains check-specific operational settings
type CheckSettings struct {
	TunnelTestURL           string `json:"tunnelTestUrl,omitempty"`
	CleanPortsBeforeRestart *bool  `json:"cleanPortsBeforeRestart,omitempty"`
	CaptureLogsOnFailure    *bool  `json:"captureLogsOnFailure,omitempty"`
	LogLinesToCapture       *int   `json:"logLinesToCapture,omitempty"`
}

// UIConfig contains UI display preferences
type UIConfig struct {
	AutoRefreshSeconds int    `json:"autoRefreshSeconds,omitempty"`
	Theme              string `json:"theme,omitempty"`
	ShowDisabledChecks bool   `json:"showDisabledChecks,omitempty"`
	DefaultTab         string `json:"defaultTab,omitempty"`
}

// ValidationError represents a config validation error
type ValidationError struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// ValidationResult contains the result of config validation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}
