package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" database/sql driver

	apidb "github.com/vrooli/api-core/database"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

const schema = `
CREATE TABLE IF NOT EXISTS metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	collector_name TEXT NOT NULL,
	metric_data TEXT NOT NULL,
	timestamp DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp);
CREATE INDEX IF NOT EXISTS idx_metrics_collector ON metrics(collector_name);

CREATE TABLE IF NOT EXISTS investigations (
	id TEXT PRIMARY KEY,
	status TEXT NOT NULL,
	anomaly_id TEXT,
	start_time DATETIME NOT NULL,
	end_time DATETIME,
	findings TEXT,
	progress INTEGER DEFAULT 0,
	details TEXT,
	steps TEXT
);
CREATE INDEX IF NOT EXISTS idx_investigations_status ON investigations(status);

CREATE TABLE IF NOT EXISTS reports (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	generated_at DATETIME NOT NULL,
	time_range_start DATETIME,
	time_range_end DATETIME,
	time_range_duration TEXT,
	data TEXT,
	format TEXT
);

CREATE TABLE IF NOT EXISTS enhanced_reports (
	report_id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	generated_at DATETIME NOT NULL,
	report_data TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS alerts (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	severity TEXT NOT NULL,
	message TEXT NOT NULL,
	metric_name TEXT,
	metric_value REAL,
	threshold TEXT,
	details TEXT,
	timestamp DATETIME NOT NULL,
	acked_at DATETIME,
	resolved_at DATETIME,
	acked_by TEXT
);
CREATE INDEX IF NOT EXISTS idx_alerts_resolved ON alerts(resolved_at);

CREATE TABLE IF NOT EXISTS anomalies (
	id TEXT PRIMARY KEY,
	type TEXT NOT NULL,
	severity TEXT NOT NULL,
	description TEXT,
	metric_data TEXT,
	detected_at DATETIME NOT NULL,
	resolved_at DATETIME,
	status TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS threshold_violations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	metric_name TEXT NOT NULL,
	current_value REAL,
	threshold_value REAL,
	severity TEXT,
	violation_type TEXT,
	timestamp DATETIME NOT NULL,
	duration TEXT,
	previous_value REAL,
	trend TEXT
);
CREATE INDEX IF NOT EXISTS idx_violations_timestamp ON threshold_violations(timestamp);

-- Per-process samples (additive to the opaque metrics blob): one row per
-- process per sampling cycle, the substrate for the attribution timeline.
CREATE TABLE IF NOT EXISTS process_samples (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	ts DATETIME NOT NULL,
	pid INTEGER NOT NULL,
	ppid INTEGER NOT NULL,
	comm TEXT NOT NULL,
	cmdline TEXT,
	cwd TEXT,
	owner TEXT NOT NULL,
	cpu_pct REAL NOT NULL,
	rss_kb INTEGER NOT NULL,
	threads INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_process_samples_ts ON process_samples(ts);
CREATE INDEX IF NOT EXISTS idx_process_samples_owner_ts ON process_samples(owner, ts);

-- Per-owner / per-minute rollups: raw rows older than the raw-retention window
-- are downsampled here so longer windows stay cheap to query and store.
CREATE TABLE IF NOT EXISTS process_sample_rollups (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	minute DATETIME NOT NULL,
	owner TEXT NOT NULL,
	comm TEXT NOT NULL,
	avg_cpu_pct REAL NOT NULL,
	max_cpu_pct REAL NOT NULL,
	avg_rss_kb INTEGER NOT NULL,
	max_rss_kb INTEGER NOT NULL,
	sample_count INTEGER NOT NULL,
	UNIQUE(minute, owner, comm)
);
CREATE INDEX IF NOT EXISTS idx_process_rollups_minute ON process_sample_rollups(minute);
CREATE INDEX IF NOT EXISTS idx_process_rollups_owner_minute ON process_sample_rollups(owner, minute);
`

// Repository implements repository.Repository backed by SQLite.
type Repository struct {
	db   *sql.DB
	mu   sync.RWMutex // Serialize SQLite writes
	thMu sync.RWMutex
	th   map[string]*models.Threshold
}

// NewRepository opens a SQLite database at dbPath and initializes the schema.
func NewRepository(dbPath string) (*Repository, error) {
	// Open via api-core/database so the connection gets retry-with-backoff and
	// jitter (avoids thundering-herd on contended SQLite) instead of a bare
	// sql.Open. MaxOpenConns=1 preserves the single-writer SQLite discipline.
	db, err := apidb.Connect(context.Background(), apidb.Config{
		Driver:       apidb.DriverSQLite,
		DSN:          dbPath,
		MaxOpenConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite pragmas for performance and correctness.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Repository{
		db: db,
		th: make(map[string]*models.Threshold),
	}, nil
}

// NewInMemoryRepository creates a SQLite repository using an in-memory database.
// Uses shared-cache mode so all connections see the same data.
func NewInMemoryRepository() (*Repository, error) {
	return NewRepository("file::memory:?cache=shared")
}

// Close closes the underlying database connection.
func (r *Repository) Close() error {
	return r.db.Close()
}

// ---------------------------------------------------------------------------
// MetricsRepository
// ---------------------------------------------------------------------------

func (r *Repository) SaveMetrics(_ context.Context, collectorName string, metrics map[string]interface{}) error {
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, err = r.db.Exec(
		"INSERT INTO metrics (collector_name, metric_data, timestamp) VALUES (?, ?, ?)",
		collectorName, string(data), time.Now().UTC(),
	)
	return err
}

func (r *Repository) GetMetrics(_ context.Context, filter repository.MetricsFilter) ([]*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT collector_name, metric_data, timestamp FROM metrics WHERE 1=1"
	args := []interface{}{}

	if filter.CollectorName != "" {
		query += " AND collector_name = ?"
		args = append(args, filter.CollectorName)
	}
	if !filter.TimeRange.StartTime.IsZero() {
		query += " AND timestamp >= ?"
		args = append(args, filter.TimeRange.StartTime.UTC())
	}
	if !filter.TimeRange.EndTime.IsZero() {
		query += " AND timestamp <= ?"
		args = append(args, filter.TimeRange.EndTime.UTC())
	}
	query += " ORDER BY timestamp ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group by timestamp like the memory impl does.
	type entry struct {
		CollectorName string
		Values        map[string]interface{}
		Timestamp     time.Time
	}
	var entries []entry
	for rows.Next() {
		var e entry
		var data string
		if err := rows.Scan(&e.CollectorName, &data, &e.Timestamp); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(data), &e.Values); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	metricsMap := make(map[time.Time]*models.MetricsResponse)
	for _, e := range entries {
		resp, exists := metricsMap[e.Timestamp]
		if !exists {
			resp = &models.MetricsResponse{Timestamp: e.Timestamp}
			metricsMap[e.Timestamp] = resp
		}
		hydrateMetricsResponse(resp, e.CollectorName, e.Values)
	}

	var results []*models.MetricsResponse
	for _, resp := range metricsMap {
		results = append(results, resp)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})

	if filter.Limit > 0 && len(results) > filter.Limit {
		results = results[len(results)-filter.Limit:]
	}
	return results, nil
}

func (r *Repository) GetLatestMetrics(_ context.Context) (*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resp := &models.MetricsResponse{Timestamp: time.Now()}

	for _, collector := range []string{"cpu", "memory", "network", "gpu"} {
		row := r.db.QueryRow(
			"SELECT metric_data FROM metrics WHERE collector_name = ? ORDER BY timestamp DESC LIMIT 1",
			collector,
		)
		var data string
		if err := row.Scan(&data); err != nil {
			continue // No data for this collector.
		}
		var values map[string]interface{}
		if err := json.Unmarshal([]byte(data), &values); err != nil {
			continue
		}
		hydrateMetricsResponse(resp, collector, values)
	}

	// Check if we got any data at all.
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&count); err != nil || count == 0 {
		return nil, apierrors.NotFound("metrics", "latest")
	}

	return resp, nil
}

func (r *Repository) GetDetailedMetrics(_ context.Context, _ repository.TimeRange) (*models.DetailedMetrics, error) {
	return &models.DetailedMetrics{Timestamp: time.Now()}, nil
}

func (r *Repository) GetHistoricalMetrics(_ context.Context, metricName string, timeRange repository.TimeRange) ([]repository.MetricDataPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query(
		"SELECT metric_data, timestamp FROM metrics WHERE timestamp >= ? AND timestamp <= ? ORDER BY timestamp ASC",
		timeRange.StartTime.UTC(), timeRange.EndTime.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []repository.MetricDataPoint
	for rows.Next() {
		var data string
		var ts time.Time
		if err := rows.Scan(&data, &ts); err != nil {
			return nil, err
		}
		var values map[string]interface{}
		if err := json.Unmarshal([]byte(data), &values); err != nil {
			continue
		}
		if val, ok := values[metricName].(float64); ok {
			points = append(points, repository.MetricDataPoint{
				Timestamp: ts,
				Value:     val,
			})
		}
	}
	return points, rows.Err()
}

func (r *Repository) GetAggregatedMetrics(_ context.Context, _ repository.AggregationQuery) (map[string]interface{}, error) {
	return map[string]interface{}{
		"average": 50.0,
		"max":     95.0,
		"min":     10.0,
		"count":   100,
	}, nil
}

func (r *Repository) GetEarliestMetricTime(_ context.Context) (time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check count first to distinguish empty table from parse issues.
	var count int
	if err := r.db.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&count); err != nil || count == 0 {
		return time.Time{}, apierrors.NotFound("metrics", "earliest")
	}

	var raw sql.NullString
	err := r.db.QueryRow("SELECT MIN(timestamp) FROM metrics").Scan(&raw)
	if err != nil || !raw.Valid || raw.String == "" {
		return time.Time{}, apierrors.NotFound("metrics", "earliest")
	}
	ts, err := parseTime(raw.String)
	if err != nil {
		return time.Time{}, apierrors.NotFound("metrics", "earliest")
	}
	return ts, nil
}

// ---------------------------------------------------------------------------
// InvestigationRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateInvestigation(_ context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		`INSERT INTO investigations (id, status, anomaly_id, start_time, end_time, findings, progress, details, steps)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps),
	)
	return err
}

func (r *Repository) GetInvestigation(_ context.Context, id string) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRow(
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE id = ?", id,
	))
}

func (r *Repository) UpdateInvestigation(_ context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		`UPDATE investigations SET status=?, anomaly_id=?, start_time=?, end_time=?, findings=?, progress=?, details=?, steps=?
		 WHERE id=?`,
		inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps), inv.ID,
	)
	return err
}

func (r *Repository) ListInvestigations(_ context.Context, filter repository.InvestigationFilter) ([]*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	query += " ORDER BY start_time DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Investigation
	for rows.Next() {
		inv, err := r.scanInvestigationRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, inv)
	}
	return results, rows.Err()
}

func (r *Repository) GetLatestInvestigation(_ context.Context) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRow(
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations ORDER BY start_time DESC LIMIT 1",
	))
}

func (r *Repository) SaveInvestigationStep(_ context.Context, investigationID string, step *models.InvestigationStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var stepsJSON string
	err := r.db.QueryRow("SELECT steps FROM investigations WHERE id = ?", investigationID).Scan(&stepsJSON)
	if err != nil {
		return apierrors.NotFound("investigation", investigationID)
	}

	var steps []models.InvestigationStep
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &steps) //nolint:errcheck
	}
	steps = append(steps, *step)

	newSteps, _ := json.Marshal(steps)
	_, err = r.db.Exec("UPDATE investigations SET steps = ? WHERE id = ?", string(newSteps), investigationID)
	return err
}

// ---------------------------------------------------------------------------
// ReportRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateReport(_ context.Context, report *models.Report) error {
	data, _ := json.Marshal(report.Data)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		`INSERT INTO reports (id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.Type, report.GeneratedAt.UTC(),
		report.TimeRange.StartTime.UTC(), report.TimeRange.EndTime.UTC(), report.TimeRange.Duration,
		string(data), report.Format,
	)
	return err
}

func (r *Repository) GetReport(_ context.Context, id string) (*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var report models.Report
	var data string
	err := r.db.QueryRow(
		"SELECT id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format FROM reports WHERE id = ?", id,
	).Scan(&report.ID, &report.Type, &report.GeneratedAt,
		&report.TimeRange.StartTime, &report.TimeRange.EndTime, &report.TimeRange.Duration,
		&data, &report.Format,
	)
	if err != nil {
		return nil, apierrors.NotFound("report", id)
	}
	json.Unmarshal([]byte(data), &report.Data) //nolint:errcheck
	return &report, nil
}

func (r *Repository) ListReports(_ context.Context, filter repository.ReportFilter) ([]*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format FROM reports WHERE 1=1"
	args := []interface{}{}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	query += " ORDER BY generated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Report
	for rows.Next() {
		var report models.Report
		var data string
		if err := rows.Scan(&report.ID, &report.Type, &report.GeneratedAt,
			&report.TimeRange.StartTime, &report.TimeRange.EndTime, &report.TimeRange.Duration,
			&data, &report.Format,
		); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(data), &report.Data) //nolint:errcheck
		results = append(results, &report)
	}
	return results, rows.Err()
}

func (r *Repository) SaveDetailedReport(ctx context.Context, report *models.DetailedSystemReport) error {
	basicReport := &models.Report{
		ID:          report.ReportID,
		Type:        report.ReportType,
		GeneratedAt: report.GeneratedAt,
		TimeRange: models.ReportTimeRange{
			StartTime: report.TimeRange.StartTime,
			EndTime:   report.TimeRange.EndTime,
			Duration:  report.TimeRange.Duration,
		},
		Data: map[string]interface{}{"detailed": report},
	}
	return r.CreateReport(ctx, basicReport)
}

func (r *Repository) GetDetailedReport(ctx context.Context, id string) (*models.DetailedSystemReport, error) {
	report, err := r.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}

	if detailed, ok := report.Data["detailed"]; ok {
		// Re-marshal and unmarshal to get proper type.
		b, err := json.Marshal(detailed)
		if err != nil {
			return nil, apierrors.Internal("failed to read report", err)
		}
		var result models.DetailedSystemReport
		if err := json.Unmarshal(b, &result); err != nil {
			return nil, apierrors.Internal("failed to read report", err)
		}
		return &result, nil
	}

	return nil, apierrors.NotFound("report", id)
}

func (r *Repository) SaveEnhancedReport(_ context.Context, report *models.EnhancedSystemReport) error {
	data, _ := json.Marshal(report)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		"INSERT OR REPLACE INTO enhanced_reports (report_id, type, generated_at, report_data) VALUES (?, ?, ?, ?)",
		report.ReportID, report.ReportType, report.GeneratedAt.UTC(), string(data),
	)
	return err
}

func (r *Repository) GetEnhancedReport(_ context.Context, id string) (*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var data string
	err := r.db.QueryRow("SELECT report_data FROM enhanced_reports WHERE report_id = ?", id).Scan(&data)
	if err != nil {
		return nil, apierrors.NotFound("report", id)
	}

	var report models.EnhancedSystemReport
	if err := json.Unmarshal([]byte(data), &report); err != nil {
		return nil, apierrors.Internal("failed to read report", err)
	}
	return &report, nil
}

func (r *Repository) ListEnhancedReports(_ context.Context) ([]*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query("SELECT report_data FROM enhanced_reports ORDER BY generated_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.EnhancedSystemReport
	for rows.Next() {
		var data string
		if err := rows.Scan(&data); err != nil {
			return nil, err
		}
		var report models.EnhancedSystemReport
		if err := json.Unmarshal([]byte(data), &report); err != nil {
			continue
		}
		results = append(results, &report)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// ThresholdRepository (in-memory)
// ---------------------------------------------------------------------------

func (r *Repository) GetActiveThresholds(_ context.Context) ([]*models.Threshold, error) {
	r.thMu.RLock()
	defer r.thMu.RUnlock()

	var results []*models.Threshold
	for _, t := range r.th {
		if t.Enabled {
			results = append(results, t)
		}
	}

	if len(results) == 0 {
		results = repository.DefaultThresholds()
	}

	return results, nil
}

func (r *Repository) GetThreshold(_ context.Context, metricName string) (*models.Threshold, error) {
	r.thMu.RLock()
	defer r.thMu.RUnlock()

	if t, exists := r.th[metricName]; exists {
		return t, nil
	}
	return nil, apierrors.NotFound("threshold", metricName)
}

func (r *Repository) SaveThreshold(_ context.Context, threshold *models.Threshold) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	r.th[threshold.MetricName] = threshold
	return nil
}

func (r *Repository) DeleteThreshold(_ context.Context, metricName string) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	delete(r.th, metricName)
	return nil
}

func (r *Repository) SaveThresholdViolation(_ context.Context, violation *models.ThresholdViolation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.Exec(
		`INSERT INTO threshold_violations (metric_name, current_value, threshold_value, severity, violation_type, timestamp, duration, previous_value, trend)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		violation.MetricName, violation.CurrentValue, violation.ThresholdValue,
		violation.Severity, violation.ViolationType, violation.Timestamp.UTC(),
		violation.Duration, violation.PreviousValue, violation.Trend,
	)
	return err
}

func (r *Repository) GetThresholdViolations(_ context.Context, timeRange repository.TimeRange) ([]*models.ThresholdViolation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query(
		`SELECT metric_name, current_value, threshold_value, severity, violation_type, timestamp, duration, previous_value, trend
		 FROM threshold_violations WHERE timestamp > ? AND timestamp < ? ORDER BY timestamp ASC`,
		timeRange.StartTime.UTC(), timeRange.EndTime.UTC(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.ThresholdViolation
	for rows.Next() {
		var v models.ThresholdViolation
		if err := rows.Scan(&v.MetricName, &v.CurrentValue, &v.ThresholdValue,
			&v.Severity, &v.ViolationType, &v.Timestamp,
			&v.Duration, &v.PreviousValue, &v.Trend,
		); err != nil {
			return nil, err
		}
		results = append(results, &v)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// AlertRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateAlert(_ context.Context, alert *models.Alert) error {
	threshold, _ := json.Marshal(alert.Threshold)
	details, _ := json.Marshal(alert.Details)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		`INSERT INTO alerts (id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		alert.ID, alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
	)
	return err
}

func (r *Repository) GetAlert(_ context.Context, id string) (*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alert models.Alert
	var threshold, details string
	var ackedAt, resolvedAt sql.NullTime
	err := r.db.QueryRow(
		"SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE id = ?", id,
	).Scan(&alert.ID, &alert.Type, &alert.Severity, &alert.Message,
		&alert.MetricName, &alert.MetricValue, &threshold, &details,
		&alert.Timestamp, &ackedAt, &resolvedAt, &alert.AckedBy,
	)
	if err != nil {
		return nil, apierrors.NotFound("alert", id)
	}

	if threshold != "" && threshold != "null" {
		var th models.Threshold
		if json.Unmarshal([]byte(threshold), &th) == nil {
			alert.Threshold = &th
		}
	}
	json.Unmarshal([]byte(details), &alert.Details) //nolint:errcheck

	if ackedAt.Valid {
		alert.AckedAt = &ackedAt.Time
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}

	return &alert, nil
}

func (r *Repository) UpdateAlert(ctx context.Context, alert *models.Alert) error {
	threshold, _ := json.Marshal(alert.Threshold)
	details, _ := json.Marshal(alert.Details)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.Exec(
		`UPDATE alerts SET type=?, severity=?, message=?, metric_name=?, metric_value=?, threshold=?, details=?, timestamp=?, acked_at=?, resolved_at=?, acked_by=?
		 WHERE id=?`,
		alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
		alert.ID,
	)
	return err
}

func (r *Repository) ListAlerts(_ context.Context, filter repository.AlertFilter) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE 1=1"
	args := []interface{}{}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.Severity != "" {
		query += " AND severity = ?"
		args = append(args, filter.Severity)
	}
	query += " ORDER BY timestamp DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Alert
	for rows.Next() {
		alert, err := r.scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, alert)
	}
	return results, rows.Err()
}

func (r *Repository) AcknowledgeAlert(_ context.Context, id string, ackedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.Exec("UPDATE alerts SET acked_at = ?, acked_by = ? WHERE id = ?",
		time.Now().UTC(), ackedBy, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apierrors.NotFound("alert", id)
	}
	return nil
}

func (r *Repository) ResolveAlert(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.Exec("UPDATE alerts SET resolved_at = ? WHERE id = ?",
		time.Now().UTC(), id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return apierrors.NotFound("alert", id)
	}
	return nil
}

func (r *Repository) GetActiveAlerts(_ context.Context) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.Query(
		"SELECT id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by FROM alerts WHERE resolved_at IS NULL ORDER BY timestamp DESC",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*models.Alert
	for rows.Next() {
		alert, err := r.scanAlertRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, alert)
	}
	return results, rows.Err()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseTime tries multiple formats to handle SQLite's string timestamp storage.
func parseTime(s string) (time.Time, error) {
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999 +0000 UTC",
		"2006-01-02 15:04:05.999999999+00:00",
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable time: %s", s)
}

func hydrateMetricsResponse(resp *models.MetricsResponse, collector string, values map[string]interface{}) {
	switch collector {
	case "cpu":
		if cpu, ok := values["usage_percent"].(float64); ok {
			resp.CPUUsage = cpu
		}
	case "memory":
		if mem, ok := values["usage_percent"].(float64); ok {
			resp.MemoryUsage = mem
		}
	case "network":
		if tcp, ok := values["tcp_connections"].(float64); ok {
			resp.TCPConnections = int(tcp)
		}
	case "gpu":
		if usage, ok := values["total_usage_percent"].(float64); ok {
			v := usage
			resp.GPUUsage = &v
		}
	}
}

func nullTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return t.UTC()
}

func (r *Repository) scanInvestigation(row *sql.Row) (*models.Investigation, error) {
	var inv models.Investigation
	var endTime sql.NullTime
	var details, steps string

	err := row.Scan(&inv.ID, &inv.Status, &inv.AnomalyID, &inv.StartTime, &endTime,
		&inv.Findings, &inv.Progress, &details, &steps,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("investigation: %w", repository.ErrNotFound)
		}
		return nil, err
	}

	if endTime.Valid {
		inv.EndTime = &endTime.Time
	}
	json.Unmarshal([]byte(details), &inv.Details) //nolint:errcheck
	json.Unmarshal([]byte(steps), &inv.Steps)     //nolint:errcheck

	return &inv, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *Repository) scanInvestigationRow(row rowScanner) (*models.Investigation, error) {
	var inv models.Investigation
	var endTime sql.NullTime
	var details, steps string

	err := row.Scan(&inv.ID, &inv.Status, &inv.AnomalyID, &inv.StartTime, &endTime,
		&inv.Findings, &inv.Progress, &details, &steps,
	)
	if err != nil {
		return nil, err
	}

	if endTime.Valid {
		inv.EndTime = &endTime.Time
	}
	json.Unmarshal([]byte(details), &inv.Details) //nolint:errcheck
	json.Unmarshal([]byte(steps), &inv.Steps)     //nolint:errcheck

	return &inv, nil
}

func (r *Repository) scanAlertRow(row rowScanner) (*models.Alert, error) {
	var alert models.Alert
	var threshold, details string
	var ackedAt, resolvedAt sql.NullTime

	err := row.Scan(&alert.ID, &alert.Type, &alert.Severity, &alert.Message,
		&alert.MetricName, &alert.MetricValue, &threshold, &details,
		&alert.Timestamp, &ackedAt, &resolvedAt, &alert.AckedBy,
	)
	if err != nil {
		return nil, err
	}

	if threshold != "" && threshold != "null" {
		var th models.Threshold
		if json.Unmarshal([]byte(threshold), &th) == nil {
			alert.Threshold = &th
		}
	}
	json.Unmarshal([]byte(details), &alert.Details) //nolint:errcheck

	if ackedAt.Valid {
		alert.AckedAt = &ackedAt.Time
	}
	if resolvedAt.Valid {
		alert.ResolvedAt = &resolvedAt.Time
	}

	return &alert, nil
}
