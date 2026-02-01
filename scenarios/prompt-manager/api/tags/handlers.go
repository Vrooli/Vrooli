// Package tags provides tag management for prompt categorization.
//
// DOC: docs/reference/api-endpoints.md#tags
package tags

import (
	"encoding/json"
	"net/http"
)

// Handlers provides HTTP handlers for tag operations.
type Handlers struct {
	repo TagRepository
}

// NewHandlers creates a new tags handler.
func NewHandlers(repo TagRepository) *Handlers {
	return &Handlers{repo: repo}
}

// List handles GET /tags - returns all tags.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tags)
}

// Create handles POST /tags - creates a new tag.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var tag Tag
	if err := json.NewDecoder(r.Body).Decode(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if tag.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if err := h.repo.Create(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}
