package workflowruntime

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type qualificationFixture struct {
	candidate                QualificationCandidate
	requests                 []QualificationRequest
	prepares, effects, waits int
	lostResponse             bool
	passed                   bool
	closes, observations     int
	draining, unknownUsage   bool
	prepareErr               error
}

func (q *qualificationFixture) Prepare(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (QualificationCandidate, error) {
	q.prepares++
	if q.prepareErr != nil {
		return QualificationCandidate{}, q.prepareErr
	}
	return q.candidate, nil
}
func (q *qualificationFixture) Start(_ context.Context, req QualificationRequest) (QualificationState, error) {
	if len(q.requests) == 0 {
		q.effects++
	}
	q.requests = append(q.requests, req)
	if q.lostResponse {
		q.lostResponse = false
		return QualificationState{}, errors.New("lost admission response")
	}
	return QualificationState{ProgramID: "program-original"}, nil
}
func (q *qualificationFixture) Wait(_ context.Context, _ QualificationRequest, id string) (QualificationState, error) {
	q.waits++
	return q.state(), nil
}
func (q *qualificationFixture) state() QualificationState {
	return QualificationState{ProgramID: "program-original", Terminal: !q.draining, Passed: q.passed && !q.draining, Output: json.RawMessage(`{"status":"ok"}`), BudgetUsage: domain.WorkflowBudgetUsage{AccountingComplete: !q.draining && !q.unknownUsage, ChargeMeasured: !q.draining && !q.unknownUsage}}
}
func (q *qualificationFixture) Observe(context.Context, QualificationRequest) (QualificationState, error) {
	q.observations++
	return q.state(), nil
}
func (q *qualificationFixture) CloseAdmission(context.Context, QualificationRequest) (QualificationState, error) {
	q.closes++
	return q.state(), nil
}

func qualificationEngine(t *testing.T) (*Engine, *memoryStore, *domain.WorkflowExecution, *qualificationFixture) {
	t.Helper()
	return qualificationEngineWithInputs(t, json.RawMessage(`{}`), nil)
}

func qualificationEngineWithInputs(t *testing.T, input json.RawMessage, bindings []domain.WorkflowInputBinding) (*Engine, *memoryStore, *domain.WorkflowExecution, *qualificationFixture) {
	t.Helper()
	d := baseDefinition()
	d.EntryNode = "qualify"
	d.Nodes = []domain.WorkflowNode{{ID: "qualify", Kind: domain.WorkflowNodeQualification, Qualification: &domain.WorkflowQualificationNode{ReviewFromNode: "review", ProgramName: "example.qualify", ProgramDigest: strings.Repeat("a", 64), Bindings: bindings}}, {ID: "end", Kind: domain.WorkflowNodeEnd, End: &domain.WorkflowEndNode{}}}
	d.Edges = []domain.WorkflowEdge{{From: "qualify", To: "end"}}
	e, store, children := testEngine(t, d)
	q := &qualificationFixture{candidate: QualificationCandidate{ReviewerRunID: uuid.New(), SourceRunID: uuid.New(), SandboxID: uuid.New(), ReviewRequestID: uuid.New(), SHA256: strings.Repeat("b", 64), VerdictSHA256: strings.Repeat("c", 64)}, passed: true}
	e.Qualifications = q
	x, err := e.Start(t.Context(), revision(d), input, "qualification-test")
	require.NoError(t, err)
	x.Version++
	now := time.Now()
	review := &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: "review", Ordinal: 1, Strategy: domain.WorkflowAttemptFreshRun, Status: domain.WorkflowAttemptCompleted, RunID: &q.candidate.ReviewerRunID, CreatedAt: now, UpdatedAt: now, CompletedAt: &now, Version: 1}
	q.candidate.ReviewAttemptID = review.ID
	children.states[q.candidate.ReviewerRunID] = ChildState{RunID: q.candidate.ReviewerRunID, Terminal: true, TokensKnown: true, ChargeMeasured: true, Tokens: 7, Turns: 1}
	ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: review})
	require.NoError(t, err)
	require.True(t, ok)
	return e, store, x, q
}

func TestQualificationOwnerInputCannotBeShrunkByWorkerOutput(t *testing.T) {
	input := json.RawMessage(`{"required_rows":["evidence-completeness","profile-durability"]}`)
	e, store, x, q := qualificationEngineWithInputs(t, input, []domain.WorkflowInputBinding{{
		Name: "required_rows", Source: domain.WorkflowBindingInput, Selector: "$.required_rows",
		Limit: 1, MaxBytes: 1024, RenderAs: "json", MissingPolicy: "error",
	}})
	// A worker's later structured result cannot replace admission
	// input selected by the owner's immutable qualification binding.
	journal, err := store.ListJournal(t.Context(), x.ID, 0, 0)
	require.NoError(t, err)
	x.Version++
	forged := json.RawMessage(`{"required_rows":["readiness"]}`)
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalStructured, "worker", nil, forged, time.Now())
	ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Journal: []*domain.WorkflowJournalEntry{entry}})
	require.NoError(t, err)
	require.True(t, ok)
	x = mustAdvance(t, e, x.ID)
	require.Zero(t, q.effects, "the owner must retain its selection before any qualification effect")
	q.lostResponse = true
	_, err = e.Advance(t.Context(), x.ID)
	require.ErrorContains(t, err, "lost admission")
	restarted := *e
	_ = mustAdvance(t, &restarted, x.ID)
	require.Len(t, q.requests, 2)
	require.JSONEq(t, string(input), string(q.requests[0].Inputs))
	require.Equal(t, q.requests[0], q.requests[1], "recovery must retain the original owner selection")
	require.Equal(t, 1, q.effects)
}

func TestQualificationPersistsAcceptanceBeforeEffectsAndRecoversLostResponse(t *testing.T) {
	e, store, x, q := qualificationEngine(t)
	q.lostResponse = true
	x = mustAdvance(t, e, x.ID)
	require.Equal(t, 0, q.effects, "prepare/commit must not execute qualification")
	attempts, err := store.ListAttempts(t.Context(), x.ID)
	require.NoError(t, err)
	require.Len(t, attempts, 2)
	_, err = e.Advance(t.Context(), x.ID)
	require.ErrorContains(t, err, "lost admission")
	restarted := *e
	x = mustAdvance(t, &restarted, x.ID)
	require.Equal(t, 1, q.effects)
	require.Equal(t, 1, q.prepares, "restart must not reselect acceptance or inputs")
	require.Len(t, q.requests, 2)
	require.Equal(t, q.requests[0], q.requests[1], "lost response renewed qualification intent")
	x = mustAdvance(t, &restarted, x.ID)
	require.Equal(t, "end", x.CurrentNodeID)
	require.Equal(t, 1, q.waits)
	journal, err := store.ListJournal(t.Context(), x.ID, 0, 0)
	require.NoError(t, err)
	values, err := EvaluateBindings([]domain.WorkflowInputBinding{{Name: "qualified", Source: domain.WorkflowBindingQualification, Selector: "node=qualify;$.passed", Order: "desc", Limit: 1, MaxBytes: 1024, RenderAs: "json", MissingPolicy: "error"}}, BindingContext{Journal: journal})
	require.NoError(t, err)
	require.Equal(t, true, values["qualified"])
}

func TestQualificationCannotConsumeSameCandidateAgainAfterFailure(t *testing.T) {
	e, store, x, q := qualificationEngine(t)
	q.passed = false
	for range 3 {
		x = mustAdvance(t, e, x.ID)
	}
	require.Equal(t, "end", x.CurrentNodeID, "a completed failed qualification must route its owner result for repair")
	x.CurrentNodeID = "qualify"
	x.Version++
	ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x})
	require.NoError(t, err)
	require.True(t, ok)
	x = mustAdvance(t, e, x.ID)
	require.Equal(t, domain.WorkflowExecutionFailed, x.Status)
	require.Equal(t, "qualification_consumed", x.TerminalReason.Code)
	require.Equal(t, 1, q.effects)
}

func TestQualificationRejectsInvalidAcceptanceBeforeEffects(t *testing.T) {
	for _, kind := range []string{"no review", "same author", "missing snapshot", "different reviewer", "malformed digest", "missing verdict digest"} {
		t.Run(kind, func(t *testing.T) {
			e, store, x, q := qualificationEngine(t)
			switch kind {
			case "no review":
				attempts, err := store.ListAttempts(t.Context(), x.ID)
				require.NoError(t, err)
				attempts[0].Status = domain.WorkflowAttemptFailed
				x.Version++
				ok, err := store.Commit(t.Context(), repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: attempts[0]})
				require.NoError(t, err)
				require.True(t, ok)
			case "same author":
				q.candidate.SourceRunID = q.candidate.ReviewerRunID
			case "missing snapshot":
				q.candidate.ReviewRequestID = uuid.Nil
			case "different reviewer":
				q.candidate.ReviewerRunID = uuid.New()
			case "malformed digest":
				q.candidate.SHA256 = "anything"
			case "missing verdict digest":
				q.candidate.VerdictSHA256 = ""
			}
			x = mustAdvance(t, e, x.ID)
			require.Equal(t, domain.WorkflowExecutionFailed, x.Status)
			require.Zero(t, q.effects)
		})
	}
}

func TestQualificationUnavailableOwnerRefusesBeforeEffects(t *testing.T) {
	e, _, x, q := qualificationEngine(t)
	e.Qualifications = nil
	x = mustAdvance(t, e, x.ID)
	require.Equal(t, domain.WorkflowExecutionFailed, x.Status)
	require.Equal(t, "qualification_unavailable", x.TerminalReason.Code)
	require.Zero(t, q.prepares)
	require.Zero(t, q.effects)
}

func TestQualificationPreparationRejectionIsPersistedAndDoesNotRetry(t *testing.T) {
	e, store, x, q := qualificationEngine(t)
	q.prepareErr = NewQualificationPreparationError("qualification_review_rejected", "independent review did not accept this candidate")

	got := mustAdvance(t, e, x.ID)
	require.Equal(t, domain.WorkflowExecutionFailed, got.Status)
	require.Equal(t, "qualification_review_rejected", got.TerminalReason.Code)
	require.True(t, got.BudgetUsage.AccountingComplete, "pre-effect rejection must retain settled child accounting")
	require.Equal(t, 1, q.prepares)
	require.Zero(t, q.effects)

	persisted, err := store.Get(t.Context(), x.ID)
	require.NoError(t, err)
	require.Equal(t, domain.WorkflowExecutionFailed, persisted.Status)
	_, err = e.Advance(t.Context(), x.ID)
	require.NoError(t, err)
	require.Equal(t, 1, q.prepares, "terminal rejection must not be retried")
}

func TestQualificationPreparationTransientErrorRemainsRetryable(t *testing.T) {
	e, store, x, q := qualificationEngine(t)
	q.prepareErr = errors.New("qualification owner unavailable")
	_, err := e.Advance(t.Context(), x.ID)
	require.ErrorContains(t, err, "qualification owner unavailable")
	persisted, getErr := store.Get(t.Context(), x.ID)
	require.NoError(t, getErr)
	require.Equal(t, domain.WorkflowExecutionRunning, persisted.Status)

	q.prepareErr = nil
	got := mustAdvance(t, e, x.ID)
	require.Equal(t, "qualify", got.CurrentNodeID, "a retry first persists the qualification intent")
	got = mustAdvance(t, e, x.ID)
	got = mustAdvance(t, e, x.ID)
	require.Equal(t, "end", got.CurrentNodeID)
	require.Equal(t, 2, q.prepares, "a transient preparation error must be retried")
}

func TestQualificationCancellationReconcilesOriginalAdmissionAfterRestart(t *testing.T) {
	for _, submitted := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-dispatch", true: "lost-dispatch-response"}[submitted], func(t *testing.T) {
			e, store, x, q := qualificationEngine(t)
			mustAdvance(t, e, x.ID) // retained consumption; no effects yet
			if submitted {
				q.lostResponse, q.draining = true, true
				_, err := e.Advance(t.Context(), x.ID)
				require.ErrorContains(t, err, "lost admission")
			}
			_, _, err := e.Cancel(t.Context(), x.ID, "cancel", "operator", 0)
			require.NoError(t, err)
			restarted := *e
			if submitted {
				_, err = restarted.RecordCleanupDisposition(t.Context(), x.ID, 0, 0, nil)
				require.ErrorContains(t, err, "terminal accounting")
				pending, err := store.Get(t.Context(), x.ID)
				require.NoError(t, err)
				require.Equal(t, domain.WorkflowExecutionCancelling, pending.Status)
				require.False(t, pending.BudgetUsage.AccountingComplete)
				q.draining = false
				q.unknownUsage = true
				_, err = restarted.RecordCleanupDisposition(t.Context(), x.ID, 0, 0, nil)
				require.ErrorContains(t, err, "terminal accounting", "terminal status alone cannot fabricate accounting")
				q.unknownUsage = false
			}
			settled, err := restarted.RecordCleanupDisposition(t.Context(), x.ID, 0, 0, nil)
			require.NoError(t, err)
			require.Equal(t, domain.WorkflowExecutionCancelled, settled.Status)
			require.True(t, settled.BudgetUsage.AccountingComplete)
			require.Equal(t, 7, settled.BudgetUsage.Tokens)
			require.Equal(t, 2, settled.BudgetUsage.Children)
			again, err := restarted.RecordCleanupDisposition(t.Context(), x.ID, 0, 0, nil)
			require.NoError(t, err)
			require.Equal(t, settled.Version, again.Version)
			require.Equal(t, settled.BudgetUsage, again.BudgetUsage)
			if submitted {
				require.Equal(t, 1, q.effects)
			} else {
				require.Zero(t, q.effects)
			}
			require.Zero(t, q.waits, "cleanup must not drive or redispatch normal advancement")
		})
	}
}
