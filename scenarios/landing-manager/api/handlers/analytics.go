package handlers

import (
	"net/http"
)

// HandleAnalyticsSummary returns aggregate generation analytics
func (h *Handler) HandleAnalyticsSummary(w http.ResponseWriter, r *http.Request) {
	if h.AnalyticsService == nil {
		h.RespondError(w, http.StatusServiceUnavailable, "Analytics service not available")
		return
	}

	summary := h.AnalyticsService.GetSummary()
	h.RespondJSON(w, http.StatusOK, summary)
}

// HandleAnalyticsEvents returns detailed generation event history
func (h *Handler) HandleAnalyticsEvents(w http.ResponseWriter, r *http.Request) {
	if h.AnalyticsService == nil {
		h.RespondError(w, http.StatusServiceUnavailable, "Analytics service not available")
		return
	}

	events := h.AnalyticsService.GetEvents()
	h.RespondJSON(w, http.StatusOK, events)
}
