//go:build linux

package executor

import "testing"

func TestProcessGroupHelpersFailClosedForInvalidGroups(t *testing.T) {
	if processGroupHasMembers(0) {
		t.Fatal("invalid group reported members")
	}
	if !waitForProcessGroupExit(0) {
		t.Fatal("invalid group did not finish immediately")
	}
	if !childLeakDetectionAvailable() {
		t.Fatal("Linux /proc leak detection should be available")
	}
}
