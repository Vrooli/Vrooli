//go:build linux

package securestore

import "testing"

// The kernel truncates /proc/<pid>/comm to 15 bytes, so the daemon appears
// there as "gnome-keyring-d". Comparing the full name never matched, which
// disabled the stale-daemon check on every Linux host without ever failing.
func TestIsKeyringDaemonComm(t *testing.T) {
	cases := []struct {
		comm string
		want bool
	}{
		{"gnome-keyring-daemon", true},
		{"gnome-keyring-d", true},
		{"gnome-keyring", false},
		{"gnome-keyring-x", false},
		{"", false},
		{"systemd", false},
	}
	for _, testCase := range cases {
		if got := isKeyringDaemonComm(testCase.comm); got != testCase.want {
			t.Fatalf("isKeyringDaemonComm(%q) = %t, want %t", testCase.comm, got, testCase.want)
		}
	}
}
