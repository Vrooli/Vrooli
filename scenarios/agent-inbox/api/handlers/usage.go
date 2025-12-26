package handlers

import (
	"net/http"
	"strconv"
	"time"
)

// GetUsageStats returns aggregated usage statistics.
// Query params:
//   - start: Start date (YYYY-MM-DD)
//   - end: End date (YYYY-MM-DD)
func (h *Handlers) GetUsageStats(w http.ResponseWriter, r *http.Request) {
	var startDate, endDate *time.Time

	// Parse optional date filters
	if start := r.URL.Query().Get("start"); start != "" {
		if t, err := time.Parse("2006-01-02", start); err == nil {
			startDate = &t
		}
	}
	if end := r.URL.Query().Get("end"); end != "" {
		if t, err := time.Parse("2006-01-02", end); err == nil {
			// Add one day to include the end date in results
			t = t.AddDate(0, 0, 1)
			endDate = &t
		}
	}

	stats, err := h.Repo.GetUsageStats(r.Context(), startDate, endDate)
	if err != nil {
		h.JSONError(w, "Failed to get usage stats", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, stats, http.StatusOK)
}

// GetChatUsageStats returns usage statistics for a specific chat.
func (h *Handlers) GetChatUsageStats(w http.ResponseWriter, r *http.Request) {
	chatID := h.ParseUUID(w, r, "id")
	if chatID == "" {
		return
	}

	stats, err := h.Repo.GetChatUsageStats(r.Context(), chatID)
	if err != nil {
		h.JSONError(w, "Failed to get chat usage stats", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, stats, http.StatusOK)
}

// GetUsageRecords returns usage records with pagination.
// Query params:
//   - chat_id: Filter by chat ID (optional)
//   - limit: Max records to return (default 50)
//   - offset: Records to skip
func (h *Handlers) GetUsageRecords(w http.ResponseWriter, r *http.Request) {
	chatID := r.URL.Query().Get("chat_id")

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	records, err := h.Repo.GetUsageRecords(r.Context(), chatID, limit, offset)
	if err != nil {
		h.JSONError(w, "Failed to get usage records", http.StatusInternalServerError)
		return
	}

	h.JSONResponse(w, records, http.StatusOK)
}
