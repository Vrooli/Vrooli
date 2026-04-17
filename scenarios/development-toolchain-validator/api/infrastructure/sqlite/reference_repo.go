package sqlite

import (
	"context"
	"database/sql"
	"development-toolchain-validator/domain/reference"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ReferenceRepository implements reference.Repository using SQLite.
type ReferenceRepository struct {
	db *sql.DB
}

// NewReferenceRepository creates a new SQLite-backed reference repository.
func NewReferenceRepository(db *sql.DB) *ReferenceRepository {
	return &ReferenceRepository{db: db}
}

// Create stores a new reference scenario.
func (r *ReferenceRepository) Create(ctx context.Context, input reference.CreateInput) (*reference.Reference, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	query := `
		INSERT INTO reference_scenarios (id, slug, name, template, path, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, slug, name, template, path, description, created_at, updated_at
	`

	var ref reference.Reference
	var desc sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		id,
		input.Slug,
		input.Name,
		input.Template,
		input.Path,
		nullString(input.Description),
		now,
		now,
	).Scan(
		&ref.ID,
		&ref.Slug,
		&ref.Name,
		&ref.Template,
		&ref.Path,
		&desc,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting reference: %w", err)
	}

	ref.Description = desc.String
	return &ref, nil
}

// GetByID retrieves a reference by its UUID.
func (r *ReferenceRepository) GetByID(ctx context.Context, id string) (*reference.Reference, error) {
	query := `
		SELECT id, slug, name, template, path, description, created_at, updated_at
		FROM reference_scenarios
		WHERE id = ?
	`
	return r.scanOne(ctx, query, id)
}

// GetBySlug retrieves a reference by its unique slug.
func (r *ReferenceRepository) GetBySlug(ctx context.Context, slug string) (*reference.Reference, error) {
	query := `
		SELECT id, slug, name, template, path, description, created_at, updated_at
		FROM reference_scenarios
		WHERE slug = ?
	`
	return r.scanOne(ctx, query, slug)
}

// List retrieves references with optional filtering and pagination.
func (r *ReferenceRepository) List(ctx context.Context, opts reference.ListOptions) ([]*reference.Reference, error) {
	var conditions []string
	var args []interface{}

	if opts.Template != "" {
		conditions = append(conditions, "template = ?")
		args = append(args, opts.Template)
	}

	query := "SELECT id, slug, name, template, path, description, created_at, updated_at FROM reference_scenarios"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY created_at DESC"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
	}
	if opts.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing references: %w", err)
	}
	defer rows.Close()

	var refs []*reference.Reference
	for rows.Next() {
		var ref reference.Reference
		var desc sql.NullString
		if err := rows.Scan(
			&ref.ID,
			&ref.Slug,
			&ref.Name,
			&ref.Template,
			&ref.Path,
			&desc,
			&ref.CreatedAt,
			&ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning reference row: %w", err)
		}
		ref.Description = desc.String
		refs = append(refs, &ref)
	}

	return refs, rows.Err()
}

// Update modifies an existing reference.
func (r *ReferenceRepository) Update(ctx context.Context, id string, input reference.UpdateInput) (*reference.Reference, error) {
	var sets []string
	var args []interface{}

	if input.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *input.Name)
	}
	if input.Template != nil {
		sets = append(sets, "template = ?")
		args = append(args, *input.Template)
	}
	if input.Path != nil {
		sets = append(sets, "path = ?")
		args = append(args, *input.Path)
	}
	if input.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *input.Description)
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC())
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE reference_scenarios
		SET %s
		WHERE id = ?
		RETURNING id, slug, name, template, path, description, created_at, updated_at
	`, strings.Join(sets, ", "))

	var ref reference.Reference
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&ref.ID,
		&ref.Slug,
		&ref.Name,
		&ref.Template,
		&ref.Path,
		&desc,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, reference.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating reference: %w", err)
	}

	ref.Description = desc.String
	return &ref, nil
}

// Delete removes a reference by ID.
func (r *ReferenceRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM reference_scenarios WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting reference: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return reference.ErrNotFound
	}

	return nil
}

func (r *ReferenceRepository) scanOne(ctx context.Context, query string, arg interface{}) (*reference.Reference, error) {
	var ref reference.Reference
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&ref.ID,
		&ref.Slug,
		&ref.Name,
		&ref.Template,
		&ref.Path,
		&desc,
		&ref.CreatedAt,
		&ref.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, reference.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying reference: %w", err)
	}

	ref.Description = desc.String
	return &ref, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
