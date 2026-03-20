package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"tunnel-manager/domain"
)

func HandleRunProbes(runner ProbeRunner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		results, err := runner.RunAll(ctx)
		if err != nil {
			writeError(w, err)
			return
		}
		if results == nil {
			results = []domain.ProbeResult{}
		}

		// Compute summary
		up, down := 0, 0
		for _, pr := range results {
			if pr.Status == "up" {
				up++
			} else {
				down++
			}
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"results": results,
			"summary": map[string]int{
				"total": len(results),
				"up":    up,
				"down":  down,
			},
		})
	}
}

// HandleProbeHistory returns recent probe results. [REQ:OBS-002]
func HandleProbeHistory(reader ProbeHistoryReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}
		if limit > maxQueryLimit {
			limit = maxQueryLimit
		}

		results, err := reader.QueryRecent(limit)
		if err != nil {
			writeError(w, err)
			return
		}
		if results == nil {
			results = []domain.StoredProbeResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}
