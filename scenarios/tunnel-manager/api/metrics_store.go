package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// MetricsRecord represents a stored metrics snapshot. [REQ:OBS-001]
type MetricsRecord struct {
	ID            int       `json:"id"`
	HAConnections int       `json:"ha_connections"`
	RequestErrors float64   `json:"request_errors"`
	ActiveStreams int       `json:"active_streams"`
	SmoothedRTT   float64   `json:"smoothed_rtt_ms"`
	ScrapedAt     time.Time `json:"scraped_at"`
}

// metricsColumns is the canonical column list for metrics queries.
const metricsColumns = "id, ha_connections, request_errors, active_streams, smoothed_rtt_ms, scraped_at"

// scanMetricsRecord scans a single MetricsRecord from the given scanner.
func scanMetricsRecord(scanner interface{ Scan(dest ...any) error }) (MetricsRecord, error) {
	var r MetricsRecord
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
func (ms *MetricsStore) Store(m *TunnelMetrics) error {
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
func (ms *MetricsStore) Query(from, to time.Time) ([]MetricsRecord, error) {
	rows, err := ms.db.Query(
		`SELECT `+metricsColumns+` FROM metrics_history WHERE scraped_at >= $1 AND scraped_at <= $2 ORDER BY scraped_at DESC`, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var records []MetricsRecord
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
func (ms *MetricsStore) Latest() (*MetricsRecord, error) {
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

// PurgeOld deletes records older than the retention period.
func (ms *MetricsStore) PurgeOld() (int64, error) {
	return purgeByTimestamp(ms.db, "metrics_history", "scraped_at", ms.retention)
}

// --- HTTP Handlers ---

// handleMetricsHistory returns time-series metrics. [REQ:OBS-001]
func handleMetricsHistory(store *MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hours := 24
		if h := r.URL.Query().Get("hours"); h != "" {
			if v, err := strconv.Atoi(h); err == nil && v > 0 {
				hours = v
			}
		}
		to := time.Now()
		from := to.Add(-time.Duration(hours) * time.Hour)

		records, err := store.Query(from, to)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if records == nil {
			records = []MetricsRecord{}
		}
		writeJSON(w, http.StatusOK, records)
	}
}

// handleMetricsLatest returns the most recent metrics snapshot. [REQ:OBS-001]
func handleMetricsLatest(store *MetricsStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := store.Latest()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if record == nil {
			writeJSON(w, http.StatusOK, map[string]string{"status": "no_data"})
			return
		}
		writeJSON(w, http.StatusOK, record)
	}
}
