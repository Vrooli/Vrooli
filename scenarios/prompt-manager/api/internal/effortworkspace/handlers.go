package effortworkspace

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Handlers struct {
	store *Store
}

func NewHandlers(store *Store) *Handlers { return &Handlers{store: store} }

// List handles GET /api/v1/effort-workspaces?teamId=<team-id>.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	teamID := strings.TrimSpace(r.URL.Query().Get("teamId"))
	if teamID == "" {
		http.Error(w, "teamId is required", http.StatusBadRequest)
		return
	}
	result, err := h.store.List(r.Context(), teamID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, result)
}

// Read handles GET /api/v1/effort-workspaces/content?effortRef=<ref>&path=<path>.
func (h *Handlers) Read(w http.ResponseWriter, r *http.Request) {
	content, err := h.store.Read(r.Context(), r.URL.Query().Get("effortRef"), r.URL.Query().Get("path"))
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "unavailable") || strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}
	writeJSON(w, content)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
