package main

// DOC: docs/reference/api-endpoints.md#search
// DOC: docs/concepts/ARCHITECTURE.md#search-flow
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

const (
	defaultSearchLimit     = 10
	maxSearchLimit         = 100
	defaultSearchThreshold = 0.3
)

// SearchRequest defines the input schema for semantic search.
// Note: the legacy records-era filters (collection, namespaces, visibility,
// tags, ingested_after, ingested_before) were removed in the Phase-7 cutover;
// the only corpus is now vrooli-docs.
type SearchRequest struct {
	Query     string  `json:"query"`
	Limit     int     `json:"limit,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	ID       string                 `json:"id"`
	Score    float64                `json:"score"`
	Content  string                 `json:"content"`
	Metadata map[string]interface{} `json:"metadata"`
}

// SearchResponse defines the output schema for semantic search
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Query   string         `json:"query"`
	Took    int64          `json:"took_ms"`
}

func validateAndNormalizeSearchRequest(req *SearchRequest) error {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return errors.New("Query parameter is required")
	}
	if req.Limit <= 0 {
		req.Limit = defaultSearchLimit
	}
	if req.Limit > maxSearchLimit {
		req.Limit = maxSearchLimit
	}
	if req.Threshold <= 0 {
		req.Threshold = defaultSearchThreshold
	}
	return nil
}

func normalizeStringList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := validateAndNormalizeSearchRequest(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if s == nil || s.docSearch == nil {
		s.respondError(w, http.StatusServiceUnavailable, "Search service unavailable")
		return
	}

	// Re-pointed at the hybrid documentation engine (Phase 6 cutover). The
	// legacy records-collection filters (collection/namespaces/visibility/tags)
	// no longer apply — the only semantic corpus is now vrooli-docs — so this
	// endpoint runs an auto (hybrid→dense→grep) documentation query and projects
	// the hit body into the historical {id,score,content,metadata} shape the UI
	// expects.
	out, err := s.docSearch.Search(r.Context(), pkg.SearchQuery{
		Query: req.Query,
		Mode:  pkg.ModeAuto,
		Limit: req.Limit,
	})
	if err != nil {
		s.log("search failed", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to execute search")
		return
	}

	results := make([]SearchResult, 0, len(out.Results))
	for _, h := range out.Results {
		results = append(results, SearchResult{
			ID:       h.ID,
			Score:    h.Score,
			Content:  h.Snippet,
			Metadata: h.Payload,
		})
	}

	response := SearchResponse{
		Results: results,
		Query:   req.Query,
		Took:    time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.log("failed to encode search response", map[string]interface{}{"error": err.Error()})
	}
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

// getCollections retrieves list of Qdrant collections
func (s *Server) getCollections(ctx context.Context) ([]string, error) {
	if collections, err := s.listQdrantCollectionsHTTP(ctx); err == nil {
		return collections, nil
	}

	output, err := s.execResourceQdrant(ctx, "collections")
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	return parseCollectionsOutput(output), nil
}

func parseCollectionsOutput(output []byte) []string {
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	collections := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "Collections:") {
			continue
		}
		collections = append(collections, line)
	}
	return collections
}
