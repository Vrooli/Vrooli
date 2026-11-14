package domain

import (
	"errors"
	"testing"
)

func TestWorkflowAttemptStatusTransitions(t *testing.T) {
	tests := []struct {
		from, to WorkflowAttemptStatus
		want     bool
	}{
		{WorkflowAttemptDispatchPending, WorkflowAttemptDispatched, true},
		{WorkflowAttemptDispatchPending, WorkflowAttemptFailed, true},
		{WorkflowAttemptDispatchPending, WorkflowAttemptCompleted, false},
		{WorkflowAttemptDispatched, WorkflowAttemptWaiting, true},
		{WorkflowAttemptDispatched, WorkflowAttemptDispatchPending, true}, // recovery can clear an unacknowledged handle and reuse idempotency
		{WorkflowAttemptWaiting, WorkflowAttemptDispatched, true},         // recovery can resume a waiting dispatch
		{WorkflowAttemptWaiting, WorkflowAttemptCompleted, true},
		{WorkflowAttemptCompleted, WorkflowAttemptFailed, false},
		{WorkflowAttemptFailed, WorkflowAttemptDispatched, false},
		{WorkflowAttemptStatus("unknown"), WorkflowAttemptFailed, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"-to-"+string(tt.to), func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.want {
				t.Fatalf("CanTransitionTo(%q) = %t, want %t", tt.to, got, tt.want)
			}
		})
	}
	if err := (*WorkflowNodeAttempt)(nil).TransitionStatus(WorkflowAttemptFailed); err == nil {
		t.Fatal("nil attempt transition unexpectedly succeeded")
	}
	a := &WorkflowNodeAttempt{Status: WorkflowAttemptCompleted}
	if err := a.TransitionStatus(WorkflowAttemptDispatched); !errors.Is(err, ErrInvalidWorkflowAttemptStatusTransition) {
		t.Fatalf("invalid transition error = %v", err)
	}
}
