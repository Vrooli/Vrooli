package wiring

import (
	"context"
	"strings"
	"time"

	"agent-manager/internal/handlers"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/supervision"

	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type watchActionAuthorizer struct {
	orchestrator interface {
		VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error)
	}
	owners authn.TokenVerifier
}

// Stable effort delegation uses only claims produced by the existing verifier.
// No family topology is fabricated and no profile/lineage text becomes authority.
func (a watchActionAuthorizer) AuthorizeEffortAction(ctx context.Context, token string, authority domainpb.WatchAuthority) (supervision.EffortActor, error) {
	if authority == domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR {
		if a.owners == nil || strings.TrimSpace(token) == "" {
			return supervision.EffortActor{}, handlers.ErrWatchActionUnauthenticated
		}
		principal, err := a.owners.Verify(ctx, token)
		if err != nil || !principal.IsHuman() || principal.ExpiresAt.IsZero() || !principal.ExpiresAt.After(time.Now()) {
			return supervision.EffortActor{}, handlers.ErrWatchActionUnauthenticated
		}
		if !hasSupervisionScope(principal.Scopes) {
			return supervision.EffortActor{}, handlers.ErrWatchActionForbidden
		}
		return supervision.EffortActor{ID: strings.TrimSpace(principal.Subject), OwnerSubject: strings.TrimSpace(principal.Subject), Scopes: append([]string{}, principal.Scopes...), Operator: true}, nil
	}
	if authority != domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT {
		r := &domainpb.RequestCohortWatchActionRequest{Authority: authority}
		err := a.AuthorizeWatchAction(ctx, token, r)
		return supervision.EffortActor{ID: r.RequestedBy, Operator: authority == domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}, err
	}
	if strings.TrimSpace(token) == "" || a.orchestrator == nil {
		return supervision.EffortActor{}, handlers.ErrWatchActionUnauthenticated
	}
	v, err := a.orchestrator.VerifyIdentityToken(ctx, token)
	if err != nil || v == nil || !v.Valid || v.Claims == nil {
		return supervision.EffortActor{}, handlers.ErrWatchActionUnauthenticated
	}
	return supervision.EffortActor{ID: v.Claims.RunID.String(), OwnerSubject: v.Claims.Subject, Scopes: append([]string(nil), v.Claims.Scopes...)}, nil
}

func (a watchActionAuthorizer) AuthorizeWatchAction(ctx context.Context, token string, request *domainpb.RequestCohortWatchActionRequest) error {
	if strings.TrimSpace(token) == "" {
		return handlers.ErrWatchActionUnauthenticated
	}
	switch request.GetAuthority() {
	case domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT:
		verified, err := a.orchestrator.VerifyIdentityToken(ctx, token)
		if err != nil || verified == nil || !verified.Valid || verified.Claims == nil {
			return handlers.ErrWatchActionUnauthenticated
		}
		request.RequestedBy = verified.Claims.RunID.String()
		return nil
	case domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR:
		if a.owners == nil {
			return handlers.ErrWatchActionUnauthenticated
		}
		principal, err := a.owners.Verify(ctx, token)
		if err != nil {
			return handlers.ErrWatchActionUnauthenticated
		}
		if principal.Kind != coreidentity.ActorHuman || !principal.Verified || !hasSupervisionScope(principal.Scopes) {
			return handlers.ErrWatchActionForbidden
		}
		request.RequestedBy = strings.TrimSpace(principal.Subject)
		return nil
	case domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM:
		return handlers.ErrWatchActionForbidden
	default:
		return handlers.ErrWatchActionForbidden
	}
}

func hasSupervisionScope(scopes []string) bool {
	for _, scope := range scopes {
		switch strings.TrimSpace(scope) {
		case "*", "agent-manager:write", "agent-manager:supervise":
			return true
		}
	}
	return false
}
