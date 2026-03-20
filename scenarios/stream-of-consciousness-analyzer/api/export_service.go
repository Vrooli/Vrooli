// DOC: docs/reference/api-endpoints.md#export
// DOC: docs/internal/SEAMS.md#cross-scenario-integration-points
package main

import (
	"database/sql"

	"github.com/lib/pq"
)

// ExportData represents the full graph export of a scheme
type ExportData struct {
	Scheme       Scheme        `json:"scheme"`
	Information  []Information `json:"information"`
	Thoughts     []Thought     `json:"thoughts"`
	Edges        []ThoughtEdge `json:"edges"`
	ExportFormat string        `json:"export_format"`
}

// ExportService handles scheme export operations
type ExportService struct {
	db *sql.DB
}

// NewExportService creates a new ExportService
func NewExportService(db *sql.DB) *ExportService {
	return &ExportService{db: db}
}

// ExportScheme exports the full graph data for a scheme
func (s *ExportService) ExportScheme(schemeID string) (*ExportData, error) {
	// Get scheme
	scheme, err := scanScheme(s.db.QueryRow(
		`SELECT id, name, created_at, updated_at FROM schemes WHERE id = $1`, schemeID,
	))
	if err != nil {
		return nil, err
	}

	// Get information items
	infoRows, err := s.db.Query(
		`SELECT id, scheme_id, type, content, canvas_x, canvas_y, created_at, updated_at
		 FROM information WHERE scheme_id = $1`, schemeID,
	)
	if err != nil {
		return nil, err
	}
	info, err := collectRows(infoRows, scanInformation)
	if err != nil {
		return nil, err
	}

	// Get thoughts
	thoughtRows, err := s.db.Query(
		`SELECT id, scheme_id, title, body, canvas_x, canvas_y, created_at, updated_at
		 FROM thoughts WHERE scheme_id = $1`, schemeID,
	)
	if err != nil {
		return nil, err
	}
	thoughts, err := collectRows(thoughtRows, scanThought)
	if err != nil {
		return nil, err
	}
	thoughtIDs := make([]string, len(thoughts))
	for i, t := range thoughts {
		thoughtIDs[i] = t.ID
	}

	// Collect edges for all thoughts in this scheme.
	// An edge can appear in multiple per-thought queries (once for its source,
	// once for its target), so we deduplicate by edge ID.
	edges, err := s.collectSchemeEdges(thoughtIDs)
	if err != nil {
		return nil, err
	}

	return &ExportData{
		Scheme:       scheme,
		Information:  info,
		Thoughts:     thoughts,
		Edges:        edges,
		ExportFormat: ExportFormatVersion,
	}, nil
}

// collectSchemeEdges returns all edges where at least one endpoint is in thoughtIDs.
// Uses a single query with ANY($1) instead of N per-thought queries.
func (s *ExportService) collectSchemeEdges(thoughtIDs []string) ([]ThoughtEdge, error) {
	if len(thoughtIDs) == 0 {
		return []ThoughtEdge{}, nil
	}

	rows, err := s.db.Query(
		`SELECT DISTINCT id, source_id, target_id, label, created_at
		 FROM thought_edges
		 WHERE source_id = ANY($1) OR target_id = ANY($1)
		 ORDER BY created_at ASC`,
		pq.Array(thoughtIDs),
	)
	if err != nil {
		return nil, err
	}
	return collectRows(rows, scanEdge)
}
