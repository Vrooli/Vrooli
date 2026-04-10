package store

import (
	"database/sql"
	"fmt"
	"time"

	"tunnel-manager/domain"
)

// metricsColumns is the canonical column list for metrics queries.
const metricsColumns = "id, ha_connections, request_errors, active_streams, smoothed_rtt_ms, scraped_at"

// scanMetricsRecord scans a single MetricsRecord from the given scanner.
func scanMetricsRecord(scanner interface{ Scan(dest ...any) error }) (domain.MetricsRecord, error) {
	var r domain.MetricsRecord
	err := scanner.Scan(&r.ID, &r.HAConnections, &r.RequestErrors, &r.ActiveStreams, &r.SmoothedRTT, &r.ScrapedAt)
	return r, err
}

// MetricsStore persists scraped cloudflared metrics as time-series data
// with configurable retention. [REQ:OBS-001]
type MetricsStore struct {
	db        *sql.DB
	retention time.Duration
}

func NewMetricsStore(db *sql.DB) *MetricsStore {
	return &MetricsStore{
		db:        db,
		retention: 7 * 24 * time.Hour,
	}
}

// Store persists a TunnelMetrics snapshot.
func (ms *MetricsStore) Store(m *domain.TunnelMetrics) error {
	_, err := ms.db.Exec(
		`INSERT INTO metrics_history (ha_connections, request_errors, active_streams, smoothed_rtt_ms) VALUES ($1, $2, $3, $4)`,
		m.HAConnections, m.RequestErrors, m.ActiveStreams, m.SmoothedRTT,
	)
	if err != nil {
		return fmt.Errorf("store metrics: %w", err)
	}
	return nil
}

// Query returns metrics records within a time range.
func (ms *MetricsStore) Query(from, to time.Time) ([]domain.MetricsRecord, error) {
	rows, err := ms.db.Query(
		`SELECT `+metricsColumns+` FROM metrics_history WHERE scraped_at >= $1 AND scraped_at <= $2 ORDER BY scraped_at DESC`, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var records []domain.MetricsRecord
	for rows.Next() {
		r, err := scanMetricsRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan metrics: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Latest returns the most recent metrics record.
func (ms *MetricsStore) Latest() (*domain.MetricsRecord, error) {
	r, err := scanMetricsRecord(ms.db.QueryRow(
		`SELECT ` + metricsColumns + ` FROM metrics_history ORDER BY scraped_at DESC LIMIT 1`,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("latest metrics: %w", err)
	}
	return &r, nil
}

// SetRetention overrides the default retention period (for testing).
func (ms *MetricsStore) SetRetention(d time.Duration) {
	ms.retention = d
}

// PurgeOld deletes records older than the retention period.
func (ms *MetricsStore) PurgeOld() (int64, error) {
	return purgeByTimestamp(ms.db, "metrics_history", "scraped_at", ms.retention)
}
