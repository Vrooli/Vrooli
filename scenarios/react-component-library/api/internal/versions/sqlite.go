package versions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"react-component-library/internal/clock"

	"github.com/google/uuid"
)

type sqliteRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

const timeFormat = time.RFC3339Nano

func (s *sqliteRepository) Insert(ctx context.Context, v Version) (Version, error) {
	if v.ID == "" {
		v.ID = uuid.NewString()
	}
	if v.RecordedAt.IsZero() {
		v.RecordedAt = s.clock.Now().UTC()
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, v.ID, v.ComponentID, v.LibraryID, v.Version, v.Status, v.SourcePath, v.Content, v.ContentSHA256, v.ChangelogMD, v.RecordedAt.UTC().Format(timeFormat), formatOptionalTime(v.ReleasedAt)); err != nil {
		return Version{}, fmt.Errorf("insert version: %w", err)
	}
	return v, nil
}

func (s *sqliteRepository) Latest(ctx context.Context, componentID string) (Version, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at
FROM component_versions
WHERE component_id = ?
ORDER BY indexed_at DESC, id ASC
LIMIT 1
`, componentID)
	v, err := scanVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, nil
	}
	if err != nil {
		return Version{}, fmt.Errorf("latest version for %q: %w", componentID, err)
	}
	return v, nil
}

func (s *sqliteRepository) List(ctx context.Context, q ListQuery) ([]Version, error) {
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at
FROM component_versions
WHERE component_id = ?
ORDER BY indexed_at DESC, id ASC
LIMIT ?
`, q.ComponentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	var out []Version
	for rows.Next() {
		v, err := scanVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate versions: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) Get(ctx context.Context, componentID, version string) (Version, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at
FROM component_versions
WHERE component_id = ? AND version = ?
ORDER BY indexed_at DESC, id ASC
LIMIT 1
`, componentID, version)
	v, err := scanVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrVersionNotFound{ComponentID: componentID, Version: version}
	}
	if err != nil {
		return Version{}, fmt.Errorf("get version: %w", err)
	}
	return v, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVersion(s rowScanner) (Version, error) {
	var (
		v          Version
		recordedAt string
		releasedAt string
	)
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &v.Status, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &recordedAt, &releasedAt); err != nil {
		return Version{}, err
	}
	t, err := time.Parse(timeFormat, recordedAt)
	if err != nil {
		return Version{}, fmt.Errorf("parse recorded_at: %w", err)
	}
	v.RecordedAt = t
	if releasedAt != "" {
		rt, err := time.Parse(timeFormat, releasedAt)
		if err != nil {
			return Version{}, fmt.Errorf("parse released_at: %w", err)
		}
		v.ReleasedAt = rt
	}
	return v, nil
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeFormat)
}
