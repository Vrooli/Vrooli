package manifest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

const manifestTimeFormat = time.RFC3339Nano

const (
	upsertManifestSQL = `
INSERT INTO manifests (
  skill_id, golden_slug,
  allowed_paths_json, content_rules_json,
  wildcard_allowed, convergence_target,
  template_version_pinned, skill_version_pinned,
  updated_at
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(skill_id, golden_slug) DO UPDATE SET
  allowed_paths_json      = excluded.allowed_paths_json,
  content_rules_json      = excluded.content_rules_json,
  wildcard_allowed        = excluded.wildcard_allowed,
  convergence_target      = excluded.convergence_target,
  template_version_pinned = excluded.template_version_pinned,
  skill_version_pinned    = excluded.skill_version_pinned,
  updated_at              = excluded.updated_at
`

	selectManifestSQL = `
SELECT skill_id, golden_slug,
       allowed_paths_json, content_rules_json,
       wildcard_allowed, convergence_target,
       template_version_pinned, skill_version_pinned,
       updated_at
FROM manifests
WHERE skill_id = ? AND golden_slug = ?
`

	listManifestsSQL = `
SELECT skill_id, golden_slug,
       allowed_paths_json, content_rules_json,
       wildcard_allowed, convergence_target,
       template_version_pinned, skill_version_pinned,
       updated_at
FROM manifests
ORDER BY skill_id ASC, golden_slug ASC
`

	upsertStaleOverrideSQL = `
INSERT INTO manifest_stale_overrides (skill_id, golden_slug, cleared_at)
VALUES (?, ?, ?)
ON CONFLICT(skill_id, golden_slug) DO UPDATE SET cleared_at = excluded.cleared_at
`

	selectStaleOverrideSQL = `
SELECT cleared_at FROM manifest_stale_overrides
WHERE skill_id = ? AND golden_slug = ?
`
)

// SQLExecutor is the narrow database surface this package's repository
// depends on. Both *sql.DB (used by repository unit tests via
// testutil/db.NewSQLite) and *database.RoutedDB (production main.go)
// satisfy it, so production wiring participates in per-request routing
// without forcing test fixtures to wrap their handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

func (s *sqliteRepository) Upsert(ctx context.Context, m Manifest) (Manifest, error) {
	if m.UpdatedAt.IsZero() {
		m.UpdatedAt = s.clock.Now().UTC()
	}
	if m.AllowedPaths == nil {
		m.AllowedPaths = []string{}
	}
	if m.ContentRules == nil {
		m.ContentRules = []ContentRule{}
	}
	allowedJSON, err := json.Marshal(m.AllowedPaths)
	if err != nil {
		return Manifest{}, fmt.Errorf("marshal allowed_paths: %w", err)
	}
	rulesJSON, err := json.Marshal(m.ContentRules)
	if err != nil {
		return Manifest{}, fmt.Errorf("marshal content_rules: %w", err)
	}
	_, err = s.db.ExecContext(ctx, upsertManifestSQL,
		m.SkillID, m.GoldenSlug,
		string(allowedJSON), string(rulesJSON),
		boolToInt(m.WildcardAllowed), int(m.ConvergenceTarget),
		m.TemplateVersionPinned, m.SkillVersionPinned,
		m.UpdatedAt.Format(manifestTimeFormat),
	)
	if err != nil {
		return Manifest{}, fmt.Errorf("upsert manifest (%q,%q): %w", m.SkillID, m.GoldenSlug, err)
	}
	return s.Get(ctx, m.SkillID, m.GoldenSlug)
}

func (s *sqliteRepository) Get(ctx context.Context, skillID, goldenSlug string) (Manifest, error) {
	row := s.db.QueryRowContext(ctx, selectManifestSQL, skillID, goldenSlug)
	m, err := scanManifest(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Manifest{}, ErrManifestNotFound{SkillID: skillID, GoldenSlug: goldenSlug}
	}
	if err != nil {
		return Manifest{}, fmt.Errorf("get manifest (%q,%q): %w", skillID, goldenSlug, err)
	}
	return m, nil
}

func (s *sqliteRepository) List(ctx context.Context) ([]Manifest, error) {
	rows, err := s.db.QueryContext(ctx, listManifestsSQL)
	if err != nil {
		return nil, fmt.Errorf("list manifests: %w", err)
	}
	defer rows.Close()

	var out []Manifest
	for rows.Next() {
		m, err := scanManifest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan manifest: %w", err)
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manifests: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) ClearStaleOverride(ctx context.Context, skillID, goldenSlug string, at time.Time) error {
	if at.IsZero() {
		at = s.clock.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, upsertStaleOverrideSQL,
		skillID, goldenSlug, at.Format(manifestTimeFormat),
	)
	if err != nil {
		return fmt.Errorf("upsert stale override (%q,%q): %w", skillID, goldenSlug, err)
	}
	return nil
}

func (s *sqliteRepository) GetStaleOverride(ctx context.Context, skillID, goldenSlug string) (time.Time, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, selectStaleOverrideSQL, skillID, goldenSlug).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get stale override (%q,%q): %w", skillID, goldenSlug, err)
	}
	t, err := time.Parse(manifestTimeFormat, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse cleared_at %q: %w", raw, err)
	}
	return t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanManifest(s rowScanner) (Manifest, error) {
	var (
		m              Manifest
		allowedRaw     string
		rulesRaw       string
		wildcardInt    int
		convergenceInt int
		updatedRaw     string
	)
	if err := s.Scan(
		&m.SkillID, &m.GoldenSlug,
		&allowedRaw, &rulesRaw,
		&wildcardInt, &convergenceInt,
		&m.TemplateVersionPinned, &m.SkillVersionPinned,
		&updatedRaw,
	); err != nil {
		return Manifest{}, err
	}
	if err := json.Unmarshal([]byte(allowedRaw), &m.AllowedPaths); err != nil {
		return Manifest{}, fmt.Errorf("unmarshal allowed_paths: %w", err)
	}
	if err := json.Unmarshal([]byte(rulesRaw), &m.ContentRules); err != nil {
		return Manifest{}, fmt.Errorf("unmarshal content_rules: %w", err)
	}
	m.WildcardAllowed = wildcardInt != 0
	m.ConvergenceTarget = ConvergenceTarget(convergenceInt)
	t, err := time.Parse(manifestTimeFormat, updatedRaw)
	if err != nil {
		return Manifest{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	m.UpdatedAt = t
	return m, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
