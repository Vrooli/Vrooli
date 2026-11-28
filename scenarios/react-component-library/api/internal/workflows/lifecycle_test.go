package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkflowStatusTransitionMatrix(t *testing.T) {
	statuses := []Status{StatusQueued, StatusRunning, StatusParked, StatusSucceeded, StatusFailed, StatusStopped, StatusUnavailable}
	for _, from := range statuses {
		for _, to := range statuses {
			t.Run(string(from)+"_to_"+string(to), func(t *testing.T) {
				workflow := Workflow{Status: from}
				err := workflow.TransitionStatus(to)
				// Map lookup is the authoritative table; absent edges must reject.
				_, explicit := workflowStatusTransitions[from][to]
				allowed := from == to || explicit
				if allowed {
					require.NoError(t, err)
					require.Equal(t, to, workflow.Status)
				} else {
					require.ErrorIs(t, err, ErrInvalidStatusTransition)
					require.Equal(t, from, workflow.Status)
				}
			})
		}
	}
	workflow := Workflow{Status: StatusQueued}
	require.ErrorIs(t, workflow.TransitionStatus("invented"), ErrInvalidStatusTransition)
}

func TestSQLiteRepositoryRejectsIllegalStatusWriteWithoutMutation(t *testing.T) {
	ctx := context.Background()
	_, repo, _ := newWorkflowService(t, &fakeDispatcher{})
	created, err := repo.Create(ctx, Workflow{ID: "workflow-1", Kind: KindExtract, IdempotencyKey: "key", Status: StatusQueued})
	require.NoError(t, err)

	require.NoError(t, created.TransitionStatus(StatusSucceeded))
	created.Summary = "completed"
	created, err = repo.Update(ctx, created)
	require.NoError(t, err)
	created.Status = StatusRunning // terminal states cannot become active again
	created.Summary = "must not be persisted"
	_, err = repo.Update(ctx, created)
	require.ErrorIs(t, err, ErrInvalidStatusTransition)
	persisted, err := repo.Get(ctx, "workflow-1")
	require.NoError(t, err)
	require.Equal(t, StatusSucceeded, persisted.Status)
	require.Equal(t, "completed", persisted.Summary)
}
