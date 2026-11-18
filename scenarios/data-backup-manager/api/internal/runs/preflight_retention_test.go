package runs

import "testing"

func TestRetentionSnapshotCountIncludesEveryConfiguredGFSBucket(t *testing.T) {
	plan := PlanForRun{
		KeepLatest:  3,
		KeepHourly:  24,
		KeepDaily:   7,
		KeepWeekly:  4,
		KeepMonthly: 6,
		KeepAnnual:  2,
	}
	if got, want := retentionSnapshotCount(plan), 46; got != want {
		t.Fatalf("retentionSnapshotCount() = %d, want all GFS buckets summed to %d", got, want)
	}
}

func TestRetentionSnapshotCountReturnsZeroForUnboundedPolicy(t *testing.T) {
	if got := retentionSnapshotCount(PlanForRun{}); got != 0 {
		t.Fatalf("retentionSnapshotCount() for an unbounded policy = %d, want 0", got)
	}
}
