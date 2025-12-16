package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultSearchLimit     = 10
	maxSearchLimit         = 100
	defaultSearchThreshold = 0.3
)

// SearchRequest defines the input schema for semantic search
type SearchRequest struct {
	Query      string  `json:"query"`
	Collection string  `json:"collection,omitempty"`
	Limit      int     `json:"limit,omitempty"`
	Threshold  float64 `json:"threshold,omitempty"`
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

	// Generate embedding via Ollama [REQ:KO-SS-002]
	embedding, err := s.generateEmbedding(r.Context(), req.Query)
	if err != nil {
		s.log("embedding generation failed", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to generate query embedding")
		return
	}

	// Search Qdrant [REQ:KO-SS-003]
	results, err := s.searchQdrant(r.Context(), req.Collection, embedding, req.Limit, req.Threshold)
	if err != nil {
		s.log("qdrant search failed", map[string]interface{}{"error": err.Error()})
		s.respondError(w, http.StatusInternalServerError, "Failed to execute search")
		return
	}

	response := SearchResponse{
		Results: results,
		Query:   req.Query,
		Took:    time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateEmbedding creates a vector embedding using Ollama [REQ:KO-SS-002]
func (s *Server) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	ollamaURL := s.ollamaURL()

	reqBody := OllamaEmbeddingRequest{
		Model:  s.ollamaEmbeddingModel(),
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", ollamaURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var embResp OllamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return embResp.Embedding, nil
}

func sortAndLimitResults(results []SearchResult, limit int) []SearchResult {
	if len(results) == 0 {
		return results
	}

	sort.SliceStable(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

func containsString(values []string, needle string) bool {
	for _, v := range values {
		if v == needle {
			return true
		}
	}
	return false
}

// searchQdrant executes vector similarity search [REQ:KO-SS-003]
func (s *Server) searchQdrant(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	// Get collections to search
	collections, err := s.getCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// If collection specified, validate it exists
	if collection != "" {
		if !containsString(collections, collection) {
			return nil, fmt.Errorf("collection %s not found", collection)
		}
		collections = []string{collection}
	}

	// Search across collections
	var allResults []SearchResult
	for _, coll := range collections {
		results, err := s.searchSingleCollection(ctx, coll, vector, limit, threshold)
		if err != nil {
			s.log("collection search failed", map[string]interface{}{
				"collection": coll,
				"error":      err.Error(),
			})
			continue
		}
		allResults = append(allResults, results...)
	}

	return sortAndLimitResults(allResults, limit), nil
}

// getCollections retrieves list of Qdrant collections
func (s *Server) getCollections(ctx context.Context) ([]string, error) {
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

func extractContentFromPayload(payload map[string]interface{}) string {
	if payload == nil {
		return ""
	}
	if c, ok := payload["content"].(string); ok {
		return c
	}
	if c, ok := payload["text"].(string); ok {
		return c
	}
	return ""
}

func stringifyQdrantID(id interface{}) string {
	switch v := id.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', 0, 64)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// searchSingleCollection searches a specific Qdrant collection
func (s *Server) searchSingleCollection(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	qdrantURL := s.qdrantURL()

	searchReq := QdrantSearchRequest{
		Vector:         vector,
		Limit:          limit,
		WithPayload:    true,
		ScoreThreshold: &threshold,
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	searchURL := fmt.Sprintf("%s/collections/%s/points/search", qdrantURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("qdrant request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("qdrant returned status %d: %s", resp.StatusCode, string(body))
	}

	var qdrantResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert Qdrant results to our format
	var results []SearchResult
	for _, point := range qdrantResp.Result {
		results = append(results, SearchResult{
			ID:       stringifyQdrantID(point.ID),
			Score:    point.Score,
			Content:  extractContentFromPayload(point.Payload),
			Metadata: point.Payload,
		})
	}

	return results, nil
}
