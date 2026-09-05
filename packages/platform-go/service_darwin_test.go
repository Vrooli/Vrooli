//go:build darwin

package platform

import "testing"

func TestParseLaunchctlStateFromPrintFixture(t *testing.T) {
	// Fixture shape captured from `launchctl print user/501/com.vrooli.autoheal`
	// on macOS: launchctl print emits a property-list-like state line, not the
	// tabular output of `launchctl list`.
	raw := "path = /Users/alice/Library/LaunchAgents/com.vrooli.autoheal.plist\nstate = running\npid = 1234\n"
	if got := parseLaunchctlState(raw, nil); got != ServiceStateRunning {
		t.Fatalf("state = %q, want running", got)
	}
}
