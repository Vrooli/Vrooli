package channels

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	sqliteRepository struct {
		db    SQLExecutor
		clock schedule.Clock
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) Create(ctx context.Context, c Channel) (Channel, error) {
	c.ID = uuid.NewString()
	c.CreatedAt = r.clock.Now().UTC()
	_, err := r.db.ExecContext(ctx, `INSERT INTO persona_channels (id, persona_id, kind, address, credential_ref, adapter, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, c.ID, c.PersonaID, c.Kind, c.Address, c.CredentialRef, c.Adapter, boolInt(c.Enabled), c.CreatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Channel{}, fmt.Errorf("insert channel: %w", err)
	}
	return c, nil
}

func (r *sqliteRepository) List(ctx context.Context, personaID string) ([]Channel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, kind, address, credential_ref, adapter, enabled, created_at FROM persona_channels WHERE persona_id = ? ORDER BY created_at`, personaID)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()
	var out []Channel
	for rows.Next() {
		c, err := scanChannel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Channel, error) {
	c, err := scanChannel(r.db.QueryRowContext(ctx, `SELECT id, persona_id, kind, address, credential_ref, adapter, enabled, created_at FROM persona_channels WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Channel{}, ErrChannelNotFound
	}
	return c, err
}

type rowScanner interface{ Scan(...any) error }

func scanChannel(row rowScanner) (Channel, error) {
	var c Channel
	var kind, created string
	var enabled int
	if err := row.Scan(&c.ID, &c.PersonaID, &kind, &c.Address, &c.CredentialRef, &c.Adapter, &enabled, &created); err != nil {
		return Channel{}, err
	}
	c.Kind = Kind(kind)
	c.Enabled = enabled == 1
	var err error
	c.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Channel{}, fmt.Errorf("parse channel created_at: %w", err)
	}
	return c, nil
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
