package programs

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "programs" {
		t.Fatal(GroupName)
	}
}
