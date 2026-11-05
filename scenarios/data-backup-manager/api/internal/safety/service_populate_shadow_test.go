package safety

import (
	"context"
	"errors"
	"testing"
)

// safetyDest is the reserved destination PopulateShadow restores from; every
// populate test seeds it so findSafetyDestination resolves.
func safetyDest() DestinationRef {
	return DestinationRef{ID: "dst-safety", Name: SafetyDestinationName, Location: "/r/baseline-safety"}
}

// populateDeps builds a service whose seams are pre-seeded for shadow population:
// the safety destination exists, one ephemeral plan with a terminal run whose
// outcomes carry a successful snapshot per target.
func populateFixture(scenario string) (Deps, *fakeRestores) {
	planName := ephemeralPlanPrefix + scenario
	run := RunDetail{
		ID:       "run-7",
		PlanID:   "plan-" + planName,
		Status:   "completed",
		Terminal: true,
		Outcomes: []TargetSnapshot{
			{TargetID: "t-pg", DestinationID: "dst-safety", SnapshotID: "snap-pg", Succeeded: true},
			{TargetID: "t-data", DestinationID: "dst-safety", SnapshotID: "snap-data", Succeeded: true},
		},
	}
	restores := &fakeRestores{}
	return Deps{
		Destinations: &fakeDestinations{list: []DestinationRef{safetyDest()}},
		Targets: &fakeTargets{byOwner: map[string][]TargetRef{
			scenario: {
				{ID: "t-pg", Owner: scenario, Name: "postgres"},
				{ID: "t-data", Owner: scenario, Name: "data"},
			},
		}},
		Plans:    &fakePlans{plans: []PlanRef{{ID: run.PlanID, Name: planName}}},
		Runs:     &fakeRuns{getByID: map[string]RunDetail{run.ID: run}, latestByPlan: map[string]RunDetail{run.PlanID: run}},
		Restores: restores,
	}, restores
}

func TestPopulateShadow_HappyPath_RestoresEachMapping(t *testing.T) {
	deps, restores := populateFixture("swarm-manager")
	svc := NewService(deps)

	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "vrooli_swarm-manager_shadow"},
		{TargetName: "data", Location: "/r/shadow/data"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	if res.RunID != "run-7" {
		t.Fatalf("run id: want run-7, got %q", res.RunID)
	}
	if len(res.Restores) != 2 || len(res.Skipped) != 0 {
		t.Fatalf("want 2 restores 0 skipped, got %d/%d (%+v)", len(res.Restores), len(res.Skipped), res)
	}
	if len(restores.calls) != 2 {
		t.Fatalf("want 2 restore calls, got %d", len(restores.calls))
	}
	// The postgres mapping must restore snap-pg of t-pg into the shadow DB name,
	// from the safety destination.
	pg := restores.calls[0]
	if pg.targetID != "t-pg" || pg.snapshotID != "snap-pg" || pg.location != "vrooli_swarm-manager_shadow" || pg.destID != "dst-safety" {
		t.Fatalf("postgres restore call wrong: %+v", pg)
	}
	if res.Restores[0].RestoreID == "" || res.Restores[0].Status != "requested" {
		t.Fatalf("restore ref not populated: %+v", res.Restores[0])
	}
}

func TestPopulateShadow_AutoResolvesLatestTerminalRun(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	svc := NewService(deps)

	// Empty run id -> resolve the latest terminal run of the ephemeral plan.
	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "vrooli_swarm-manager_shadow"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	if res.RunID != "run-7" {
		t.Fatalf("want auto-resolved run-7, got %q", res.RunID)
	}
}

func TestPopulateShadow_ExplicitRunID(t *testing.T) {
	deps, restores := populateFixture("swarm-manager")
	svc := NewService(deps)

	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "run-7", []ShadowMapping{
		{TargetName: "data", Location: "/r/shadow/data"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	if len(res.Restores) != 1 || restores.calls[0].snapshotID != "snap-data" {
		t.Fatalf("explicit run did not restore the data snapshot: %+v", res)
	}
}

func TestPopulateShadow_ExplicitRunNotTerminal_Errors(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	// Re-point the explicit run to a non-terminal state.
	deps.Runs = &fakeRuns{getByID: map[string]RunDetail{
		"run-live": {ID: "run-live", Status: "capturing", Terminal: false},
	}}
	svc := NewService(deps)

	_, err := svc.PopulateShadow(context.Background(), "swarm-manager", "run-live", []ShadowMapping{
		{TargetName: "postgres", Location: "x"},
	})
	if !errors.Is(err, ErrRunNotTerminal) {
		t.Fatalf("want ErrRunNotTerminal, got %v", err)
	}
}

func TestPopulateShadow_NoSafetyDestination_Errors(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	deps.Destinations = &fakeDestinations{} // no baseline-safety destination
	svc := NewService(deps)

	_, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "x"},
	})
	if !errors.Is(err, ErrNoSafetyBackup) {
		t.Fatalf("want ErrNoSafetyBackup, got %v", err)
	}
}

func TestPopulateShadow_NoTerminalRun_Errors(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	// Plan exists but has no terminal run.
	deps.Runs = &fakeRuns{latestByPlan: map[string]RunDetail{}}
	svc := NewService(deps)

	_, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "x"},
	})
	if !errors.Is(err, ErrNoSafetyBackup) {
		t.Fatalf("want ErrNoSafetyBackup, got %v", err)
	}
}

func TestPopulateShadow_NoEphemeralPlan_Errors(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	deps.Plans = &fakePlans{} // no ephemeral safety plan
	svc := NewService(deps)

	_, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "x"},
	})
	if !errors.Is(err, ErrNoSafetyBackup) {
		t.Fatalf("want ErrNoSafetyBackup, got %v", err)
	}
}

func TestPopulateShadow_UnknownTarget_Skipped(t *testing.T) {
	deps, restores := populateFixture("swarm-manager")
	svc := NewService(deps)

	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "redis", Location: "shadow:prefix"}, // not registered
		{TargetName: "postgres", Location: "vrooli_x_shadow"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	if len(res.Restores) != 1 || len(res.Skipped) != 1 {
		t.Fatalf("want 1 restore 1 skip, got %d/%d", len(res.Restores), len(res.Skipped))
	}
	if res.Skipped[0].TargetName != "redis" {
		t.Fatalf("want redis skipped, got %+v", res.Skipped)
	}
	if len(restores.calls) != 1 {
		t.Fatalf("the known target must still restore: %d calls", len(restores.calls))
	}
}

func TestPopulateShadow_TargetWithoutSnapshot_Skipped(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	// A run where the data target failed (no usable snapshot).
	planID := "plan-" + ephemeralPlanPrefix + "swarm-manager"
	run := RunDetail{
		ID: "run-9", PlanID: planID, Status: "partial_failed", Terminal: true,
		Outcomes: []TargetSnapshot{
			{TargetID: "t-pg", DestinationID: "dst-safety", SnapshotID: "snap-pg", Succeeded: true},
			{TargetID: "t-data", DestinationID: "dst-safety", SnapshotID: "", Succeeded: false},
		},
	}
	deps.Runs = &fakeRuns{latestByPlan: map[string]RunDetail{planID: run}}
	svc := NewService(deps)

	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "data", Location: "/r/shadow/data"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow: %v", err)
	}
	if len(res.Restores) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("want 0 restore 1 skip, got %d/%d", len(res.Restores), len(res.Skipped))
	}
}

func TestPopulateShadow_RestoreError_SkipsThatMappingOnly(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	deps.Restores = &fakeRestores{err: errors.New("destination not empty")}
	svc := NewService(deps)

	res, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", []ShadowMapping{
		{TargetName: "postgres", Location: "vrooli_x_shadow"},
	})
	if err != nil {
		t.Fatalf("PopulateShadow should not fail the call on a per-mapping restore error: %v", err)
	}
	if len(res.Restores) != 0 || len(res.Skipped) != 1 {
		t.Fatalf("want the failed mapping skipped, got %d/%d", len(res.Restores), len(res.Skipped))
	}
}

func TestPopulateShadow_Validation(t *testing.T) {
	deps, _ := populateFixture("swarm-manager")
	svc := NewService(deps)

	if _, err := svc.PopulateShadow(context.Background(), "", "", []ShadowMapping{{TargetName: "postgres", Location: "x"}}); err == nil {
		t.Fatalf("want error on empty scenario")
	}
	if _, err := svc.PopulateShadow(context.Background(), "swarm-manager", "", nil); err == nil {
		t.Fatalf("want error on empty mappings")
	}
}
