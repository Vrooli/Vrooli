package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"agent-manager/internal/domain"
)

// =============================================================================
// DEFAULTS
// =============================================================================

func TestDefaultOrchestrationSettings_IsValid(t *testing.T) {
	s := DefaultOrchestrationSettings()
	if err := s.Validate(); err != nil {
		t.Fatalf("default settings should be valid: %v", err)
	}
}

// =============================================================================
// RUN EXECUTION VALIDATION
// =============================================================================

func TestOrchestrationSettings_Validate_RunExecution(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*OrchestrationSettings)
		wantErr bool
	}{
		{"runTimeoutMinutes at min", func(s *OrchestrationSettings) { s.RunExecution.RunTimeoutMinutes = 1 }, false},
		{"runTimeoutMinutes at max", func(s *OrchestrationSettings) { s.RunExecution.RunTimeoutMinutes = 9999 }, false},
		{"runTimeoutMinutes below min", func(s *OrchestrationSettings) { s.RunExecution.RunTimeoutMinutes = 0 }, true},
		{"runTimeoutMinutes above max", func(s *OrchestrationSettings) { s.RunExecution.RunTimeoutMinutes = 10000 }, true},
		{"maxConcurrentRuns at min", func(s *OrchestrationSettings) { s.RunExecution.MaxConcurrentRuns = 1 }, false},
		{"maxConcurrentRuns at max", func(s *OrchestrationSettings) { s.RunExecution.MaxConcurrentRuns = 9999 }, false},
		{"maxConcurrentRuns below min", func(s *OrchestrationSettings) { s.RunExecution.MaxConcurrentRuns = 0 }, true},
		{"maxConcurrentRuns above max", func(s *OrchestrationSettings) { s.RunExecution.MaxConcurrentRuns = 10000 }, true},
		{"maxTurns at min", func(s *OrchestrationSettings) { s.RunExecution.MaxTurns = 1 }, false},
		{"maxTurns at max", func(s *OrchestrationSettings) { s.RunExecution.MaxTurns = 9999 }, false},
		{"maxTurns below min", func(s *OrchestrationSettings) { s.RunExecution.MaxTurns = 0 }, true},
		{"maxTurns above max", func(s *OrchestrationSettings) { s.RunExecution.MaxTurns = 10000 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultOrchestrationSettings()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// SAFETY ISOLATION VALIDATION
// =============================================================================

func TestOrchestrationSettings_Validate_SafetyIsolation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*OrchestrationSettings)
		wantErr bool
	}{
		{"networkAccess none", func(s *OrchestrationSettings) { s.SafetyIsolation.NetworkAccess = "none" }, false},
		{"networkAccess localhost", func(s *OrchestrationSettings) { s.SafetyIsolation.NetworkAccess = "localhost" }, false},
		{"networkAccess full", func(s *OrchestrationSettings) { s.SafetyIsolation.NetworkAccess = "full" }, false},
		{"networkAccess invalid", func(s *OrchestrationSettings) { s.SafetyIsolation.NetworkAccess = "partial" }, true},
		{"networkAccess empty", func(s *OrchestrationSettings) { s.SafetyIsolation.NetworkAccess = "" }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultOrchestrationSettings()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// HEALTH DETECTION VALIDATION
// =============================================================================

func TestOrchestrationSettings_Validate_HealthDetection(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*OrchestrationSettings)
		wantErr bool
	}{
		{"heartbeatIntervalSeconds at min", func(s *OrchestrationSettings) { s.HealthDetection.HeartbeatIntervalSeconds = 1 }, false},
		{"heartbeatIntervalSeconds at max", func(s *OrchestrationSettings) {
			s.HealthDetection.HeartbeatIntervalSeconds = 9999
			s.HealthDetection.StaleThresholdSeconds = 9999 // will fail cross-field, so bump
		}, true}, // cross-field violation
		{"heartbeatIntervalSeconds below min", func(s *OrchestrationSettings) { s.HealthDetection.HeartbeatIntervalSeconds = 0 }, true},
		{"staleThresholdSeconds at min", func(s *OrchestrationSettings) { s.HealthDetection.StaleThresholdSeconds = 10 }, true}, // < heartbeat default of 15 cross-field
		{"staleThresholdSeconds at min valid", func(s *OrchestrationSettings) {
			s.HealthDetection.HeartbeatIntervalSeconds = 1
			s.HealthDetection.StaleThresholdSeconds = 10
			s.HealthDetection.MaxRecoveryAgeSeconds = 30
			s.ProcessTermination.GracePeriodSeconds = 1
			s.ProcessTermination.TerminationMaxRetries = 1
		}, false},
		{"staleThresholdSeconds below min", func(s *OrchestrationSettings) { s.HealthDetection.StaleThresholdSeconds = 9 }, true},
		{"maxRecoveryAgeSeconds at min", func(s *OrchestrationSettings) {
			s.HealthDetection.HeartbeatIntervalSeconds = 1
			s.HealthDetection.StaleThresholdSeconds = 10
			s.HealthDetection.MaxRecoveryAgeSeconds = 30
			s.ProcessTermination.GracePeriodSeconds = 1
			s.ProcessTermination.TerminationMaxRetries = 1
		}, false},
		{"maxRecoveryAgeSeconds below min", func(s *OrchestrationSettings) { s.HealthDetection.MaxRecoveryAgeSeconds = 29 }, true},
		{"reconcilerIntervalSeconds at min", func(s *OrchestrationSettings) { s.HealthDetection.ReconcilerIntervalSeconds = 5 }, false},
		{"reconcilerIntervalSeconds below min", func(s *OrchestrationSettings) { s.HealthDetection.ReconcilerIntervalSeconds = 4 }, true},
		{"reconcilerIntervalSeconds above max", func(s *OrchestrationSettings) { s.HealthDetection.ReconcilerIntervalSeconds = 10000 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultOrchestrationSettings()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// PROCESS TERMINATION VALIDATION
// =============================================================================

func TestOrchestrationSettings_Validate_ProcessTermination(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*OrchestrationSettings)
		wantErr bool
	}{
		{"gracePeriodSeconds at min", func(s *OrchestrationSettings) { s.ProcessTermination.GracePeriodSeconds = 1 }, false},
		{"gracePeriodSeconds at max", func(s *OrchestrationSettings) {
			s.ProcessTermination.GracePeriodSeconds = 9999
		}, true}, // cross-field: 9999 * 3 >= 300
		{"gracePeriodSeconds below min", func(s *OrchestrationSettings) { s.ProcessTermination.GracePeriodSeconds = 0 }, true},
		{"orphanGracePeriodSeconds at min", func(s *OrchestrationSettings) { s.ProcessTermination.OrphanGracePeriodSeconds = 30 }, false},
		{"orphanGracePeriodSeconds below min", func(s *OrchestrationSettings) { s.ProcessTermination.OrphanGracePeriodSeconds = 29 }, true},
		{"orphanGracePeriodSeconds above max", func(s *OrchestrationSettings) { s.ProcessTermination.OrphanGracePeriodSeconds = 10000 }, true},
		{"terminationMaxRetries at min", func(s *OrchestrationSettings) { s.ProcessTermination.TerminationMaxRetries = 1 }, false},
		{"terminationMaxRetries at max", func(s *OrchestrationSettings) {
			s.ProcessTermination.TerminationMaxRetries = 99
		}, true}, // cross-field: 5 * 99 >= 300
		{"terminationMaxRetries below min", func(s *OrchestrationSettings) { s.ProcessTermination.TerminationMaxRetries = 0 }, true},
		{"terminationMaxRetries above max", func(s *OrchestrationSettings) { s.ProcessTermination.TerminationMaxRetries = 100 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultOrchestrationSettings()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// CROSS-FIELD VALIDATION
// =============================================================================

func TestOrchestrationSettings_Validate_CrossField(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*OrchestrationSettings)
		wantErr  bool
		errField string
	}{
		{
			name: "heartbeatInterval equals staleThreshold fails",
			mutate: func(s *OrchestrationSettings) {
				s.HealthDetection.HeartbeatIntervalSeconds = 300
				s.HealthDetection.StaleThresholdSeconds = 300
			},
			wantErr:  true,
			errField: "heartbeatIntervalSeconds",
		},
		{
			name: "heartbeatInterval greater than staleThreshold fails",
			mutate: func(s *OrchestrationSettings) {
				s.HealthDetection.HeartbeatIntervalSeconds = 500
				s.HealthDetection.StaleThresholdSeconds = 300
			},
			wantErr:  true,
			errField: "heartbeatIntervalSeconds",
		},
		{
			name: "staleThreshold equals maxRecoveryAge fails",
			mutate: func(s *OrchestrationSettings) {
				s.HealthDetection.StaleThresholdSeconds = 600
				s.HealthDetection.MaxRecoveryAgeSeconds = 600
			},
			wantErr:  true,
			errField: "staleThresholdSeconds",
		},
		{
			name: "staleThreshold greater than maxRecoveryAge fails",
			mutate: func(s *OrchestrationSettings) {
				s.HealthDetection.StaleThresholdSeconds = 700
				s.HealthDetection.MaxRecoveryAgeSeconds = 600
			},
			wantErr:  true,
			errField: "staleThresholdSeconds",
		},
		{
			name: "gracePeriod * retries equals staleThreshold fails",
			mutate: func(s *OrchestrationSettings) {
				s.ProcessTermination.GracePeriodSeconds = 100
				s.ProcessTermination.TerminationMaxRetries = 3
				s.HealthDetection.StaleThresholdSeconds = 300
			},
			wantErr:  true,
			errField: "gracePeriodSeconds",
		},
		{
			name: "gracePeriod * retries exceeds staleThreshold fails",
			mutate: func(s *OrchestrationSettings) {
				s.ProcessTermination.GracePeriodSeconds = 100
				s.ProcessTermination.TerminationMaxRetries = 5
				s.HealthDetection.StaleThresholdSeconds = 300
			},
			wantErr:  true,
			errField: "gracePeriodSeconds",
		},
		{
			name: "gracePeriod * retries under staleThreshold passes",
			mutate: func(s *OrchestrationSettings) {
				s.ProcessTermination.GracePeriodSeconds = 10
				s.ProcessTermination.TerminationMaxRetries = 5
				s.HealthDetection.StaleThresholdSeconds = 300
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := DefaultOrchestrationSettings()
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errField != "" {
				cfgErr, ok := err.(*domain.ConfigError)
				if !ok {
					t.Fatalf("expected *domain.ConfigError, got %T", err)
				}
				if cfgErr.Setting == "" {
					t.Fatal("expected non-empty Setting on ConfigError")
				}
				// The setting should end with the expected field name.
				wantSuffix := "." + tt.errField
				if len(cfgErr.Setting) < len(wantSuffix) || cfgErr.Setting[len(cfgErr.Setting)-len(wantSuffix):] != wantSuffix {
					t.Errorf("expected Setting to end with %q, got %q", wantSuffix, cfgErr.Setting)
				}
			}
		})
	}
}

// =============================================================================
// STORE: ROUND-TRIP
// =============================================================================

func TestOrchestrationSettingsStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orchestration.json")

	store, err := NewOrchestrationSettingsStore(path)
	if err != nil {
		t.Fatalf("NewOrchestrationSettingsStore: %v", err)
	}

	// Modify and update.
	updated := DefaultOrchestrationSettings()
	updated.RunExecution.RunTimeoutMinutes = 60
	updated.RunExecution.MaxTurns = 200

	if err := store.Update(updated); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got := store.Get()
	if got.RunExecution.RunTimeoutMinutes != 60 {
		t.Errorf("RunTimeoutMinutes = %d, want 60", got.RunExecution.RunTimeoutMinutes)
	}
	if got.RunExecution.MaxTurns != 200 {
		t.Errorf("MaxTurns = %d, want 200", got.RunExecution.MaxTurns)
	}

	// Re-open store from same file and verify persistence.
	store2, err := NewOrchestrationSettingsStore(path)
	if err != nil {
		t.Fatalf("NewOrchestrationSettingsStore (reopen): %v", err)
	}
	got2 := store2.Get()
	if got2.RunExecution.RunTimeoutMinutes != 60 {
		t.Errorf("Reopened RunTimeoutMinutes = %d, want 60", got2.RunExecution.RunTimeoutMinutes)
	}
	if got2.RunExecution.MaxTurns != 200 {
		t.Errorf("Reopened MaxTurns = %d, want 200", got2.RunExecution.MaxTurns)
	}
}

// =============================================================================
// STORE: MISSING FILE
// =============================================================================

func TestOrchestrationSettingsStore_MissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "orchestration.json")

	store, err := NewOrchestrationSettingsStore(path)
	if err != nil {
		t.Fatalf("NewOrchestrationSettingsStore: %v", err)
	}

	// Should have created the file with defaults.
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist at %s: %v", path, err)
	}

	got := store.Get()
	defaults := DefaultOrchestrationSettings()
	if got != defaults {
		t.Errorf("expected defaults, got %+v", got)
	}

	// Verify file content parses back to defaults.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var fromDisk OrchestrationSettings
	if err := json.Unmarshal(data, &fromDisk); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if fromDisk != defaults {
		t.Errorf("on-disk settings differ from defaults: %+v", fromDisk)
	}
}

// =============================================================================
// STORE: RESET
// =============================================================================

func TestOrchestrationSettingsStore_Reset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "orchestration.json")

	store, err := NewOrchestrationSettingsStore(path)
	if err != nil {
		t.Fatalf("NewOrchestrationSettingsStore: %v", err)
	}

	// Modify settings.
	modified := DefaultOrchestrationSettings()
	modified.RunExecution.RunTimeoutMinutes = 120
	modified.SafetyIsolation.NetworkAccess = "full"
	if err := store.Update(modified); err != nil {
		t.Fatalf("Update: %v", err)
	}

	// Verify modification took effect.
	got := store.Get()
	if got.RunExecution.RunTimeoutMinutes != 120 {
		t.Fatalf("expected 120, got %d", got.RunExecution.RunTimeoutMinutes)
	}

	// Reset and verify defaults restored.
	if err := store.Reset(); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	defaults := DefaultOrchestrationSettings()
	got = store.Get()
	if got != defaults {
		t.Errorf("after reset, expected defaults, got %+v", got)
	}

	// Verify disk was also updated.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var fromDisk OrchestrationSettings
	if err := json.Unmarshal(data, &fromDisk); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if fromDisk != defaults {
		t.Errorf("on-disk settings differ from defaults after reset: %+v", fromDisk)
	}
}
