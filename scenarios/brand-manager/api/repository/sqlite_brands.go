// DOC: docs/internal/SEAMS.md#2-repository--database-storage-seam
// DOC: docs/concepts/ARCHITECTURE.md#data-flow
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"brand-manager/domain"
)

// SQLiteBrandRepository implements BrandRepository using SQLite.
type SQLiteBrandRepository struct {
	db *sql.DB
}

// NewSQLiteBrandRepository creates a new SQLite-backed brand repository.
func NewSQLiteBrandRepository(db *sql.DB) *SQLiteBrandRepository {
	return &SQLiteBrandRepository{db: db}
}

// Create inserts a new brand. [REQ:BM-REQ-CRUD-CREATE]
func (r *SQLiteBrandRepository) Create(ctx context.Context, brand *domain.Brand) error {
	identityJSON, _ := json.Marshal(brand.Identity)
	colorsJSON, _ := json.Marshal(brand.Colors)
	typographyJSON, _ := json.Marshal(brand.Typography)
	voiceJSON, _ := json.Marshal(brand.Voice)

	t, now := nowUTC()
	brand.CreatedAt = t
	brand.UpdatedAt = brand.CreatedAt
	brand.Version = 1

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO brands (id, name, description, identity, colors, typography, voice, notes, version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		brand.ID, brand.Name, brand.Description,
		string(identityJSON), string(colorsJSON), string(typographyJSON), string(voiceJSON),
		brand.Notes, brand.Version, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert brand: %w", err)
	}
	return nil
}

// GetByID retrieves a single brand by ID. [REQ:BM-REQ-CRUD-READ]
func (r *SQLiteBrandRepository) GetByID(ctx context.Context, id string) (*domain.Brand, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, identity, colors, typography, voice, notes, version, created_at, updated_at
		 FROM brands WHERE id = ?`, id)
	return scanBrand(row)
}

// List returns brands matching the optional filter. [REQ:BM-REQ-CRUD-READ]
func (r *SQLiteBrandRepository) List(ctx context.Context, filter domain.BrandFilter) ([]*domain.Brand, error) {
	query := `SELECT id, name, description, identity, colors, typography, voice, notes, version, created_at, updated_at FROM brands`
	var args []interface{}

	if filter.NameContains != "" {
		query += ` WHERE name LIKE ?`
		args = append(args, "%"+filter.NameContains+"%")
	}

	query += ` ORDER BY updated_at DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}
	if filter.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filter.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list brands: %w", err)
	}
	defer rows.Close()

	var brands []*domain.Brand
	for rows.Next() {
		b, err := scanBrandRow(rows)
		if err != nil {
			return nil, err
		}
		brands = append(brands, b)
	}
	return brands, rows.Err()
}

// Update modifies a brand and increments its version. [REQ:BM-REQ-CRUD-UPDATE]
func (r *SQLiteBrandRepository) Update(ctx context.Context, brand *domain.Brand) error {
	identityJSON, _ := json.Marshal(brand.Identity)
	colorsJSON, _ := json.Marshal(brand.Colors)
	typographyJSON, _ := json.Marshal(brand.Typography)
	voiceJSON, _ := json.Marshal(brand.Voice)

	t, now := nowUTC()
	brand.UpdatedAt = t
	brand.Version++

	result, err := r.db.ExecContext(ctx,
		`UPDATE brands SET name=?, description=?, identity=?, colors=?, typography=?, voice=?, notes=?, version=?, updated_at=?
		 WHERE id=?`,
		brand.Name, brand.Description,
		string(identityJSON), string(colorsJSON), string(typographyJSON), string(voiceJSON),
		brand.Notes, brand.Version, now, brand.ID,
	)
	if err != nil {
		return fmt.Errorf("update brand: %w", err)
	}

	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// Delete removes a brand by ID.
func (r *SQLiteBrandRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM brands WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete brand: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// scanBrand scans a single row into a Brand.
func scanBrand(row *sql.Row) (*domain.Brand, error) {
	var b domain.Brand
	var identityJSON, colorsJSON, typographyJSON, voiceJSON, createdAt, updatedAt string

	err := row.Scan(&b.ID, &b.Name, &b.Description,
		&identityJSON, &colorsJSON, &typographyJSON, &voiceJSON,
		&b.Notes, &b.Version, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return parseBrandJSON(&b, identityJSON, colorsJSON, typographyJSON, voiceJSON, createdAt, updatedAt)
}

// scanBrandRow scans a rows iterator into a Brand.
func scanBrandRow(rows *sql.Rows) (*domain.Brand, error) {
	var b domain.Brand
	var identityJSON, colorsJSON, typographyJSON, voiceJSON, createdAt, updatedAt string

	err := rows.Scan(&b.ID, &b.Name, &b.Description,
		&identityJSON, &colorsJSON, &typographyJSON, &voiceJSON,
		&b.Notes, &b.Version, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	return parseBrandJSON(&b, identityJSON, colorsJSON, typographyJSON, voiceJSON, createdAt, updatedAt)
}

// hasContent decides whether a JSON column from SQLite contains meaningful data.
// SQLite stores empty optional facets as "" or "{}" (empty object); in either case
// the domain pointer should remain nil to signal "not set" to the API consumer.
func hasContent(jsonCol string) bool {
	return jsonCol != "" && jsonCol != "null" && jsonCol != "{}"
}

func parseBrandJSON(b *domain.Brand, identityJSON, colorsJSON, typographyJSON, voiceJSON, createdAt, updatedAt string) (*domain.Brand, error) {
	if hasContent(identityJSON) {
		b.Identity = &domain.Identity{}
		json.Unmarshal([]byte(identityJSON), b.Identity)
	}
	if hasContent(colorsJSON) {
		b.Colors = &domain.Colors{}
		json.Unmarshal([]byte(colorsJSON), b.Colors)
	}
	if hasContent(typographyJSON) {
		b.Typography = &domain.Typography{}
		json.Unmarshal([]byte(typographyJSON), b.Typography)
	}
	if hasContent(voiceJSON) {
		b.Voice = &domain.Voice{}
		json.Unmarshal([]byte(voiceJSON), b.Voice)
	}

	b.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	b.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return b, nil
}
