package settings

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestGetSettings verifies that GetSettings returns current settings safely
func TestGetSettings(t *testing.T) {
	// Reset to known state
	ResetSettings()

	settings := GetSettings()

	// Verify defaults
	if settings.Slots != DefaultSlots {
		t.Errorf("Expected Slots=%d, got %d", DefaultSlots, settings.Slots)
	}
	if settings.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("Expected RefreshInterval=%d, got %d", DefaultRefreshInterval, settings.RefreshInterval)
	}
	if settings.MaxTurns != DefaultMaxTurns {
		t.Errorf("Expected MaxTurns=%d, got %d", DefaultMaxTurns, settings.MaxTurns)
	}
	if settings.TaskTimeout != DefaultTaskTimeout {
		t.Errorf("Expected TaskTimeout=%d, got %d", DefaultTaskTimeout, settings.TaskTimeout)
	}
	if settings.IdleTimeoutCap != DefaultIdleTimeoutCap {
		t.Errorf("Expected IdleTimeoutCap=%d, got %d", DefaultIdleTimeoutCap, settings.IdleTimeoutCap)
	}
	if settings.AllowedTools != DefaultAllowedTools {
		t.Errorf("Expected AllowedTools=%s, got %s", DefaultAllowedTools, settings.AllowedTools)
	}
	if settings.SkipPermissions != DefaultSkipPermissions {
		t.Errorf("Expected SkipPermissions=%v, got %v", DefaultSkipPermissions, settings.SkipPermissions)
	}
	if settings.Active != DefaultActive {
		t.Errorf("Expected Active=%v, got %v", DefaultActive, settings.Active)
	}
	if settings.CondensedMode != DefaultCondensedMode {
		t.Errorf("Expected CondensedMode=%v, got %v", DefaultCondensedMode, settings.CondensedMode)
	}
}

// TestUpdateSettings verifies that UpdateSettings correctly updates settings
func TestUpdateSettings(t *testing.T) {
	// Reset to known state
	ResetSettings()

	// Create new settings
	newSettings := Settings{
		Theme:           "dark",
		Slots:           3,
		RefreshInterval: 60,
		Active:          true,
		MaxTurns:        50,
		AllowedTools:    "Read,Write,Edit",
		SkipPermissions: false,
		TaskTimeout:     120,
		IdleTimeoutCap:  90,
		CondensedMode:   true,
		Recycler: RecyclerSettings{
			EnabledFor:          "both",
			IntervalSeconds:     600,
			ModelProvider:       "openrouter",
			ModelName:           "claude-3-opus",
			CompletionThreshold: 5,
			FailureThreshold:    8,
		},
	}

	UpdateSettings(newSettings)

	// Verify update
	retrieved := GetSettings()
	if retrieved.Theme != "dark" {
		t.Errorf("Expected Theme='dark', got '%s'", retrieved.Theme)
	}
	if retrieved.Slots != 3 {
		t.Errorf("Expected Slots=3, got %d", retrieved.Slots)
	}
	if retrieved.RefreshInterval != 60 {
		t.Errorf("Expected RefreshInterval=60, got %d", retrieved.RefreshInterval)
	}
	if !retrieved.Active {
		t.Error("Expected Active=true, got false")
	}
	if retrieved.MaxTurns != 50 {
		t.Errorf("Expected MaxTurns=50, got %d", retrieved.MaxTurns)
	}
	if retrieved.AllowedTools != "Read,Write,Edit" {
		t.Errorf("Expected AllowedTools='Read,Write,Edit', got '%s'", retrieved.AllowedTools)
	}
	if retrieved.SkipPermissions {
		t.Error("Expected SkipPermissions=false, got true")
	}
	if retrieved.TaskTimeout != 120 {
		t.Errorf("Expected TaskTimeout=120, got %d", retrieved.TaskTimeout)
	}
	if retrieved.IdleTimeoutCap != 90 {
		t.Errorf("Expected IdleTimeoutCap=90, got %d", retrieved.IdleTimeoutCap)
	}
	if !retrieved.CondensedMode {
		t.Error("Expected CondensedMode=true, got false")
	}
	if retrieved.Recycler.EnabledFor != "both" {
		t.Errorf("Expected Recycler.EnabledFor='both', got '%s'", retrieved.Recycler.EnabledFor)
	}
	if retrieved.Recycler.IntervalSeconds != 600 {
		t.Errorf("Expected Recycler.IntervalSeconds=600, got %d", retrieved.Recycler.IntervalSeconds)
	}
	if retrieved.Recycler.ModelProvider != "openrouter" {
		t.Errorf("Expected Recycler.ModelProvider='openrouter', got '%s'", retrieved.Recycler.ModelProvider)
	}
	if retrieved.Recycler.CompletionThreshold != 5 {
		t.Errorf("Expected Recycler.CompletionThreshold=5, got %d", retrieved.Recycler.CompletionThreshold)
	}
	if retrieved.Recycler.FailureThreshold != 8 {
		t.Errorf("Expected Recycler.FailureThreshold=8, got %d", retrieved.Recycler.FailureThreshold)
	}
}

// TestResetSettings verifies that ResetSettings restores defaults
func TestResetSettings(t *testing.T) {
	// Modify settings
	UpdateSettings(Settings{
		Theme:           "dark",
		Slots:           5,
		RefreshInterval: 120,
		Active:          true,
		MaxTurns:        80,
		AllowedTools:    "Read",
		SkipPermissions: false,
		TaskTimeout:     240,
		IdleTimeoutCap:  180,
		CondensedMode:   true,
	})

	// Reset
	reset := ResetSettings()

	// Verify reset to defaults
	if reset.Slots != DefaultSlots {
		t.Errorf("Expected Slots=%d after reset, got %d", DefaultSlots, reset.Slots)
	}
	if reset.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("Expected RefreshInterval=%d after reset, got %d", DefaultRefreshInterval, reset.RefreshInterval)
	}
	if reset.Active != DefaultActive {
		t.Errorf("Expected Active=%v after reset, got %v", DefaultActive, reset.Active)
	}
	if reset.MaxTurns != DefaultMaxTurns {
		t.Errorf("Expected MaxTurns=%d after reset, got %d", DefaultMaxTurns, reset.MaxTurns)
	}
	if reset.TaskTimeout != DefaultTaskTimeout {
		t.Errorf("Expected TaskTimeout=%d after reset, got %d", DefaultTaskTimeout, reset.TaskTimeout)
	}
	if reset.IdleTimeoutCap != DefaultIdleTimeoutCap {
		t.Errorf("Expected IdleTimeoutCap=%d after reset, got %d", DefaultIdleTimeoutCap, reset.IdleTimeoutCap)
	}
	if reset.AllowedTools != DefaultAllowedTools {
		t.Errorf("Expected AllowedTools=%s after reset, got %s", DefaultAllowedTools, reset.AllowedTools)
	}
	if reset.SkipPermissions != DefaultSkipPermissions {
		t.Errorf("Expected SkipPermissions=%v after reset, got %v", DefaultSkipPermissions, reset.SkipPermissions)
	}
	if reset.CondensedMode != DefaultCondensedMode {
		t.Errorf("Expected CondensedMode=%v after reset, got %v", DefaultCondensedMode, reset.CondensedMode)
	}

	// Verify GetSettings also returns defaults
	current := GetSettings()
	if current.Slots != DefaultSlots {
		t.Errorf("Expected GetSettings().Slots=%d after reset, got %d", DefaultSlots, current.Slots)
	}
}

// TestIsActive verifies the IsActive helper function
func TestIsActive(t *testing.T) {
	// Reset to known state
	ResetSettings()

	// Default should be inactive
	if IsActive() {
		t.Error("Expected IsActive()=false by default, got true")
	}

	// Activate
	UpdateSettings(Settings{
		Active:          true,
		Slots:           DefaultSlots,
		RefreshInterval: DefaultRefreshInterval,
		MaxTurns:        DefaultMaxTurns,
		AllowedTools:    DefaultAllowedTools,
		SkipPermissions: DefaultSkipPermissions,
		TaskTimeout:     DefaultTaskTimeout,
		IdleTimeoutCap:  DefaultIdleTimeoutCap,
		CondensedMode:   false,
	})

	if !IsActive() {
		t.Error("Expected IsActive()=true after activation, got false")
	}

	// Deactivate
	UpdateSettings(Settings{
		Active:          false,
		Slots:           DefaultSlots,
		RefreshInterval: DefaultRefreshInterval,
		MaxTurns:        DefaultMaxTurns,
		AllowedTools:    DefaultAllowedTools,
		SkipPermissions: DefaultSkipPermissions,
		TaskTimeout:     DefaultTaskTimeout,
		IdleTimeoutCap:  DefaultIdleTimeoutCap,
		CondensedMode:   false,
	})

	if IsActive() {
		t.Error("Expected IsActive()=false after deactivation, got true")
	}
}

// TestGetRecyclerSettings verifies the GetRecyclerSettings helper function
func TestGetRecyclerSettings(t *testing.T) {
	// Reset to known state
	ResetSettings()

	recycler := GetRecyclerSettings()

	if recycler.EnabledFor != DefaultRecyclerEnabledFor {
		t.Errorf("Expected EnabledFor=%s, got %s", DefaultRecyclerEnabledFor, recycler.EnabledFor)
	}
	if recycler.IntervalSeconds != DefaultRecyclerInterval {
		t.Errorf("Expected IntervalSeconds=%d, got %d", DefaultRecyclerInterval, recycler.IntervalSeconds)
	}
	if recycler.ModelProvider != DefaultRecyclerModelProvider {
		t.Errorf("Expected ModelProvider=%s, got %s", DefaultRecyclerModelProvider, recycler.ModelProvider)
	}
	if recycler.ModelName != DefaultRecyclerModelID {
		t.Errorf("Expected ModelName=%s, got %s", DefaultRecyclerModelID, recycler.ModelName)
	}
	if recycler.CompletionThreshold != DefaultRecyclerCompletionThreshold {
		t.Errorf("Expected CompletionThreshold=%d, got %d", DefaultRecyclerCompletionThreshold, recycler.CompletionThreshold)
	}
	if recycler.FailureThreshold != DefaultRecyclerFailureThreshold {
		t.Errorf("Expected FailureThreshold=%d, got %d", DefaultRecyclerFailureThreshold, recycler.FailureThreshold)
	}
}

// TestConcurrentAccess verifies thread safety of settings operations
func TestConcurrentAccess(t *testing.T) {
	// Reset to known state
	ResetSettings()

	// Run concurrent operations
	const numGoroutines = 50
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 types of operations

	// Concurrent readers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = GetSettings()
			}
		}()
	}

	// Concurrent IsActive callers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = IsActive()
			}
		}()
	}

	// Concurrent writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				UpdateSettings(Settings{
					Slots:           DefaultSlots + (id % 3),
					RefreshInterval: DefaultRefreshInterval,
					Active:          (id+j)%2 == 0,
					MaxTurns:        DefaultMaxTurns,
					AllowedTools:    DefaultAllowedTools,
					SkipPermissions: DefaultSkipPermissions,
					TaskTimeout:     DefaultTaskTimeout,
				})
			}
		}(i)
	}

	wg.Wait()

	// If we got here without deadlock or race detector errors, test passes
	t.Log("Concurrent access test completed without deadlock")
}

// TestDefaultsMatchConstants verifies that newDefaultSettings uses constants correctly
func TestDefaultsMatchConstants(t *testing.T) {
	defaults := newDefaultSettings()

	// Core settings
	if defaults.Slots != DefaultSlots {
		t.Errorf("Default Slots mismatch: constant=%d, function=%d", DefaultSlots, defaults.Slots)
	}
	if defaults.RefreshInterval != DefaultRefreshInterval {
		t.Errorf("Default RefreshInterval mismatch: constant=%d, function=%d", DefaultRefreshInterval, defaults.RefreshInterval)
	}
	if defaults.MaxTurns != DefaultMaxTurns {
		t.Errorf("Default MaxTurns mismatch: constant=%d, function=%d", DefaultMaxTurns, defaults.MaxTurns)
	}
	if defaults.TaskTimeout != DefaultTaskTimeout {
		t.Errorf("Default TaskTimeout mismatch: constant=%d, function=%d", DefaultTaskTimeout, defaults.TaskTimeout)
	}
	if defaults.AllowedTools != DefaultAllowedTools {
		t.Errorf("Default AllowedTools mismatch: constant=%s, function=%s", DefaultAllowedTools, defaults.AllowedTools)
	}
	if defaults.SkipPermissions != DefaultSkipPermissions {
		t.Errorf("Default SkipPermissions mismatch: constant=%v, function=%v", DefaultSkipPermissions, defaults.SkipPermissions)
	}
	if defaults.Active != DefaultActive {
		t.Errorf("Default Active mismatch: constant=%v, function=%v", DefaultActive, defaults.Active)
	}
	if defaults.CondensedMode != DefaultCondensedMode {
		t.Errorf("Default CondensedMode mismatch: constant=%v, function=%v", DefaultCondensedMode, defaults.CondensedMode)
	}

	// Recycler settings
	if defaults.Recycler.EnabledFor != DefaultRecyclerEnabledFor {
		t.Errorf("Default Recycler.EnabledFor mismatch: constant=%s, function=%s", DefaultRecyclerEnabledFor, defaults.Recycler.EnabledFor)
	}
	if defaults.Recycler.IntervalSeconds != DefaultRecyclerInterval {
		t.Errorf("Default Recycler.IntervalSeconds mismatch: constant=%d, function=%d", DefaultRecyclerInterval, defaults.Recycler.IntervalSeconds)
	}
	if defaults.Recycler.ModelProvider != DefaultRecyclerModelProvider {
		t.Errorf("Default Recycler.ModelProvider mismatch: constant=%s, function=%s", DefaultRecyclerModelProvider, defaults.Recycler.ModelProvider)
	}
	if defaults.Recycler.ModelName != DefaultRecyclerModelID {
		t.Errorf("Default Recycler.ModelName mismatch: constant=%s, function=%s", DefaultRecyclerModelID, defaults.Recycler.ModelName)
	}
	if defaults.Recycler.CompletionThreshold != DefaultRecyclerCompletionThreshold {
		t.Errorf("Default Recycler.CompletionThreshold mismatch: constant=%d, function=%d", DefaultRecyclerCompletionThreshold, defaults.Recycler.CompletionThreshold)
	}
	if defaults.Recycler.FailureThreshold != DefaultRecyclerFailureThreshold {
		t.Errorf("Default Recycler.FailureThreshold mismatch: constant=%d, function=%d", DefaultRecyclerFailureThreshold, defaults.Recycler.FailureThreshold)
	}
}

// TestSettingsAreIndependent verifies that GetSettings returns a copy, not a reference
func TestSettingsAreIndependent(t *testing.T) {
	ResetSettings()

	// Get settings
	settings1 := GetSettings()
	settings1.Slots = 999 // Modify the copy

	// Get settings again
	settings2 := GetSettings()

	// Should still be default, not 999
	if settings2.Slots != DefaultSlots {
		t.Errorf("Settings are not independent: expected Slots=%d, got %d (modified copy leaked)", DefaultSlots, settings2.Slots)
	}
}

// TestValidateAndNormalize ensures defaults are carried forward and constraints enforced.
func TestValidateAndNormalize(t *testing.T) {
	previous := newDefaultSettings()
	previous.IdleTimeoutCap = 45
	previous.Recycler.IntervalSeconds = 120

	input := Settings{
		Slots:           2,
		RefreshInterval: 45,
		MaxTurns:        60,
		TaskTimeout:     90,
		IdleTimeoutCap:  0, // should fall back to previous
		AllowedTools:    "Read,Bash",
		SkipPermissions: true,
		Recycler: RecyclerSettings{
			EnabledFor:          "",
			IntervalSeconds:     0,
			ModelProvider:       "ollama",
			ModelName:           "llama3.1:8b",
			CompletionThreshold: 4,
			FailureThreshold:    6,
		},
	}

	result, err := ValidateAndNormalize(input, previous)
	if err != nil {
		t.Fatalf("expected validation to succeed: %v", err)
	}

	if result.IdleTimeoutCap != previous.IdleTimeoutCap {
		t.Fatalf("expected IdleTimeoutCap to fall back to previous (%d), got %d", previous.IdleTimeoutCap, result.IdleTimeoutCap)
	}
	if result.Recycler.IntervalSeconds != previous.Recycler.IntervalSeconds {
		t.Fatalf("expected recycler interval to fall back to previous (%d), got %d", previous.Recycler.IntervalSeconds, result.Recycler.IntervalSeconds)
	}
	if result.Recycler.EnabledFor != "off" {
		t.Fatalf("expected enabled_for to default to off, got %s", result.Recycler.EnabledFor)
	}
	if result.Slots != 2 {
		t.Fatalf("unexpected Slots value %d", result.Slots)
	}
}

// TestSaveAndLoadToDisk ensures settings persist across calls and never persist Active=true.
func TestSaveAndLoadToDisk(t *testing.T) {
	ResetSettings()
	tempDir := t.TempDir()
	SetPersistencePath(filepath.Join(tempDir, "settings.json"))
	defer SetPersistencePath("")

	updated := GetSettings()
	updated.Theme = "dark"
	updated.Slots = 3
	updated.RefreshInterval = 75
	updated.Active = true
	updated.Recycler.EnabledFor = "both"
	updated.Recycler.IntervalSeconds = 200
	updated.Recycler.ModelProvider = "openrouter"
	updated.Recycler.ModelName = "claude-test"
	updated.Recycler.CompletionThreshold = 6
	updated.Recycler.FailureThreshold = 7
	UpdateSettings(updated)

	if err := SaveToDisk(); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}

	// Force different in-memory state before loading
	ResetSettings()
	if err := LoadFromDisk(); err != nil {
		t.Fatalf("failed to load settings: %v", err)
	}

	result := GetSettings()
	if result.Theme != "dark" || result.Slots != 3 || result.RefreshInterval != 75 {
		t.Fatalf("loaded settings do not match saved values: %+v", result)
	}
	if result.Active {
		t.Fatalf("expected Active to be false after load, got true")
	}

	// Ensure file exists
	if _, err := os.Stat(filepath.Join(tempDir, "settings.json")); err != nil {
		t.Fatalf("expected settings file to exist: %v", err)
	}
}
