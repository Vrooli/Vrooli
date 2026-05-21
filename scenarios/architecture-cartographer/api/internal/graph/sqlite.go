package graph

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"architecture-cartographer/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const snapshotTimeFormat = time.RFC3339Nano

const (
	insertSnapshotSQL = `
INSERT INTO graph_snapshots (id, scenario, content_hash, payload, extracted_at, extraction_ms)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario, content_hash) DO UPDATE SET
  payload = excluded.payload,
  extracted_at = excluded.extracted_at,
  extraction_ms = excluded.extraction_ms`

	selectSnapshotByIDSQL = `
SELECT id, scenario, content_hash, payload, extracted_at, extraction_ms
FROM graph_snapshots WHERE id = ?`

	selectSnapshotByHashSQL = `
SELECT id, scenario, content_hash, payload, extracted_at, extraction_ms
FROM graph_snapshots WHERE scenario = ? AND content_hash = ?`

	listSnapshotsSQL = `
SELECT id, scenario, content_hash, payload, extracted_at, extraction_ms
FROM graph_snapshots
WHERE (? = '' OR scenario = ?)
ORDER BY extracted_at DESC, id DESC
LIMIT ?`

	clearSnapshotsSQL = `DELETE FROM graph_snapshots WHERE scenario = ?`
)

func (r *sqliteRepository) SaveSnapshot(ctx context.Context, s GraphSnapshot) (GraphSnapshot, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if s.ExtractedAt.IsZero() {
		s.ExtractedAt = r.clock.Now().UTC()
	}
	payload, err := json.Marshal(snapshotPayload{
		Languages: s.Languages,
		Files:     s.Files,
		Packages:  s.Packages,
		Symbols:   s.Symbols,
		Imports:   s.Imports,
	})
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("encode snapshot %q: %w", s.ID, err)
	}
	_, err = r.db.ExecContext(ctx, insertSnapshotSQL,
		s.ID, s.Scenario, s.ContentHash, payload,
		s.ExtractedAt.Format(snapshotTimeFormat), s.ExtractionMS,
	)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("insert snapshot %q: %w", s.ID, err)
	}
	return s, nil
}

func (r *sqliteRepository) GetSnapshot(ctx context.Context, id string) (GraphSnapshot, error) {
	row := r.db.QueryRowContext(ctx, selectSnapshotByIDSQL, id)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshot{}, ErrSnapshotNotFound{ID: id}
	}
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("get snapshot %q: %w", id, err)
	}
	return s, nil
}

func (r *sqliteRepository) FindByHash(ctx context.Context, scenario, contentHash string) (GraphSnapshot, error) {
	row := r.db.QueryRowContext(ctx, selectSnapshotByHashSQL, scenario, contentHash)
	s, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return GraphSnapshot{}, ErrSnapshotNotFound{ID: fmt.Sprintf("scenario=%s hash=%s", scenario, contentHash)}
	}
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("find by hash: %w", err)
	}
	return s, nil
}

func (r *sqliteRepository) ListSnapshots(ctx context.Context, f ListSnapshotsFilter) (SnapshotPage, error) {
	limit := f.PageSize
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, listSnapshotsSQL, f.Scenario, f.Scenario, limit)
	if err != nil {
		return SnapshotPage{}, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var out []GraphSnapshot
	for rows.Next() {
		s, err := scanSnapshot(rows)
		if err != nil {
			return SnapshotPage{}, err
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return SnapshotPage{}, fmt.Errorf("iterate snapshots: %w", err)
	}
	return SnapshotPage{Snapshots: out}, nil
}

func (r *sqliteRepository) ClearSnapshots(ctx context.Context, scenario string) (int, error) {
	res, err := r.db.ExecContext(ctx, clearSnapshotsSQL, scenario)
	if err != nil {
		return 0, fmt.Errorf("clear snapshots: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected: %w", err)
	}
	return int(n), nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

type snapshotPayload struct {
	Languages []Language    `json:"languages"`
	Files     []FileNode    `json:"files"`
	Packages  []PackageNode `json:"packages"`
	Symbols   []SymbolNode  `json:"symbols"`
	Imports   []ImportEdge  `json:"imports"`
}

func scanSnapshot(s rowScanner) (GraphSnapshot, error) {
	var (
		snap         GraphSnapshot
		payload      []byte
		extractedRaw string
	)
	if err := s.Scan(
		&snap.ID, &snap.Scenario, &snap.ContentHash,
		&payload, &extractedRaw, &snap.ExtractionMS,
	); err != nil {
		return GraphSnapshot{}, err
	}
	t, err := time.Parse(snapshotTimeFormat, extractedRaw)
	if err != nil {
		return GraphSnapshot{}, fmt.Errorf("parse extracted_at: %w", err)
	}
	snap.ExtractedAt = t

	var p snapshotPayload
	if len(payload) > 0 {
		if err := json.Unmarshal(payload, &p); err != nil {
			return GraphSnapshot{}, fmt.Errorf("decode payload: %w", err)
		}
		snap.Languages = p.Languages
		snap.Files = p.Files
		snap.Packages = p.Packages
		snap.Symbols = p.Symbols
		snap.Imports = p.Imports
	}
	return snap, nil
}
