// Package tags provides tag management for prompt categorization.
//
// DOC: docs/reference/api-endpoints.md#tags
package tags

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

// Handlers provides HTTP handlers for tag operations.
type Handlers struct {
	repo TagRepository
}

// NewHandlers creates a new tags handler.
func NewHandlers(repo TagRepository) *Handlers {
	return &Handlers{repo: repo}
}

func (h *Handlers) repoFor(ctx context.Context) TagRepository {
	if scoped, ok := h.repo.(interface {
		WithRequestContext(context.Context) TagRepository
	}); ok {
		return scoped.WithRequestContext(ctx)
	}
	return h.repo
}

// List handles GET /tags - returns all tags.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tags, err := h.repoFor(r.Context()).GetAll()
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

	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	repo := h.repoFor(r.Context())
	existing, err := repo.GetAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, candidate := range existing {
		if strings.EqualFold(strings.TrimSpace(candidate.Name), tag.Name) {
			http.Error(w, "tag already exists", http.StatusConflict)
			return
		}
	}

	if err := repo.Create(&tag); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}
