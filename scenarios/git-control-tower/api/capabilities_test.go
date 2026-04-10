package main

import (
	"context"
	"testing"
	"time"
)

// FakeStatusChecker is a test double for StatusChecker.
type FakeStatusChecker struct {
	Status  CapabilityStatus
	Message string
}

func (f *FakeStatusChecker) Check(_ context.Context) (CapabilityStatus, string) {
	return f.Status, f.Message
}

func TestCapabilityRegistry_Resolve_AllAvailable(t *testing.T) {
	defs := []CapabilityDef{
		{ID: "svc-a", Name: "Service A", DependencyKind: DependencyScenario, DependencySlug: "svc-a", Features: []string{"feat1"}},
		{ID: "svc-b", Name: "Service B", DependencyKind: DependencyScenario, DependencySlug: "svc-b", Features: []string{"feat2"}},
	}
	checkers := map[string]StatusChecker{
		"svc-a": &FakeStatusChecker{Status: StatusAvailable, Message: "running"},
		"svc-b": &FakeStatusChecker{Status: StatusAvailable, Message: "running"},
	}
	reg := NewCapabilityRegistry(defs, checkers, time.Minute)

	states := reg.Resolve(context.Background())
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	for _, s := range states {
		if s.Status != StatusAvailable {
			t.Errorf("expected %q to be available, got %s", s.ID, s.Status)
		}
		if s.CheckedAt == "" {
			t.Errorf("expected CheckedAt to be set for %q", s.ID)
		}
	}
}

func TestCapabilityRegistry_Resolve_MixedStatus(t *testing.T) {
	defs := []CapabilityDef{
		{ID: "up", Name: "Up"},
		{ID: "down", Name: "Down"},
	}
	checkers := map[string]StatusChecker{
		"up":   &FakeStatusChecker{Status: StatusAvailable, Message: "ok"},
		"down": &FakeStatusChecker{Status: StatusUnavailable, Message: "not running"},
	}
	reg := NewCapabilityRegistry(defs, checkers, time.Minute)

	states := reg.Resolve(context.Background())
	statusMap := map[string]CapabilityStatus{}
	for _, s := range states {
		statusMap[s.ID] = s.Status
	}

	if statusMap["up"] != StatusAvailable {
		t.Errorf("expected 'up' to be available, got %s", statusMap["up"])
	}
	if statusMap["down"] != StatusUnavailable {
		t.Errorf("expected 'down' to be unavailable, got %s", statusMap["down"])
	}
}

func TestCapabilityRegistry_CacheTTL(t *testing.T) {
	callCount := 0
	checker := &CountingChecker{Status: StatusAvailable}
	defs := []CapabilityDef{{ID: "svc", Name: "SVC"}}
	checkers := map[string]StatusChecker{"svc": checker}

	reg := NewCapabilityRegistry(defs, checkers, 50*time.Millisecond)
	ctx := context.Background()

	// First call populates cache.
	reg.Resolve(ctx)
	callCount = checker.Count

	// Second call within TTL should use cache.
	reg.Resolve(ctx)
	if checker.Count != callCount {
		t.Error("expected cached result within TTL")
	}

	// Wait for cache to expire.
	time.Sleep(60 * time.Millisecond)

	reg.Resolve(ctx)
	if checker.Count == callCount {
		t.Error("expected fresh check after TTL expiry")
	}
}

func TestCapabilityRegistry_IsAvailable(t *testing.T) {
	defs := []CapabilityDef{{ID: "svc", Name: "SVC"}}
	checkers := map[string]StatusChecker{
		"svc": &FakeStatusChecker{Status: StatusAvailable},
	}
	reg := NewCapabilityRegistry(defs, checkers, time.Minute)

	if !reg.IsAvailable(context.Background(), "svc") {
		t.Error("expected svc to be available")
	}
}

func TestCapabilityRegistry_UnknownID(t *testing.T) {
	reg := NewCapabilityRegistry(nil, nil, time.Minute)
	if reg.IsAvailable(context.Background(), "nonexistent") {
		t.Error("expected false for unknown capability ID")
	}
}

func TestCapabilityRegistry_NoChecker(t *testing.T) {
	defs := []CapabilityDef{{ID: "orphan", Name: "Orphan"}}
	reg := NewCapabilityRegistry(defs, map[string]StatusChecker{}, time.Minute)

	states := reg.Resolve(context.Background())
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if states[0].Status != StatusUnknown {
		t.Errorf("expected unknown status for capability with no checker, got %s", states[0].Status)
	}
}

// CountingChecker tracks how many times Check is called.
type CountingChecker struct {
	Status CapabilityStatus
	Count  int
}

func (c *CountingChecker) Check(_ context.Context) (CapabilityStatus, string) {
	c.Count++
	return c.Status, "counted"
}
