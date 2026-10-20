package capacity

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/testenv"
)

// fakeExecutor records the degrade verbs it was asked to run and can be made to
// fail, so the actuator's success/failure/idempotency paths are exercised with
// no real exec.
type fakeExecutor struct {
	calls   []string
	failOn  map[string]bool // owner -> should fail
	callsBy map[string]int
}

func newFakeExecutor() *fakeExecutor {
	return &fakeExecutor{failOn: map[string]bool{}, callsBy: map[string]int{}}
}

func (f *fakeExecutor) Apply(_ context.Context, owner, verb string, argv []string) error {
	f.calls = append(f.calls, owner+" "+verb+" "+joinArgs(argv))
	f.callsBy[owner]++
	if f.failOn[owner] {
		return errors.New("simulated adopter resize failure")
	}
	return nil
}

func joinArgs(a []string) string {
	out := ""
	for i, s := range a {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}

// ladderProfile is a whisper-like degrade ladder.
func ladderProfile() *DegradeProfile {
	return &DegradeProfile{
		Steps: []DegradeStep{
			{Label: "large-v3", AmountBytes: 8 << 30},
			{Label: "medium", AmountBytes: 4 << 30},
			{Label: "small", AmountBytes: 2 << 30},
		},
		Apply:   DegradeApply{Verb: "capacity-degrade", Argv: []string{"--to", "{label}"}},
		Upshift: true,
	}
}

// idleClaim builds an idle, unprotected claim with a ladder profile at its top
// step, created long enough ago to be past idle-grace.
func idleLadderClaim(t *testing.T, store *SQLiteStore, owner string, priority int) CapacityClaim {
	t.Helper()
	c := CapacityClaim{
		OwnerKind:      OwnerKindResource,
		OwnerID:        owner,
		ResourceKind:   ResourceKindVRAM,
		GPUIndex:       gpu(0),
		AmountBytes:    8 << 30,
		PreferredBytes: 8 << 30,
		FloorBytes:     2 << 30,
		Priority:       priority,
		Status:         StatusGranted,
		ActivityState:  ActivityIdle,
		DegradeProfile: ladderProfile(),
	}
	created, err := store.CreateClaim(context.Background(), c, time.Hour)
	if err != nil {
		t.Fatalf("CreateClaim(%s) error = %v", owner, err)
	}
	return created
}

// A lower-priority idle holder is degraded one step (actuator invoked + ledger
// recorded) to free space for a higher-priority requester.
func TestActuateDegradesIdleLowerPriorityHolder(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	holder := idleLadderClaim(t, store, "whisper", PriorityService)
	clk.Advance(2 * DefaultIdleGrace) // past idle-grace

	// Need to free 4 GiB; whisper large-v3(8) -> medium(4) frees 4 GiB.
	plan := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, DefaultPolicy(), clk.Now())
	exec := newFakeExecutor()

	res, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Actuate() error = %v", err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("executor calls = %v, want exactly one degrade", exec.calls)
	}
	if res.FreedBytes != 4<<30 {
		t.Errorf("freed = %d, want 4 GiB", res.FreedBytes)
	}
	got := mustGet(t, store, holder.ClaimID)
	if got.Status != StatusDegraded || got.AmountBytes != 4<<30 {
		t.Errorf("holder = {status %q amount %d}, want degraded@4GiB", got.Status, got.AmountBytes)
	}
}

// Protected and active claims are NEVER eligible — the planner excludes them, so
// the actuator never produces an action against them and never mutates them.
func TestActuateNeverTouchesProtectedOrActive(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})

	// A protected lower-priority holder and an active lower-priority holder, both
	// otherwise reclaim-eligible (idle past grace / lower priority).
	protected := idleLadderClaim(t, store, "protected-srv", PriorityBatch)
	active := idleLadderClaim(t, store, "active-srv", PriorityBatch)
	if _, err := store.ReportActivity(ctx, active.ClaimID, active.Generation, ActivityActive); err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultIdleGrace)

	candidates := []CapacityClaim{mustGet(t, store, protected.ClaimID), mustGet(t, store, active.ClaimID)}
	candidates[0].Protected = true // the protected holder

	plan := PlanEscalation(PriorityInteractive, 4<<30, candidates, DefaultPolicy(), clk.Now())
	if len(plan.Actions) != 0 {
		t.Fatalf("plan produced actions against protected/active holders: %+v", plan.Actions)
	}
	res, err := Actuate(ctx, plan, store, newFakeExecutor(), DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Actuate() error = %v", err)
	}
	if len(res.Outcomes) != 0 {
		t.Errorf("actuation outcomes = %+v, want none", res.Outcomes)
	}
	if got := mustGet(t, store, protected.ClaimID); got.Status == StatusDegraded {
		t.Error("protected claim must never be degraded")
	}
	if got := mustGet(t, store, active.ClaimID); got.Status == StatusDegraded {
		t.Error("active claim must never be degraded")
	}
}

// Actuator failure is non-fatal: the claim is left unchanged and a warn is
// surfaced (never strand a resource off-GPU).
func TestActuateFailureIsNonFatalAndLeavesClaim(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	holder := idleLadderClaim(t, store, "whisper", PriorityService)
	clk.Advance(2 * DefaultIdleGrace)

	plan := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, DefaultPolicy(), clk.Now())
	exec := newFakeExecutor()
	exec.failOn["whisper"] = true

	res, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
	if err != nil {
		t.Fatalf("Actuate() error = %v (failure must be non-fatal, not returned)", err)
	}
	if res.FreedBytes != 0 {
		t.Errorf("freed = %d, want 0 (actuation failed)", res.FreedBytes)
	}
	if len(res.Outcomes) != 1 || res.Outcomes[0].Err == "" || res.Outcomes[0].Applied {
		t.Fatalf("outcome = %+v, want one un-applied outcome carrying the error", res.Outcomes)
	}
	got := mustGet(t, store, holder.ClaimID)
	if got.Status != StatusGranted || got.AmountBytes != 8<<30 {
		t.Errorf("holder = {status %q amount %d}, want UNCHANGED granted@8GiB", got.Status, got.AmountBytes)
	}
}

// Re-degrading an already-degraded target within the debounce window is skipped;
// and re-degrading to a step it's already at is an idempotent no-op.
func TestActuateDebounceAndIdempotency(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	holder := idleLadderClaim(t, store, "whisper", PriorityService)
	clk.Advance(2 * DefaultIdleGrace)

	policy := DefaultPolicy() // DegradeDebounce 30s
	exec := newFakeExecutor()

	// First degrade to medium succeeds.
	plan := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, policy, clk.Now())
	if _, err := Actuate(ctx, plan, store, exec, policy, clk.Now()); err != nil {
		t.Fatal(err)
	}
	if exec.callsBy["whisper"] != 1 {
		t.Fatalf("first degrade calls = %d, want 1", exec.callsBy["whisper"])
	}

	// Idempotent: a plan that targets the step it's already at (medium) is a no-op.
	idem := EscalationPlan{Actions: []EscalationAction{{ClaimID: holder.ClaimID, Generation: mustGet(t, store, holder.ClaimID).Generation, OwnerID: "whisper", Action: ActionRequestDegrade, ToStep: "medium", FreesBytes: 0}}}
	res, err := Actuate(ctx, idem, store, exec, policy, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if exec.callsBy["whisper"] != 1 || !res.Outcomes[0].Skipped {
		t.Errorf("idempotent re-degrade ran the verb again or did not skip: calls=%d outcome=%+v", exec.callsBy["whisper"], res.Outcomes[0])
	}

	// Debounce: a degrade to a LOWER step (small) within the window is skipped.
	clk.Advance(5 * time.Second) // < 30s debounce
	deeper := EscalationPlan{Actions: []EscalationAction{{ClaimID: holder.ClaimID, Generation: mustGet(t, store, holder.ClaimID).Generation, OwnerID: "whisper", Action: ActionRequestDegrade, ToStep: "small", FreesBytes: 2 << 30}}}
	res, err = Actuate(ctx, deeper, store, exec, policy, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if exec.callsBy["whisper"] != 1 || !res.Outcomes[0].Skipped {
		t.Errorf("debounced degrade ran the verb or did not skip: calls=%d outcome=%+v", exec.callsBy["whisper"], res.Outcomes[0])
	}

	// Past the debounce window it runs again.
	clk.Advance(policy.DegradeDebounce + time.Second)
	res, err = Actuate(ctx, EscalationPlan{Actions: []EscalationAction{{ClaimID: holder.ClaimID, Generation: mustGet(t, store, holder.ClaimID).Generation, OwnerID: "whisper", Action: ActionRequestDegrade, ToStep: "small", FreesBytes: 2 << 30}}}, store, exec, policy, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if exec.callsBy["whisper"] != 2 || !res.Outcomes[0].Applied {
		t.Errorf("post-debounce degrade did not run: calls=%d outcome=%+v", exec.callsBy["whisper"], res.Outcomes[0])
	}
}

// Degrade is always preferred over preempt: a holder WITH a degrade rung is
// degraded (not preempted) even when preempt is enabled.
func TestActuatePrefersDegradeOverPreempt(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	holder := idleLadderClaim(t, store, "whisper", PriorityService)
	clk.Advance(2 * DefaultIdleGrace)

	policy := DefaultPolicy()
	policy.PreemptEnabled = true // even so, degrade is chosen because a rung exists

	plan := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, policy, clk.Now())
	for _, a := range plan.Actions {
		if a.Action == ActionPreempt {
			t.Fatalf("plan chose preempt while a degrade rung exists: %+v", plan.Actions)
		}
	}
	exec := newFakeExecutor()
	if _, err := Actuate(ctx, plan, store, exec, policy, clk.Now()); err != nil {
		t.Fatal(err)
	}
	if got := mustGet(t, store, holder.ClaimID); got.Status != StatusDegraded {
		t.Errorf("holder status = %q, want degraded (not preempted)", got.Status)
	}
}

// Preempt only runs when policy.PreemptEnabled; a profile-less idle holder is
// preempted (recorded) when enabled, and merely warned when disabled.
func TestActuatePreemptGatedByPolicy(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 6, 22, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) {
		return NewSQLiteStore(context.Background(), Config{DBPath: path, Clock: clk})
	})
	// A profile-less holder: no degrade rung, so reclaim must preempt.
	c := CapacityClaim{
		OwnerKind: OwnerKindResource, OwnerID: "bulk", ResourceKind: ResourceKindVRAM,
		GPUIndex: gpu(0), AmountBytes: 4 << 30, PreferredBytes: 4 << 30, FloorBytes: 4 << 30,
		Priority: PriorityBatch, Status: StatusGranted, ActivityState: ActivityIdle,
	}
	holder, err := store.CreateClaim(ctx, c, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultIdleGrace)

	// Disabled: plan warns, actuator does not preempt.
	disabled := DefaultPolicy()
	planOff := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, disabled, clk.Now())
	if _, err := Actuate(ctx, planOff, store, newFakeExecutor(), disabled, clk.Now()); err != nil {
		t.Fatal(err)
	}
	if got := mustGet(t, store, holder.ClaimID); got.Status == StatusPreempted {
		t.Error("claim preempted while preempt disabled")
	}

	// Enabled: plan preempts, actuator records it.
	enabled := DefaultPolicy()
	enabled.PreemptEnabled = true
	planOn := PlanEscalation(PriorityInteractive, 4<<30, []CapacityClaim{mustGet(t, store, holder.ClaimID)}, enabled, clk.Now())
	res, err := Actuate(ctx, planOn, store, newFakeExecutor(), enabled, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.FreedBytes != 0 {
		t.Errorf("freed = %d, want zero from ledger-only intent", res.FreedBytes)
	}
	if got := mustGet(t, store, holder.ClaimID); got.Status != StatusPreempted {
		t.Errorf("holder status = %q, want preempted", got.Status)
	}
}

func mustGet(t *testing.T, store *SQLiteStore, id string) CapacityClaim {
	t.Helper()
	c, err := store.GetClaim(context.Background(), id)
	if err != nil {
		t.Fatalf("GetClaim(%s) error = %v", id, err)
	}
	return c
}

func TestActuateRefusesTargetsChangedAfterPlanning(t *testing.T) {
	for _, mutation := range []string{"active", "protected", "terminal", "resize"} {
		t.Run(mutation, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c := idleLadderClaim(t, store, "holder", PriorityService)
			clk.Advance(2 * DefaultIdleGrace)
			plan := PlanEscalation(PriorityInteractive, 4*gib, []CapacityClaim{c}, DefaultPolicy(), clk.Now())
			if len(plan.Actions) != 1 {
				t.Fatal(plan)
			}
			var err error
			switch mutation {
			case "active":
				_, err = store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive)
			case "protected":
				_, err = store.db.ExecContext(ctx, "UPDATE capacity_claims SET protected=1 WHERE claim_id=?", c.ClaimID)
			case "terminal":
				_, err = store.ReleaseClaim(ctx, c.ClaimID)
			case "resize":
				_, err = store.ResizeClaim(ctx, c.ClaimID, c.Generation, 9*gib)
			}
			if err != nil {
				t.Fatal(err)
			}
			before := mustGet(t, store, c.ClaimID)
			exec := newFakeExecutor()
			result, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			after := mustGet(t, store, c.ClaimID)
			if len(exec.calls) != 0 || result.FreedBytes != 0 || after.Generation != before.Generation || after.AmountBytes != before.AmountBytes || after.Status != before.Status {
				t.Fatalf("stale plan acted: calls=%v result=%+v before=%+v after=%+v", exec.calls, result, before, after)
			}
		})
	}
}

func TestActuateHeartbeatDoesNotInvalidatePlan(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	c := idleLadderClaim(t, store, "holder", PriorityService)
	clk.Advance(2 * DefaultIdleGrace)
	plan := PlanEscalation(PriorityInteractive, 4*gib, []CapacityClaim{c}, DefaultPolicy(), clk.Now())
	if _, err := store.HeartbeatClaim(ctx, c.ClaimID, c.Generation, time.Hour); err != nil {
		t.Fatal(err)
	}
	exec := newFakeExecutor()
	if _, err := Actuate(ctx, plan, store, exec, DefaultPolicy(), clk.Now()); err != nil {
		t.Fatal(err)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("safe heartbeat invalidated plan: %v", exec.calls)
	}
}

func TestPreemptIntentDoesNotClaimPhysicalMemoryFreed(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	c := residentClaim("holder")
	c.DegradeProfile = nil
	c.Priority = PriorityBatch
	created, err := store.CreateClaim(ctx, c, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultIdleGrace)
	policy := DefaultPolicy()
	policy.PreemptEnabled = true
	plan := PlanEscalation(PriorityInteractive, gib, []CapacityClaim{created}, policy, clk.Now())
	exec := newFakeExecutor()
	result, err := Actuate(ctx, plan, store, exec, policy, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if result.FreedBytes != 0 || len(result.Outcomes) != 1 || result.Outcomes[0].FreedBytes != 0 || result.Outcomes[0].Applied || len(exec.calls) != 0 {
		t.Fatalf("ledger-only preempt falsely claimed physical reclaim: %+v calls=%v", result, exec.calls)
	}
}

type callbackExecutor func(context.Context, string, string, []string) error

func (f callbackExecutor) Apply(ctx context.Context, owner, verb string, args []string) error {
	return f(ctx, owner, verb, args)
}

func TestActuateDoesNotRewriteWorkActivatedDuringCallback(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		t.Run(fmt.Sprint(replacement), func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c := idleLadderClaim(t, store, "holder", PriorityService)
			clk.Advance(2 * DefaultIdleGrace)
			plan := PlanEscalation(PriorityInteractive, 4*gib, []CapacityClaim{c}, DefaultPolicy(), clk.Now())
			activeID := c.ClaimID
			entered := false
			executor := callbackExecutor(func(context.Context, string, string, []string) error {
				entered = true
				if !replacement {
					_, err := store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive)
					return err
				}
				if _, err := store.ReleaseClaim(ctx, c.ClaimID); err != nil {
					return err
				}
				next := c
				next.ClaimID = ""
				next.ActivityState = ActivityActive
				created, err := store.CreateClaim(ctx, next, time.Hour)
				activeID = created.ClaimID
				return err
			})
			result, err := Actuate(ctx, plan, store, executor, DefaultPolicy(), clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			got := mustGet(t, store, activeID)
			if !entered || got.ActivityState != ActivityActive || got.Status != StatusGranted || got.AmountBytes != 8*gib || result.FreedBytes != 0 || len(result.Outcomes) != 1 || result.Outcomes[0].Err == "" {
				t.Fatalf("callback window was hidden or active row was rewritten: result=%+v claim=%+v", result, got)
			}
		})
	}
}

func TestUpshiftAndPreemptRefuseWorkActivatedAfterPlanning(t *testing.T) {
	for _, kind := range []string{"upshift", "preempt"} {
		t.Run(kind, func(t *testing.T) {
			ctx := context.Background()
			clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
			store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
			c := residentClaim("holder")
			c.DegradeProfile = ladderProfile()
			c.AmountBytes = 8 * gib
			c.PreferredBytes = 8 * gib
			c.FloorBytes = 2 * gib
			c.Priority = PriorityBatch
			if kind == "preempt" {
				c.DegradeProfile = nil
			}
			created, err := store.CreateClaim(ctx, c, time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			policy := DefaultPolicy()
			policy.PreemptEnabled = true
			if kind == "upshift" {
				created, err = store.DegradeClaim(ctx, created.ClaimID, created.Generation, "small", 2*gib)
				if err != nil {
					t.Fatal(err)
				}
			}
			clk.Advance(2 * DefaultDegradeDebounce)
			var plan EscalationPlan
			if kind == "upshift" {
				plan = PlanUpshiftAll([]CapacityClaim{created}, snapshotWith(16, 2), policy, clk.Now())
			} else {
				plan = PlanEscalation(PriorityInteractive, gib, []CapacityClaim{created}, policy, clk.Now())
			}
			if len(plan.Actions) != 1 {
				t.Fatal(plan)
			}
			active, err := store.ReportActivity(ctx, created.ClaimID, created.Generation, ActivityActive)
			if err != nil {
				t.Fatal(err)
			}
			exec := newFakeExecutor()
			res, err := Actuate(ctx, plan, store, exec, policy, clk.Now())
			if err != nil {
				t.Fatal(err)
			}
			after := mustGet(t, store, created.ClaimID)
			if len(exec.calls) != 0 || after.Status != active.Status || after.Generation != active.Generation || res.FreedBytes != 0 || !res.Outcomes[0].Skipped {
				t.Fatalf("stale %s acted: %+v / %+v", kind, res, after)
			}
		})
	}
}

type activityOnReadStore struct {
	ClaimRepository
	beforeReturn func()
}

func (s activityOnReadStore) GetClaim(ctx context.Context, id string) (CapacityClaim, error) {
	c, err := s.ClaimRepository.GetClaim(ctx, id)
	if err == nil {
		s.beforeReturn()
	}
	return c, err
}

func TestPreemptChecksActivityAtomicallyAtLedgerMutation(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	c, err := store.CreateClaim(ctx, residentClaim("holder"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultIdleGrace)
	policy := DefaultPolicy()
	policy.PreemptEnabled = true
	plan := PlanEscalation(PriorityInteractive, gib, []CapacityClaim{c}, policy, clk.Now())
	wrapped := activityOnReadStore{ClaimRepository: store, beforeReturn: func() {
		if _, err := store.ReportActivity(ctx, c.ClaimID, c.Generation, ActivityActive); err != nil {
			t.Fatal(err)
		}
	}}
	result, err := Actuate(ctx, plan, wrapped, newFakeExecutor(), policy, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	after := mustGet(t, store, c.ClaimID)
	if after.Status != StatusGranted || after.ActivityState != ActivityActive || result.FreedBytes != 0 || len(result.Outcomes) != 1 || result.Outcomes[0].Err == "" {
		t.Fatalf("preempt raced activation: %+v / %+v", result, after)
	}
}

func TestDecideEnforceReclaimNeverSubstitutesOtherGPU(t *testing.T) {
	ctx := context.Background()
	clk := testenv.NewClock(time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC))
	store := testenv.NewSQLiteStore(t, "capacity.db", func(path string) (*SQLiteStore, error) { return NewSQLiteStore(ctx, Config{DBPath: path, Clock: clk}) })
	wanted := idleLadderClaim(t, store, "gpu-zero", PriorityService)
	foreign := residentClaim("gpu-one")
	foreign.GPUIndex = gpu(1)
	foreign.Priority = PriorityBatch
	foreign.AmountBytes = 8 * gib
	foreign.PreferredBytes = 8 * gib
	foreign.FloorBytes = 2 * gib
	foreign.DegradeProfile = ladderProfile()
	other, err := store.CreateClaim(ctx, foreign, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	clk.Advance(2 * DefaultIdleGrace)
	ledger := []CapacityClaim{other, wanted}
	req := CapacityRequest{OwnerID: "new", ResourceKind: ResourceKindVRAM, GPUIndex: gpu(0), PreferredBytes: 12 * gib, FloorBytes: 12 * gib, Priority: PriorityInteractive}
	verdict := Decide(req, snapshotWith(16, 8), ledger, DefaultPolicy(), clk.Now())
	if !verdict.Granted() || verdict.ReclaimBytes != 4*gib || len(verdict.ReclaimTargets) != 1 || verdict.ReclaimTargets[0] != wanted.ClaimID {
		t.Fatalf("unexpected admission cohort: %+v", verdict)
	}
	exec := newFakeExecutor()
	plan, result, err := EnforceReclaim(ctx, store, req.Priority, verdict, ledger, exec, DefaultPolicy(), EnforceOn, clk.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Satisfied || result.FreedBytes != 4*gib || exec.callsBy["gpu-zero"] != 1 || exec.callsBy["gpu-one"] != 0 {
		t.Fatalf("cross-GPU substitution: plan=%+v result=%+v calls=%v", plan, result, exec.calls)
	}
	if got := mustGet(t, store, other.ClaimID); got.Generation != other.Generation || got.AmountBytes != other.AmountBytes {
		t.Fatalf("foreign GPU mutated: %+v", got)
	}
}
