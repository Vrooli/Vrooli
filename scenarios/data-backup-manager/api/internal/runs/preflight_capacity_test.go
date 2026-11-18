package runs

import (
	"testing"
	"time"

	"data-backup-manager/internal/capacity"
)

func TestBuildPreflightPlanUsesMeasuredIncrementAndRetentionCount(t *testing.T) {
	plan := PlanForRun{
		ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"},
		KeepLatest: 2, KeepHourly: 24, KeepDaily: 7, KeepWeekly: 4, KeepMonthly: 6, KeepAnnual: 2,
	}
	got := buildPreflightPlan(plan, 2<<30, true, []TargetDestinationStatus{{
		TargetID: "t1", DestinationID: "d1", LastSuccessAt: time.Now(),
	}}, map[string]string{"d1": "device:a"})
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
	}}, map[string]string{"d1": "device:a"})
	if !got.FirstCopy {
		t.Fatal("first copy = false, want true when t2 has no copy on d1")
	}
}

func TestBuildPreflightPlanAllowsDegradedOnlyWithAnotherVerifiedCopy(t *testing.T) {
	plan := PlanForRun{ID: "plan", TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1", "d2"}}
	now := time.Now()
	got := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{
		{TargetID: "t1", DestinationID: "d1", LastSuccessAt: now},
		{TargetID: "t2", DestinationID: "d1", LastSuccessAt: now},
		{TargetID: "t1", DestinationID: "d2", LastSuccessAt: now},
		{TargetID: "t2", DestinationID: "d2", LastSuccessAt: now},
	}, map[string]string{"d1": "device:a", "d2": "device:b"})
	if !got.DegradedDestinations["d1"] || !got.DegradedDestinations["d2"] {
		t.Fatalf("degraded destinations = %+v, want both when each target has another copy", got.DegradedDestinations)
	}

	missing := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{
		{TargetID: "t1", DestinationID: "d1", LastSuccessAt: now},
	}, map[string]string{"d1": "device:a", "d2": "device:b"})
	if missing.DegradedDestinations["d1"] || missing.DegradedDestinations["d2"] {
		t.Fatalf("degraded destinations = %+v, want strict when a target lacks another copy", missing.DegradedDestinations)
	}
}

func TestBuildPreflightPlanAcceptsAnotherCopyFromOutsideThePlan(t *testing.T) {
	// A plan with one destination is still allowed to write degraded when every
	// target already has a verified copy on a *different device* from another
	// plan. Restricting the rule to this plan's destinations made it inert in a
	// single-destination deployment.
	plan := PlanForRun{ID: "plan", TargetIDs: []string{"t1", "t2"}, DestinationIDs: []string{"d1"}}
	now := time.Now()
	got := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{
		{TargetID: "t1", DestinationID: "local", LastSuccessAt: now},
		{TargetID: "t2", DestinationID: "local", LastSuccessAt: now},
	}, map[string]string{"d1": "device:external", "local": "device:internal"})
	if !got.DegradedDestinations["d1"] {
		t.Fatalf("degraded destinations = %+v, want d1 allowed via the independent local copy", got.DegradedDestinations)
	}
}

func TestBuildPreflightPlanUsesRuntimeDeviceProofForLegacyDestination(t *testing.T) {
	// A legacy destination has no persisted identity. Two paths on the same
	// mounted filesystem must not count as independent copies.
	plan := PlanForRun{ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"external"}}
	now := time.Now()
	sameDevice := runtimeDeviceKey(t.TempDir())
	if sameDevice == "" {
		t.Skip("runtime device identity is unavailable on this platform")
	}
	got := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{
		{TargetID: "t1", DestinationID: "legacy", LastSuccessAt: now},
	}, map[string]string{"external": sameDevice, "legacy": sameDevice})
	if got.DegradedDestinations["external"] {
		t.Fatal("a second path on the same device must not count as an independent copy")
	}
}

func TestBuildPreflightPlanDoesNotTreatSameDeviceAsAnotherCopy(t *testing.T) {
	plan := PlanForRun{ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1", "d2"}}
	now := time.Now()
	got := buildPreflightPlan(plan, 0, false, []TargetDestinationStatus{
		{TargetID: "t1", DestinationID: "d1", LastSuccessAt: now},
		{TargetID: "t1", DestinationID: "d2", LastSuccessAt: now},
	}, map[string]string{"d1": "device:a", "d2": "device:a"})
	if got.DegradedDestinations["d1"] || got.DegradedDestinations["d2"] {
		t.Fatalf("degraded destinations = %+v, want strict when both destinations share one device", got.DegradedDestinations)
	}
}

func TestBuildPreflightPlanDoesNotInventHistory(t *testing.T) {
	got := buildPreflightPlan(PlanForRun{ID: "plan", TargetIDs: []string{"t1"}, DestinationIDs: []string{"d1"}}, 0, false, nil, nil)
	if got.MeasuredIncrementalBytes != 0 {
		t.Fatalf("measured increment = %d, want zero without history", got.MeasuredIncrementalBytes)
	}
	if !got.FirstCopy {
		t.Fatal("first copy = false, want true without any target×destination success")
	}
}
