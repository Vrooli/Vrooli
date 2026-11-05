package runner_test

import (
	"reflect"
	"testing"

	adapterrunner "agent-manager/internal/adapters/runner"
)

// TestMissingProtectedEnforcements_SeatbeltPayload feeds a simulated macOS
// Seatbelt capability payload through the pure MissingProtectedEnforcements
// helper. Seatbelt enforces exactly the two guarantees protected mode depends
// on (filesystem-write-containment + network-deny), so despite lacking path
// illusion and a pid namespace it fully honors protected mode: nothing missing.
func TestMissingProtectedEnforcements_SeatbeltPayload(t *testing.T) {
	seatbelt := &adapterrunner.Containment{
		Level:   "required",
		Backend: "seatbelt",
		Enforcements: []string{
			adapterrunner.EnforcementFilesystemWriteContainment,
			adapterrunner.EnforcementNetworkDeny,
		},
	}
	if missing := seatbelt.MissingProtectedEnforcements(); len(missing) != 0 {
		t.Errorf("seatbelt honors protected mode (fs-write + network-deny); want none missing, got %v", missing)
	}
}

// TestMissingProtectedEnforcements_SeatbeltUnavailable pins the honest gap
// when sandbox-exec is absent and the sandbox falls through to the direct
// path: both protected-mode enforcements are reported missing.
func TestMissingProtectedEnforcements_SeatbeltUnavailable(t *testing.T) {
	none := &adapterrunner.Containment{Level: "preferred", Backend: "none", Enforcements: []string{}}
	got := none.MissingProtectedEnforcements()
	want := []string{
		adapterrunner.EnforcementFilesystemWriteContainment,
		adapterrunner.EnforcementNetworkDeny,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("uncontained sandbox: got missing %v, want %v", got, want)
	}
}

// TestMissingProtectedEnforcements_NilContainment confirms a nil report
// (containment could not be resolved) treats every protected enforcement as
// missing rather than panicking.
func TestMissingProtectedEnforcements_NilContainment(t *testing.T) {
	var c *adapterrunner.Containment
	if got := c.MissingProtectedEnforcements(); len(got) != 2 {
		t.Errorf("nil containment: want both enforcements missing, got %v", got)
	}
}
