// DOC: docs/internal/SEAMS.md#error-handling
package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

/**
 * ERROR CATEGORIES - Read before modifying
 *
 * Each category has a specific recovery path. Changing these
 * affects UI error states and automated retry logic.
 *
 * To add a new category:
 * 1. Ensure it has a distinct recovery path
 * 2. Update UI error handling to recognize it
 * 3. Add to API error mapping if client-facing
 */

// ErrorCategory classifies errors for consistent handling and recovery.
type ErrorCategory string

const (
	// ErrCategoryValidation indicates bad user input that can be fixed by the caller.
	ErrCategoryValidation ErrorCategory = "validation"
	// ErrCategoryNotFound indicates the requested resource does not exist.
	ErrCategoryNotFound ErrorCategory = "not_found"
	// ErrCategoryConflict indicates a uniqueness or state constraint was violated.
	ErrCategoryConflict ErrorCategory = "conflict"
	// ErrCategoryDependency indicates an external service (DB, LLM) is unavailable.
	ErrCategoryDependency ErrorCategory = "dependency"
	// ErrCategoryInternal indicates an unexpected server error.
	ErrCategoryInternal ErrorCategory = "internal"
)

// APIError is the structured error response returned by all API endpoints.
type APIError struct {
	Category  ErrorCategory `json:"category"`
	Message   string        `json:"message"`
	Retryable bool          `json:"retryable"`
}

// writeAPIError writes a structured error response. It logs the internal error
// details while returning a safe message to the client.
func writeAPIError(w http.ResponseWriter, status int, category ErrorCategory, userMsg string, internalErr error) {
	if internalErr != nil {
		log.Printf("[ERROR] %s: %s (internal: %v)", category, userMsg, internalErr)
	} else {
		log.Printf("[ERROR] %s: %s", category, userMsg)
	}

	retryable := category == ErrCategoryDependency
	resp := APIError{
		Category:  category,
		Message:   userMsg,
		Retryable: retryable,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// classifyAndWriteError maps common Go errors to structured API responses.
// It prevents raw internal error messages from leaking to clients.
func classifyAndWriteError(w http.ResponseWriter, err error, resourceName string) {
	switch {
	case err == sql.ErrNoRows:
		writeAPIError(w, http.StatusNotFound, ErrCategoryNotFound,
			resourceName+" not found", nil)
	case isUniqueViolation(err):
		writeAPIError(w, http.StatusConflict, ErrCategoryConflict,
			resourceName+" already exists or violates a uniqueness constraint", err)
	case isForeignKeyViolation(err):
		writeAPIError(w, http.StatusBadRequest, ErrCategoryValidation,
			"referenced "+resourceName+" does not exist", err)
	default:
		writeAPIError(w, http.StatusInternalServerError, ErrCategoryInternal,
			"an unexpected error occurred while processing "+resourceName, err)
	}
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation (23505).
func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23505")
}

// isForeignKeyViolation checks if the error is a PostgreSQL FK violation (23503).
func isForeignKeyViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "23503")
}

// writeValidationError is a convenience for input validation failures.
func writeValidationError(w http.ResponseWriter, msg string) {
	writeAPIError(w, http.StatusBadRequest, ErrCategoryValidation, msg, nil)
}
