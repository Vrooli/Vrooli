package capabilities

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Handler exposes the capability registry description used by operator UI
// surfaces and the generic dependency-conformance gate.
type Handler struct {
	Registry *Registry
}

func NewHandler(registry *Registry) *Handler { return &Handler{Registry: registry} }

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/capabilities/describe", h.Describe).Methods(http.MethodGet)
}

func (h *Handler) Describe(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Registry == nil {
		http.Error(w, "capabilities registry is unavailable", http.StatusServiceUnavailable)
		return
	}
	data, err := h.Registry.Describe(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
