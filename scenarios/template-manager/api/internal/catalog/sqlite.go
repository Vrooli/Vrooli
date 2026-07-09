package catalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) *sqliteRepository {
	return &sqliteRepository{db: db}
}

const timeFormat = time.RFC3339Nano

func (r *sqliteRepository) ListTemplates(ctx context.Context, kind TemplateKind) ([]TemplateRecord, error) {
	query := `SELECT id, kind, display_name, version, manifest_path, source_path, tags_json, status, current_version, latest_version, lag_count, updated_at FROM template_records`
	args := []any{}
	if kind != "" {
		query += ` WHERE kind = ?`
		args = append(args, string(kind))
	}
	query += ` ORDER BY kind, id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var out []TemplateRecord
	for rows.Next() {
		record, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate templates: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) GetTemplate(ctx context.Context, id string) (TemplateRecord, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, kind, display_name, version, manifest_path, source_path, tags_json, status, current_version, latest_version, lag_count, updated_at FROM template_records WHERE id = ?`, id)
	record, err := scanTemplate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TemplateRecord{}, ErrNotFound{Kind: "template", ID: id}
	}
	if err != nil {
		return TemplateRecord{}, fmt.Errorf("get template %q: %w", id, err)
	}
	return record, nil
}

func (r *sqliteRepository) SaveValidationRun(ctx context.Context, run ValidationRun) error {
	phases, err := json.Marshal(run.PhaseResults)
	if err != nil {
		return fmt.Errorf("encode validation phases for %q: %w", run.ID, err)
	}
	findings, err := json.Marshal(run.Findings)
	if err != nil {
		return fmt.Errorf("encode validation findings for %q: %w", run.ID, err)
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO validation_runs
  (id, template_id, mode, target, status, started_at, finished_at, phase_results_json, findings_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID,
		run.TemplateID,
		string(run.Mode),
		run.Target,
		run.Status,
		run.StartedAt.UTC().Format(timeFormat),
		run.FinishedAt.UTC().Format(timeFormat),
		string(phases),
		string(findings),
	)
	if err != nil {
		return fmt.Errorf("save validation run %q: %w", run.ID, err)
	}
	trigger := run.Trigger
	if trigger == "" {
		trigger = "manual"
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO validation_run_attributions (run_id, trigger)
VALUES (?, ?)
ON CONFLICT(run_id) DO UPDATE SET trigger = excluded.trigger`,
		run.ID,
		trigger,
	); err != nil {
		return fmt.Errorf("save validation run attribution %q: %w", run.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListValidationRuns(ctx context.Context, templateID string) ([]ValidationRun, error) {
	query := `
SELECT validation_runs.id, template_id, mode, target, status, COALESCE(validation_run_attributions.trigger, 'manual'), started_at, finished_at, phase_results_json, findings_json
FROM validation_runs
LEFT JOIN validation_run_attributions ON validation_run_attributions.run_id = validation_runs.id`
	args := []any{}
	if templateID != "" {
		query += ` WHERE template_id = ?`
		args = append(args, templateID)
	}
	query += ` ORDER BY started_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list validation runs: %w", err)
	}
	defer rows.Close()

	var out []ValidationRun
	for rows.Next() {
		run, err := scanValidationRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate validation runs: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) GetValidationRun(ctx context.Context, id string) (ValidationRun, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT validation_runs.id, template_id, mode, target, status, COALESCE(validation_run_attributions.trigger, 'manual'), started_at, finished_at, phase_results_json, findings_json
FROM validation_runs
LEFT JOIN validation_run_attributions ON validation_run_attributions.run_id = validation_runs.id
WHERE validation_runs.id = ?`, id)
	run, err := scanValidationRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ValidationRun{}, ErrNotFound{Kind: "validation run", ID: id}
	}
	if err != nil {
		return ValidationRun{}, fmt.Errorf("get validation run %q: %w", id, err)
	}
	return run, nil
}

func (r *sqliteRepository) GetMonitorStatus(ctx context.Context) (MonitorStatus, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, enabled, interval_seconds, in_flight, last_run_id, last_status, last_started_at, last_finished_at, next_run_at, green_streak, updated_at FROM monitor_state WHERE id = 'default'`)
	status, err := scanMonitorStatus(row)
	if errors.Is(err, sql.ErrNoRows) {
		return MonitorStatus{}, ErrNotFound{Kind: "monitor status", ID: "default"}
	}
	if err != nil {
		return MonitorStatus{}, fmt.Errorf("get monitor status: %w", err)
	}
	return status, nil
}

func (r *sqliteRepository) SaveMonitorStatus(ctx context.Context, status MonitorStatus) error {
	if status.ID == "" {
		status.ID = "default"
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO monitor_state
  (id, enabled, interval_seconds, in_flight, last_run_id, last_status, last_started_at, last_finished_at, next_run_at, green_streak, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  enabled = excluded.enabled,
  interval_seconds = excluded.interval_seconds,
  in_flight = excluded.in_flight,
  last_run_id = excluded.last_run_id,
  last_status = excluded.last_status,
  last_started_at = excluded.last_started_at,
  last_finished_at = excluded.last_finished_at,
  next_run_at = excluded.next_run_at,
  green_streak = excluded.green_streak,
  updated_at = excluded.updated_at`,
		status.ID,
		boolInt(status.Enabled),
		status.IntervalSeconds,
		boolInt(status.InFlight),
		status.LastRunID,
		status.LastStatus,
		formatOptionalTime(status.LastStartedAt),
		formatOptionalTime(status.LastFinishedAt),
		formatOptionalTime(status.NextRunAt),
		status.GreenStreak,
		formatOptionalTime(status.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("save monitor status: %w", err)
	}
	return nil
}

func (r *sqliteRepository) SaveDriftSnapshot(ctx context.Context, snapshot DriftSnapshot) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO drift_snapshots
  (id, template_id, target, status, drift_count, captured_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		snapshot.ID,
		snapshot.TemplateID,
		snapshot.Target,
		snapshot.Status,
		snapshot.DriftCount,
		snapshot.CapturedAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("save drift snapshot %q: %w", snapshot.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListDriftSnapshots(ctx context.Context, templateID string) ([]DriftSnapshot, error) {
	query := `SELECT id, template_id, target, status, drift_count, captured_at FROM drift_snapshots`
	args := []any{}
	if templateID != "" {
		query += ` WHERE template_id = ?`
		args = append(args, templateID)
	}
	query += ` ORDER BY captured_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list drift snapshots: %w", err)
	}
	defer rows.Close()

	var out []DriftSnapshot
	for rows.Next() {
		snapshot, err := scanDriftSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, snapshot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate drift snapshots: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) UpsertDebt(ctx context.Context, entry DebtEntry) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO debt_entries
  (key, template_id, source, severity, status, title, detail, first_seen_at, last_seen_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(key) DO UPDATE SET
  template_id = excluded.template_id,
  source = excluded.source,
  severity = excluded.severity,
  status = excluded.status,
  title = excluded.title,
  detail = excluded.detail,
  last_seen_at = excluded.last_seen_at`,
		entry.Key,
		entry.TemplateID,
		entry.Source,
		entry.Severity,
		entry.Status,
		entry.Title,
		entry.Detail,
		entry.FirstSeenAt.UTC().Format(timeFormat),
		entry.LastSeenAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("upsert debt %q: %w", entry.Key, err)
	}
	return nil
}

func (r *sqliteRepository) ListDebt(ctx context.Context, templateID, status string) ([]DebtEntry, error) {
	query := `SELECT key, template_id, source, severity, status, title, detail, first_seen_at, last_seen_at FROM debt_entries WHERE 1=1`
	args := []any{}
	if templateID != "" {
		query += ` AND template_id = ?`
		args = append(args, templateID)
	}
	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY severity DESC, key`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list debt: %w", err)
	}
	defer rows.Close()

	var out []DebtEntry
	for rows.Next() {
		entry, err := scanDebtEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate debt: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) GetDebt(ctx context.Context, key string) (DebtEntry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT key, template_id, source, severity, status, title, detail, first_seen_at, last_seen_at FROM debt_entries WHERE key = ?`, key)
	entry, err := scanDebtEntry(row)
	if errors.Is(err, sql.ErrNoRows) {
		return DebtEntry{}, ErrNotFound{Kind: "debt entry", ID: key}
	}
	if err != nil {
		return DebtEntry{}, fmt.Errorf("get debt %q: %w", key, err)
	}
	return entry, nil
}

func (r *sqliteRepository) CountOpenDebt(ctx context.Context, window MeasureWindow) (int64, error) {
	query := `SELECT COUNT(*) FROM debt_entries WHERE status = 'open'`
	args := []any{}
	if !window.From.IsZero() {
		query += ` AND last_seen_at >= ?`
		args = append(args, window.From.UTC().Format(timeFormat))
	}
	if !window.To.IsZero() {
		query += ` AND last_seen_at < ?`
		args = append(args, window.To.UTC().Format(timeFormat))
	}
	var count int64
	if err := r.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count open debt: %w", err)
	}
	return count, nil
}

func (r *sqliteRepository) DeepValidateGreenStreak(ctx context.Context, templateID string) (int64, error) {
	query := `SELECT status FROM validation_runs WHERE mode = 'deep'`
	args := []any{}
	if templateID != "" {
		query += ` AND template_id = ?`
		args = append(args, templateID)
	}
	query += ` ORDER BY finished_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("list deep validation statuses: %w", err)
	}
	defer rows.Close()

	var streak int64
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, fmt.Errorf("scan deep validation status: %w", err)
		}
		if status != "passed" && status != "ok" && status != "success" && status != "green" {
			break
		}
		streak++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate deep validation statuses: %w", err)
	}
	return streak, nil
}

func (r *sqliteRepository) FleetStandingDistribution(ctx context.Context) ([]StandingBucket, error) {
	rows, err := r.db.QueryContext(ctx, `
WITH latest_drift AS (
  SELECT template_id, drift_count
  FROM drift_snapshots d
  WHERE captured_at = (
    SELECT MAX(captured_at) FROM drift_snapshots WHERE template_id = d.template_id
  )
),
open_debt AS (
  SELECT template_id, COUNT(*) AS debt_count
  FROM debt_entries
  WHERE status = 'open'
  GROUP BY template_id
),
standing AS (
  SELECT
    CASE
      WHEN COALESCE(open_debt.debt_count, 0) > 0 THEN 'open_debt'
      WHEN template_records.lag_count > 0 THEN 'version_lag'
      WHEN COALESCE(latest_drift.drift_count, 0) > 0 THEN 'drift'
      ELSE 'current'
    END AS standing
  FROM template_records
  LEFT JOIN latest_drift ON latest_drift.template_id = template_records.id
  LEFT JOIN open_debt ON open_debt.template_id = template_records.id
)
SELECT standing, COUNT(*) FROM standing GROUP BY standing ORDER BY standing`)
	if err != nil {
		return nil, fmt.Errorf("fleet standing distribution: %w", err)
	}
	defer rows.Close()

	var out []StandingBucket
	for rows.Next() {
		var bucket StandingBucket
		if err := rows.Scan(&bucket.Standing, &bucket.Count); err != nil {
			return nil, fmt.Errorf("scan fleet standing bucket: %w", err)
		}
		out = append(out, bucket)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fleet standing buckets: %w", err)
	}
	return out, nil
}

func (r *sqliteRepository) MaxVersionLag(ctx context.Context) (int64, error) {
	var lag int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(lag_count), 0) FROM template_records`).Scan(&lag); err != nil {
		return 0, fmt.Errorf("max version lag: %w", err)
	}
	return lag, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTemplate(s rowScanner) (TemplateRecord, error) {
	var record TemplateRecord
	var tagsRaw, updatedRaw string
	if err := s.Scan(&record.ID, &record.Kind, &record.DisplayName, &record.Version, &record.ManifestPath, &record.SourcePath, &tagsRaw, &record.Status, &record.CurrentVersion, &record.LatestVersion, &record.LagCount, &updatedRaw); err != nil {
		return TemplateRecord{}, err
	}
	if err := json.Unmarshal([]byte(tagsRaw), &record.Tags); err != nil {
		return TemplateRecord{}, fmt.Errorf("decode template tags for %q: %w", record.ID, err)
	}
	updated, err := time.Parse(timeFormat, updatedRaw)
	if err != nil {
		return TemplateRecord{}, fmt.Errorf("parse template updated_at for %q: %w", record.ID, err)
	}
	record.UpdatedAt = updated
	return record, nil
}

func scanValidationRun(s rowScanner) (ValidationRun, error) {
	var run ValidationRun
	var startedRaw, finishedRaw, phasesRaw, findingsRaw string
	if err := s.Scan(&run.ID, &run.TemplateID, &run.Mode, &run.Target, &run.Status, &run.Trigger, &startedRaw, &finishedRaw, &phasesRaw, &findingsRaw); err != nil {
		return ValidationRun{}, err
	}
	if err := json.Unmarshal([]byte(phasesRaw), &run.PhaseResults); err != nil {
		return ValidationRun{}, fmt.Errorf("decode validation phases for %q: %w", run.ID, err)
	}
	if err := json.Unmarshal([]byte(findingsRaw), &run.Findings); err != nil {
		return ValidationRun{}, fmt.Errorf("decode validation findings for %q: %w", run.ID, err)
	}
	started, err := time.Parse(timeFormat, startedRaw)
	if err != nil {
		return ValidationRun{}, fmt.Errorf("parse validation started_at for %q: %w", run.ID, err)
	}
	finished, err := time.Parse(timeFormat, finishedRaw)
	if err != nil {
		return ValidationRun{}, fmt.Errorf("parse validation finished_at for %q: %w", run.ID, err)
	}
	run.StartedAt = started
	run.FinishedAt = finished
	return run, nil
}

func scanDriftSnapshot(s rowScanner) (DriftSnapshot, error) {
	var snapshot DriftSnapshot
	var capturedRaw string
	if err := s.Scan(&snapshot.ID, &snapshot.TemplateID, &snapshot.Target, &snapshot.Status, &snapshot.DriftCount, &capturedRaw); err != nil {
		return DriftSnapshot{}, err
	}
	captured, err := time.Parse(timeFormat, capturedRaw)
	if err != nil {
		return DriftSnapshot{}, fmt.Errorf("parse drift captured_at for %q: %w", snapshot.ID, err)
	}
	snapshot.CapturedAt = captured
	return snapshot, nil
}

func scanDebtEntry(s rowScanner) (DebtEntry, error) {
	var entry DebtEntry
	var firstRaw, lastRaw string
	if err := s.Scan(&entry.Key, &entry.TemplateID, &entry.Source, &entry.Severity, &entry.Status, &entry.Title, &entry.Detail, &firstRaw, &lastRaw); err != nil {
		return DebtEntry{}, err
	}
	first, err := time.Parse(timeFormat, firstRaw)
	if err != nil {
		return DebtEntry{}, fmt.Errorf("parse debt first_seen_at for %q: %w", entry.Key, err)
	}
	last, err := time.Parse(timeFormat, lastRaw)
	if err != nil {
		return DebtEntry{}, fmt.Errorf("parse debt last_seen_at for %q: %w", entry.Key, err)
	}
	entry.FirstSeenAt = first
	entry.LastSeenAt = last
	return entry, nil
}

func scanMonitorStatus(s rowScanner) (MonitorStatus, error) {
	var status MonitorStatus
	var enabled, inFlight int
	var lastStartedRaw, lastFinishedRaw, nextRaw, updatedRaw string
	if err := s.Scan(&status.ID, &enabled, &status.IntervalSeconds, &inFlight, &status.LastRunID, &status.LastStatus, &lastStartedRaw, &lastFinishedRaw, &nextRaw, &status.GreenStreak, &updatedRaw); err != nil {
		return MonitorStatus{}, err
	}
	status.Enabled = enabled != 0
	status.InFlight = inFlight != 0
	var err error
	if status.LastStartedAt, err = parseOptionalTime(lastStartedRaw); err != nil {
		return MonitorStatus{}, fmt.Errorf("parse monitor last_started_at: %w", err)
	}
	if status.LastFinishedAt, err = parseOptionalTime(lastFinishedRaw); err != nil {
		return MonitorStatus{}, fmt.Errorf("parse monitor last_finished_at: %w", err)
	}
	if status.NextRunAt, err = parseOptionalTime(nextRaw); err != nil {
		return MonitorStatus{}, fmt.Errorf("parse monitor next_run_at: %w", err)
	}
	if status.UpdatedAt, err = parseOptionalTime(updatedRaw); err != nil {
		return MonitorStatus{}, fmt.Errorf("parse monitor updated_at: %w", err)
	}
	return status, nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(timeFormat)
}

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeFormat, value)
}
