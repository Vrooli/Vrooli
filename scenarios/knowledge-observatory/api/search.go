package main

// DOC: docs/reference/api-endpoints.md#search
// DOC: docs/concepts/ARCHITECTURE.md#search-flow
import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	pkg "github.com/vrooli/aisearch-go"
)

const (
	defaultSearchLimit     = 10
	maxSearchLimit         = 100
	defaultSearchThreshold = 0.3
)

// SearchRequest defines the input schema for semantic search
type SearchRequest struct {
	Query          string   `json:"query"`
	Collection     string   `json:"collection,omitempty"`
	Namespaces     []string `json:"namespaces,omitempty"`
	Visibility     []string `json:"visibility,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	IngestedAfter  string   `json:"ingested_after,omitempty"`
	IngestedBefore string   `json:"ingested_before,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Threshold      float64  `json:"threshold,omitempty"`
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

// OllamaEmbeddingRequest represents a request to Ollama's embedding API
type OllamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaEmbeddingResponse represents the response from Ollama
type OllamaEmbeddingResponse struct {
	Embedding []float64 `json:"embedding"`
}

// QdrantSearchRequest represents a vector search request to Qdrant
type QdrantSearchRequest struct {
	Vector         []float64 `json:"vector"`
	Limit          int       `json:"limit"`
	WithPayload    bool      `json:"with_payload"`
	ScoreThreshold *float64  `json:"score_threshold,omitempty"`
}

// QdrantSearchResponse represents Qdrant's search response
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
}

// QdrantSearchResult represents a single result from Qdrant
type QdrantSearchResult struct {
	ID      interface{}            `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

func validateAndNormalizeSearchRequest(req *SearchRequest) error {
	req.Query = strings.TrimSpace(req.Query)
	if req.Query == "" {
		return errors.New("Query parameter is required")
	}

	req.Collection = strings.TrimSpace(req.Collection)
	req.Namespaces = normalizeStringList(req.Namespaces)
	req.Tags = normalizeStringList(req.Tags)
	visibility, err := normalizeVisibilityListStrict(req.Visibility)
	if err != nil {
		return err
	}
	req.Visibility = visibility

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

func parseRFC3339Millis(value string) (*int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	ms := parsed.UnixMilli()
	return &ms, nil
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

	// IngestedAfter/Before were records-era filters; the documentation corpus
	// has no ingest timestamps, but keep validating the RFC3339 shape so
	// malformed input is still rejected with a 400.
	if _, err := parseRFC3339Millis(req.IngestedAfter); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid ingested_after (must be RFC3339)")
		return
	}
	if _, err := parseRFC3339Millis(req.IngestedBefore); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid ingested_before (must be RFC3339)")
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

func sortAndLimitResults(results []SearchResult, limit int) []SearchResult {
	if len(results) == 0 {
		return []SearchResult{}
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
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
