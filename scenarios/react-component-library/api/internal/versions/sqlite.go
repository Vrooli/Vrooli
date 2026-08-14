package versions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

type sqliteRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk schedule.Clock) Repository {
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
	if v.CreatedAt.IsZero() {
		v.CreatedAt = v.RecordedAt
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, v.ID, v.ComponentID, v.LibraryID, v.Version, v.Status, v.SourcePath, v.Content, v.ContentSHA256, v.ChangelogMD, v.RecordedAt.UTC().Format(timeFormat), v.CreatedAt.UTC().Format(timeFormat), formatOptionalTime(v.ReleasedAt)); err != nil {
		if isUniqueVersionError(err) {
			return Version{}, ErrVersionExists{ComponentID: v.ComponentID, Version: v.Version}
		}
		return Version{}, fmt.Errorf("insert version: %w", err)
	}
	return v, nil
}

func (s *sqliteRepository) Latest(ctx context.Context, componentID string) (Version, error) {
	rows, err := s.List(ctx, ListQuery{ComponentID: componentID, Limit: 1})
	if err != nil {
		return Version{}, err
	}
	if len(rows) == 0 {
		return Version{}, nil
	}
	return rows[0], nil
}

func (s *sqliteRepository) List(ctx context.Context, q ListQuery) ([]Version, error) {
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at
FROM component_versions
WHERE (component_id = ? OR library_id = ?)
`, q.ComponentID, q.ComponentID)
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
	for i := range out {
		out[i].RequiredTokens, err = s.requiredTokens(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].RequiredTokenPatterns, err = s.requiredTokenPatterns(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
	}
	sortVersions(out)
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (s *sqliteRepository) Get(ctx context.Context, componentID, version string) (Version, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at
FROM component_versions
WHERE (component_id = ? OR library_id = ?) AND version = ?
ORDER BY version DESC, id ASC
LIMIT 1
`, componentID, componentID, version)
	v, err := scanVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Version{}, ErrVersionNotFound{ComponentID: componentID, Version: version}
	}
	if err != nil {
		return Version{}, fmt.Errorf("get version: %w", err)
	}
	v.RequiredTokens, err = s.requiredTokens(ctx, v.ID)
	if err != nil {
		return Version{}, err
	}
	v.RequiredTokenPatterns, err = s.requiredTokenPatterns(ctx, v.ID)
	if err != nil {
		return Version{}, err
	}
	return v, nil
}

func (s *sqliteRepository) requiredTokens(ctx context.Context, versionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT property FROM component_version_required_tokens WHERE version_id = ? ORDER BY property`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list version required tokens: %w", err)
	}
	defer rows.Close()
	var properties []string
	for rows.Next() {
		var property string
		if err := rows.Scan(&property); err != nil {
			return nil, err
		}
		properties = append(properties, property)
	}
	return properties, rows.Err()
}

func (s *sqliteRepository) requiredTokenPatterns(ctx context.Context, versionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT pattern FROM component_version_required_token_patterns WHERE version_id = ? ORDER BY pattern`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list version required token patterns: %w", err)
	}
	defer rows.Close()
	var patterns []string
	for rows.Next() {
		var pattern string
		if err := rows.Scan(&pattern); err != nil {
			return nil, err
		}
		patterns = append(patterns, pattern)
	}
	return patterns, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanVersion(s rowScanner) (Version, error) {
	var (
		v          Version
		recordedAt string
		createdAt  string
		releasedAt string
	)
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &v.Status, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &recordedAt, &createdAt, &releasedAt); err != nil {
		return Version{}, err
	}
	t, err := time.Parse(timeFormat, recordedAt)
	if err != nil {
		return Version{}, fmt.Errorf("parse recorded_at: %w", err)
	}
	v.RecordedAt = t
	if createdAt != "" {
		created, err := time.Parse(timeFormat, createdAt)
		if err != nil {
			return Version{}, fmt.Errorf("parse created_at: %w", err)
		}
		v.CreatedAt = created
	}
	if releasedAt != "" {
		rt, err := time.Parse(timeFormat, releasedAt)
		if err != nil {
			return Version{}, fmt.Errorf("parse released_at: %w", err)
		}
		v.ReleasedAt = rt
	}
	return v, nil
}

func isUniqueVersionError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique constraint failed: component_versions.component_id, component_versions.version")
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeFormat)
}
