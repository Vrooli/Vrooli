// DOC: docs/concepts/ARCHITECTURE.md#Presentation-Layer
// DOC: docs/QUICKSTART.md
// DOC: docs/internal/ERROR_SEMANTICS.md
//
// Package handlers contains HTTP handlers for the lifestyle dashboard API.
// This package separates presentation/entry concerns from domain logic,
// applying the Boundary of Responsibility principle.
//
// Handlers delegate storage operations to repository interfaces, enabling:
// - Unit testing with mock repositories
// - Storage backend swapping (SQLite, PostgreSQL, in-memory)
// - Clear separation between HTTP handling and persistence
package handlers

import (
	"encoding/json"
	"net/http"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
	"lifestyle-dashboard/repository"
)

// Handler provides shared dependencies for all HTTP handlers.
// It uses repository interfaces rather than direct database access,
// following the Storage Architecture skill's abstraction pattern.
type Handler struct {
	Events      repository.EventRepository
	Domains     repository.DomainRepository
	Stats       repository.StatsRepository
	Storage     repository.StorageRepository
	Briefs      repository.BriefRepository
	ScoreConfig repository.ScoreConfigRepository
	Digest      repository.DigestRepository
}

// New creates a Handler with all repository implementations.
// All repositories are required for the handler to function correctly.
func New(events repository.EventRepository, domains repository.DomainRepository, stats repository.StatsRepository, storage repository.StorageRepository, briefs repository.BriefRepository, scoreConfig repository.ScoreConfigRepository, digest repository.DigestRepository) *Handler {
	return &Handler{
		Events:      events,
		Domains:     domains,
		Stats:       stats,
		Storage:     storage,
		Briefs:      briefs,
		ScoreConfig: scoreConfig,
		Digest:      digest,
	}
}

// WriteJSON encodes data as JSON and writes it to the response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes a JSON error response (legacy format for backward compatibility).
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, domain.ErrorResponse{
		Error:   true,
		Message: message,
	})
}

// WriteAPIError writes a structured error response with category, code, and recovery hints.
// This is the preferred method for new code.
func WriteAPIError(w http.ResponseWriter, err *errors.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.StatusCode())
	json.NewEncoder(w).Encode(err)
}
