// Package evidence owns append-only, replayable spend-attempt records.
package evidence

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"treasury/internal/authorization"
)

type Record struct {
	ID                 string
	AuthorizationID    string
	MandateID          string
	AgentSubject       string
	Verdict            string
	ViolatedConstraint string
	Detail             string
	CreatedAt          string
}

// Attempt is the immutable terminal record for one proposed spend. JSON
// documents are retained verbatim so replay never depends on later mutable
// projections.
type Attempt struct {
	ID, AuthorizationID, MandateID, ApprovalID, SettlementID, InstrumentID string
	AgentSubject, Outcome, Basis                                           string
	RequestJSON, RailResponseJSON, ReceiptJSON                             string
	RecordedAt, RetainUntil                                                time.Time
}

type RequestSnapshot struct {
	AuthorizationID string `json:"authorization_id"`
	MandateID       string `json:"mandate_id"`
	IdempotencyKey  string `json:"idempotency_key"`
	AgentSubject    string `json:"agent_subject"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	Counterparty    string `json:"counterparty"`
}

type Appender interface {
	Append(context.Context, Record) error
	AppendAttempt(context.Context, Attempt) error
	Replay(context.Context, string) (Attempt, error)
}

type Recorder struct{ appender Appender }

func NewRecorder(appender Appender) *Recorder { return &Recorder{appender: appender} }

func (r *Recorder) RecordDecision(ctx context.Context, value authorization.DecisionEvidence) error {
	if r == nil || r.appender == nil {
		return fmt.Errorf("evidence appender is required")
	}
	if strings.TrimSpace(value.ID) == "" || strings.TrimSpace(value.AuthorizationID) == "" {
		return fmt.Errorf("evidence id and authorization id are required")
	}
	createdAt := value.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Unix(0, 0).UTC()
	}
	if err := r.appender.Append(ctx, Record{ID: value.ID, AuthorizationID: value.AuthorizationID, MandateID: value.MandateID, AgentSubject: value.AgentSubject, Verdict: string(value.Verdict), ViolatedConstraint: value.ViolatedConstraint, Detail: value.Detail, CreatedAt: createdAt.Format(time.RFC3339Nano)}); err != nil {
		return err
	}
	if value.Verdict != authorization.VerdictRefused {
		return nil
	}
	request, err := json.Marshal(RequestSnapshot{AuthorizationID: value.AuthorizationID, MandateID: value.MandateID, IdempotencyKey: value.IdempotencyKey, AgentSubject: value.AgentSubject, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty})
	if err != nil {
		return fmt.Errorf("encode refused attempt: %w", err)
	}
	receipt, err := json.Marshal(map[string]string{"constraint": value.ViolatedConstraint, "detail": value.Detail})
	if err != nil {
		return fmt.Errorf("encode refusal receipt: %w", err)
	}
	return r.appender.AppendAttempt(ctx, Attempt{ID: value.AuthorizationID + ":attempt", AuthorizationID: value.AuthorizationID, MandateID: value.MandateID, AgentSubject: value.AgentSubject, Outcome: "refused", Basis: "policy_evaluation", RequestJSON: string(request), RailResponseJSON: `{}`, ReceiptJSON: string(receipt), RecordedAt: createdAt, RetainUntil: createdAt.Add(180 * 24 * time.Hour)})
}

func (r *Recorder) RecordApprovalTerminal(ctx context.Context, approvalID string, value authorization.Record, outcome, resolver string, recordedAt time.Time) error {
	request, err := json.Marshal(RequestSnapshot{AuthorizationID: value.ID, MandateID: value.MandateID, IdempotencyKey: value.IdempotencyKey, AgentSubject: value.RequestingAgent, AmountMinor: value.AmountMinor, Currency: value.Currency, Counterparty: value.Counterparty})
	if err != nil {
		return err
	}
	receipt, err := json.Marshal(map[string]string{"approval_id": approvalID, "resolver": resolver, "outcome": outcome})
	if err != nil {
		return err
	}
	return r.appender.AppendAttempt(ctx, Attempt{ID: value.ID + ":attempt", AuthorizationID: value.ID, MandateID: value.MandateID, ApprovalID: approvalID, AgentSubject: value.RequestingAgent, Outcome: outcome, Basis: "human_approval", RequestJSON: string(request), RailResponseJSON: `{}`, ReceiptJSON: string(receipt), RecordedAt: recordedAt.UTC(), RetainUntil: recordedAt.UTC().Add(180 * 24 * time.Hour)})
}

func (r *Recorder) Replay(ctx context.Context, authorizationID string) (Attempt, error) {
	if r == nil || r.appender == nil {
		return Attempt{}, fmt.Errorf("evidence appender is required")
	}
	return r.appender.Replay(ctx, strings.TrimSpace(authorizationID))
}

var _ authorization.EvidenceRecorder = (*Recorder)(nil)
