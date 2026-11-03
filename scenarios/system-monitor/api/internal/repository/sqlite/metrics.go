package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/apierrors"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/models"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

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

// Hot-path SQL. Every statement here runs on a timer (the CPU threshold source
// calls GetLatestMetrics every 20 s) against a table that reaches gigabytes
// between retention passes, so each one must be served by an index and must
// never touch more rows than it returns. TestMetricsHotQueriesUseIndexes pins
// the query plans.
const (
	// latestMetricForCollectorSQL is one index seek on
	// idx_metrics_collector_observed_at; the id tie-break is the index rowid.
	latestMetricForCollectorSQL = "SELECT cycle_id, observed_at, metric_data FROM metrics WHERE collector_name = ? ORDER BY observed_at DESC, id DESC LIMIT 1"
	// anyMetricExistsSQL answers "is the table empty" from the first row of
	// the primary key instead of counting every row.
	anyMetricExistsSQL = "SELECT EXISTS(SELECT 1 FROM metrics)"
	// earliestMetricSQL is the first entry of idx_metrics_observed_at. MIN
	// over a text expression loses the column affinity, so it is scanned as
	// text and parsed; a NULL result means the table is empty.
	earliestMetricSQL = "SELECT MIN(observed_at) FROM metrics"
)

// metricsWhere renders the filter's WHERE clause. observed_at is stored as a
// fixed-width UTC string, so text comparison is chronological comparison and
// both idx_metrics_observed_at and idx_metrics_collector_observed_at serve the
// range directly.
func metricsWhere(filter repository.MetricsFilter) (string, []interface{}) {
	where := " WHERE 1=1"
	args := []interface{}{}
	if filter.CollectorName != "" {
		where += " AND collector_name = ?"
		args = append(args, filter.CollectorName)
	}
	if !filter.TimeRange.StartTime.IsZero() {
		where += " AND observed_at >= ?"
		args = append(args, filter.TimeRange.StartTime.UTC())
	}
	if !filter.TimeRange.EndTime.IsZero() {
		where += " AND observed_at <= ?"
		args = append(args, filter.TimeRange.EndTime.UTC())
	}
	return where, args
}

// metricsCutoffSQL finds the observation time of the Nth newest cycle that
// matches the filter. All rows of a cycle share one observed_at (SaveMetricCycle
// writes them with a single timestamp), so grouping on observed_at walks the
// index backwards and stops after N distinct cycles. No row returned means the
// filter matches fewer than N cycles and no cutoff is needed.
func metricsCutoffSQL(where string) string {
	return "SELECT observed_at FROM metrics" + where + " GROUP BY observed_at ORDER BY observed_at DESC LIMIT 1 OFFSET ?"
}

// metricsSelectSQL reads every row of the cycles selected by the cutoff.
func metricsSelectSQL(where string) string {
	return "SELECT cycle_id, collector_name, metric_data, observed_at FROM metrics" + where + " ORDER BY observed_at ASC"
}

func (r *Repository) GetMetrics(ctx context.Context, filter repository.MetricsFilter) ([]*models.MetricsResponse, error) {
	if r.compacting.Load() {
		return nil, repository.ErrMaintenance
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	where, args := metricsWhere(filter)
	limit := filter.EffectiveLimit()

	// Bound the read in SQL, not after loading every matching row: keep only
	// the newest `limit` cycles by locating the oldest one to include.
	var cutoff sql.NullString
	if len(filter.SampleBuckets) == 0 {
		cutoffArgs := append(append([]interface{}{}, args...), limit-1)
		if err := r.db.QueryRowContext(ctx, metricsCutoffSQL(where), cutoffArgs...).Scan(&cutoff); err != nil && err != sql.ErrNoRows {
			return nil, err
		}
		if cutoff.Valid {
			// Bind the cutoff as a time so the driver renders it in the stored
			// text layout; the scanned string is RFC3339 and would not compare.
			cutoffAt, err := parseTime(cutoff.String)
			if err != nil {
				return nil, fmt.Errorf("parse metrics cutoff %q: %w", cutoff.String, err)
			}
			where += " AND observed_at >= ?"
			args = append(args, cutoffAt.UTC())
		}
	}

	// Group by explicit cycle identity, never by wall-clock coincidence.
	type entry struct {
		CycleID       string
		CollectorName string
		Values        map[string]interface{}
		Timestamp     time.Time
	}
	var entries []entry
	readRows := func(query string, queryArgs ...interface{}) error {
		rows, err := r.db.QueryContext(ctx, query, queryArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var e entry
			var data string
			if err := rows.Scan(&e.CycleID, &e.CollectorName, &data, &e.Timestamp); err != nil {
				return err
			}
			if err := json.Unmarshal([]byte(data), &e.Values); err != nil {
				return err
			}
			entries = append(entries, e)
		}
		return rows.Err()
	}
	if len(filter.SampleBuckets) == 0 {
		if err := readRows(metricsSelectSQL(where), args...); err != nil {
			return nil, err
		}
	} else {
		// Each inner SELECT is a backward seek on idx_metrics_observed_at.
		// Chunk UNION terms to stay below SQLite's compound-select and bind limits.
		buckets := filter.SampleBuckets
		if len(buckets) > limit {
			buckets = buckets[len(buckets)-limit:]
		}
		for offset := 0; offset < len(buckets); offset += 200 {
			end := min(offset+200, len(buckets))
			parts := make([]string, 0, end-offset)
			queryArgs := append([]interface{}{}, args...)
			for _, bucket := range buckets[offset:end] {
				collector := ""
				if filter.CollectorName != "" {
					collector = " AND collector_name = ?"
				}
				parts = append(parts, "SELECT cycle_id, observed_at FROM (SELECT cycle_id, observed_at FROM metrics WHERE observed_at > ? AND observed_at <= ?"+collector+" ORDER BY observed_at DESC, cycle_id DESC LIMIT 1)")
				queryArgs = append(queryArgs, bucket.StartTime.UTC(), bucket.EndTime.UTC())
				if filter.CollectorName != "" {
					queryArgs = append(queryArgs, filter.CollectorName)
				}
			}
			query := "SELECT cycle_id, collector_name, metric_data, observed_at FROM metrics" + where + " AND (cycle_id, observed_at) IN (" + strings.Join(parts, " UNION ALL ") + ") ORDER BY observed_at ASC"
			if err := readRows(query, queryArgs...); err != nil {
				return nil, err
			}
		}
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
		repository.HydrateMetricObservation(resp, e.CycleID, e.Timestamp, e.CollectorName, e.Values)
	}

	var results []*models.MetricsResponse
	for _, resp := range metricsMap {
		results = append(results, resp)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})

	// The cutoff already bounded the rows; this only guards the degenerate case
	// of two cycles sharing an observation time.
	if len(results) > limit {
		results = results[len(results)-limit:]
	}
	return results, nil
}

func (r *Repository) GetLatestMetrics(ctx context.Context) (*models.MetricsResponse, error) {
	if r.compacting.Load() {
		return nil, repository.ErrMaintenance
	}
	r.mu.RLock()
	defer r.mu.RUnlock()

	resp := &models.MetricsResponse{}
	seen := false

	for _, collector := range []string{"cpu", "memory", "network", "gpu", "disk", "pressure"} {
		row := r.db.QueryRowContext(ctx, latestMetricForCollectorSQL, collector)
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
		repository.HydrateMetricObservation(resp, cycleID, observedAt, collector, values)
	}

	// Only an empty table is "not found"; a table holding just collectors
	// outside the list above still yields an (empty) latest response. That
	// question is answered by the first primary-key row, never by a count.
	if !seen {
		var exists bool
		if err := r.db.QueryRowContext(ctx, anyMetricExistsSQL).Scan(&exists); err != nil || !exists {
			return nil, apierrors.NotFound("metrics", "latest")
		}
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

	// MIN on an indexed column is one index seek; an empty table yields NULL,
	// which is the not-found case. No count is needed to tell them apart.
	var raw sql.NullString
	err := r.db.QueryRowContext(ctx, earliestMetricSQL).Scan(&raw)
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
