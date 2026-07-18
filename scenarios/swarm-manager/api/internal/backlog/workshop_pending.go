// Deferred workshop auto-advance cancel endpoint.
//
// The auto-advance timer is now a durable scheduler intent on the item's
// workflow (see ops_reroute.go); this endpoint cancels it. The legacy
// pending-advance file + in-memory ticker were removed with the reroute — the
// scheduler reloads intents from persisted workflow state, so nothing in-memory
// is load-bearing across a restart.
package backlog

import (
	"log/slog"
	"net/http"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/httputil"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

// WorkshopCancelPendingAdvance cancels a pending auto-advance for a backlog item
// by removing its durable scheduler intent.
func (h *Handler) WorkshopCancelPendingAdvance(w http.ResponseWriter, r *http.Request) {
	kind, name, ok := h.parseKindAndName(w, r, "workshop-cancel-pending")
	if !ok {
		return
	}

	// Auto-advance no longer creates a Swarm scheduler intent. A declared
	// workflow must model any future wait itself, so historical requests are a
	// harmless no-op.
	cancelled := false

	if cancelled {
		slog.Info("workshop pending advance cancelled", "kind", kind, "name", name)
	}

	resp := &apipb.WorkshopCancelPendingAdvanceResponse{Cancelled: cancelled}
	if err := httputil.ProtoJSON(w, resp); err != nil {
		apierr.MapError(w, "[backlog] workshop-cancel-pending", apierr.Internal("failed to encode response"))
	}
}
