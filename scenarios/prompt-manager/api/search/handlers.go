// Package search provides skill search functionality.
//
// DOC: docs/reference/api-endpoints.md#search
package search

import (
	"encoding/json"
	"net/http"
)

// Handlers provides HTTP handlers for search operations.
type Handlers struct {
	service *Service
}

// NewHandlers creates a new search handler.
func NewHandlers(service *Service) *Handlers {
	return &Handlers{service: service}
}

// Search handles GET /api/v1/search/skills - searches skills.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	query := SearchQuery{
		Query:  r.URL.Query().Get("q"),
		Tag:    r.URL.Query().Get("tag"),
		Folder: r.URL.Query().Get("folder"),
	}

	response, err := h.service.Search(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
