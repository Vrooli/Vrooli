package attestation

import "context"

// Repository is the append-only manual attestation ledger.
type Repository interface {
	AppendAttestation(ctx context.Context, attestation Attestation) error
	ListAttestations(ctx context.Context, filter Filter) ([]Attestation, error)
}

// Attestation is human evidence for a manual-tier claim.
type Attestation struct {
	ID        string
	Scenario  string
	PageID    string
	ClaimID   string
	Author    string
	Rationale string
	ExpiresAt string
	CreatedAt string
}

// Filter narrows attestation reads.
type Filter struct {
	Scenario string
	PageID   string
	ClaimID  string
}
