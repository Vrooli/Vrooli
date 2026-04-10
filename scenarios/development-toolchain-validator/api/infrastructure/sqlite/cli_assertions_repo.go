package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"development-toolchain-validator/domain/expectation"
)

// CLIAssertionsRepository implements expectation.CLIRepository using SQLite.
type CLIAssertionsRepository struct {
	db *sql.DB
}

// NewCLIAssertionsRepository creates a new SQLite-backed CLI assertions repository.
func NewCLIAssertionsRepository(db *sql.DB) *CLIAssertionsRepository {
	return &CLIAssertionsRepository{db: db}
}

// Create adds a new CLI assertion.
func (r *CLIAssertionsRepository) Create(ctx context.Context, input expectation.CreateCLIInput) (*expectation.CLIAssertion, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	var expectedValueJSON sql.NullString
	if input.ExpectedValue != nil {
		b, err := json.Marshal(input.ExpectedValue)
		if err != nil {
			return nil, fmt.Errorf("marshaling expected value: %w", err)
		}
		expectedValueJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := `
		INSERT INTO cli_assertions (id, connection_id, command, json_path, operator, expected_value, description, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, connection_id, command, json_path, operator, expected_value, description, created_at
	`

	var assertion expectation.CLIAssertion
	var expVal, desc sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		id,
		input.ConnectionID,
		input.Command,
		input.JSONPath,
		string(input.Operator),
		expectedValueJSON,
		nullString(input.Description),
		now,
	).Scan(
		&assertion.ID,
		&assertion.ConnectionID,
		&assertion.Command,
		&assertion.JSONPath,
		&assertion.Operator,
		&expVal,
		&desc,
		&assertion.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting CLI assertion: %w", err)
	}

	if expVal.Valid {
		var v interface{}
		if err := json.Unmarshal([]byte(expVal.String), &v); err == nil {
			assertion.ExpectedValue = v
		}
	}
	assertion.Description = desc.String
	return &assertion, nil
}

// GetByID retrieves a CLI assertion by ID.
func (r *CLIAssertionsRepository) GetByID(ctx context.Context, id string) (*expectation.CLIAssertion, error) {
	query := `
		SELECT id, connection_id, command, json_path, operator, expected_value, description, created_at
		FROM cli_assertions
		WHERE id = ?
	`

	var assertion expectation.CLIAssertion
	var expVal, desc sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&assertion.ID,
		&assertion.ConnectionID,
		&assertion.Command,
		&assertion.JSONPath,
		&assertion.Operator,
		&expVal,
		&desc,
		&assertion.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, expectation.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying CLI assertion: %w", err)
	}

	if expVal.Valid {
		var v interface{}
		if err := json.Unmarshal([]byte(expVal.String), &v); err == nil {
			assertion.ExpectedValue = v
		}
	}
	assertion.Description = desc.String
	return &assertion, nil
}

// List retrieves CLI assertions with optional filtering.
func (r *CLIAssertionsRepository) List(ctx context.Context, opts expectation.ListOptions) ([]*expectation.CLIAssertion, error) {
	var conditions []string
	var args []interface{}

	if opts.ConnectionID != "" {
		conditions = append(conditions, "connection_id = ?")
		args = append(args, opts.ConnectionID)
	}

	query := "SELECT id, connection_id, command, json_path, operator, expected_value, description, created_at FROM cli_assertions"
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
		return nil, fmt.Errorf("listing CLI assertions: %w", err)
	}
	defer rows.Close()

	var assertions []*expectation.CLIAssertion
	for rows.Next() {
		var assertion expectation.CLIAssertion
		var expVal, desc sql.NullString
		if err := rows.Scan(
			&assertion.ID,
			&assertion.ConnectionID,
			&assertion.Command,
			&assertion.JSONPath,
			&assertion.Operator,
			&expVal,
			&desc,
			&assertion.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning CLI assertion row: %w", err)
		}
		if expVal.Valid {
			var v interface{}
			if err := json.Unmarshal([]byte(expVal.String), &v); err == nil {
				assertion.ExpectedValue = v
			}
		}
		assertion.Description = desc.String
		assertions = append(assertions, &assertion)
	}

	return assertions, rows.Err()
}

// Delete removes a CLI assertion by ID.
func (r *CLIAssertionsRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM cli_assertions WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("deleting CLI assertion: %w", err)
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

// DeleteByConnection removes all CLI assertions for a connection.
func (r *CLIAssertionsRepository) DeleteByConnection(ctx context.Context, connectionID string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM cli_assertions WHERE connection_id = ?", connectionID)
	if err != nil {
		return fmt.Errorf("deleting CLI assertions by connection: %w", err)
	}
	return nil
}
