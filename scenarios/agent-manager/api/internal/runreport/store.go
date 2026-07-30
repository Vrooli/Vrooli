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

// EpisodeStore retains the deterministic friction projection beside its source
// facts. Replacing one run is atomic and idempotent.
type EpisodeStore interface {
	ReplaceEpisodes(context.Context, uuid.UUID, []FrictionEpisode) error
	Episodes(context.Context, uuid.UUID) ([]FrictionEpisode, error)
}

type SelfReportStore interface {
	ReplaceSelfReportSpans(context.Context, uuid.UUID, []SelfReportSpan) error
	SelfReportSpans(context.Context, uuid.UUID) ([]SelfReportSpan, error)
}

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
	ClassifierVersion string           `json:"classifierVersion"`
	Availability      Availability     `json:"availability"`
	Facts             []InvocationFact `json:"facts"`
}
