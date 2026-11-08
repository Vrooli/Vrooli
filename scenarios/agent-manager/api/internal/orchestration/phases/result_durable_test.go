package phases

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

// cancelRaceRunRepo is a minimal in-memory RunRepository for the delayed-exit
// reconciliation test. It keeps the durability seam explicit without pulling
// the full database schema into the phases test binary.
type cancelRaceRunRepo struct {
	repository.RunRepository
	run *domain.Run
}

type failingRunReadRepository struct {
	repository.RunRepository
	err error
}

type failingRunUpdateRepository struct {
	repository.RunRepository
	err error
}

func (r failingRunReadRepository) Get(context.Context, uuid.UUID) (*domain.Run, error) {
	return nil, r.err
}

func (r failingRunUpdateRepository) Update(context.Context, *domain.Run) error {
	return r.err
}

func (r *cancelRaceRunRepo) Get(_ context.Context, id uuid.UUID) (*domain.Run, error) {
	if r.run == nil || r.run.ID != id {
		return nil, domain.NewNotFoundError("Run", id)
	}
	clone := *r.run
	return &clone, nil
}

func (r *cancelRaceRunRepo) Update(_ context.Context, run *domain.Run) error {
	clone := *run
	r.run = &clone
	return nil
}

func (r *cancelRaceRunRepo) RequestCancellation(_ context.Context, id uuid.UUID, at time.Time) (bool, error) {
	if r.run == nil || r.run.ID != id {
		return false, domain.NewNotFoundError("Run", id)
	}
	if r.run.CancelRequestedAt != nil || (r.run.Status != domain.RunStatusPending && r.run.Status != domain.RunStatusStarting && r.run.Status != domain.RunStatusRunning) {
		return false, nil
	}
	clone := *r.run
	clone.CancelRequestedAt = &at
	clone.LifecycleVersion++
	r.run = &clone
	return true, nil
}

// TestHandleResultDoesNotResurrectDurablyCancelledRun guards Phase 3 of the
// delegation foundation: a delayed process exit must reconcile against durable
// owner state exactly once and must not overwrite an accepted cancellation.
//
// Scenario: an owner stop request persists RunStatusCancelled while the runner
// process is still alive. That process then exits successfully. The executor's
// in-memory snapshot is stale (still running), so HandleResult must adopt the
// durable terminal state instead of resurrecting the run to complete.
func TestHandleResultDoesNotResurrectDurablyCancelledRun(t *testing.T) {
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(),
		RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning,
		Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig(),
	}
	repo := &cancelRaceRunRepo{run: run}

	// Simulate the accepted stop request: durable status is cancelled while the
	// executor's snapshot is still running.
	cancelled, err := repo.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	cancelled.Status = domain.RunStatusCancelled
	cancelled.Phase = domain.RunPhaseCompleted
	if err := repo.Update(context.Background(), cancelled); err != nil {
		t.Fatal(err)
	}

	out := HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{Runs: repo},
		Run:  run,
		Result: &runner.ExecuteResult{
			Success: true, ExitCode: 0,
			Result: &domain.RunResult{
				Success:   true,
				Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected},
			},
		},
	})

	if run.Status != domain.RunStatusCancelled {
		t.Fatalf("delayed exit resurrected in-memory run: status=%s", run.Status)
	}
	if repo.run.Status != domain.RunStatusCancelled {
		t.Fatalf("delayed exit resurrected durable run: status=%s", repo.run.Status)
	}
	if out.Outcome != domain.RunOutcomeCancelled {
		t.Fatalf("outcome=%s, want cancelled", out.Outcome)
	}
}

// TestHandleResultReconcilesDurableCancellationIntent guards the Phase 3
// cancel-before-terminate rule: an owner stop request stamps
// Run.CancelRequestedAt while the runner process is still alive. If that
// process then exits successfully, the persisted intent — not the successful
// exit — decides the terminal state. HandleResult must reconcile the delayed
// exit to cancelled and retain the intent on the durable record.
func TestHandleResultReconcilesDurableCancellationIntent(t *testing.T) {
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(),
		RunMode: domain.RunModeInPlace, Status: domain.RunStatusRunning,
		Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig(), OwnerIdentity: "owner-a", OwnerEpoch: 4,
	}
	repo := &cancelRaceRunRepo{run: run}

	// Simulate the exact atomic RequestCancellation write while the process
	// remains active; it advances one lifecycle version and stamps the intent.
	stamp := time.Now().Add(-time.Second)
	if changed, err := repo.RequestCancellation(context.Background(), run.ID, stamp); err != nil || !changed {
		t.Fatalf("request cancellation: changed=%v err=%v", changed, err)
	}

	out := HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{Runs: repo},
		Run:  run,
		Result: &runner.ExecuteResult{
			Success: true, ExitCode: 0,
			Result: &domain.RunResult{
				Success:   true,
				Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected},
			},
		},
	})

	if out.Outcome != domain.RunOutcomeCancelled {
		t.Fatalf("outcome=%s, want cancelled", out.Outcome)
	}
	if run.Status != domain.RunStatusCancelled {
		t.Fatalf("delayed exit ignored cancellation intent: in-memory status=%s", run.Status)
	}
	if repo.run.Status != domain.RunStatusCancelled {
		t.Fatalf("delayed exit ignored cancellation intent: durable status=%s", repo.run.Status)
	}
	if repo.run.CancelRequestedAt == nil {
		t.Fatal("cancellation intent was not retained on the durable run")
	}
}

func TestHandleResultPublishesOnlyPersistedTerminalStatus(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeInPlace,
		Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ResolvedConfig: domain.DefaultRunConfig(),
	}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	events := mocks.NewFakeEventStore()
	HandleResult(t.Context(), HandleResultInput{
		Deps: Deps{Runs: repos.Runs, Events: events},
		Run:  run,
		Result: &runner.ExecuteResult{
			Success: true,
			Result:  &domain.RunResult{Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected}},
		},
	})

	statusEvents := events.TypedEvents(run.ID, domain.EventTypeStatus)
	if len(statusEvents) != 1 {
		t.Fatalf("status event count = %d, want exactly one persisted terminal transition", len(statusEvents))
	}
	status, ok := statusEvents[0].Data.(*domain.StatusEventData)
	if !ok || status.OldStatus != string(domain.RunStatusRunning) || status.NewStatus != string(domain.RunStatusComplete) {
		t.Fatalf("status event = %#v, want running -> complete", statusEvents[0].Data)
	}
}

func TestHandleResultPublishesNeedsReviewAfterFinalization(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	cfg := domain.DefaultSandboxConfig()
	cfg.ManualReview = true
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeSandboxed,
		Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ResolvedConfig: domain.DefaultRunConfig(), SandboxConfig: cfg,
	}
	if err := repos.Runs.Create(t.Context(), run); err != nil {
		t.Fatal(err)
	}
	events := mocks.NewFakeEventStore()
	provider := mocks.NewFakeSandboxProvider()
	provider.GetDiffFunc = func(context.Context, uuid.UUID) (*sandbox.DiffResult, error) {
		return nil, errors.New("diff evidence unavailable")
	}
	sandboxID := uuid.New()
	HandleResult(t.Context(), HandleResultInput{
		Deps: Deps{Runs: repos.Runs, Events: events},
		Run:  run, Sandbox: provider, SandboxID: &sandboxID,
		Result: &runner.ExecuteResult{
			Success: true,
			Result:  &domain.RunResult{Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected}},
		},
	})

	if run.Status != domain.RunStatusNeedsReview {
		t.Fatalf("run status = %s, want needs_review", run.Status)
	}
	statusEvents := events.TypedEvents(run.ID, domain.EventTypeStatus)
	if len(statusEvents) != 1 {
		t.Fatalf("status event count = %d, want exactly one persisted transition", len(statusEvents))
	}
	status, ok := statusEvents[0].Data.(*domain.StatusEventData)
	if !ok || status.NewStatus != string(domain.RunStatusNeedsReview) {
		t.Fatalf("status event = %#v, want needs_review transition", statusEvents[0].Data)
	}
}

func TestHandleResultDoesNotAdoptCancellationIntentFromNewerOwner(t *testing.T) {
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeInPlace,
		Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ResolvedConfig: domain.DefaultRunConfig(), OwnerIdentity: "owner-a", OwnerEpoch: 4,
	}
	repo := &cancelRaceRunRepo{run: run}
	if changed, err := repo.RequestCancellation(context.Background(), run.ID, time.Now().UTC()); err != nil || !changed {
		t.Fatalf("request cancellation: changed=%v err=%v", changed, err)
	}
	// A newer owner claims the same run after cancellation intent. The stale
	// executor must not consume that owner transition as its own cancellation.
	repo.run.OwnerIdentity = "owner-b"
	repo.run.OwnerEpoch++
	out := HandleResult(context.Background(), HandleResultInput{
		Deps: Deps{Runs: repo}, Run: run,
		Result: &runner.ExecuteResult{Success: true, ExitCode: 0, Result: &domain.RunResult{Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected}}},
	})
	if out.Outcome == domain.RunOutcomeCancelled || run.Status != domain.RunStatusRunning || run.CancelRequestedAt != nil || run.LifecycleVersion != 0 {
		t.Fatalf("stale owner adopted newer cancellation state: outcome=%s run=%+v", out.Outcome, run)
	}
	if repo.run.Status != domain.RunStatusRunning || repo.run.OwnerIdentity != "owner-b" || repo.run.OwnerEpoch != 5 {
		t.Fatalf("newer durable owner was changed: %+v", repo.run)
	}
}

func TestHandleResultDoesNotProceedWhenDurableRunCannotBeRead(t *testing.T) {
	for _, test := range []struct {
		name    string
		readErr error
	}{
		{name: "read error", readErr: errors.New("temporary durable owner read failure")},
		{name: "missing record"},
	} {
		t.Run(test.name, func(t *testing.T) {
			repos, _, cleanup := testutil.SetupTestRepos(t)
			t.Cleanup(cleanup)
			ctx := t.Context()
			run := &domain.Run{
				ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeSandboxed,
				Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
				ResolvedConfig: domain.DefaultRunConfig(), SandboxConfig: domain.DefaultSandboxConfig(),
			}
			if err := repos.Runs.Create(ctx, run); err != nil {
				t.Fatal(err)
			}
			stale := *run
			canceled, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			canceled.Status = domain.RunStatusCancelled
			canceled.Phase = domain.RunPhaseCompleted
			if err := repos.Runs.Update(ctx, canceled); err != nil {
				t.Fatal(err)
			}
			provider := mocks.NewFakeSandboxProvider()
			diffReads := 0
			provider.GetDiffFunc = func(context.Context, uuid.UUID) (*sandbox.DiffResult, error) {
				diffReads++
				return &sandbox.DiffResult{}, nil
			}

			out := HandleResult(ctx, HandleResultInput{
				Deps:      Deps{Runs: failingRunReadRepository{RunRepository: repos.Runs, err: test.readErr}},
				Run:       &stale,
				Sandbox:   provider,
				SandboxID: ptrUUID(uuid.New()),
				Result: &runner.ExecuteResult{
					Success: true, ExitCode: 0,
					Result: &domain.RunResult{Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected}},
				},
			})

			if !out.Deferred {
				t.Fatal("result without a durable lifecycle read was not marked deferred for recovery")
			}
			if diffReads != 0 || provider.ApplyAtRunEndCallCount() != 0 {
				t.Fatalf("durable state unavailable but result handling entered sandbox path: GetDiff calls=%d apply calls=%d", diffReads, provider.ApplyAtRunEndCallCount())
			}
			if stale.Status != domain.RunStatusRunning || stale.LifecycleVersion != run.LifecycleVersion {
				t.Fatalf("run state advanced without durable reconciliation: status=%s version=%d", stale.Status, stale.LifecycleVersion)
			}
			durable, err := repos.Runs.Get(ctx, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if durable.Status != domain.RunStatusCancelled {
				t.Fatalf("durable terminal owner state changed: status=%s", durable.Status)
			}
		})
	}
}

func TestHandleResultDefersWhenTerminalWriteFails(t *testing.T) {
	repos, _, cleanup := testutil.SetupTestRepos(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	run := &domain.Run{
		ID: uuid.New(), Tag: uuid.NewString(), RunMode: domain.RunModeSandboxed,
		Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting,
		ResolvedConfig: domain.DefaultRunConfig(), SandboxConfig: domain.DefaultSandboxConfig(),
	}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	provider := mocks.NewFakeSandboxProvider()
	provider.ApplyAtRunEndResult = &sandbox.ApplyAtRunEndResult{Success: true, Applied: 1}

	out := HandleResult(ctx, HandleResultInput{
		Deps:      Deps{Runs: failingRunUpdateRepository{RunRepository: repos.Runs, err: errors.New("durable terminal write unavailable")}},
		Run:       run,
		Sandbox:   provider,
		SandboxID: ptrUUID(uuid.New()),
		Result: &runner.ExecuteResult{
			Success: true, ExitCode: 0,
			Result: &domain.RunResult{Success: true, Selection: domain.FinalOutputSelection{Status: domain.FinalOutputSelectionSelected}},
		},
	})

	if !out.Deferred || out.Persisted {
		t.Fatalf("terminal write failure was not deferred: %+v", out)
	}
	durable, err := repos.Runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if durable.Status != domain.RunStatusRunning {
		t.Fatalf("test seam changed durable run despite rejected write: status=%s", durable.Status)
	}
	if provider.ApplyAtRunEndCallCount() != 1 {
		t.Fatalf("expected one idempotent apply attempt before failed run write, got %d", provider.ApplyAtRunEndCallCount())
	}
}
