package execution

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCircuitBreaker_RecordFailureAndTrip(t *testing.T) {
	dir := t.TempDir()
	cb := NewCircuitBreaker(filepath.Join(dir, "cb.json"))

	// Record 2 failures with threshold 3 — should not trip.
	if err := cb.RecordFailure("execute/test-item", 3); err != nil {
		t.Fatalf("RecordFailure 1: %v", err)
	}
	if err := cb.RecordFailure("execute/test-item", 3); err != nil {
		t.Fatalf("RecordFailure 2: %v", err)
	}

	broken, _, err := cb.IsBroken("execute/test-item", 60)
	if err != nil {
		t.Fatalf("IsBroken: %v", err)
	}
	if broken {
		t.Fatal("expected not broken after 2 failures with threshold 3")
	}

	// Third failure should trip the breaker.
	if err := cb.RecordFailure("execute/test-item", 3); err != nil {
		t.Fatalf("RecordFailure 3: %v", err)
	}

	broken, remaining, err := cb.IsBroken("execute/test-item", 60)
	if err != nil {
		t.Fatalf("IsBroken: %v", err)
	}
	if !broken {
		t.Fatal("expected broken after 3 failures with threshold 3")
	}
	if remaining <= 0 {
		t.Fatal("expected positive cooldown remaining")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	dir := t.TempDir()
	cb := NewCircuitBreaker(filepath.Join(dir, "cb.json"))

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.RecordFailure("execute/test-item", 3)
	}

	// Record success should clear.
	if err := cb.RecordSuccess("execute/test-item"); err != nil {
		t.Fatalf("RecordSuccess: %v", err)
	}

	broken, _, err := cb.IsBroken("execute/test-item", 60)
	if err != nil {
		t.Fatalf("IsBroken: %v", err)
	}
	if broken {
		t.Fatal("expected not broken after success")
	}
}

func TestCircuitBreaker_ExplicitReset(t *testing.T) {
	dir := t.TempDir()
	cb := NewCircuitBreaker(filepath.Join(dir, "cb.json"))

	// Trip the breaker.
	for i := 0; i < 3; i++ {
		_ = cb.RecordFailure("execute/test-item", 3)
	}

	// Reset should clear regardless of cooldown.
	if err := cb.Reset("execute/test-item"); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	broken, _, _ := cb.IsBroken("execute/test-item", 60)
	if broken {
		t.Fatal("expected not broken after explicit reset")
	}
}

func TestCircuitBreaker_ResetNonExistent(t *testing.T) {
	dir := t.TempDir()
	cb := NewCircuitBreaker(filepath.Join(dir, "cb.json"))

	err := cb.Reset("execute/nonexistent")
	if err == nil {
		t.Fatal("expected error resetting nonexistent item")
	}
}

func TestCircuitBreaker_BrokenItems(t *testing.T) {
	dir := t.TempDir()
	cb := NewCircuitBreaker(filepath.Join(dir, "cb.json"))

	// Trip two items.
	for i := 0; i < 3; i++ {
		_ = cb.RecordFailure("execute/item-a", 3)
		_ = cb.RecordFailure("execute/item-b", 3)
	}

	items, err := cb.BrokenItems(60)
	if err != nil {
		t.Fatalf("BrokenItems: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 broken items, got %d", len(items))
	}
}

func TestCircuitBreaker_CooldownExpiry(t *testing.T) {
	dir := t.TempDir()
	cbPath := filepath.Join(dir, "cb.json")
	cb := NewCircuitBreaker(cbPath)

	// Manually write a broken entry with brokenAt in the past.
	pastTime := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	state := CircuitBreakerState{
		Items: map[string]*CircuitBreakerEntry{
			"execute/expired": {
				ConsecutiveFailures: 3,
				LastFailure:         pastTime,
				BrokenAt:            pastTime,
			},
		},
	}
	data, _ := os.CreateTemp(dir, "")
	_ = data.Close()
	_ = os.Remove(data.Name())
	// Write directly.
	cb.mu.Lock()
	_ = cb.saveLocked(state)
	cb.mu.Unlock()

	// With 60 minute cooldown, 2 hours ago should be expired.
	broken, _, err := cb.IsBroken("execute/expired", 60)
	if err != nil {
		t.Fatalf("IsBroken: %v", err)
	}
	if broken {
		t.Fatal("expected not broken after cooldown expired")
	}
}
