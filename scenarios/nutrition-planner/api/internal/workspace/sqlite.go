package workspace

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
	return &sqliteRepository{db, clock}
}

const timeFormat = time.RFC3339Nano

func (r *sqliteRepository) Create(ctx context.Context, w Workspace) (Workspace, error) {
	if w.IdempotencyKey != "" {
		var hash, existingID string
		err := r.db.QueryRowContext(ctx, `SELECT request_hash, workspace_id FROM workspace_idempotency WHERE owner_subject=? AND idempotency_key=?`, w.OwnerSubject, w.IdempotencyKey).Scan(&hash, &existingID)
		if err == nil {
			if hash != w.RequestHash {
				return Workspace{}, ErrIdempotencyConflict{w.IdempotencyKey}
			}
			return r.Get(ctx, existingID, w.OwnerSubject)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return Workspace{}, fmt.Errorf("check idempotency key: %w", err)
		}
	}
	if w.ID == "" {
		w.ID = uuid.NewString()
	}
	now := r.clock.Now().UTC()
	if w.CreatedAt.IsZero() {
		w.CreatedAt = now
	}
	if w.UpdatedAt.IsZero() {
		w.UpdatedAt = w.CreatedAt
	}
	if w.Revision == 0 {
		w.Revision = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO workspaces(id,name,owner_subject,revision,created_at,updated_at) VALUES(?,?,?,?,?,?)`, w.ID, w.Name, w.OwnerSubject, w.Revision, w.CreatedAt.Format(timeFormat), w.UpdatedAt.Format(timeFormat))
	if err != nil {
		return Workspace{}, fmt.Errorf("insert workspace: %w", err)
	}
	if w.IdempotencyKey != "" {
		if _, err := r.db.ExecContext(ctx, `INSERT INTO workspace_idempotency(owner_subject,idempotency_key,request_hash,workspace_id,created_at) VALUES(?,?,?,?,?)`, w.OwnerSubject, w.IdempotencyKey, w.RequestHash, w.ID, now.Format(timeFormat)); err != nil {
			return Workspace{}, fmt.Errorf("record idempotency key: %w", err)
		}
	}
	return w, nil
}

func (r *sqliteRepository) List(ctx context.Context, owner string) ([]Workspace, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,name,owner_subject,revision,created_at,updated_at FROM workspaces WHERE owner_subject=? ORDER BY created_at DESC,id DESC`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Workspace
	for rows.Next() {
		w, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Get(ctx context.Context, id, owner string) (Workspace, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id,name,owner_subject,revision,created_at,updated_at FROM workspaces WHERE id=?`, id)
	w, err := scan(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Workspace{}, ErrNotFound{id}
	}
	if err != nil {
		return Workspace{}, err
	}
	if w.OwnerSubject != owner {
		return Workspace{}, ErrForbidden{id}
	}
	return w, nil
}

func scan(s interface{ Scan(...any) error }) (Workspace, error) {
	var w Workspace
	var c, u string
	if err := s.Scan(&w.ID, &w.Name, &w.OwnerSubject, &w.Revision, &c, &u); err != nil {
		return Workspace{}, err
	}
	var err error
	w.CreatedAt, err = time.Parse(timeFormat, c)
	if err != nil {
		return Workspace{}, err
	}
	w.UpdatedAt, err = time.Parse(timeFormat, u)
	return w, err
}
