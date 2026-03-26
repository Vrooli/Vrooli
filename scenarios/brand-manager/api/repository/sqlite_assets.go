// DOC: docs/internal/SEAMS.md#2-repository--database-storage-seam
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"brand-manager/domain"
)

// SQLiteAssetRepository implements AssetRepository using SQLite.
type SQLiteAssetRepository struct {
	db *sql.DB
}

// NewSQLiteAssetRepository creates a new SQLite-backed asset repository.
func NewSQLiteAssetRepository(db *sql.DB) *SQLiteAssetRepository {
	return &SQLiteAssetRepository{db: db}
}

// Create inserts a new asset record. [REQ:BM-REQ-STORE-ASSETS]
func (r *SQLiteAssetRepository) Create(ctx context.Context, asset *domain.Asset) error {
	t, now := nowUTC()
	asset.CreatedAt = t

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO assets (id, brand_id, filename, mime_type, file_path, size, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		asset.ID, asset.BrandID, asset.Filename, asset.MimeType, asset.FilePath, asset.Size, now,
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}

// GetByID retrieves a single asset by ID. [REQ:BM-REQ-STORE-ASSETS]
func (r *SQLiteAssetRepository) GetByID(ctx context.Context, id string) (*domain.Asset, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, brand_id, filename, mime_type, file_path, size, created_at
		 FROM assets WHERE id = ?`, id)

	var a domain.Asset
	var createdAt string
	err := row.Scan(&a.ID, &a.BrandID, &a.Filename, &a.MimeType, &a.FilePath, &a.Size, &createdAt)
	if err != nil {
		return nil, err
	}
	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &a, nil
}

// ListByBrandID returns all assets for a brand. [REQ:BM-REQ-STORE-ASSETS]
func (r *SQLiteAssetRepository) ListByBrandID(ctx context.Context, brandID string) ([]*domain.Asset, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, brand_id, filename, mime_type, file_path, size, created_at
		 FROM assets WHERE brand_id = ? ORDER BY created_at DESC`, brandID)
	if err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		var a domain.Asset
		var createdAt string
		if err := rows.Scan(&a.ID, &a.BrandID, &a.Filename, &a.MimeType, &a.FilePath, &a.Size, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		assets = append(assets, &a)
	}
	return assets, rows.Err()
}

// Delete removes an asset by ID. [REQ:BM-REQ-STORE-ASSETS]
func (r *SQLiteAssetRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM assets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete asset: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
