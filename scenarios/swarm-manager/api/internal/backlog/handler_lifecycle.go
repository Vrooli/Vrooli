package backlog

import (
	"errors"
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"
)

// Recreate is the direct, operator-confirmed counterpart of recreate_item.
// It deliberately delegates every mutation to Service.RecreateItem.
func (h *Handler) Recreate(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "recreate")
	if !ok {
		return
	}
	if h.lifecycleService == nil {
		apierr.MapError(w, "[backlog] recreate", apierr.Unavailable("backlog lifecycle service is unavailable"))
		return
	}
	item, err := h.lifecycleService.RecreateItem(r.Context(), kind, name)
	if err != nil {
		h.mapLifecycleError(w, "recreate", err)
		return
	}
	h.invalidateAllGraphLenses()
	if err := httputil.JSON(w, item); err != nil {
		apierr.MapError(w, "[backlog] recreate", apierr.Internal("failed to encode response"))
	}
}

type resetArtifactsRequest struct {
	Scope []ResetArtifactScope `json:"scope"`
}

// ResetArtifacts removes explicitly selected derived artifact groups. An
// omitted or invalid scope is rejected before the service can mutate state.
func (h *Handler) ResetArtifacts(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "reset-artifacts")
	if !ok {
		return
	}
	if h.lifecycleService == nil {
		apierr.MapError(w, "[backlog] reset-artifacts", apierr.Unavailable("backlog lifecycle service is unavailable"))
		return
	}
	var req resetArtifactsRequest
	if err := httputil.DecodeJSONStrict(r, &req); err != nil {
		apierr.MapError(w, "[backlog] reset-artifacts", apierr.BadRequest("invalid request body"))
		return
	}
	result, err := h.lifecycleService.ResetArtifacts(r.Context(), kind, name, req.Scope)
	if err != nil {
		h.mapLifecycleError(w, "reset-artifacts", err)
		return
	}
	h.invalidateAllGraphLenses()
	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[backlog] reset-artifacts", apierr.Internal("failed to encode response"))
	}
}

func (h *Handler) mapLifecycleError(w http.ResponseWriter, operation string, err error) {
	if errors.Is(err, ErrNotFound) || strings.Contains(err.Error(), "not found") {
		apierr.MapError(w, "[backlog] "+operation, apierr.NotFound("backlog item not found"))
		return
	}
	if strings.Contains(err.Error(), "agent is currently working") {
		apierr.MapError(w, "[backlog] "+operation, apierr.Conflict("%s", err.Error()))
		return
	}
	apierr.MapError(w, "[backlog] "+operation, apierr.BadRequest("%s", err.Error()))
}
