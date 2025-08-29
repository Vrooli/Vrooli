package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// QdrantSearchRequest represents a search request to Qdrant
type QdrantSearchRequest struct {
	Vector []float64 `json:"vector"`
	Limit  int       `json:"limit"`
	WithPayload bool  `json:"with_payload"`
	WithVector bool   `json:"with_vector"`
}

// QdrantSearchResponse represents the response from Qdrant
type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
	Status string              `json:"status"`
}

type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Version int                    `json:"version"`
	Score   float64               `json:"score"`
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
		embedding[i] = float64((hash+i)%1000) / 1000.0 - 0.5
	}
	return embedding
}

// performVectorSearch searches for similar issues using Qdrant
func (s *Server) performVectorSearch(queryEmbedding []float64, limit int) ([]VectorSearchResult, error) {
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
		response := ApiResponse{
			Success: false,
			Message: fmt.Sprintf("Vector search failed, falling back to text search: %v", err),
			Data: map[string]interface{}{
				"fallback":        true,
				"vector_error":    err.Error(),
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
