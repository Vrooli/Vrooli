package goals

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	"github.com/vrooli/api-core/authn"
)

func authorizeBookkeeping(ctx context.Context, name string) error {
	_, err := authn.RequireOperation(ctx, "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:goal/"+name)
	return err
}

func requireBookkeeping(w http.ResponseWriter, r *http.Request, name string) bool {
	if err := authorizeBookkeeping(r.Context(), name); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return false
	}
	return true
}

func requireGoalAcceptance(ctx context.Context) error {
	if _, err := authn.RequireIndependentHuman(ctx); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	if _, err := authn.RequireCapability(ctx, "swarm-manager:write"); err != nil {
		return connect.NewError(connect.CodePermissionDenied, err)
	}
	return nil
}

// The domain's existing change events retain graph history. This additional
// request receipt records the actual actor and independent grant approver.
func (s *Service) auditBookkeeping(ctx context.Context, name string) {
	if logger, ok := s.eventLogger.(interface{ EmitGoalBookkeepingChanged(context.Context, string) }); ok {
		logger.EmitGoalBookkeepingChanged(ctx, name)
	}
}
