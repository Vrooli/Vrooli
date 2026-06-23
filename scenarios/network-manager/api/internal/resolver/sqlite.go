package resolver

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) SaveBackend(ctx context.Context, cfg BackendConfig) (BackendConfig, error) {
	now := time.Now().UTC()
	if cfg.CreatedAt.IsZero() {
		cfg.CreatedAt = now
	}
	cfg.UpdatedAt = now
	if _, err := r.db.ExecContext(ctx, `
INSERT INTO resolver_backends (backend, base_url, username, token_ref, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(backend) DO UPDATE SET
  base_url = excluded.base_url,
  username = excluded.username,
  token_ref = excluded.token_ref,
  updated_at = excluded.updated_at
`, cfg.Backend, cfg.BaseURL, cfg.Username, cfg.TokenRef, cfg.CreatedAt.UTC().Format(TimeFormat), cfg.UpdatedAt.UTC().Format(TimeFormat)); err != nil {
		return BackendConfig{}, fmt.Errorf("save resolver backend %q: %w", cfg.Backend, err)
	}
	return cfg, nil
}

func (r *sqliteRepository) GetBackend(ctx context.Context, backend string) (BackendConfig, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT backend, base_url, username, token_ref, created_at, updated_at
FROM resolver_backends
WHERE backend = ?
`, backend)
	cfg, err := scanBackend(row)
	if errors.Is(err, sql.ErrNoRows) {
		return BackendConfig{}, ErrNotFound
	}
	if err != nil {
		return BackendConfig{}, err
	}
	return cfg, nil
}

func (r *sqliteRepository) UpdateUpstreams(ctx context.Context, backend string, upstreams []string) error {
	if _, err := r.db.ExecContext(ctx, `DELETE FROM resolver_upstreams WHERE backend = ?`, backend); err != nil {
		return fmt.Errorf("clear resolver upstreams for %q: %w", backend, err)
	}
	now := time.Now().UTC().Format(TimeFormat)
	for i, upstream := range upstreams {
		if _, err := r.db.ExecContext(ctx, `
INSERT INTO resolver_upstreams (id, backend, upstream, sort_order, updated_at)
VALUES (?, ?, ?, ?, ?)
`, uuid.NewString(), backend, upstream, i, now); err != nil {
			return fmt.Errorf("insert resolver upstream %q: %w", upstream, err)
		}
	}
	return nil
}

func (r *sqliteRepository) GetUpstreams(ctx context.Context, backend string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT upstream
FROM resolver_upstreams
WHERE backend = ?
ORDER BY sort_order ASC
`, backend)
	if err != nil {
		return nil, fmt.Errorf("list resolver upstreams: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var upstream string
		if err := rows.Scan(&upstream); err != nil {
			return nil, fmt.Errorf("scan resolver upstream: %w", err)
		}
		out = append(out, upstream)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate resolver upstreams: %w", err)
	}
	return out, nil
}

type backendScanner interface {
	Scan(dest ...any) error
}

func scanBackend(row backendScanner) (BackendConfig, error) {
	var cfg BackendConfig
	var createdAt, updatedAt string
	if err := row.Scan(&cfg.Backend, &cfg.BaseURL, &cfg.Username, &cfg.TokenRef, &createdAt, &updatedAt); err != nil {
		return BackendConfig{}, err
	}
	var err error
	cfg.CreatedAt, err = time.Parse(TimeFormat, createdAt)
	if err != nil {
		return BackendConfig{}, fmt.Errorf("parse resolver backend created_at: %w", err)
	}
	cfg.UpdatedAt, err = time.Parse(TimeFormat, updatedAt)
	if err != nil {
		return BackendConfig{}, fmt.Errorf("parse resolver backend updated_at: %w", err)
	}
	return cfg, nil
}
