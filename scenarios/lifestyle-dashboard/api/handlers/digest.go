// DOC: docs/concepts/ARCHITECTURE.md#Handler-Layer
// DOC: PRD.md#OT-P1-002
//
// Package handlers provides HTTP handlers for the weekly digest endpoints.
//
// [REQ:LD-DIGEST-WEEKLY] Weekly digest endpoints.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"lifestyle-dashboard/domain"
	appErrors "lifestyle-dashboard/errors"
)

// GetCurrentDigest returns the most recent weekly digest.
// GET /api/v1/digests/current
// [REQ:LD-DIGEST-WEEKLY]
func (h *Handler) GetCurrentDigest(w http.ResponseWriter, r *http.Request) {
	if h.Digest == nil {
		WriteAPIError(w, appErrors.NewUnavailableError(appErrors.CodeDependencyUnavailable, "digest service not configured"))
		return
	}

	digest, err := h.Digest.GetLatestDigest(r.Context())
	if err != nil {
		log.Printf("[ERROR] Failed to generate digest: %v", err)
		WriteAPIError(w, appErrors.NewInternalError(appErrors.CodeDatabaseError, "Failed to generate digest"))
		return
	}

	if digest == nil {
		WriteAPIError(w, appErrors.NewNotFoundError(appErrors.CodeDomainNotFound, "digest", "current"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.WeeklyDigestResponse{Digest: *digest})
}

// GetDigestByWeek returns the digest for a specific week.
// GET /api/v1/digests/{week_start}
// week_start should be the Monday of the target week (YYYY-MM-DD)
// [REQ:LD-DIGEST-WEEKLY]
func (h *Handler) GetDigestByWeek(w http.ResponseWriter, r *http.Request) {
	if h.Digest == nil {
		WriteAPIError(w, appErrors.NewUnavailableError(appErrors.CodeDependencyUnavailable, "digest service not configured"))
		return
	}

	weekStart := r.PathValue("week_start")
	if weekStart == "" {
		WriteAPIError(w, appErrors.NewValidationError(appErrors.CodeMissingField, "Week start date required").WithDetails("field", "week_start"))
		return
	}

	digest, err := h.Digest.GenerateWeeklyDigest(r.Context(), weekStart)
	if err != nil {
		log.Printf("[ERROR] Failed to generate digest for %s: %v", weekStart, err)
		WriteAPIError(w, appErrors.NewInternalError(appErrors.CodeDatabaseError, "Failed to generate digest"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(domain.WeeklyDigestResponse{Digest: *digest})
}
