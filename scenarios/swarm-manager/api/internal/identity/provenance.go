// Package identity contains Swarm Manager's session-enrichment middleware.
// Token verification and request provenance are owned by api-core/provenance.
package identity

import (
	"context"

	"github.com/vrooli/api-core/provenance"
)

const (
	TypeOperator = provenance.ActorOperator
	TypeAgent    = provenance.ActorAgent

	VerificationVerified = provenance.VerificationVerified
)

type Provenance = provenance.Provenance

func NewContext(ctx context.Context, p Provenance) context.Context {
	return provenance.NewContext(ctx, p)
}

func FromContext(ctx context.Context) Provenance { return provenance.FromContext(ctx) }

// VerifiedOperatorActor returns the operator attribution only when request
// provenance proves an authenticated operator. An absent, invalid or unavailable
// verification is not an operator and yields an empty string, so callers that
// need a real human act cannot inherit the fail-open started-by default.
func VerifiedOperatorActor(p Provenance) string {
	if p.Actor == TypeOperator && p.VerificationStatus == VerificationVerified {
		return p.FormatStartedBy()
	}
	return ""
}
