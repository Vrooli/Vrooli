package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"architecture-cartographer/internal/clock"
)

// SQLExecutor is the narrow database surface (mirrors conflicts).
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

const migrationTimeFormat = time.RFC3339Nano

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const (
	insertMigrationSQL = `
INSERT INTO migrations (id, scenario, name, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`

	selectMigrationSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM migrations WHERE id = ?`

	listMigrationsSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM migrations
ORDER BY created_at DESC, id DESC`

	listMigrationsByScenarioSQL = `
SELECT id, scenario, name, status, created_at, updated_at
FROM migrations WHERE scenario = ?
ORDER BY created_at DESC, id DESC`

	updateMigrationStatusSQL = `
UPDATE migrations SET status = ?, updated_at = ? WHERE id = ?`

	upsertFindingSQL = `
INSERT INTO migration_findings
  (migration_id, stable_id, scenario, source, code, severity, locations, domains,
   message, suggestion, status, resolution_note, regressed, first_seen_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(migration_id, stable_id) DO UPDATE SET
  scenario = excluded.scenario,
  source = excluded.source,
  code = excluded.code,
  severity = excluded.severity,
  locations = excluded.locations,
  domains = excluded.domains,
  message = excluded.message,
  suggestion = excluded.suggestion,
  status = excluded.status,
  resolution_note = excluded.resolution_note,
  regressed = excluded.regressed,
  updated_at = excluded.updated_at`

	selectFindingSQL = `
SELECT migration_id, stable_id, scenario, source, code, severity, locations, domains,
       message, suggestion, status, resolution_note, regressed, first_seen_at, updated_at
FROM migration_findings WHERE migration_id = ? AND stable_id = ?`

	listFindingsSQL = `
SELECT migration_id, stable_id, scenario, source, code, severity, locations, domains,
       message, suggestion, status, resolution_note, regressed, first_seen_at, updated_at
FROM migration_findings WHERE migration_id = ?
ORDER BY severity, code, stable_id`
)

func (r *sqliteRepository) now() time.Time { return r.clock.Now().UTC() }

func (r *sqliteRepository) CreateMigration(ctx context.Context, m Migration) error {
	if m.CreatedAt.IsZero() {
		m.CreatedAt = r.now()
	}
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = m.CreatedAt
	}
	_, err := r.db.ExecContext(ctx, insertMigrationSQL,
		m.ID, m.Scenario, m.Name, string(m.Status),
		m.CreatedAt.Format(migrationTimeFormat), m.UpdatedAt.Format(migrationTimeFormat))
	return err
}

func (r *sqliteRepository) GetMigration(ctx context.Context, id string) (Migration, error) {
	row := r.db.QueryRowContext(ctx, selectMigrationSQL, id)
	var (
		m                    Migration
		status               string
		createdAt, updatedAt string
	)
	if err := row.Scan(&m.ID, &m.Scenario, &m.Name, &status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Migration{}, ErrMigrationNotFound{ID: id}
		}
		return Migration{}, err
	}
	m.Status = MigrationStatus(status)
	m.CreatedAt, _ = time.Parse(migrationTimeFormat, createdAt)
	m.UpdatedAt, _ = time.Parse(migrationTimeFormat, updatedAt)
	return m, nil
}

func (r *sqliteRepository) ListMigrations(ctx context.Context, scenario string) ([]Migration, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if scenario == "" {
		rows, err = r.db.QueryContext(ctx, listMigrationsSQL)
	} else {
		rows, err = r.db.QueryContext(ctx, listMigrationsByScenarioSQL, scenario)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Migration
	for rows.Next() {
		var (
			m                    Migration
			status               string
			createdAt, updatedAt string
		)
		if err := rows.Scan(&m.ID, &m.Scenario, &m.Name, &status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		m.Status = MigrationStatus(status)
		m.CreatedAt, _ = time.Parse(migrationTimeFormat, createdAt)
		m.UpdatedAt, _ = time.Parse(migrationTimeFormat, updatedAt)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) UpdateMigrationStatus(ctx context.Context, id string, status MigrationStatus) error {
	res, err := r.db.ExecContext(ctx, updateMigrationStatusSQL, string(status), r.now().Format(migrationTimeFormat), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrMigrationNotFound{ID: id}
	}
	return nil
}

func (r *sqliteRepository) UpsertFinding(ctx context.Context, migrationID string, f Finding) error {
	if f.FirstSeenAt.IsZero() {
		f.FirstSeenAt = r.now()
	}
	f.UpdatedAt = r.now()
	locs, err := json.Marshal(nonNil(f.Locations))
	if err != nil {
		return err
	}
	doms, err := json.Marshal(nonNil(f.Domains))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, upsertFindingSQL,
		migrationID, f.StableID, f.Scenario, f.Source, f.Code, f.Severity,
		string(locs), string(doms), f.Message, f.Suggestion, string(f.Status),
		f.ResolutionNote, boolToInt(f.Regressed),
		f.FirstSeenAt.Format(migrationTimeFormat), f.UpdatedAt.Format(migrationTimeFormat))
	return err
}

func (r *sqliteRepository) GetFinding(ctx context.Context, migrationID, stableID string) (Finding, error) {
	row := r.db.QueryRowContext(ctx, selectFindingSQL, migrationID, stableID)
	f, err := scanFinding(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Finding{}, ErrFindingNotFound{MigrationID: migrationID, StableID: stableID}
		}
		return Finding{}, err
	}
	return f, nil
}

func (r *sqliteRepository) ListFindings(ctx context.Context, migrationID string) ([]Finding, error) {
	rows, err := r.db.QueryContext(ctx, listFindingsSQL, migrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Finding
	for rows.Next() {
		f, err := scanFinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// rowScanner unifies *sql.Row and *sql.Rows for scanFinding.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanFinding(row rowScanner) (Finding, error) {
	var (
		f                      Finding
		migrationID            string
		locs, doms             string
		status                 string
		regressed              int
		firstSeenAt, updatedAt string
	)
	if err := row.Scan(&migrationID, &f.StableID, &f.Scenario, &f.Source, &f.Code, &f.Severity,
		&locs, &doms, &f.Message, &f.Suggestion, &status, &f.ResolutionNote, &regressed,
		&firstSeenAt, &updatedAt); err != nil {
		return Finding{}, err
	}
	_ = json.Unmarshal([]byte(locs), &f.Locations)
	_ = json.Unmarshal([]byte(doms), &f.Domains)
	f.Status = FindingStatus(status)
	f.Regressed = regressed != 0
	f.FirstSeenAt, _ = time.Parse(migrationTimeFormat, firstSeenAt)
	f.UpdatedAt, _ = time.Parse(migrationTimeFormat, updatedAt)
	return f, nil
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
