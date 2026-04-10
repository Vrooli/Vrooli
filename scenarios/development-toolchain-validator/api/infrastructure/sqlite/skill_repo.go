package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"development-toolchain-validator/domain/skill"
)

// SkillRepository implements skill.Repository using SQLite.
type SkillRepository struct {
	db *sql.DB
}

// NewSkillRepository creates a new SQLite-backed skill connection repository.
func NewSkillRepository(db *sql.DB) *SkillRepository {
	return &SkillRepository{db: db}
}

// Connect creates a new skill-reference connection.
func (r *SkillRepository) Connect(ctx context.Context, input skill.ConnectInput) (*skill.Connection, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	query := `
		INSERT INTO skill_connections (id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
	`

	var conn skill.Connection
	var version, hash sql.NullString

	err := r.db.QueryRowContext(ctx, query,
		id,
		input.ReferenceID,
		input.SkillID,
		nullString(input.SkillVersion),
		nullString(input.SkillContentHash),
		now,
		now,
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
		WHERE id = ?
	`
	return r.scanOne(ctx, query, id)
}

// GetByReferenceAndSkill retrieves a connection by reference ID and skill ID.
func (r *SkillRepository) GetByReferenceAndSkill(ctx context.Context, referenceID, skillID string) (*skill.Connection, error) {
	query := `
		SELECT id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
		FROM skill_connections
		WHERE reference_id = ? AND skill_id = ?
	`

	var conn skill.Connection
	var version, hash sql.NullString
	err := r.db.QueryRowContext(ctx, query, referenceID, skillID).Scan(
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

// List retrieves connections with optional filtering and pagination.
func (r *SkillRepository) List(ctx context.Context, opts skill.ListOptions) ([]*skill.Connection, error) {
	var conditions []string
	var args []interface{}

	if opts.ReferenceID != "" {
		conditions = append(conditions, "reference_id = ?")
		args = append(args, opts.ReferenceID)
	}
	if opts.SkillID != "" {
		conditions = append(conditions, "skill_id = ?")
		args = append(args, opts.SkillID)
	}

	query := "SELECT id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at FROM skill_connections"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY connected_at DESC"

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

	if input.SkillVersion != nil {
		sets = append(sets, "skill_version = ?")
		args = append(args, *input.SkillVersion)
	}
	if input.SkillContentHash != nil {
		sets = append(sets, "skill_content_hash = ?")
		args = append(args, *input.SkillContentHash)
	}

	if len(sets) == 0 {
		return r.GetByID(ctx, id)
	}

	sets = append(sets, "updated_at = ?")
	args = append(args, time.Now().UTC())
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE skill_connections
		SET %s
		WHERE id = ?
		RETURNING id, reference_id, skill_id, skill_version, skill_content_hash, connected_at, updated_at
	`, strings.Join(sets, ", "))

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
	result, err := r.db.ExecContext(ctx, "DELETE FROM skill_connections WHERE id = ?", id)
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
		"DELETE FROM skill_connections WHERE reference_id = ? AND skill_id = ?",
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
