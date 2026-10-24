package store

import (
	"context"
	"sync"
	"time"

	"github.com/vrooli/api-core/identity"
)

// finiteLeaderMu serializes heartbeat-config and team mutations so concurrent
// updates to the same member directory cannot interleave. The name is retained
// for continuity; it guards all heartbeat.json writes, not only leader state.
var finiteLeaderMu sync.Mutex

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

// callerIsVerifiedTeamCreator reports whether the current verified caller is the
// recorded creator of team id and that identity is still live (unexpired, per
// verifiedTeamCreator). The creator updating its own team needs no separate
// operation grant; any other actor does. Fail-closed: a missing/expired proof,
// an unknown team, or a legacy team with no creation proof returns false so the
// normal operation gate still applies. A different agent's subject never matches.
func (s *FileTeamStore) callerIsVerifiedTeamCreator(ctx context.Context, id string) bool {
	creator := verifiedTeamCreator(ctx)
	if creator == nil {
		return false
	}
	team, err := s.Get(ctx, id)
	if err != nil || team == nil || team.CreationProof == nil {
		return false
	}
	return *team.CreationProof == *creator
}
