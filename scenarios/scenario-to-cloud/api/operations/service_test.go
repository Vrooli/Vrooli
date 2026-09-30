package operations

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/persistence"
)

// ---- fixtures ---------------------------------------------------------------

func testRepo(t *testing.T, name string) *persistence.Repository {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+name+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	repo := persistence.NewRepository(db)
	if err := repo.InitSchemaOnDialect(context.Background(), db, "sqlite"); err != nil {
		t.Fatal(err)
	}
	m := domain.CloudManifest{Version: "1", Target: domain.ManifestTarget{Type: "vps", VPS: &domain.ManifestVPS{Host: "203.0.113.10", Port: 22, User: "root", Workdir: "/root/Vrooli"}}, Scenario: domain.ManifestScenario{ID: "app"}}
	raw, _ := json.Marshal(m)
	now := time.Now().UTC()
	if err := repo.CreateDeployment(context.Background(), &domain.Deployment{ID: "dep-1", Name: "app", ScenarioID: "app", Environment: "production", Status: domain.StatusPending, Manifest: raw, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	return repo
}

// testPlan is a three-action plan covering every retry contract.
func testPlan(schema string) *execplan.Plan {
	return &execplan.Plan{
		SchemaVersion: schema, DeploymentID: "dep-1", ScenarioID: "app", Environment: "production", Scope: execplan.ScopeFull, Outcome: execplan.OutcomeApply,
		Actions: []execplan.Action{
			{ID: "release.stage", OwnerOperation: "release.stage", Effect: "deployment_write", Retry: execplan.RetrySafeReplay, CancelPoint: true},
			{ID: "release.activate", OwnerOperation: "release.activate", Effect: "deployment_write", Retry: execplan.RetryRecover, Recovery: "rollback_to_predecessor", CancelPoint: false},
			{ID: "verify.readiness", OwnerOperation: "verify.readiness", Effect: "none", Retry: execplan.RetryObserveThenReplay, CancelPoint: true},
		},
	}
}

func admitPlan(t *testing.T, repo *persistence.Repository, id, key string, plan *execplan.Plan) *domain.CloudOperation {
	t.Helper()
	raw, _ := json.Marshal(plan)
	digest, _ := plan.SemanticDigest()
	op, err := repo.AdmitOperation(context.Background(), &domain.CloudOperation{ID: id, DeploymentID: "dep-1", RequestKey: key, PlanDigest: digest, Plan: raw})
	if err != nil {
		t.Fatalf("admit: %v", err)
	}
	return op
}

// fakeTarget is the target owner: effects are receipted per step, a lower
// fence than the highest seen is refused, and reads answer from receipts.
type fakeTarget struct {
	mu       sync.Mutex
	maxFence uint64
	receipts map[string]TargetReceipt
	absent   bool // no native CLI
	refused  int
}

func newFakeTarget() *fakeTarget { return &fakeTarget{receipts: map[string]TargetReceipt{}} }

func (f *fakeTarget) apply(fence uint64, step string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if fence < f.maxFence {
		f.refused++
		return apierrors.New(apierrors.CodeFenceStale, "target refuses stale fence").WithDetail("step", step)
	}
	f.maxFence = fence
	f.receipts[step] = TargetReceipt{Found: true, Outcome: StepSucceeded, Fence: fence, Detail: "applied"}
	return nil
}

func (f *fakeTarget) Read(_ context.Context, _ *domain.CloudOperation, step string) (TargetReceipt, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.absent {
		return TargetReceipt{}, ErrNativeCLIAbsent
	}
	r, ok := f.receipts[step]
	if !ok {
		return TargetReceipt{Found: false}, nil
	}
	return r, nil
}

// fakeRunner walks the plan through the StepSink exactly as the vps adapter
// does, with per-step gates and failures.
type fakeRunner struct {
	target   *fakeTarget
	mu       sync.Mutex
	executed map[string]int
	gates    map[string]chan struct{} // wait before executing the step
	fail     map[string]error
	dropAt   string // step whose reply is lost once
	dropped  int32
	attempts int32
}

func newFakeRunner(target *fakeTarget) *fakeRunner {
	return &fakeRunner{target: target, executed: map[string]int{}, gates: map[string]chan struct{}{}, fail: map[string]error{}}
}

func (r *fakeRunner) count(step string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.executed[step]
}

func (r *fakeRunner) Execute(ctx context.Context, ec *ExecutionContext) error {
	atomic.AddInt32(&r.attempts, 1)
	for _, action := range ec.Plan.Actions {
		decision, err := ec.Steps.Begin(ctx, action)
		if err != nil {
			return err
		}
		if decision == DecisionSkip {
			continue
		}
		for attempt := 0; attempt < 2; attempt++ {
			r.mu.Lock()
			gate := r.gates[action.ID]
			failure := r.fail[action.ID]
			r.mu.Unlock()
			if gate != nil {
				select {
				case <-gate:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
			r.mu.Lock()
			r.executed[action.ID]++
			r.mu.Unlock()
			var execErr error
			outcome := StepSucceeded
			if failure != nil {
				execErr, outcome = failure, StepFailed
			} else if err := r.target.apply(ec.Fence, action.ID); err != nil {
				execErr, outcome = err, StepFailed
			} else if r.dropAt == action.ID && atomic.CompareAndSwapInt32(&r.dropped, 0, 1) {
				execErr, outcome = &faultinject.Fault{Point: faultinject.TransportReply, Kind: faultinject.KindDropReply}, StepUnknown
			}
			err := ec.Steps.Commit(ctx, action, outcome, "fake", execErr)
			if errors.Is(err, ErrReplayStep) && attempt == 0 {
				continue
			}
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

type recordingProjection struct {
	mu       sync.Mutex
	started  int
	finished []State
}

func (p *recordingProjection) OperationStarted(context.Context, *domain.CloudOperation, *execplan.Plan) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.started++
}

func (p *recordingProjection) OperationFinished(_ context.Context, _ *domain.CloudOperation, _ *execplan.Plan, to State, _ *Result, _ *apierrors.Error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.finished = append(p.finished, to)
}

func fastConfig(worker string) Config {
	return Config{WorkerID: worker, Workers: 2, LeaseTTL: 150 * time.Millisecond, HeartbeatInterval: 40 * time.Millisecond, ReconcileInterval: time.Hour, ExecutionTimeout: 10 * time.Second, TransportTimeout: time.Second, ObserverTimeout: 5 * time.Second, QueueTimeout: time.Minute}
}

func newService(t *testing.T, repo *persistence.Repository, worker string, runner Runner, target TargetReceipts, proj Projection) *Service {
	t.Helper()
	svc := NewService(fastConfig(worker), repo, runner, target, proj)
	svc.Start()
	t.Cleanup(svc.Stop)
	return svc
}

func waitTerminal(t *testing.T, svc *Service, id string) *domain.CloudOperation {
	t.Helper()
	op, pending, err := svc.Wait(context.Background(), id, 5*time.Second)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if pending {
		t.Fatalf("operation %s still pending in state %s", id, op.State)
	}
	return op
}

func testCtx(reg *faultinject.Registry) context.Context {
	ctx := database.WithTestMode(context.Background())
	if reg != nil {
		ctx = faultinject.WithRegistry(ctx, reg)
	}
	return ctx
}

// ---- tests ------------------------------------------------------------------

// TestTransitionTableRefusesIllegalMoves [REQ:STC-P0-018] pins the state
// vocabulary and the transition table of design D.
func TestTransitionTableRefusesIllegalMoves(t *testing.T) {
	want := []State{"admitted", "waiting_input", "running", "verifying", "reconciling", "recovering", "cancel_requested", "succeeded", "failed", "failed_recovery", "cancelled"}
	if got := States(); len(got) != len(want) {
		t.Fatalf("states = %v", got)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("state %d = %s, want %s", i, got[i], want[i])
			}
		}
	}
	for _, tc := range []struct {
		from, to State
		ok       bool
	}{
		{Admitted, Running, true},
		{Admitted, Succeeded, false},
		{Running, Verifying, true},
		{Verifying, Succeeded, true},
		{Running, Succeeded, false},
		{Running, Reconciling, true},
		{Reconciling, Running, true},
		{Reconciling, Recovering, true},
		{Recovering, Failed, true},
		{Recovering, FailedRecovery, true},
		{Recovering, Succeeded, false},
		{Running, CancelRequested, true},
		{CancelRequested, Cancelled, true},
		{CancelRequested, Succeeded, false},
		{Succeeded, Running, false},
		{Failed, Running, false},
		{Cancelled, Admitted, false},
		{WaitingInput, Admitted, true},
		{WaitingInput, Running, false},
	} {
		err := Transition(tc.from, tc.to)
		if tc.ok && err != nil {
			t.Errorf("%s→%s should be legal: %v", tc.from, tc.to, err)
		}
		if !tc.ok && !apierrors.Is(err, apierrors.CodeOperationConflict) {
			t.Errorf("%s→%s should be refused, got %v", tc.from, tc.to, err)
		}
	}
	for _, terminal := range []State{Succeeded, Failed, FailedRecovery, Cancelled} {
		if len(legal[terminal]) != 0 {
			t.Errorf("terminal %s has outgoing edges", terminal)
		}
	}
}

// TestClientDisconnectKeepsServerOwnedWorkAttachable [REQ:STC-P0-020] proves
// P07-A01 / RUN-07: the submitting context is cancelled immediately, the
// work still runs to completion under the owner, and a later Wait by
// operation id observes the terminal outcome with every receipt.
func TestClientDisconnectKeepsServerOwnedWorkAttachable(t *testing.T) {
	repo := testRepo(t, "svc-disconnect")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	gate := make(chan struct{})
	runner.gates["release.stage"] = gate
	proj := &recordingProjection{}
	svc := newService(t, repo, "owner-a", runner, target, proj)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))

	clientCtx, disconnect := context.WithCancel(context.Background())
	svc.Submit(clientCtx, op.ID, ExecuteOptions{})
	disconnect()
	close(gate)

	got := waitTerminal(t, svc, op.ID)
	if got.State != Succeeded {
		t.Fatalf("state = %s (%s)", got.State, got.Error)
	}
	standing := StandingOf(got)
	if len(standing.CompletedSteps) != 3 || standing.Fence == 0 || standing.WorkerID != "" || standing.Terminal != true {
		t.Fatalf("standing = %+v", standing)
	}
	if standing.ReattachCommand != "scenario-to-cloud operation wait op-1" {
		t.Fatalf("reattach = %q", standing.ReattachCommand)
	}
	if proj.started != 1 || len(proj.finished) != 1 || proj.finished[0] != Succeeded {
		t.Fatalf("projection = %+v", proj)
	}
	// A second Wait on a terminal record returns immediately.
	again, pending, err := svc.Wait(context.Background(), op.ID, time.Millisecond)
	if err != nil || pending || again.State != Succeeded {
		t.Fatalf("re-wait: %v %v %s", err, pending, again.State)
	}
}

// TestEqualReplayReturnsSameOperationAndConflictOnDifferentInput
// [REQ:STC-P0-018] proves P07-A05 / RUN-01 and RUN-02: the same request key
// with the same digest yields one operation and one execution; a different
// digest under the same key is refused and the original intent is intact.
func TestEqualReplayReturnsSameOperationAndConflictOnDifferentInput(t *testing.T) {
	repo := testRepo(t, "svc-replay")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	svc := newService(t, repo, "owner-a", runner, target, nil)
	plan := testPlan(execplan.SchemaVersion)
	first := admitPlan(t, repo, "op-1", "k1", plan)
	svc.Submit(context.Background(), first.ID, ExecuteOptions{})
	replay := admitPlan(t, repo, "op-2", "k1", plan)
	if replay.ID != first.ID {
		t.Fatalf("replay must return the same operation, got %s", replay.ID)
	}
	svc.Submit(context.Background(), replay.ID, ExecuteOptions{})
	got := waitTerminal(t, svc, first.ID)
	if got.State != Succeeded || runner.count("release.activate") != 1 {
		t.Fatalf("state=%s activate executions=%d", got.State, runner.count("release.activate"))
	}
	different := testPlan(execplan.SchemaVersion)
	different.Actions = different.Actions[:1]
	raw, _ := json.Marshal(different)
	digest, _ := different.SemanticDigest()
	_, err := repo.AdmitOperation(context.Background(), &domain.CloudOperation{ID: "op-3", DeploymentID: "dep-1", RequestKey: "k1", PlanDigest: digest, Plan: raw})
	if !apierrors.Is(err, apierrors.CodeRequestKeyConflict) {
		t.Fatalf("different input under the same key must conflict: %v", err)
	}
	intact, _ := repo.GetOperation(context.Background(), first.ID)
	if intact.PlanDigest != first.PlanDigest || intact.State != Succeeded {
		t.Fatalf("original intent changed: %+v", intact)
	}
}

// TestOwnerRestartAfterAdmissionAndDuringEachMutatingStep [REQ:STC-P0-019]
// proves P07-A02 / RUN-03: the owner dies (fault crash at worker_before_commit)
// after every mutating step in turn; a fresh Service over the same database
// reacquires with a higher fence once the lease lapses, skips committed
// steps, reads the target receipt for the in-flight step instead of
// replaying it, and reaches a truthful terminal state. Restart before any
// worker attempt (right after admission) is the first case.
func TestOwnerRestartAfterAdmissionAndDuringEachMutatingStep(t *testing.T) {
	for _, crashAt := range []string{"", "release.stage", "release.activate", "verify.readiness"} {
		name := crashAt
		if name == "" {
			name = "after-admission"
		}
		t.Run(name, func(t *testing.T) {
			repo := testRepo(t, "svc-restart-"+name)
			target := newFakeTarget()
			reg := faultinject.New()
			crashed := make(chan *faultinject.Fault, 1)
			reg.SetCrashFunc(func(f *faultinject.Fault) {
				select {
				case crashed <- f:
				default:
				}
				panic(f)
			})
			op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
			ctx := testCtx(reg)

			runnerA := newFakeRunner(target)
			if crashAt != "" {
				// Crash exactly once, at the commit of crashAt: the effect happened
				// on the target, the receipt was never committed by owner A.
				if err := reg.Arm(ctx, faultinject.WorkerBeforeCommit, faultinject.Behaviour{Kind: faultinject.KindCrash, Once: true}); err != nil {
					t.Fatal(err)
				}
				// Arm at the right step by gating: the fault fires on the first commit.
				ownerA := NewService(fastConfig("owner-a"), repo, RunnerFunc(func(ctx context.Context, ec *ExecutionContext) error {
					// Walk to crashAt with faults disarmed, then re-arm.
					reg.Disarm(faultinject.WorkerBeforeCommit)
					for _, action := range ec.Plan.Actions {
						if action.ID == crashAt {
							_ = reg.Arm(ctx, faultinject.WorkerBeforeCommit, faultinject.Behaviour{Kind: faultinject.KindCrash, Once: true})
						}
						sub := &execplan.Plan{SchemaVersion: ec.Plan.SchemaVersion, Actions: []execplan.Action{action}}
						if err := runnerA.Execute(ctx, &ExecutionContext{Operation: ec.Operation, Plan: sub, Fence: ec.Fence, Steps: ec.Steps}); err != nil {
							return err
						}
					}
					return nil
				}), target, nil)
				ownerA.Start()
				ownerA.Submit(ctx, op.ID, ExecuteOptions{})
				select {
				case f := <-crashed:
					if f.Point != faultinject.WorkerBeforeCommit {
						t.Fatalf("crash at %s", f.Point)
					}
				case <-time.After(5 * time.Second):
					t.Fatal("owner A never crashed")
				}
				ownerA.Stop()
				mid, _ := repo.GetOperation(context.Background(), op.ID)
				if mid.State != Running || mid.WorkerID != "owner-a" || mid.ActiveStepMarker() == nil || mid.ActiveStepMarker().Step != crashAt {
					t.Fatalf("after crash the record must stay running with the in-flight marker: %+v", mid)
				}
				if _, ok := mid.Receipt(crashAt); ok {
					t.Fatalf("crashed step must have no receipt yet")
				}
			}

			// Owner B starts over the same database. Startup reconciliation must
			// wait for the lease, then take over with a new fence.
			runnerB := newFakeRunner(target)
			ownerB := newService(t, repo, "owner-b", runnerB, target, nil)
			deadline := time.Now().Add(3 * time.Second)
			var taken []string
			for time.Now().Before(deadline) {
				taken, _ = ownerB.Reconcile(context.Background())
				if len(taken) == 1 {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if len(taken) != 1 || taken[0] != op.ID {
				t.Fatalf("owner B must reacquire after lease expiry, taken=%v", taken)
			}
			got := waitTerminal(t, ownerB, op.ID)
			if got.State != Succeeded {
				t.Fatalf("state = %s error=%s", got.State, got.Error)
			}
			receipts, _ := got.Receipts()
			byStep := map[string]domain.StepReceipt{}
			for _, r := range receipts {
				byStep[r.Step] = r
			}
			for _, step := range []string{"release.stage", "release.activate", "verify.readiness"} {
				if byStep[step].Outcome != StepSucceeded {
					t.Fatalf("step %s receipt = %+v", step, byStep[step])
				}
			}
			if crashAt != "" {
				// The in-flight step's outcome came from the target receipt for
				// observe/recover contracts; safe_replay replays. Either way the
				// destructive effect happened at most as often as its contract allows.
				total := runnerA.count(crashAt) + runnerB.count(crashAt)
				switch crashAt {
				case "release.stage":
					if total != 2 || byStep[crashAt].Source != "worker" {
						t.Fatalf("safe_replay step executed %d times, receipt %+v", total, byStep[crashAt])
					}
				default:
					if total != 1 || byStep[crashAt].Source != "target_receipt" || !byStep[crashAt].Replayed {
						t.Fatalf("%s executed %d times, receipt %+v (must be proven by the target receipt, never replayed)", crashAt, total, byStep[crashAt])
					}
				}
				if mid, _ := repo.GetOperation(context.Background(), op.ID); mid.Fence < 2 {
					t.Fatalf("successor fence must exceed the predecessor's: %d", mid.Fence)
				}
			}
		})
	}
}

// TestLostReplyReadsReceiptBeforeReplaying [REQ:STC-P0-019] proves P07-A03 /
// RUN-05: the target applied release.activate but the reply was dropped;
// the owner reads the target receipt, commits it as proven and continues
// without re-running the destructive effect.
func TestLostReplyReadsReceiptBeforeReplaying(t *testing.T) {
	repo := testRepo(t, "svc-lost-reply")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	runner.dropAt = "release.activate"
	svc := newService(t, repo, "owner-a", runner, target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	got := waitTerminal(t, svc, op.ID)
	if got.State != Succeeded {
		t.Fatalf("state = %s (%s)", got.State, got.Error)
	}
	if runner.count("release.activate") != 1 {
		t.Fatalf("destructive step replayed %d times", runner.count("release.activate"))
	}
	r, _ := got.Receipt("release.activate")
	if r == nil || r.Source != "target_receipt" || !r.Replayed || r.Outcome != StepSucceeded {
		t.Fatalf("receipt = %+v", r)
	}
	effects, _ := got.UnknownEffectList()
	if len(effects) != 0 {
		t.Fatalf("resolved lost reply must not leave an unknown effect: %+v", effects)
	}
}

// TestLostReplyWithoutReceiptOwnerRecordsUnknownEffect [REQ:STC-P0-019]
// proves RUN-06: when the target cannot answer (no native CLI), a lost reply
// is recorded as an unknown effect with one next action and the operation
// moves to reconciling — never to failed or succeeded by inference.
func TestLostReplyWithoutReceiptOwnerRecordsUnknownEffect(t *testing.T) {
	repo := testRepo(t, "svc-unknown")
	target := newFakeTarget()
	target.absent = true
	runner := newFakeRunner(target)
	runner.dropAt = "release.activate"
	svc := newService(t, repo, "owner-a", runner, target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	deadline := time.Now().Add(3 * time.Second)
	var got *domain.CloudOperation
	for time.Now().Before(deadline) {
		got, _ = repo.GetOperation(context.Background(), op.ID)
		if got.State == Reconciling {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got == nil || got.State != Reconciling {
		t.Fatalf("state = %v", got)
	}
	standing := StandingOf(got)
	if len(standing.UnknownEffects) != 1 || standing.UnknownEffects[0].Step != "release.activate" || standing.NextAction == nil || standing.NextAction.Kind != "reconcile" {
		t.Fatalf("standing = %+v", standing)
	}
	if runner.count("release.activate") != 1 {
		t.Fatalf("replayed a recover-class step blind: %d", runner.count("release.activate"))
	}
	// The lease is released so a later owner can reconcile once reach returns.
	if got.WorkerID != "" {
		t.Fatalf("reconciling must release the lease: %+v", got)
	}
	target.mu.Lock()
	target.absent = false
	target.mu.Unlock()
	taken, err := svc.Reconcile(context.Background())
	if err != nil || len(taken) != 1 {
		t.Fatalf("reconcile: %v %v", taken, err)
	}
	final := waitTerminal(t, svc, op.ID)
	if final.State != Succeeded || runner.count("release.activate") != 1 {
		t.Fatalf("after reach returns: %s executions=%d", final.State, runner.count("release.activate"))
	}
}

// TestStaleWorkerIsRefusedOnCommitAndTargetMutation [REQ:STC-P0-018] proves
// P07-A04 / RUN-04: owner A stalls (heartbeats paused) mid-plan, owner B
// takes over with a higher fence and finishes; when A wakes, its commit is
// refused with fence_stale, the target refuses its lower fence, and A writes
// nothing terminal.
func TestStaleWorkerIsRefusedOnCommitAndTargetMutation(t *testing.T) {
	repo := testRepo(t, "svc-stale")
	target := newFakeTarget()
	runnerA := newFakeRunner(target)
	stall := make(chan struct{})
	runnerA.gates["release.activate"] = stall
	ownerA := newService(t, repo, "owner-a", runnerA, target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	ownerA.Submit(context.Background(), op.ID, ExecuteOptions{})
	// Wait until A committed release.stage and is stalled before activate.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cur, _ := repo.GetOperation(context.Background(), op.ID); cur != nil {
			if _, ok := cur.Receipt("release.stage"); ok {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	ownerA.PauseHeartbeats(true)
	first, _ := repo.GetOperation(context.Background(), op.ID)

	runnerB := newFakeRunner(target)
	ownerB := newService(t, repo, "owner-b", runnerB, target, nil)
	var taken []string
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		taken, _ = ownerB.Reconcile(context.Background())
		if len(taken) == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(taken) != 1 {
		t.Fatalf("owner B must take over the expired lease")
	}
	final := waitTerminal(t, ownerB, op.ID)
	if final.State != Succeeded || final.Fence <= first.Fence {
		t.Fatalf("B outcome = %s fence %d (A had %d)", final.State, final.Fence, first.Fence)
	}
	// A wakes up: its target mutation carries the old fence and is refused;
	// its commit is refused with fence_stale; the record is untouched.
	close(stall)
	time.Sleep(200 * time.Millisecond)
	after, _ := repo.GetOperation(context.Background(), op.ID)
	if after.State != Succeeded || after.Fence != final.Fence || after.UpdatedAt.After(final.UpdatedAt.Add(time.Millisecond)) {
		t.Fatalf("stale worker changed the record: %+v", after)
	}
	target.mu.Lock()
	refused := target.refused
	target.mu.Unlock()
	if refused != 1 {
		t.Fatalf("target must refuse the stale fence exactly once, got %d", refused)
	}
	receipts, _ := after.Receipts()
	for _, r := range receipts {
		if r.Step == "release.activate" && r.Fence != final.Fence {
			t.Fatalf("activate receipt carries fence %d, want %d", r.Fence, final.Fence)
		}
	}
	if runnerA.count("release.activate") != 1 {
		t.Fatalf("A must have attempted activate exactly once")
	}
}

// TestCancellationStopsAtDeclaredSafeBoundary [REQ:STC-P0-020] proves
// P07-A06: a cancel requested while release.stage executes is honoured only
// at the next cancel point — release.activate (cancel_point=false) still
// runs to completion, verify.readiness (cancel_point=true) never starts.
func TestCancellationStopsAtDeclaredSafeBoundary(t *testing.T) {
	repo := testRepo(t, "svc-cancel")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	gate := make(chan struct{})
	runner.gates["release.stage"] = gate
	proj := &recordingProjection{}
	svc := newService(t, repo, "owner-a", runner, target, proj)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cur, _ := repo.GetOperation(context.Background(), op.ID); cur != nil && cur.State == Running && cur.ActiveStepMarker() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancelled, err := svc.Cancel(context.Background(), op.ID)
	if err != nil || cancelled.State != CancelRequested {
		t.Fatalf("cancel: %+v %v", cancelled, err)
	}
	close(gate)
	got := waitTerminal(t, svc, op.ID)
	if got.State != Cancelled {
		t.Fatalf("state = %s", got.State)
	}
	if runner.count("release.stage") != 1 || runner.count("release.activate") != 1 || runner.count("verify.readiness") != 0 {
		t.Fatalf("executions stage=%d activate=%d verify=%d", runner.count("release.stage"), runner.count("release.activate"), runner.count("verify.readiness"))
	}
	standing := StandingOf(got)
	if len(standing.CompletedSteps) != 2 || standing.Result == nil || standing.Result.Outcome != "cancelled" {
		t.Fatalf("standing = %+v", standing)
	}
	// Cancelling a terminal operation is refused; cancelling an admitted one
	// (no worker) cancels outright.
	if _, err := svc.Cancel(context.Background(), op.ID); !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("terminal cancel: %v", err)
	}
	idle := admitPlan(t, repo, "op-2", "k2", testPlan(execplan.SchemaVersion))
	if out, err := svc.Cancel(context.Background(), idle.ID); err != nil || out.State != Cancelled {
		t.Fatalf("admitted cancel: %+v %v", out, err)
	}
}

// TestFailedRecoveryCannotReportSuccess [REQ:STC-P0-018] proves P07-A07: a
// hard failure of a recover-class step moves through recovering to
// failed_recovery with recovery_outcome recovery_failed; the projection
// never reports a deployed service; a failed change keeps its recovery
// outcome distinct from operation success.
func TestFailedRecoveryCannotReportSuccess(t *testing.T) {
	repo := testRepo(t, "svc-failed-recovery")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	runner.fail["release.activate"] = errors.New("activation exploded")
	proj := &recordingProjection{}
	svc := newService(t, repo, "owner-a", runner, target, proj)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	got := waitTerminal(t, svc, op.ID)
	if got.State != FailedRecovery {
		t.Fatalf("state = %s", got.State)
	}
	result := got.ResultValue()
	if result == nil || result.Outcome != string(FailedRecovery) || result.RecoveryOutcome != RecoveryFailed {
		t.Fatalf("result = %+v", result)
	}
	if len(proj.finished) != 1 || proj.finished[0] != FailedRecovery {
		t.Fatalf("projection = %+v", proj.finished)
	}
	if r, _ := got.Receipt("release.activate"); r == nil || r.Outcome != StepFailed || r.Error == "" {
		t.Fatalf("failed receipt = %+v", r)
	}
	// A plain failure (non-recover step) is failed with not_attempted.
	runner2 := newFakeRunner(newFakeTarget())
	runner2.fail["release.stage"] = errors.New("stage exploded")
	svc2 := newService(t, repo, "owner-b", runner2, target, nil)
	op2 := admitPlan(t, repo, "op-2", "k2", testPlan(execplan.SchemaVersion))
	svc2.Submit(context.Background(), op2.ID, ExecuteOptions{})
	got2 := waitTerminal(t, svc2, op2.ID)
	if got2.State != Failed || got2.ResultValue().RecoveryOutcome != RecoveryNotAttempted {
		t.Fatalf("plain failure = %s %+v", got2.State, got2.ResultValue())
	}
	if StandingOf(got2).NextAction == nil {
		t.Fatalf("failed standing must carry a next action")
	}
}

// TestCompetingOperationsSerialiseThroughTheOwner [REQ:STC-P0-018] proves
// RUN-08 end to end: two operations on one deployment submitted together
// run one after the other; the target sees strictly increasing fences.
func TestCompetingOperationsSerialiseThroughTheOwner(t *testing.T) {
	repo := testRepo(t, "svc-compete")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	gate := make(chan struct{})
	runner.gates["release.activate"] = gate
	svc := newService(t, repo, "owner-a", runner, target, nil)
	a := admitPlan(t, repo, "op-a", "ka", testPlan(execplan.SchemaVersion))
	b := admitPlan(t, repo, "op-b", "kb", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), a.ID, ExecuteOptions{})
	time.Sleep(50 * time.Millisecond)
	svc.Submit(context.Background(), b.ID, ExecuteOptions{})
	time.Sleep(100 * time.Millisecond)
	states := map[string]State{}
	for _, id := range []string{a.ID, b.ID} {
		cur, _ := repo.GetOperation(context.Background(), id)
		states[id] = cur.State
	}
	running := 0
	for _, st := range states {
		if st == Running {
			running++
		}
	}
	if running != 1 {
		t.Fatalf("exactly one operation may run at a time: %v", states)
	}
	close(gate)
	first := waitTerminal(t, svc, a.ID)
	// The second was refused acquisition while the first ran; the reconciler
	// picks it up afterwards.
	deadline := time.Now().Add(3 * time.Second)
	var second *domain.CloudOperation
	for time.Now().Before(deadline) {
		_, _ = svc.Reconcile(context.Background())
		second, _ = repo.GetOperation(context.Background(), b.ID)
		if second.State.IsTerminal() {
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if first.State != Succeeded || second == nil || second.State != Succeeded || second.Fence <= first.Fence {
		t.Fatalf("first=%s(%d) second=%v", first.State, first.Fence, second)
	}
}

// TestObserverTimeoutNeverMutates [REQ:STC-P0-020] proves the wait contract:
// a short observer bound returns still_pending with the live state and
// changes nothing; the operation finishes on its own afterwards.
func TestObserverTimeoutNeverMutates(t *testing.T) {
	repo := testRepo(t, "svc-observer")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	gate := make(chan struct{})
	runner.gates["release.stage"] = gate
	svc := newService(t, repo, "owner-a", runner, target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	deadline := time.Now().Add(3 * time.Second)
	var before *domain.CloudOperation
	for time.Now().Before(deadline) {
		before, _ = repo.GetOperation(context.Background(), op.ID)
		if before.State == Running && before.ActiveStepMarker() != nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	got, pending, err := svc.Wait(context.Background(), op.ID, 30*time.Millisecond)
	if err != nil || !pending || got.State.IsTerminal() {
		t.Fatalf("wait: %v pending=%v state=%s", err, pending, got.State)
	}
	standing := StandingOf(got).Pending(svc.Config().LeaseTTL)
	if !standing.StillPending || standing.RecommendedNextCheckSeconds < 1 {
		t.Fatalf("pending standing = %+v", standing)
	}
	after, _ := repo.GetOperation(context.Background(), op.ID)
	if after.State != before.State || after.Fence != before.Fence {
		t.Fatalf("observer changed the record: %+v -> %+v", before, after)
	}
	close(gate)
	if final := waitTerminal(t, svc, op.ID); final.State != Succeeded {
		t.Fatalf("final = %s", final.State)
	}
}

// TestUnknownPlanSchemaIsExplicitIncompatibility [REQ:STC-P0-019] proves
// RUN-10: a stored operation with an unsupported plan schema fails with
// unsupported_schema_version; nothing is guessed.
func TestUnknownPlanSchemaIsExplicitIncompatibility(t *testing.T) {
	repo := testRepo(t, "svc-schema")
	target := newFakeTarget()
	runner := newFakeRunner(target)
	svc := newService(t, repo, "owner-a", runner, target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan("99"))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	got := waitTerminal(t, svc, op.ID)
	if got.State != Failed || runner.count("release.stage") != 0 {
		t.Fatalf("state=%s executions=%d", got.State, runner.count("release.stage"))
	}
	if st := StandingOf(got); st.Error == nil || st.Error.Code != apierrors.CodeUnsupportedSchemaVersion {
		t.Fatalf("error = %+v", st.Error)
	}
}

// TestWaitingInputIsDistinctFromDeadWorker [REQ:STC-P0-020] proves the
// waiting_input state: a runner that needs non-durable input parks the
// operation with a handoff, releases the lease and is not reacquired by
// reconciliation until the input arrives.
func TestWaitingInputIsDistinctFromDeadWorker(t *testing.T) {
	repo := testRepo(t, "svc-waiting")
	target := newFakeTarget()
	svc := newService(t, repo, "owner-a", RunnerFunc(func(context.Context, *ExecutionContext) error {
		return WaitingInputError(apierrors.New(apierrors.CodeNeedsInput, "operator secret required").WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "resume", Reference: "/resume"}))
	}), target, nil)
	op := admitPlan(t, repo, "op-1", "k1", testPlan(execplan.SchemaVersion))
	svc.Submit(context.Background(), op.ID, ExecuteOptions{})
	deadline := time.Now().Add(3 * time.Second)
	var got *domain.CloudOperation
	for time.Now().Before(deadline) {
		got, _ = repo.GetOperation(context.Background(), op.ID)
		if got.State == WaitingInput {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if got == nil || got.State != WaitingInput || got.WorkerID != "" {
		t.Fatalf("got = %+v", got)
	}
	if taken, _ := svc.Reconcile(context.Background()); len(taken) != 0 {
		t.Fatalf("waiting_input must not be reacquired: %v", taken)
	}
	if st := StandingOf(got); st.NextAction == nil || st.NextAction.Kind != "resume" {
		t.Fatalf("standing = %+v", st)
	}
}

func TestStandingDefaultNextAction(t *testing.T) {
	if got := fmt.Sprint(StandingOf(&domain.CloudOperation{ID: "x", State: Admitted}).NextAction.Kind); got != "wait" {
		t.Fatalf("admitted next action = %s", got)
	}
}
