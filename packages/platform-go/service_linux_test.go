//go:build linux

package platform

import "testing"

func TestParseSystemdState(t *testing.T) {
	// Fixture values captured from `systemctl is-active` and
	// `systemctl is-enabled` on a Linux systemd host.
	for _, tc := range []struct {
		raw  string
		want ServiceState
	}{
		{"active\n", ServiceStateRunning},
		{"inactive\n", ServiceStateStopped},
		{"failed\n", ServiceStateFailed},
		{"garbage\n", ServiceStateUnknown},
	} {
		if got := parseSystemdState(tc.raw); got != tc.want {
			t.Errorf("parseSystemdState(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
