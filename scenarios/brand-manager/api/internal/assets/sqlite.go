package assets

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"brand-manager/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (used by repository
// unit tests via testutil/db.NewSQLite) and *database.RoutedDB (used in
// production by main.go) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// sqliteRepository is the production impl of Repository. Unexported so callers
// depend on the interface — tests substitute fakes without reaching inside the
// struct.
type sqliteRepository struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteRepository constructs the production asset Repository.
func NewSQLiteRepository(db SQLExecutor, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// assetTimeFormat matches brands (RFC3339Nano), which sorts lexicographically in
// time order for a fixed zone so string range/ordering on the column is correct.
const assetTimeFormat = time.RFC3339Nano

const (
	selectAssetColumns = `id, brand_id, filename, mime_type, file_path, size, created_at`

	// upsertAssetSQL inserts a new row or, on a (brand_id, filename) conflict,
	// replaces the mutable columns while preserving the original id + created_at.
	upsertAssetSQL = `
INSERT INTO assets (id, brand_id, filename, mime_type, file_path, size, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(brand_id, filename) DO UPDATE SET
  mime_type = excluded.mime_type,
  file_path = excluded.file_path,
  size      = excluded.size
`
	selectAssetByIDSQL           = `SELECT ` + selectAssetColumns + ` FROM assets WHERE id = ?`
	selectAssetByBrandAndNameSQL = `SELECT ` + selectAssetColumns + ` FROM assets WHERE brand_id = ? AND filename = ?`
	deleteAssetSQL               = `DELETE FROM assets WHERE id = ?`
)

func (s *sqliteRepository) Upsert(ctx context.Context, a Asset) (Asset, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = s.clock.Now().UTC()
	}

	if _, err := s.db.ExecContext(ctx, upsertAssetSQL,
		a.ID, a.BrandID, a.Filename, a.MimeType, a.FilePath, a.Size,
		a.CreatedAt.Format(assetTimeFormat),
	); err != nil {
		return Asset{}, fmt.Errorf("upsert asset %q/%q: %w", a.BrandID, a.Filename, err)
	}

	// Re-read by the natural key so the returned row carries the canonical id +
	// created_at (which the caller's values do not when a conflict updated an
	// existing row).
	row := s.db.QueryRowContext(ctx, selectAssetByBrandAndNameSQL, a.BrandID, a.Filename)
	stored, err := scanAsset(row)
	if err != nil {
		return Asset{}, fmt.Errorf("reload upserted asset %q/%q: %w", a.BrandID, a.Filename, err)
	}
	return stored, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Asset, error) {
	row := s.db.QueryRowContext(ctx, selectAssetByIDSQL, id)
	a, err := scanAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrAssetNotFound{ID: id}
	}
	if err != nil {
		return Asset{}, fmt.Errorf("get asset %q: %w", id, err)
	}
	return a, nil
}

func (s *sqliteRepository) ListByBrand(ctx context.Context, brandID string) ([]Asset, error) {
	query := `SELECT ` + selectAssetColumns + ` FROM assets`
	var args []any
	if brandID != "" {
		query += ` WHERE brand_id = ?`
		args = append(args, brandID)
	}
	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		a, err := scanAsset(rows)
		if err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}
	return assets, nil
}

func (s *sqliteRepository) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, deleteAssetSQL, id)
	if err != nil {
		return fmt.Errorf("delete asset %q: %w", id, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete asset %q rows: %w", id, err)
	}
	if n == 0 {
		return ErrAssetNotFound{ID: id}
	}
	return nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAsset(sc rowScanner) (Asset, error) {
	var (
		a          Asset
		createdRaw string
	)
	if err := sc.Scan(&a.ID, &a.BrandID, &a.Filename, &a.MimeType, &a.FilePath, &a.Size, &createdRaw); err != nil {
		return Asset{}, err
	}
	created, err := time.Parse(assetTimeFormat, createdRaw)
	if err != nil {
		return Asset{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	a.CreatedAt = created
	return a, nil
}
