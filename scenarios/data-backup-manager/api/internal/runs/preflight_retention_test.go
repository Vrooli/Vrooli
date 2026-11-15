package runs

import "testing"

func TestRetentionCopiesIncludesEveryConfiguredGFSBucket(t *testing.T) {
	plan := PlanForRun{
		KeepLatest:  3,
		KeepHourly:  24,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
		KeepAnnual:  2,
	}
	if got, want := retentionCopies(plan), 46; got != want {
		t.Fatalf("retentionCopies() = %d, want all GFS buckets summed to %d", got, want)
	}
}

func TestRetentionCopiesReturnsZeroForUnboundedPolicy(t *testing.T) {
	if got := retentionCopies(PlanForRun{}); got != 0 {
		t.Fatalf("retentionCopies() for an unbounded policy = %d, want 0", got)
	}
}
