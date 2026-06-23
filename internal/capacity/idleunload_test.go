package capacity

import (
	"context"
	"testing"
	"time"
)

// idleUnloadClaim returns an active, idle claim at its top step with an
// idle_unload_ttl, last active `idleFor` ago relative to `now`.
func idleUnloadClaim(now time.Time, idleFor time.Duration) CapacityClaim {
	la := now.Add(-idleFor)
	return CapacityClaim{
		ClaimID: "clm-idle", OwnerID: "whisper", OwnerKind: OwnerKindResource,
		ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0),
		AmountBytes: 8 * gib, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		Priority: PriorityInteractive, Status: StatusGranted, ActivityState: ActivityIdle,
		LastActiveAt: &la, IdleUnloadTTL: 30 * time.Minute, DegradeProfile: ladderProfile(),
	}
}

func TestIdleUnloadEligibilityTable(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	policy := DefaultPolicy()

	mutate := func(f func(*CapacityClaim)) CapacityClaim {
		c := idleUnloadClaim(now, time.Hour) // idle 1h > 30m ttl
		f(&c)
		return c
	}

	cases := []struct {
		name string
		c    CapacityClaim
		want bool
	}{
		{"idle beyond ttl with profile", mutate(func(*CapacityClaim) {}), true},
		{"active is never unloaded", mutate(func(c *CapacityClaim) { c.ActivityState = ActivityActive }), false},
		{"sub-ttl idle is skipped", idleUnloadClaim(now, 5*time.Minute), false},
		{"no profile is skipped", mutate(func(c *CapacityClaim) { c.DegradeProfile = nil }), false},
		{"already at floor is skipped", mutate(func(c *CapacityClaim) { c.AmountBytes = 2 * gib }), false},
		{"protected is skipped", mutate(func(c *CapacityClaim) { c.Protected = true }), false},
		{"no ttl + no policy default is skipped", mutate(func(c *CapacityClaim) { c.IdleUnloadTTL = 0 }), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IdleUnloadEligible(tc.c, policy, now); got != tc.want {
				t.Errorf("IdleUnloadEligible = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestIdleUnloadHonorsPolicyDefaultTTL proves a claim without its own TTL uses the
// policy default_idle_unload_ttl.
func TestIdleUnloadHonorsPolicyDefaultTTL(t *testing.T) {
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	c := idleUnloadClaim(now, time.Hour)
	c.IdleUnloadTTL = 0
	policy := DefaultPolicy()
	if IdleUnloadEligible(c, policy, now) {
		t.Fatal("with no per-claim TTL and policy default 0, must not be eligible")
	}
	policy.DefaultIdleUnloadTTL = 30 * time.Minute
	if !IdleUnloadEligible(c, policy, now) {
		t.Fatal("policy default_idle_unload_ttl should make an idle claim eligible")
	}
}

// TestRunIdleUnloadAdvisoryVsEnforce proves advisory plans-without-actuating and
// enforce=on actuates the degrade to floor.
func TestRunIdleUnloadAdvisoryVsEnforce(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	clk := newFixedClock(now)
	store := newTestStore(t, clk)
	policy := DefaultPolicy()

	created, err := store.CreateClaim(ctx, idleUnloadClaim(now, time.Hour), time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	stored, _ := store.GetClaim(ctx, created.ClaimID)

	// Advisory: a plan is produced but nothing is actuated.
	advExec := newFakeExecutor()
	plan, _, err := RunIdleUnload(ctx, store, []CapacityClaim{stored}, advExec, policy, EnforceAdvisory, now)
	if err != nil {
		t.Fatalf("advisory run: %v", err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].ToStep != "small" {
		t.Fatalf("advisory plan = %+v, want one unload to floor 'small'", plan.Actions)
	}
	if len(advExec.calls) != 0 {
		t.Errorf("advisory must NOT actuate, got calls %v", advExec.calls)
	}
	if got, _ := store.GetClaim(ctx, created.ClaimID); got.Status != StatusGranted {
		t.Errorf("advisory left the claim %s, want granted (untouched)", got.Status)
	}

	// Enforce=on: actuates the degrade to floor.
	enfExec := newFakeExecutor()
	_, res, err := RunIdleUnload(ctx, store, []CapacityClaim{stored}, enfExec, policy, EnforceOn, now)
	if err != nil {
		t.Fatalf("enforce run: %v", err)
	}
	if len(enfExec.calls) != 1 {
		t.Fatalf("enforce must actuate once, got %v", enfExec.calls)
	}
	if len(res.Outcomes) != 1 || !res.Outcomes[0].Applied {
		t.Fatalf("enforce outcome not applied: %+v", res.Outcomes)
	}
	got, _ := store.GetClaim(ctx, created.ClaimID)
	if got.Status != StatusDegraded || got.AmountBytes != 2*gib {
		t.Errorf("after enforce unload: status=%s amount=%d, want degraded/2GiB", got.Status, got.AmountBytes)
	}
}
