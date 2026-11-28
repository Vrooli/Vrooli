package components

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const maxAffinityIDsPerQuery = 400

// loadDesignAffinitiesBatch keeps the catalog list path bounded as the library
// grows. A per-component query made Start/Library scale linearly in round trips
// and could exceed the UI request deadline on a populated registry.
func (s *sqliteRepository) loadDesignAffinitiesBatch(ctx context.Context, components []*Component) error {
	byID := make(map[string]*Component, len(components))
	ids := make([]string, 0, len(components))
	for _, component := range components {
		if component == nil || component.ID == "" {
			continue
		}
		byID[component.ID] = component
		ids = append(ids, component.ID)
	}
	for start := 0; start < len(ids); start += maxAffinityIDsPerQuery {
		end := start + maxAffinityIDsPerQuery
		if end > len(ids) {
			end = len(ids)
		}
		chunk := ids[start:end]
		placeholders := make([]string, len(chunk))
		args := make([]any, len(chunk))
		for i, id := range chunk {
			placeholders[i] = "?"
			args[i] = id
		}
		query := `SELECT component_id, style_id, affinity, reason
FROM component_design_affinities
WHERE component_id IN (` + strings.Join(placeholders, ",") + `)
ORDER BY component_id ASC, style_id ASC`
		rows, err := s.db.QueryContext(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("load design affinities: %w", err)
		}
		for rows.Next() {
			var componentID, styleID, kind, reason string
			if err := rows.Scan(&componentID, &styleID, &kind, &reason); err != nil {
				_ = rows.Close()
				return fmt.Errorf("scan design affinity for %q: %w", componentID, err)
			}
			component, ok := byID[componentID]
			if !ok {
				_ = rows.Close()
				return fmt.Errorf("load design affinities: unexpected component id %q", componentID)
			}
			component.DesignStyles = append(component.DesignStyles, ComponentDesignAffinity{
				StyleID: styleID, Affinity: DesignAffinity(kind), Reason: reason,
			})
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			return fmt.Errorf("iterate design affinities: %w", err)
		}
		if err := rows.Close(); err != nil {
			return fmt.Errorf("close design affinities: %w", err)
		}
	}
	return nil
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
	compatRows, err := s.db.QueryContext(ctx, `SELECT v.component_id, k.verdict, k.compatible_kit_ids_json, k.unsatisfied_properties_json FROM component_versions v JOIN components c ON c.id = v.component_id AND c.latest_version = v.version JOIN component_version_kit_compatibility k ON k.version_id = v.id WHERE v.component_id IN (`+placeholders+`)`, ids...)
	if err != nil {
		return fmt.Errorf("load latest kit compatibility: %w", err)
	}
	defer compatRows.Close()
	for compatRows.Next() {
		var id, verdict, compatible, unsatisfied string
		if err := compatRows.Scan(&id, &verdict, &compatible, &unsatisfied); err != nil {
			return fmt.Errorf("scan latest kit compatibility: %w", err)
		}
		compatibility := ComponentKitCompatibility{Verdict: KitCompatibilityVerdict(verdict)}
		_ = json.Unmarshal([]byte(compatible), &compatibility.CompatibleKitIDs)
		_ = json.Unmarshal([]byte(unsatisfied), &compatibility.UnsatisfiedProperties)
		byID[id].KitCompatibility = compatibility
	}
	if err := compatRows.Err(); err != nil {
		return fmt.Errorf("iterate latest kit compatibility: %w", err)
	}
	if err := compatRows.Close(); err != nil {
		return fmt.Errorf("close latest kit compatibility rows: %w", err)
	}
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
