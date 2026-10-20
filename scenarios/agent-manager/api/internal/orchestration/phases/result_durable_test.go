package phases

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
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
