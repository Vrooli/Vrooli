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

	"react-component-library/internal/sqliterows"
	"react-component-library/internal/textutil"
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
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, v.ID, v.ComponentID, v.LibraryID, v.Version, v.Status, v.SourcePath, v.Content, v.ContentSHA256, v.ChangelogMD, v.RecordedAt.UTC().Format(timeFormat), v.CreatedAt.UTC().Format(timeFormat), formatOptionalTime(v.ReleasedAt), textutil.FirstNonBlank(v.Presence, "materialized")); err != nil {
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
	out, err := sqliterows.Collect(ctx, s.db, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence
FROM component_versions
WHERE (component_id = ? OR library_id = ?)
`, []any{q.ComponentID, q.ComponentID}, false, func(row sqliterows.Scanner) (Version, error) {
		version, err := scanVersion(row)
		if err != nil {
			return Version{}, fmt.Errorf("scan version: %w", err)
		}
		return version, nil
	}, sqliterows.ErrorContext{Query: "list versions", Iterate: "iterate versions"})
	if err != nil {
		return nil, err
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
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence
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
	properties, err := sqliterows.Collect(ctx, s.db, `SELECT property FROM component_version_required_tokens WHERE version_id = ? ORDER BY property`, []any{versionID}, false, func(row sqliterows.Scanner) (string, error) {
		var property string
		return property, row.Scan(&property)
	}, sqliterows.ErrorContext{Query: "list version required tokens"})
	if err != nil {
		return nil, err
	}
	return properties, nil
}

func (s *sqliteRepository) requiredTokenPatterns(ctx context.Context, versionID string) ([]string, error) {
	patterns, err := sqliterows.Collect(ctx, s.db, `SELECT pattern FROM component_version_required_token_patterns WHERE version_id = ? ORDER BY pattern`, []any{versionID}, false, func(row sqliterows.Scanner) (string, error) {
		var pattern string
		return pattern, row.Scan(&pattern)
	}, sqliterows.ErrorContext{Query: "list version required token patterns"})
	if err != nil {
		return nil, err
	}
	return patterns, nil
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
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &v.Status, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &recordedAt, &createdAt, &releasedAt, &v.Presence); err != nil {
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
