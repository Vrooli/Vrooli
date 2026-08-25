package main

import "testing"

func TestBackendRegistry_StandardUnavailableWithoutPTY(t *testing.T) {
	previous := hostSupportsPTY
	hostSupportsPTY = false
	t.Cleanup(func() { hostSupportsPTY = previous })

	available, reason := probeStandard()
	if available {
		t.Fatal("standard backend reported available without a PTY implementation")
	}
	if reason != "no PTY implementation for this platform" {
		t.Fatalf("reason = %q", reason)
	}
}
