package workflowruntime

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// QualificationOwner is the deterministic owner adapter, not an agent role.
// Prepare is read-only and verifies the retained review's lineage and verdict.
// Start must be idempotent over the entire retained request, including failures.
type QualificationOwner interface {
	Prepare(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (QualificationCandidate, error)
	Start(context.Context, QualificationRequest) (QualificationState, error)
	Wait(context.Context, QualificationRequest, string) (QualificationState, error)
	Observe(context.Context, QualificationRequest) (QualificationState, error)
	CloseAdmission(context.Context, QualificationRequest) (QualificationState, error)
}

// QualificationPreparationError is returned when the retained review is
// durably known to be ineligible for qualification.  The workflow engine
// persists this as a terminal failure instead of retrying the same immutable
// candidate forever.  Owner/transport errors remain ordinary errors so the
// exact qualification attempt can be retried after recovery.
type QualificationPreparationError struct {
	Code    string
	Message string
}

func (e *QualificationPreparationError) Error() string {
	if e == nil {
		return "qualification preparation rejected"
	}
	return e.Message
}

func NewQualificationPreparationError(code, message string) error {
	return &QualificationPreparationError{Code: code, Message: message}
}

func qualificationPreparationError(err error) (*QualificationPreparationError, bool) {
	var target *QualificationPreparationError
	if !errors.As(err, &target) || target == nil || target.Code == "" || target.Message == "" {
		return nil, false
	}
	return target, true
}

type QualificationCandidate struct {
	ReviewAttemptID uuid.UUID `json:"reviewAttemptId"`
	VerdictSHA256   string    `json:"verdictSha256"`
	ReviewerRunID   uuid.UUID `json:"reviewerRunId"`
	SourceRunID     uuid.UUID `json:"sourceRunId"`
	SandboxID       uuid.UUID `json:"sandboxId"`
	ReviewRequestID uuid.UUID `json:"reviewRequestId"`
	SHA256          string    `json:"sha256"`
}

type QualificationRequest struct {
	Grants            []string               `json:"grants,omitempty"`
	ExecutionID       uuid.UUID              `json:"executionId"`
	Candidate         QualificationCandidate `json:"candidate"`
	ProgramName       string                 `json:"programName"`
	ProgramDigest     string                 `json:"programDigest"`
	Inputs            json.RawMessage        `json:"inputs"`
	IdempotencyKey    string                 `json:"idempotencyKey"`
	AdmissionDeadline time.Time              `json:"admissionDeadline"`
}

type QualificationState struct {
	ProgramID   string                     `json:"programId"`
	Terminal    bool                       `json:"terminal"`
	Passed      bool                       `json:"passed"`
	Output      json.RawMessage            `json:"output,omitempty"`
	BudgetUsage domain.WorkflowBudgetUsage `json:"budgetUsage"`
}

func (e *Engine) advanceQualification(ctx context.Context, x *domain.WorkflowExecution, r *domain.WorkflowRevision, node *domain.WorkflowNode) (*domain.WorkflowExecution, error) {
	if node.Qualification == nil || e.Qualifications == nil {
		return e.fail(ctx, x, "qualification_unavailable", "deterministic qualification owner is unavailable")
	}
	attempts, err := e.Store.ListAttempts(ctx, x.ID)
	if err != nil {
		return nil, err
	}
	journal, err := e.Store.ListJournal(ctx, x.ID, 0, 0)
	if err != nil {
		return nil, err
	}
	var active, review *domain.WorkflowNodeAttempt
	ordinal := 1
	for _, a := range attempts {
		if a.NodeID == node.ID {
			if a.Ordinal >= ordinal {
				ordinal = a.Ordinal + 1
			}
			if a.Status != domain.WorkflowAttemptCompleted && a.Status != domain.WorkflowAttemptFailed {
				active = a
			}
		}
		if a.NodeID == node.Qualification.ReviewFromNode && (review == nil || a.Ordinal > review.Ordinal) {
			review = a
		}
	}
	if active == nil {
		if x.EngagementGrant != nil && !x.BudgetUsage.AccountingComplete {
			return e.fail(ctx, x, "accounting_unknown", "qualification requires settled prior child usage")
		}
		if x.BudgetUsage.NodeAttempts >= r.Definition.Budgets.MaxNodeAttempts {
			return e.exhaust(ctx, x, "node_attempts")
		}
		if x.BudgetUsage.Children >= r.Definition.Budgets.MaxChildren {
			return e.exhaust(ctx, x, "children")
		}
		if len(node.Qualification.Grants) > 0 && x.EngagementGrant != nil && !slices.Contains(x.EngagementGrant.AllowedEffects, "destructive") {
			return e.fail(ctx, x, "qualification_effect_denied", "engagement grant does not permit the declared destructive binding")
		}
		if review == nil || review.Status != domain.WorkflowAttemptCompleted || review.RunID == nil {
			return e.fail(ctx, x, "qualification_review_missing", "qualification requires its completed independent review attempt")
		}
		candidate, err := e.Qualifications.Prepare(ctx, x.ID, review.ID, *review.RunID)
		if err != nil {
			if rejected, ok := qualificationPreparationError(err); ok {
				// The review and every prior child are already terminal and
				// accounted for. Do not manufacture an accounting-unknown
				// cleanup obligation for a deterministic pre-effect refusal.
				return e.commitFailure(ctx, x, nil, nil, rejected.Code, rejected.Message)
			}
			return x, err
		}
		digest, digestErr := hex.DecodeString(candidate.SHA256)
		verdict, verdictErr := hex.DecodeString(candidate.VerdictSHA256)
		if digestErr != nil || len(digest) != 32 || hex.EncodeToString(digest) != candidate.SHA256 || verdictErr != nil || len(verdict) != 32 || hex.EncodeToString(verdict) != candidate.VerdictSHA256 || candidate.ReviewAttemptID != review.ID || candidate.ReviewerRunID != *review.RunID || candidate.SourceRunID == uuid.Nil || candidate.SourceRunID == candidate.ReviewerRunID || candidate.SandboxID == uuid.Nil || candidate.ReviewRequestID == uuid.Nil {
			return e.fail(ctx, x, "qualification_review_invalid", "review owner returned invalid candidate identity")
		}
		// One consumption per candidate in this boundary, including new review
		// attempts, different qualification nodes and workflow retry/resume.
		key := uuid.NewSHA1(x.ID, []byte("qualification:"+candidate.SHA256)).String()
		for _, a := range attempts {
			if a.Strategy == domain.WorkflowAttemptQualification && a.IdempotencyKey == key {
				return e.fail(ctx, x, "qualification_consumed", "candidate qualification was already consumed; do not mint another attempt")
			}
		}
		inputs, err := EvaluateBindings(node.Qualification.Bindings, BindingContext{Input: x.Input, Journal: journal, ExecutionID: x.ID.String()})
		if err != nil {
			return e.fail(ctx, x, "qualification_input_invalid", err.Error())
		}
		if _, exists := inputs["candidate"]; exists {
			return e.fail(ctx, x, "qualification_input_invalid", "candidate is reserved for retained owner evidence")
		}
		encoded, err := json.Marshal(inputs)
		if err != nil {
			return nil, err
		}
		if len(encoded) > 65536 {
			return e.fail(ctx, x, "qualification_input_invalid", "qualification inputs exceed 64 KiB")
		}
		now := e.now()
		request := QualificationRequest{ExecutionID: x.ID, Candidate: candidate, ProgramName: node.Qualification.ProgramName, ProgramDigest: node.Qualification.ProgramDigest, Inputs: encoded, IdempotencyKey: key, AdmissionDeadline: now.Add(time.Hour)}
		request.Grants = slices.Clone(node.Qualification.Grants)
		snapshot, err := json.Marshal(request)
		if err != nil {
			return nil, err
		}
		active = &domain.WorkflowNodeAttempt{ID: uuid.New(), ExecutionID: x.ID, NodeID: node.ID, Ordinal: ordinal, Strategy: domain.WorkflowAttemptQualification, Status: domain.WorkflowAttemptDispatchPending, IdempotencyKey: key, InputSnapshot: snapshot, SourceAttemptID: &review.ID, Version: 1, CreatedAt: now, UpdatedAt: now}
		x.BudgetUsage.NodeAttempts++
		x.Version++
		x.UpdatedAt = now
		entry := nextJournal(x.ID, journal, domain.WorkflowJournalAttempt, node.ID, &active.ID, snapshot, now)
		if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil {
			return nil, err
		} else if !ok {
			return nil, ErrConcurrentAdvance
		}
		return x, nil
	}
	var request QualificationRequest
	if err := json.Unmarshal(active.InputSnapshot, &request); err != nil || request.ExecutionID != x.ID || request.IdempotencyKey != active.IdempotencyKey || request.ProgramName != node.Qualification.ProgramName || request.ProgramDigest != node.Qualification.ProgramDigest || !slices.Equal(request.Grants, node.Qualification.Grants) {
		return e.fail(ctx, x, "qualification_intent_invalid", "persisted qualification does not match the immutable revision")
	}
	var state QualificationState
	if active.Status == domain.WorkflowAttemptDispatchPending {
		state, err = e.Qualifications.Start(ctx, request)
	} else {
		var accepted QualificationState
		for _, entry := range journal {
			if entry.Kind == domain.WorkflowJournalQualification && entry.AttemptID != nil && *entry.AttemptID == active.ID {
				if decodeErr := json.Unmarshal(entry.Payload, &accepted); decodeErr != nil {
					return nil, decodeErr
				}
			}
		}
		if accepted.ProgramID == "" {
			return e.fail(ctx, x, "qualification_identity_missing", "qualification has no retained program handle")
		}
		state, err = e.Qualifications.Wait(ctx, request, accepted.ProgramID)
		if err == nil && state.ProgramID != accepted.ProgramID {
			return e.fail(ctx, x, "qualification_identity_changed", "qualification owner changed the retained program identity")
		}
	}
	if err != nil {
		return x, err
	} // Unknown acceptance retains this exact attempt.
	if state.ProgramID == "" || len(state.Output) > 65536 || (len(state.Output) > 0 && !json.Valid(state.Output)) || (!state.Terminal && state.Passed) {
		return x, fmt.Errorf("qualification owner returned invalid execution evidence")
	}
	if !state.Terminal && active.Status == domain.WorkflowAttemptDispatched {
		return x, nil
	}
	now := e.now()
	if active.Status == domain.WorkflowAttemptDispatchPending {
		x.BudgetUsage.Children++
	}
	active.Status, active.UpdatedAt = domain.WorkflowAttemptDispatched, now
	active.Version++
	payload, _ := json.Marshal(state)
	entry := nextJournal(x.ID, journal, domain.WorkflowJournalQualification, node.ID, &active.ID, payload, now)
	if state.Terminal {
		active.Status, active.CompletedAt = domain.WorkflowAttemptCompleted, &now
		if _, _, err := e.rebuildOrdinaryUsage(ctx, x, journal, attempts, false, false); err != nil {
			return x, err
		}
		next, err := selectUnconditionalEdge(r.Definition.Edges, node.ID)
		if err != nil {
			return e.fail(ctx, x, "edge_invalid", err.Error())
		}
		if err := takeEdge(x, next, r.Definition.Budgets); err != nil {
			return e.exhaust(ctx, x, "edge_traversal")
		}
		x.CurrentNodeID = next.To
	} else {
		x.BudgetUsage.AccountingComplete = false
	}
	x.Status, x.UpdatedAt = domain.WorkflowExecutionRunning, now
	x.Version++
	if ok, err := e.Store.Commit(ctx, repository.WorkflowCommit{ExpectedVersion: x.Version - 1, Execution: x, Attempt: active, Journal: []*domain.WorkflowJournalEntry{entry}}); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrConcurrentAdvance
	}
	return x, nil
}
