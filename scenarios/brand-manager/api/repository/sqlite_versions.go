// DOC: docs/reference/api-endpoints.md#versions
package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"brand-manager/domain"
)

// SQLiteVersionRepository implements VersionRepository using SQLite.
type SQLiteVersionRepository struct {
	db *sql.DB
}

// NewSQLiteVersionRepository creates a new SQLite-backed version repository.
func NewSQLiteVersionRepository(db *sql.DB) *SQLiteVersionRepository {
	return &SQLiteVersionRepository{db: db}
}

// Create inserts a new brand version snapshot. [REQ:BM-REQ-CRUD-VERSION]
func (r *SQLiteVersionRepository) Create(ctx context.Context, v *domain.BrandVersion) error {
	t, now := nowUTC()
	v.CreatedAt = t

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO brand_versions (id, brand_id, version, snapshot, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		v.ID, v.BrandID, v.Version, v.Snapshot, now,
	)
	if err != nil {
		return fmt.Errorf("insert brand version: %w", err)
	}
	return nil
}

// ListByBrandID returns all versions for a brand, newest first. [REQ:BM-REQ-CRUD-VERSION]
func (r *SQLiteVersionRepository) ListByBrandID(ctx context.Context, brandID string) ([]*domain.BrandVersion, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, brand_id, version, snapshot, created_at FROM brand_versions
		 WHERE brand_id = ? ORDER BY version DESC`, brandID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	var versions []*domain.BrandVersion
	for rows.Next() {
		var v domain.BrandVersion
		var createdAt string
		if err := rows.Scan(&v.ID, &v.BrandID, &v.Version, &v.Snapshot, &createdAt); err != nil {
			return nil, err
		}
		v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		versions = append(versions, &v)
	}
	return versions, rows.Err()
}

// GetByBrandIDAndVersion returns a specific version snapshot.
func (r *SQLiteVersionRepository) GetByBrandIDAndVersion(ctx context.Context, brandID string, version int) (*domain.BrandVersion, error) {
	var v domain.BrandVersion
	var createdAt string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, brand_id, version, snapshot, created_at FROM brand_versions
		 WHERE brand_id = ? AND version = ?`, brandID, version).
		Scan(&v.ID, &v.BrandID, &v.Version, &v.Snapshot, &createdAt)
	if err != nil {
		return nil, err
	}
	v.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	return &v, nil
}
