package journal

import (
	"context"
	"strings"

	"github.com/vrooli/api-core/provenance"
)

const (
	ActorKindOperator = provenance.ActorOperator
	ActorKindAgent    = provenance.ActorAgent

	VerificationVerified    = provenance.VerificationVerified
	VerificationUnavailable = provenance.VerificationUnavailable
	VerificationInvalid     = provenance.VerificationInvalid
	VerificationAbsent      = provenance.VerificationAbsent
)

// Actor is the bounded provenance written beside an immutable journal event.
// Identity verification itself remains owned by api-core's CLIUtilVerifier,
// which delegates to packages/cli-core/cliutil.IdentityEnv.VerifyIdentity.
type Actor struct {
	Identity           string
	Kind               string
	VerificationStatus string
	RunID              string
}

// ProvenanceResolver is the single attribution seam for journal writes.
type ProvenanceResolver interface {
	Resolve(context.Context, string) Actor
}

type ContextProvenanceResolver struct{}

func (ContextProvenanceResolver) Resolve(ctx context.Context, fallbackIdentity string) Actor {
	p := provenance.FromContext(ctx)
	status := strings.TrimSpace(p.VerificationStatus)
	if status == "" {
		status = VerificationAbsent
	}
	actor := Actor{
		Identity:           strings.TrimSpace(fallbackIdentity),
		Kind:               ActorKindOperator,
		VerificationStatus: status,
	}
	if p.IsVerifiedAgent() {
		actor.Kind = ActorKindAgent
		actor.Identity = strings.TrimSpace(p.Subject)
		if actor.Identity == "" {
			actor.Identity = strings.TrimSpace(p.ProfileKey)
		}
		actor.RunID = strings.TrimSpace(p.RunID)
	} else if subject := strings.TrimSpace(p.Subject); subject != "" {
		actor.Identity = subject
	}
	if actor.Identity == "" {
		actor.Identity = ActorKindOperator
	}
	return actor
}

func stampEventAttribution(ctx context.Context, event Event, resolver ProvenanceResolver) Event {
	actor := resolver.Resolve(ctx, event.ActorIdentity)
	event.ActorIdentity = actor.Identity
	event.ActorKind = actor.Kind
	event.ActorVerificationStatus = actor.VerificationStatus
	event.ActorRunID = actor.RunID
	return event
}
