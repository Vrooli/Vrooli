// DOC: docs/internal/SEAMS.md#postgresql-database
// DOC: docs/internal/STORAGE_AUDIT.md
// DOC: docs/reference/data-model.md#skill-connections
// DOC: initialization/postgres/schema.sql
//
// Package postgres implements PostgreSQL storage for domain entities.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"development-toolchain-validator/domain/skill"
)

// SkillRepository implements skill.Repository using PostgreSQL.
type SkillRepository struct {
	db *sql.DB
}

// NewSkillRepository creates a new PostgreSQL-backed skill connection repository.
func NewSkillRepository(db *sql.DB) *SkillRepository {
	return &SkillRepository{db: db}
}

// Connect creates a new skill-reference connection.
func (r *SkillRepository) Connect(ctx context.Context, input skill.ConnectInput) (*skill.Connection, error) {
	query := `
		INSERT INTO skill_connections (reference_id, skill_id, skill_version, skill_content_hash)
		VALUES ($1, $2, $3, $4)
		RETURNING id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
	`

	var conn skill.Connection
	var version, hash sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		input.ReferenceID,
		input.SkillID,
		nullString(input.SkillVersion),
		nullString(input.SkillContentHash),
	).Scan(
		&conn.ID,
		&conn.ReferenceID,
		&conn.SkillID,
		&version,
		&hash,
		&conn.ConnectedAt,
		&conn.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting skill connection: %w", err)
	}

	conn.SkillVersion = version.String
	conn.SkillContentHash = hash.String
	return &conn, nil
}

// GetByID retrieves a connection by its UUID.
func (r *SkillRepository) GetByID(ctx context.Context, id string) (*skill.Connection, error) {
	query := `
		SELECT id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
		FROM skill_connections
		WHERE id = $1
	`
	return r.scanOne(ctx, query, id)
}

// GetByReferenceAndSkill retrieves a connection by reference ID and skill ID.
func (r *SkillRepository) GetByReferenceAndSkill(ctx context.Context, referenceID, skillID string) (*skill.Connection, error) {
	query := `
		SELECT id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
		FROM skill_connections
		WHERE reference_id = $1 AND skill_id = $2
	`
	return r.scanTwo(ctx, query, referenceID, skillID)
}

// List retrieves connections with optional filtering and pagination.
func (r *SkillRepository) List(ctx context.Context, opts skill.ListOptions) ([]*skill.Connection, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	if opts.ReferenceID != "" {
		conditions = append(conditions, fmt.Sprintf("reference_id = $%d", argIdx))
		args = append(args, opts.ReferenceID)
		argIdx++
	}
	if opts.SkillID != "" {
		conditions = append(conditions, fmt.Sprintf("skill_id = $%d", argIdx))
		args = append(args, opts.SkillID)
		argIdx++
	}

	query := "SELECT id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at FROM skill_connections"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY connected_at DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIdx)
		args = append(args, opts.Limit)
		argIdx++
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIdx)
		args = append(args, opts.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}
	defer rows.Close()

	var conns []*skill.Connection
	for rows.Next() {
		var conn skill.Connection
		var version, hash sql.NullString
		if err := rows.Scan(
			&conn.ID,
			&conn.ReferenceID,
			&conn.SkillID,
			&version,
			&hash,
			&conn.ConnectedAt,
			&conn.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning connection row: %w", err)
		}
		conn.SkillVersion = version.String
		conn.SkillContentHash = hash.String
		conns = append(conns, &conn)
	}

	return conns, rows.Err()
}

// Update modifies an existing connection.
func (r *SkillRepository) Update(ctx context.Context, id string, input skill.UpdateInput) (*skill.Connection, error) {
	var sets []string
	var args []interface{}
	argIdx := 1

	if input.SkillVersion != nil {
		sets = append(sets, fmt.Sprintf("skill_version = $%d", argIdx))
		args = append(args, *input.SkillVersion)
		argIdx++
	}
	if input.SkillContentHash != nil {
		sets = append(sets, fmt.Sprintf("skill_content_hash = $%d", argIdx))
		args = append(args, *input.SkillContentHash)
		argIdx++
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	query := fmt.Sprintf(`
		UPDATE skill_connections
		SET %s
		WHERE id = $%d
		RETURNING id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
	`, strings.Join(sets, ", "), argIdx)
	args = append(args, id)

	var conn skill.Connection
	var version, hash sql.NullString
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&conn.ID,
		&conn.ReferenceID,
		&conn.SkillID,
		&version,
		&hash,
		&conn.ConnectedAt,
		&conn.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, skill.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("updating connection: %w", err)
	}

	conn.SkillVersion = version.String
	conn.SkillContentHash = hash.String
	return &conn, nil
}

// Disconnect removes a skill-reference connection by ID.
func (r *SkillRepository) Disconnect(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM skill_connections WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("disconnecting skill: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return skill.ErrNotFound
	}

	return nil
}

// DisconnectByReferenceAndSkill removes a connection by reference ID and skill ID.
func (r *SkillRepository) DisconnectByReferenceAndSkill(ctx context.Context, referenceID, skillID string) error {
	result, err := r.db.ExecContext(ctx,
		"DELETE FROM skill_connections WHERE reference_id = $1 AND skill_id = $2",
		referenceID, skillID,
	)
	if err != nil {
		return fmt.Errorf("disconnecting skill: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		return skill.ErrNotFound
	}

	return nil
}

func (r *SkillRepository) scanOne(ctx context.Context, query string, arg interface{}) (*skill.Connection, error) {
	var conn skill.Connection
	var version, hash sql.NullString
	err := r.db.QueryRowContext(ctx, query, arg).Scan(
		&conn.ID,
		&conn.ReferenceID,
		&conn.SkillID,
		&version,
		&hash,
		&conn.ConnectedAt,
		&conn.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, skill.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying connection: %w", err)
	}

	conn.SkillVersion = version.String
	conn.SkillContentHash = hash.String
	return &conn, nil
}

func (r *SkillRepository) scanTwo(ctx context.Context, query string, arg1, arg2 interface{}) (*skill.Connection, error) {
	var conn skill.Connection
	var version, hash sql.NullString
	err := r.db.QueryRowContext(ctx, query, arg1, arg2).Scan(
		&conn.ID,
		&conn.ReferenceID,
		&conn.SkillID,
		&version,
		&hash,
		&conn.ConnectedAt,
		&conn.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, skill.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("querying connection: %w", err)
	}

	conn.SkillVersion = version.String
	conn.SkillContentHash = hash.String
	return &conn, nil
}
