package execution

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitions"
	"swarm-manager/internal/workflowcontract"
)

func TestGoalRunStatusMapsVerdictsAndInterruptions(t *testing.T) {
	cases := []struct {
		name       string
		state      agentmanager.GoalRunState
		wantStatus Status
		terminal   bool
	}{
		{"complete", agentmanager.GoalRunState{TerminalClass: "verdict", StopReason: "complete"}, StatusValidating, true},
		{"blocked", agentmanager.GoalRunState{TerminalClass: "verdict", StopReason: "blocked"}, StatusNeedsAttention, true},
		{"abstained", agentmanager.GoalRunState{TerminalClass: "verdict", StopReason: "abstained"}, StatusNeedsReview, true},
		{"timeout interruption", agentmanager.GoalRunState{TerminalClass: "interruption", StopReason: "timeout"}, StatusInterrupted, true},
		{"running", agentmanager.GoalRunState{TerminalClass: "", StopReason: ""}, "", false},
		{"running with status", agentmanager.GoalRunState{Status: "RUN_STATUS_RUNNING"}, "", false},
		{"starting with status", agentmanager.GoalRunState{Status: "RUN_STATUS_STARTING"}, "", false},
		{"launch failure", agentmanager.GoalRunState{Status: "RUN_STATUS_FAILED", ErrorMessage: "sandbox create failed"}, StatusFailed, true},
		{"cancelled before start", agentmanager.GoalRunState{Status: "RUN_STATUS_CANCELLED"}, StatusFailed, true},
		{"complete without verdict", agentmanager.GoalRunState{Status: "RUN_STATUS_COMPLETE"}, StatusNeedsReview, true},
		{"typed class wins over status", agentmanager.GoalRunState{Status: "RUN_STATUS_FAILED", TerminalClass: "interruption", StopReason: "timeout"}, StatusInterrupted, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, _, terminal := goalRunStatus(tc.state)
			if terminal != tc.terminal {
				t.Fatalf("terminal = %v, want %v", terminal, tc.terminal)
			}
			if terminal && status != tc.wantStatus {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
		})
	}
}

type stubGoalRunReader struct {
	state    agentmanager.GoalRunState
	err      error
	usage    *workflowcontract.Usage
	terminal bool
	usageErr error
}

func (s stubGoalRunReader) GetGoalRunState(context.Context, string) (agentmanager.GoalRunState, error) {
	return s.state, s.err
}

func (s stubGoalRunReader) GetGoalRunUsage(context.Context, string) (*workflowcontract.Usage, bool, error) {
	return s.usage, s.terminal, s.usageErr
}

// TestApplyReconciledGoalRunInitializesFinalization proves a goal verdict that
// maps to validating also initializes the finalization record. Without it the
// validating poller (which requires record.Finalization != nil) never picks the
// execution up and it sits in validating forever.
func TestApplyReconciledGoalRunInitializesFinalization(t *testing.T) {
	svc := newTestPollingService(t)
	svc.goalRunReader = stubGoalRunReader{state: agentmanager.GoalRunState{TerminalClass: "verdict", StopReason: "complete"}}
	rec := Record{
		ExecutionID:   "exec-goal",
		BacklogKind:   "execute",
		BacklogName:   "goal-item",
		RunID:         "run-goal",
		ExecutionMode: transitions.ExecutionModeGoal,
		Status:        StatusStarting,
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	changed, err := svc.applyReconciledGoalRun(context.Background(), "exec-goal")
	if err != nil || !changed {
		t.Fatalf("applyReconciledGoalRun: changed=%v err=%v", changed, err)
	}

	records, err := svc.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := records[0]
	if got.Status != StatusValidating {
		t.Fatalf("status = %q, want %q", got.Status, StatusValidating)
	}
	if got.Finalization == nil {
		t.Fatal("goal verdict did not initialize finalization; the validating poller will never pick it up")
	}
}

// TestApplyReconciledGoalRunFailsALaunchFailure proves a goal run that Agent
// Manager ended before its harness reported a typed terminal leaves `starting`.
// Before this, the execution sat starting forever and held its lane.
func TestApplyReconciledGoalRunFailsALaunchFailure(t *testing.T) {
	svc := newTestPollingService(t)
	svc.goalRunReader = stubGoalRunReader{state: agentmanager.GoalRunState{Status: "RUN_STATUS_FAILED", ErrorMessage: "sandbox create failed"}, terminal: true}
	rec := Record{
		ExecutionID:   "exec-goal",
		BacklogKind:   "execute",
		BacklogName:   "goal-item",
		RunID:         "run-goal",
		ExecutionMode: transitions.ExecutionModeGoal,
		Status:        StatusStarting,
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	changed, err := svc.applyReconciledGoalRun(context.Background(), "exec-goal")
	if err != nil || !changed {
		t.Fatalf("applyReconciledGoalRun: changed=%v err=%v", changed, err)
	}
	records, err := svc.store.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := records[0]
	if got.Status != StatusFailed {
		t.Fatalf("status = %q, want %q", got.Status, StatusFailed)
	}
	if !strings.Contains(got.FailureReason, "sandbox create failed") {
		t.Fatalf("failure reason %q does not carry the owner's error", got.FailureReason)
	}
	if isInFlightRecord(got) {
		t.Fatal("a launch-failed goal execution still holds its lane")
	}
}
