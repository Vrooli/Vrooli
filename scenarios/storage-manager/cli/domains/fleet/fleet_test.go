package fleet

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "fleet" {
		t.Fatalf("group name = %q", GroupName)
	}
}
