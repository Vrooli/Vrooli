package projects

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Repository is the engine-independent project persistence contract.
type Repository interface {
	ListActive(context.Context) ([]Project, error)
	Create(context.Context, string) (Project, error)
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) ListActive(ctx context.Context) ([]Project, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, status FROM projects WHERE status = ? ORDER BY name COLLATE NOCASE, id`, StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Status); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) Create(ctx context.Context, name string) (Project, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.EqualFold(name, UnassignedID) {
		return Project{}, ErrInvalidName
	}
	p := Project{ID: uuid.NewString(), Name: name, Status: StatusActive}
	_, err := r.db.ExecContext(ctx, `INSERT INTO projects (id, name, status, created_at) VALUES (?, ?, ?, ?)`, p.ID, p.Name, p.Status, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}
	return p, nil
}
