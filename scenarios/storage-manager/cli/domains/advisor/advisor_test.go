package advisor

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "advisor" {
		t.Fatalf("group name = %q", GroupName)
	}
}
