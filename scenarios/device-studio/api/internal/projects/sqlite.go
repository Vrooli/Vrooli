package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}
type sqliteRepository struct {
	db  SQLExecutor
	now func() time.Time
}

func NewSQLiteRepository(db SQLExecutor, now func() time.Time) Repository {
	if now == nil {
		now = time.Now
	}
	return &sqliteRepository{db: db, now: now}
}

const (
	timeFormat      = "2006-01-02T15:04:05.000000000Z07:00"
	parseTimeFormat = time.RFC3339Nano
)

// timestampOrderKey canonicalizes legacy RFC3339Nano values (which may have
// variable fractional precision) for lexicographic ordering without rewriting
// previously persisted development rows.
func timestampOrderKey(column string) string {
	return `(substr(` + column + `,1,19) || '.' || CASE WHEN substr(` + column + `,20,1)='Z' THEN '000000000' ELSE substr(substr(` + column + `,21,instr(substr(` + column + `,21),'Z')-1) || '000000000',1,9) END)`
}

func (r *sqliteRepository) Create(ctx context.Context, p Project) (Project, error) {
	p.ID = uuid.NewString()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = r.now().UTC()
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = p.CreatedAt
	}
	if p.Revision <= 0 {
		p.Revision = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO projects(id,name,created_at,updated_at,revision) VALUES(?,?,?,?,?)`, p.ID, p.Name, p.CreatedAt.Format(timeFormat), p.UpdatedAt.Format(timeFormat), p.Revision)
	if err != nil {
		return Project{}, fmt.Errorf("insert project: %w", err)
	}
	return p, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Project, error) {
	p, err := scan(r.db.QueryRowContext(ctx, `SELECT id,name,created_at,updated_at,revision FROM projects WHERE id=?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrProjectNotFound{ID: id}
	}
	if err != nil {
		return Project{}, fmt.Errorf("get project: %w", err)
	}
	return p, nil
}

func (r *sqliteRepository) List(ctx context.Context, limit int, after *Project) ([]Project, error) {
	createdAtKey := timestampOrderKey("created_at")
	query := `SELECT id,name,created_at,updated_at,revision FROM projects ORDER BY ` + createdAtKey + ` DESC,id DESC LIMIT ?1`
	args := []any{limit}
	if after != nil {
		query = `SELECT id,name,created_at,updated_at,revision FROM projects WHERE ` + createdAtKey + ` < ` + timestampOrderKey("?1") + ` OR (` + createdAtKey + ` = ` + timestampOrderKey("?1") + ` AND id < ?2) ORDER BY ` + createdAtKey + ` DESC,id DESC LIMIT ?3`
		args = []any{after.CreatedAt.Format(timeFormat), after.ID, limit}
	}
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	out := make([]Project, 0)
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *sqliteRepository) Rename(ctx context.Context, in RenameInput, name string, expected int64) (Project, error) {
	now := r.now().UTC()
	p, err := scan(r.db.QueryRowContext(ctx, `UPDATE projects SET name=?,updated_at=?,revision=revision+1 WHERE id=? AND revision=? RETURNING id,name,created_at,updated_at,revision`, name, now.Format(timeFormat), in.ID, expected))
	if errors.Is(err, sql.ErrNoRows) {
		current, getErr := r.Get(ctx, in.ID)
		if getErr != nil {
			return Project{}, getErr
		}
		return Project{}, ErrRevisionConflict{ID: in.ID, Expected: expected, Current: current.Revision}
	}
	if err != nil {
		return Project{}, fmt.Errorf("rename project: %w", err)
	}
	return p, nil
}

type scanner interface{ Scan(...any) error }

func scan(s scanner) (Project, error) {
	var p Project
	var created, updated string
	if err := s.Scan(&p.ID, &p.Name, &created, &updated, &p.Revision); err != nil {
		return Project{}, err
	}
	var err error
	p.CreatedAt, err = time.Parse(parseTimeFormat, created)
	if err != nil {
		return Project{}, err
	}
	p.UpdatedAt, err = time.Parse(parseTimeFormat, updated)
	return p, err
}
