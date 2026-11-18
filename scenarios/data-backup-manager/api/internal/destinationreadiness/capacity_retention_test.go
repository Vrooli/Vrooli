package destinationreadiness

import (
	"testing"

	"data-backup-manager/internal/capacity"
)

func TestCheckCapacityDoesNotTreatEveryGFSBucketAsAFullCopy(t *testing.T) {
	estimate := capacity.EstimateRequired(capacity.Input{
		LogicalSourceBytes: 100 << 30,
		RetainedSnapshots:  46, // latest + hourly + daily + weekly + monthly + annual
		FirstCopy:          true,
		ReserveBytes:       20 << 30,
		Policy:             capacity.PolicyAdvisory,
	})
	got := checkCapacity(157<<30, estimate)
	if got.Severity == SeverityFail {
		t.Fatalf("capacity = %+v, want the first copy admitted rather than 46 full copies", got)
	}
}

func TestCheckCapacityBlocksWhenTheImmediateWriteDoesNotFit(t *testing.T) {
	estimate := capacity.EstimateRequired(capacity.Input{
		LogicalSourceBytes: 200 << 30,
		RetainedSnapshots:  2,
		FirstCopy:          true,
		ReserveBytes:       20 << 30,
		Policy:             capacity.PolicyAdvisory,
	})
	got := checkCapacity(100<<30, estimate)
	if got.Severity != SeverityFail {
		t.Fatalf("capacity = %+v, want fail when the immediate write does not fit", got)
	}
}

func TestCheckCapacityUnknownWhenSourceEstimateIsMissing(t *testing.T) {
	estimate := capacity.EstimateRequired(capacity.Input{LogicalSourceBytes: 0, RetainedSnapshots: 2})
	got := checkCapacity(1<<40, estimate)
	if got.Severity != SeverityUnknown {
		t.Fatalf("capacity = %+v, want unknown without a source estimate", got)
	}
}

func TestCheckCapacityWarnsForUnboundedRetention(t *testing.T) {
	estimate := capacity.EstimateRequired(capacity.Input{
		LogicalSourceBytes: 1 << 30,
		RetainedSnapshots:  0,
		Policy:             capacity.PolicyAdvisory,
	})
	got := checkCapacity(100<<30, estimate)
	if got.Severity != SeverityWarning {
		t.Fatalf("capacity = %+v, want an explicit unbounded-retention warning", got)
	}
}
