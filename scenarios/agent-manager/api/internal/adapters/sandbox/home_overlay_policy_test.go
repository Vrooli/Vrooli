package sandbox

import "testing"

// TestIsHomeOverlayPresent_ParityWithWorkspaceSandbox pins the predicate
// agent-manager applies for $HOME-relative operations.
//
// This test is the cross-scenario parity gate: the same matrix is
// asserted in workspace-sandbox/internal/policy/home_overlay_test.go
// (TestIsHomeOverlayPresent). If anyone adds a fifth state to the
// HomeOverlayState alphabet on either side, BOTH tests must be updated
// together, OR the predicate has drifted and one side lets through a
// state the other side rejects.
//
// DOC: home-overlay seam — cross-scenario contract test.
func TestIsHomeOverlayPresent_ParityWithWorkspaceSandbox(t *testing.T) {
	cases := map[HomeOverlayState]bool{
		HomeOverlayPresent:      true,
		HomeOverlayAbsent:       false,
		HomeOverlayUnsupported:  false,
		HomeOverlayNotRequested: false,
	}
	for state, want := range cases {
		if got := IsHomeOverlayPresent(state); got != want {
			t.Errorf("IsHomeOverlayPresent(%q) = %v, want %v (must match workspace-sandbox/internal/policy.IsHomeOverlayPresent)",
				state, got, want)
		}
	}
}

// TestTranslateCommand_HomeOverlayParity is the path-driven parity test:
// for every HomeOverlayState value, agent-manager's
// translateCommandToNamespace decides "$HOME-relative command + state X
// = ?" the same way workspace-sandbox's DecideHomeOverlay does
// (refuse iff state != Present).
func TestTranslateCommand_HomeOverlayParity(t *testing.T) {
	const home = "/home/u"
	cases := []struct {
		state     HomeOverlayState
		wantAllow bool
	}{
		{HomeOverlayPresent, true},
		{HomeOverlayAbsent, false},
		{HomeOverlayUnsupported, false},
		{HomeOverlayNotRequested, false},
	}
	for _, tc := range cases {
		t.Run(string(tc.state), func(t *testing.T) {
			layout := NamespaceLayout{HostHome: home, HomeOverlayState: tc.state}
			_, err := translateCommandToNamespace(home+"/.local/bin/agent", layout)
			gotAllow := err == nil
			if gotAllow != tc.wantAllow {
				t.Errorf("translateCommandToNamespace allow = %v (err=%v); want %v", gotAllow, err, tc.wantAllow)
			}
		})
	}
}
