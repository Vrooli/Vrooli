package snapshot

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

func (r *sqliteRepository) Create(ctx context.Context, s Snapshot) (Snapshot, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now().UTC()
	}
	findings, err := json.Marshal(s.Findings)
	if err != nil {
		return Snapshot{}, fmt.Errorf("marshal findings: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO network_snapshots (id, status, profile, summary, findings_json, created_at)
VALUES (?, ?, ?, ?, ?, ?)
`, s.ID, s.Status, s.Profile, s.Summary, string(findings), s.CreatedAt.UTC().Format(TimeFormat))
	if err != nil {
		return Snapshot{}, fmt.Errorf("insert snapshot %q: %w", s.ID, err)
	}
	for i, m := range s.Metrics {
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO snapshot_probe_results (id, snapshot_id, name, value, unit, status, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?)
`, uuid.NewString(), s.ID, m.Name, m.Value, m.Unit, m.Status, i); err != nil {
			return Snapshot{}, fmt.Errorf("insert snapshot metric %q: %w", m.Name, err)
		}
	}
	return s, nil
}

func (r *sqliteRepository) List(ctx context.Context) ([]Snapshot, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT id, status, profile, summary, findings_json, created_at
FROM network_snapshots
ORDER BY created_at DESC, id DESC
`)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var out []Snapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshots: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close snapshot rows: %w", err)
	}
	for i := range out {
		metrics, err := r.metrics(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Metrics = metrics
	}
	return out, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Snapshot, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, status, profile, summary, findings_json, created_at
FROM network_snapshots
WHERE id = ?
`, id)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Snapshot{}, ErrNotFound
	}
	if err != nil {
		return Snapshot{}, err
	}
	s.Metrics, err = r.metrics(ctx, s.ID)
	if err != nil {
		return Snapshot{}, err
	}
	return s, nil
}

func (r *sqliteRepository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_snapshots`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count snapshots: %w", err)
	}
	return count, nil
}

func (r *sqliteRepository) metrics(ctx context.Context, snapshotID string) ([]Metric, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT name, value, unit, status
FROM snapshot_probe_results
WHERE snapshot_id = ?
ORDER BY sort_order ASC
`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("list snapshot metrics: %w", err)
	}
	defer rows.Close()
	var out []Metric
	for rows.Next() {
		var m Metric
		if err := rows.Scan(&m.Name, &m.Value, &m.Unit, &m.Status); err != nil {
			return nil, fmt.Errorf("scan snapshot metric: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot metrics: %w", err)
	}
	return out, nil
}

type snapshotScanner interface {
	Scan(dest ...any) error
}

func scanSnapshot(row snapshotScanner) (Snapshot, error) {
	var s Snapshot
	var findingsJSON string
	var createdAt string
	if err := row.Scan(&s.ID, &s.Status, &s.Profile, &s.Summary, &findingsJSON, &createdAt); err != nil {
		return Snapshot{}, err
	}
	if err := json.Unmarshal([]byte(findingsJSON), &s.Findings); err != nil {
		return Snapshot{}, fmt.Errorf("unmarshal findings for snapshot %q: %w", s.ID, err)
	}
	t, err := time.Parse(TimeFormat, createdAt)
	if err != nil {
		return Snapshot{}, fmt.Errorf("parse created_at for snapshot %q: %w", s.ID, err)
	}
	s.CreatedAt = t
	return s, nil
}
