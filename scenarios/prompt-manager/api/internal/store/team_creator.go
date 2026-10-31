package store

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// teamConfigMu serializes team and heartbeat configuration writes.
var teamConfigMu sync.Mutex

// TeamCreationProof records a verified actor, independently of authored team
// members, labels and harness headers. Legacy/unverified creation has no proof.
type TeamCreationProof struct {
	Kind    identity.ActorKind  `json:"kind"`
	Subject string              `json:"subject"`
	Owner   string              `json:"owner"`
	Source  identity.AuthSource `json:"source"`
}

func verifiedTeamCreator(ctx context.Context) *TeamCreationProof {
	p, ok := identity.PrincipalFromContext(ctx)
	if !ok || !p.IsVerified() || (!p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt)) {
		return nil
	}
	return &TeamCreationProof{Kind: p.Kind, Subject: p.Subject, Owner: p.OwnerSubject(), Source: p.Source}
}

// TeamReader is the authoritative team lookup used by domain authorization.
type TeamReader interface {
	Get(context.Context, string) (*Team, error)
}

// requireTeamIdentity keeps an offered but invalid credential from becoming an
// unattributed legacy write. Authentication status remains separate from grants.
func requireTeamIdentity(ctx context.Context) error {
	observed := provenance.FromContext(ctx)
	switch observed.VerificationStatus {
	case provenance.VerificationInvalid:
		return identity.NewFailure(identity.FailureInvalid, identity.SourceAgentProvenance)
	case provenance.VerificationUnavailable:
		return identity.NewFailure(identity.FailureUnavailable, identity.SourceAgentProvenance)
	}
	if status, ok := identity.StatusFromContext(ctx); ok && status.FailureClass != "" && status.FailureClass != identity.FailureMissing {
		return identity.NewFailure(status.FailureClass, status.Source)
	}
	p, present := identity.PrincipalFromContext(ctx)
	if (observed.IsVerifiedAgent() && !present) || (p.Kind == identity.ActorAgent && !present) {
		return identity.NewFailure(identity.FailureInvalid, identity.SourceAgentProvenance)
	}
	if present && p.IsVerified() && !p.ExpiresAt.IsZero() && !time.Now().Before(p.ExpiresAt) {
		return identity.NewFailure(identity.FailureExpired, p.Source)
	}
	return nil
}

// AuthorizeTeamManagement applies creator ownership and the canonical exact
// operation evaluator at every team mutation owner. It preserves uncredentialed
// legacy maintenance, but an agent must be the verified creator or have coverage.
func AuthorizeTeamManagement(ctx context.Context, teams TeamReader, id string) error {
	if err := requireTeamIdentity(ctx); err != nil {
		return err
	}
	p, present := identity.PrincipalFromContext(ctx)
	_, bounded := authn.RequestOperationAuthorityFromContext(ctx)
	if p.Kind == identity.ActorAgent && teams != nil {
		if creator := verifiedTeamCreator(ctx); creator != nil {
			team, err := teams.Get(ctx, id)
			if err == nil && team != nil && team.CreationProof != nil && *team.CreationProof == *creator {
				return nil
			}
		}
	}
	if bounded || (present && (p.Kind == identity.ActorAgent || p.Grant != nil)) {
		_, err := authn.RequireOperation(ctx, "prompt-manager:write", "team.manage", "prompt-manager:team/"+id)
		return err
	}
	return nil
}

// CanManageTeam is a caller-specific read result, never a bearer credential.
func CanManageTeam(ctx context.Context, teams TeamReader, id string) bool {
	return verifiedTeamCreator(ctx) != nil && AuthorizeTeamManagement(ctx, teams, id) == nil
}
