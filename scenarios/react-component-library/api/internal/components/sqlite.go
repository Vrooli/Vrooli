package components

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// sqliteRepository is the production Repository impl. Unexported so
// callers depend on the Repository interface — tests substitute the
// fake without reaching inside the struct.
type sqliteRepository struct {
	db    *sql.DB
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db *sql.DB, clk schedule.Clock) Repository {
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
		Slot:          in.Slot,
		Category:      firstNonEmpty(in.Category, in.Headers["category"]),
		ManifestPath:  in.ManifestPath,
		LatestVersion: firstNonEmpty(in.LatestVersion, in.Version),
		DraftVersion:  in.DraftVersion,
		Tags:          in.Tags,
		DesignStyles:  in.DesignStyles,
		AssetKind:     in.AssetKind,
		Dependencies:  in.Dependencies,
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
	if err := s.replaceDesignAffinities(ctx, c.ID, in.DesignStyles); err != nil {
		return Component{}, err
	}
	if err := s.replaceAssetDependencies(ctx, c.ID, in.Dependencies); err != nil {
		return Component{}, err
	}
	return s.Get(ctx, c.ID)
}

func (s *sqliteRepository) UpsertManifest(ctx context.Context, in IndexManifestInput) (Component, error) {
	if strings.TrimSpace(in.Manifest.LibraryID) == "" {
		return Component{}, ErrInvalidHeader{SourcePath: in.Manifest.ManifestPath, Field: "libraryId", Reason: "required"}
	}
	if strings.TrimSpace(in.Manifest.Category) == "" {
		in.Manifest.Category = strings.TrimSpace(in.Headers["category"])
	}
	if err := s.checkReleasedVersionHashes(ctx, in); err != nil {
		return Component{}, err
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
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_files WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ?)`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component version files for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_required_tokens WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ?)`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component version required tokens for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_required_token_patterns WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ?)`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component version required token patterns for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_parity_reports WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ?)`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component parity reports for %q: %w", c.ID, err)
	}
	existingVersions, err := s.versionMetadata(ctx, c.ID)
	if err != nil {
		return Component{}, err
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_stories WHERE component_id = ?`, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component stories for %q: %w", c.ID, err)
	}
	now := s.clock.Now().UTC()
	for _, v := range in.Versions {
		previous, existed := existingVersions[v.Version]
		if existed {
			v.ID = previous.id
		}
		if v.ID == "" {
			v.ID = uuid.NewString()
		}
		v.ComponentID = c.ID
		v.LibraryID = c.LibraryID
		if v.IndexedAt.IsZero() {
			v.IndexedAt = now
		}
		createdAt := previous.createdAt
		if createdAt == "" {
			createdAt = now.Format(timeFormat)
		}
		releasedAt := previous.releasedAt
		if v.Version == in.Manifest.LatestVersion && releasedAt == "" {
			releasedAt = now.Format(timeFormat)
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_versions
  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(component_id, version) DO UPDATE SET
  library_id = excluded.library_id,
  status = excluded.status,
  source_path = excluded.source_path,
  content = excluded.content,
  content_sha256 = excluded.content_sha256,
  changelog_md = excluded.changelog_md,
  indexed_at = excluded.indexed_at,
  released_at = CASE
    WHEN component_versions.released_at <> '' THEN component_versions.released_at
    ELSE excluded.released_at
  END
`, v.ID, v.ComponentID, v.LibraryID, v.Version, string(v.Status), v.SourcePath, v.Content, v.ContentSHA256,
			v.ChangelogMD, v.IndexedAt.UTC().Format(timeFormat), createdAt, releasedAt); err != nil {
			return Component{}, fmt.Errorf("upsert component version %s@%s: %w", c.LibraryID, v.Version, err)
		}
		for _, file := range v.Files {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO component_version_files (version_id, path, content, content_sha256, is_entry, slot) VALUES (?, ?, ?, ?, ?, ?)`, v.ID, file.Path, file.Content, file.ContentSHA256, file.IsEntry, file.Slot); err != nil {
				return Component{}, fmt.Errorf("upsert component version file %s@%s/%s: %w", c.LibraryID, v.Version, file.Path, err)
			}
		}
		for _, property := range v.RequiredTokens {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO component_version_required_tokens (version_id, property) VALUES (?, ?)`, v.ID, property); err != nil {
				return Component{}, fmt.Errorf("upsert component version required token %s@%s/%s: %w", c.LibraryID, v.Version, property, err)
			}
		}
		for _, pattern := range v.RequiredTokenPatterns {
			if _, err := s.db.ExecContext(ctx, `INSERT INTO component_version_required_token_patterns (version_id, pattern) VALUES (?, ?)`, v.ID, pattern); err != nil {
				return Component{}, fmt.Errorf("upsert component version required token pattern %s@%s/%s: %w", c.LibraryID, v.Version, pattern, err)
			}
		}
		if v.ParityReport != nil {
			report, err := json.Marshal(v.ParityReport)
			if err != nil {
				return Component{}, fmt.Errorf("encode component parity report %s@%s: %w", c.LibraryID, v.Version, err)
			}
			if _, err := s.db.ExecContext(ctx, `INSERT INTO component_version_parity_reports (version_id, report_json) VALUES (?, ?)`, v.ID, string(report)); err != nil {
				return Component{}, fmt.Errorf("upsert component parity report %s@%s: %w", c.LibraryID, v.Version, err)
			}
		}
	}
	if len(in.Versions) > 0 {
		placeholders := strings.TrimRight(strings.Repeat("?,", len(in.Versions)), ",")
		args := make([]any, 0, len(in.Versions)+1)
		args = append(args, c.ID)
		for _, v := range in.Versions {
			args = append(args, v.Version)
		}
		deleteStale := `DELETE FROM component_versions WHERE component_id = ? AND version NOT IN (` + placeholders + `)`
		if _, err := s.db.ExecContext(ctx, deleteStale, args...); err != nil {
			return Component{}, fmt.Errorf("reconcile stale component versions for %q: %w", c.ID, err)
		}
	}
	for _, story := range in.Stories {
		if story.ID == "" {
			story.ID = uuid.NewString()
		}
		story.ComponentID = c.ID
		story.LibraryID = c.LibraryID
		if story.IndexedAt.IsZero() {
			story.IndexedAt = now
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_stories (id, component_id, library_id, version, schema_version, kind, title, args_json, environment_json, stories_json, contract_json, source_path, indexed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(component_id, version) DO UPDATE SET
  library_id=excluded.library_id, schema_version=excluded.schema_version, kind=excluded.kind, title=excluded.title,
  args_json=excluded.args_json, environment_json=excluded.environment_json, stories_json=excluded.stories_json,
  contract_json=excluded.contract_json, source_path=excluded.source_path, indexed_at=excluded.indexed_at
`, story.ID, story.ComponentID, story.LibraryID, story.Version, story.SchemaVersion, string(story.Kind), story.Title, story.ArgsJSON, story.EnvironmentJSON, story.StoriesJSON, story.ContractJSON, story.SourcePath, story.IndexedAt.UTC().Format(timeFormat)); err != nil {
			return Component{}, fmt.Errorf("upsert component story %s@%s: %w", c.LibraryID, story.Version, err)
		}
	}
	if err := s.replaceHeaders(ctx, c.ID, in.Headers); err != nil {
		return Component{}, err
	}
	if err := s.replaceDesignAffinities(ctx, c.ID, in.Manifest.DesignStyles); err != nil {
		return Component{}, err
	}
	if err := s.replaceAssetDependencies(ctx, c.ID, in.Manifest.Dependencies); err != nil {
		return Component{}, err
	}
	return s.Get(ctx, c.ID)
}

func (s *sqliteRepository) checkReleasedVersionHashes(ctx context.Context, in IndexManifestInput) error {
	// The entry source is the immutable release artifact. Declarative companion
	// contracts are re-indexed independently so validation vocabulary can evolve
	// without silently changing the code consumers pinned by source hash.
	for _, version := range in.Versions {
		var status, recorded string
		err := s.db.QueryRowContext(ctx, `SELECT v.status, v.content_sha256 FROM component_versions v JOIN components c ON c.id = v.component_id WHERE c.library_id = ? AND v.version = ?`, in.Manifest.LibraryID, version.Version).Scan(&status, &recorded)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return fmt.Errorf("check released version %s@%s: %w", in.Manifest.LibraryID, version.Version, err)
		}
		if status == string(VersionStatusReleased) && recorded != "" && !releaseHashMatches(recorded, version.Content, version.ContentSHA256) {
			return ErrReleasedVersionMutated{ComponentID: in.Manifest.LibraryID, Version: version.Version, Recorded: recorded, Incoming: version.ContentSHA256}
		}
		if status != string(VersionStatusReleased) {
			continue
		}
	}
	return nil
}

func releaseHashMatches(recorded, content, incoming string) bool {
	if recorded == incoming {
		return true
	}
	if !bytes.HasSuffix([]byte(content), []byte("\n")) {
		return false
	}
	trimmed := sha256.Sum256(bytes.TrimSuffix([]byte(content), []byte("\n")))
	return recorded == hex.EncodeToString(trimmed[:])
}

type versionMetadata struct {
	id         string
	createdAt  string
	releasedAt string
}

func (s *sqliteRepository) versionMetadata(ctx context.Context, componentID string) (map[string]versionMetadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, version, created_at, released_at FROM component_versions WHERE component_id = ?`, componentID)
	if err != nil {
		return nil, fmt.Errorf("read existing component versions for %q: %w", componentID, err)
	}
	defer rows.Close()
	out := map[string]versionMetadata{}
	for rows.Next() {
		var version versionMetadata
		var label string
		if err := rows.Scan(&version.id, &label, &version.createdAt, &version.releasedAt); err != nil {
			return nil, fmt.Errorf("scan existing component version for %q: %w", componentID, err)
		}
		out[label] = version
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate existing component versions for %q: %w", componentID, err)
	}
	return out, nil
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
INSERT INTO components (id, library_id, slug, display_name, description, slot, category, asset_kind, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(library_id) DO UPDATE SET
  slug         = excluded.slug,
  display_name = excluded.display_name,
  description  = excluded.description,
  slot         = excluded.slot,
  category     = excluded.category,
  asset_kind   = excluded.asset_kind,
  source_path  = excluded.source_path,
  version      = excluded.version,
  latest_version = excluded.latest_version,
  draft_version = excluded.draft_version,
  manifest_path = excluded.manifest_path,
  tags         = excluded.tags,
  updated_at   = excluded.updated_at
`,
		id, in.LibraryID, slug, in.DisplayName, in.Description, in.Slot, in.Category, assetKindOrDefault(in.AssetKind), sourcePath, in.LatestVersion, in.LatestVersion, in.DraftVersion, in.ManifestPath,
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
	if err := s.loadDesignAffinities(ctx, &c); err != nil {
		return Component{}, err
	}
	if err := s.loadAssetProjection(ctx, []*Component{&c}); err != nil {
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
	if err := s.loadDesignAffinities(ctx, &c); err != nil {
		return Component{}, err
	}
	if err := s.loadAssetProjection(ctx, []*Component{&c}); err != nil {
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
		  lower(slot)         LIKE ? OR
		  lower(source_path)  LIKE ?
		)`)
		args = append(args, pat, pat, pat, pat, pat)
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
		clauses = append(clauses, `lower(category) = ?`)
		args = append(args, strings.ToLower(cat))
	}
	if q.AssetKind.Valid() {
		clauses = append(clauses, `asset_kind = ?`)
		args = append(args, string(q.AssetKind))
	}
	styleID := strings.TrimSpace(q.StyleID)
	affinity := strings.TrimSpace(q.Affinity)
	if styleID != "" || affinity != "" {
		subClauses := []string{`cda.component_id = components.id`}
		if styleID != "" {
			subClauses = append(subClauses, `lower(cda.style_id) = ?`)
			args = append(args, strings.ToLower(styleID))
		}
		if affinity != "" {
			subClauses = append(subClauses, `lower(cda.affinity) = ?`)
			args = append(args, strings.ToLower(affinity))
		}
		clauses = append(clauses, `EXISTS (
		  SELECT 1 FROM component_design_affinities cda
		  WHERE `+strings.Join(subClauses, " AND ")+`
		)`)
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
SELECT id, library_id, slug, display_name, description, slot, category, asset_kind, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
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
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close component rows: %w", err)
	}
	for i := range out {
		if err := s.loadDesignAffinities(ctx, &out[i]); err != nil {
			return nil, err
		}
	}
	assets := make([]*Component, 0, len(out))
	for i := range out {
		assets = append(assets, &out[i])
	}
	if err := s.loadAssetProjection(ctx, assets); err != nil {
		return nil, err
	}
	if err := s.loadCatalogProjectionBatch(ctx, assets); err != nil {
		return nil, err
	}
	return out, nil
}

// loadAssetProjection uses a fixed number of batched queries. Catalog list
// presentation never performs one count query per asset.
func (s *sqliteRepository) loadAssetProjection(ctx context.Context, assets []*Component) error {
	if len(assets) == 0 {
		return nil
	}
	ids := make([]any, 0, len(assets))
	byID := make(map[string]*Component, len(assets))
	for _, asset := range assets {
		ids = append(ids, asset.ID)
		byID[asset.ID] = asset
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	rows, err := s.db.QueryContext(ctx, `SELECT component_id, library_id, version FROM component_asset_dependencies WHERE component_id IN (`+placeholders+`) ORDER BY component_id, library_id, version`, ids...)
	if err != nil {
		return fmt.Errorf("load asset dependencies: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var dep AssetDependency
		if err := rows.Scan(&id, &dep.LibraryID, &dep.Version); err != nil {
			rows.Close()
			return fmt.Errorf("scan asset dependency: %w", err)
		}
		byID[id].Dependencies = append(byID[id].Dependencies, dep)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate asset dependencies: %w", err)
	}
	rows, err = s.db.QueryContext(ctx, `
SELECT c.id,
  (SELECT COUNT(*) FROM adoption_records a WHERE a.component_id = c.id),
  (
    SELECT COUNT(DISTINCT a.id)
    FROM adoption_records a
    LEFT JOIN adoption_files f ON f.adoption_id = a.id
    WHERE a.component_id = c.id OR f.source_asset_id = c.id
  ),
  (SELECT COUNT(*) FROM component_versions v WHERE v.component_id = c.id)
FROM components c WHERE c.id IN (`+placeholders+`)`, ids...)
	if err != nil {
		// Components repository tests (and isolated tooling) intentionally apply
		// only the components schema. Keep that seam usable while production's
		// all-domain schema supplies adoption_records for the direct count.
		if !strings.Contains(err.Error(), "no such table: adoption_records") {
			return fmt.Errorf("load asset metrics: %w", err)
		}
		rows, err = s.db.QueryContext(ctx, `SELECT component_id, COUNT(*) FROM component_versions WHERE component_id IN (`+placeholders+`) GROUP BY component_id`, ids...)
		if err != nil {
			return fmt.Errorf("load asset version metrics: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			var versionCount int
			if err := rows.Scan(&id, &versionCount); err != nil {
				return fmt.Errorf("scan asset version metric: %w", err)
			}
			byID[id].Metrics.VersionCount = versionCount
		}
		return rows.Err()
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var directAdoptions, effectiveAdoptions, versions int
		if err := rows.Scan(&id, &directAdoptions, &effectiveAdoptions, &versions); err != nil {
			return fmt.Errorf("scan asset metrics: %w", err)
		}
		byID[id].Metrics = AssetMetrics{DirectAdoptionCount: directAdoptions, EffectiveAdoptionCount: effectiveAdoptions, VersionCount: versions}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	rows, err = s.db.QueryContext(ctx, `SELECT component_id, adopted_version, COUNT(*) FROM adoption_records WHERE component_id IN (`+placeholders+`) GROUP BY component_id, adopted_version ORDER BY component_id, adopted_version`, ids...)
	if err != nil {
		return fmt.Errorf("load version adoption metrics: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, version string
		var count int
		if err := rows.Scan(&id, &version, &count); err != nil {
			return err
		}
		byID[id].Metrics.VersionAdoptions = append(byID[id].Metrics.VersionAdoptions, VersionAdoptionMetric{Version: version, CurrentCount: count, PeakCount: count})
	}
	return rows.Err()
}

func (s *sqliteRepository) replaceAssetDependencies(ctx context.Context, componentID string, deps []AssetDependency) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_asset_dependencies WHERE component_id = ?`, componentID); err != nil {
		return fmt.Errorf("clear asset dependencies for %q: %w", componentID, err)
	}
	for _, dep := range deps {
		if _, err := s.db.ExecContext(ctx, `INSERT INTO component_asset_dependencies (component_id, library_id, version) VALUES (?, ?, ?)`, componentID, dep.LibraryID, dep.Version); err != nil {
			return fmt.Errorf("insert asset dependency %s@%s for %q: %w", dep.LibraryID, dep.Version, componentID, err)
		}
	}
	return nil
}

func (s *sqliteRepository) replaceDesignAffinities(ctx context.Context, componentID string, affinities []ComponentDesignAffinity) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_design_affinities WHERE component_id = ?`, componentID); err != nil {
		return fmt.Errorf("clear design affinities for %q: %w", componentID, err)
	}
	for _, affinity := range affinities {
		styleID := strings.TrimSpace(affinity.StyleID)
		kind := strings.TrimSpace(string(affinity.Affinity))
		if styleID == "" || kind == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_design_affinities (component_id, style_id, affinity, reason)
VALUES (?, ?, ?, ?)
`, componentID, styleID, kind, strings.TrimSpace(affinity.Reason)); err != nil {
			return fmt.Errorf("insert design affinity %s=%s for %q: %w", styleID, kind, componentID, err)
		}
	}
	return nil
}

func (s *sqliteRepository) loadDesignAffinities(ctx context.Context, c *Component) error {
	rows, err := s.db.QueryContext(ctx, `
SELECT style_id, affinity, reason
FROM component_design_affinities
WHERE component_id = ?
ORDER BY style_id ASC
`, c.ID)
	if err != nil {
		return fmt.Errorf("load design affinities for %q: %w", c.ID, err)
	}
	defer rows.Close()
	var out []ComponentDesignAffinity
	for rows.Next() {
		var affinity ComponentDesignAffinity
		var kind string
		if err := rows.Scan(&affinity.StyleID, &kind, &affinity.Reason); err != nil {
			return fmt.Errorf("scan design affinity for %q: %w", c.ID, err)
		}
		affinity.Affinity = DesignAffinity(kind)
		out = append(out, affinity)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate design affinities for %q: %w", c.ID, err)
	}
	c.DesignStyles = out
	return nil
}

func (s *sqliteRepository) replaceHeaders(ctx context.Context, componentID string, headers map[string]string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_headers WHERE component_id = ?`, componentID); err != nil {
		return fmt.Errorf("clear headers for %q: %w", componentID, err)
	}
	for field, value := range headers {
		field = strings.TrimSpace(field)
		value = strings.TrimSpace(value)
		if field == "" || value == "" || isStructuredHeaderField(field) {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `INSERT INTO component_headers (component_id, field, value) VALUES (?, ?, ?)`, componentID, field, value); err != nil {
			return fmt.Errorf("insert header %s=%q for %q: %w", field, value, componentID, err)
		}
	}
	return nil
}

func isStructuredHeaderField(field string) bool {
	switch strings.ToLower(strings.TrimSpace(field)) {
	case "libraryid", "version", "deps", "category":
		return true
	default:
		return false
	}
}

// catalogHeaderFields are the header keys that carry the typed catalog
// projection. Only these are read on the list path.
var catalogHeaderFields = []string{"catalogId", "catalogExpects", "catalogSatisfies"}

// loadCatalogProjectionBatch populates the typed catalog fields for a list of
// components without attaching the raw header map.
//
// The distinction matters and is asserted by the repository tests: List
// deliberately omits Headers so list payloads stay lean, while Get carries the
// full map. That contract was correct, but CatalogID was only ever set as a
// side effect of loading the whole map — so every component returned by List
// had an empty CatalogID, which silently disabled the catalog projection
// (domain, rung, dependent counts) and rendered the entire catalog browser
// under "Other / Rung 0".
//
// Reading just these three fields honours both requirements: the projection is
// present, the arbitrary header map is not. Batched for the same reason
// loadAssetProjection is — list presentation must not issue one query per row.
func (s *sqliteRepository) loadCatalogProjectionBatch(ctx context.Context, assets []*Component) error {
	if len(assets) == 0 {
		return nil
	}
	args := make([]any, 0, len(assets)+len(catalogHeaderFields))
	byID := make(map[string]*Component, len(assets))
	for _, asset := range assets {
		args = append(args, asset.ID)
		byID[asset.ID] = asset
	}
	idPlaceholders := strings.TrimRight(strings.Repeat("?,", len(assets)), ",")
	for _, field := range catalogHeaderFields {
		args = append(args, field)
	}
	fieldPlaceholders := strings.TrimRight(strings.Repeat("?,", len(catalogHeaderFields)), ",")
	rows, err := s.db.QueryContext(ctx,
		`SELECT component_id, field, value FROM component_headers WHERE component_id IN (`+idPlaceholders+`) AND field IN (`+fieldPlaceholders+`)`, args...)
	if err != nil {
		return fmt.Errorf("load catalog projection batch: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var componentID, field, value string
		if err := rows.Scan(&componentID, &field, &value); err != nil {
			return fmt.Errorf("scan catalog projection row: %w", err)
		}
		asset, ok := byID[componentID]
		if !ok {
			continue
		}
		applyCatalogHeader(asset, field, value)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate catalog projection batch: %w", err)
	}
	return nil
}

// applyCatalogHeader projects one header onto its typed field. Shared by the
// single-row and batched loaders so the two paths cannot disagree about what a
// header means.
func applyCatalogHeader(c *Component, field, value string) {
	switch field {
	case "catalogId":
		c.CatalogID = strings.TrimSpace(value)
	case "catalogExpects":
		if raw := strings.TrimSpace(value); raw != "" {
			_ = json.Unmarshal([]byte(raw), &c.Expects)
		}
	case "catalogSatisfies":
		if raw := strings.TrimSpace(value); raw != "" {
			_ = json.Unmarshal([]byte(raw), &c.Satisfies)
		}
	}
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
	for field, value := range headers {
		applyCatalogHeader(c, field, value)
	}
	return nil
}

func (s *sqliteRepository) DeleteMissing(ctx context.Context, keep []string) (int, error) {
	var n int64
	if len(keep) == 0 {
		res, err := s.db.ExecContext(ctx, `DELETE FROM components`)
		if err != nil {
			return 0, fmt.Errorf("delete all components: %w", err)
		}
		n, _ = res.RowsAffected()
	} else {
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
		n, _ = res.RowsAffected()
	}
	// Cascade: the soft-FK model has no ON DELETE CASCADE, so removing a
	// registry row above would strand its child rows. Clear any child
	// rows now lacking a registry parent (both the ones just deleted and
	// any pre-existing cruft).
	if err := s.deleteOrphanChildRows(ctx); err != nil {
		return 0, err
	}
	return int(n), nil
}

// deleteOrphanChildRows removes every component-scoped row whose owning
// registry row is gone. Version-keyed child tables (files, parity) are
// cleared first so no row is stranded once the version rows go.
func (s *sqliteRepository) deleteOrphanChildRows(ctx context.Context) error {
	stmts := []string{
		`DELETE FROM component_version_files WHERE version_id IN (
			SELECT id FROM component_versions WHERE component_id NOT IN (SELECT id FROM components))`,
		`DELETE FROM component_version_parity_reports WHERE version_id IN (
			SELECT id FROM component_versions WHERE component_id NOT IN (SELECT id FROM components))`,
		`DELETE FROM component_versions WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_headers WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_design_affinities WHERE component_id NOT IN (SELECT id FROM components)`,
		`DELETE FROM component_asset_dependencies WHERE component_id NOT IN (SELECT id FROM components)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("sweep orphan child rows: %w", err)
		}
	}
	return nil
}

func (s *sqliteRepository) SweepOrphans(ctx context.Context) ([]OrphanVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT component_id, library_id, version, source_path
FROM component_versions
WHERE component_id NOT IN (SELECT id FROM components)
ORDER BY library_id, version`)
	if err != nil {
		return nil, fmt.Errorf("scan registry-orphaned versions: %w", err)
	}
	defer rows.Close()
	var orphans []OrphanVersion
	for rows.Next() {
		var o OrphanVersion
		if err := rows.Scan(&o.ComponentID, &o.LibraryID, &o.Version, &o.SourcePath); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan orphan version: %w", err)
		}
		orphans = append(orphans, o)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate orphan versions: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close orphan versions: %w", err)
	}
	if err := s.deleteOrphanChildRows(ctx); err != nil {
		return nil, err
	}
	return orphans, nil
}

func (s *sqliteRepository) ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at
FROM component_versions
WHERE component_id = ?
ORDER BY version DESC
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close component version rows: %w", err)
	}
	for i := range out {
		files, err := s.listVersionFiles(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].Files = files
		out[i].ExperienceContract = experienceContractFromFiles(files)
		out[i].RequiredTokens, err = s.listVersionRequiredTokens(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].RequiredTokenPatterns, err = s.listVersionRequiredTokenPatterns(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
		out[i].ParityReport, err = s.getVersionParity(ctx, out[i].ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *sqliteRepository) GetVersion(ctx context.Context, componentID, version string) (ComponentVersion, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at
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
	v.Files, err = s.listVersionFiles(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	v.ExperienceContract = experienceContractFromFiles(v.Files)
	v.RequiredTokens, err = s.listVersionRequiredTokens(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	v.RequiredTokenPatterns, err = s.listVersionRequiredTokenPatterns(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	v.ParityReport, err = s.getVersionParity(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	return v, nil
}

func experienceContractFromFiles(files []ComponentVersionFile) string {
	for _, file := range files {
		if file.Path == "experience-contract.json" {
			return file.Content
		}
	}
	return ""
}

func (s *sqliteRepository) getVersionParity(ctx context.Context, versionID string) (*IngestParityReport, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT report_json FROM component_version_parity_reports WHERE version_id = ?`, versionID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get component parity report: %w", err)
	}
	var report IngestParityReport
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		return nil, fmt.Errorf("decode component parity report: %w", err)
	}
	return &report, nil
}

func (s *sqliteRepository) listVersionFiles(ctx context.Context, versionID string) ([]ComponentVersionFile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, content, content_sha256, is_entry, slot FROM component_version_files WHERE version_id = ? ORDER BY is_entry DESC, path ASC`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list component version files: %w", err)
	}
	defer rows.Close()
	var files []ComponentVersionFile
	for rows.Next() {
		var f ComponentVersionFile
		if err := rows.Scan(&f.Path, &f.Content, &f.ContentSHA256, &f.IsEntry, &f.Slot); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, rows.Err()
}

func (s *sqliteRepository) listVersionRequiredTokens(ctx context.Context, versionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT property FROM component_version_required_tokens WHERE version_id = ? ORDER BY property`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list component version required tokens: %w", err)
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

func (s *sqliteRepository) listVersionRequiredTokenPatterns(ctx context.Context, versionID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT pattern FROM component_version_required_token_patterns WHERE version_id = ? ORDER BY pattern`, versionID)
	if err != nil {
		return nil, fmt.Errorf("list component version required token patterns: %w", err)
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

func (s *sqliteRepository) ListStories(ctx context.Context, q StoryQuery) ([]ComponentStory, error) {
	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	var clauses []string
	var args []any
	if q.ComponentID != "" {
		clauses = append(clauses, "component_id = ?")
		args = append(args, q.ComponentID)
	}
	if q.Version != "" {
		clauses = append(clauses, "version = ?")
		args = append(args, q.Version)
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`SELECT id, component_id, library_id, version, schema_version, kind, title, args_json, environment_json, stories_json, contract_json, source_path, indexed_at FROM component_stories %s ORDER BY library_id ASC, version DESC LIMIT ?`, where), args...)
	if err != nil {
		return nil, fmt.Errorf("list component stories: %w", err)
	}
	defer rows.Close()
	var out []ComponentStory
	for rows.Next() {
		story, err := scanComponentStory(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, story)
	}
	return out, rows.Err()
}

const (
	selectComponentByIDSQL = `
SELECT id, library_id, slug, display_name, description, slot, category, asset_kind, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
FROM components WHERE id = ?
`
	selectComponentByLibraryIDSQL = `
SELECT id, library_id, slug, display_name, description, slot, category, asset_kind, source_path, version, latest_version, draft_version, manifest_path, tags, indexed_at, updated_at
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
	if err := s.Scan(&c.ID, &c.LibraryID, &c.Slug, &c.DisplayName, &c.Description, &c.Slot, &c.Category, &c.AssetKind, &c.SourcePath, &c.Version, &c.LatestVersion, &c.DraftVersion, &c.ManifestPath, &tagsRaw, &indexedRaw, &updatedRaw); err != nil {
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

func assetKindOrDefault(kind AssetKind) AssetKind {
	if !kind.Valid() {
		return AssetKindComponent
	}
	return kind
}

func scanComponentVersion(s rowScanner) (ComponentVersion, error) {
	var v ComponentVersion
	var statusRaw, indexedRaw, createdRaw, releasedRaw string
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &statusRaw, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &indexedRaw, &createdRaw, &releasedRaw); err != nil {
		return ComponentVersion{}, err
	}
	v.Status = ComponentVersionStatus(statusRaw)
	indexed, err := time.Parse(timeFormat, indexedRaw)
	if err != nil {
		return ComponentVersion{}, fmt.Errorf("parse indexed_at %q: %w", indexedRaw, err)
	}
	v.IndexedAt = indexed
	if createdRaw != "" {
		created, err := time.Parse(timeFormat, createdRaw)
		if err != nil {
			return ComponentVersion{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
		}
		v.CreatedAt = created
	}
	if releasedRaw != "" {
		released, err := time.Parse(timeFormat, releasedRaw)
		if err != nil {
			return ComponentVersion{}, fmt.Errorf("parse released_at %q: %w", releasedRaw, err)
		}
		v.ReleasedAt = released
	}
	return v, nil
}

func scanComponentStory(s rowScanner) (ComponentStory, error) {
	var story ComponentStory
	var kind, indexedRaw string
	if err := s.Scan(&story.ID, &story.ComponentID, &story.LibraryID, &story.Version, &story.SchemaVersion, &kind, &story.Title, &story.ArgsJSON, &story.EnvironmentJSON, &story.StoriesJSON, &story.ContractJSON, &story.SourcePath, &indexedRaw); err != nil {
		return ComponentStory{}, err
	}
	story.Kind = StoryKind(kind)
	indexed, err := time.Parse(timeFormat, indexedRaw)
	if err != nil {
		return ComponentStory{}, fmt.Errorf("parse indexed_at %q: %w", indexedRaw, err)
	}
	story.IndexedAt = indexed
	return story, nil
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
