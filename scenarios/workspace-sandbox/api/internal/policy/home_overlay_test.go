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
func TestDecideHomeOverlay_Matrix(t *testing.T) {
	cases := []struct {
		name        string
		driverCap   bool                   // caps.HomeOverlay
		requires    bool                   // profile.RequiresHomeOverlay
		state       types.HomeOverlayState // sandbox.HomeOverlayState
		wantAllowed bool
		wantCode    string
	}{
		// Profile does NOT require home overlay → always allowed.
		{"no-req/cap/present", true, false, types.HomeOverlayPresent, true, ""},
		{"no-req/cap/absent", true, false, types.HomeOverlayAbsent, true, ""},
		{"no-req/no-cap/present", false, false, types.HomeOverlayPresent, true, ""},
		{"no-req/no-cap/absent", false, false, types.HomeOverlayAbsent, true, ""},
		{"no-req/no-cap/unsupported", false, false, types.HomeOverlayUnsupported, true, ""},
		{"no-req/cap/notrequested", true, false, types.HomeOverlayNotRequested, true, ""},

		// Profile requires home overlay; driver doesn't support it →
		// HOME_OVERLAY_UNSUPPORTED_DRIVER regardless of state.
		{"req/no-cap/present", false, true, types.HomeOverlayPresent, false, CodeHomeOverlayUnsupportedDriver},
		{"req/no-cap/absent", false, true, types.HomeOverlayAbsent, false, CodeHomeOverlayUnsupportedDriver},
		{"req/no-cap/unsupported", false, true, types.HomeOverlayUnsupported, false, CodeHomeOverlayUnsupportedDriver},
		{"req/no-cap/notrequested", false, true, types.HomeOverlayNotRequested, false, CodeHomeOverlayUnsupportedDriver},

		// Profile requires home overlay; driver supports it.
		{"req/cap/present", true, true, types.HomeOverlayPresent, true, ""},
		{"req/cap/absent", true, true, types.HomeOverlayAbsent, false, CodeHomeOverlayRequired},
		{"req/cap/unsupported", true, true, types.HomeOverlayUnsupported, false, CodeHomeOverlayRequired},
		{"req/cap/notrequested", true, true, types.HomeOverlayNotRequested, false, CodeHomeOverlayRequired},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caps := driver.DriverCapabilities{HomeOverlay: tc.driverCap}
			profile := config.IsolationProfile{ID: "test", RequiresHomeOverlay: tc.requires}
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
		})
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
