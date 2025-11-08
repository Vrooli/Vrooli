package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
)

// ErrorResponse represents a structured JSON error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    int    `json:"code"`
}

// respondError writes a structured JSON error response
func respondError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
	if err != nil {
		slog.Error("failed to encode error response", "error", err)
	}
}

// respondInternalError responds with a 500 Internal Server Error
func respondInternalError(w http.ResponseWriter, message string, err error) {
	fullMessage := message
	if err != nil {
		fullMessage = fmt.Sprintf("%s: %v", message, err)
	}
	slog.Error(message, "error", err)
	respondError(w, fullMessage, http.StatusInternalServerError)
}

// respondBadRequest responds with a 400 Bad Request
func respondBadRequest(w http.ResponseWriter, message string) {
	respondError(w, message, http.StatusBadRequest)
}

// respondNotFound responds with a 404 Not Found
func respondNotFound(w http.ResponseWriter, resource string) {
	respondError(w, fmt.Sprintf("%s not found", resource), http.StatusNotFound)
}

// respondServiceUnavailable responds with a 503 Service Unavailable
func respondServiceUnavailable(w http.ResponseWriter, message string) {
	respondError(w, message, http.StatusServiceUnavailable)
}

// handleDraftError handles common draft-related errors with appropriate responses
func handleDraftError(w http.ResponseWriter, err error, context string) bool {
	if err == nil {
		return false
	}

	if err == sql.ErrNoRows {
		respondNotFound(w, "Draft")
		return true
	}

	if errors.Is(err, ErrDatabaseNotAvailable) {
		respondServiceUnavailable(w, "Database not available")
		return true
	}

	respondInternalError(w, context, err)
	return true
}

// respondInvalidEntityType responds when entity type validation fails
func respondInvalidEntityType(w http.ResponseWriter) {
	respondBadRequest(w, "Invalid entity type. Must be 'scenario' or 'resource'")
}

// respondInvalidJSON responds when JSON decoding fails
func respondInvalidJSON(w http.ResponseWriter, err error) {
	respondBadRequest(w, fmt.Sprintf("Invalid request body: %v", err))
}
