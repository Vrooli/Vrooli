package handlers

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetObservedReceipts reads durable, post-response observations from Vrooli
// Events. Receipt absence is represented explicitly as an empty observation
// set; it is never interpreted as a failed or incomplete Agent Manager run.
func (h *Handler) GetObservedReceipts(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeSimpleError(w, r, "run_id", "invalid UUID format for run ID")
		return
	}
	if _, err := h.svc.GetRun(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeSimpleError(w, r, "limit", "limit must be between 1 and 100")
			return
		}
		limit = parsed
	}
	if !h.receipts.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "unavailable", "observations": []any{}, "message": "vrooli-events observations are not configured"})
		return
	}
	observations, err := h.receipts.ReceiptQuery(r.Context(), id.String(), limit)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"status": "degraded", "observations": []any{}, "message": "vrooli-events observations unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "available", "observations": observations})
}
