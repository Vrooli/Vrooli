package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// QdrantSearchRequest represents a search request to Qdrant
type QdrantSearchRequest struct {
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVector  bool      `json:"with_vector"`
}

// QdrantSearchResponse represents the response from Qdrant
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
	Status string               `json:"status"`
}

type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// VectorSearchResult represents a search result for API response
type VectorSearchResult struct {
	IssueID     string  `json:"issue_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Status      string  `json:"status"`
	Similarity  float64 `json:"similarity"`
}

// generateMockEmbedding creates a mock embedding for testing
func generateMockEmbedding(text string) []float64 {
	// Simple hash-based mock embedding
	embedding := make([]float64, 1536)
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	for i := range embedding {
		embedding[i] = float64((hash+i)%1000)/1000.0 - 0.5
	}
	return embedding
}

// performVectorSearch searches for similar issues using Qdrant
func (s *Server) performVectorSearch(queryEmbedding []float64, limit int) ([]VectorSearchResult, error) {
	if strings.TrimSpace(s.config.QdrantURL) == "" {
		return nil, fmt.Errorf("vector search is not configured")
	}
	// Prepare search request
	searchReq := QdrantSearchRequest{
		Vector:      queryEmbedding,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("error marshaling search request: %v", err)
	}

	// Make request to Qdrant
	qdrantURL := fmt.Sprintf("%s/collections/issue_embeddings/points/search", s.config.QdrantURL)
	client := &http.Client{}

	resp, err := client.Post(qdrantURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("error calling Qdrant: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Qdrant returned status: %d", resp.StatusCode)
	}

	// Parse response
	var qdrantResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&qdrantResp); err != nil {
		return nil, fmt.Errorf("error parsing Qdrant response: %v", err)
	}

	// Convert to our result format
	results := make([]VectorSearchResult, 0, len(qdrantResp.Result))
	for _, result := range qdrantResp.Result {
		vectorResult := VectorSearchResult{
			IssueID:    result.ID,
			Similarity: result.Score,
		}

		// Extract payload fields
		if title, ok := result.Payload["title"].(string); ok {
			vectorResult.Title = title
		}
		if desc, ok := result.Payload["description"].(string); ok {
			vectorResult.Description = desc
		}
		if issueType, ok := result.Payload["type"].(string); ok {
			vectorResult.Type = issueType
		}
		if priority, ok := result.Payload["priority"].(string); ok {
			vectorResult.Priority = priority
		}
		if status, ok := result.Payload["status"].(string); ok {
			vectorResult.Status = status
		}

		results = append(results, vectorResult)
	}

	return results, nil
}

// indexIssueInVectorStore adds or updates an issue in the vector store
func (s *Server) indexIssueInVectorStore(issue Issue) error {
	if strings.TrimSpace(s.config.QdrantURL) == "" {
		return nil
	}
	// Generate embedding for the issue
	text := fmt.Sprintf("%s %s %s", issue.Title, issue.Description, issue.Type)
	embedding := generateMockEmbedding(text)

	// Prepare the payload
	payload := map[string]interface{}{
		"issue_id":    issue.ID,
		"title":       issue.Title,
		"description": issue.Description,
		"type":        issue.Type,
		"priority":    issue.Priority,
		"status":      issue.Status,
		"app_id":      issue.AppID,
	}

	// Create the point
	point := map[string]interface{}{
		"id":      issue.ID,
		"vector":  embedding,
		"payload": payload,
	}

	// Create the upsert request
	upsertReq := map[string]interface{}{
		"points": []interface{}{point},
	}

	reqBody, err := json.Marshal(upsertReq)
	if err != nil {
		return fmt.Errorf("error marshaling upsert request: %v", err)
	}

	// Make request to Qdrant
	qdrantURL := fmt.Sprintf("%s/collections/issue_embeddings/points", s.config.QdrantURL)
	client := &http.Client{}

	req, err := http.NewRequest("PUT", qdrantURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// Log error but don't fail the operation
		logWarn("Failed to index issue in vector store", "error", err, "issue_id", issue.ID)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logWarn("Qdrant returned unexpected status while indexing issue", "status", resp.StatusCode, "issue_id", issue.ID)
	}

	return nil
}

// createVectorCollection creates the issue_embeddings collection in Qdrant if it doesn't exist
func (s *Server) createVectorCollection() error {
	if strings.TrimSpace(s.config.QdrantURL) == "" {
		return nil
	}
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/issue_embeddings", s.config.QdrantURL)
	resp, err := http.Get(checkURL)
	if err == nil && resp.StatusCode == http.StatusOK {
		resp.Body.Close()
		return nil // Collection already exists
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Create collection
	collectionConfig := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     1536, // Size for OpenAI embeddings
			"distance": "Cosine",
		},
	}

	reqBody, err := json.Marshal(collectionConfig)
	if err != nil {
		return fmt.Errorf("error marshaling collection config: %v", err)
	}

	createURL := fmt.Sprintf("%s/collections/issue_embeddings", s.config.QdrantURL)
	req, err := http.NewRequest("PUT", createURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	createResp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error creating collection: %v", err)
	}
	defer createResp.Body.Close()

	if createResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create collection, status: %d", createResp.StatusCode)
	}

	return nil
}

// vectorSearchHandler handles semantic search requests
func (s *Server) vectorSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Generate embedding for the query
	// In production, this would use OpenAI API or local embedding model
	queryEmbedding := generateMockEmbedding(query)

	// Perform vector search
	results, err := s.performVectorSearch(queryEmbedding, limit)
	if err != nil {
		// Fallback to text search if vector search fails
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		response := ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Vector search failed, falling back to text search: %v", err),
			Data: map[string]interface{}{
				"fallback":     true,
				"vector_error": err.Error(),
			},
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Return results
	response := ApiResponse{
		Success: true,
		Message: "Vector search completed successfully",
		Data: map[string]interface{}{
			"results": results,
			"count":   len(results),
			"query":   query,
			"method":  "vector_similarity",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
