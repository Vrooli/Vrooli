package destinationreadiness

import "testing"

func TestCheckCapacityDoesNotClaimFitForUnboundedRetention(t *testing.T) {
	got := checkCapacity(1<<40, 1<<30, 0)
	if got.Severity != SeverityWarning || got.Code != "capacity" {
		t.Fatalf("unbounded retention capacity = %+v, want explicit growth warning", got)
	}
}

func TestCheckCapacityUsesAllConfiguredRetentionCopies(t *testing.T) {
	got := checkCapacity(10<<30, 1<<30, 11)
	if got.Severity != SeverityFail {
		t.Fatalf("capacity below 11 retained copies = %+v, want fail", got)
	}
}

func TestCheckCapacityDoesNotOverflow(t *testing.T) {
	got := checkCapacity(1<<40, 1<<62, 4)
	if got.Severity != SeverityUnknown {
		t.Fatalf("overflowing capacity estimate = %+v, want unknown capacity", got)
	}
}
