package policy

import (
	"testing"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/types"
)

// TestDecideHomeOverlay_Matrix exhausts the (driver capability, profile
// requirement, sandbox state) decision space. Every cell of the matrix
// is asserted to produce the documented decision so a future refactor
// that "fixes" one row can't silently break another.
//
// I-HOME-2: HomeOverlayOptional + non-Present state ⇒ allowed +
// HOME_OVERLAY_FALLBACK code (asserted by the rows tagged "optional").
func TestDecideHomeOverlay_Matrix(t *testing.T) {
	cases := []struct {
		name        string
		driverCap   bool                         // caps.HomeOverlay
		requirement types.HomeOverlayRequirement // profile.HomeOverlayRequirement
		state       types.HomeOverlayState       // sandbox.HomeOverlayState
		wantAllowed bool
		wantCode    string
	}{
		// not_needed: always allowed, no code, regardless of driver/state.
		{"not_needed/cap/present", true, types.HomeOverlayNotNeeded, types.HomeOverlayPresent, true, ""},
		{"not_needed/cap/absent", true, types.HomeOverlayNotNeeded, types.HomeOverlayAbsent, true, ""},
		{"not_needed/no-cap/present", false, types.HomeOverlayNotNeeded, types.HomeOverlayPresent, true, ""},
		{"not_needed/no-cap/absent", false, types.HomeOverlayNotNeeded, types.HomeOverlayAbsent, true, ""},
		{"not_needed/no-cap/unsupported", false, types.HomeOverlayNotNeeded, types.HomeOverlayUnsupported, true, ""},
		{"not_needed/cap/notrequested", true, types.HomeOverlayNotNeeded, types.HomeOverlayNotRequested, true, ""},

		// optional: always allowed; non-Present state carries
		// HOME_OVERLAY_FALLBACK. The "I-HOME-2" row name pins the
		// invariant ID listed in docs/internal/INVARIANTS.md so the CI
		// scan can find it via t.Run("I-HOME-2").
		{"optional/cap/present", true, types.HomeOverlayOptional, types.HomeOverlayPresent, true, ""},
		{"I-HOME-2", true, types.HomeOverlayOptional, types.HomeOverlayAbsent, true, CodeHomeOverlayFallback},
		{"optional/no-cap/present", false, types.HomeOverlayOptional, types.HomeOverlayPresent, true, ""},
		{"optional/no-cap/absent", false, types.HomeOverlayOptional, types.HomeOverlayAbsent, true, CodeHomeOverlayFallback},
		{"optional/no-cap/unsupported", false, types.HomeOverlayOptional, types.HomeOverlayUnsupported, true, CodeHomeOverlayFallback},
		{"optional/cap/notrequested", true, types.HomeOverlayOptional, types.HomeOverlayNotRequested, true, CodeHomeOverlayFallback},

		// required + driver doesn't support → HOME_OVERLAY_UNSUPPORTED_DRIVER, regardless of state.
		{"required/no-cap/present", false, types.HomeOverlayRequired, types.HomeOverlayPresent, false, CodeHomeOverlayUnsupportedDriver},
		{"required/no-cap/absent", false, types.HomeOverlayRequired, types.HomeOverlayAbsent, false, CodeHomeOverlayUnsupportedDriver},
		{"required/no-cap/unsupported", false, types.HomeOverlayRequired, types.HomeOverlayUnsupported, false, CodeHomeOverlayUnsupportedDriver},
		{"required/no-cap/notrequested", false, types.HomeOverlayRequired, types.HomeOverlayNotRequested, false, CodeHomeOverlayUnsupportedDriver},

		// required + driver supports it.
		{"required/cap/present", true, types.HomeOverlayRequired, types.HomeOverlayPresent, true, ""},
		{"required/cap/absent", true, types.HomeOverlayRequired, types.HomeOverlayAbsent, false, CodeHomeOverlayRequired},
		{"required/cap/unsupported", true, types.HomeOverlayRequired, types.HomeOverlayUnsupported, false, CodeHomeOverlayRequired},
		{"required/cap/notrequested", true, types.HomeOverlayRequired, types.HomeOverlayNotRequested, false, CodeHomeOverlayRequired},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caps := driver.DriverCapabilities{HomeOverlay: tc.driverCap}
			profile := config.IsolationProfile{ID: "test", HomeOverlayRequirement: tc.requirement}
			sb := types.Sandbox{HomeOverlayState: tc.state}
			got := DecideHomeOverlay(caps, profile, sb)

			if got.Allowed != tc.wantAllowed {
				t.Errorf("Allowed = %v, want %v (reason=%q code=%q)", got.Allowed, tc.wantAllowed, got.Reason, got.Code)
			}
			if got.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", got.Code, tc.wantCode)
			}
			if !got.Allowed && got.Reason == "" {
				t.Error("Reason must be non-empty on refusal")
			}
			if got.Code == CodeHomeOverlayFallback && got.Reason == "" {
				t.Error("Reason must be non-empty when HOME_OVERLAY_FALLBACK is emitted")
			}
		})
	}
}

// TestDecideHomeOverlay_DefaultsToNotNeeded covers the empty-string
// (zero-value / partial-decode) HomeOverlayRequirement. The validator
// rewrites empty to "not_needed" at load time, but the policy must also
// behave safely if it ever sees one — refuse-by-default would be
// dangerous for existing flows.
func TestDecideHomeOverlay_DefaultsToNotNeeded(t *testing.T) {
	caps := driver.DriverCapabilities{HomeOverlay: false}
	profile := config.IsolationProfile{ID: "test", HomeOverlayRequirement: ""}
	sb := types.Sandbox{HomeOverlayState: types.HomeOverlayAbsent}
	got := DecideHomeOverlay(caps, profile, sb)
	if !got.Allowed || got.Code != "" {
		t.Errorf("empty HomeOverlayRequirement: got %+v, want allowed:true code:\"\"", got)
	}
}

// TestIsHomeOverlayPresent pins the sole authority on what "Present"
// means. If anyone widens this predicate (say, accepting Absent as
// "good enough"), this test fails and surfaces the change loudly.
func TestIsHomeOverlayPresent(t *testing.T) {
	cases := map[types.HomeOverlayState]bool{
		types.HomeOverlayPresent:      true,
		types.HomeOverlayAbsent:       false,
		types.HomeOverlayUnsupported:  false,
		types.HomeOverlayNotRequested: false,
	}
	for state, want := range cases {
		if got := IsHomeOverlayPresent(state); got != want {
			t.Errorf("IsHomeOverlayPresent(%q) = %v, want %v", state, got, want)
		}
	}
}
