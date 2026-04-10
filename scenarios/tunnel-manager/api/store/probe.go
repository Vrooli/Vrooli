package store

import (
	"database/sql"
	"fmt"
	"time"

	"tunnel-manager/domain"
)

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

// PersistResult writes a probe result to the database.
func (ps *ProbeStore) PersistResult(pr domain.ProbeResult) error {
	var statusCode *int
	if pr.StatusCode != 0 {
		statusCode = &pr.StatusCode
	}
	var errMsg *string
	if pr.ErrorMsg != "" {
		errMsg = &pr.ErrorMsg
	}
	_, err := ps.db.Exec(
		`INSERT INTO probe_results (route_id, probe_type, status, latency_ms, status_code, error_msg) VALUES ($1, $2, $3, $4, $5, $6)`,
		pr.RouteID, pr.ProbeType, pr.Status, pr.LatencyMs, statusCode, errMsg,
	)
	if err != nil {
		return fmt.Errorf("persist probe result: %w", err)
	}
	return nil
}

// QueryByRoute returns probe results for a specific route within a time range.
func (ps *ProbeStore) QueryByRoute(routeID int, from, to time.Time) ([]domain.StoredProbeResult, error) {
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
func (ps *ProbeStore) QueryRecent(limit int) ([]domain.StoredProbeResult, error) {
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

// SetRetention overrides the default retention period (for testing).
func (ps *ProbeStore) SetRetention(d time.Duration) {
	ps.retention = d
}

// PurgeOld deletes probe results older than the retention period.
func (ps *ProbeStore) PurgeOld() (int64, error) {
	return purgeByTimestamp(ps.db, "probe_results", "created_at", ps.retention)
}

func scanProbeResults(rows *sql.Rows) ([]domain.StoredProbeResult, error) {
	var results []domain.StoredProbeResult
	for rows.Next() {
		var r domain.StoredProbeResult
		if err := rows.Scan(&r.ID, &r.RouteID, &r.ProbeType, &r.Status, &r.LatencyMs, &r.StatusCode, &r.ErrorMsg, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan probe: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}
