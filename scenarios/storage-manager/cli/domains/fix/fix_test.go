package fix

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "fix" {
		t.Fatalf("group name = %q", GroupName)
	}
}
