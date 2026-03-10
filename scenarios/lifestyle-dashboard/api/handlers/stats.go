package handlers

import (
	"log"
	"net/http"
	"strconv"

	"lifestyle-dashboard/domain"
)

// GetTimeline handles GET /api/v1/stats/timeline - P0-003, P0-004
// [REQ:LD-QUERY-AGGREGATE] Returns event counts grouped by day and domain.
func (h *Handler) GetTimeline(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	daysStr := query.Get("days")
	days := 7
	if daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			days = d
		}
	}

	timeline, err := h.Stats.GetTimeline(r.Context(), days)
	if err != nil {
		log.Printf("Error getting timeline: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to get timeline")
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
	summary, err := h.Stats.GetSummary(r.Context())
	if err != nil {
		log.Printf("Error getting summary: %v", err)
		WriteError(w, http.StatusInternalServerError, "failed to get summary")
		return
	}

	WriteJSON(w, http.StatusOK, summary)
}
