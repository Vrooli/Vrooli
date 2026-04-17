// DOC: docs/internal/SEAMS.md#error-semantics
// DOC: docs/internal/SEAMS.md#decision-error-category-mapping
//
// Package handlers provides HTTP error response helpers.
//
// # Purpose
//
// This file centralizes error-to-HTTP response mapping, ensuring:
//   - Consistent API error format across all handlers
//   - Domain errors are translated to user-friendly messages
//   - Structured responses enable programmatic client handling
//   - Sensitive details stay in logs, not responses
//
// # Error Flow
//
//	Handler catches error → MapDomainError → logError → WriteStructuredError
//	                            ↓                           ↓
//	                    Category + Code               HTTP Status + JSON
//
// # When to Modify
//
// Add to MapDomainError when:
//   - New domain sentinel errors are introduced
//   - Error message should include request context
//
// Do NOT modify when:
//   - Adding new error categories (go to internal/errors)
//   - Changing HTTP status mappings (go to internal/errors)
package handlers

import (
	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/domain/skill"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	apierrors "development-toolchain-validator/internal/errors"
)

// ErrorResponse represents the JSON structure for error responses.
// It maintains backward compatibility with existing clients while adding
// structured error information.
type ErrorResponse struct {
	// Error is the primary error message (backward compatible)
	Error string `json:"error"`

	// Code is the machine-readable error code (optional)
	Code string `json:"code,omitempty"`

	// Category classifies the error for recovery decisions (optional)
	Category string `json:"category,omitempty"`

	// Details provides additional context (optional)
	Details map[string]interface{} `json:"details,omitempty"`

	// Recovery suggests what the user/agent should do next (optional)
	Recovery string `json:"recovery,omitempty"`
}

// WriteStructuredError writes a structured error response.
// It logs the error for observability and returns appropriate HTTP status.
func WriteStructuredError(w http.ResponseWriter, err *apierrors.Error) {
	// Log for observability (severity determines log level)
	logError(err)

	response := ErrorResponse{
		Error:    err.Message,
		Code:     err.Code,
		Category: string(err.Category),
		Details:  err.Details,
		Recovery: err.Recovery,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.ToHTTPStatus())
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		log.Printf("[ERROR] failed to encode error response: %v", encodeErr)
	}
}

// MapDomainError converts domain-layer errors to structured API errors.
// This centralizes the mapping from domain semantics to API semantics.
func MapDomainError(err error, cfg ErrorMappingConfig) *apierrors.Error {
	// Check for known domain errors first
	switch {
	case errors.Is(err, reference.ErrNotFound):
		return apierrors.ReferenceNotFound(cfg.ResourceID)

	case errors.Is(err, reference.ErrInvalidSlug):
		return apierrors.InvalidSlug(cfg.Slug, cfg.SlugMinLen, cfg.SlugMaxLen)

	case errors.Is(err, reference.ErrSlugExists):
		return apierrors.SlugExists(cfg.Slug)

	case errors.Is(err, reference.ErrPathNotExists):
		return apierrors.PathNotExists(cfg.Path)

	default:
		// Unknown errors become internal errors
		// The underlying error is wrapped but not exposed to clients
		return apierrors.Internal("An unexpected error occurred").WithCause(err)
	}
}

// ErrorMappingConfig provides context for error mapping.
type ErrorMappingConfig struct {
	// ResourceID is the identifier used in the request (for not-found errors)
	ResourceID string

	// Slug is the slug provided in the request (for slug validation errors)
	Slug string

	// Path is the path provided in the request (for path validation errors)
	Path string

	// SlugMinLen/SlugMaxLen are from service config (for validation messages)
	SlugMinLen int
	SlugMaxLen int
}

// logError logs the error with appropriate severity.
func logError(err *apierrors.Error) {
	switch err.Severity {
	case apierrors.SeverityCritical:
		log.Printf("[CRITICAL] %s: %s (code=%s, category=%s)", err.Code, err.Message, err.Code, err.Category)
		if err.Cause != nil {
			log.Printf("[CRITICAL] cause: %v", err.Cause)
		}
	case apierrors.SeverityHigh:
		log.Printf("[ERROR] %s: %s", err.Code, err.Message)
		if err.Cause != nil {
			log.Printf("[ERROR] cause: %v", err.Cause)
		}
	case apierrors.SeverityMedium:
		log.Printf("[WARN] %s: %s", err.Code, err.Message)
	case apierrors.SeverityLow:
		// Low severity validation errors are expected and don't need logging
		// unless debugging is enabled
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Backward-Compatible Error Helpers
// These maintain existing API behavior while transitioning to structured errors
// ─────────────────────────────────────────────────────────────────────────────

// writeErrorCompat writes an error response in the legacy format.
// Used to maintain backward compatibility during transition.
func writeErrorCompat(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if encodeErr := json.NewEncoder(w).Encode(map[string]string{"error": message}); encodeErr != nil {
		log.Printf("[ERROR] failed to encode error response: %v", encodeErr)
	}
}

// HandleCreateError processes errors from the Create operation.
// Returns the HTTP status and message for backward compatibility.
func HandleCreateError(err error, cfg ErrorMappingConfig) (int, string) {
	apiErr := MapDomainError(err, cfg)
	logError(apiErr)
	return apiErr.ToHTTPStatus(), apiErr.Message
}

// HandleGetError processes errors from Get operations.
// Returns the HTTP status and message for backward compatibility.
func HandleGetError(err error, resourceID string) (int, string) {
	cfg := ErrorMappingConfig{ResourceID: resourceID}
	apiErr := MapDomainError(err, cfg)
	logError(apiErr)
	return apiErr.ToHTTPStatus(), apiErr.Message
}

// ─────────────────────────────────────────────────────────────────────────────
// Skill Connection Error Mapping
// [REQ:REQ-P0-003] Prompt-Manager Skill Connection Store - Error handling
// ─────────────────────────────────────────────────────────────────────────────

// SkillErrorMappingConfig provides context for skill error mapping.
type SkillErrorMappingConfig struct {
	// ConnectionID is the connection identifier (for not-found errors)
	ConnectionID string
	// ReferenceID is the reference identifier (for validation/conflict errors)
	ReferenceID string
	// SkillID is the skill identifier (for validation errors)
	SkillID string
}

// MapSkillDomainError converts skill domain errors to structured API errors.
// This centralizes the mapping from skill domain semantics to API semantics.
func MapSkillDomainError(err error, cfg SkillErrorMappingConfig) *apierrors.Error {
	switch {
	case errors.Is(err, skill.ErrNotFound):
		return apierrors.ConnectionNotFound(cfg.ConnectionID)

	case errors.Is(err, skill.ErrInvalidSkillID):
		return apierrors.InvalidSkillID(cfg.SkillID)

	case errors.Is(err, skill.ErrInvalidReferenceID):
		return apierrors.InvalidReferenceID()

	case errors.Is(err, skill.ErrConnectionExists):
		return apierrors.ConnectionExists(cfg.ReferenceID, cfg.SkillID)

	default:
		// Unknown errors become internal errors
		return apierrors.Internal("An unexpected error occurred").WithCause(err)
	}
}

// HandleConnectError processes errors from the Connect operation.
// Returns the HTTP status and message for backward compatibility.
func HandleConnectError(err error, referenceID, skillID string) (int, string) {
	cfg := SkillErrorMappingConfig{
		ReferenceID: referenceID,
		SkillID:     skillID,
	}
	apiErr := MapSkillDomainError(err, cfg)
	logError(apiErr)
	return apiErr.ToHTTPStatus(), apiErr.Message
}

// HandleConnectionGetError processes errors from connection Get/Delete operations.
// Returns the HTTP status and message for backward compatibility.
func HandleConnectionGetError(err error, connectionID string) (int, string) {
	cfg := SkillErrorMappingConfig{ConnectionID: connectionID}
	apiErr := MapSkillDomainError(err, cfg)
	logError(apiErr)
	return apiErr.ToHTTPStatus(), apiErr.Message
}
