// DOC: docs/reference/api-endpoints.md#references
// DOC: docs/internal/SEAMS.md#api-handlers--domain-services
// DOC: docs/internal/SEAMS.md#error-semantics
//
// Package handlers implements HTTP request handling.
// Handlers parse HTTP requests, delegate to domain services, and serialize responses.
// They do not contain business logic or access the database directly.
//
// Error Handling Strategy:
// - Domain errors (ErrNotFound, ErrInvalidSlug, etc.) are mapped to structured API errors
// - Each error category has a specific HTTP status and recovery guidance
// - Errors are logged with appropriate severity for observability
// - Client responses include both message and machine-readable code
//
// Dry-Run Support:
// - Mutating endpoints (Create, Update, Delete) support the X-Dry-Run header
// - When X-Dry-Run: true, validation runs but no persistence occurs
// - Responses include "dry_run": true and realistic response data
// - See: CLI Steer skill for details on dry-run conventions
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	"development-toolchain-validator/domain/reference"
	"development-toolchain-validator/internal/config"
)

// ReferenceHandler handles HTTP requests for reference scenarios.
// [REQ:P0-002] Reference Scenario API Endpoints
type ReferenceHandler struct {
	service *reference.Service
	config  config.ValidationConfig
}

// NewReferenceHandler creates a new reference handler.
func NewReferenceHandler(service *reference.Service) *ReferenceHandler {
	return &ReferenceHandler{
		service: service,
		config:  config.DefaultConfig().Validation,
	}
}

// NewReferenceHandlerWithConfig creates a handler with explicit configuration.
// Used for testing and when custom validation constraints are needed.
func NewReferenceHandlerWithConfig(service *reference.Service, cfg config.ValidationConfig) *ReferenceHandler {
	return &ReferenceHandler{
		service: service,
		config:  cfg,
	}
}

// RegisterRoutes adds reference routes to the router.
func (h *ReferenceHandler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/api/v1/references", h.List).Methods("GET")
	r.HandleFunc("/api/v1/references", h.Create).Methods("POST")
	r.HandleFunc("/api/v1/references/{id}", h.GetByID).Methods("GET")
	r.HandleFunc("/api/v1/references/{id}", h.Update).Methods("PATCH")
	r.HandleFunc("/api/v1/references/{id}", h.Delete).Methods("DELETE")
	r.HandleFunc("/api/v1/references/by-slug/{slug}", h.GetBySlug).Methods("GET")
}

// List returns all reference scenarios with optional filtering.
func (h *ReferenceHandler) List(w http.ResponseWriter, r *http.Request) {
	opts := reference.ListOptions{
		Template: r.URL.Query().Get("template"),
	}

	refs, err := h.service.List(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list references")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"references": refs,
		"count":      len(refs),
	})
}

// Create adds a new reference scenario.
// Supports dry-run: X-Dry-Run: true header runs validation without persistence.
func (h *ReferenceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input reference.CreateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Dry-run: validate input and return realistic response without persistence
	if isDryRun(r) {
		absPath, err := h.service.ValidateCreate(r.Context(), input)
		if err != nil {
			cfg := ErrorMappingConfig{
				Slug:       input.Slug,
				Path:       input.Path,
				SlugMinLen: h.config.SlugMinLength,
				SlugMaxLen: h.config.SlugMaxLength,
			}
			status, msg := HandleCreateError(err, cfg)
			writeError(w, status, msg)
			return
		}

		// Return realistic response with generated values
		now := time.Now()
		dryRunRef := reference.Reference{
			ID:          uuid.New().String(),
			Slug:        input.Slug,
			Name:        input.Name,
			Template:    input.Template,
			Path:        absPath,
			Description: input.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"data":    dryRunRef,
		})
		return
	}

	ref, err := h.service.Create(r.Context(), input)
	if err != nil {
		// Map domain error to API error with context for rich error messages
		cfg := ErrorMappingConfig{
			Slug:       input.Slug,
			Path:       input.Path,
			SlugMinLen: h.config.SlugMinLength,
			SlugMaxLen: h.config.SlugMaxLength,
		}
		status, msg := HandleCreateError(err, cfg)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, ref)
}

// GetByID retrieves a reference by its ID.
func (h *ReferenceHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	ref, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		status, msg := HandleGetError(err, id)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, ref)
}

// GetBySlug retrieves a reference by its slug.
func (h *ReferenceHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]

	ref, err := h.service.GetBySlug(r.Context(), slug)
	if err != nil {
		status, msg := HandleGetError(err, slug)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, ref)
}

// Update modifies an existing reference.
// Supports dry-run: X-Dry-Run: true header runs validation without persistence.
func (h *ReferenceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var input reference.UpdateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Dry-run: validate input and return realistic response without persistence
	if isDryRun(r) {
		_, err := h.service.ValidateUpdate(r.Context(), id, input)
		if err != nil {
			cfg := ErrorMappingConfig{ResourceID: id}
			if input.Path != nil {
				cfg.Path = *input.Path
			}
			status, msg := HandleCreateError(err, cfg)
			writeError(w, status, msg)
			return
		}

		// Get the existing reference to build a realistic response
		existing, err := h.service.GetByID(r.Context(), id)
		if err != nil {
			status, msg := HandleGetError(err, id)
			writeError(w, status, msg)
			return
		}

		// Apply updates to the existing reference for response
		now := time.Now()
		dryRunRef := *existing
		dryRunRef.UpdatedAt = now
		if input.Name != nil {
			dryRunRef.Name = *input.Name
		}
		if input.Template != nil {
			dryRunRef.Template = *input.Template
		}
		if input.Path != nil {
			dryRunRef.Path = *input.Path
		}
		if input.Description != nil {
			dryRunRef.Description = *input.Description
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"data":    dryRunRef,
		})
		return
	}

	ref, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		// Build error mapping config with available context
		cfg := ErrorMappingConfig{ResourceID: id}
		if input.Path != nil {
			cfg.Path = *input.Path
		}
		status, msg := HandleCreateError(err, cfg)
		writeError(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, ref)
}

// Delete removes a reference by ID.
// Supports dry-run: X-Dry-Run: true header checks existence without deletion.
func (h *ReferenceHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	// Dry-run: verify reference exists without deletion
	if isDryRun(r) {
		_, err := h.service.GetByID(r.Context(), id)
		if err != nil {
			status, msg := HandleGetError(err, id)
			writeError(w, status, msg)
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"dry_run": true,
			"deleted": id,
		})
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		status, msg := HandleGetError(err, id)
		writeError(w, status, msg)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// ─────────────────────────────────────────────────────────────────────────────
// Dry-Run Support
// Per CLI Steer skill, mutating endpoints support dry-run via X-Dry-Run header.
// ─────────────────────────────────────────────────────────────────────────────

// isDryRun checks if the request is a dry-run request.
// Dry-run requests have the X-Dry-Run header set to "true".
func isDryRun(r *http.Request) bool {
	return r.Header.Get("X-Dry-Run") == "true"
}
