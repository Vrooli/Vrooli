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
	MaxTasks        int  `json:"max_tasks"` // Maximum tasks to process (0 = unlimited)

	// Agent settings
	MaxTurns        int    `json:"max_turns"`
	AllowedTools    string `json:"allowed_tools"`
	SkipPermissions bool   `json:"skip_permissions"`
	TaskTimeout     int    `json:"task_timeout"` // Task execution timeout in minutes

	// Recycler automation settings
	Recycler RecyclerSettings `json:"recycler"`
}

// Global settings with thread safety
var (
	currentSettings = Settings{
		Theme:           "light",
		Slots:           1,
		RefreshInterval: 30,
		Active:          false, // ALWAYS start inactive for safety
		MaxTasks:        0,     // 0 = unlimited
		MaxTurns:        60,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		SkipPermissions: true,
		TaskTimeout:     30, // 30 minutes default timeout
		Recycler: RecyclerSettings{
			EnabledFor:          "off",
			IntervalSeconds:     60,
			ModelProvider:       "ollama",
			ModelName:           "llama3.1:8b",
			CompletionThreshold: 3,
			FailureThreshold:    5,
		},
	}
	settingsMutex sync.RWMutex
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

	currentSettings = Settings{
		Theme:           "light",
		Slots:           1,
		RefreshInterval: 30,
		Active:          false, // ALWAYS reset to inactive for safety
		MaxTasks:        0,     // 0 = unlimited
		MaxTurns:        60,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		SkipPermissions: true,
		TaskTimeout:     30, // 30 minutes default timeout
		Recycler: RecyclerSettings{
			EnabledFor:          "off",
			IntervalSeconds:     60,
			ModelProvider:       "ollama",
			ModelName:           "llama3.1:8b",
			CompletionThreshold: 3,
			FailureThreshold:    5,
		},
	}

	return currentSettings
}

// IsActive returns whether the processor should be active
func IsActive() bool {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.Active
}

// GetMaxTurns returns the current max turns setting
func GetMaxTurns() int {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.MaxTurns
}

// GetAllowedTools returns the current allowed tools setting
func GetAllowedTools() string {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.AllowedTools
}

// GetSkipPermissions returns the current skip permissions setting
func GetSkipPermissions() bool {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.SkipPermissions
}

// GetTaskTimeout returns the current task timeout setting in minutes
func GetTaskTimeout() int {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.TaskTimeout
}

// GetRecyclerSettings returns current recycler configuration.
func GetRecyclerSettings() RecyclerSettings {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.Recycler
}

// GetMaxTasks returns the maximum number of tasks to process (0 = unlimited)
func GetMaxTasks() int {
	settingsMutex.RLock()
	defer settingsMutex.RUnlock()
	return currentSettings.MaxTasks
}
