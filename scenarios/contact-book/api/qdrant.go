package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
)

// QdrantClient handles vector search operations
type QdrantClient struct {
	baseURL    string
	collection string
	httpClient *http.Client
}

// Vector represents an embedding vector
type Vector struct {
	ID     string    `json:"id"`
	Vector []float32 `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

// SearchResult represents a search result from Qdrant
type SearchResult struct {
	ID      string                 `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

// NewQdrantClient creates a new Qdrant client
func NewQdrantClient() *QdrantClient {
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}

	return &QdrantClient{
		baseURL:    qdrantURL,
		collection: "contact_book_persons",
		httpClient: &http.Client{},
	}
}

// CreateCollection creates the vector collection if it doesn't exist
func (q *QdrantClient) CreateCollection() error {
	// Check if collection exists
	checkURL := fmt.Sprintf("%s/collections/%s", q.baseURL, q.collection)
	resp, err := q.httpClient.Get(checkURL)
	if err == nil && resp.StatusCode == 200 {
		resp.Body.Close()
		return nil // Collection already exists
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Create collection with 384 dimensions (for all-MiniLM-L6-v2 model)
	createURL := fmt.Sprintf("%s/collections/%s", q.baseURL, q.collection)
	body := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     384,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err = q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(bodyBytes))
	}

	return nil
}

// GenerateEmbedding generates an embedding for the given text
// This is a placeholder - in production, you would use a proper embedding model
func (q *QdrantClient) GenerateEmbedding(text string) ([]float32, error) {
	// For now, we'll use a simple hash-based approach for demonstration
	// In production, use sentence-transformers or OpenAI embeddings

	// Create a deterministic pseudo-embedding based on text content
	embedding := make([]float32, 384)

	// Simple hash-based approach for demo purposes
	words := strings.Fields(strings.ToLower(text))
	for i, word := range words {
		for j, char := range word {
			idx := (i*31 + j*17 + int(char)) % 384
			embedding[idx] += float32(char) / 1000.0
		}
	}

	// Normalize the vector
	var sum float32
	for _, v := range embedding {
		sum += v * v
	}
	norm := float32(1.0)
	if sum > 0 {
		norm = 1.0 / float32(math.Sqrt(float64(sum)))
	}
	for i := range embedding {
		embedding[i] *= norm
	}

	return embedding, nil
}

// IndexPerson adds a person to the vector index
func (q *QdrantClient) IndexPerson(person Person) error {
	// Create searchable text from person data
	searchText := fmt.Sprintf("%s %s %s %s %s",
		person.FullName,
		strings.Join(person.Emails, " "),
		strings.Join(person.Tags, " "),
		getStringValue(person.DisplayName),
		getStringValue(person.Notes),
	)

	// Generate embedding
	embedding, err := q.GenerateEmbedding(searchText)
	if err != nil {
		return err
	}

	// Prepare the point for Qdrant
	point := map[string]interface{}{
		"id":     person.ID,
		"vector": embedding,
		"payload": map[string]interface{}{
			"full_name":    person.FullName,
			"display_name": person.DisplayName,
			"emails":       person.Emails,
			"tags":         person.Tags,
			"notes":        person.Notes,
		},
	}

	// Upsert the point
	upsertURL := fmt.Sprintf("%s/collections/%s/points", q.baseURL, q.collection)
	body := map[string]interface{}{
		"points": []interface{}{point},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", upsertURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to index person: %s", string(bodyBytes))
	}

	return nil
}

// SearchPersons performs semantic search for persons
func (q *QdrantClient) SearchPersons(query string, limit int) ([]SearchResult, error) {
	// Generate embedding for query
	embedding, err := q.GenerateEmbedding(query)
	if err != nil {
		return nil, err
	}

	// Search in Qdrant
	searchURL := fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, q.collection)
	body := map[string]interface{}{
		"vector": embedding,
		"limit":  limit,
		"with_payload": true,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", searchURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(bodyBytes))
	}

	// Parse response
	var searchResp struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float32                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(searchResp.Result))
	for i, r := range searchResp.Result {
		results[i] = SearchResult{
			ID:      r.ID,
			Score:   r.Score,
			Payload: r.Payload,
		}
	}

	return results, nil
}

// DeletePerson removes a person from the vector index
func (q *QdrantClient) DeletePerson(personID string) error {
	deleteURL := fmt.Sprintf("%s/collections/%s/points/delete", q.baseURL, q.collection)
	body := map[string]interface{}{
		"points": []string{personID},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", deleteURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete person: %s", string(bodyBytes))
	}

	return nil
}

// Helper function to get string value from pointer
func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}