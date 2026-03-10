// DOC: docs/concepts/ARCHITECTURE.md#Presentation-Layer
// DOC: PRD.md#OT-P0-005
//
// Package handlers provides HTTP handlers for the daily brief system.
//
// [REQ:LD-BRIEF-MORNING] Morning brief endpoint.
// [REQ:LD-BRIEF-EVENING] Evening review endpoint.
// [REQ:LD-BRIEF-CONSOLIDATE] Cross-domain consolidation.
package handlers

import (
	"net/http"
	"time"

	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
)

// GetCurrentBrief returns the appropriate brief based on current time.
// Before 21:00: morning brief, after 21:00: evening brief.
// GET /api/v1/briefs/current
// [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING]
func (h *Handler) GetCurrentBrief(w http.ResponseWriter, r *http.Request) {
	if h.Briefs == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "brief service not configured"))
		return
	}

	brief, err := h.Briefs.GetCurrentBrief(r.Context())
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "failed to generate brief"))
		return
	}

	WriteJSON(w, http.StatusOK, domain.BriefResponse{
		Brief: *brief,
		Config: domain.BriefConfig{
			MorningHour: 7,
			EveningHour: 21,
		},
	})
}

// GetMorningBrief returns the morning brief for a given date.
// GET /api/v1/briefs/morning?date=2026-03-10
// [REQ:LD-BRIEF-MORNING]
func (h *Handler) GetMorningBrief(w http.ResponseWriter, r *http.Request) {
	if h.Briefs == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "brief service not configured"))
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		WriteAPIError(w, errors.NewValidationError(errors.CodeInvalidField, "invalid date format, use YYYY-MM-DD"))
		return
	}

	brief, err := h.Briefs.GenerateMorningBrief(r.Context(), date)
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "failed to generate morning brief"))
		return
	}

	WriteJSON(w, http.StatusOK, domain.BriefResponse{
		Brief: *brief,
		Config: domain.BriefConfig{
			MorningHour: 7,
			EveningHour: 21,
		},
	})
}

// GetEveningBrief returns the evening brief for a given date.
// GET /api/v1/briefs/evening?date=2026-03-10
// [REQ:LD-BRIEF-EVENING]
func (h *Handler) GetEveningBrief(w http.ResponseWriter, r *http.Request) {
	if h.Briefs == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "brief service not configured"))
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", date); err != nil {
		WriteAPIError(w, errors.NewValidationError(errors.CodeInvalidField, "invalid date format, use YYYY-MM-DD"))
		return
	}

	brief, err := h.Briefs.GenerateEveningBrief(r.Context(), date)
	if err != nil {
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "failed to generate evening brief"))
		return
	}

	WriteJSON(w, http.StatusOK, domain.BriefResponse{
		Brief: *brief,
		Config: domain.BriefConfig{
			MorningHour: 7,
			EveningHour: 21,
		},
	})
}
