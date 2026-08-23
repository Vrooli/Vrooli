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

	"github.com/vrooli/api-core/database"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

const schema = `
CREATE TABLE IF NOT EXISTS metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	cycle_id TEXT NOT NULL,
	collector_name TEXT NOT NULL,
	metric_data TEXT NOT NULL,
	observed_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_metrics_observed_at ON metrics(observed_at);
CREATE INDEX IF NOT EXISTS idx_metrics_collector ON metrics(collector_name);
CREATE INDEX IF NOT EXISTS idx_metrics_cycle ON metrics(cycle_id, observed_at);

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
	cpu_seconds REAL NOT NULL DEFAULT 0,
	cpu_seconds_status TEXT NOT NULL DEFAULT 'not_yet_sampled',
	cpu_seconds_reason TEXT NOT NULL DEFAULT '',
	rss_kb INTEGER NOT NULL,
	threads INTEGER NOT NULL,
	gpu_vram_mb REAL NOT NULL DEFAULT 0,
	swap_kb INTEGER NOT NULL DEFAULT 0,
	major_faults_per_second REAL NOT NULL DEFAULT 0,
	metrics_status TEXT NOT NULL DEFAULT '',
	metrics_reason TEXT NOT NULL DEFAULT ''
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
	cpu_seconds REAL NOT NULL DEFAULT 0,
	avg_rss_kb INTEGER NOT NULL,
	max_rss_kb INTEGER NOT NULL,
	avg_major_faults_per_second REAL NOT NULL DEFAULT 0,
	max_major_faults_per_second REAL NOT NULL DEFAULT 0,
	sample_count INTEGER NOT NULL,
	UNIQUE(minute, owner, comm)
);
CREATE INDEX IF NOT EXISTS idx_process_rollups_minute ON process_sample_rollups(minute);
CREATE INDEX IF NOT EXISTS idx_process_rollups_owner_minute ON process_sample_rollups(owner, minute);
`

// Schema returns the SQLite DDL owned by the system-monitor repository.
func Schema() string {
	return schema
}

// Repository implements repository.Repository backed by SQLite.
type Repository struct {
	db   *database.RoutedDB
	mu   sync.RWMutex // Serialize SQLite writes
	thMu sync.RWMutex
	th   map[string]*models.Threshold
}

// NewRepository opens a SQLite database at dbPath and initializes the schema.
func NewRepository(dbPath string) (*Repository, error) {
	// Open via api-core/database so the connection gets retry-with-backoff and
	// jitter (avoids thundering-herd on contended SQLite) instead of a bare
	// sql.Open. MaxOpenConns=1 preserves the single-writer SQLite discipline.
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          dbPath,
		MaxOpenConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite pragmas for performance and correctness.
	primary := db.Primary()
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := primary.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma %s: %w", pragma, err)
		}
	}

	if err := database.EnsureSchemas(context.Background(), primary, database.SchemaProviderFunc(Schema)); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return NewRepositoryFromDB(db), nil
}

// NewRepositoryFromDB wraps an already-open routed database.
func NewRepositoryFromDB(db *database.RoutedDB) *Repository {
	return &Repository{
		db: db,
		th: make(map[string]*models.Threshold),
	}
}

// RoutedDB returns the routed database used by this repository.
func (r *Repository) RoutedDB() *database.RoutedDB {
	return r.db
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

func (r *Repository) SaveMetricCycle(ctx context.Context, cycleID string, observedAt time.Time, observations []repository.MetricObservation) error {
	if cycleID == "" || observedAt.IsZero() {
		return fmt.Errorf("cycle id and observation time are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT 1 FROM metrics WHERE cycle_id = ? LIMIT 1", cycleID).Scan(&exists); err == nil {
		_ = tx.Rollback()
		return fmt.Errorf("metric cycle %q already exists", cycleID)
	} else if err != sql.ErrNoRows {
		_ = tx.Rollback()
		return fmt.Errorf("check metric cycle: %w", err)
	}
	for _, observation := range observations {
		data, err := json.Marshal(observation.Values)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("marshal metrics: %w", err)
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO metrics (cycle_id, collector_name, metric_data, observed_at) VALUES (?, ?, ?, ?)", cycleID, observation.CollectorName, string(data), observedAt.UTC()); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) GetMetrics(ctx context.Context, filter repository.MetricsFilter) ([]*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT cycle_id, collector_name, metric_data, observed_at FROM metrics WHERE 1=1"
	args := []interface{}{}

	if filter.CollectorName != "" {
		query += " AND collector_name = ?"
		args = append(args, filter.CollectorName)
	}
	if !filter.TimeRange.StartTime.IsZero() {
		query += " AND observed_at >= ?"
		args = append(args, filter.TimeRange.StartTime.UTC())
	}
	if !filter.TimeRange.EndTime.IsZero() {
		query += " AND observed_at <= ?"
		args = append(args, filter.TimeRange.EndTime.UTC())
	}
	query += " ORDER BY observed_at ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Group by explicit cycle identity, never by wall-clock coincidence.
	type entry struct {
		CycleID       string
		CollectorName string
		Values        map[string]interface{}
		Timestamp     time.Time
	}
	var entries []entry
	for rows.Next() {
		var e entry
		var data string
		if err := rows.Scan(&e.CycleID, &e.CollectorName, &data, &e.Timestamp); err != nil {
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

	metricsMap := make(map[string]*models.MetricsResponse)
	for _, e := range entries {
		key := e.CycleID
		if key == "" {
			key = e.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		resp, exists := metricsMap[key]
		if !exists {
			resp = &models.MetricsResponse{CycleID: e.CycleID, Timestamp: e.Timestamp}
			metricsMap[key] = resp
		}
		hydrateMetricsResponse(resp, e.CycleID, e.Timestamp, e.CollectorName, e.Values)
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

func (r *Repository) GetLatestMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	resp := &models.MetricsResponse{}
	seen := false

	for _, collector := range []string{"cpu", "memory", "network", "gpu", "disk"} {
		row := r.db.QueryRowContext(ctx,
			"SELECT cycle_id, observed_at, metric_data FROM metrics WHERE collector_name = ? ORDER BY observed_at DESC, id DESC LIMIT 1",
			collector,
		)
		var data string
		var cycleID string
		var observedAt time.Time
		if err := row.Scan(&cycleID, &observedAt, &data); err != nil {
			continue // No data for this collector.
		}
		var values map[string]interface{}
		if err := json.Unmarshal([]byte(data), &values); err != nil {
			continue
		}
		if !seen || observedAt.After(resp.Timestamp) {
			resp.CycleID, resp.Timestamp, seen = cycleID, observedAt, true
		}
		hydrateMetricsResponse(resp, cycleID, observedAt, collector, values)
	}

	// Check if we got any data at all.
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM metrics").Scan(&count); err != nil || count == 0 {
		return nil, apierrors.NotFound("metrics", "latest")
	}
	if resp.Timestamp.IsZero() {
		resp.Timestamp = time.Now().UTC()
	}

	return resp, nil
}

func (r *Repository) GetDetailedMetrics(ctx context.Context, _ repository.TimeRange) (*models.DetailedMetrics, error) {
	return &models.DetailedMetrics{Timestamp: time.Now()}, nil
}

func (r *Repository) GetHistoricalMetrics(ctx context.Context, metricName string, timeRange repository.TimeRange) ([]repository.MetricDataPoint, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx,
		"SELECT metric_data, observed_at FROM metrics WHERE observed_at >= ? AND observed_at <= ? ORDER BY observed_at ASC",
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

func (r *Repository) GetAggregatedMetrics(ctx context.Context, _ repository.AggregationQuery) (map[string]interface{}, error) {
	return map[string]interface{}{
		"average": 50.0,
		"max":     95.0,
		"min":     10.0,
		"count":   100,
	}, nil
}

func (r *Repository) GetEarliestMetricTime(ctx context.Context) (time.Time, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check count first to distinguish empty table from parse issues.
	var count int
	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM metrics").Scan(&count); err != nil || count == 0 {
		return time.Time{}, apierrors.NotFound("metrics", "earliest")
	}

	var raw sql.NullString
	err := r.db.QueryRowContext(ctx, "SELECT MIN(observed_at) FROM metrics").Scan(&raw)
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

func (r *Repository) CreateInvestigation(ctx context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO investigations (id, status, anomaly_id, start_time, end_time, findings, progress, details, steps)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps),
	)
	return err
}

func (r *Repository) GetInvestigation(ctx context.Context, id string) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRowContext(ctx,
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE id = ?", id,
	))
}

func (r *Repository) UpdateInvestigation(ctx context.Context, inv *models.Investigation) error {
	details, _ := json.Marshal(inv.Details)
	steps, _ := json.Marshal(inv.Steps)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`UPDATE investigations SET status=?, anomaly_id=?, start_time=?, end_time=?, findings=?, progress=?, details=?, steps=?
		 WHERE id=?`,
		inv.Status, inv.AnomalyID, inv.StartTime.UTC(), nullTime(inv.EndTime),
		inv.Findings, inv.Progress, string(details), string(steps), inv.ID,
	)
	return err
}

func (r *Repository) ListInvestigations(ctx context.Context, filter repository.InvestigationFilter) ([]*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations WHERE 1=1"
	args := []interface{}{}
	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}
	query += " ORDER BY start_time DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *Repository) GetLatestInvestigation(ctx context.Context) (*models.Investigation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scanInvestigation(r.db.QueryRowContext(ctx,
		"SELECT id, status, anomaly_id, start_time, end_time, findings, progress, details, steps FROM investigations ORDER BY start_time DESC LIMIT 1",
	))
}

func (r *Repository) SaveInvestigationStep(ctx context.Context, investigationID string, step *models.InvestigationStep) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var stepsJSON string
	err := r.db.QueryRowContext(ctx, "SELECT steps FROM investigations WHERE id = ?", investigationID).Scan(&stepsJSON)
	if err != nil {
		return apierrors.NotFound("investigation", investigationID)
	}

	var steps []models.InvestigationStep
	if stepsJSON != "" {
		json.Unmarshal([]byte(stepsJSON), &steps) //nolint:errcheck
	}
	steps = append(steps, *step)

	newSteps, _ := json.Marshal(steps)
	_, err = r.db.ExecContext(ctx, "UPDATE investigations SET steps = ? WHERE id = ?", string(newSteps), investigationID)
	return err
}

// ---------------------------------------------------------------------------
// ReportRepository
// ---------------------------------------------------------------------------

func (r *Repository) CreateReport(ctx context.Context, report *models.Report) error {
	data, _ := json.Marshal(report.Data)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO reports (id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, report.Type, report.GeneratedAt.UTC(),
		report.TimeRange.StartTime.UTC(), report.TimeRange.EndTime.UTC(), report.TimeRange.Duration,
		string(data), report.Format,
	)
	return err
}

func (r *Repository) GetReport(ctx context.Context, id string) (*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var report models.Report
	var data string
	err := r.db.QueryRowContext(ctx,
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

func (r *Repository) ListReports(ctx context.Context, filter repository.ReportFilter) ([]*models.Report, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query := "SELECT id, type, generated_at, time_range_start, time_range_end, time_range_duration, data, format FROM reports WHERE 1=1"
	args := []interface{}{}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	query += " ORDER BY generated_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *Repository) SaveEnhancedReport(ctx context.Context, report *models.EnhancedSystemReport) error {
	data, _ := json.Marshal(report)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		"INSERT OR REPLACE INTO enhanced_reports (report_id, type, generated_at, report_data) VALUES (?, ?, ?, ?)",
		report.ReportID, report.ReportType, report.GeneratedAt.UTC(), string(data),
	)
	return err
}

func (r *Repository) GetEnhancedReport(ctx context.Context, id string) (*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var data string
	err := r.db.QueryRowContext(ctx, "SELECT report_data FROM enhanced_reports WHERE report_id = ?", id).Scan(&data)
	if err != nil {
		return nil, apierrors.NotFound("report", id)
	}

	var report models.EnhancedSystemReport
	if err := json.Unmarshal([]byte(data), &report); err != nil {
		return nil, apierrors.Internal("failed to read report", err)
	}
	return &report, nil
}

func (r *Repository) ListEnhancedReports(ctx context.Context) ([]*models.EnhancedSystemReport, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx, "SELECT report_data FROM enhanced_reports ORDER BY generated_at DESC")
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

func (r *Repository) GetActiveThresholds(ctx context.Context) ([]*models.Threshold, error) {
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

func (r *Repository) GetThreshold(ctx context.Context, metricName string) (*models.Threshold, error) {
	r.thMu.RLock()
	defer r.thMu.RUnlock()

	if t, exists := r.th[metricName]; exists {
		return t, nil
	}
	return nil, apierrors.NotFound("threshold", metricName)
}

func (r *Repository) SaveThreshold(ctx context.Context, threshold *models.Threshold) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	r.th[threshold.MetricName] = threshold
	return nil
}

func (r *Repository) DeleteThreshold(ctx context.Context, metricName string) error {
	r.thMu.Lock()
	defer r.thMu.Unlock()

	delete(r.th, metricName)
	return nil
}

func (r *Repository) SaveThresholdViolation(ctx context.Context, violation *models.ThresholdViolation) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO threshold_violations (metric_name, current_value, threshold_value, severity, violation_type, timestamp, duration, previous_value, trend)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		violation.MetricName, violation.CurrentValue, violation.ThresholdValue,
		violation.Severity, violation.ViolationType, violation.Timestamp.UTC(),
		violation.Duration, violation.PreviousValue, violation.Trend,
	)
	return err
}

func (r *Repository) GetThresholdViolations(ctx context.Context, timeRange repository.TimeRange) ([]*models.ThresholdViolation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx,
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

func (r *Repository) CreateAlert(ctx context.Context, alert *models.Alert) error {
	threshold, _ := json.Marshal(alert.Threshold)
	details, _ := json.Marshal(alert.Details)

	r.mu.Lock()
	defer r.mu.Unlock()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO alerts (id, type, severity, message, metric_name, metric_value, threshold, details, timestamp, acked_at, resolved_at, acked_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		alert.ID, alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
	)
	return err
}

func (r *Repository) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var alert models.Alert
	var threshold, details string
	var ackedAt, resolvedAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
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
	_, err := r.db.ExecContext(ctx,
		`UPDATE alerts SET type=?, severity=?, message=?, metric_name=?, metric_value=?, threshold=?, details=?, timestamp=?, acked_at=?, resolved_at=?, acked_by=?
		 WHERE id=?`,
		alert.Type, alert.Severity, alert.Message,
		alert.MetricName, alert.MetricValue, string(threshold), string(details),
		alert.Timestamp.UTC(), nullTime(alert.AckedAt), nullTime(alert.ResolvedAt), alert.AckedBy,
		alert.ID,
	)
	return err
}

func (r *Repository) ListAlerts(ctx context.Context, filter repository.AlertFilter) ([]*models.Alert, error) {
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

	rows, err := r.db.QueryContext(ctx, query, args...)
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

func (r *Repository) AcknowledgeAlert(ctx context.Context, id string, ackedBy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "UPDATE alerts SET acked_at = ?, acked_by = ? WHERE id = ?",
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

func (r *Repository) ResolveAlert(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx, "UPDATE alerts SET resolved_at = ? WHERE id = ?",
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

func (r *Repository) GetActiveAlerts(ctx context.Context) ([]*models.Alert, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rows, err := r.db.QueryContext(ctx,
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

func hydrateMetricsResponse(resp *models.MetricsResponse, cycleID string, observedAt time.Time, collector string, values map[string]interface{}) {
	state := storedMetricState(cycleID, observedAt, collector, values)
	switch collector {
	case "cpu":
		resp.CPUState = state
		if cpu, ok := values["usage_percent"].(float64); ok {
			resp.CPUUsage = cpu
		}
		resp.CPUContextSwitchesPerSecond = cpuMetricState(cycleID, observedAt, values, "context_switches_per_second")
		resp.CPUInterruptsPerSecond = cpuMetricState(cycleID, observedAt, values, "interrupts_per_second")
		resp.CPUNormalizedLoad1 = cpuMetricState(cycleID, observedAt, values, "normalized_load_1")
		resp.CPUNormalizedLoad5 = cpuMetricState(cycleID, observedAt, values, "normalized_load_5")
		resp.CPURunQueueDepth = cpuMetricState(cycleID, observedAt, values, "run_queue_depth")
		resp.CPUCoreImbalanceIndex = cpuMetricState(cycleID, observedAt, values, "core_imbalance_index")
		resp.CPUModeIowait = cpuModeMetricState(cycleID, observedAt, values, "iowait")
		resp.CPUModeSteal = cpuModeMetricState(cycleID, observedAt, values, "steal")
	case "memory":
		resp.MemoryState = state
		if mem, ok := values["usage_percent"].(float64); ok {
			resp.MemoryUsage = mem
		}
		// Swap rides along in the memory collector's payload. It is projected
		// as its own series because memory utilisation can read healthy while
		// swap fills, and a single memory line cannot show that divergence.
		if swap, ok := values["swap"].(map[string]interface{}); ok {
			if percent, ok := swap["percent"].(float64); ok {
				v := percent
				resp.SwapUsage = &v
				swapState := state
				// The copied state carries the memory reading; swap must
				// overwrite it or every consumer that reads MetricState.Value
				// (the typed MetricValue the UI plots) shows memory twice.
				swapState.Value = percent
				swapState.Provenance = "system-monitor/memory.swap"
				swapState.Units = "percent"
				resp.SwapState = swapState
			}
		}
	case "network":
		resp.ConnectionsState = state
		if tcp, ok := values["tcp_connections"].(float64); ok {
			resp.TCPConnections = int(tcp)
		}
	case "gpu":
		resp.GPUState = state
		if usage, ok := values["total_usage_percent"].(float64); ok {
			v := usage
			resp.GPUUsage = &v
		}
	case "disk":
		resp.DiskState = state
		if usage, ok := values["usage"].(map[string]interface{}); ok {
			if percent, ok := usage["percent"].(float64); ok {
				resp.DiskUsage = percent
			}
		}
	case "pressure":
		resp.CPUStallSomeAvg10 = pressureMetricState(cycleID, observedAt, collector, values, "cpu_psi_some_avg10", "cpu_psi_status", "cpu_psi_reason")
		resp.CPUStallFullAvg10 = pressureMetricState(cycleID, observedAt, collector, values, "cpu_psi_full_avg10", "cpu_psi_status", "cpu_psi_reason")
		resp.SwapTrafficState = pressureMetricState(cycleID, observedAt, collector, values, "swap_traffic_pages_per_second", "swap_traffic_rate_status", "swap_traffic_rate_reason")
		resp.MajorFaultsState = pressureMetricState(cycleID, observedAt, collector, values, "pgmajfault_per_second", "pgmajfault_rate_status", "pgmajfault_rate_reason")
		resp.FragmentationIndexState = pressureMetricState(cycleID, observedAt, collector, values, "fragmentation_max_free_order", "fragmentation_status", "fragmentation_reason")
	}
}

func cpuMetricState(cycleID string, observedAt time.Time, values map[string]interface{}, key string) models.MetricState {
	return pressureMetricState(cycleID, observedAt, "cpu", values, key, key+"_status", key+"_reason")
}

func cpuModeMetricState(cycleID string, observedAt time.Time, values map[string]interface{}, mode string) models.MetricState {
	state := models.MetricState{Status: "not_yet_sampled", Reason: "CPU mode breakdown has not been sampled", CycleID: cycleID, ObservedAt: observedAt, Provenance: "system-monitor/cpu", Units: "percent"}
	raw, ok := values["mode_breakdown"].(map[string]interface{})
	if !ok {
		if typed, typedOK := values["mode_breakdown"].(map[string]float64); typedOK {
			if value, exists := typed[mode]; exists {
				state.Status, state.Value, state.Reason = "measured", value, ""
				return state
			}
		}
		return state
	}
	if value, exists := raw[mode].(float64); exists {
		state.Status, state.Value, state.Reason = "measured", value, ""
		return state
	}
	return state
}

func pressureMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}, valueKey, statusKey, reasonKey string) models.MetricState {
	if _, hasSignalStatus := values[valueKey+"_status"]; hasSignalStatus {
		statusKey = valueKey + "_status"
		reasonKey = valueKey + "_reason"
	}
	state := models.MetricState{Status: "not_yet_sampled", CycleID: cycleID, ObservedAt: observedAt, Provenance: "system-monitor/" + collector}
	if status, ok := values[statusKey].(string); ok && status != "" {
		state.Status = status
	}
	if reason, ok := values[reasonKey].(string); ok {
		state.Reason = reason
	}
	if value, ok := values[valueKey].(float64); ok {
		state.Status, state.Value = "measured", value
		state.Reason = ""
	}
	if state.Status == "unsupported" && state.Reason == "" {
		state.Reason = "metric unsupported on this platform"
	}
	if state.Status == "not_yet_sampled" && state.Reason == "" {
		state.Reason = "rate has not been sampled"
	}
	return state
}

func storedMetricState(cycleID string, observedAt time.Time, collector string, values map[string]interface{}) models.MetricState {
	state := models.MetricState{
		Status:     "failed",
		Reason:     "collector did not return a measurement",
		Provenance: "system-monitor/" + collector,
		Units:      sqliteMetricUnits(collector),
		CycleID:    cycleID,
		ObservedAt: observedAt,
	}
	if status, _ := values["status"].(string); status != "" {
		state.Status = status
	}
	if reason, _ := values["reason"].(string); reason != "" {
		state.Reason = reason
	}
	if source, _ := values["source"].(string); source != "" {
		state.Provenance = source
	}
	if _, explicitStatus := values["status"].(string); !explicitStatus {
		measured := false
		switch collector {
		case "cpu", "memory":
			_, measured = values["usage_percent"].(float64)
		case "network":
			_, measured = values["tcp_connections"].(float64)
		case "gpu":
			_, measured = values["total_usage_percent"].(float64)
		case "disk":
			usage, ok := values["usage"].(map[string]interface{})
			if ok {
				_, measured = usage["percent"].(float64)
			}
		}
		if measured {
			state.Status = "measured"
		}
	}
	if state.Status == "" {
		state.Status = "failed"
	}
	if state.Status == "measured" {
		state.Reason = ""
		switch collector {
		case "cpu", "memory":
			state.Value, _ = values["usage_percent"].(float64)
		case "network":
			if value, ok := values["tcp_connections"].(float64); ok {
				state.Value = value
			} else if value, ok := values["tcp_connections"].(int); ok {
				state.Value = float64(value)
			}
		case "gpu":
			state.Value, _ = values["total_usage_percent"].(float64)
		case "disk":
			if usage, ok := values["usage"].(map[string]interface{}); ok {
				state.Value, _ = usage["percent"].(float64)
			}
		}
	}
	return state
}

func sqliteMetricUnits(collector string) string {
	if collector == "network" {
		return "count"
	}
	return "percent"
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
