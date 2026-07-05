package authoring

import (
	"context"
	"database/sql"
	"fmt"
)

type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) SaveSession(ctx context.Context, s Session) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO authoring_sessions (id, scenario, target_path, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  scenario=excluded.scenario,
  target_path=excluded.target_path,
  status=excluded.status,
  updated_at=excluded.updated_at`,
		s.ID, s.Scenario, s.TargetPath, s.Status, s.CreatedAt, s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save authoring session %q: %w", s.ID, err)
	}
	return nil
}

func (r *sqliteRepository) GetSession(ctx context.Context, id string) (Session, error) {
	var s Session
	err := r.db.QueryRowContext(ctx, `SELECT id, scenario, target_path, status, created_at, updated_at FROM authoring_sessions WHERE id = ?`, id).
		Scan(&s.ID, &s.Scenario, &s.TargetPath, &s.Status, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return Session{}, fmt.Errorf("authoring session %q not found", id)
		}
		return Session{}, fmt.Errorf("get authoring session %q: %w", id, err)
	}
	return s, nil
}

func (r *sqliteRepository) DeleteSession(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM authoring_sessions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete authoring session %q: %w", id, err)
	}
	return nil
}

func (r *sqliteRepository) SavePage(ctx context.Context, p PageDraft) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO authoring_pages (session_id, page_id, path, title, status, json, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(session_id, page_id) DO UPDATE SET
  path=excluded.path,
  title=excluded.title,
  status=excluded.status,
  json=excluded.json,
  updated_at=excluded.updated_at`,
		p.SessionID, p.PageID, p.Path, p.Title, p.Status, p.JSON, p.UpdatedAt)
	if err != nil {
		return fmt.Errorf("save authoring page %q/%q: %w", p.SessionID, p.PageID, err)
	}
	return nil
}

func (r *sqliteRepository) ListPages(ctx context.Context, sessionID string) ([]PageDraft, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT session_id, page_id, path, title, status, json, updated_at FROM authoring_pages WHERE session_id = ? ORDER BY page_id`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list authoring pages %q: %w", sessionID, err)
	}
	defer rows.Close()
	var out []PageDraft
	for rows.Next() {
		var p PageDraft
		if err := rows.Scan(&p.SessionID, &p.PageID, &p.Path, &p.Title, &p.Status, &p.JSON, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan authoring page: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate authoring pages: %w", err)
	}
	return out, nil
}
