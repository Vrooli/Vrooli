package orchestration

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type parkTurnPIDRunner struct {
	*runner.MockRunner
	pid atomic.Int32
}

func (r *parkTurnPIDRunner) PID(uuid.UUID) int { return int(r.pid.Load()) }

func TestWaitForProcessExitReportsUnconfirmedDrain(t *testing.T) {
	r := &parkTurnPIDRunner{MockRunner: runner.NewMockRunner(domain.RunnerTypeClaudeCode)}
	r.pid.Store(1234)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := waitForProcessExit(ctx, r, uuid.New())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("waitForProcessExit error = %v, want context cancellation while PID remains live", err)
	}
}

func TestWaitForProcessExitRejectsRunnerWithoutPIDEvidence(t *testing.T) {
	r := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	if err := waitForProcessExit(context.Background(), r, uuid.New()); err == nil {
		t.Fatal("waitForProcessExit accepted Stop-only evidence without a PID reporter")
	}
}

func TestAwaitParkedTurnEndRetainsFailedDrainUntilPIDExit(t *testing.T) {
	runID := uuid.New()
	r := &parkTurnPIDRunner{MockRunner: runner.NewMockRunner(domain.RunnerTypeClaudeCode)}
	r.pid.Store(1234)
	barrier := &parkedTurnEnd{done: make(chan struct{}), runner: r, err: errors.New("process exit was not confirmed")}
	close(barrier.done)
	o := &Orchestrator{}
	o.parkTurnEnds.Store(runID, barrier)

	if err := o.awaitParkedTurnEnd(context.Background(), runID); err == nil {
		t.Fatal("awaitParkedTurnEnd succeeded while the original process was still live")
	}
	if _, ok := o.parkTurnEnds.Load(runID); !ok {
		t.Fatal("failed drain barrier was discarded before process exit was confirmed")
	}

	r.pid.Store(0)
	if err := o.awaitParkedTurnEnd(context.Background(), runID); err != nil {
		t.Fatalf("awaitParkedTurnEnd after PID exit = %v, want success", err)
	}
	if _, ok := o.parkTurnEnds.Load(runID); ok {
		t.Fatal("confirmed-exited barrier was not retired")
	}
}

func TestPrepareParkedTurnEndPreservesExistingBarrier(t *testing.T) {
	runID := uuid.New()
	r := &parkTurnPIDRunner{MockRunner: runner.NewMockRunner(domain.RunnerTypeClaudeCode)}
	runners := runner.NewRegistry()
	if err := runners.Register(r); err != nil {
		t.Fatal(err)
	}
	o := &Orchestrator{runners: runners}
	existing := &parkedTurnEnd{done: make(chan struct{}), runner: r}
	o.parkTurnEnds.Store(runID, existing)
	run := &domain.Run{ID: runID, Status: domain.RunStatusRunning, ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}}

	if _, err := o.prepareParkedTurnEnd(run); err == nil {
		t.Fatal("prepareParkedTurnEnd accepted a duplicate barrier for the same run")
	}
	got, ok := o.parkTurnEnds.Load(runID)
	if !ok || got != existing {
		t.Fatal("duplicate prepare replaced the existing process-exit barrier")
	}
}
