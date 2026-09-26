package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/findings"
)

// ListFindings returns recurrence-aware investigation findings. It is a
// bounded read surface; recommendation payloads remain intentionally concise.
func (h *Handler) ListFindings(w http.ResponseWriter, r *http.Request) {
	filter := findings.Filter{Fingerprint: strings.TrimSpace(r.URL.Query().Get("fingerprint")), Severity: strings.TrimSpace(r.URL.Query().Get("severity")), Decision: strings.TrimSpace(r.URL.Query().Get("decision")), Limit: 100}
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeSimpleError(w, r, "since", "must be RFC3339")
			return
		}
		filter.Since = &parsed
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeSimpleError(w, r, "limit", "must be between 1 and 200")
			return
		}
		filter.Limit = parsed
	}
	items, err := h.svc.ListFindings(r.Context(), filter)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"findings": items})
}
