package wiring

import (
	"context"
	"strings"

	"agent-manager/internal/handlers"
	"agent-manager/internal/orchestration"

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
