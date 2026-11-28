package runs

import (
	"testing"
	"time"

	"data-backup-manager/internal/capacity"
)

// The redundancy policy that used to live here (degraded writes only when a
// different device held another verified copy) is gone. Copy count never gates
// a write: the write gate proves writability with a bounded probe, and coverage
// reports how many devices hold a copy. These tests cover what preflight still
// owns — capacity inputs.

func TestBuildPreflightPlanUsesMeasuredIncrementAndRetentionCount(t *testing.T) {
	plan := PlanForRun{
		ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"},
		KeepLatest: 2, KeepHourly: 24, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 6, KeepAnnual: 2,
	}
	got := buildPreflightPlan(plan, 2<<30, true, []TargetDestinationStatus{{
		TargetID: "t1", DestinationID: "d1", LastSuccessAt: time.Now(),
	}})
	if got.MeasuredIncrementalBytes != 2<<30 {
		t.Fatalf("measured increment = %d, want the last physical growth", got.MeasuredIncrementalBytes)
	}
	if got.RetainedSnapshots != 45 {
		t.Fatalf("retained snapshots = %d, want the GFS snapshot count", got.RetainedSnapshots)
	}
	if got.FirstCopy {
		t.Fatal("first copy = true, want false when the target already succeeded on the destination")
	}
	if got.CapacityPolicy != capacity.PolicyAdvisory {
		t.Fatalf("policy = %q, want advisory", got.CapacityPolicy)
	}
	if !got.AllowWriteProbe {
		t.Fatal("run preflight should opt into the bounded write probe")
	}
}

func TestBuildPreflightPlanIsFirstCopyWhenAnyPairIsMissing(t *testing.T) {
	plan := PlanForRun{ID: "plan", TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1"}}
	got := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{{
		TargetID: "t1", DestinationID: "d1", LastSuccessAt: time.Now(),
	}})
	if !got.FirstCopy {
		t.Fatal("first copy = false, want true when t2 has no copy on d1")
	}
}

func TestBuildPreflightPlanDoesNotInventHistory(t *testing.T) {
	got := buildPreflightPlan(PlanForRun{ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}}, 0, false, nil)
	if got.MeasuredIncrementalBytes != 0 {
		t.Fatalf("measured increment = %d, want zero without history", got.MeasuredIncrementalBytes)
	}
	if !got.FirstCopy {
		t.Fatal("first copy = false, want true without any target×destination success")
	}
}
