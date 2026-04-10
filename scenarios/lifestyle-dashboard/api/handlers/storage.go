// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: PRD.md#OT-P0-006
// DOC: docs/internal/ERROR_SEMANTICS.md
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
)

// GetStorageInfo handles GET /api/v1/storage - P0-006
// [REQ:LD-UI-STORAGE] Returns storage information for the settings page.
func (h *Handler) GetStorageInfo(w http.ResponseWriter, r *http.Request) {
	if h.Storage == nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Storage repository not configured"))
		return
	}

	info, err := h.Storage.GetStorageInfo(r.Context())
	if err != nil {
		log.Printf("[ERROR] GetStorageInfo: database error: %v", err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to get storage info. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, info)
}

// CleanupEvents handles DELETE /api/v1/storage/events - P0-006
// [REQ:LD-UI-STORAGE] Cleans up events matching the request criteria.
func (h *Handler) CleanupEvents(w http.ResponseWriter, r *http.Request) {
	if h.Storage == nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Storage repository not configured"))
		return
	}

	var req domain.CleanupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Empty body is valid - means "clear all"
		if err.Error() != "EOF" {
			WriteAPIError(w, errors.NewValidationError(errors.CodeInvalidJSON, "Invalid JSON body"))
			return
		}
	}

	result, err := h.Storage.CleanupEvents(r.Context(), req)
	if err != nil {
		log.Printf("[ERROR] CleanupEvents: database error: %v", err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to cleanup events. Please try again."))
		return
	}

	log.Printf("[INFO] CleanupEvents: %s", result.Message)
	WriteJSON(w, http.StatusOK, result)
}
