package access

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) CreateGrant(ctx context.Context, grant Grant) (Grant, error) {
	grant.ID = uuid.NewString()
	grant.CreatedAt = r.clock.Now().UTC()
	grant.UpdatedAt = grant.CreatedAt
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_grants (id, persona_id, human_subject, level, source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, grant.ID, grant.PersonaID, grant.HumanSubject, grant.Level, grant.Source, grant.CreatedAt.Format(time.RFC3339Nano), grant.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Grant{}, fmt.Errorf("insert persona grant: %w", err)
	}
	return grant, nil
}

func (r *sqliteRepository) ListGrants(ctx context.Context, personaID string) ([]Grant, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, human_subject, level, source, created_at, updated_at FROM persona_grants WHERE persona_id = ? ORDER BY human_subject`, personaID)
	if err != nil {
		return nil, fmt.Errorf("list persona grants: %w", err)
	}
	defer rows.Close()
	var out []Grant
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, grant)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) GetGrant(ctx context.Context, id string) (Grant, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, persona_id, human_subject, level, source, created_at, updated_at FROM persona_grants WHERE id = ?`, id)
	grant, err := scanGrant(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Grant{}, ErrGrantNotFound
	}
	return grant, err
}

func (r *sqliteRepository) RemoveGrant(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM persona_grants WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("remove persona grant: %w", err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return ErrGrantNotFound
	}
	return nil
}

type rowScanner interface{ Scan(...any) error }

func scanGrant(row rowScanner) (Grant, error) {
	var grant Grant
	var level, created, updated string
	if err := row.Scan(&grant.ID, &grant.PersonaID, &grant.HumanSubject, &level, &grant.Source, &created, &updated); err != nil {
		return Grant{}, err
	}
	grant.Level = GrantLevel(level)
	var err error
	grant.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Grant{}, fmt.Errorf("parse grant created_at: %w", err)
	}
	grant.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Grant{}, fmt.Errorf("parse grant updated_at: %w", err)
	}
	return grant, nil
}
