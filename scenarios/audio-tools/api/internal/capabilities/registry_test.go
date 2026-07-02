package capabilities_test

import (
	"context"
	"testing"
	"time"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/capabilities/mocks"
)

func TestRegistry_Resolve(t *testing.T) {
	defs := []capabilities.Def{
		{ID: "cap-a", Name: "Cap A"},
		{ID: "cap-b", Name: "Cap B"},
		{ID: "cap-c", Name: "Cap C"},
	}

	checkerA := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	checkerB := mocks.NewFakeChecker(capabilities.StatusUnavailable, "down")

	checkers := map[string]capabilities.Checker{
		"cap-a": checkerA,
		"cap-b": checkerB,
	}

	tests := []struct {
		name       string
		capID      string
		wantStatus capabilities.Status
		wantMsg    string
	}{
		{"available checker", "cap-a", capabilities.StatusAvailable, "ok"},
		{"unavailable checker", "cap-b", capabilities.StatusUnavailable, "down"},
		{"no checker defaults to unknown", "cap-c", capabilities.StatusUnknown, ""},
	}

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)
	states := reg.Resolve(context.Background())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var found *capabilities.State
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
	checker := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	defs := []capabilities.Def{{ID: "cap-x", Name: "Cap X"}}
	checkers := map[string]capabilities.Checker{"cap-x": checker}

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", got)
	}

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Errorf("after second Resolve (cached): calls = %d, want 1", got)
	}
}

func TestRegistry_ResolveLiveness(t *testing.T) {
	fullChecker := mocks.NewFakeChecker(capabilities.StatusAvailable, "full check ok")
	livenessChecker := mocks.NewFakeChecker(capabilities.StatusAvailable, "liveness ok")
	defs := []capabilities.Def{{ID: "cap-x", Name: "Cap X"}}

	t.Run("returns cached full-check results when fresh", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, time.Minute)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-x": livenessChecker})

		fullChecker.ResetCalls()
		livenessChecker.ResetCalls()

		reg.Resolve(context.Background())
		if got := fullChecker.CallCount(); got != 1 {
			t.Fatalf("full checker should be called once, got %d", got)
		}

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 0 {
			t.Errorf("liveness checker should not be called when cache is fresh, got %d calls", got)
		}
		if len(states) != 1 || states[0].Message != "full check ok" {
			t.Errorf("expected cached full check result, got %+v", states)
		}
	})

	t.Run("uses liveness checker when cache is stale", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, 0)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-x": livenessChecker})

		livenessChecker.ResetCalls()

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 1 {
			t.Errorf("liveness checker should be called once, got %d", got)
		}
		if len(states) != 1 || states[0].Message != "liveness ok" {
			t.Errorf("expected liveness result, got %+v", states)
		}
	})

	t.Run("falls back to full resolve when no liveness checker map is configured", func(t *testing.T) {
		reg := capabilities.NewRegistry(defs, map[string]capabilities.Checker{"cap-x": fullChecker}, 0)

		fullChecker.ResetCalls()

		reg.ResolveLiveness(context.Background())
		if got := fullChecker.CallCount(); got != 1 {
			t.Errorf("should fall back to full checker, got %d calls", got)
		}
	})

	t.Run("does not run full checker for missing liveness entries once liveness is configured", func(t *testing.T) {
		fullOnly := mocks.NewFakeChecker(capabilities.StatusAvailable, "full-only ok")
		reg := capabilities.NewRegistry(
			[]capabilities.Def{
				{ID: "cap-live", Name: "Cap Live"},
				{ID: "cap-full", Name: "Cap Full"},
			},
			map[string]capabilities.Checker{
				"cap-live": fullChecker,
				"cap-full": fullOnly,
			},
			0,
		)
		reg.SetLivenessCheckers(map[string]capabilities.Checker{"cap-live": livenessChecker})

		fullChecker.ResetCalls()
		fullOnly.ResetCalls()
		livenessChecker.ResetCalls()

		states := reg.ResolveLiveness(context.Background())
		if got := livenessChecker.CallCount(); got != 1 {
			t.Errorf("liveness checker should be called once, got %d", got)
		}
		if got := fullChecker.CallCount(); got != 0 {
			t.Errorf("full checker with liveness replacement should not be called, got %d", got)
		}
		if got := fullOnly.CallCount(); got != 0 {
			t.Errorf("full checker without liveness replacement should not be called, got %d", got)
		}
		if len(states) != 2 {
			t.Fatalf("states len = %d, want 2", len(states))
		}
		if states[0].Status != capabilities.StatusAvailable || states[0].Message != "liveness ok" {
			t.Errorf("state[0] = %+v, want liveness result", states[0])
		}
		if states[1].Status != capabilities.StatusUnknown || states[1].Message != "" {
			t.Errorf("state[1] = %+v, want unknown without full fallback", states[1])
		}
	})
}

func TestRegistry_ResolveForce(t *testing.T) {
	// A long TTL would normally cause Resolve to return cached values
	// after the first call. ResolveForce must bust the cache regardless.
	checker := mocks.NewFakeChecker(capabilities.StatusAvailable, "ok")
	defs := []capabilities.Def{{ID: "cap-y", Name: "Cap Y"}}
	checkers := map[string]capabilities.Checker{"cap-y": checker}

	reg := capabilities.NewRegistry(defs, checkers, time.Hour)

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after first Resolve: calls = %d, want 1", got)
	}

	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 1 {
		t.Fatalf("after second Resolve (cached): calls = %d, want 1", got)
	}

	states := reg.ResolveForce(context.Background())
	if got := checker.CallCount(); got != 2 {
		t.Errorf("after ResolveForce: calls = %d, want 2 (cache should be busted)", got)
	}
	if len(states) != 1 || states[0].Status != capabilities.StatusAvailable {
		t.Errorf("unexpected ResolveForce result: %+v", states)
	}

	// And the freshly-forced value should now be cached for the next
	// (TTL-respecting) Resolve.
	reg.Resolve(context.Background())
	if got := checker.CallCount(); got != 2 {
		t.Errorf("Resolve after ResolveForce should hit cache: calls = %d, want 2", got)
	}
}

func TestRegistry_IsAvailable(t *testing.T) {
	defs := []capabilities.Def{
		{ID: "avail", Name: "Available"},
		{ID: "unavail", Name: "Unavailable"},
	}
	checkers := map[string]capabilities.Checker{
		"avail":   mocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
		"unavail": mocks.NewFakeChecker(capabilities.StatusUnavailable, "down"),
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

	reg := capabilities.NewRegistry(defs, checkers, time.Minute)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reg.IsAvailable(context.Background(), tt.id)
			if got != tt.want {
				t.Errorf("IsAvailable(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
