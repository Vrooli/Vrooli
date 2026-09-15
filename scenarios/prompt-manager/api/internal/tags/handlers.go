// Package tags provides tag management for prompt categorization.
//
// DOC: docs/reference/api-endpoints.md#tags
package tags

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Handlers provides HTTP handlers for tag operations.
type Handlers struct {
	repo TagRepository
}

// ListTags returns tags through the domain repository without an HTTP boundary.
func ListTags(ctx context.Context, repo TagRepository) ([]Tag, error) {
	return repoFor(ctx, repo).GetAll()
}

// CreateTag validates, de-duplicates, and persists a tag without an HTTP boundary.
func CreateTag(ctx context.Context, repo TagRepository, tag Tag) (Tag, error) {
	tag.Name = strings.TrimSpace(tag.Name)
	if tag.Name == "" {
		return Tag{}, fmt.Errorf("Name is required")
	}
	repo = repoFor(ctx, repo)
	existing, err := repo.GetAll()
	if err != nil {
		return Tag{}, err
	}
	for _, candidate := range existing {
		if strings.EqualFold(strings.TrimSpace(candidate.Name), tag.Name) {
			return Tag{}, ErrDuplicate
		}
	}
	if err := repo.Create(&tag); err != nil {
		return Tag{}, err
	}
	return tag, nil
}

var ErrDuplicate = fmt.Errorf("tag already exists")

// NewHandlers creates a new tags handler.
func NewHandlers(repo TagRepository) *Handlers {
	return &Handlers{repo: repo}
}

func (h *Handlers) repoFor(ctx context.Context) TagRepository {
	return repoFor(ctx, h.repo)
}

func repoFor(ctx context.Context, repo TagRepository) TagRepository {
	if scoped, ok := repo.(interface {
		WithRequestContext(context.Context) TagRepository
	}); ok {
		return scoped.WithRequestContext(ctx)
	}
	return repo
}

// List handles GET /tags - returns all tags.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	tags, err := ListTags(r.Context(), h.repo)
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

	created, err := CreateTag(r.Context(), h.repo, tag)
	if err != nil {
		if err == ErrDuplicate {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		if err.Error() == "Name is required" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tag = created

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(tag)
}
