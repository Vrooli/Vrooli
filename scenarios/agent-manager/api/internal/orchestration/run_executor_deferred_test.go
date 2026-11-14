package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"
	"agent-manager/internal/repository"

	"github.com/google/uuid"
)

type deferredFinalizeRunRepository struct {
	repository.RunRepository
	run      *domain.Run
	getCalls int
	updates  int
}

func (r *deferredFinalizeRunRepository) Get(context.Context, uuid.UUID) (*domain.Run, error) {
	r.getCalls++
	copy := *r.run
	return &copy, nil
}

func (r *deferredFinalizeRunRepository) Update(context.Context, *domain.Run) error {
	r.updates++
	return nil
}

func TestDeferredResultFinalizationPreservesSandboxForRecovery(t *testing.T) {
	run := &domain.Run{ID: uuid.New(), Status: domain.RunStatusRunning, LifecycleVersion: 4}
	runs := &deferredFinalizeRunRepository{run: run}
	sandbox := mocks.NewFakeSandboxProvider()
	sandboxID := uuid.New()
	deleted, stopped := 0, 0
	terminalCallbacks := 0
	sandbox.DeleteFunc = func(context.Context, uuid.UUID) error { deleted++; return nil }
	sandbox.StopFunc = func(context.Context, uuid.UUID) error { stopped++; return nil }

	executor := &RunExecutor{
		runs:           runs,
		run:            run,
		sandbox:        sandbox,
		sandboxID:      &sandboxID,
		resultDeferred: true,
		onTerminal:     func(context.Context, *domain.Run) { terminalCallbacks++ },
	}
	executor.finalize(t.Context())

	if runs.getCalls != 1 {
		t.Fatalf("deferred result finalizer owner reads = %d, want one lifecycle fence", runs.getCalls)
	}
	if deleted != 0 || stopped != 0 {
		t.Fatalf("deferred result finalizer destroyed recovery sandbox: deletes=%d stops=%d", deleted, stopped)
	}
	if runs.updates != 0 || terminalCallbacks != 0 {
		t.Fatalf("deferred result was projected as terminal: updates=%d callbacks=%d", runs.updates, terminalCallbacks)
	}
	if executor.finalized {
		t.Fatal("deferred result was marked finalized before recovery")
	}
}
