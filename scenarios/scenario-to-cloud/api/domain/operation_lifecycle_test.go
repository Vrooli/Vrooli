package domain

import "testing"

func TestCloudOperationTransitionOwnerMutatesOnlyLegalStates(t *testing.T) {
	valid := &CloudOperation{State: OperationAdmitted}
	if err := valid.TransitionTo(OperationRunning); err != nil {
		t.Fatalf("admitted→running: %v", err)
	}
	if valid.State != OperationRunning {
		t.Fatalf("state = %s, want running", valid.State)
	}

	for _, tc := range []struct {
		name string
		from OperationState
		to   OperationState
	}{
		{name: "skipped phase", from: OperationRunning, to: OperationSucceeded},
		{name: "terminal revival", from: OperationFailed, to: OperationRunning},
		{name: "unknown source", from: OperationState("mystery"), to: OperationCancelled},
		{name: "unknown target", from: OperationRunning, to: OperationState("mystery")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			op := &CloudOperation{State: tc.from}
			if err := op.TransitionTo(tc.to); err == nil {
				t.Fatalf("TransitionTo(%s) unexpectedly succeeded", tc.to)
			}
			if op.State != tc.from {
				t.Fatalf("rejected transition mutated state to %s, want %s", op.State, tc.from)
			}
		})
	}
}
