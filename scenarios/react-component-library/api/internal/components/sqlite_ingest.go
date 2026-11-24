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
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"react-component-library/internal/sqliterows"
)

const timeFormat = time.RFC3339Nano

// tagSep is the in-row tag separator. Comma chosen for readability in
// SQL queries; tags themselves are forbidden from containing commas by
// the indexer's header parser.
const tagSep = ","

func (s *sqliteRepository) Upsert(ctx context.Context, in UpsertInput) (Component, error) {
	manifest := ComponentManifest{
		CatalogID:     in.CatalogID,
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
	current := firstNonEmpty(in.Manifest.LatestVersion, in.Manifest.DraftVersion)
	for _, v := range in.Versions {
		if v.Version == current {
			sourcePath = v.SourcePath
			break
		}
	}
	c, err := s.upsertComponent(ctx, in.Manifest, sourcePath)
	if err != nil {
		return Component{}, err
	}
	existingVersions, err := s.versionMetadata(ctx, c.ID)
	if err != nil {
		return Component{}, err
	}
	versionPlaceholders := strings.TrimRight(strings.Repeat("?,", len(in.Versions)), ",")
	refreshPredicate := "presence <> 'evicted'"
	refreshArgs := []any{c.ID}
	if versionPlaceholders != "" {
		refreshPredicate += " OR version IN (" + versionPlaceholders + ")"
		for _, v := range in.Versions {
			refreshArgs = append(refreshArgs, v.Version)
		}
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_files WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ? AND (`+refreshPredicate+`))`, refreshArgs...); err != nil {
		return Component{}, fmt.Errorf("clear component version files for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_required_tokens WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ? AND (`+refreshPredicate+`))`, refreshArgs...); err != nil {
		return Component{}, fmt.Errorf("clear component version required tokens for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_required_token_patterns WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ? AND (`+refreshPredicate+`))`, refreshArgs...); err != nil {
		return Component{}, fmt.Errorf("clear component version required token patterns for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_kit_compatibility WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ? AND (`+refreshPredicate+`))`, refreshArgs...); err != nil {
		return Component{}, fmt.Errorf("clear component version kit compatibility for %q: %w", c.ID, err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM component_version_parity_reports WHERE version_id IN (SELECT id FROM component_versions WHERE component_id = ? AND (`+refreshPredicate+`))`, refreshArgs...); err != nil {
		return Component{}, fmt.Errorf("clear component parity reports for %q: %w", c.ID, err)
	}
	// Evicted versions are intentionally absent from the working tree. Their
	// story projection is still durable registry data, so a reindex must not
	// treat the absence of story.json on disk as deletion. Materialized
	// versions remain rebuildable from the working tree and retain the old
	// replacement behavior when their story disappears.
	if _, err := s.db.ExecContext(ctx, `
DELETE FROM component_stories
WHERE component_id = ?
  AND version NOT IN (
	    SELECT version FROM component_versions
	    WHERE component_id = ? AND presence = 'evicted'
  )`, c.ID, c.ID); err != nil {
		return Component{}, fmt.Errorf("clear component stories for %q: %w", c.ID, err)
	}
	evictedStories, err := s.restoreEvictedStories(ctx, c.ID, c.LibraryID)
	if err != nil {
		return Component{}, err
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
			  (id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(component_id, version) DO UPDATE SET
  library_id = excluded.library_id,
  status = excluded.status,
  source_path = excluded.source_path,
  content = excluded.content,
  content_sha256 = excluded.content_sha256,
  changelog_md = excluded.changelog_md,
  indexed_at = excluded.indexed_at,
  -- A re-index observes the authored working tree, but it must not promote a
  -- cold release back into the hot tier. Presence is lifecycle state owned by
  -- the reconciler; only an explicit materialization operation may change an
  -- existing evicted row.
  presence = CASE
    WHEN component_versions.presence = 'evicted' THEN 'evicted'
    ELSE excluded.presence
  END,
  released_at = CASE
    WHEN component_versions.released_at <> '' THEN component_versions.released_at
    ELSE excluded.released_at
  END
`, v.ID, v.ComponentID, v.LibraryID, v.Version, string(v.Status), v.SourcePath, v.Content, v.ContentSHA256,
			v.ChangelogMD, v.IndexedAt.UTC().Format(timeFormat), createdAt, releasedAt, firstNonEmpty(v.Presence, "materialized")); err != nil {
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
		if v.KitCompatibility.Verdict != "" {
			compatible, _ := json.Marshal(v.KitCompatibility.CompatibleKitIDs)
			unsatisfied, _ := json.Marshal(v.KitCompatibility.UnsatisfiedProperties)
			if _, err := s.db.ExecContext(ctx, `INSERT INTO component_version_kit_compatibility (version_id, verdict, compatible_kit_ids_json, unsatisfied_properties_json) VALUES (?, ?, ?, ?)`, v.ID, string(v.KitCompatibility.Verdict), string(compatible), string(unsatisfied)); err != nil {
				return Component{}, fmt.Errorf("upsert component kit compatibility %s@%s: %w", c.LibraryID, v.Version, err)
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
		deleteStale := `DELETE FROM component_versions WHERE component_id = ? AND presence <> 'evicted' AND version NOT IN (` + placeholders + `)`
		if _, err := s.db.ExecContext(ctx, deleteStale, args...); err != nil {
			return Component{}, fmt.Errorf("reconcile stale component versions for %q: %w", c.ID, err)
		}
	}
	for _, story := range append(in.Stories, evictedStories...) {
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

// RestoreEvictedStories rebuilds only projections that are absent. It runs
// independently of manifest validation so an immutable-source mismatch on a
// different component cannot prevent recovery of durable cold-tier metadata.
func (s *sqliteRepository) RestoreEvictedStories(ctx context.Context) (int, error) {
	type componentRef struct{ id, libraryID string }
	components, err := sqliterows.Collect(ctx, s.db, `SELECT id, library_id FROM components ORDER BY library_id`, nil, true, func(row sqliterows.Scanner) (componentRef, error) {
		var ref componentRef
		if err := row.Scan(&ref.id, &ref.libraryID); err != nil {
			return componentRef{}, fmt.Errorf("scan component for evicted story restoration: %w", err)
		}
		return ref, nil
	}, sqliterows.ErrorContext{
		Query: "list components for evicted story restoration", Iterate: "iterate components for evicted story restoration", Close: "close components for evicted story restoration",
	})
	if err != nil {
		return 0, err
	}

	restored := 0
	for _, ref := range components {
		stories, err := s.restoreEvictedStories(ctx, ref.id, ref.libraryID)
		if err != nil {
			return restored, err
		}
		for _, story := range stories {
			if _, err := s.db.ExecContext(ctx, `
INSERT INTO component_stories (id, component_id, library_id, version, schema_version, kind, title, args_json, environment_json, stories_json, contract_json, source_path, indexed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(component_id, version) DO NOTHING
`, story.ID, ref.id, ref.libraryID, story.Version, story.SchemaVersion, string(story.Kind), story.Title, story.ArgsJSON, story.EnvironmentJSON, story.StoriesJSON, story.ContractJSON, story.SourcePath, s.clock.Now().UTC().Format(timeFormat)); err != nil {
				return restored, fmt.Errorf("restore component story %s@%s: %w", ref.libraryID, story.Version, err)
			}
			restored++
		}
	}
	return restored, nil
}

// restoreEvictedStories repairs a missing typed story projection from the
// durable story.json file mirror. This is deliberately scoped to evicted
// versions: materialized versions are parsed by the filesystem indexer, while
// evicted versions have no source path for that parser to read.
func (s *sqliteRepository) restoreEvictedStories(ctx context.Context, componentID, libraryID string) ([]ComponentStory, error) {
	type mirror struct {
		version, sourcePath, content string
	}
	mirrors, err := sqliterows.Collect(ctx, s.db, `
SELECT v.version, v.source_path, f.content
FROM component_versions v
JOIN component_version_files f ON f.version_id = v.id AND f.path = 'story.json'
LEFT JOIN component_stories story ON story.component_id = v.component_id AND story.version = v.version
WHERE v.component_id = ? AND v.presence = 'evicted' AND story.id IS NULL
ORDER BY v.version`, []any{componentID}, true, func(row sqliterows.Scanner) (mirror, error) {
		var item mirror
		if err := row.Scan(&item.version, &item.sourcePath, &item.content); err != nil {
			return mirror{}, fmt.Errorf("scan evicted story mirror for %q: %w", libraryID, err)
		}
		return item, nil
	}, sqliterows.ErrorContext{
		Query: fmt.Sprintf("find missing evicted story projections for %q", libraryID), Iterate: fmt.Sprintf("iterate evicted story mirrors for %q", libraryID), Close: fmt.Sprintf("close evicted story mirrors for %q", libraryID),
	})
	if err != nil {
		return nil, err
	}

	stories := make([]ComponentStory, 0, len(mirrors))
	for _, item := range mirrors {
		contract, diagnostics := ParseStoryContract([]byte(item.content))
		if contract == nil || len(storyErrors(diagnostics)) > 0 {
			return nil, fmt.Errorf("restore evicted story %s@%s: durable story.json is invalid", libraryID, item.version)
		}
		args, err := json.Marshal(contract.Args)
		if err != nil {
			return nil, fmt.Errorf("encode evicted story args %s@%s: %w", libraryID, item.version, err)
		}
		environment, err := json.Marshal(contract.Environment)
		if err != nil {
			return nil, fmt.Errorf("encode evicted story environment %s@%s: %w", libraryID, item.version, err)
		}
		storyDefinitions, err := json.Marshal(contract.Stories)
		if err != nil {
			return nil, fmt.Errorf("encode evicted story definitions %s@%s: %w", libraryID, item.version, err)
		}
		normalized, err := json.Marshal(contract)
		if err != nil {
			return nil, fmt.Errorf("encode evicted story contract %s@%s: %w", libraryID, item.version, err)
		}
		stories = append(stories, ComponentStory{
			ID:              uuid.NewString(),
			LibraryID:       libraryID,
			Version:         item.version,
			SchemaVersion:   contract.SchemaVersion,
			Kind:            contract.Kind,
			Title:           contract.Title,
			ArgsJSON:        string(args),
			EnvironmentJSON: string(environment),
			StoriesJSON:     string(storyDefinitions),
			ContractJSON:    string(normalized),
			SourcePath:      path.Join(path.Dir(item.sourcePath), "story.json"),
		})
	}
	return stories, nil
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
			// The index disagrees with the source. Before refusing, ask the
			// committed hash registry, which is the tracked record a reviewer
			// reads and the one a migration updates beside the bytes it
			// rewrote. When it vouches for these bytes, the stale side is this
			// projection and the upsert may refresh it.
			if attestsRelease(in.ReleaseAttestations, version.SourcePath, version.ContentSHA256) {
				continue
			}
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
	presence   string
}

func (s *sqliteRepository) versionMetadata(ctx context.Context, componentID string) (map[string]versionMetadata, error) {
	type metadataRow struct {
		label string
		versionMetadata
	}
	versions, err := sqliterows.Collect(ctx, s.db, `SELECT id, version, created_at, released_at, presence FROM component_versions WHERE component_id = ?`, []any{componentID}, false, func(scanner sqliterows.Scanner) (metadataRow, error) {
		var item metadataRow
		if err := scanner.Scan(&item.id, &item.label, &item.createdAt, &item.releasedAt, &item.presence); err != nil {
			return metadataRow{}, fmt.Errorf("scan existing component version for %q: %w", componentID, err)
		}
		return item, nil
	}, sqliterows.ErrorContext{
		Query: fmt.Sprintf("read existing component versions for %q", componentID), Iterate: fmt.Sprintf("iterate existing component versions for %q", componentID),
	})
	if err != nil {
		return nil, err
	}
	out := make(map[string]versionMetadata, len(versions))
	for _, item := range versions {
		out[item.label] = item.versionMetadata
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
