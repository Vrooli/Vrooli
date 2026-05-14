package capabilities

import (
	"context"
	"testing"
	"time"
)

type fakeChecker struct {
	status  Status
	message string
	calls   int
}

func (f *fakeChecker) Check(_ context.Context) (Status, string) {
	f.calls++
	return f.status, f.message
}

func TestRegistry_Resolve(t *testing.T) {
	defs := []Def{
		{ID: "cap-a", Name: "Cap A"},
		{ID: "cap-b", Name: "Cap B"},
		{ID: "cap-c", Name: "Cap C"},
	}

	checkerA := &fakeChecker{status: StatusAvailable, message: "ok"}
	checkerB := &fakeChecker{status: StatusUnavailable, message: "down"}

	checkers := map[string]Checker{
		"cap-a": checkerA,
		"cap-b": checkerB,
	}

	tests := []struct {
		name       string
		capID      string
		wantStatus Status
		wantMsg    string
	}{
		{"available checker", "cap-a", StatusAvailable, "ok"},
		{"unavailable checker", "cap-b", StatusUnavailable, "down"},
		{"no checker defaults to unknown", "cap-c", StatusUnknown, ""},
	}

	reg := NewRegistry(defs, checkers, time.Minute)
	states := reg.Resolve(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *State
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

func TestRegistry_Caching(t *testing.T) {
	checker := &fakeChecker{status: StatusAvailable, message: "ok"}
	defs := []Def{{ID: "cap-x", Name: "Cap X"}}
	checkers := map[string]Checker{"cap-x": checker}

	reg := NewRegistry(defs, checkers, time.Minute)

	reg.Resolve(context.Background())
	if checker.calls != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", checker.calls)
	}

	reg.Resolve(context.Background())
	if checker.calls != 1 {
		t.Errorf("after second Resolve (cached): calls = %d, want 1", checker.calls)
	}
}

func TestRegistry_ResolveLiveness(t *testing.T) {
	fullChecker := &fakeChecker{status: StatusAvailable, message: "full check ok"}
	livenessChecker := &fakeChecker{status: StatusAvailable, message: "liveness ok"}
	defs := []Def{{ID: "cap-x", Name: "Cap X"}}

	t.Run("returns cached full-check results when fresh", func(t *testing.T) {
		reg := NewRegistry(defs, map[string]Checker{"cap-x": fullChecker}, time.Minute)
		reg.SetLivenessCheckers(map[string]Checker{"cap-x": livenessChecker})

		fullChecker.calls = 0
		livenessChecker.calls = 0

		reg.Resolve(context.Background())
		if fullChecker.calls != 1 {
			t.Fatalf("full checker should be called once, got %d", fullChecker.calls)
		}

		states := reg.ResolveLiveness(context.Background())
		if livenessChecker.calls != 0 {
			t.Errorf("liveness checker should not be called when cache is fresh, got %d calls", livenessChecker.calls)
		}
		if len(states) != 1 || states[0].Message != "full check ok" {
			t.Errorf("expected cached full check result, got %+v", states)
		}
	})

	t.Run("uses liveness checker when cache is stale", func(t *testing.T) {
		reg := NewRegistry(defs, map[string]Checker{"cap-x": fullChecker}, 0)
		reg.SetLivenessCheckers(map[string]Checker{"cap-x": livenessChecker})

		livenessChecker.calls = 0

		states := reg.ResolveLiveness(context.Background())
		if livenessChecker.calls != 1 {
			t.Errorf("liveness checker should be called once, got %d", livenessChecker.calls)
		}
		if len(states) != 1 || states[0].Message != "liveness ok" {
			t.Errorf("expected liveness result, got %+v", states)
		}
	})

	t.Run("falls back to full resolve when no liveness checkers configured", func(t *testing.T) {
		reg := NewRegistry(defs, map[string]Checker{"cap-x": fullChecker}, 0)

		fullChecker.calls = 0

		reg.ResolveLiveness(context.Background())
		if fullChecker.calls != 1 {
			t.Errorf("should fall back to full checker, got %d calls", fullChecker.calls)
		}
	})
}

func TestRegistry_IsAvailable(t *testing.T) {
	defs := []Def{
		{ID: "avail", Name: "Available"},
		{ID: "unavail", Name: "Unavailable"},
	}
	checkers := map[string]Checker{
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

	reg := NewRegistry(defs, checkers, time.Minute)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.IsAvailable(context.Background(), tt.id)
			if got != tt.want {
				t.Errorf("IsAvailable(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
