package execution

import (
	"context"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/opsrunner"
)

// commitExecutionRoundForTest drives the completion handler with a synthetic
// ActionContext correlating to a record by its OpExecutionID, mirroring what the
// dispatcher passes when the completion bridge commits an execution operation.
func commitExecutionRoundForTest(t *testing.T, svc *Service, opExecutionID, outcome string) {
	t.Helper()
	if err := svc.commitExecutionRound(context.Background(), opsrunner.ActionContext{
		ExecutionID: opExecutionID,
		Outcome:     outcome,
	}); err != nil {
		t.Fatalf("commitExecutionRound(%q): %v", outcome, err)
	}
}

// TestCommitExecutionRound_CompletedDrivesRecordToValidating proves the bridge is
// the completion authority for a runner-owned record: a completed execution
// operation drives the correlated record into finalization exactly as the poller
// did — WITHOUT polling agent-manager.
func TestCommitExecutionRound_CompletedDrivesRecordToValidating(t *testing.T) {
	// Nil inspector: if the handler tried to poll it would panic. It must not.
	svc := newTestPollingService(t, nil)
	seedBacklogSpec(t, svc, "execute", "runner-owned", "in_progress")

	rec := Record{
		ExecutionID:   "exec-owned",
		BacklogKind:   "execute",
		BacklogName:   "runner-owned",
		RunID:         "run-owned",
		OpExecutionID: "op-owned",
		Status:        StatusRunning,
		PromptTrace:   &PromptTrace{Purpose: "process"}, // finalization-eligible
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if !isFinalizationEligible(rec) {
		t.Fatal("test setup invalid: record must be finalization-eligible")
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	commitExecutionRoundForTest(t, svc, "op-owned", "completed")

	got, _, err := svc.loadRecordLocked("exec-owned")
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got[0].Status != StatusValidating {
		t.Fatalf("record status = %q, want %q (bridge-driven finalization)", got[0].Status, StatusValidating)
	}
	if got[0].Finalization == nil || !got[0].Finalization.Eligible {
		t.Fatal("expected finalization state to be seeded by the completion handler")
	}
}

// TestCommitExecutionRound_BlockedDrivesRecordToFailed proves a blocked/abstain
// execution operation parks the record failed (operator review), mirroring the
// poller's run-failed path.
func TestCommitExecutionRound_BlockedDrivesRecordToFailed(t *testing.T) {
	svc := newTestPollingService(t, nil)
	specPath := seedBacklogSpec(t, svc, "execute", "blocked-item", "in_progress")

	rec := Record{
		ExecutionID:   "exec-blocked",
		BacklogKind:   "execute",
		BacklogName:   "blocked-item",
		RunID:         "run-blocked",
		OpExecutionID: "op-blocked",
		Status:        StatusRunning,
		PromptTrace:   &PromptTrace{Purpose: "process"},
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	commitExecutionRoundForTest(t, svc, "op-blocked", "blocked")

	got, _, err := svc.loadRecordLocked("exec-blocked")
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got[0].Status != StatusFailed {
		t.Fatalf("record status = %q, want %q", got[0].Status, StatusFailed)
	}
	// A failed run lands the backlog item in in_review (operator decides terminal).
	if s := loadSpecStatus(t, specPath); s != backlogStatusInReview {
		t.Fatalf("backlog status = %q, want %q", s, backlogStatusInReview)
	}
}

// TestCommitExecutionRound_ContinueIsNoOp proves a "continue" outcome (the
// operation loops another round) leaves the record running.
func TestCommitExecutionRound_ContinueIsNoOp(t *testing.T) {
	svc := newTestPollingService(t, nil)
	rec := Record{
		ExecutionID:   "exec-cont",
		RunID:         "run-cont", // non-empty so the store does not self-heal a run-less record
		OpExecutionID: "op-cont",
		Status:        StatusRunning,
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}
	commitExecutionRoundForTest(t, svc, "op-cont", "continue")
	got, _, err := svc.loadRecordLocked("exec-cont")
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got[0].Status != StatusRunning {
		t.Fatalf("record status = %q, want %q (continue is not terminal)", got[0].Status, StatusRunning)
	}
}

// TestPolling_SkipsRunnerOwnedRecords proves the poller DEFERS a runner-owned
// record (OpExecutionID set): even though agent-manager would report the run
// complete, inspectRunningRecordsLocked leaves the record untouched so the
// completion bridge is the sole authority.
func TestPolling_SkipsRunnerOwnedRecords(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-owned": {RunID: "run-owned", Status: "complete"},
		},
	}
	svc := newTestPollingService(t, inspector)
	seedBacklogSpec(t, svc, "execute", "owned", "in_progress")

	rec := Record{
		ExecutionID:   "exec-owned",
		BacklogKind:   "execute",
		BacklogName:   "owned",
		RunID:         "run-owned",
		OpExecutionID: "op-owned", // runner-owned: poller must defer
		Status:        StatusRunning,
		PromptTrace:   &PromptTrace{Purpose: "process"},
		CreatedAt:     nowRFC3339(),
		UpdatedAt:     nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	got, _, err := svc.loadRecordLocked("exec-owned")
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got[0].Status != StatusRunning {
		t.Fatalf("runner-owned record status = %q, want %q (poller must defer)", got[0].Status, StatusRunning)
	}
}

// TestPolling_StillDrivesNonRunnerRecords proves the deferral is scoped: a record
// with NO OpExecutionID (a pre-migration/legacy run) is still driven by the
// poller, so the authority flip does not strand non-runner records.
func TestPolling_StillDrivesNonRunnerRecords(t *testing.T) {
	inspector := &mockInspector{
		states: map[string]agentmanager.RunState{
			"run-legacy": {RunID: "run-legacy", Status: "complete"},
		},
	}
	svc := newTestPollingService(t, inspector)
	seedBacklogSpec(t, svc, "execute", "legacy", "in_progress")

	rec := Record{
		ExecutionID: "exec-legacy",
		BacklogKind: "execute",
		BacklogName: "legacy",
		RunID:       "run-legacy",
		// No OpExecutionID: legacy/non-runner record — poller still owns it.
		Status:      StatusRunning,
		PromptTrace: &PromptTrace{Purpose: "process"},
		CreatedAt:   nowRFC3339(),
		UpdatedAt:   nowRFC3339(),
	}
	if err := svc.store.Save([]Record{rec}); err != nil {
		t.Fatal(err)
	}

	runRefresh(t, svc)

	got, _, err := svc.loadRecordLocked("exec-legacy")
	if err != nil {
		t.Fatalf("load record: %v", err)
	}
	if got[0].Status != StatusValidating {
		t.Fatalf("non-runner record status = %q, want %q (poller still drives it)", got[0].Status, StatusValidating)
	}
}
