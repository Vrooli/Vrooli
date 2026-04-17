package sqlite

import (
	"context"
	"database/sql"
	"development-toolchain-validator/domain/expectation"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StructuralExpectationsRepository implements expectation.StructuralRepository using SQLite.
type StructuralExpectationsRepository struct {
	db *sql.DB
}

// NewStructuralExpectationsRepository creates a new SQLite-backed structural expectations repository.
func NewStructuralExpectationsRepository(db *sql.DB) *StructuralExpectationsRepository {
	return &StructuralExpectationsRepository{db: db}
}

// Create adds a new structural expectation.
func (r *StructuralExpectationsRepository) Create(ctx context.Context, input expectation.CreateStructuralInput) (*expectation.StructuralExpectation, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	required := 0
	if input.Required {
		required = 1
	}

	query := `
		INSERT INTO structural_expectations (id, connection_id, type, pattern, required, expected_content, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, connection_id, type, pattern, required, expected_content, description, created_at
	`

	var exp expectation.StructuralExpectation
	var expContent, desc sql.NullString
	var reqInt int

	err := r.db.QueryRowContext(ctx, query,
		id,
		input.ConnectionID,
		string(input.Type),
		input.Pattern,
		required,
		nullString(input.ExpectedContent),
		nullString(input.Description),
		now,
	).Scan(
		&exp.ID,
		&exp.ConnectionID,
		&exp.Type,
		&exp.Pattern,
		&reqInt,
		&expContent,
		&desc,
		&exp.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting structural expectation: %w", err)
	}

	exp.Required = reqInt != 0
	exp.ExpectedContent = expContent.String
	exp.Description = desc.String
	return &exp, nil
}

// GetByID retrieves a structural expectation by ID.
func (r *StructuralExpectationsRepository) GetByID(ctx context.Context, id string) (*expectation.StructuralExpectation, error) {
	query := `
		SELECT id, connection_id, type, pattern, required, expected_content, description, created_at
		FROM structural_expectations
		WHERE id = ?
	`

	var exp expectation.StructuralExpectation
	var expContent, desc sql.NullString
	var reqInt int

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&exp.ID,
		&exp.ConnectionID,
		&exp.Type,
		&exp.Pattern,
		&reqInt,
		&expContent,
		&desc,
		&exp.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, expectation.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying structural expectation: %w", err)
	}

	exp.Required = reqInt != 0
	exp.ExpectedContent = expContent.String
	exp.Description = desc.String
	return &exp, nil
}

// List retrieves structural expectations with optional filtering.
func (r *StructuralExpectationsRepository) List(ctx context.Context, opts expectation.ListOptions) ([]*expectation.StructuralExpectation, error) {
	var conditions []string
	var args []interface{}

	if opts.ConnectionID != "" {
		conditions = append(conditions, "connection_id = ?")
		args = append(args, opts.ConnectionID)
	}

	query := "SELECT id, connection_id, type, pattern, required, expected_content, description, created_at FROM structural_expectations"
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
		return nil, fmt.Errorf("listing structural expectations: %w", err)
	}
	defer rows.Close()

	var exps []*expectation.StructuralExpectation
	for rows.Next() {
		var exp expectation.StructuralExpectation
		var expContent, desc sql.NullString
		var reqInt int
		if err := rows.Scan(
			&exp.ID,
			&exp.ConnectionID,
			&exp.Type,
			&exp.Pattern,
			&reqInt,
			&expContent,
			&desc,
			&exp.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning structural expectation row: %w", err)
		}
		exp.Required = reqInt != 0
		exp.ExpectedContent = expContent.String
		exp.Description = desc.String
		exps = append(exps, &exp)
	}

	return exps, rows.Err()
}

// Delete removes a structural expectation by ID.
func (r *StructuralExpectationsRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM structural_expectations WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting structural expectation: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return expectation.ErrNotFound
	}

	return nil
}

// DeleteByConnection removes all structural expectations for a connection.
func (r *StructuralExpectationsRepository) DeleteByConnection(ctx context.Context, connectionID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM structural_expectations WHERE connection_id = ?", connectionID)
	if err != nil {
		return fmt.Errorf("deleting structural expectations by connection: %w", err)
	}
	return nil
}
