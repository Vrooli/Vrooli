package review

import (
	"context"
	"encoding/json"
	"testing"

	"swarm-manager/internal/agentops"
	"swarm-manager/internal/opsrunner"
)

// reviewResultEnvelopeJSON builds the reviewResult contract payload the completion
// bridge delivers: the normalized verdict plus the enriched review handoff.
func reviewResultEnvelopeJSON(t *testing.T, verdict, assessment string, evidence []EvidenceItem) json.RawMessage {
	t.Helper()
	handoff := map[string]any{
		"verdict":               verdict,
		"agent_assessment":      assessment,
		"evidence":              evidence,
		"regression_introduced": false,
		"notes":                 []string{"reviewed via operation runner"},
	}
	handoffJSON, err := json.Marshal(handoff)
	if err != nil {
		t.Fatalf("marshal handoff: %v", err)
	}
	envelope, err := json.Marshal(map[string]any{
		"verdict": verdict,
		"handoff": json.RawMessage(handoffJSON),
	})
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return envelope
}

// newCommitReviewActionContext assembles the ActionContext the runner completion
// bridge hands the commit-review-round handler for a completing review operation.
func newCommitReviewActionContext(targetID, executionID, runID, outcome string, result json.RawMessage) opsrunner.ActionContext {
	return opsrunner.ActionContext{
		Target: opsrunner.TargetRef{Kind: agentops.TargetBacklogItem, ID: targetID},
		Workflow: agentops.WorkflowInstance{
			Operations: []agentops.OperationExecutionRecord{
				{ExecutionID: executionID, RunID: runID},
			},
		},
		Action:      agentops.ActionCommitReviewRound,
		ExecutionID: executionID,
		Outcome:     outcome,
		Result:      result,
	}
}

// TestCommitReviewRound_MaterializesResultAndOpensGate is the core new-behavior
// test: a completing review-round operation's validated result is materialized
// into the runner-owned gathering round (evidence + assessment + classification),
// the round becomes complete, and the operator review gate (onRoundTerminal) fires.
func TestCommitReviewRound_MaterializesResultAndOpensGate(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "")
	itemDir := t.TempDir()
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	// A runner-owned gathering round, correlated by run id.
	writeRound(t, itemDir, Round{
		RoundNum:      1,
		GeneratedAt:   "2026-04-02T00:00:00Z",
		ExecutionID:   "exec-1",
		Status:        RoundStatusGathering,
		RunID:         "run-1",
		OpWorkflowID:  "wf-1",
		OpExecutionID: "e1",
		Evidence:      []EvidenceItem{},
	})

	var gotKind, gotName string
	var gotRound Round
	var callbacks int
	svc.onRoundTerminal = func(_ context.Context, kind, name string, r Round) {
		callbacks++
		gotKind, gotName, gotRound = kind, name, r
	}

	result := reviewResultEnvelopeJSON(t, "ready", "The item met its acceptance criteria.",
		[]EvidenceItem{{ID: "ev1", Type: EvidenceTypeCLIOutput, Title: "tests", Description: "green"}})
	ac := newCommitReviewActionContext("task/x", "e1", "run-1", "accepted", result)

	if err := svc.commitReviewRound(context.Background(), ac); err != nil {
		t.Fatalf("commitReviewRound: %v", err)
	}

	round, err := LoadRound(itemDir, 1)
	if err != nil || round == nil {
		t.Fatalf("load round: err=%v round=%v", err, round)
	}
	if round.Status != RoundStatusComplete {
		t.Fatalf("round status = %q, want complete", round.Status)
	}
	if round.Classification != "ready" {
		t.Errorf("classification = %q, want ready", round.Classification)
	}
	if round.AgentAssessment != "The item met its acceptance criteria." {
		t.Errorf("assessment = %q", round.AgentAssessment)
	}
	if len(round.Evidence) != 1 || round.Evidence[0].ID != "ev1" {
		t.Errorf("evidence not materialized: %#v", round.Evidence)
	}

	// The operator review gate must have opened (in_review -> review_pending).
	if callbacks != 1 {
		t.Fatalf("onRoundTerminal fired %d times, want 1", callbacks)
	}
	if gotKind != "task" || gotName != "x" {
		t.Errorf("callback ref = %s/%s, want task/x", gotKind, gotName)
	}
	if gotRound.Status != RoundStatusComplete {
		t.Errorf("callback round status = %q, want complete", gotRound.Status)
	}
}

// TestCommitReviewRound_AbstainMarksFailedPreservingArtifacts pins the
// abstain/failure contract: a non-success outcome finalizes the round FAILED
// while preserving the gathered evidence + assessment for operator inspection.
func TestCommitReviewRound_AbstainMarksFailedPreservingArtifacts(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "")
	itemDir := t.TempDir()
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	writeRound(t, itemDir, Round{
		RoundNum:      1,
		GeneratedAt:   "2026-04-02T00:00:00Z",
		ExecutionID:   "exec-1",
		Status:        RoundStatusGathering,
		RunID:         "run-1",
		OpExecutionID: "e1",
		Evidence:      []EvidenceItem{},
	})

	var callbacks int
	svc.onRoundTerminal = func(_ context.Context, _, _ string, _ Round) { callbacks++ }

	result := reviewResultEnvelopeJSON(t, "not_assessable", "Could not derive an honest verdict.",
		[]EvidenceItem{{ID: "ev1", Type: EvidenceTypeCLIOutput, Title: "partial", Description: "incomplete"}})
	ac := newCommitReviewActionContext("task/x", "e1", "run-1", "needs-attention", result)

	if err := svc.commitReviewRound(context.Background(), ac); err != nil {
		t.Fatalf("commitReviewRound: %v", err)
	}

	round, _ := LoadRound(itemDir, 1)
	if round.Status != RoundStatusFailed {
		t.Fatalf("round status = %q, want failed", round.Status)
	}
	if round.FailureReason == "" {
		t.Error("expected a failure reason on the abstained round")
	}
	// Artifacts must survive so the operator can see what the agent did gather.
	if round.AgentAssessment == "" {
		t.Error("assessment should be preserved on an abstained round")
	}
	if len(round.Evidence) != 1 {
		t.Errorf("evidence should be preserved on an abstained round, got %d", len(round.Evidence))
	}
	// The gate still opens so the item leaves in_review.
	if callbacks != 1 {
		t.Fatalf("onRoundTerminal fired %d times, want 1", callbacks)
	}
}

// TestCommitReviewRound_IdempotentReplay verifies a re-delivered completion for an
// already-terminal round is a no-op: the round is untouched and the gate does not
// re-fire.
func TestCommitReviewRound_IdempotentReplay(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "")
	itemDir := t.TempDir()
	svc.itemDirFn = func(_, _ string) string { return itemDir }

	writeRound(t, itemDir, Round{
		RoundNum:      1,
		GeneratedAt:   "2026-04-02T00:00:00Z",
		ExecutionID:   "exec-1",
		Status:        RoundStatusGathering,
		RunID:         "run-1",
		OpExecutionID: "e1",
		Evidence:      []EvidenceItem{},
	})

	var callbacks int
	svc.onRoundTerminal = func(_ context.Context, _, _ string, _ Round) { callbacks++ }

	result := reviewResultEnvelopeJSON(t, "ready", "Looks good.",
		[]EvidenceItem{{ID: "ev1", Type: EvidenceTypeCLIOutput, Title: "t", Description: "d"}})
	ac := newCommitReviewActionContext("task/x", "e1", "run-1", "accepted", result)

	if err := svc.commitReviewRound(context.Background(), ac); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	// Second delivery of the same operation result: must be a no-op.
	if err := svc.commitReviewRound(context.Background(), ac); err != nil {
		t.Fatalf("replay commit: %v", err)
	}

	if callbacks != 1 {
		t.Fatalf("onRoundTerminal fired %d times across duplicate deliveries, want 1", callbacks)
	}
}

// TestRegisterOpsHandlers_BindsClosedVocabulary verifies the review completion
// handlers register onto a fresh action registry under the closed-vocabulary
// action names. Register panics on an unregistered name, so a clean return proves
// commit-review-round + request-evidence are legal registered actions.
func TestRegisterOpsHandlers_BindsClosedVocabulary(t *testing.T) {
	spawner := &capturingSpawner{enabled: true}
	svc := newTestService(spawner, "")
	svc.itemDirFn = func(_, _ string) string { return t.TempDir() }

	// Would panic if commit-review-round / request-evidence weren't registered
	// actions; reaching the assertion means the binding is legal.
	svc.RegisterOpsHandlers(opsrunner.NewActionRegistry())
}
