package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
	"react-component-library/internal/availability"
)

// CatalogMetadata preserves authored catalog fields in the unified search projection.
type CatalogMetadata struct {
	CatalogID            string   `json:"catalogId"`
	Name                 string   `json:"name"`
	Kind                 string   `json:"kind"`
	Slot                 string   `json:"slot,omitempty"`
	Domain               string   `json:"domain,omitempty"`
	Surface              string   `json:"surface,omitempty"`
	Description          string   `json:"description,omitempty"`
	DeclarationPath      string   `json:"declarationPath,omitempty"`
	RequiredCapabilities []string `json:"requiredCapabilities,omitempty"`
	Regions              []string `json:"regions,omitempty"`
	Targets              []string `json:"targets,omitempty"`
	Kits                 []string `json:"kits,omitempty"`
	Requires             []string `json:"requires,omitempty"`
}

// CatalogResult combines one authored declaration with current implementation availability.
type CatalogResult struct {
	CatalogMetadata
	Availability availability.Result
	Implemented  bool
	Score        float64
}

// CatalogAvailabilityFactory creates a request-scoped view of current published implementations.
type CatalogAvailabilityFactory func(context.Context) (*availability.Snapshot, error)

// CatalogSearch is the search-owned typed seam used by sketch import and proposal.
type CatalogSearch interface {
	SearchCatalog(context.Context, string, int, string, string, string) ([]CatalogResult, error)
	LookupCatalogID(context.Context, string) (CatalogResult, bool, error)
}

// WithCatalogAvailability attaches the same canonical availability resolver used by the catalog owner.
func (s *Service) WithCatalogAvailability(factory CatalogAvailabilityFactory) *Service {
	if s != nil {
		s.catalogAvailability = factory
	}
	return s
}

// LookupCatalogID performs exact declared-ID lookup in the unified projection. It never ranks or tokenizes IDs.
func (s *Service) LookupCatalogID(ctx context.Context, id string) (CatalogResult, bool, error) {
	if s == nil || s.db == nil {
		return CatalogResult{}, false, ErrNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return CatalogResult{}, false, err
	}
	var raw string
	var storedAvailability string
	err := s.db.QueryRowContext(ctx, `SELECT catalog_json, availability FROM search_docs WHERE subject_id = ? AND kind IN (?, ?) ORDER BY kind LIMIT 1`, id, KindText(searchv1.SearchKind_SEARCH_KIND_ASSET), KindText(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE)).Scan(&raw, &storedAvailability)
	if errors.Is(err, sql.ErrNoRows) {
		return CatalogResult{}, false, nil
	}
	if err != nil {
		return CatalogResult{}, false, fmt.Errorf("lookup catalog declaration %q: %w", id, err)
	}
	metadata, err := decodeCatalogMetadata(raw)
	if err != nil {
		return CatalogResult{}, false, err
	}
	if metadata.CatalogID == "" {
		return CatalogResult{}, false, nil
	}
	result := CatalogResult{CatalogMetadata: metadata, Availability: storedCatalogAvailability(metadata.CatalogID, storedAvailability)}
	result.Implemented = result.Availability.IsBuilt()
	if err := ctx.Err(); err != nil {
		return CatalogResult{}, false, err
	}
	return result, true, nil
}

// SearchCatalog ranks authored declarations through the unified SearchService projection.
func (s *Service) SearchCatalog(ctx context.Context, query string, limit int, kind, domain, accepts string) ([]CatalogResult, error) {
	if s == nil || s.db == nil {
		return nil, ErrNotConfigured
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > maxSearchLimit {
		limit = maxSearchLimit
	}
	plan := planQuery(query)
	if plan.empty() {
		return []CatalogResult{}, nil
	}
	kinds := map[string]bool{KindText(searchv1.SearchKind_SEARCH_KIND_ASSET): true, KindText(searchv1.SearchKind_SEARCH_KIND_PAGE_TYPE): true}
	candidates, err := s.matchLexical(ctx, plan)
	if err != nil {
		return nil, err
	}
	candidates = filterKinds(candidates, kinds)
	sortRanked(candidates)
	if len(candidates) == 0 {
		return []CatalogResult{}, nil
	}
	ids := make([]string, 0, len(candidates))
	for _, result := range candidates {
		ids = append(ids, result.doc.ID)
	}
	metadataByID, err := s.catalogMetadataForDocuments(ctx, ids)
	if err != nil {
		return nil, err
	}
	results := make([]CatalogResult, 0, limit)
	for _, ranked := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		metadata, ok := metadataByID[ranked.doc.ID]
		if !ok {
			continue
		}
		if kind != "" && metadata.Kind != kind || domain != "" && metadata.Domain != domain || accepts != "" && metadata.Domain != accepts && metadata.Kind != accepts {
			continue
		}
		availabilityResult := storedCatalogAvailability(metadata.CatalogID, AvailabilityText(ranked.doc.Availability))
		results = append(results, CatalogResult{CatalogMetadata: metadata, Availability: availabilityResult, Implemented: availabilityResult.IsBuilt(), Score: ranked.score})
		if len(results) == limit {
			break
		}
	}
	if s.catalogAvailability != nil && len(results) > 0 {
		snapshot, err := s.catalogAvailability(ctx)
		if err != nil {
			return nil, fmt.Errorf("load current catalog availability: %w", err)
		}
		for i := range results {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			results[i].Availability = snapshot.Resolve(ctx, results[i].CatalogID, snapshot.Latest(results[i].CatalogID))
			results[i].Implemented = results[i].Availability.IsBuilt()
		}
	}
	return results, nil
}

func (s *Service) catalogMetadataForDocuments(ctx context.Context, ids []string) (map[string]CatalogMetadata, error) {
	metadata := make(map[string]CatalogMetadata, len(ids))
	const batchSize = 400 // remain below SQLite's bind-variable limit on older builds
	for start := 0; start < len(ids); start += batchSize {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[start:end]
		placeholders := strings.TrimRight(strings.Repeat("?,", len(batch)), ",")
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows, err := s.db.QueryContext(ctx, `SELECT id, catalog_json FROM search_docs WHERE id IN (`+placeholders+`)`, args...)
		if err != nil {
			return nil, fmt.Errorf("read catalog metadata from unified projection: %w", err)
		}
		for rows.Next() {
			var id, raw string
			if err := rows.Scan(&id, &raw); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan catalog metadata: %w", err)
			}
			item, err := decodeCatalogMetadata(raw)
			if err != nil {
				rows.Close()
				return nil, err
			}
			if item.CatalogID != "" {
				metadata[id] = item
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read catalog metadata: %w", err)
		}
		if err := rows.Close(); err != nil {
			return nil, fmt.Errorf("close catalog metadata: %w", err)
		}
	}
	return metadata, nil
}

func decodeCatalogMetadata(raw string) (CatalogMetadata, error) {
	var metadata CatalogMetadata
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return CatalogMetadata{}, fmt.Errorf("decode catalog metadata: %w", err)
	}
	if metadata.CatalogID == "" {
		return CatalogMetadata{}, nil
	}
	if metadata.Kind == "" {
		return CatalogMetadata{}, errors.New("catalog projection row has incomplete authored metadata")
	}
	return metadata, nil
}

func storedCatalogAvailability(id, value string) availability.Result {
	result := availability.Result{CatalogID: id, State: availability.Unavailable, ReasonCode: "availability_unknown"}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "available", "searchavailability_search_availability_available":
		result.State = availability.Built
	case "planned", "searchavailability_search_availability_planned":
		result.State = availability.Declared
	case "draft", "searchavailability_search_availability_draft":
		result.State = availability.Draft
	default:
		result.State = availability.Unavailable
	}
	return result
}
