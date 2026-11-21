package capacity

import (
	"testing"
	"time"
)

func idleClaim(id string, priority int, amount int64, idleFor time.Duration, now time.Time) CapacityClaim {
	last := now.Add(-idleFor)
	return CapacityClaim{
		ClaimID: id, OwnerID: id, ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0),
		AmountBytes: amount, Status: StatusGranted, Priority: priority,
		ActivityState: ActivityIdle, LastActiveAt: &last,
	}
}

func TestEscalationProtectedAndActiveNeverTouched(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true

	protectedLast := now.Add(-1 * time.Hour)
	activeLast := now.Add(-1 * time.Hour)
	candidates := []CapacityClaim{
		{ClaimID: "prot", OwnerID: "prot", ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch, ActivityState: ActivityIdle, Protected: true, LastActiveAt: &protectedLast},
		{ClaimID: "act", OwnerID: "act", ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch, ActivityState: ActivityActive, LastActiveAt: &activeLast},
	}
	plan := PlanEscalation(PriorityInteractive, 6*gib, candidates, pol, now)
	if len(plan.Actions) != 0 {
		t.Fatalf("protected/active claims must never be reclaimed; got %+v", plan.Actions)
	}
	if plan.Satisfied {
		t.Error("plan should not be satisfied when nothing is eligible")
	}
}

func TestEscalationAgeAloneDoesNotReclaim(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	// Very old, but currently ACTIVE -> never eligible.
	last := now.Add(-5 * time.Hour)
	c := CapacityClaim{ClaimID: "old", OwnerID: "old", ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0), AmountBytes: 8 * gib, Status: StatusGranted, Priority: PriorityBatch, ActivityState: ActivityActive, LastActiveAt: &last}
	plan := PlanEscalation(PriorityInteractive, 6*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 0 {
		t.Fatalf("age alone must not reclaim an active claim; got %+v", plan.Actions)
	}
}

func TestEscalationWithinIdleGraceNotEligible(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy() // 60s grace
	pol.PreemptEnabled = true
	c := idleClaim("fresh", PriorityBatch, 8*gib, 5*time.Second, now)
	plan := PlanEscalation(PriorityInteractive, 6*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 0 {
		t.Fatalf("claim within idle grace must not be reclaimed; got %+v", plan.Actions)
	}
}

func TestEscalationDegradesBeforePreempt(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	c := idleClaim("whisper", PriorityService, 7*gib, 10*time.Minute, now)
	c.DegradeProfile = &DegradeProfile{Steps: []DegradeStep{
		{Label: "large-v3", AmountBytes: 7 * gib},
		{Label: "medium", AmountBytes: 2 * gib},
		{Label: "small", AmountBytes: 1 * gib},
	}}
	// Need 3 GiB; degrading large-v3->medium frees 5 GiB without any preempt.
	plan := PlanEscalation(PriorityInteractive, 3*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 1 {
		t.Fatalf("want one action; got %+v", plan.Actions)
	}
	a := plan.Actions[0]
	if a.Action != ActionRequestDegrade {
		t.Errorf("action = %q, want request-degrade (degrade before preempt)", a.Action)
	}
	if a.ToStep != "medium" {
		t.Errorf("to_step = %q, want medium (highest rung that frees enough)", a.ToStep)
	}
	if !plan.Satisfied {
		t.Error("plan should be satisfied by the degrade")
	}
}

func TestEscalationPreemptOnlyWhenNoDegradeHeadroom(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	// No degrade profile -> the only reclaim is preempt.
	c := idleClaim("blob", PriorityBatch, 8*gib, 10*time.Minute, now)
	plan := PlanEscalation(PriorityInteractive, 6*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 1 || plan.Actions[0].Action != ActionPreempt {
		t.Fatalf("want one preempt; got %+v", plan.Actions)
	}
	if !plan.Satisfied {
		t.Error("preempt should satisfy the deficit")
	}
}

func TestEscalationPreemptGatedByPolicy(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy() // PreemptEnabled false by default
	c := idleClaim("blob", PriorityBatch, 8*gib, 10*time.Minute, now)
	plan := PlanEscalation(PriorityInteractive, 6*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 1 || plan.Actions[0].Action != ActionWarn {
		t.Fatalf("with preempt disabled, want a warn (no preempt); got %+v", plan.Actions)
	}
	if plan.Satisfied {
		t.Error("plan must not be satisfied when preempt is disabled and only preempt would free space")
	}
}

func TestEscalationReclaimsLowestPriorityFirst(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	batch := idleClaim("batch", PriorityBatch, 4*gib, 10*time.Minute, now)
	service := idleClaim("service", PriorityService, 4*gib, 10*time.Minute, now)
	// Need only 3 GiB; should reclaim the batch (lowest priority) and stop.
	plan := PlanEscalation(PriorityInteractive, 3*gib, []CapacityClaim{service, batch}, pol, now)
	if len(plan.Actions) != 1 || plan.Actions[0].OwnerID != "batch" {
		t.Fatalf("want only the batch reclaimed first; got %+v", plan.Actions)
	}
}
