package pipeline

import "testing"

func TestGateBlockedState_Transitions(t *testing.T) {
	tests := []struct {
		name     string
		from     PipelineState
		to       PipelineState
		expected bool
	}{
		{"ExecutingStage → GateBlocked", PipelineStateExecutingStage, PipelineStateGateBlocked, true},
		{"GateBlocked → ExecutingStage", PipelineStateGateBlocked, PipelineStateExecutingStage, true},
		{"GateBlocked → Failed", PipelineStateGateBlocked, PipelineStateFailed, true},
		{"GateBlocked → Cancelled", PipelineStateGateBlocked, PipelineStateCancelled, true},
		{"GateBlocked → Completed (invalid)", PipelineStateGateBlocked, PipelineStateCompleted, false},
		{"GateBlocked → ProcessingResult (invalid)", PipelineStateGateBlocked, PipelineStateProcessingResult, false},
		{"QueueingStage → GateBlocked (invalid)", PipelineStateQueueingStage, PipelineStateGateBlocked, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.from.CanTransitionTo(tt.to)
			if result != tt.expected {
				t.Errorf("CanTransitionTo(%s → %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
			}
		})
	}
}

func TestGateBlockedState_IsNotTerminal(t *testing.T) {
	if PipelineStateGateBlocked.IsTerminal() {
		t.Error("GateBlocked should not be a terminal state")
	}
}

func TestStatus_TransitionToGateBlocked(t *testing.T) {
	status := &Status{
		CurrentState: PipelineStateExecutingStage,
		CurrentStage: "deploy",
	}

	ok := status.TransitionTo(PipelineStateGateBlocked, "Waiting for approval")
	if !ok {
		t.Fatal("expected transition to GateBlocked to succeed")
	}
	if status.CurrentState != PipelineStateGateBlocked {
		t.Errorf("expected state GateBlocked, got %s", status.CurrentState)
	}
	if len(status.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(status.Transitions))
	}
	if status.Transitions[0].From != PipelineStateExecutingStage {
		t.Errorf("expected from ExecutingStage, got %s", status.Transitions[0].From)
	}
	if status.Transitions[0].To != PipelineStateGateBlocked {
		t.Errorf("expected to GateBlocked, got %s", status.Transitions[0].To)
	}
}

func TestStatus_TransitionFromGateBlockedToExecuting(t *testing.T) {
	status := &Status{
		CurrentState: PipelineStateGateBlocked,
		CurrentStage: "deploy",
	}

	ok := status.TransitionTo(PipelineStateExecutingStage, "Gate cleared")
	if !ok {
		t.Fatal("expected transition from GateBlocked to ExecutingStage to succeed")
	}
	if status.CurrentState != PipelineStateExecutingStage {
		t.Errorf("expected state ExecutingStage, got %s", status.CurrentState)
	}
}

func TestStatus_ProgressMessage_GateBlocked(t *testing.T) {
	status := &Status{
		Status:       StatusRunning,
		CurrentState: PipelineStateGateBlocked,
		CurrentStage: "deploy",
		StageOrder:   []string{"bundle", "preflight", "generate", "build", "smoketest", "deploy"},
		Stages:       map[string]*StageResult{},
	}

	msg := status.ComputeProgressMessage()
	if msg != "Waiting for approval gate (deploy stage)" {
		t.Errorf("unexpected progress message: %s", msg)
	}
}
