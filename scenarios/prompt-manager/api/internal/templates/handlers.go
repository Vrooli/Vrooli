package templates

import (
	"encoding/json"
	"net/http"
)

// Handlers provides HTTP handlers for agent file templates.
type Handlers struct {
	store *Store
}

// NewHandlers creates a new templates handler.
func NewHandlers(store *Store) *Handlers {
	return &Handlers{store: store}
}

// ListAgentFileTemplates handles GET /agent-file-templates.
func (h *Handlers) ListAgentFileTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := h.store.ListAgentFileTemplates(r.Context())
	if err != nil {
		http.Error(w, "Failed to load templates", http.StatusInternalServerError)
		return
	}

	response := AgentFileTemplateListResponse{
		Templates: templates,
		Count:     len(templates),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
