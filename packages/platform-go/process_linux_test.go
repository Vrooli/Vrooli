//go:build linux

package platform

import "testing"

func TestParseProcessStateHandlesParentheses(t *testing.T) {
	state, ok := parseProcessState([]byte("12345 (test process) Z 1 2 3 4\n"))
	if !ok || state != 'Z' {
		t.Fatalf("state=%q ok=%v, want Z/true", state, ok)
	}
}
