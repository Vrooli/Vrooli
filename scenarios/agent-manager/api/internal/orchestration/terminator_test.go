package orchestration

import (
	"testing"
	"time"
)

// =============================================================================
// TERMINATOR CONFIG TESTS
// =============================================================================

func TestDefaultTerminatorConfig(t *testing.T) {
	cfg := DefaultTerminatorConfig()

	if cfg.GracePeriod != 5*time.Second {
		t.Errorf("GracePeriod = %v, want 5s", cfg.GracePeriod)
	}

	if cfg.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", cfg.MaxRetries)
	}

	if cfg.BaseBackoff != 500*time.Millisecond {
		t.Errorf("BaseBackoff = %v, want 500ms", cfg.BaseBackoff)
	}

	if cfg.MaxBackoff != 5*time.Second {
		t.Errorf("MaxBackoff = %v, want 5s", cfg.MaxBackoff)
	}

	if cfg.VerifyTimeout != 2*time.Second {
		t.Errorf("VerifyTimeout = %v, want 2s", cfg.VerifyTimeout)
	}

	if !cfg.KillProcessGroup {
		t.Error("KillProcessGroup should be true by default")
	}
}

func TestTerminatorConfig_CustomValues(t *testing.T) {
	cfg := TerminatorConfig{
		GracePeriod:      10 * time.Second,
		MaxRetries:       5,
		BaseBackoff:      1 * time.Second,
		MaxBackoff:       30 * time.Second,
		VerifyTimeout:    5 * time.Second,
		KillProcessGroup: false,
	}

	if cfg.GracePeriod != 10*time.Second {
		t.Errorf("GracePeriod = %v, want 10s", cfg.GracePeriod)
	}
	if cfg.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", cfg.MaxRetries)
	}
	if cfg.BaseBackoff != 1*time.Second {
		t.Errorf("BaseBackoff = %v, want 1s", cfg.BaseBackoff)
	}
	if cfg.MaxBackoff != 30*time.Second {
		t.Errorf("MaxBackoff = %v, want 30s", cfg.MaxBackoff)
	}
	if cfg.VerifyTimeout != 5*time.Second {
		t.Errorf("VerifyTimeout = %v, want 5s", cfg.VerifyTimeout)
	}
	if cfg.KillProcessGroup {
		t.Error("KillProcessGroup should be false")
	}
}

// =============================================================================
// TERMINATE RESULT TESTS
// =============================================================================

func TestTerminateResult_SuccessCase(t *testing.T) {
	result := &TerminateResult{
		Success:        true,
		Attempts:       1,
		FinalMethod:    "sigterm",
		Duration:       500 * time.Millisecond,
		Error:          nil,
		ProcessWasGone: false,
	}

	if !result.Success {
		t.Error("Success should be true")
	}
	if result.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", result.Attempts)
	}
	if result.FinalMethod != "sigterm" {
		t.Errorf("FinalMethod = %s, want sigterm", result.FinalMethod)
	}
	if result.Duration != 500*time.Millisecond {
		t.Errorf("Duration = %v, want 500ms", result.Duration)
	}
	if result.Error != nil {
		t.Errorf("Error should be nil, got %v", result.Error)
	}
}

func TestTerminateResult_FailureCase(t *testing.T) {
	result := &TerminateResult{
		Success:        false,
		Attempts:       3,
		FinalMethod:    "",
		Duration:       15 * time.Second,
		Error:          nil,
		ProcessWasGone: false,
	}

	if result.Success {
		t.Error("Success should be false")
	}
	if result.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", result.Attempts)
	}
	if result.FinalMethod != "" {
		t.Errorf("FinalMethod should be empty, got %s", result.FinalMethod)
	}
}

func TestTerminateResult_ProcessWasGone(t *testing.T) {
	result := &TerminateResult{
		Success:        true,
		Attempts:       1,
		FinalMethod:    "not_found",
		Duration:       10 * time.Millisecond,
		ProcessWasGone: true,
	}

	if !result.Success {
		t.Error("Success should be true when process was already gone")
	}
	if !result.ProcessWasGone {
		t.Error("ProcessWasGone should be true")
	}
	if result.FinalMethod != "not_found" {
		t.Errorf("FinalMethod = %s, want not_found", result.FinalMethod)
	}
}

func TestTerminateResult_AllMethods(t *testing.T) {
	methods := []string{"runner", "cli", "sigterm", "sigkill", "pgkill", "not_found"}

	for _, method := range methods {
		result := &TerminateResult{
			Success:     true,
			FinalMethod: method,
		}
		if result.FinalMethod != method {
			t.Errorf("FinalMethod = %s, want %s", result.FinalMethod, method)
		}
	}
}

// =============================================================================
// CALCULATE BACKOFF TESTS
// =============================================================================

func TestTerminator_calculateBackoff(t *testing.T) {
	cfg := TerminatorConfig{
		BaseBackoff: 500 * time.Millisecond,
		MaxBackoff:  5 * time.Second,
	}
	term := &Terminator{config: cfg}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 500 * time.Millisecond},  // 500ms * 2^0 = 500ms
		{2, 1 * time.Second},          // 500ms * 2^1 = 1s
		{3, 2 * time.Second},          // 500ms * 2^2 = 2s
		{4, 4 * time.Second},          // 500ms * 2^3 = 4s
		{5, 5 * time.Second},          // 500ms * 2^4 = 8s, capped at 5s
		{10, 5 * time.Second},         // Would be huge, capped at 5s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := term.calculateBackoff(tt.attempt)
			if got != tt.expected {
				t.Errorf("calculateBackoff(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

func TestTerminator_calculateBackoff_SmallMaxBackoff(t *testing.T) {
	cfg := TerminatorConfig{
		BaseBackoff: 100 * time.Millisecond,
		MaxBackoff:  200 * time.Millisecond,
	}
	term := &Terminator{config: cfg}

	// First attempt should be base
	got := term.calculateBackoff(1)
	if got != 100*time.Millisecond {
		t.Errorf("attempt 1: got %v, want 100ms", got)
	}

	// Second attempt should hit max
	got = term.calculateBackoff(2)
	if got != 200*time.Millisecond {
		t.Errorf("attempt 2: got %v, want 200ms", got)
	}

	// Third attempt should still be capped
	got = term.calculateBackoff(3)
	if got != 200*time.Millisecond {
		t.Errorf("attempt 3: got %v, want 200ms", got)
	}
}

// =============================================================================
// NEW TERMINATOR TESTS
// =============================================================================

func TestNewTerminator(t *testing.T) {
	cfg := DefaultTerminatorConfig()
	term := NewTerminator(nil, nil, cfg)

	if term == nil {
		t.Fatal("NewTerminator returned nil")
	}

	if term.config.GracePeriod != cfg.GracePeriod {
		t.Error("Config not set correctly")
	}

	if term.config.MaxRetries != cfg.MaxRetries {
		t.Error("Config not set correctly")
	}
}

func TestNewTerminator_CustomConfig(t *testing.T) {
	cfg := TerminatorConfig{
		GracePeriod: 30 * time.Second,
		MaxRetries:  10,
	}
	term := NewTerminator(nil, nil, cfg)

	if term.config.GracePeriod != 30*time.Second {
		t.Errorf("GracePeriod = %v, want 30s", term.config.GracePeriod)
	}
	if term.config.MaxRetries != 10 {
		t.Errorf("MaxRetries = %d, want 10", term.config.MaxRetries)
	}
}
