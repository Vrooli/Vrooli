package capacity

import (
	"testing"
	"time"
)

// whisperYieldClaim builds an interactive, yield_when_idle whisper claim idle for
// `idleFor`, holding `amount` at the top of a large-v3/medium/small ladder.
func whisperYieldClaim(amount int64, idleFor time.Duration, now time.Time) CapacityClaim {
	c := idleClaim("whisper", PriorityInteractive, amount, idleFor, now)
	c.YieldWhenIdle = true
	c.DegradeProfile = &DegradeProfile{Steps: []DegradeStep{
		{Label: "large-v3", AmountBytes: 8 * gib},
		{Label: "medium", AmountBytes: 4 * gib},
		{Label: "small", AmountBytes: 2 * gib},
	}}
	return c
}

// (a) An idle, yield-opted interactive claim is reclaim-eligible for a BATCH
// requester and the planner degrades large-v3 -> medium.
func TestIdleYieldEligibleForBatchAndDegrades(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy() // idle_yield_floor = batch
	c := whisperYieldClaim(8*gib, 10*time.Minute, now)

	// Batch SD job needs 3 GiB; degrading large-v3 -> medium frees 4 GiB.
	plan := PlanEscalation(PriorityBatch, 3*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 1 {
		t.Fatalf("want one action; got %+v", plan.Actions)
	}
	a := plan.Actions[0]
	if a.Action != ActionRequestDegrade || a.ToStep != "medium" {
		t.Fatalf("action = %+v, want request-degrade to medium", a)
	}
	if !plan.Satisfied {
		t.Error("plan should be satisfied by the idle-yield degrade")
	}
}

// (b) The SAME claim while ACTIVE is never eligible — active claims are never
// demoted, even with yield_when_idle.
func TestActiveYieldClaimNotEligible(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	c := whisperYieldClaim(8*gib, 10*time.Minute, now)
	// Flip to active (as the edge would on an in-flight /asr). Interactive-active
	// is also protected in production; either guard must keep it untouched.
	c.ActivityState = ActivityActive
	c.Protected = true

	plan := PlanEscalation(PriorityBatch, 3*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 0 {
		t.Fatalf("an active yield claim must never be reclaimed; got %+v", plan.Actions)
	}
}

// (c) A non-opt-in interactive claim keeps today's strict behavior: a batch
// requester can NEVER reclaim it, idle or not (byte-identical to pre-idle-yield).
func TestNonYieldInteractiveKeepsStrictBehavior(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.PreemptEnabled = true
	c := whisperYieldClaim(8*gib, 10*time.Minute, now)
	c.YieldWhenIdle = false // opt out

	plan := PlanEscalation(PriorityBatch, 3*gib, []CapacityClaim{c}, pol, now)
	if len(plan.Actions) != 0 {
		t.Fatalf("a non-yield interactive claim must keep strict priority (no reclaim by batch); got %+v", plan.Actions)
	}
}

// (d) idle_yield_floor is respected: with the floor raised to service, a batch
// requester can no longer reclaim the idle yield claim, but a service requester can.
func TestIdleYieldFloorRespected(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	pol.IdleYieldFloor = PriorityService

	c := whisperYieldClaim(8*gib, 10*time.Minute, now)

	below := PlanEscalation(PriorityBatch, 3*gib, []CapacityClaim{c}, pol, now)
	if len(below.Actions) != 0 {
		t.Fatalf("batch requester is below the service floor; must not reclaim; got %+v", below.Actions)
	}
	atFloor := PlanEscalation(PriorityService, 3*gib, []CapacityClaim{c}, pol, now)
	if len(atFloor.Actions) != 1 || atFloor.Actions[0].Action != ActionRequestDegrade {
		t.Fatalf("service requester is at the floor; should reclaim via degrade; got %+v", atFloor.Actions)
	}
}

// Decide must set ReclaimBytes/ReclaimTargets for a batch grant that depends on
// reclaiming an idle yield-opted interactive claim — this is what makes the
// enforce-mode admission actuate the degrade ladder.
func TestDecideIdleYieldGrantNamesReclaim(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	// 8 GiB GPU fully held by an idle, yield-opted whisper claim.
	whisper := whisperYieldClaim(8*gib, 10*time.Minute, now)
	whisper.ClaimID = "clm-whisper"
	snap := snapshotWith(8, 8)

	req := vramReq(6, 0, PriorityBatch)
	v := Decide(req, snap, []CapacityClaim{whisper}, pol, now)
	if !v.Granted() {
		t.Fatalf("verdict = %q (%s), want a grant backed by idle-yield reclaim", v.Kind, v.Reason)
	}
	if v.ReclaimBytes != 6*gib {
		t.Errorf("reclaim_bytes = %d, want %d (deficit beyond free)", v.ReclaimBytes, 6*gib)
	}
	if len(v.ReclaimTargets) != 1 || v.ReclaimTargets[0] != "clm-whisper" {
		t.Errorf("reclaim_targets = %v, want [clm-whisper]", v.ReclaimTargets)
	}
}

// The non-opt-in contrast: the same fully-held GPU with a NON-yield interactive
// whisper denies the batch requester (it cannot reclaim a strictly-higher
// priority claim) — proving idle-yield is what unlocks the grant.
func TestDecideNonYieldInteractiveDeniesBatch(t *testing.T) {
	now := time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC)
	pol := DefaultPolicy()
	whisper := whisperYieldClaim(8*gib, 10*time.Minute, now)
	whisper.YieldWhenIdle = false
	snap := snapshotWith(8, 8)

	v := Decide(vramReq(6, 0, PriorityBatch), snap, []CapacityClaim{whisper}, pol, now)
	if v.Kind != VerdictDeny {
		t.Fatalf("verdict = %q, want deny (batch cannot reclaim a non-yield interactive claim)", v.Kind)
	}
}
