package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) ListSchedules(ctx context.Context, includeDisabled bool) ([]Schedule, error) {
	query := `
SELECT id, name, profile, baseline_snapshot_id, interval_minutes, enabled, latency_threshold_ms, unavailable_threshold, effects_json, created_at, updated_at
FROM monitoring_schedules
`
	args := []any{}
	if !includeDisabled {
		query += `WHERE enabled = ?` + "\n"
		args = append(args, 1)
	}
	query += `ORDER BY updated_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list monitoring schedules: %w", err)
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		schedule, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, schedule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitoring schedules: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) GetSchedule(ctx context.Context, id string) (Schedule, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, name, profile, baseline_snapshot_id, interval_minutes, enabled, latency_threshold_ms, unavailable_threshold, effects_json, created_at, updated_at
FROM monitoring_schedules
WHERE id = ?
`, id)
	schedule, err := scanSchedule(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Schedule{}, ErrNotFound
	}
	return schedule, err
}

func (r *sqliteRepository) UpsertSchedule(ctx context.Context, schedule Schedule) (Schedule, error) {
	if schedule.ID == "" {
		schedule.ID = uuid.NewString()
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = schedule.UpdatedAt
	}
	effectsJSON, err := encodeStrings(schedule.Effects)
	if err != nil {
		return Schedule{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO monitoring_schedules (
  id, name, profile, baseline_snapshot_id, interval_minutes, enabled,
  latency_threshold_ms, unavailable_threshold, effects_json, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  name = excluded.name,
  profile = excluded.profile,
  baseline_snapshot_id = excluded.baseline_snapshot_id,
  interval_minutes = excluded.interval_minutes,
  enabled = excluded.enabled,
  latency_threshold_ms = excluded.latency_threshold_ms,
  unavailable_threshold = excluded.unavailable_threshold,
  effects_json = excluded.effects_json,
  updated_at = excluded.updated_at
`, schedule.ID, schedule.Name, schedule.Profile, schedule.BaselineSnapshotID, schedule.IntervalMinutes, boolInt(schedule.Enabled), schedule.LatencyThresholdMS, schedule.UnavailableThreshold, effectsJSON, formatTime(schedule.CreatedAt), formatTime(schedule.UpdatedAt)); err != nil {
		return Schedule{}, fmt.Errorf("upsert monitoring schedule: %w", err)
	}
	return schedule, nil
}

func (r *sqliteRepository) SaveRun(ctx context.Context, run Run) (Run, error) {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	effectsJSON, err := encodeStrings(run.Effects)
	if err != nil {
		return Run{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO monitoring_runs (id, schedule_id, snapshot_id, status, summary, regression_detected, effects_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, run.ID, run.ScheduleID, run.SnapshotID, run.Status, run.Summary, boolInt(run.RegressionDetected), effectsJSON, formatTime(run.CreatedAt)); err != nil {
		return Run{}, fmt.Errorf("save monitoring run: %w", err)
	}
	return run, nil
}

func (r *sqliteRepository) SaveAlert(ctx context.Context, alert Alert) (Alert, error) {
	if alert.ID == "" {
		alert.ID = uuid.NewString()
	}
	evidenceJSON, err := encodeStrings(alert.Evidence)
	if err != nil {
		return Alert{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO monitoring_alerts (id, schedule_id, snapshot_id, severity, status, summary, evidence_json, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
`, alert.ID, alert.ScheduleID, alert.SnapshotID, alert.Severity, alert.Status, alert.Summary, evidenceJSON, formatTime(alert.CreatedAt)); err != nil {
		return Alert{}, fmt.Errorf("save monitoring alert: %w", err)
	}
	return alert, nil
}

func (r *sqliteRepository) ListAlerts(ctx context.Context, scheduleID string, openOnly bool) ([]Alert, error) {
	query := `
SELECT id, schedule_id, snapshot_id, severity, status, summary, evidence_json, created_at
FROM monitoring_alerts
WHERE (? = '' OR schedule_id = ?)
`
	args := []any{scheduleID, scheduleID}
	if openOnly {
		query += `AND status = ?` + "\n"
		args = append(args, "open")
	}
	query += `ORDER BY created_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list monitoring alerts: %w", err)
	}
	defer rows.Close()
	var out []Alert
	for rows.Next() {
		alert, err := scanAlert(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate monitoring alerts: %w", err)
	}
	return out, nil
}

type scheduleScanner interface {
	Scan(dest ...any) error
}

func scanSchedule(row scheduleScanner) (Schedule, error) {
	var schedule Schedule
	var enabled int
	var effectsJSON, createdAt, updatedAt string
	if err := row.Scan(&schedule.ID, &schedule.Name, &schedule.Profile, &schedule.BaselineSnapshotID, &schedule.IntervalMinutes, &enabled, &schedule.LatencyThresholdMS, &schedule.UnavailableThreshold, &effectsJSON, &createdAt, &updatedAt); err != nil {
		return Schedule{}, err
	}
	effects, err := decodeStrings(effectsJSON)
	if err != nil {
		return Schedule{}, err
	}
	schedule.Effects = effects
	schedule.Enabled = enabled == 1
	schedule.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse monitoring schedule created_at: %w", err)
	}
	schedule.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse monitoring schedule updated_at: %w", err)
	}
	return schedule, nil
}

func scanAlert(row scheduleScanner) (Alert, error) {
	var alert Alert
	var evidenceJSON, createdAt string
	if err := row.Scan(&alert.ID, &alert.ScheduleID, &alert.SnapshotID, &alert.Severity, &alert.Status, &alert.Summary, &evidenceJSON, &createdAt); err != nil {
		return Alert{}, err
	}
	evidence, err := decodeStrings(evidenceJSON)
	if err != nil {
		return Alert{}, err
	}
	alert.Evidence = evidence
	alert.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Alert{}, fmt.Errorf("parse monitoring alert created_at: %w", err)
	}
	return alert, nil
}

func encodeStrings(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	b, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("encode string list: %w", err)
	}
	return string(b), nil
}

func decodeStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("decode string list: %w", err)
	}
	return values, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatTime(value time.Time) string {
	return value.UTC().Format(TimeFormat)
}
