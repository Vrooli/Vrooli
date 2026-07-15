package initiativereview

import (
	"context"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/opsrunner"
	"swarm-manager/internal/review"
)

// TestCommitInitiativeReview_MaterializesAndFlipsToReviewPending is the core
// new-behavior test: the completing initiative-review operation's validated result
// is materialized into the runner-owned round (assessment + classification +
// evidence), the round becomes complete, and the initiative flips
// in_review -> review_pending via handleTerminalRound.
func TestCommitInitiativeReview_MaterializesAndFlipsToReviewPending(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("commit-init", "Commit", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "commit-init")

	// A runner-owned gathering round, correlated by run id, plus in_review status.
	e.writeRunnerOwnedGatheringRound("commit-init", "run-1", "e1")

	ac := newCommitInitiativeReviewActionContext("commit-init", "e1", "run-1",
		"accepted", initiativeReviewResultJSON(t, "ready", "The initiative delivered its stated goal."))
	if err := e.svc.commitInitiativeReview(context.Background(), ac); err != nil {
		t.Fatalf("commitInitiativeReview: %v", err)
	}

	round, err := review.LoadRound(e.initStore.InitDir("commit-init"), 1)
	if err != nil || round == nil {
		t.Fatalf("load round: err=%v round=%v", err, round)
	}
	if round.Status != review.RoundStatusComplete {
		t.Fatalf("round status = %q, want complete", round.Status)
	}
	if round.Classification != "ready" {
		t.Errorf("classification = %q, want ready", round.Classification)
	}
	if round.AgentAssessment != "The initiative delivered its stated goal." {
		t.Errorf("assessment = %q", round.AgentAssessment)
	}
	if len(round.Evidence) != 1 || round.Evidence[0].ID != "ev1" {
		t.Errorf("evidence not materialized: %#v", round.Evidence)
	}

	init, _ := e.initStore.Load("commit-init")
	if init.Status != initiatives.InitiativeStatusReviewPending {
		t.Fatalf("expected review_pending, got %s", init.Status)
	}
}

// TestCommitInitiativeReview_IdempotentReplay verifies a re-delivered completion
// for an already-terminal round is a no-op — the round is untouched.
func TestCommitInitiativeReview_IdempotentReplay(t *testing.T) {
	e := newEnv(t)
	e.seedItem("execute", "alpha", "Alpha", backlog.StatusCompleted)
	e.createInitiative("replay-init", "Replay", "execute/alpha")
	e.setItemInitiative("execute", "alpha", "replay-init")
	e.writeRunnerOwnedGatheringRound("replay-init", "run-1", "e1")

	ac := newCommitInitiativeReviewActionContext("replay-init", "e1", "run-1",
		"accepted", initiativeReviewResultJSON(t, "ready", "Delivered."))
	if err := e.svc.commitInitiativeReview(context.Background(), ac); err != nil {
		t.Fatalf("first commit: %v", err)
	}
	// Flip the initiative to completed to prove the replay does not re-open the gate.
	init, _ := e.initStore.Load("replay-init")
	init.Status = initiatives.InitiativeStatusCompleted
	_ = e.initStore.Save(init)

	if err := e.svc.commitInitiativeReview(context.Background(), ac); err != nil {
		t.Fatalf("replay commit: %v", err)
	}
	reloaded, _ := e.initStore.Load("replay-init")
	if reloaded.Status != initiatives.InitiativeStatusCompleted {
		t.Fatalf("replay must not re-flip status; got %s", reloaded.Status)
	}
}

// TestRegisterOpsHandlers_BindsClosedVocabulary verifies the initiative-review
// completion handler registers under the closed-vocabulary action name. Register
// panics on an unregistered name, so a clean return proves commit-initiative-review
// is a legal registered action.
func TestRegisterOpsHandlers_BindsClosedVocabulary(t *testing.T) {
	e := newEnv(t)
	// Would panic if commit-initiative-review weren't a registered action.
	e.svc.RegisterOpsHandlers(opsrunner.NewActionRegistry())
}

// writeRunnerOwnedGatheringRound writes a runner-owned gathering round (carrying
// OpExecutionID + RunID) directly to disk and puts the initiative in_review, so a
// commit-initiative-review completion can drive it to terminal.
func (e *env) writeRunnerOwnedGatheringRound(initiativeName, runID, opExecutionID string) {
	e.t.Helper()
	itemDir := e.initStore.InitDir(initiativeName)
	if err := review.SaveRound(itemDir, review.Round{
		RoundNum:      1,
		GeneratedAt:   e.clock().UTC().Format("2006-01-02T15:04:05Z07:00"),
		Status:        review.RoundStatusGathering,
		RunID:         runID,
		OpWorkflowID:  "wf-1",
		OpExecutionID: opExecutionID,
		Evidence:      []review.EvidenceItem{},
	}); err != nil {
		e.t.Fatal(err)
	}
	init, err := e.initStore.Load(initiativeName)
	if err != nil {
		e.t.Fatal(err)
	}
	init.Status = initiatives.InitiativeStatusInReview
	if err := e.initStore.Save(init); err != nil {
		e.t.Fatal(err)
	}
}
