// Package circuitbreaker tests
// [REQ:SCS-CB-001] Test auto-disable failing collectors
// [REQ:SCS-CB-002] Test failure tracking
// [REQ:SCS-CB-003] Test periodic retry
// [REQ:SCS-CB-004] Test circuit breaker reset
package circuitbreaker

import (
	"testing"
	"time"
)

// [REQ:SCS-CB-001] Test that breaker trips after threshold failures
func TestBreakerTripsAfterThreshold(t *testing.T) {
	cfg := Config{
		FailureThreshold: 3,
		RetryInterval:    1 * time.Second,
		Timeout:          1 * time.Second,
	}
	cb := New("test", cfg)

	// Verify starts closed
	if cb.State() != Closed {
		t.Errorf("expected initial state Closed, got %v", cb.State())
	}

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Errorf("expected Allow() to return true before threshold, iteration %d", i)
		}
		cb.RecordFailure()
	}

	// Should now be open
	if cb.State() != Open {
		t.Errorf("expected state Open after threshold failures, got %v", cb.State())
	}

	// Allow should return false when open
	if cb.Allow() {
		t.Error("expected Allow() to return false when circuit is open")
	}
}

// [REQ:SCS-CB-002] Test that failure count is tracked correctly
func TestFailureTracking(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 5
	cb := New("test", cfg)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	if cb.FailureCount() != 2 {
		t.Errorf("expected failure count 2, got %d", cb.FailureCount())
	}

	// Success should reset the counter
	cb.RecordSuccess()
	if cb.FailureCount() != 0 {
		t.Errorf("expected failure count 0 after success, got %d", cb.FailureCount())
	}
}

// [REQ:SCS-CB-003] Test half-open state and recovery
func TestHalfOpenRecovery(t *testing.T) {
	cfg := Config{
		FailureThreshold: 1,
		RetryInterval:    10 * time.Millisecond,
		Timeout:          1 * time.Second,
	}
	cb := New("test", cfg)

	// Trip the breaker
	cb.RecordFailure()
	if cb.State() != Open {
		t.Errorf("expected Open state, got %v", cb.State())
	}

	// Wait for retry interval
	time.Sleep(20 * time.Millisecond)

	// Should transition to half-open and allow one request
	if !cb.Allow() {
		t.Error("expected Allow() to return true after retry interval (half-open)")
	}

	if cb.State() != HalfOpen {
		t.Errorf("expected HalfOpen state, got %v", cb.State())
	}

	// Record success to close the circuit
	cb.RecordSuccess()
	if cb.State() != Closed {
		t.Errorf("expected Closed state after success in half-open, got %v", cb.State())
	}
}

// [REQ:SCS-CB-003] Test half-open failure goes back to open
func TestHalfOpenFailure(t *testing.T) {
	cfg := Config{
		FailureThreshold: 1,
		RetryInterval:    10 * time.Millisecond,
		Timeout:          1 * time.Second,
	}
	cb := New("test", cfg)

	// Trip the breaker
	cb.RecordFailure()

	// Wait and transition to half-open
	time.Sleep(20 * time.Millisecond)
	cb.Allow()

	// Fail in half-open
	cb.RecordFailure()
	if cb.State() != Open {
		t.Errorf("expected Open state after failure in half-open, got %v", cb.State())
	}
}

// [REQ:SCS-CB-004] Test manual reset
func TestManualReset(t *testing.T) {
	cfg := Config{
		FailureThreshold: 1,
		RetryInterval:    1 * time.Hour, // Long to ensure no auto-recovery
		Timeout:          1 * time.Second,
	}
	cb := New("test", cfg)

	// Trip the breaker
	cb.RecordFailure()
	if cb.State() != Open {
		t.Errorf("expected Open state, got %v", cb.State())
	}

	// Manual reset
	cb.Reset()
	if cb.State() != Closed {
		t.Errorf("expected Closed state after reset, got %v", cb.State())
	}
	if cb.FailureCount() != 0 {
		t.Errorf("expected failure count 0 after reset, got %d", cb.FailureCount())
	}
}

// [REQ:SCS-CB-001] Test default config values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.FailureThreshold != 3 {
		t.Errorf("expected default FailureThreshold 3, got %d", cfg.FailureThreshold)
	}
	if cfg.RetryInterval != 30*time.Second {
		t.Errorf("expected default RetryInterval 30s, got %v", cfg.RetryInterval)
	}
}

// [REQ:SCS-CB-002] Test Status snapshot
func TestStatus(t *testing.T) {
	cfg := DefaultConfig()
	cb := New("test-collector", cfg)

	status := cb.Status()
	if status.Name != "test-collector" {
		t.Errorf("expected name 'test-collector', got %q", status.Name)
	}
	if status.State != "closed" {
		t.Errorf("expected state 'closed', got %q", status.State)
	}
	if status.FailureThreshold != 3 {
		t.Errorf("expected threshold 3, got %d", status.FailureThreshold)
	}
}

// [REQ:SCS-CB-001] Test state string representations
func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{Closed, "closed"},
		{Open, "open"},
		{HalfOpen, "half_open"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

// [REQ:SCS-CB-002] Test timestamps are recorded
func TestTimestamps(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 10 // Don't trip
	cb := New("test", cfg)

	before := time.Now()
	cb.RecordFailure()
	after := time.Now()

	lastFailure := cb.LastFailure()
	if lastFailure.Before(before) || lastFailure.After(after) {
		t.Errorf("LastFailure time %v not in expected range [%v, %v]", lastFailure, before, after)
	}

	before = time.Now()
	cb.RecordSuccess()
	after = time.Now()

	lastSuccess := cb.LastSuccess()
	if lastSuccess.Before(before) || lastSuccess.After(after) {
		t.Errorf("LastSuccess time %v not in expected range [%v, %v]", lastSuccess, before, after)
	}
}
