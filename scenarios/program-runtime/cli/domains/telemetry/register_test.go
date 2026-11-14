package telemetry

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "telemetry" {
		t.Fatal(GroupName)
	}
}
