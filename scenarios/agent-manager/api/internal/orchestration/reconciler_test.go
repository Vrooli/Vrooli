package orchestration

import (
	"testing"
	"time"
)

// =============================================================================
// RECONCILER CONFIG TESTS
// =============================================================================

func TestDefaultReconcilerConfig(t *testing.T) {
	cfg := DefaultReconcilerConfig()

	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}

	if cfg.StaleThreshold != 2*time.Minute {
		t.Errorf("StaleThreshold = %v, want 2m", cfg.StaleThreshold)
	}

	if cfg.OrphanGracePeriod != 5*time.Minute {
		t.Errorf("OrphanGracePeriod = %v, want 5m", cfg.OrphanGracePeriod)
	}

	if cfg.MaxStaleRuns != 10 {
		t.Errorf("MaxStaleRuns = %d, want 10", cfg.MaxStaleRuns)
	}

	// Production defaults - always kill orphans and auto-recover
	if !cfg.KillOrphans {
		t.Error("KillOrphans should be true by default (production mode)")
	}

	if !cfg.AutoRecover {
		t.Error("AutoRecover should be true by default (production mode)")
	}
}

func TestReconcilerConfig_CustomValues(t *testing.T) {
	cfg := ReconcilerConfig{
		Interval:          1 * time.Minute,
		StaleThreshold:    5 * time.Minute,
		OrphanGracePeriod: 10 * time.Minute,
		MaxStaleRuns:      20,
		KillOrphans:       true,
		AutoRecover:       true,
	}

	if cfg.Interval != 1*time.Minute {
		t.Errorf("Interval = %v, want 1m", cfg.Interval)
	}
	if cfg.StaleThreshold != 5*time.Minute {
		t.Errorf("StaleThreshold = %v, want 5m", cfg.StaleThreshold)
	}
	if cfg.OrphanGracePeriod != 10*time.Minute {
		t.Errorf("OrphanGracePeriod = %v, want 10m", cfg.OrphanGracePeriod)
	}
	if cfg.MaxStaleRuns != 20 {
		t.Errorf("MaxStaleRuns = %d, want 20", cfg.MaxStaleRuns)
	}
	if !cfg.KillOrphans {
		t.Error("KillOrphans should be true")
	}
	if !cfg.AutoRecover {
		t.Error("AutoRecover should be true")
	}
}

// =============================================================================
// RECONCILE STATS TESTS
// =============================================================================

func TestReconcileStats_ZeroValues(t *testing.T) {
	stats := ReconcileStats{}

	if !stats.Timestamp.IsZero() {
		t.Error("Timestamp should be zero value")
	}
	if stats.Duration != 0 {
		t.Errorf("Duration = %v, want 0", stats.Duration)
	}
	if stats.RunsChecked != 0 {
		t.Errorf("RunsChecked = %d, want 0", stats.RunsChecked)
	}
	if stats.StaleRuns != 0 {
		t.Errorf("StaleRuns = %d, want 0", stats.StaleRuns)
	}
	if stats.OrphansFound != 0 {
		t.Errorf("OrphansFound = %d, want 0", stats.OrphansFound)
	}
	if stats.RunsRecovered != 0 {
		t.Errorf("RunsRecovered = %d, want 0", stats.RunsRecovered)
	}
	if stats.OrphansKilled != 0 {
		t.Errorf("OrphansKilled = %d, want 0", stats.OrphansKilled)
	}
	if stats.Errors != nil && len(stats.Errors) > 0 {
		t.Error("Errors should be empty")
	}
}

func TestReconcileStats_WithData(t *testing.T) {
	now := time.Now()
	stats := ReconcileStats{
		Timestamp:     now,
		Duration:      500 * time.Millisecond,
		RunsChecked:   100,
		StaleRuns:     5,
		OrphansFound:  3,
		RunsRecovered: 2,
		OrphansKilled: 1,
		Errors:        []string{"error 1", "error 2"},
	}

	if stats.Timestamp != now {
		t.Error("Timestamp not set correctly")
	}
	if stats.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", stats.Duration)
	}
	if stats.RunsChecked != 100 {
		t.Errorf("RunsChecked = %d, want 100", stats.RunsChecked)
	}
	if stats.StaleRuns != 5 {
		t.Errorf("StaleRuns = %d, want 5", stats.StaleRuns)
	}
	if stats.OrphansFound != 3 {
		t.Errorf("OrphansFound = %d, want 3", stats.OrphansFound)
	}
	if stats.RunsRecovered != 2 {
		t.Errorf("RunsRecovered = %d, want 2", stats.RunsRecovered)
	}
	if stats.OrphansKilled != 1 {
		t.Errorf("OrphansKilled = %d, want 1", stats.OrphansKilled)
	}
	if len(stats.Errors) != 2 {
		t.Errorf("Errors length = %d, want 2", len(stats.Errors))
	}
}

// =============================================================================
// ORPHAN PROCESS TESTS
// =============================================================================

func TestOrphanProcess_Fields(t *testing.T) {
	now := time.Now()
	orphan := OrphanProcess{
		PID:       12345,
		Tag:       "test-run-abc123",
		Command:   "claude-code run --tag test-run-abc123",
		StartTime: now,
	}

	if orphan.PID != 12345 {
		t.Errorf("PID = %d, want 12345", orphan.PID)
	}
	if orphan.Tag != "test-run-abc123" {
		t.Errorf("Tag = %s, want test-run-abc123", orphan.Tag)
	}
	if orphan.Command != "claude-code run --tag test-run-abc123" {
		t.Errorf("Command = %s", orphan.Command)
	}
	if orphan.StartTime != now {
		t.Error("StartTime not set correctly")
	}
}

// =============================================================================
// EXTRACT TAG FROM COMMAND TESTS
// =============================================================================

func TestExtractTagFromCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "tag with space separator",
			command:  "claude-code run --tag my-tag-123 -p test",
			expected: "my-tag-123",
		},
		{
			name:     "tag with equals separator",
			command:  "claude-code run --tag=my-tag-456 -p test",
			expected: "my-tag-456",
		},
		{
			name:     "UUID tag",
			command:  "resource-claude-code run --tag 123e4567-e89b-12d3-a456-426614174000",
			expected: "123e4567-e89b-12d3-a456-426614174000",
		},
		{
			name:     "no tag argument",
			command:  "claude-code run -p test",
			expected: "",
		},
		{
			name:     "empty command",
			command:  "",
			expected: "",
		},
		{
			name:     "tag at end of command",
			command:  "resource-claude-code run --verbose --tag final-tag",
			expected: "final-tag",
		},
		{
			name:     "tag with complex value",
			command:  "resource-claude-code run --tag ecosystem-task-12345-run-1",
			expected: "ecosystem-task-12345-run-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractTagFromCommand(tt.command)
			if got != tt.expected {
				t.Errorf("extractTagFromCommand(%q) = %q, want %q", tt.command, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// LOOKS LIKE AGENT MANAGER TAG TESTS
// =============================================================================

func TestLooksLikeAgentManagerTag(t *testing.T) {
	tests := []struct {
		tag      string
		expected bool
	}{
		// Valid UUIDs
		{"123e4567-e89b-12d3-a456-426614174000", true},
		{"a1b2c3d4-e5f6-7890-abcd-ef1234567890", true},

		// Known prefixes
		{"ecosystem-task-123", true},
		{"test-genie-run-456", true},
		{"agent-manager-task-789", true},
		{"run-12345", true},

		// Not agent-manager tags
		{"random-tag", false},
		{"user-process", false},
		{"my-script", false},
		{"", false},
		{"abc123", false},
		{"process-12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := looksLikeAgentManagerTag(tt.tag)
			if got != tt.expected {
				t.Errorf("looksLikeAgentManagerTag(%q) = %v, want %v", tt.tag, got, tt.expected)
			}
		})
	}
}

// =============================================================================
// NEW RECONCILER TESTS
// =============================================================================

func TestNewReconciler(t *testing.T) {
	rec := NewReconciler(nil, nil)

	if rec == nil {
		t.Fatal("NewReconciler returned nil")
	}

	// Should use defaults
	if rec.config.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", rec.config.Interval)
	}

	if rec.running {
		t.Error("should not be running initially")
	}
}

func TestNewReconciler_WithConfig(t *testing.T) {
	customCfg := ReconcilerConfig{
		Interval:       1 * time.Minute,
		StaleThreshold: 10 * time.Minute,
		KillOrphans:    true,
	}

	rec := NewReconciler(nil, nil, WithReconcilerConfig(customCfg))

	if rec.config.Interval != 1*time.Minute {
		t.Errorf("Interval = %v, want 1m", rec.config.Interval)
	}
	if rec.config.StaleThreshold != 10*time.Minute {
		t.Errorf("StaleThreshold = %v, want 10m", rec.config.StaleThreshold)
	}
	if !rec.config.KillOrphans {
		t.Error("KillOrphans should be true")
	}
}

func TestReconciler_IsRunning_Initial(t *testing.T) {
	rec := NewReconciler(nil, nil)

	if rec.IsRunning() {
		t.Error("should not be running initially")
	}
}

func TestReconciler_LastStats_Initial(t *testing.T) {
	rec := NewReconciler(nil, nil)
	stats := rec.LastStats()

	if !stats.Timestamp.IsZero() {
		t.Error("initial stats should have zero timestamp")
	}
	if stats.RunsChecked != 0 {
		t.Error("initial stats should have zero runs checked")
	}
}

// =============================================================================
// RECONCILER OPTION TESTS
// =============================================================================

func TestWithReconcilerConfig(t *testing.T) {
	cfg := ReconcilerConfig{
		Interval:   5 * time.Minute,
		KillOrphans: true,
	}

	rec := &Reconciler{}
	opt := WithReconcilerConfig(cfg)
	opt(rec)

	if rec.config.Interval != 5*time.Minute {
		t.Errorf("Interval = %v, want 5m", rec.config.Interval)
	}
	if !rec.config.KillOrphans {
		t.Error("KillOrphans should be true")
	}
}

// =============================================================================
// RECONCILER STOP BEFORE START TESTS
// =============================================================================

func TestReconciler_Stop_NotRunning(t *testing.T) {
	rec := NewReconciler(nil, nil)

	// Should not error when stopping a non-running reconciler
	err := rec.Stop()
	if err != nil {
		t.Errorf("Stop() returned error for non-running reconciler: %v", err)
	}
}
