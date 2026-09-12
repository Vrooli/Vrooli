package integrationstatus

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

// Handler is the shared API projection consumed by operator surfaces. It does
// not expose a second per-integration health decision.
type Handler struct{ provider *Provider }

func NewHandler(provider *Provider) *Handler { return &Handler{provider: provider} }

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/v1/integrations", h.List).Methods(http.MethodGet)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		http.Error(w, "integration status provider is unavailable", http.StatusServiceUnavailable)
		return
	}
	statuses, err := h.provider.Statuses(r.Context())
	if err != nil {
		http.Error(w, "failed to check integrations", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(struct {
		Integrations []Status `json:"integrations"`
	}{Integrations: statuses}); err != nil {
		http.Error(w, "failed to encode integration status", http.StatusInternalServerError)
	}
}
