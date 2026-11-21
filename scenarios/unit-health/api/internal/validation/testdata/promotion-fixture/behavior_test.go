package promotionfixture

import "testing"

func TestBehavior(t *testing.T) {
	got := 2 + 2
	if got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
}
