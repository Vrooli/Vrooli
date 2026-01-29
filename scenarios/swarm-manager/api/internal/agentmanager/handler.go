package agentmanager

import (
	"net/http"

	"github.com/gorilla/mux"
	"swarm-manager/internal/httputil"
)

// StatusResponse reports agent-manager integration status.
type StatusResponse struct {
	Enabled   bool   `json:"enabled"`
	Available bool   `json:"available"`
	URL       string `json:"url,omitempty"`
	ProfileID string `json:"profileId,omitempty"`
}

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
	response := StatusResponse{}
	if h.service != nil {
		response.Enabled = h.service.IsEnabled()
		if response.Enabled {
			response.Available = h.service.IsAvailable(r.Context())
			if url, err := h.service.ResolveURL(r.Context()); err == nil {
				response.URL = url
			}
		}
		response.ProfileID = h.service.GetProfileID()
	}

	if err := httputil.JSON(w, response); err != nil {
		httputil.InternalError(w, "[agent-manager] status", "failed to encode response")
	}
}
