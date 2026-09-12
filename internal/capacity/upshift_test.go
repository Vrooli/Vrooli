package capacity

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/testenv"
)

// snapshotWithGPU0 builds a single-GPU host snapshot with the given total/used GiB.
func snapshotWithGPU0(totalGiB, usedGiB int64) hostinventory.Snapshot {
	return hostinventory.Snapshot{
		GPUs: []hostinventory.GPU{{
			Index:         0,
			Name:          "Test",
			Source:        "nvidia-smi",
			VRAMBytes:     uint64(totalGiB * gib),
			VRAMUsedBytes: uint64(usedGiB * gib),
		}},
	}
}

// degradedIdleClaim is a whisper-like claim sitting at the `small` rung (degraded),
// idle, with an upshift-enabled ladder up to large-v3.
func degradedIdleClaim() CapacityClaim {
	return CapacityClaim{
		ClaimID:        "c-whisper",
		OwnerKind:      OwnerKindResource,
		OwnerID:        "whisper",
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		AmountBytes:    2 * gib, // small
		PreferredBytes: 8 * gib, // large-v3
		FloorBytes:     2 * gib,
		Priority:       PriorityInteractive,
		Status:         StatusDegraded,
		ActivityState:  ActivityIdle,
		DegradeProfile: ladderProfile(), // large-v3(8)/medium(4)/small(2), Upshift:true
	}
}

func TestPlanUpshift(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	policy := DefaultPolicy() // UpshiftHeadroom 2 GiB

	cases := []struct {
		name       string
		mutate     func(c *CapacityClaim)
		headroom   int64
		wantOK     bool
		wantToStep string
	}{
		{
			name:       "idle + ample headroom climbs to highest fitting rung (large-v3)",
			headroom:   8 * gib,
			wantOK:     true,
			wantToStep: "large-v3",
		},
		{
			name:       "headroom fits only medium (growth 2 GiB), not large-v3 (growth 6 GiB)",
			headroom:   2 * gib,
			wantOK:     true,
			wantToStep: "medium",
		},
		{
			name:     "headroom below hysteresis floor -> no-op",
			headroom: 1 * gib, // < UpshiftHeadroom (2 GiB)
			wantOK:   false,
		},
		{
			name:     "active claim -> no-op (never upshift mid-transcription)",
			mutate:   func(c *CapacityClaim) { c.ActivityState = ActivityActive },
			headroom: 8 * gib,
			wantOK:   false,
		},
		{
			name:     "profile upshift disabled -> no-op",
			mutate:   func(c *CapacityClaim) { c.DegradeProfile.Upshift = false },
			headroom: 8 * gib,
			wantOK:   false,
		},
		{
			name:     "already at preferred -> no-op (nothing to climb to)",
			mutate:   func(c *CapacityClaim) { c.AmountBytes = 8 * gib; c.Status = StatusGranted },
			headroom: 8 * gib,
			wantOK:   false,
		},
		{
			name:     "granted (not degraded) -> no-op",
			mutate:   func(c *CapacityClaim) { c.Status = StatusGranted },
			headroom: 8 * gib,
			wantOK:   false,
		},
		{
			name:     "protected -> no-op",
			mutate:   func(c *CapacityClaim) { c.Protected = true },
			headroom: 8 * gib,
			wantOK:   false,
		},
		{
			name:     "preferred caps the climb: medium ceiling, ample headroom stops at medium",
			mutate:   func(c *CapacityClaim) { c.PreferredBytes = 4 * gib }, // cap at medium
			headroom: 8 * gib,
			wantOK:   true, wantToStep: "medium",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := degradedIdleClaim()
			if tc.mutate != nil {
				tc.mutate(&c)
			}
			action, ok := PlanUpshift(c, tc.headroom, policy, now)
			if ok != tc.wantOK {
				t.Fatalf("PlanUpshift ok = %v, want %v (action %+v)", ok, tc.wantOK, action)
			}
			if !tc.wantOK {
				return
			}
			if action.Action != ActionRequestUpshift {
				t.Errorf("action = %q, want %q", action.Action, ActionRequestUpshift)
			}
			if action.ToStep != tc.wantToStep {
				t.Errorf("toStep = %q, want %q", action.ToStep, tc.wantToStep)
			}
		})
	}
}

// PlanUpshiftAll computes per-GPU free headroom from the snapshot: with the whisper
// claim degraded to small (2 GiB) on a 16 GiB GPU reporting only those 2 GiB used,
// 14 GiB is free → it climbs straight to large-v3.
func TestPlanUpshiftAllFromSnapshot(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	c := degradedIdleClaim()
	snap := snapshotWithGPU0(16, 2) // 14 GiB free
	plan := PlanUpshiftAll([]CapacityClaim{c}, snap, DefaultPolicy(), now)
	if len(plan.Actions) != 1 {
		t.Fatalf("actions = %+v, want exactly one upshift", plan.Actions)
	}
	if plan.Actions[0].ToStep != "large-v3" {
		t.Errorf("toStep = %q, want large-v3", plan.Actions[0].ToStep)
	}
}

// A busy GPU (no real free headroom because a batch consumer holds the rest)
// produces no upshift even though the claim is idle and degraded.
func TestPlanUpshiftAllNoHeadroom(t *testing.T) {
	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	whisper := degradedIdleClaim()
	// image-tools batch claim eats the rest of a 16 GiB GPU: 2 (whisper) + 13 = 15,
	// leaving 1 GiB free, below the 2 GiB hysteresis floor.
	batch := CapacityClaim{
		ClaimID: "c-batch", OwnerKind: OwnerKindOp, OwnerID: "image-tools:job-1",
		ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0), AmountBytes: 13 * gib,
		Priority: PriorityBatch, Status: StatusGranted, ActivityState: ActivityActive,
	}
	snap := snapshotWithGPU0(16, 15)
	plan := PlanUpshiftAll([]CapacityClaim{whisper, batch}, snap, DefaultPolicy(), now)
	if len(plan.Actions) != 0 {
		t.Fatalf("actions = %+v, want none (no headroom)", plan.Actions)
	}
}

// actuateUpshift invokes the resize verb with --upshift, restores status=granted,
// raises amount_bytes, and bumps the generation.
func TestActuateUpshiftRestoresClaim(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	// Seed a degraded whisper claim at small.
	created, err := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		Priority: PriorityInteractive, Status: StatusGranted, ActivityState: ActivityIdle,
		DegradeProfile: ladderProfile(),
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	degraded, err := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 2*gib)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultDegradeDebounce) // clear the debounce window

	exec := newFakeExecutor()
	plan := EscalationPlan{Actions: []EscalationAction{{
		ClaimID: degraded.ClaimID, OwnerID: "whisper", Action: ActionRequestUpshift, ToStep: "large-v3",
	}}}
	res, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Actuate() error = %v", err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor calls = %v, want one upshift", exec.calls)
	}
	if exec.calls[0] != "whisper capacity-degrade --to large-v3 --upshift" {
		t.Errorf("verb = %q, want '... --to large-v3 --upshift'", exec.calls[0])
	}
	if len(res.Outcomes) != 1 || !res.Outcomes[0].Applied {
		t.Fatalf("outcomes = %+v, want one applied", res.Outcomes)
	}
	got := mustGet(t, store, degraded.ClaimID)
	if got.Status != StatusGranted || got.AmountBytes != 8*gib {
		t.Errorf("claim = {status %q amount %d}, want granted@8GiB", got.Status, got.AmountBytes)
	}
	if got.Generation <= degraded.Generation {
		t.Errorf("generation = %d, want > %d", got.Generation, degraded.Generation)
	}
}

// A failing resize leaves the claim degraded (never strands it upshifted in the
// ledger when the container did not actually grow).
func TestActuateUpshiftFailureLeavesClaim(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	created, _ := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		Priority: PriorityInteractive, Status: StatusGranted, ActivityState: ActivityIdle,
		DegradeProfile: ladderProfile(),
	}, time.Hour)
	degraded, _ := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 2*gib)
	clk.Advance(2 * DefaultDegradeDebounce)

	exec := newFakeExecutor()
	exec.failOn["whisper"] = true
	plan := EscalationPlan{Actions: []EscalationAction{{
		ClaimID: degraded.ClaimID, OwnerID: "whisper", Action: ActionRequestUpshift, ToStep: "large-v3",
	}}}
	res, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Actuate() error = %v (failure must be non-fatal)", err)
	}
	if len(res.Outcomes) != 1 || res.Outcomes[0].Err == "" || res.Outcomes[0].Applied {
		t.Fatalf("outcome = %+v, want one un-applied error", res.Outcomes)
	}
	got := mustGet(t, store, degraded.ClaimID)
	if got.Status != StatusDegraded || got.AmountBytes != 2*gib {
		t.Errorf("claim = {status %q amount %d}, want UNCHANGED degraded@2GiB", got.Status, got.AmountBytes)
	}
}

// Within the debounce window an upshift is skipped (anti-thrash with degrade).
func TestActuateUpshiftDebounced(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	created, _ := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		Priority: PriorityInteractive, Status: StatusGranted, ActivityState: ActivityIdle,
		DegradeProfile: ladderProfile(),
	}, time.Hour)
	degraded, _ := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 2*gib)
	// Do NOT advance the clock: the degrade just happened, inside the debounce window.

	exec := newFakeExecutor()
	plan := EscalationPlan{Actions: []EscalationAction{{
		ClaimID: degraded.ClaimID, OwnerID: "whisper", Action: ActionRequestUpshift, ToStep: "large-v3",
	}}}
	res, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.calls) != 0 {
		t.Errorf("executor ran during debounce window: %v", exec.calls)
	}
	if len(res.Outcomes) != 1 || !res.Outcomes[0].Skipped {
		t.Errorf("outcome = %+v, want skipped", res.Outcomes)
	}
}

// RunUpshift actuates ONLY under enforce=on; advisory returns the plan without
// touching the executor.
func TestRunUpshiftGatedByEnforce(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	created, _ := store.CreateClaim(ctx, CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "whisper", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 8 * gib, PreferredBytes: 8 * gib, FloorBytes: 2 * gib,
		Priority: PriorityInteractive, Status: StatusGranted, ActivityState: ActivityIdle,
		DegradeProfile: ladderProfile(),
	}, time.Hour)
	if _, err := store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 2*gib); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultDegradeDebounce)
	snap := snapshotWithGPU0(16, 2)
	active, _ := store.ListClaims(ctx, ClaimFilter{Statuses: ActiveClaimStatuses()})

	// Advisory: a plan is produced but nothing is actuated.
	exec := newFakeExecutor()
	plan, _, err := RunUpshift(ctx, store, active, snap, exec, DefaultPolicy(), EnforceAdvisory, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Actions) != 1 {
		t.Fatalf("advisory plan actions = %+v, want one (recommendation)", plan.Actions)
	}
	if len(exec.calls) != 0 {
		t.Errorf("advisory mode actuated: %v", exec.calls)
	}
	if got := mustGet(t, store, created.ClaimID); got.Status != StatusDegraded {
		t.Errorf("advisory mode mutated the claim to %q", got.Status)
	}

	// Enforce: the same plan actuates.
	exec = newFakeExecutor()
	_, res, err := RunUpshift(ctx, store, active, snap, exec, DefaultPolicy(), EnforceOn, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(exec.calls) != 1 || len(res.Outcomes) != 1 || !res.Outcomes[0].Applied {
		t.Fatalf("enforce mode did not actuate: calls=%v outcomes=%+v", exec.calls, res.Outcomes)
	}
	if got := mustGet(t, store, created.ClaimID); got.Status != StatusGranted || got.AmountBytes != 8*gib {
		t.Errorf("enforce upshift = {status %q amount %d}, want granted@8GiB", got.Status, got.AmountBytes)
	}
}
