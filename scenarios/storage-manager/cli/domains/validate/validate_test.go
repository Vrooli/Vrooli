package validate

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "validate" {
		t.Fatalf("group name = %q", GroupName)
	}
}
