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
	"lifestyle-dashboard/repository"
)

// Handler provides shared dependencies for all HTTP handlers.
// It uses repository interfaces rather than direct database access,
// following the Storage Architecture skill's abstraction pattern.
type Handler struct {
	Events  repository.EventRepository
	Domains repository.DomainRepository
	Stats   repository.StatsRepository
}

// New creates a new Handler with the given repository implementations.
func New(events repository.EventRepository, domains repository.DomainRepository, stats repository.StatsRepository) *Handler {
	return &Handler{
		Events:  events,
		Domains: domains,
		Stats:   stats,
	}
}

// WriteJSON encodes data as JSON and writes it to the response.
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, domain.ErrorResponse{
		Error:   true,
		Message: message,
	})
}
