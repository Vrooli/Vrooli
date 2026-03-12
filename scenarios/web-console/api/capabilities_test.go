package main

import (
	"context"
	"testing"
	"time"
)

type fakeChecker struct {
	status  CapabilityStatus
	message string
	calls   int
}

func (f *fakeChecker) Check(_ context.Context) (CapabilityStatus, string) {
	f.calls++
	return f.status, f.message
}

func TestCapabilityRegistry_Resolve(t *testing.T) {
	defs := []CapabilityDef{
		{ID: "cap-a", Name: "Cap A"},
		{ID: "cap-b", Name: "Cap B"},
		{ID: "cap-c", Name: "Cap C"},
	}

	checkerA := &fakeChecker{status: StatusAvailable, message: "ok"}
	checkerB := &fakeChecker{status: StatusUnavailable, message: "down"}
	// cap-c intentionally has no checker

	checkers := map[string]StatusChecker{
		"cap-a": checkerA,
		"cap-b": checkerB,
	}

	tests := []struct {
		name       string
		capID      string
		wantStatus CapabilityStatus
		wantMsg    string
	}{
		{"available checker", "cap-a", StatusAvailable, "ok"},
		{"unavailable checker", "cap-b", StatusUnavailable, "down"},
		{"no checker defaults to unknown", "cap-c", StatusUnknown, ""},
	}

	reg := NewCapabilityRegistry(defs, checkers, time.Minute)
	states := reg.Resolve(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *CapabilityState
			for i := range states {
				if states[i].ID == tt.capID {
					found = &states[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("capability %q not found in results", tt.capID)
			}
			if found.Status != tt.wantStatus {
				t.Errorf("status = %q, want %q", found.Status, tt.wantStatus)
			}
			if found.Message != tt.wantMsg {
				t.Errorf("message = %q, want %q", found.Message, tt.wantMsg)
			}
		})
	}
}

func TestCapabilityRegistry_Caching(t *testing.T) {
	checker := &fakeChecker{status: StatusAvailable, message: "ok"}
	defs := []CapabilityDef{{ID: "cap-x", Name: "Cap X"}}
	checkers := map[string]StatusChecker{"cap-x": checker}

	reg := NewCapabilityRegistry(defs, checkers, time.Minute)

	// First call should invoke checker
	reg.Resolve(context.Background())
	if checker.calls != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", checker.calls)
	}

	// Second call within TTL should use cache
	reg.Resolve(context.Background())
	if checker.calls != 1 {
		t.Errorf("after second Resolve (cached): calls = %d, want 1", checker.calls)
	}
}

func TestCapabilityRegistry_IsAvailable(t *testing.T) {
	defs := []CapabilityDef{
		{ID: "avail", Name: "Available"},
		{ID: "unavail", Name: "Unavailable"},
	}
	checkers := map[string]StatusChecker{
		"avail":   &fakeChecker{status: StatusAvailable, message: "ok"},
		"unavail": &fakeChecker{status: StatusUnavailable, message: "down"},
	}

	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"available capability", "avail", true},
		{"unavailable capability", "unavail", false},
		{"unknown capability ID", "nonexistent", false},
	}

	reg := NewCapabilityRegistry(defs, checkers, time.Minute)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.IsAvailable(context.Background(), tt.id)
			if got != tt.want {
				t.Errorf("IsAvailable(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
