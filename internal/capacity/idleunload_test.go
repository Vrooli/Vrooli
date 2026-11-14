package capacity

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

type verifiedUnloadExecutor struct {
	store *SQLiteStore
}

func (e verifiedUnloadExecutor) Apply(ctx context.Context, owner, _ string, _ []string) error {
	claim, err := e.store.GetClaim(ctx, "clm-idle")
	if err != nil {
		return err
	}
	startedAt := time.Now().UTC()
	preAt := *claim.ObservedAt
	transition := ResidencyTransition{
		ID: fmt.Sprintf("%s-unload-%d", owner, startedAt.UnixNano()), Model: claim.DegradeProfile.Model,
		Desired: ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}, Reason: "idle_unload", StartedAt: startedAt,
		PreFootprint: ResidencyFootprint{Bytes: claim.ObservedBytes, Source: FootprintSourceMeasured, ObservedAt: preAt, Verified: true},
	}
	started, err := e.store.BeginResidencyTransition(ctx, claim.ClaimID, claim.Generation, transition)
	if err != nil {
		return err
	}
	postAt := startedAt.Add(time.Millisecond)
	time.Sleep(2 * time.Millisecond)
	if _, err := e.store.RecordObserved(ctx, claim.ClaimID, 0, 0, postAt, &AbsenceProof{Generation: claim.Generation, ObservedAt: postAt}); err != nil {
		return err
	}
	completedAt := time.Now().UTC()
	transition.CompletedAt = &completedAt
	transition.Observed = ResidencyPlacement{Backend: ResidencyBackendUnloaded, Known: true}
	transition.Footprint = ResidencyFootprint{Bytes: 0, Source: FootprintSourceMeasured, ObservedAt: postAt, Verified: true}
	transition.EffectsVerified = true
	completed, err := e.store.CompleteResidencyTransition(ctx, claim.ClaimID, claim.Generation, started.Revision, transition)
	if err != nil {
		return err
	}
	_, err = e.store.ReleaseClaimAfterResidency(ctx, owner, claim.ClaimID, claim.Generation, transition.ID, completed.Revision, transition.Model)
	return err
}

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
		{"CPU residency can still unload", mutate(func(c *CapacityClaim) {
			c.AmountBytes = 0
			c.DegradeProfile = &DegradeProfile{Model: residencyTestModel, Steps: []DegradeStep{{Label: "vulkan", AmountBytes: 8 * gib}, {Label: "cpu", AmountBytes: 0}, {Label: "unloaded", AmountBytes: 0}}}
		}), true},
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
	clk := testenv.NewClock(now)
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
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

func TestRunIdleUnloadUsesVerifiedOwnerTransitionAndMeasuredFreedBytes(t *testing.T) {
	ctx := context.Background()
	now := time.Now().UTC()
	store := testenv.NewSQLiteStore(t, "capacity-unload.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(ctx, Config{DBPath: path})
	})
	claim := idleUnloadClaim(now, time.Hour)
	claim.ClaimID = "clm-idle"
	claim.AmountBytes = 1 << 30
	claim.PreferredBytes = 1 << 30
	claim.FloorBytes = 0
	claim.DegradeProfile.Model = residencyTestModel
	claim.DegradeProfile.Steps = []DegradeStep{{Label: "vulkan", AmountBytes: 1 << 30}, {Label: "cpu", AmountBytes: 0}, {Label: "unloaded", AmountBytes: 0}}
	claim.ObservedBytes = 1 << 30
	observedAt := now.Add(-time.Second)
	claim.ObservedAt = &observedAt
	created, err := store.CreateClaim(ctx, claim, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := store.GetClaim(ctx, created.ClaimID)
	if err != nil {
		t.Fatal(err)
	}
	policy := DefaultPolicy()
	plan, result, err := RunIdleUnload(ctx, store, []CapacityClaim{stored}, verifiedUnloadExecutor{store: store}, policy, EnforceOn, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 || plan.Actions[0].Action != ActionRequestResidencyUnload || plan.Actions[0].ToStep != "unloaded" {
		t.Fatalf("idle unload plan = %+v", plan.Actions)
	}
	if len(result.Outcomes) != 1 || !result.Outcomes[0].Applied || result.FreedBytes != 1<<30 || result.Outcomes[0].FreedBytes != 1<<30 {
		t.Fatalf("verified unload result = %+v", result)
	}
	if got, err := store.GetClaim(ctx, created.ClaimID); err != nil || got.Status != StatusReleased {
		t.Fatalf("unloaded claim = %+v, %v", got, err)
	}
}

func TestIdleDwellStartsWhenWorkEnds(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	c := idleLadderClaim(t, store, "working", PriorityService)
	c, err := store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(30 * time.Minute)
	c, err = store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityIdle)
	if err != nil {
		t.Fatal(err)
	}
	ended := clk.Now()
	p := DefaultPolicy()
	p.DefaultIdleUnloadTTL = 5 * time.Minute
	if isReclaimEligible(c, p.IdleGrace, clk.Now()) || IdleUnloadEligible(c, p, clk.Now()) {
		t.Fatal("work ending must begin a fresh idle dwell")
	}
	clk.Advance(30 * time.Second)
	c, err = store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityIdle)
	if err != nil {
		t.Fatal(err)
	}
	c, err = store.HeartbeatClaim(ctx, c.ClaimID, c.Generation, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if c.LastActiveAt == nil || !c.LastActiveAt.Equal(ended) {
		t.Fatalf("idle boundary moved or was not recorded: %+v, want %v", c.LastActiveAt, ended)
	}
	clk.Advance(30 * time.Second)
	if !isReclaimEligible(c, p.IdleGrace, clk.Now()) || IdleUnloadEligible(c, p, clk.Now()) {
		t.Fatal("grace and unload boundaries must remain independent")
	}
	clk.Advance(4 * time.Minute)
	if !IdleUnloadEligible(c, p, clk.Now()) {
		t.Fatal("unload must become eligible at the original idle boundary plus TTL")
	}
}
