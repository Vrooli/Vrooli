package reconcile

import "context"

// EvidenceRepository persists per-claim reconciliation evidence.
type EvidenceRepository interface {
	SaveEvidence(ctx context.Context, evidence Evidence) error
	ListEvidence(ctx context.Context, filter EvidenceFilter) ([]Evidence, error)
}

// Evidence records the AX evidence used for one claim verdict.
type Evidence struct {
	ID         string
	Scenario   string
	PageID     string
	Route      string
	StateID    string
	ClaimID    string
	ClaimType  string
	Verdict    string
	CaptureRef string
	AXNodeJSON string
	Message    string
	CheckedAt  string
}

// EvidenceFilter narrows evidence reads.
type EvidenceFilter struct {
	Scenario string
	PageID   string
	ClaimID  string
	Limit    int
}
