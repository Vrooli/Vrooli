package runreport

import (
	"context"
	"github.com/google/uuid"
)

// InvocationFactStore persists immutable derivations by source run. Replace is
// idempotent: rebuilding from unchanged durable events yields the same rows.
type InvocationFactStore interface {
	ReplaceInvocationFacts(context.Context, uuid.UUID, []InvocationFact) error
	InvocationFacts(context.Context, uuid.UUID) ([]InvocationFact, error)
}

// ReceiptJoinStore persists only receipt identifiers proven by the Vrooli
// Events verifier and matching Agent Manager run correlation.
type ReceiptJoinStore interface {
	ReplaceReceiptEvidence(context.Context, uuid.UUID, string, []string) error
	ReceiptEvidence(context.Context, uuid.UUID) ([]string, error)
}

type InvocationFactsResponse struct {
	ClassifierVersion string           `json:"classifierVersion"`
	Availability      Availability     `json:"availability"`
	Facts             []InvocationFact `json:"facts"`
}
