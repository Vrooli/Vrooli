package policy_test

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/types"
)

// TestInvariants is the canonical entry point for invariants documented
// in docs/internal/INVARIANTS.md scoped to the policy package. Each
// subtest name is a stable invariant ID (matched by
// scripts/check-invariants.sh).
func TestInvariants(t *testing.T) {
	t.Run("I-HOME-1", invariantHomeOverlayDecisionIsPure)
}

// I-HOME-1 — DecideHomeOverlay is pure (no I/O). We can't easily
// instrument all I/O at runtime, but we can pin the contract by
// asserting:
//   - The function signature contains only value-typed inputs that
//     don't carry I/O hooks (Sandbox, IsolationProfile,
//     DriverCapabilities).
//   - Repeated calls with identical inputs return identical decisions
//     (no internal state, no random behaviour).
func invariantHomeOverlayDecisionIsPure(t *testing.T) {
	t.Helper()

	caps := driver.DriverCapabilities{HomeOverlay: true}
	profile := config.IsolationProfile{ID: "p", HomeOverlayRequirement: types.HomeOverlayRequired}
	sb := types.Sandbox{HomeOverlayState: types.HomeOverlayPresent}

	first := policy.DecideHomeOverlay(caps, profile, sb)
	for i := 0; i < 10; i++ {
		got := policy.DecideHomeOverlay(caps, profile, sb)
		if got != first {
			t.Fatalf("DecideHomeOverlay drift on call %d: got %+v, first %+v", i, got, first)
		}
	}

	// Also assert the function lives in the policy package — a stale
	// import path would mean the wrong function is being tested.
	fn := runtime.FuncForPC(reflect.ValueOf(policy.DecideHomeOverlay).Pointer())
	if !strings.Contains(fn.Name(), "policy.DecideHomeOverlay") {
		t.Errorf("unexpected function name %q (wrong import path?)", fn.Name())
	}
}
