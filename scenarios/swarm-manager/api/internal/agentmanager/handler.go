package agentmanager

import (
	"net/http"

	"github.com/gorilla/mux"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	"swarm-manager/internal/httputil"
)

// Handler exposes agent-manager status endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new status handler.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers agent-manager endpoints.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/agent-manager/status", h.Status).Methods("GET")
}

// Status returns agent-manager availability.
func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	response := &apipb.AgentManagerStatusResponse{}
	if h.service != nil {
		response.Enabled = h.service.IsEnabled()
		if response.Enabled {
			response.Available = h.service.IsAvailable(r.Context())
			if url, err := h.service.ResolveURL(r.Context()); err == nil && url != "" {
				response.Url = &url
			}
		}
		if profileID := h.service.GetProfileID(); profileID != "" {
			response.ProfileId = &profileID
		}
	}

	if err := httputil.ProtoJSON(w, response); err != nil {
		httputil.InternalError(w, "[agent-manager] status", "failed to encode response")
	}
}
