package runreport

import (
	"context"

	"agent-manager/internal/runsignal"

	"github.com/google/uuid"
)

// ReceiptJoinStore persists only receipt identifiers proven by the Vrooli
// Events verifier and matching Agent Manager run correlation.
type ReceiptJoinStore interface {
	ReplaceReceiptEvidence(context.Context, uuid.UUID, string, []string) error
	ReceiptEvidence(context.Context, uuid.UUID) ([]string, error)
}

type LedgerStore interface {
	ReplaceCrossScenarioCalls(context.Context, uuid.UUID, string, []CrossScenarioCall) error
	CrossScenarioCalls(context.Context, uuid.UUID) ([]CrossScenarioCall, error)
}

type InvocationFactsResponse struct {
	ClassifierVersion string                     `json:"classifierVersion"`
	Availability      Availability               `json:"availability"`
	Facts             []runsignal.InvocationFact `json:"facts"`
}
