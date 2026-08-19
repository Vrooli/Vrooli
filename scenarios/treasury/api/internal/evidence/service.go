// Package evidence owns append-only, replayable spend-attempt records.
package evidence

import (
	"context"
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

type Appender interface {
	Append(context.Context, Record) error
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
	return r.appender.Append(ctx, Record{ID: value.ID, AuthorizationID: value.AuthorizationID, MandateID: value.MandateID, AgentSubject: value.AgentSubject, Verdict: string(value.Verdict), ViolatedConstraint: value.ViolatedConstraint, Detail: value.Detail, CreatedAt: createdAt.Format(time.RFC3339Nano)})
}

var _ authorization.EvidenceRecorder = (*Recorder)(nil)
