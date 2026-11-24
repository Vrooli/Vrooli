package components

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"react-component-library/internal/sqliterows"
)

func (s *sqliteRepository) ListVersions(ctx context.Context, componentID string, limit int) ([]ComponentVersion, error) {
	if limit <= 0 {
		limit = 100
	}
	out, err := sqliterows.Collect(ctx, s.db, `
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence
FROM component_versions
WHERE component_id = ?
ORDER BY version DESC
LIMIT ?
	`, []any{componentID, limit}, true, func(row sqliterows.Scanner) (ComponentVersion, error) {
		return scanComponentVersion(row)
	}, sqliterows.ErrorContext{Query: "list component versions", Close: "close component version rows"})
	if err != nil {
		return nil, err
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
		out[i].KitCompatibility, err = s.getVersionKitCompatibility(ctx, out[i].ID)
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
SELECT id, component_id, library_id, version, status, source_path, content, content_sha256, changelog_md, indexed_at, created_at, released_at, presence
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
	v.Dependencies, v.DependencyLockPresent, err = dependenciesFromFiles(v.Files)
	if err != nil {
		return ComponentVersion{}, fmt.Errorf("decode dependency lock for %s@%s: %w", v.LibraryID, v.Version, err)
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
	v.KitCompatibility, err = s.getVersionKitCompatibility(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	v.ParityReport, err = s.getVersionParity(ctx, v.ID)
	if err != nil {
		return ComponentVersion{}, err
	}
	return v, nil
}

func dependenciesFromFiles(files []ComponentVersionFile) ([]AssetDependency, bool, error) {
	for _, file := range files {
		if file.Path != "dependencies.json" {
			continue
		}
		var lock VersionLock
		if err := json.Unmarshal([]byte(file.Content), &lock); err != nil {
			return nil, false, err
		}
		dependencies := make([]AssetDependency, 0, len(lock.Dependencies))
		for _, dependency := range lock.Dependencies {
			dependencies = append(dependencies, AssetDependency{LibraryID: strings.TrimSpace(dependency.LibraryID), Version: dependency.Pin()})
		}
		return dependencies, true, nil
	}
	return nil, false, nil
}

func (s *sqliteRepository) SetVersionPresence(ctx context.Context, componentID, version, presence string) error {
	if presence != "materialized" && presence != "evicted" {
		return fmt.Errorf("invalid version presence %q", presence)
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE component_versions SET presence=? WHERE component_id=? AND version=?`, presence, componentID, version); err != nil {
		return fmt.Errorf("set version presence: %w", err)
	}
	return nil
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
	files, err := sqliterows.Collect(ctx, s.db, `SELECT path, content, content_sha256, is_entry, slot FROM component_version_files WHERE version_id = ? ORDER BY is_entry DESC, path ASC`, []any{versionID}, false, func(row sqliterows.Scanner) (ComponentVersionFile, error) {
		var file ComponentVersionFile
		return file, row.Scan(&file.Path, &file.Content, &file.ContentSHA256, &file.IsEntry, &file.Slot)
	}, sqliterows.ErrorContext{Query: "list component version files"})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *sqliteRepository) listVersionRequiredTokens(ctx context.Context, versionID string) ([]string, error) {
	properties, err := sqliterows.Collect(ctx, s.db, `SELECT property FROM component_version_required_tokens WHERE version_id = ? ORDER BY property`, []any{versionID}, false, func(row sqliterows.Scanner) (string, error) {
		var property string
		return property, row.Scan(&property)
	}, sqliterows.ErrorContext{Query: "list component version required tokens"})
	if err != nil {
		return nil, err
	}
	return properties, nil
}

func (s *sqliteRepository) listVersionRequiredTokenPatterns(ctx context.Context, versionID string) ([]string, error) {
	patterns, err := sqliterows.Collect(ctx, s.db, `SELECT pattern FROM component_version_required_token_patterns WHERE version_id = ? ORDER BY pattern`, []any{versionID}, false, func(row sqliterows.Scanner) (string, error) {
		var pattern string
		return pattern, row.Scan(&pattern)
	}, sqliterows.ErrorContext{Query: "list component version required token patterns"})
	if err != nil {
		return nil, err
	}
	return patterns, nil
}

func (s *sqliteRepository) getVersionKitCompatibility(ctx context.Context, versionID string) (ComponentKitCompatibility, error) {
	var verdict, compatible, unsatisfied string
	err := s.db.QueryRowContext(ctx, `SELECT verdict, compatible_kit_ids_json, unsatisfied_properties_json FROM component_version_kit_compatibility WHERE version_id = ?`, versionID).Scan(&verdict, &compatible, &unsatisfied)
	if errors.Is(err, sql.ErrNoRows) {
		return ComponentKitCompatibility{}, nil
	}
	if err != nil {
		return ComponentKitCompatibility{}, fmt.Errorf("get component version kit compatibility: %w", err)
	}
	result := ComponentKitCompatibility{Verdict: KitCompatibilityVerdict(verdict)}
	if err := json.Unmarshal([]byte(compatible), &result.CompatibleKitIDs); err != nil {
		return ComponentKitCompatibility{}, err
	}
	if err := json.Unmarshal([]byte(unsatisfied), &result.UnsatisfiedProperties); err != nil {
		return ComponentKitCompatibility{}, err
	}
	return result, nil
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
	query := fmt.Sprintf(`SELECT id, component_id, library_id, version, schema_version, kind, title, args_json, environment_json, stories_json, contract_json, source_path, indexed_at FROM component_stories %s ORDER BY library_id ASC, version DESC LIMIT ?`, where)
	out, err := sqliterows.Collect(ctx, s.db, query, args, false, func(row sqliterows.Scanner) (ComponentStory, error) {
		return scanComponentStory(row)
	}, sqliterows.ErrorContext{Query: "list component stories"})
	if err != nil {
		return nil, err
	}
	return out, nil
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

type rowScanner = sqliterows.Scanner

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
	if err := s.Scan(&v.ID, &v.ComponentID, &v.LibraryID, &v.Version, &statusRaw, &v.SourcePath, &v.Content, &v.ContentSHA256, &v.ChangelogMD, &indexedRaw, &createdRaw, &releasedRaw, &v.Presence); err != nil {
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
