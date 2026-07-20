package initiatives

import (
	"net/http"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	"github.com/gorilla/mux"
)

// Recreate runs the service-owned archive-and-clone lifecycle action.
func (h *Handler) Recreate(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(mux.Vars(r)["name"])
	if name == "" {
		apierr.MapError(w, "[initiatives] recreate", apierr.BadRequest("name is required"))
		return
	}
	if err := h.service.RecreateInitiative(r.Context(), name); err != nil {
		if strings.Contains(err.Error(), "not found") {
			apierr.MapError(w, "[initiatives] recreate", apierr.NotFound("initiative not found"))
		} else if strings.Contains(err.Error(), "agent is currently working") {
			apierr.MapError(w, "[initiatives] recreate", apierr.Conflict("%s", err.Error()))
		} else {
			apierr.MapError(w, "[initiatives] recreate", apierr.BadRequest("%s", err.Error()))
		}
		return
	}
	cloneName := name + "-recreated"
	// The service resolves a suffix when needed; find the actual successor by
	// lineage rather than assuming the first candidate was available.
	for _, candidate := range mustListInitiatives(h.service) {
		if candidate.SpawnedFrom == name {
			cloneName = candidate.Name
		}
	}
	result, err := h.service.Get(cloneName)
	if err != nil {
		apierr.MapError(w, "[initiatives] recreate", apierr.Internal("recreated initiative could not be loaded"))
		return
	}
	if err := httputil.JSON(w, result); err != nil {
		apierr.MapError(w, "[initiatives] recreate", apierr.Internal("failed to encode response"))
	}
}

func mustListInitiatives(service *Service) []Initiative {
	items, err := service.store.LoadAll()
	if err != nil {
		return nil
	}
	return items
}
