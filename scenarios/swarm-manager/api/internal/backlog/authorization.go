package backlog

import (
	"context"
	"net/http"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"

	"swarm-manager/internal/apierr"
)

func (h *Handler) authorizeBookkeeping(ctx context.Context, kind BacklogKind, name string) *apierr.DomainError {
	p, err := authn.RequireOperation(ctx, "swarm-manager:write", "swarm.bookkeeping", "swarm-manager:backlog/"+string(kind)+"/"+name)
	if err != nil {
		return &apierr.DomainError{Status: http.StatusForbidden, Message: err.Error()}
	}
	if p.Kind == identity.ActorAgent && h.activityChecker == nil && h.executionActivity == nil {
		return &apierr.DomainError{Status: http.StatusForbidden, Message: "authoritative work activity is unavailable"}
	}
	if p.Kind == identity.ActorAgent && ((h.activityChecker != nil && h.activityChecker.HasActiveAgent(ctx, string(kind), name)) || (h.executionActivity != nil && h.executionActivity.HasActiveForBacklog(ctx, string(kind), name))) {
		return &apierr.DomainError{Status: http.StatusForbidden, Message: "active work cannot be accepted or archived by a bookkeeping agent"}
	}
	return nil
}

func (h *Handler) requireBookkeeping(w http.ResponseWriter, r *http.Request, kind BacklogKind, name string) bool {
	if err := h.authorizeBookkeeping(r.Context(), kind, name); err != nil {
		apierr.MapError(w, "[backlog] authorization", err)
		return false
	}
	return true
}
