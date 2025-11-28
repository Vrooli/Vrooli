package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	ScoreThreshold *float64 `json:"score_threshold,omitempty"`
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

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate query
	if strings.TrimSpace(req.Query) == "" {
		s.respondError(w, http.StatusBadRequest, "Query parameter is required")
		return
	}

	// Set defaults
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.Threshold <= 0 {
		req.Threshold = 0.3 // Default similarity threshold
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
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	reqBody := OllamaEmbeddingRequest{
		Model:  "nomic-embed-text",
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

// searchQdrant executes vector similarity search [REQ:KO-SS-003]
func (s *Server) searchQdrant(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	// Get collections to search
	collections, err := s.getCollections(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	// If collection specified, validate it exists
	if collection != "" {
		found := false
		for _, c := range collections {
			if c == collection {
				found = true
				break
			}
		}
		if !found {
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

	// Sort by score (highest first) and limit
	// TODO: Implement proper sorting and limiting across all collections
	if len(allResults) > limit {
		allResults = allResults[:limit]
	}

	return allResults, nil
}

// getCollections retrieves list of Qdrant collections
func (s *Server) getCollections(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "resource-qdrant", "collections")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var collections []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, "Collections:") {
			collections = append(collections, line)
		}
	}

	return collections, nil
}

// searchSingleCollection searches a specific Qdrant collection
func (s *Server) searchSingleCollection(ctx context.Context, collection string, vector []float64, limit int, threshold float64) ([]SearchResult, error) {
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	searchReq := QdrantSearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
		ScoreThreshold: &threshold,
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/collections/%s/points/search", qdrantURL, collection)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
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
	for _, r := range qdrantResp.Result {
		// Extract content from payload
		content := ""
		if c, ok := r.Payload["content"].(string); ok {
			content = c
		} else if c, ok := r.Payload["text"].(string); ok {
			content = c
		}

		// Convert ID to string
		idStr := ""
		switch v := r.ID.(type) {
		case string:
			idStr = v
		case float64:
			idStr = strconv.FormatFloat(v, 'f', 0, 64)
		default:
			idStr = fmt.Sprintf("%v", v)
		}

		results = append(results, SearchResult{
			ID:       idStr,
			Score:    r.Score,
			Content:  content,
			Metadata: r.Payload,
		})
	}

	return results, nil
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
