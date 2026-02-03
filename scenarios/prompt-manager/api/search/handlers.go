// Package search provides skill search functionality.
//
// DOC: docs/reference/api-endpoints.md#search
package search

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

// ContentSearch handles GET /api/v1/search/skills/content - searches skill contents.
func (h *Handlers) ContentSearch(w http.ResponseWriter, r *http.Request) {
	query := ContentSearchQuery{
		Query:         r.URL.Query().Get("q"),
		Tags:          parseMultiValues(r.URL.Query()["tag"], r.URL.Query()["tags"]),
		Folders:       parseMultiValues(r.URL.Query()["folder"], r.URL.Query()["folders"]),
		CaseSensitive: parseBool(r.URL.Query().Get("caseSensitive")),
		WholeWord:     parseBool(r.URL.Query().Get("wholeWord")),
		Regex:         parseBool(r.URL.Query().Get("regex")),
		Limit:         parseInt(r.URL.Query().Get("limit")),
	}

	response, err := h.service.SearchContent(query)
	if err != nil {
		if errors.Is(err, ErrInvalidPattern) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func parseBool(raw string) bool {
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(raw)
	if err != nil {
		return false
	}
	return parsed
}

func parseInt(raw string) int {
	if raw == "" {
		return 0
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return parsed
}

func parseMultiValues(primary []string, alternate []string) []string {
	combined := append([]string{}, primary...)
	combined = append(combined, alternate...)

	var values []string
	for _, entry := range combined {
		for _, part := range strings.Split(entry, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			values = append(values, trimmed)
		}
	}

	return values
}
