package sessions

import "testing"

func TestGroupName(t *testing.T) {
	if GroupName != "sessions" {
		t.Fatal(GroupName)
	}
}
