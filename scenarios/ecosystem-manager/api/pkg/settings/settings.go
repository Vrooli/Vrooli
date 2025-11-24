package settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	MaxRetries          int    `json:"max_retries"`
	RetryDelaySeconds   int    `json:"retry_delay_seconds"`
}

// Settings represents the application settings
type Settings struct {
	// Display settings
	Theme         string `json:"theme"`
	CondensedMode bool   `json:"condensed_mode"`

	// Queue processor settings
	Slots           int `json:"slots"`
	CooldownSeconds int `json:"cooldown_seconds"`
	// Legacy field for backward compatibility; populated only on load and not persisted.
	RefreshInterval int  `json:"refresh_interval,omitempty"`
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

// persistencePath points to the on-disk settings file (optional).
var persistencePath string

// newDefaultSettings returns a fresh Settings instance with default values.
// This is the single source of truth for default settings.
func newDefaultSettings() Settings {
	return Settings{
		Theme:           "light",
		CondensedMode:   DefaultCondensedMode,
		Slots:           DefaultSlots,
		CooldownSeconds: DefaultCooldownSeconds,
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
			MaxRetries:          DefaultRecyclerMaxRetries,
			RetryDelaySeconds:   DefaultRecyclerRetryDelaySeconds,
		},
	}
}

// Global settings with thread safety
var (
	currentSettings = newDefaultSettings()
	settingsMutex   sync.RWMutex
	persistMutex    sync.Mutex
)

// SetPersistencePath configures the on-disk path used to persist settings.
func SetPersistencePath(path string) {
	persistencePath = path
}

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

// ValidateAndNormalize enforces constraints and fills missing fields using the previous values.
func ValidateAndNormalize(input Settings, previous Settings) (Settings, error) {
	s := input

	if s.Slots < MinSlots || s.Slots > MaxSlots {
		return previous, fmt.Errorf("Slots must be between %d and %d", MinSlots, MaxSlots)
	}
	// Migrate legacy refresh_interval to cooldown_seconds
	if s.CooldownSeconds == 0 && s.RefreshInterval > 0 {
		s.CooldownSeconds = s.RefreshInterval
	}
	if s.CooldownSeconds < MinCooldownSeconds || s.CooldownSeconds > MaxCooldownSeconds {
		return previous, fmt.Errorf("Cooldown must be between %d and %d seconds", MinCooldownSeconds, MaxCooldownSeconds)
	}
	if s.MaxTurns < MinMaxTurns || s.MaxTurns > MaxMaxTurns {
		return previous, fmt.Errorf("Max turns must be between %d and %d", MinMaxTurns, MaxMaxTurns)
	}
	if s.TaskTimeout < MinTaskTimeout || s.TaskTimeout > MaxTaskTimeout {
		return previous, fmt.Errorf("Task timeout must be between %d and %d minutes", MinTaskTimeout, MaxTaskTimeout)
	}
	if s.IdleTimeoutCap == 0 {
		s.IdleTimeoutCap = previous.IdleTimeoutCap
	}
	if s.IdleTimeoutCap < MinIdleTimeoutCap || s.IdleTimeoutCap > MaxIdleTimeoutCap {
		return previous, fmt.Errorf("Idle timeout cap must be between %d and %d minutes", MinIdleTimeoutCap, MaxIdleTimeoutCap)
	}

	recycler := s.Recycler
	if recycler.EnabledFor == "" {
		recycler.EnabledFor = previous.Recycler.EnabledFor
	}
	recycler.EnabledFor = strings.ToLower(strings.TrimSpace(recycler.EnabledFor))
	switch recycler.EnabledFor {
	case "", "off", "resources", "scenarios", "both":
		if recycler.EnabledFor == "" {
			recycler.EnabledFor = "off"
		}
	default:
		return previous, fmt.Errorf("Recycler enabled_for must be one of off, resources, scenarios, both")
	}

	if recycler.IntervalSeconds == 0 {
		recycler.IntervalSeconds = previous.Recycler.IntervalSeconds
	}
	if recycler.IntervalSeconds < MinRecyclerInterval || recycler.IntervalSeconds > MaxRecyclerInterval {
		return previous, fmt.Errorf("Recycler interval must be between %d and %d seconds", MinRecyclerInterval, MaxRecyclerInterval)
	}

	if recycler.ModelProvider == "" {
		recycler.ModelProvider = previous.Recycler.ModelProvider
	}
	recycler.ModelProvider = strings.ToLower(strings.TrimSpace(recycler.ModelProvider))
	if recycler.ModelProvider != "ollama" && recycler.ModelProvider != "openrouter" {
		return previous, fmt.Errorf("Recycler model_provider must be 'ollama' or 'openrouter'")
	}

	if recycler.ModelName == "" {
		recycler.ModelName = previous.Recycler.ModelName
	}

	if recycler.CompletionThreshold == 0 {
		recycler.CompletionThreshold = previous.Recycler.CompletionThreshold
	}
	if recycler.CompletionThreshold < MinRecyclerCompletionThreshold || recycler.CompletionThreshold > MaxRecyclerCompletionThreshold {
		return previous, fmt.Errorf("Recycler completion_threshold must be between %d and %d", MinRecyclerCompletionThreshold, MaxRecyclerCompletionThreshold)
	}

	if recycler.FailureThreshold == 0 {
		recycler.FailureThreshold = previous.Recycler.FailureThreshold
	}
	if recycler.FailureThreshold < MinRecyclerFailureThreshold || recycler.FailureThreshold > MaxRecyclerFailureThreshold {
		return previous, fmt.Errorf("Recycler failure_threshold must be between %d and %d", MinRecyclerFailureThreshold, MaxRecyclerFailureThreshold)
	}

	if recycler.MaxRetries < MinRecyclerMaxRetries || recycler.MaxRetries > MaxRecyclerMaxRetries {
		return previous, fmt.Errorf("Recycler max_retries must be between %d and %d", MinRecyclerMaxRetries, MaxRecyclerMaxRetries)
	}
	if recycler.RetryDelaySeconds == 0 {
		recycler.RetryDelaySeconds = previous.Recycler.RetryDelaySeconds
	}
	if recycler.RetryDelaySeconds < MinRecyclerRetryDelaySecs || recycler.RetryDelaySeconds > MaxRecyclerRetryDelaySecs {
		return previous, fmt.Errorf("Recycler retry_delay_seconds must be between %d and %d", MinRecyclerRetryDelaySecs, MaxRecyclerRetryDelaySecs)
	}

	s.Recycler = recycler
	return s, nil
}

// SaveToDisk persists current settings to the configured path, always forcing Active=false.
func SaveToDisk() error {
	if persistencePath == "" {
		return nil
	}

	persistMutex.Lock()
	defer persistMutex.Unlock()

	settings := GetSettings()
	settings.Active = false

	if err := os.MkdirAll(filepath.Dir(persistencePath), 0755); err != nil {
		return fmt.Errorf("failed creating persistence directory: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed marshaling settings: %w", err)
	}

	tmpPath := persistencePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed writing temp settings file: %w", err)
	}

	if err := os.Rename(tmpPath, persistencePath); err != nil {
		return fmt.Errorf("failed replacing settings file: %w", err)
	}

	return nil
}

// LoadFromDisk loads persisted settings (if present) and applies them, forcing Active=false.
func LoadFromDisk() error {
	if persistencePath == "" {
		return nil
	}

	data, err := os.ReadFile(persistencePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed reading settings file: %w", err)
	}

	var loaded Settings
	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed unmarshaling settings file: %w", err)
	}

	loaded.Active = false // always start inactive
	validated, err := ValidateAndNormalize(loaded, currentSettings)
	if err != nil {
		return fmt.Errorf("persisted settings invalid: %w", err)
	}

	UpdateSettings(validated)
	return nil
}
