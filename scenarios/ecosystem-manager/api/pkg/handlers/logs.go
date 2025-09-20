package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/ecosystem-manager/api/pkg/systemlog"
)

// LogsHandler returns recent aggregated logs.
func LogsHandler(w http.ResponseWriter, r *http.Request) {
	limit := 500
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	entries, err := systemlog.RecentEntries(limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entries": entries,
	})

	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
