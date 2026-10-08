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
		Phase: domain.RunPhaseExecuting, ResolvedConfig: domain.DefaultRunConfig(),
	}
	repo := &cancelRaceRunRepo{run: run}

	// Simulate a stop request that persisted cancellation intent while the
	// process was still running (status is not yet terminal).
	stamp := time.Now().Add(-time.Second)
	intended, err := repo.Get(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	intended.CancelRequestedAt = &stamp
	if err := repo.Update(context.Background(), intended); err != nil {
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
