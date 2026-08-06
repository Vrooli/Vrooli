package bindings

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "bindings" {
		t.Fatal(GroupName)
	}
}
