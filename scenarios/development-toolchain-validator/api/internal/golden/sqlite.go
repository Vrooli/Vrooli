package golden

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

const (
	goldenTimeFormat = time.RFC3339Nano

	insertGoldenSQL = `
INSERT INTO goldens (
  id, slug, template_id, template_version_pinned, path,
  generation_options_json, materialization_mode, logical_root,
  last_materialized_path, last_materialized_status,
  created_at, last_regenerated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectGoldenBySlugSQL = `
SELECT id, slug, template_id, template_version_pinned, path,
       generation_options_json, materialization_mode, logical_root,
       last_materialized_path, last_materialized_status,
       created_at, last_regenerated_at
FROM goldens
WHERE slug = ?
`

	listGoldensSQL = `
SELECT id, slug, template_id, template_version_pinned, path,
       generation_options_json, materialization_mode, logical_root,
       last_materialized_path, last_materialized_status,
       created_at, last_regenerated_at
FROM goldens
ORDER BY slug ASC
`

	updateGoldenSQL = `
UPDATE goldens
SET path = ?, template_version_pinned = ?, generation_options_json = ?,
    materialization_mode = ?, logical_root = ?, last_materialized_path = ?,
    last_materialized_status = ?, last_regenerated_at = ?
WHERE slug = ?
`

	deleteGoldenSQL = `
DELETE FROM goldens WHERE slug = ?
`
)

// SQLExecutor is the narrow database surface this package's repository
// depends on. Both *sql.DB (used by repository unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production main.go)
// satisfy it, so production wiring participates in per-request routing
// without forcing test fixtures to wrap their handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Create(ctx context.Context, g Golden) (Golden, error) {
	g = normalizeGolden(g)
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	now := s.clock.Now().UTC()
	if g.CreatedAt.IsZero() {
		g.CreatedAt = now
	}
	if g.LastRegeneratedAt.IsZero() {
		g.LastRegeneratedAt = g.CreatedAt
	}

	_, err := s.db.ExecContext(ctx, insertGoldenSQL,
		g.ID, g.Slug, g.TemplateID, g.TemplateVersionPinned, g.Path,
		g.GenerationOptionsJSON, g.MaterializationMode, g.LogicalRoot,
		g.LastMaterializedPath, g.LastMaterializedStatus,
		g.CreatedAt.Format(goldenTimeFormat),
		g.LastRegeneratedAt.Format(goldenTimeFormat),
	)
	if err != nil {
		if isUniqueViolation(err) {
			return Golden{}, ErrGoldenAlreadyExists{Slug: g.Slug}
		}
		return Golden{}, fmt.Errorf("insert golden %q: %w", g.Slug, err)
	}
	return g, nil
}

func (s *sqliteRepository) Get(ctx context.Context, slug string) (Golden, error) {
	row := s.db.QueryRowContext(ctx, selectGoldenBySlugSQL, slug)
	g, err := scanGolden(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Golden{}, ErrGoldenNotFound{Slug: slug}
	}
	if err != nil {
		return Golden{}, fmt.Errorf("get golden %q: %w", slug, err)
	}
	return g, nil
}

func (s *sqliteRepository) List(ctx context.Context) ([]Golden, error) {
	rows, err := s.db.QueryContext(ctx, listGoldensSQL)
	if err != nil {
		return nil, fmt.Errorf("list goldens: %w", err)
	}
	defer rows.Close()

	var out []Golden
	for rows.Next() {
		g, err := scanGolden(rows)
		if err != nil {
			return nil, fmt.Errorf("scan golden: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate goldens: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) Update(ctx context.Context, g Golden) (Golden, error) {
	g = normalizeGolden(g)
	if g.LastRegeneratedAt.IsZero() {
		g.LastRegeneratedAt = s.clock.Now().UTC()
	}
	res, err := s.db.ExecContext(ctx, updateGoldenSQL,
		g.Path, g.TemplateVersionPinned, g.GenerationOptionsJSON,
		g.MaterializationMode, g.LogicalRoot, g.LastMaterializedPath,
		g.LastMaterializedStatus,
		g.LastRegeneratedAt.Format(goldenTimeFormat),
		g.Slug,
	)
	if err != nil {
		return Golden{}, fmt.Errorf("update golden %q: %w", g.Slug, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return Golden{}, fmt.Errorf("update golden %q rows affected: %w", g.Slug, err)
	}
	if n == 0 {
		return Golden{}, ErrGoldenNotFound{Slug: g.Slug}
	}
	return s.Get(ctx, g.Slug)
}

func (s *sqliteRepository) Delete(ctx context.Context, slug string) error {
	res, err := s.db.ExecContext(ctx, deleteGoldenSQL, slug)
	if err != nil {
		return fmt.Errorf("delete golden %q: %w", slug, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete golden %q rows affected: %w", slug, err)
	}
	if n == 0 {
		return ErrGoldenNotFound{Slug: slug}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanGolden(s rowScanner) (Golden, error) {
	var (
		g          Golden
		createdRaw string
		regenRaw   string
	)
	if err := s.Scan(
		&g.ID, &g.Slug, &g.TemplateID, &g.TemplateVersionPinned, &g.Path,
		&g.GenerationOptionsJSON, &g.MaterializationMode, &g.LogicalRoot,
		&g.LastMaterializedPath, &g.LastMaterializedStatus,
		&createdRaw, &regenRaw,
	); err != nil {
		return Golden{}, err
	}
	g = normalizeGolden(g)
	created, err := time.Parse(goldenTimeFormat, createdRaw)
	if err != nil {
		return Golden{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	regen, err := time.Parse(goldenTimeFormat, regenRaw)
	if err != nil {
		return Golden{}, fmt.Errorf("parse last_regenerated_at %q: %w", regenRaw, err)
	}
	g.CreatedAt = created
	g.LastRegeneratedAt = regen
	return g, nil
}

func normalizeGolden(g Golden) Golden {
	if g.MaterializationMode == "" {
		g.MaterializationMode = MaterializationModeEphemeral
	}
	if g.LogicalRoot == "" {
		g.LogicalRoot = g.Path
	}
	if g.LastMaterializedStatus == "" {
		g.LastMaterializedStatus = MaterializationStatusNever
	}
	return g
}

// isUniqueViolation detects modernc.org/sqlite's UNIQUE constraint error
// without importing the driver-specific error type (keeping the repository
// portable to other SQLite drivers).
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "constraint failed: UNIQUE")
}
