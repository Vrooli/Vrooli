// Package circuitbreaker registry tests
// [REQ:SCS-CB-004] Test registry reset functionality
package circuitbreaker

import (
	"testing"
)

// [REQ:SCS-CB-004] Test registry creation and Get
func TestRegistryGet(t *testing.T) {
	reg := NewRegistry(DefaultConfig())

	// Get should create a new breaker
	cb1 := reg.Get("collector1")
	if cb1 == nil {
		t.Fatal("expected Get to return a circuit breaker")
	}
	if cb1.Name() != "collector1" {
		t.Errorf("expected name 'collector1', got %q", cb1.Name())
	}

	// Get again should return the same breaker
	cb2 := reg.Get("collector1")
	if cb1 != cb2 {
		t.Error("expected Get to return the same circuit breaker instance")
	}
}

// [REQ:SCS-CB-004] Test GetIfExists
func TestRegistryGetIfExists(t *testing.T) {
	reg := NewRegistry(DefaultConfig())

	// Should return nil for non-existent
	cb := reg.GetIfExists("nonexistent")
	if cb != nil {
		t.Error("expected GetIfExists to return nil for non-existent breaker")
	}

	// Create one
	reg.Get("collector1")

	// Now it should exist
	cb = reg.GetIfExists("collector1")
	if cb == nil {
		t.Error("expected GetIfExists to return the breaker after creation")
	}
}

// [REQ:SCS-CB-004] Test List
func TestRegistryList(t *testing.T) {
	reg := NewRegistry(DefaultConfig())

	reg.Get("collector1")
	reg.Get("collector2")
	reg.Get("collector3")

	breakers := reg.List()
	if len(breakers) != 3 {
		t.Errorf("expected 3 breakers, got %d", len(breakers))
	}
}

// [REQ:SCS-CB-004] Test ListStatus
func TestRegistryListStatus(t *testing.T) {
	reg := NewRegistry(DefaultConfig())

	reg.Get("collector1")
	cb2 := reg.Get("collector2")
	cb2.RecordFailure()

	statuses := reg.ListStatus()
	if len(statuses) != 2 {
		t.Errorf("expected 2 statuses, got %d", len(statuses))
	}
}

// [REQ:SCS-CB-004] Test Reset single breaker
func TestRegistryReset(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 1
	reg := NewRegistry(cfg)

	cb := reg.Get("collector1")
	cb.RecordFailure() // Trip it

	if cb.State() != Open {
		t.Errorf("expected Open state, got %v", cb.State())
	}

	// Reset via registry
	if !reg.Reset("collector1") {
		t.Error("expected Reset to return true")
	}

	if cb.State() != Closed {
		t.Errorf("expected Closed state after reset, got %v", cb.State())
	}

	// Reset non-existent should return false
	if reg.Reset("nonexistent") {
		t.Error("expected Reset to return false for non-existent breaker")
	}
}

// [REQ:SCS-CB-004] Test ResetAll
func TestRegistryResetAll(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 1
	reg := NewRegistry(cfg)

	cb1 := reg.Get("collector1")
	cb2 := reg.Get("collector2")

	cb1.RecordFailure()
	cb2.RecordFailure()

	if cb1.State() != Open || cb2.State() != Open {
		t.Error("expected both breakers to be Open")
	}

	count := reg.ResetAll()
	if count != 2 {
		t.Errorf("expected ResetAll to reset 2 breakers, got %d", count)
	}

	if cb1.State() != Closed || cb2.State() != Closed {
		t.Error("expected both breakers to be Closed after ResetAll")
	}
}

// [REQ:SCS-CB-001] Test Stats
func TestRegistryStats(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 1
	reg := NewRegistry(cfg)

	// Create 3 breakers
	cb1 := reg.Get("collector1") // Will stay closed
	cb2 := reg.Get("collector2") // Will trip to open
	reg.Get("collector3")        // Will stay closed

	cb1.RecordSuccess() // Keep closed
	cb2.RecordFailure() // Trip to open

	stats := reg.Stats()

	if stats.Total != 3 {
		t.Errorf("expected Total 3, got %d", stats.Total)
	}
	if stats.Closed != 2 {
		t.Errorf("expected Closed 2, got %d", stats.Closed)
	}
	if stats.Open != 1 {
		t.Errorf("expected Open 1, got %d", stats.Open)
	}
}

// [REQ:SCS-CB-001] Test OpenBreakers
func TestRegistryOpenBreakers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.FailureThreshold = 1
	reg := NewRegistry(cfg)

	reg.Get("collector1")
	cb2 := reg.Get("collector2")
	cb3 := reg.Get("collector3")

	cb2.RecordFailure()
	cb3.RecordFailure()

	open := reg.OpenBreakers()
	if len(open) != 2 {
		t.Errorf("expected 2 open breakers, got %d", len(open))
	}

	// Verify they're the right ones
	names := make(map[string]bool)
	for _, s := range open {
		names[s.Name] = true
	}
	if !names["collector2"] || !names["collector3"] {
		t.Error("expected collector2 and collector3 to be in open list")
	}
}
