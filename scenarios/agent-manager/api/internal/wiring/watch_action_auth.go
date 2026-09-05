package wiring

import (
	"context"
	"errors"
	"strings"

	"agent-manager/internal/handlers"
	"agent-manager/internal/orchestration"

	"github.com/vrooli/api-core/owneridentity"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type watchActionAuthorizer struct {
	orchestrator interface {
		VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error)
	}
	owners owneridentity.Validator
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
		identity, err := a.owners.Validate(ctx, token)
		if err != nil {
			if errors.Is(err, owneridentity.ErrUnauthenticated) {
				return handlers.ErrWatchActionUnauthenticated
			}
			return handlers.ErrWatchActionForbidden
		}
		if !hasSupervisionScope(identity.Scopes) {
			return handlers.ErrWatchActionForbidden
		}
		request.RequestedBy = strings.TrimSpace(identity.Subject)
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
