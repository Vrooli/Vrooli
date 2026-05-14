package components

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"react-component-library/internal/clock"

	"github.com/google/uuid"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    *sql.DB
	clock clock.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk clock.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

var _ Repository = (*sqliteRepository)(nil)

// timeFormat matches the convention used by internal/notes: RFC3339
// with nanoseconds, parseable by time.Parse on read.
const timeFormat = time.RFC3339Nano

// tagSep is the in-row tag separator. Comma chosen for readability in
// SQL queries; tags themselves are forbidden from containing commas by
// the indexer's header parser.
const tagSep = ","

func (s *sqliteRepository) Upsert(ctx context.Context, in UpsertInput) (Component, error) {
	manifest := ComponentManifest{
		LibraryID:     in.LibraryID,
		Slug:          in.Slug,
		DisplayName:   in.DisplayName,
		Description:   in.Description,
		ManifestPath:  in.ManifestPath,
		LatestVersion: firstNonEmpty(in.LatestVersion, in.Version),
		DraftVersion:  in.DraftVersion,
		Tags:          in.Tags,
	}
	if manifest.Slug == "" {
		manifest.Slug = slugFromLibraryID(in.LibraryID)
	}
	c, err := s.upsertComponent(ctx, manifest, in.SourcePath)
	if err != nil {
		return Component{}, err
	}
	if err := s.replaceHeaders(ctx, c.ID, in.Headers); err != nil {
		return Component{}, err
	}
	return s.Get(ctx, c.ID)
}

func (s *sqliteRepository) UpsertManifest(ctx context.Context, in IndexManifestInput) (Component, error) {
	if strings.TrimSpace(in.Manifest.LibraryID) == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.Manifest.ManifestPath, Field: "libraryId", Reason: "required"}
	}
	sourcePath := ""
	for _, v := range in.Versions {
		if v.Version == in.Manifest.LatestVersion {
			sourcePath = v.SourcePath
			break
		}
	}
	c, err := s.upsertComponent(ctx, in.Manifest, sourcePath)
	if err != nil {
		return Component{}, err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_versions WHERE component_id = ?`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component versions for %q: %w", c.ID, err)
	}
	now := s.clock.Now().UTC()
	for _, v := range in.Versions {
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
		v.ComponentID = c.ID
		v.LibraryID = c.LibraryID
		if v.IndexedAt.IsZero() {
			v.IndexedAt = now
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(component_id, version) DO UPDATE SET
  library_id = excluded.library_id,
  status = excluded.status,
  source_path = excluded.source_path,
  content = excluded.content,
  content_sha256 = excluded.content_sha256,
  changelog_md = excluded.changelog_md,
  indexed_at = excluded.indexed_at,
  released_at = excluded.released_at
`, v.ID, v.ComponentID, v.LibraryID, v.Version, string(v.Status), v.SourcePath, v.Content, v.ContentSHA256,
			v.ChangelogMD, v.IndexedAt.UTC().Format(timeFormat), formatOptionalTime(v.ReleasedAt)); err != nil {
			return Component{}, fmt.Errorf("upsert component version %s@%s: %w", c.LibraryID, v.Version, err)
		}
	}
	return s.Get(ctx, c.ID)
}

func (s *sqliteRepository) upsertComponent(ctx context.Context, in ComponentManifest, sourcePath string) (Component, error) {
	if strings.TrimSpace(in.LibraryID) == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.ManifestPath, Field: "libraryId", Reason: "required"}
	}
	now := s.clock.Now().UTC()
	tagsCol := strings.Join(in.Tags, tagSep)

	existing, err := s.GetByLibraryID(ctx, in.LibraryID)
	if err != nil && !errors.As(err, &ErrComponentNotFound{}) {
		return Component{}, err
	}

	id := existing.ID
	indexedAt := existing.IndexedAt
	if id == "" {
		id = uuid.NewString()
		indexedAt = now
	}

	slug := strings.TrimSpace(in.Slug)
	if slug == "" {
		slug = slugFromLibraryID(in.LibraryID)
	}

	if _, err := s.db.ExecContext(ctx, `
INSERT INTO components (id, library_id, slug, display_name, description, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(library_id) DO UPDATE SET
  slug         = excluded.slug,
  display_name = excluded.display_name,
  description  = excluded.description,
  source_path  = excluded.source_path,
  version      = excluded.version,
  latest_version = excluded.latest_version,
  draft_version = excluded.draft_version,
  manifest_path = excluded.manifest_path,
  tags         = excluded.tags,
  updated_at   = excluded.updated_at
`,
		id, in.LibraryID, slug, in.DisplayName, in.Description, sourcePath, in.LatestVersion, in.LatestVersion, in.DraftVersion, in.ManifestPath,
		tagsCol, indexedAt.Format(timeFormat), now.Format(timeFormat),
	); err != nil {
		return Component{}, fmt.Errorf("upsert component %q: %w", in.LibraryID, err)
	}
	return s.Get(ctx, id)
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Component, error) {
	row := s.db.QueryRowContext(ctx, selectComponentByIDSQL, id)
	c, err := scanComponent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: id}
	}
	if err != nil {
		return Component{}, fmt.Errorf("get component %q: %w", id, err)
	}
	if err := s.loadHeaders(ctx, &c); err != nil {
		return Component{}, err
	}
	return c, nil
}

func (s *sqliteRepository) GetByLibraryID(ctx context.Context, libraryID string) (Component, error) {
	row := s.db.QueryRowContext(ctx, selectComponentByLibraryIDSQL, libraryID)
	c, err := scanComponent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: libraryID}
	}
	if err != nil {
		return Component{}, fmt.Errorf("get component by libraryId %q: %w", libraryID, err)
	}
	if err := s.loadHeaders(ctx, &c); err != nil {
		return Component{}, err
	}
	return c, nil
}

func (s *sqliteRepository) List(ctx context.Context, q SearchQuery) ([]Component, error) {
	limit := q.Limit
	if limit <= 0 {
		return nil, nil
	}
	var (
		clauses []string
		args    []any
	)
	if match := strings.TrimSpace(q.Match); match != "" {
		pat := "%" + strings.ToLower(match) + "%"
		clauses = append(clauses, `(
		  lower(library_id)   LIKE ? OR
		  lower(display_name) LIKE ? OR
		  lower(description)  LIKE ? OR
		  lower(source_path)  LIKE ?
		)`)
		args = append(args, pat, pat, pat, pat)
	}
	if tag := strings.TrimSpace(q.Tag); tag != "" {
		// Match the tag as a token within the comma-joined column.
		clauses = append(clauses, `(',' || lower(tags) || ',') LIKE ?`)
		args = append(args, "%,"+strings.ToLower(tag)+",%")
	}
	// Multi-tag OR: any-of semantics, per req SF-002. Empty / whitespace-
	// only entries are dropped silently so callers can pass raw splits.
	var multiTagPredicates []string
	for _, t := range q.Tags {
		trimmed := strings.TrimSpace(t)
		if trimmed == "" {
			continue
		}
		multiTagPredicates = append(multiTagPredicates, `(',' || lower(tags) || ',') LIKE ?`)
		args = append(args, "%,"+strings.ToLower(trimmed)+",%")
	}
	if len(multiTagPredicates) > 0 {
		clauses = append(clauses, "("+strings.Join(multiTagPredicates, " OR ")+")")
	}
	if cat := strings.TrimSpace(q.Category); cat != "" {
		clauses = append(clauses, `EXISTS (
		  SELECT 1 FROM component_headers ch
		  WHERE ch.component_id = components.id
		    AND ch.field = 'category'
		    AND lower(ch.value) = ?
		)`)
		args = append(args, strings.ToLower(cat))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	orderBy := "ORDER BY indexed_at DESC, library_id ASC"
	if strings.TrimSpace(q.Match) != "" {
		// Match queries surface a discoverability list — alphabetical
		// by display_name (case-insensitive) matches the spec's "ordered
		// by name" wording (req SF-001).
		orderBy = "ORDER BY display_name COLLATE NOCASE ASC, library_id ASC"
	}
	query := fmt.Sprintf(`
SELECT id, library_id, slug, display_name, description, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
FROM components
%s
%s
LIMIT ?
`, where, orderBy)
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list components: %w", err)
	}
	defer rows.Close()

	var out []Component
	for rows.Next() {
		c, err := scanComponent(rows)
		if err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate components: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) replaceHeaders(ctx context.Context, componentID string, headers map[string]string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_headers WHERE component_id = ?`, componentID); err != nil {
		return fmt.Errorf("clear headers for %q: %w", componentID, err)
	}
	for field, value := range headers {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO component_headers (component_id, field, value) VALUES (?, ?, ?)`, componentID, field, value); err != nil {
			return fmt.Errorf("insert header %s=%q for %q: %w", field, value, componentID, err)
		}
	}
	return nil
}

func (s *sqliteRepository) loadHeaders(ctx context.Context, c *Component) error {
	rows, err := s.db.QueryContext(ctx, `SELECT field, value FROM component_headers WHERE component_id = ?`, c.ID)
	if err != nil {
		return fmt.Errorf("load headers for %q: %w", c.ID, err)
	}
	defer rows.Close()
	headers := map[string]string{}
	for rows.Next() {
		var f, v string
		if err := rows.Scan(&f, &v); err != nil {
			return fmt.Errorf("scan header for %q: %w", c.ID, err)
		}
		headers[f] = v
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate headers for %q: %w", c.ID, err)
	}
	c.Headers = headers
	return nil
}

func (s *sqliteRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	if len(keep) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM components`)
		if err != nil {
			return 0, fmt.Errorf("delete all components: %w", err)
		}
		n, _ := res.RowsAffected()
		return int(n), nil
	}
	placeholders := strings.Repeat("?,", len(keep))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(keep))
	for i, k := range keep {
		args[i] = k
	}
	res, err := s.db.ExecContext(ctx, fmt.Sprintf(
		`DELETE FROM components WHERE library_id NOT IN (%s)`, placeholders), args...)
	if err != nil {
		return 0, fmt.Errorf("delete missing components: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (s *sqliteRepository) ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at
FROM component_versions
WHERE component_id = ?
ORDER BY indexed_at DESC, version DESC
LIMIT ?
`, componentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list component versions: %w", err)
	}
	defer rows.Close()
	var out []ComponentVersion
	for rows.Next() {
		v, err := scanComponentVersion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *sqliteRepository) GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, released_at
FROM component_versions
WHERE component_id = ? AND version = ?
`, componentID, version)
	v, err := scanComponentVersion(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ComponentVersion{}, ErrComponentNotFound{IDOrLibraryID: componentID + "@" + version}
	}
	if err != nil {
		return ComponentVersion{}, fmt.Errorf("get component version: %w", err)
	}
	return v, nil
}

const (
	selectComponentByIDSQL = `
SELECT id, library_id, slug, display_name, description, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
FROM components WHERE id = ?
`
	selectComponentByLibraryIDSQL = `
SELECT id, library_id, slug, display_name, description, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
FROM components WHERE library_id = ?
`
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanComponent(s rowScanner) (Component, error) {
	var (
		c          Component
		tagsRaw    string
		indexedRaw string
		updatedRaw string
	)
	if err := s.Scan(&c.ID, &c.LibraryID, &c.Slug, &c.DisplayName, &c.Description, &c.SourcePath, &c.Version, &c.LatestVersion, &c.DraftVersion, &c.ManifestPath, &tagsRaw, &indexedRaw, &updatedRaw); err != nil {
		return Component{}, err
	}
	if tagsRaw != "" {
		c.Tags = strings.Split(tagsRaw, tagSep)
	}
	indexed, err := time.Parse(timeFormat, indexedRaw)
	if err != nil {
		return Component{}, fmt.Errorf("parse indexed_at %q: %w", indexedRaw, err)
	}
	updated, err := time.Parse(timeFormat, updatedRaw)
	if err != nil {
		return Component{}, fmt.Errorf("parse updated_at %q: %w", updatedRaw, err)
	}
	c.IndexedAt = indexed
	c.UpdatedAt = updated
	return c, nil
}

func scanComponentVersion(s rowScanner) (ComponentVersion, error) {
	var v ComponentVersion
	var statusRaw, indexedRaw, releasedRaw string
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &statusRaw, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &indexedRaw, &releasedRaw); err != nil {
		return ComponentVersion{}, err
	}
	v.Status = ComponentVersionStatus(statusRaw)
	indexed, err := time.Parse(timeFormat, indexedRaw)
	if err != nil {
		return ComponentVersion{}, fmt.Errorf("parse indexed_at %q: %w", indexedRaw, err)
	}
	v.IndexedAt = indexed
	if releasedRaw != "" {
		released, err := time.Parse(timeFormat, releasedRaw)
		if err != nil {
			return ComponentVersion{}, fmt.Errorf("parse released_at %q: %w", releasedRaw, err)
		}
		v.ReleasedAt = released
	}
	return v, nil
}

func formatOptionalTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeFormat)
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func slugFromLibraryID(libraryID string) string {
	if _, slug, ok := strings.Cut(libraryID, ":"); ok && strings.TrimSpace(slug) != "" {
		return strings.TrimSpace(slug)
	}
	return strings.TrimSpace(libraryID)
}
