package overview

import (
	"net/http"

	"github.com/gorilla/mux"

	"swarm-manager/internal/httputil"
)

// Handler provides the HTTP endpoint for the overview aggregation.
type Handler struct {
	service *Service
}

// NewHandler creates an overview Handler backed by the given service.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers the overview endpoint on the given router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/overview", h.GetOverview).Methods("GET")
}

// GetOverview returns the aggregated overview payload.
func (h *Handler) GetOverview(w http.ResponseWriter, _ *http.Request) {
	resp, err := h.service.GetOverview()
	if err != nil {
		httputil.InternalError(w, "[overview] get", "failed to build overview")
		return
	}
	if err := httputil.JSON(w, resp); err != nil {
		httputil.InternalError(w, "[overview] get", "failed to encode response")
	}
}
