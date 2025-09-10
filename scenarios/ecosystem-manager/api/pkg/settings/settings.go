package settings

import (
	"sync"
)

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
	TaskTimeout     int    `json:"task_timeout"` // Task execution timeout in minutes
}

// Global settings with thread safety
var (
	currentSettings = Settings{
		Theme:           "light",
		Slots:           1,
		RefreshInterval: 30,
		Active:          false, // ALWAYS start inactive for safety
		MaxTurns:        60,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		SkipPermissions: true,
		TaskTimeout:     30, // 30 minutes default timeout
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
		MaxTurns:        60,
		AllowedTools:    "Read,Write,Edit,Bash,LS,Glob,Grep",
		SkipPermissions: true,
		TaskTimeout:     30, // 30 minutes default timeout
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
