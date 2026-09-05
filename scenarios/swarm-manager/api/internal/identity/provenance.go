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
)

type Provenance = provenance.Provenance

func NewContext(ctx context.Context, p Provenance) context.Context {
	return provenance.NewContext(ctx, p)
}

func FromContext(ctx context.Context) Provenance { return provenance.FromContext(ctx) }
