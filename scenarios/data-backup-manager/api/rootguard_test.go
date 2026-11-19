package main

import "testing"

func TestRefuseRootRejectsRootAndAcceptsUser(t *testing.T) {
	if err := refuseRoot(func() int { return 0 }); err == nil {
		t.Fatal("running as root must be refused: a root DBM breaks every repository read")
	}
	if err := refuseRoot(func() int { return 1000 }); err != nil {
		t.Fatalf("a normal user must be accepted: %v", err)
	}
}
