package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const activeRepoKey = "active_repo_id"

// RepoRecord represents a tracked repository in the registry.
type RepoRecord struct {
	ID           int64  `json:"id"`
	Path         string `json:"path"`
	Name         string `json:"name"`
	RemoteURL    string `json:"remote_url,omitempty"`
	AddedAt      string `json:"added_at"`
	LastOpenedAt string `json:"last_opened_at,omitempty"`
	Favorite     bool   `json:"favorite"`
}

// RepoStore defines persistence for repository registry state.
type RepoStore interface {
	List(ctx context.Context) ([]RepoRecord, error)
	GetByID(ctx context.Context, id int64) (*RepoRecord, error)
	GetByPath(ctx context.Context, path string) (*RepoRecord, error)
	Upsert(ctx context.Context, record RepoRecord) (RepoRecord, error)
	Delete(ctx context.Context, id int64) error
	GetActive(ctx context.Context) (*RepoRecord, error)
	SetActive(ctx context.Context, id int64) error
	ClearActive(ctx context.Context) error
	TouchLastOpened(ctx context.Context, id int64) error
}

// SQLiteRepoStore stores repository records in SQLite.
type SQLiteRepoStore struct {
	db  *sql.DB
	now func() time.Time
}

// NewSQLiteRepoStore creates a new repo store backed by SQLite.
func NewSQLiteRepoStore(db *sql.DB) *SQLiteRepoStore {
	return &SQLiteRepoStore{db: db, now: time.Now}
}

func (s *SQLiteRepoStore) List(ctx context.Context) ([]RepoRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, path, name, remote_url, added_at, last_opened_at, is_favorite
		FROM git_repos
		ORDER BY
			CASE WHEN last_opened_at IS NULL THEN 1 ELSE 0 END,
			last_opened_at DESC,
			added_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list repos: %w", err)
	}
	defer rows.Close()

	repos := make([]RepoRecord, 0)
	for rows.Next() {
		var repo RepoRecord
		var remote sql.NullString
		var lastOpened sql.NullString
		var favorite int
		if err := rows.Scan(&repo.ID, &repo.Path, &repo.Name, &remote, &repo.AddedAt, &lastOpened, &favorite); err != nil {
			return nil, fmt.Errorf("scan repo: %w", err)
		}
		if remote.Valid {
			repo.RemoteURL = remote.String
		}
		if lastOpened.Valid {
			repo.LastOpenedAt = lastOpened.String
		}
		repo.Favorite = favorite != 0
		repos = append(repos, repo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repos: %w", err)
	}
	return repos, nil
}

func (s *SQLiteRepoStore) GetByID(ctx context.Context, id int64) (*RepoRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, path, name, remote_url, added_at, last_opened_at, is_favorite
		FROM git_repos
		WHERE id = ?
	`, id)
	return scanRepoRow(row)
}

func (s *SQLiteRepoStore) GetByPath(ctx context.Context, path string) (*RepoRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, path, name, remote_url, added_at, last_opened_at, is_favorite
		FROM git_repos
		WHERE path = ?
	`, path)
	return scanRepoRow(row)
}

func (s *SQLiteRepoStore) Upsert(ctx context.Context, record RepoRecord) (RepoRecord, error) {
	if s.db == nil {
		return RepoRecord{}, fmt.Errorf("repo store not configured")
	}
	path := strings.TrimSpace(record.Path)
	if path == "" {
		return RepoRecord{}, fmt.Errorf("repo path is required")
	}
	name := strings.TrimSpace(record.Name)
	if name == "" {
		return RepoRecord{}, fmt.Errorf("repo name is required")
	}

	now := s.now().UTC().Format(time.RFC3339)
	var addedAt interface{} = strings.TrimSpace(record.AddedAt)
	if addedAt == "" {
		addedAt = nil
	}
	favorite := 0
	if record.Favorite {
		favorite = 1
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_repos (path, name, remote_url, added_at, last_opened_at, is_favorite)
		VALUES (?, ?, ?, COALESCE(?, strftime('%Y-%m-%dT%H:%M:%fZ','now')), ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			remote_url = CASE
				WHEN excluded.remote_url IS NOT NULL AND excluded.remote_url != '' THEN excluded.remote_url
				ELSE git_repos.remote_url
			END,
			last_opened_at = excluded.last_opened_at,
			is_favorite = excluded.is_favorite
	`, path, name, strings.TrimSpace(record.RemoteURL), addedAt, now, favorite)
	if err != nil {
		return RepoRecord{}, fmt.Errorf("upsert repo: %w", err)
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, path, name, remote_url, added_at, last_opened_at, is_favorite
		FROM git_repos
		WHERE path = ?
	`, path)
	repo, err := scanRepoRow(row)
	if err != nil {
		return RepoRecord{}, err
	}
	return *repo, nil
}

func (s *SQLiteRepoStore) Delete(ctx context.Context, id int64) error {
	if s.db == nil {
		return fmt.Errorf("repo store not configured")
	}
	if id <= 0 {
		return fmt.Errorf("repo id is required")
	}

	result, err := s.db.ExecContext(ctx, `DELETE FROM git_repos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete repo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err == nil && rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *SQLiteRepoStore) GetActive(ctx context.Context) (*RepoRecord, error) {
	id, err := s.activeID(ctx)
	if err != nil || id == 0 {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *SQLiteRepoStore) SetActive(ctx context.Context, id int64) error {
	if s.db == nil {
		return fmt.Errorf("repo store not configured")
	}
	if id <= 0 {
		return fmt.Errorf("repo id is required")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO git_repo_state (key, value)
		VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, activeRepoKey, fmt.Sprintf("%d", id))
	if err != nil {
		return fmt.Errorf("set active repo: %w", err)
	}
	return nil
}

func (s *SQLiteRepoStore) ClearActive(ctx context.Context) error {
	if s.db == nil {
		return fmt.Errorf("repo store not configured")
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM git_repo_state WHERE key = ?`, activeRepoKey)
	if err != nil {
		return fmt.Errorf("clear active repo: %w", err)
	}
	return nil
}

func (s *SQLiteRepoStore) TouchLastOpened(ctx context.Context, id int64) error {
	if s.db == nil {
		return fmt.Errorf("repo store not configured")
	}
	if id <= 0 {
		return fmt.Errorf("repo id is required")
	}

	now := s.now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `UPDATE git_repos SET last_opened_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("touch repo: %w", err)
	}
	return nil
}

func (s *SQLiteRepoStore) activeID(ctx context.Context) (int64, error) {
	row := s.db.QueryRowContext(ctx, `SELECT value FROM git_repo_state WHERE key = ?`, activeRepoKey)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("read active repo: %w", err)
	}
	var id int64
	if _, err := fmt.Sscanf(raw, "%d", &id); err != nil {
		return 0, fmt.Errorf("parse active repo: %w", err)
	}
	return id, nil
}

func scanRepoRow(row *sql.Row) (*RepoRecord, error) {
	var repo RepoRecord
	var remote sql.NullString
	var lastOpened sql.NullString
	var favorite int
	if err := row.Scan(&repo.ID, &repo.Path, &repo.Name, &remote, &repo.AddedAt, &lastOpened, &favorite); err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("scan repo: %w", err)
	}
	if remote.Valid {
		repo.RemoteURL = remote.String
	}
	if lastOpened.Valid {
		repo.LastOpenedAt = lastOpened.String
	}
	repo.Favorite = favorite != 0
	return &repo, nil
}
