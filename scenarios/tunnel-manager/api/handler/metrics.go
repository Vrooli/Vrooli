package handler

import (
	"net/http"
	"strconv"
	"time"

	"tunnel-manager/domain"
)

// HandleMetricsHistory returns time-series metrics. [REQ:OBS-001]
func HandleMetricsHistory(q MetricsQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hours := 24
		if h := r.URL.Query().Get("hours"); h != "" {
			if v, err := strconv.Atoi(h); err == nil && v > 0 {
				hours = v
			}
		}
		if hours > maxQueryHours {
			hours = maxQueryHours
		}
		to := time.Now()
		from := to.Add(-time.Duration(hours) * time.Hour)

		records, err := q.Query(from, to)
		if err != nil {
			writeError(w, err)
			return
		}
		if records == nil {
			records = []domain.MetricsRecord{}
		}
		writeJSON(w, http.StatusOK, records)
	}
}

// HandleMetricsLatest returns the most recent metrics snapshot. [REQ:OBS-001]
func HandleMetricsLatest(q MetricsQuerier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := q.Latest()
		if err != nil {
			writeError(w, err)
			return
		}
		if record == nil {
			writeJSON(w, http.StatusOK, map[string]string{"status": "no_data"})
			return
		}
		writeJSON(w, http.StatusOK, record)
	}
}
