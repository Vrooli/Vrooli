// DOC: docs/QUICKSTART.md#Query-Data
// DOC: PRD.md#OT-P0-003
// DOC: README.md#Statistics
// DOC: docs/internal/ERROR_SEMANTICS.md
package handlers

import (
	"log"
	"net/http"
	"strconv"

	"lifestyle-dashboard/config"
	"lifestyle-dashboard/domain"
	"lifestyle-dashboard/errors"
)

// GetTimeline handles GET /api/v1/stats/timeline - P0-003, P0-004
// [REQ:LD-QUERY-AGGREGATE] Returns event counts grouped by day and domain.
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	if h.Stats == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "stats service not configured"))
		return
	}

	cfg := config.DefaultQueryConfig()
	query := r.URL.Query()
	daysStr := query.Get("days")
	days := cfg.DefaultTimelineDays
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
			// Apply max limit
			if days > cfg.MaxTimelineDays {
				days = cfg.MaxTimelineDays
			}
		}
	}

	timeline, err := h.Stats.GetTimeline(r.Context(), days)
	if err != nil {
		log.Printf("[ERROR] GetTimeline(days=%d): database error: %v", days, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to get timeline. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, domain.TimelineResponse{
		Timeline: timeline,
		Days:     strconv.Itoa(days),
	})
}

// GetSummary handles GET /api/v1/stats/summary - P0-003, P0-004
// [REQ:LD-QUERY-AGGREGATE] Returns aggregated statistics across all domains.
func (h *Handler) GetSummary(w http.ResponseWriter, r *http.Request) {
	if h.Stats == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "stats service not configured"))
		return
	}

	summary, err := h.Stats.GetSummary(r.Context())
	if err != nil {
		log.Printf("[ERROR] GetSummary: database error: %v", err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to get summary. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, summary)
}

// GetScore handles GET /api/v1/stats/score - P0-004, P1-003
// [REQ:LD-UI-SCORE] Returns the daily lifestyle score for dashboard display.
func (h *Handler) GetScore(w http.ResponseWriter, r *http.Request) {
	if h.Stats == nil {
		WriteAPIError(w, errors.NewUnavailableError(errors.CodeDependencyUnavailable, "stats service not configured"))
		return
	}

	cfg := config.DefaultQueryConfig()
	query := r.URL.Query()
	historyDaysStr := query.Get("history_days")
	historyDays := cfg.DefaultTimelineDays
	if historyDaysStr != "" {
		if d, err := strconv.Atoi(historyDaysStr); err == nil && d > 0 {
			historyDays = d
			if historyDays > cfg.MaxTimelineDays {
				historyDays = cfg.MaxTimelineDays
			}
		}
	}

	score, err := h.Stats.GetLifestyleScore(r.Context(), historyDays)
	if err != nil {
		log.Printf("[ERROR] GetScore(history_days=%d): database error: %v", historyDays, err)
		WriteAPIError(w, errors.NewInternalError(errors.CodeDatabaseError, "Failed to calculate lifestyle score. Please try again."))
		return
	}

	WriteJSON(w, http.StatusOK, score)
}
