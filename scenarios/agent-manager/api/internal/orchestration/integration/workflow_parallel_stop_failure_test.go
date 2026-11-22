package integration

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

type failingStopLifecycleLauncher struct {
	*lifecycleChildLauncher
	stopErr   error
	stopCalls int
}

func (l *failingStopLifecycleLauncher) Stop(context.Context, uuid.UUID) error {
	l.stopCalls++
	return l.stopErr
}

// A provider failure while stopping an any-join sibling must leave the join
// uncommitted. The winner's success alone is not enough to persist a
// short-circuit disposition for a sibling the provider could not stop.
func TestWorkflowLifecycle_AnyJoinStopFailureDoesNotAdvanceOrShortCircuit(t *testing.T) {
	ctx := context.Background()
	stopErr := errors.New("provider refused stop")
	launcher := &failingStopLifecycleLauncher{lifecycleChildLauncher: newLifecycleChildLauncher(), stopErr: stopErr}
	fixture := newAnyJoinLifecycleFixture(t, launcher, launcher.lifecycleChildLauncher)
	harness, engine, execution := fixture.Harness, fixture.Engine, fixture.Execution
	launcher.lifecycleChildLauncher.complete(fixture.WinnerID)
	if _, err := engine.Advance(ctx, execution.ID); err != nil {
		t.Fatalf("persist winner result: %v", err)
	}
	if _, err := engine.Advance(ctx, execution.ID); !errors.Is(err, stopErr) {
		t.Fatalf("advance any join error = %v, want provider stop error %v", err, stopErr)
	}
	if launcher.stopCalls != 1 {
		t.Fatalf("provider Stop calls = %d, want one", launcher.stopCalls)
	}

	stored, err := harness.Repos.WorkflowExecutions.Get(ctx, execution.ID)
	if err != nil {
		t.Fatalf("get execution after stop failure: %v", err)
	}
	if stored.CurrentNodeID == "joined" || stored.Status != domain.WorkflowExecutionRunning {
		t.Fatalf("failed stop advanced execution: status=%s currentNode=%q", stored.Status, stored.CurrentNodeID)
	}
	loserState, err := launcher.Inspect(ctx, fixture.LoserID)
	if err != nil {
		t.Fatalf("inspect active sibling: %v", err)
	}
	if loserState.Terminal || loserState.Failed {
		t.Fatalf("provider state changed despite failed Stop: %+v", loserState)
	}
	attempts, err := harness.Repos.WorkflowExecutions.ListAttempts(ctx, execution.ID)
	if err != nil {
		t.Fatalf("list attempts after stop failure: %v", err)
	}
	var winnerCompleted, loserStillActive bool
	for _, attempt := range attempts {
		switch attempt.NodeID {
		case "winner":
			winnerCompleted = attempt.Status == domain.WorkflowAttemptCompleted
		case "loser":
			loserStillActive = (attempt.Status == domain.WorkflowAttemptDispatched || attempt.Status == domain.WorkflowAttemptWaiting) && attempt.ErrorCode != "parallel_join_short_circuit"
		}
	}
	if !winnerCompleted || !loserStillActive {
		t.Fatalf("failed stop changed durable join disposition: winnerCompleted=%v loserStillActive=%v attempts=%+v", winnerCompleted, loserStillActive, attempts)
	}
}
