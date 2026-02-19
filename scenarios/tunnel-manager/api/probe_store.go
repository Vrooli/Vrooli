package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// StoredProbeResult is a probe result read from the database. [REQ:OBS-002]
type StoredProbeResult struct {
	ID         int       `json:"id"`
	RouteID    int       `json:"route_id"`
	ProbeType  string    `json:"probe_type"`
	Status     string    `json:"status"`
	LatencyMs  *int      `json:"latency_ms"`
	StatusCode *int      `json:"status_code,omitempty"`
	ErrorMsg   *string   `json:"error_msg,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// probeColumns is the canonical column list for probe result queries.
const probeColumns = "id, route_id, probe_type, status, latency_ms, status_code, error_msg, created_at"

// ProbeStore provides query and retention operations for probe history. [REQ:OBS-002]
type ProbeStore struct {
	db        *sql.DB
	retention time.Duration
}

func NewProbeStore(db *sql.DB) *ProbeStore {
	return &ProbeStore{
		db:        db,
		retention: 7 * 24 * time.Hour,
	}
}

// QueryByRoute returns probe results for a specific route within a time range.
func (ps *ProbeStore) QueryByRoute(routeID int, from, to time.Time) ([]StoredProbeResult, error) {
	rows, err := ps.db.Query(
		`SELECT `+probeColumns+` FROM probe_results WHERE route_id = $1 AND created_at >= $2 AND created_at <= $3 ORDER BY created_at DESC`, routeID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("query probes: %w", err)
	}
	defer rows.Close()
	return scanProbeResults(rows)
}

// QueryRecent returns the most recent probe results across all routes.
func (ps *ProbeStore) QueryRecent(limit int) ([]StoredProbeResult, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := ps.db.Query(
		`SELECT `+probeColumns+` FROM probe_results ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query recent probes: %w", err)
	}
	defer rows.Close()
	return scanProbeResults(rows)
}

// PurgeOld deletes probe results older than the retention period.
func (ps *ProbeStore) PurgeOld() (int64, error) {
	return purgeByTimestamp(ps.db, "probe_results", "created_at", ps.retention)
}

func scanProbeResults(rows *sql.Rows) ([]StoredProbeResult, error) {
	var results []StoredProbeResult
	for rows.Next() {
		var r StoredProbeResult
		if err := rows.Scan(&r.ID, &r.RouteID, &r.ProbeType, &r.Status, &r.LatencyMs, &r.StatusCode, &r.ErrorMsg, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan probe: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// --- HTTP Handlers ---

// handleProbeHistory returns recent probe results. [REQ:OBS-002]
func handleProbeHistory(store *ProbeStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if v, err := strconv.Atoi(l); err == nil && v > 0 {
				limit = v
			}
		}

		results, err := store.QueryRecent(limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []StoredProbeResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}
