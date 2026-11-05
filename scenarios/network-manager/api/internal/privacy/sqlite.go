package privacy

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
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) GetRetention(ctx context.Context) (RetentionSettings, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT query_log_days, snapshot_days, experiment_days, profile, updated_at
FROM retention_settings
WHERE id = ?
`, SettingsID)
	settings, err := scanRetention(row)
	if errors.Is(err, sql.ErrNoRows) {
		settings = DefaultRetention(time.Now().UTC())
		return r.SaveRetention(ctx, settings)
	}
	return settings, err
}

func (r *sqliteRepository) SaveRetention(ctx context.Context, settings RetentionSettings) (RetentionSettings, error) {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = time.Now().UTC()
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO retention_settings (id, query_log_days, snapshot_days, experiment_days, profile, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  query_log_days = excluded.query_log_days,
  snapshot_days = excluded.snapshot_days,
  experiment_days = excluded.experiment_days,
  profile = excluded.profile,
  updated_at = excluded.updated_at
`, SettingsID, settings.QueryLogDays, settings.SnapshotDays, settings.ExperimentDays, settings.Profile, formatTime(settings.UpdatedAt)); err != nil {
		return RetentionSettings{}, fmt.Errorf("save retention settings: %w", err)
	}
	return settings, nil
}

func (r *sqliteRepository) GetVisibility(ctx context.Context) (VisibilitySettings, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT show_query_domains, show_device_history, household_mode, notes_json, updated_at
FROM visibility_settings
WHERE id = ?
`, SettingsID)
	settings, err := scanVisibility(row)
	if errors.Is(err, sql.ErrNoRows) {
		settings = DefaultVisibility(time.Now().UTC())
		return r.SaveVisibility(ctx, settings)
	}
	return settings, err
}

func (r *sqliteRepository) SaveVisibility(ctx context.Context, settings VisibilitySettings) (VisibilitySettings, error) {
	if settings.UpdatedAt.IsZero() {
		settings.UpdatedAt = time.Now().UTC()
	}
	notesJSON, err := encodeStrings(settings.Notes)
	if err != nil {
		return VisibilitySettings{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO visibility_settings (id, show_query_domains, show_device_history, household_mode, notes_json, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  show_query_domains = excluded.show_query_domains,
  show_device_history = excluded.show_device_history,
  household_mode = excluded.household_mode,
  notes_json = excluded.notes_json,
  updated_at = excluded.updated_at
`, SettingsID, boolInt(settings.ShowQueryDomains), boolInt(settings.ShowDeviceHistory), boolInt(settings.HouseholdMode), notesJSON, formatTime(settings.UpdatedAt)); err != nil {
		return VisibilitySettings{}, fmt.Errorf("save visibility settings: %w", err)
	}
	return settings, nil
}

func (r *sqliteRepository) Sweep(ctx context.Context, settings RetentionSettings, now time.Time) (SweepResult, error) {
	result := SweepResult{
		ID:        uuid.NewString(),
		Profile:   settings.Profile,
		CreatedAt: now.UTC(),
		Notes: []string{
			"DNS query-level log table is not implemented; default visibility remains disabled.",
			"Optimization experiment retention is recorded but no optimization storage exists yet.",
		},
	}
	if settings.SnapshotDays > 0 {
		result.SnapshotCutoff = now.UTC().AddDate(0, 0, -int(settings.SnapshotDays))
		res, err := r.db.ExecContext(ctx, `
DELETE FROM network_snapshots
WHERE status <> 'baseline' AND created_at < ?
`, formatTime(result.SnapshotCutoff))
		if err != nil {
			return SweepResult{}, fmt.Errorf("sweep snapshots: %w", err)
		}
		if rows, err := res.RowsAffected(); err == nil {
			result.SnapshotsDeleted = int(rows)
		}
	}
	notesJSON, err := encodeStrings(result.Notes)
	if err != nil {
		return SweepResult{}, err
	}
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO privacy_sweep_records (id, profile, snapshot_cutoff, snapshots_deleted, notes_json, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`, result.ID, result.Profile, formatOptionalTime(result.SnapshotCutoff), result.SnapshotsDeleted, notesJSON, formatTime(result.CreatedAt)); err != nil {
		return SweepResult{}, fmt.Errorf("save privacy sweep record: %w", err)
	}
	return result, nil
}

type retentionScanner interface {
	Scan(dest ...any) error
}

func scanRetention(row retentionScanner) (RetentionSettings, error) {
	var settings RetentionSettings
	var updatedAt string
	if err := row.Scan(&settings.QueryLogDays, &settings.SnapshotDays, &settings.ExperimentDays, &settings.Profile, &updatedAt); err != nil {
		return RetentionSettings{}, err
	}
	var err error
	settings.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return RetentionSettings{}, fmt.Errorf("parse retention updated_at: %w", err)
	}
	return settings, nil
}

func scanVisibility(row retentionScanner) (VisibilitySettings, error) {
	var settings VisibilitySettings
	var showQueryDomains, showDeviceHistory, householdMode int
	var notesJSON, updatedAt string
	if err := row.Scan(&showQueryDomains, &showDeviceHistory, &householdMode, &notesJSON, &updatedAt); err != nil {
		return VisibilitySettings{}, err
	}
	notes, err := decodeStrings(notesJSON)
	if err != nil {
		return VisibilitySettings{}, err
	}
	settings.ShowQueryDomains = showQueryDomains == 1
	settings.ShowDeviceHistory = showDeviceHistory == 1
	settings.HouseholdMode = householdMode == 1
	settings.Notes = notes
	settings.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return VisibilitySettings{}, fmt.Errorf("parse visibility updated_at: %w", err)
	}
	return settings, nil
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

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return formatTime(value)
}
