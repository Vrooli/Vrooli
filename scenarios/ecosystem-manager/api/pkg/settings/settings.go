package settings

import (
	"sync"
)

// RecyclerSettings encapsulates configuration for the recycler daemon.
type RecyclerSettings struct {
	EnabledFor          string `json:"enabled_for"`
	IntervalSeconds     int    `json:"interval_seconds"`
	ModelProvider       string `json:"model_provider"`
	ModelName           string `json:"model_name"`
	CompletionThreshold int    `json:"completion_threshold"`
	FailureThreshold    int    `json:"failure_threshold"`
}

// Settings represents the application settings
type Settings struct {
	// Display settings
	Theme string `json:"theme"`

	// Queue processor settings
	Slots           int  `json:"slots"`
	RefreshInterval int  `json:"refresh_interval"`
	Active          bool `json:"active"`

	// Agent settings
	MaxTurns        int    `json:"max_turns"`
	AllowedTools    string `json:"allowed_tools"`
	SkipPermissions bool   `json:"skip_permissions"`
	TaskTimeout     int    `json:"task_timeout"`     // Task execution timeout in minutes
	IdleTimeoutCap  int    `json:"idle_timeout_cap"` // Max idle time allowed before watchdog cancellation (minutes)

	// Recycler automation settings
	Recycler RecyclerSettings `json:"recycler"`
}

// newDefaultSettings returns a fresh Settings instance with default values.
// This is the single source of truth for default settings.
func newDefaultSettings() Settings {
	return Settings{
		Theme:           "light",
		Slots:           DefaultSlots,
		RefreshInterval: DefaultRefreshInterval,
		Active:          DefaultActive, // ALWAYS start/reset inactive for safety
		MaxTurns:        DefaultMaxTurns,
		AllowedTools:    DefaultAllowedTools,
		SkipPermissions: DefaultSkipPermissions,
		TaskTimeout:     DefaultTaskTimeout,
		IdleTimeoutCap:  DefaultIdleTimeoutCap,
		Recycler: RecyclerSettings{
			EnabledFor:          DefaultRecyclerEnabledFor,
			IntervalSeconds:     DefaultRecyclerInterval,
			ModelProvider:       DefaultRecyclerModelProvider,
			ModelName:           DefaultRecyclerModelID,
			CompletionThreshold: DefaultRecyclerCompletionThreshold,
			FailureThreshold:    DefaultRecyclerFailureThreshold,
		},
	}
}

// Global settings with thread safety
var (
	currentSettings = newDefaultSettings()
	settingsMutex   sync.RWMutex
)

// GetSettings returns a copy of the current settings (thread-safe)
func GetSettings() Settings {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings
}

// UpdateSettings updates the current settings (thread-safe)
func UpdateSettings(newSettings Settings) {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()
	currentSettings = newSettings
}

// ResetSettings resets settings to defaults (thread-safe)
func ResetSettings() Settings {
	settingsMutex.Lock()
	defer settingsMutex.Unlock()

	currentSettings = newDefaultSettings()
	return currentSettings
}

// IsActive returns whether the processor should be active
func IsActive() bool {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.Active
}

// GetRecyclerSettings returns current recycler configuration.
func GetRecyclerSettings() RecyclerSettings {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.Recycler
}
