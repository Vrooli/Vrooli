package policybridge

import "testing"

func TestRepublisherTriggerIsNonBlockingAndCoalesces(t *testing.T) {
	r := NewRepublisher()
	// Many triggers with no reader must not block and collapse to one pending.
	for i := 0; i < 5; i++ {
		r.Trigger()
	}
	got := 0
	for draining := true; draining; {
		select {
		case <-r.signal():
			got++
		default:
			draining = false
		}
	}
	if got != 1 {
		t.Fatalf("expected one coalesced pending refresh, got %d", got)
	}
}

func TestNilRepublisherIsSafe(t *testing.T) {
	var r *Republisher
	r.Trigger() // must not panic
	if r.signal() != nil {
		t.Fatal("a nil republisher must yield a nil signal (ticker-only select)")
	}
}
