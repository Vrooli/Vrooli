package persistence

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
)

func operationsTestRepo(t *testing.T, name string) *Repository {
	t.Helper()
	db := openIdentityTestDB(t, name)
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-a", "demo-app", "production", "203.0.113.10", "demo.example")); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	return repo
}

func admit(t *testing.T, repo *Repository, id, key, digest string) *domain.CloudOperation {
	t.Helper()
	op, err := repo.AdmitOperation(context.Background(), &domain.CloudOperation{
		ID: id, DeploymentID: "dep-a", RequestKey: key, PlanDigest: digest,
		Plan: json.RawMessage(`{"schema_version":"1","actions":[]}`),
	})
	if err != nil {
		t.Fatalf("admit %s: %v", id, err)
	}
	return op
}

// TestAcquireWorkerBumpsFenceAndFencesEveryWrite [REQ:STC-P0-018] proves the
// ownership generation: acquisition bumps the deployment fence, a successor
// acquiring after lease expiry gets a higher fence, and the predecessor's
// heartbeat, step commit, unknown-effect record and state change are all
// refused with fence_stale (P07-A04, RUN-04).
func TestAcquireWorkerBumpsFenceAndFencesEveryWrite(t *testing.T) {
	repo := operationsTestRepo(t, "ops-fence")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")

	first, err := repo.AcquireWorker(ctx, "op-1", "worker-a", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if first != 1 {
		t.Fatalf("first fence = %d, want 1", first)
	}
	op, _ := repo.GetOperation(ctx, "op-1")
	if op.State != domain.OperationRunning || op.WorkerID != "worker-a" || op.LeaseExpiresAt == nil {
		t.Fatalf("after acquire: %+v", op)
	}
	dep, _ := repo.GetDeployment(ctx, "dep-a")
	if dep.Fence != 1 {
		t.Fatalf("deployment fence = %d, want 1", dep.Fence)
	}

	// A live lease held by another worker refuses acquisition.
	if _, err := repo.AcquireWorker(ctx, "op-1", "worker-b", time.Second); !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("live lease must refuse: %v", err)
	}
	time.Sleep(60 * time.Millisecond)
	second, err := repo.AcquireWorker(ctx, "op-1", "worker-b", time.Second)
	if err != nil {
		t.Fatalf("successor acquire: %v", err)
	}
	if second <= first {
		t.Fatalf("successor fence %d must exceed %d", second, first)
	}

	// The stale worker fails on every write.
	if err := repo.Heartbeat(ctx, "op-1", "worker-a", first, time.Second); !apierrors.Is(err, apierrors.CodeFenceStale) {
		t.Fatalf("stale heartbeat: %v", err)
	}
	if err := repo.CommitStep(ctx, "op-1", first, domain.StepReceipt{Step: "release.stage", Outcome: domain.StepSucceeded}); !apierrors.Is(err, apierrors.CodeFenceStale) {
		t.Fatalf("stale commit: %v", err)
	}
	if err := repo.RecordUnknownEffect(ctx, "op-1", first, domain.UnknownEffect{Step: "release.stage", Reason: "x"}); !apierrors.Is(err, apierrors.CodeFenceStale) {
		t.Fatalf("stale unknown effect: %v", err)
	}
	if err := repo.SetActiveStep(ctx, "op-1", first, &domain.ActiveStep{Step: "release.stage"}); !apierrors.Is(err, apierrors.CodeFenceStale) {
		t.Fatalf("stale active step: %v", err)
	}
	if err := repo.SetState(ctx, "op-1", first, domain.OperationRunning, domain.OperationSucceeded, StatePatch{}); !apierrors.Is(err, apierrors.CodeFenceStale) {
		t.Fatalf("stale set state: %v", err)
	}
	op, _ = repo.GetOperation(ctx, "op-1")
	if op.State != domain.OperationRunning || len(op.StepReceipts) != 0 || op.Fence != second {
		t.Fatalf("stale writes must leave the record untouched: %+v", op)
	}

	// The successor's writes land.
	if err := repo.CommitStep(ctx, "op-1", second, domain.StepReceipt{Step: "release.stage", Outcome: domain.StepSucceeded, Source: "worker"}); err != nil {
		t.Fatalf("successor commit: %v", err)
	}
	if err := repo.Heartbeat(ctx, "op-1", "worker-b", second, time.Second); err != nil {
		t.Fatalf("successor heartbeat: %v", err)
	}
	op, _ = repo.GetOperation(ctx, "op-1")
	receipts, _ := op.Receipts()
	if len(receipts) != 1 || receipts[0].Fence != second || receipts[0].Step != "release.stage" {
		t.Fatalf("receipts = %+v", receipts)
	}
}

// TestSetStateIsAtomicOnExpectedState [REQ:STC-P0-018] proves the state CAS:
// a write naming the wrong current state is refused and terminal states
// release the lease.
func TestSetStateIsAtomicOnExpectedState(t *testing.T) {
	repo := operationsTestRepo(t, "ops-setstate")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")
	fence, err := repo.AcquireWorker(ctx, "op-1", "w", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetState(ctx, "op-1", fence, domain.OperationAdmitted, domain.OperationRunning, StatePatch{}); !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("wrong expected state must be a conflict: %v", err)
	}
	result := &domain.OperationResult{Outcome: "failed", RecoveryOutcome: "service_restored", CompletedSteps: 2}
	if err := repo.SetState(ctx, "op-1", fence, domain.OperationRunning, domain.OperationFailed, StatePatch{Result: result, Error: apierrors.New(apierrors.CodeInternal, "boom")}); err != nil {
		t.Fatal(err)
	}
	op, _ := repo.GetOperation(ctx, "op-1")
	if op.State != domain.OperationFailed || op.TerminalAt == nil || op.WorkerID != "" || op.LeaseExpiresAt != nil {
		t.Fatalf("terminal bookkeeping: %+v", op)
	}
	if got := op.ResultValue(); got == nil || got.RecoveryOutcome != "service_restored" {
		t.Fatalf("result = %+v", got)
	}
	if _, err := repo.AcquireWorker(ctx, "op-1", "w2", time.Second); !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("terminal operation must refuse acquisition: %v", err)
	}
}

// TestCompetingOperationsSerialise [REQ:STC-P0-018] proves RUN-08: a second
// operation on the same deployment cannot acquire while the first holds a
// live lease, and can once the first is terminal.
func TestCompetingOperationsSerialise(t *testing.T) {
	repo := operationsTestRepo(t, "ops-serialise")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")
	admit(t, repo, "op-2", "k2", "sha256:b")
	fence, err := repo.AcquireWorker(ctx, "op-1", "w", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.AcquireWorker(ctx, "op-2", "w", time.Second)
	typed := apierrors.As(err)
	if typed == nil || typed.Code != apierrors.CodeOperationConflict || typed.Details["blocking_operation_id"] != "op-1" {
		t.Fatalf("competing acquire must name the blocker: %v", err)
	}
	if err := repo.SetState(ctx, "op-1", fence, domain.OperationRunning, domain.OperationFailed, StatePatch{}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.AcquireWorker(ctx, "op-2", "w", time.Second); err != nil {
		t.Fatalf("after the blocker is terminal: %v", err)
	}
}

// TestConcurrentAcquireAdmitsExactlyOneWorker [REQ:STC-P0-018] races many
// acquirers on one admitted operation: exactly one wins and the fence is
// monotonic. Run with -race on this package.
func TestConcurrentAcquireAdmitsExactlyOneWorker(t *testing.T) {
	repo := operationsTestRepo(t, "ops-race")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")
	const workers = 8
	var wg sync.WaitGroup
	wins := make(chan uint64, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			fence, err := repo.AcquireWorker(ctx, "op-1", "w"+string(rune('a'+i)), time.Second)
			if err == nil {
				wins <- fence
			}
		}(i)
	}
	wg.Wait()
	close(wins)
	count := 0
	for range wins {
		count++
	}
	if count != 1 {
		t.Fatalf("exactly one acquirer must win, got %d", count)
	}
	op, _ := repo.GetOperation(ctx, "op-1")
	if op.WorkerID == "" || op.State != domain.OperationRunning {
		t.Fatalf("winner not recorded: %+v", op)
	}
}

// TestRequestCancelSemantics [REQ:STC-P0-020] proves cancellation: an
// admitted operation is cancelled outright; a running one becomes
// cancel_requested and keeps its fence; a terminal one refuses.
func TestRequestCancelSemantics(t *testing.T) {
	repo := operationsTestRepo(t, "ops-cancel")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")
	op, err := repo.RequestCancel(ctx, "op-1")
	if err != nil || op.State != domain.OperationCancelled || !op.CancelRequested || op.TerminalAt == nil {
		t.Fatalf("admitted cancel: %+v %v", op, err)
	}
	if _, err := repo.RequestCancel(ctx, "op-1"); !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("terminal cancel must refuse: %v", err)
	}
	admit(t, repo, "op-2", "k2", "sha256:b")
	fence, err := repo.AcquireWorker(ctx, "op-2", "w", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	op, err = repo.RequestCancel(ctx, "op-2")
	if err != nil || op.State != domain.OperationCancelRequested || op.Fence != fence || op.WorkerID != "w" {
		t.Fatalf("running cancel: %+v %v", op, err)
	}
	// The worker still owns the fence and can commit the step in flight, then
	// honour the request at its boundary.
	if err := repo.CommitStep(ctx, "op-2", fence, domain.StepReceipt{Step: "release.stage", Outcome: domain.StepSucceeded}); err != nil {
		t.Fatalf("commit under cancel_requested: %v", err)
	}
	if err := repo.SetState(ctx, "op-2", fence, domain.OperationCancelRequested, domain.OperationCancelled, StatePatch{}); err != nil {
		t.Fatal(err)
	}
	ops, err := repo.ListNonTerminal(ctx)
	if err != nil || len(ops) != 0 {
		t.Fatalf("non-terminal = %d %v", len(ops), err)
	}
	all, _ := repo.ListOperationsByDeployment(ctx, "dep-a")
	if len(all) != 2 {
		t.Fatalf("list by deployment = %d", len(all))
	}
}

// TestUnknownEffectsAndActiveStepSurviveForReconciliation [REQ:STC-P0-019]
// proves the durable markers a successor reads: the active-step marker of a
// dead worker and its recorded unknown effect are visible to the next
// owner and the marker clears on commit.
func TestUnknownEffectsAndActiveStepSurviveForReconciliation(t *testing.T) {
	repo := operationsTestRepo(t, "ops-unknown")
	ctx := context.Background()
	admit(t, repo, "op-1", "k1", "sha256:a")
	fence, _ := repo.AcquireWorker(ctx, "op-1", "w", 20*time.Millisecond)
	if err := repo.SetActiveStep(ctx, "op-1", fence, &domain.ActiveStep{Step: "release.activate", Fence: fence, StartedAt: "t0"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.RecordUnknownEffect(ctx, "op-1", fence, domain.UnknownEffect{Step: "release.activate", Reason: "reply lost", Retry: "observe_then_replay"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(30 * time.Millisecond)
	next, err := repo.AcquireWorker(ctx, "op-1", "w2", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	op, _ := repo.GetOperation(ctx, "op-1")
	if m := op.ActiveStepMarker(); m == nil || m.Step != "release.activate" || m.Fence != fence {
		t.Fatalf("marker must survive reacquisition: %+v", m)
	}
	effects, _ := op.UnknownEffectList()
	if len(effects) != 1 || effects[0].Reason != "reply lost" {
		t.Fatalf("effects = %+v", effects)
	}
	if err := repo.CommitStep(ctx, "op-1", next, domain.StepReceipt{Step: "release.activate", Outcome: domain.StepSucceeded, Source: "target_receipt", Replayed: true}); err != nil {
		t.Fatal(err)
	}
	op, _ = repo.GetOperation(ctx, "op-1")
	if op.ActiveStepMarker() != nil {
		t.Fatalf("commit must clear the marker")
	}
	if r, ok := op.Receipt("release.activate"); !ok || r.Source != "target_receipt" || r.Fence != next {
		t.Fatalf("receipt = %+v", r)
	}
}

// TestAcquireWorkerBoundsEffectfulOperationsPerHost [REQ:STC-P0-028] proves
// the phase-23 admission bound: deployments sharing one target key may run
// at most effectful_operations_per_host_max live operations at once, a
// deployment on another target is unaffected, and an expired lease frees the
// slot.
func TestAcquireWorkerBoundsEffectfulOperationsPerHost(t *testing.T) {
	repo := operationsTestRepo(t, "ops-host-bound")
	ctx := context.Background()
	// dep-b shares dep-a's host (same target key); dep-c is elsewhere.
	if err := repo.CreateDeployment(ctx, newDeployment("dep-b", "other-app", "production", "203.0.113.10", "other.example")); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateDeployment(ctx, newDeployment("dep-c", "third-app", "production", "203.0.113.20", "third.example")); err != nil {
		t.Fatal(err)
	}
	repo.SetHostConcurrencyLimit(1)
	admitFor := func(id, dep string) {
		t.Helper()
		if _, err := repo.AdmitOperation(ctx, &domain.CloudOperation{ID: id, DeploymentID: dep, RequestKey: id, PlanDigest: "sha256:" + id, Plan: json.RawMessage(`{"schema_version":"1","actions":[]}`)}); err != nil {
			t.Fatalf("admit %s: %v", id, err)
		}
	}
	admitFor("op-a", "dep-a")
	admitFor("op-b", "dep-b")
	admitFor("op-c", "dep-c")

	if _, err := repo.AcquireWorker(ctx, "op-a", "worker-a", 80*time.Millisecond); err != nil {
		t.Fatalf("first acquisition on the host: %v", err)
	}
	_, err := repo.AcquireWorker(ctx, "op-b", "worker-b", time.Second)
	if !apierrors.Is(err, apierrors.CodeOperationConflict) {
		t.Fatalf("second live operation on the same host must be refused: %v", err)
	}
	if typed := apierrors.As(err); typed == nil || typed.Details["effectful_operations_per_host_max"] != "1" {
		t.Fatalf("refusal must name the bound: %v", err)
	}
	if _, err := repo.AcquireWorker(ctx, "op-c", "worker-c", time.Second); err != nil {
		t.Fatalf("another target is unaffected: %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if _, err := repo.AcquireWorker(ctx, "op-b", "worker-b", time.Second); err != nil {
		t.Fatalf("expired lease frees the host slot: %v", err)
	}
}

// TestPruneTerminalOperationsKeepsProtectedAndLiveRecords [REQ:STC-P0-028]
// proves the operation-receipt retention budget: only terminal records older
// than the cutoff are deleted, and incident-referenced ids and every
// non-terminal record survive.
func TestPruneTerminalOperationsKeepsProtectedAndLiveRecords(t *testing.T) {
	repo := operationsTestRepo(t, "ops-prune")
	ctx := context.Background()
	for _, id := range []string{"old-done", "old-incident", "fresh-done", "still-running"} {
		admit(t, repo, id, id, "sha256:"+id)
	}
	finish := func(id string, at time.Time) {
		t.Helper()
		fence, err := repo.AcquireWorker(ctx, id, "worker", time.Second)
		if err != nil {
			t.Fatalf("acquire %s: %v", id, err)
		}
		if err := repo.SetState(ctx, id, fence, domain.OperationRunning, domain.OperationSucceeded, StatePatch{}); err != nil {
			t.Fatalf("finish %s: %v", id, err)
		}
		if _, err := repo.db.ExecContext(ctx, `UPDATE cloud_operations SET terminal_at = $1 WHERE id = $2`, at, id); err != nil {
			t.Fatalf("age %s: %v", id, err)
		}
	}
	old := time.Now().UTC().Add(-120 * 24 * time.Hour)
	finish("old-done", old)
	finish("old-incident", old)
	finish("fresh-done", time.Now().UTC())
	if _, err := repo.AcquireWorker(ctx, "still-running", "worker", time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.ExecContext(ctx, `UPDATE cloud_operations SET created_at = $1, updated_at = $1 WHERE id = 'still-running'`, old); err != nil {
		t.Fatal(err)
	}

	deleted, err := repo.PruneTerminalOperations(ctx, time.Now().UTC().Add(-90*24*time.Hour), []string{"old-incident"})
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted = %d, want exactly the unprotected old record", deleted)
	}
	for _, id := range []string{"old-incident", "fresh-done", "still-running"} {
		if _, err := repo.GetOperation(ctx, id); err != nil {
			t.Fatalf("%s must survive pruning: %v", id, err)
		}
	}
	if op, err := repo.GetOperation(ctx, "old-done"); err == nil && op != nil {
		t.Fatal("old-done must be pruned")
	}
}
