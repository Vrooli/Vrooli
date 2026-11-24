package components

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"react-component-library/internal/sqliterows"
)

func (s *sqliteRepository) Get(ctx context.Context, id string) (Component, error) {
	return s.get(ctx, selectComponentByIDSQL, id, "get component")
}

func (s *sqliteRepository) GetByLibraryID(ctx context.Context, libraryID string) (Component, error) {
	return s.get(ctx, selectComponentByLibraryIDSQL, libraryID, "get component by libraryId")
}

func (s *sqliteRepository) get(ctx context.Context, query, id, operation string) (Component, error) {
	row := s.db.QueryRowContext(ctx, query, id)
	c, err := scanComponent(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Component{}, ErrComponentNotFound{IDOrLibraryID: id}
	}
	if err != nil {
		return Component{}, fmt.Errorf("%s %q: %w", operation, id, err)
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

	out, err := sqliterows.Collect(ctx, s.db, query, args, true, func(row sqliterows.Scanner) (Component, error) {
		component, err := scanComponent(row)
		if err != nil {
			return Component{}, fmt.Errorf("scan component: %w", err)
		}
		return component, nil
	}, sqliterows.ErrorContext{Query: "list components", Iterate: "iterate components", Close: "close component rows"})
	if err != nil {
		return nil, err
	}
	assets := make([]*Component, 0, len(out))
	for i := range out {
		assets = append(assets, &out[i])
	}
	if err := s.loadDesignAffinitiesBatch(ctx, assets); err != nil {
		return nil, err
	}
	if err := s.loadAssetProjection(ctx, assets); err != nil {
		return nil, err
	}
	if err := s.loadCatalogProjectionBatch(ctx, assets); err != nil {
		return nil, err
	}
	return out, nil
}
