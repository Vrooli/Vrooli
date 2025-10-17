package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// QdrantClient handles vector database operations
type QdrantClient struct {
	baseURL    string
	httpClient *http.Client
}

// VectorPoint represents a point in the vector database
type VectorPoint struct {
	ID      string                 `json:"id"`
	Vector  []float32              `json:"vector"`
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
	baseURL := os.Getenv("QDRANT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:6333"
	}

	return &QdrantClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// CreateCollection creates a new collection in Qdrant
func (q *QdrantClient) CreateCollection(collectionName string, vectorSize int) error {
	payload := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s", q.baseURL, collectionName), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection: %s", string(body))
	}

	return nil
}

// UpsertPoints adds or updates points in the collection
func (q *QdrantClient) UpsertPoints(collectionName string, points []VectorPoint) error {
	payload := map[string]interface{}{
		"points": points,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/collections/%s/points", q.baseURL, collectionName), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to upsert points: %s", string(body))
	}

	return nil
}

// SearchSimilar searches for similar vectors
func (q *QdrantClient) SearchSimilar(collectionName string, vector []float32, limit int) ([]SearchResult, error) {
	payload := map[string]interface{}{
		"vector":       vector,
		"limit":        limit,
		"with_payload": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/collections/%s/points/search", q.baseURL, collectionName), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := q.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search failed: %s", string(body))
	}

	var response struct {
		Result []SearchResult `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Result, nil
}

// GenerateEmbedding generates embeddings using Ollama
func GenerateEmbedding(text string) ([]float32, error) {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}

	payload := map[string]interface{}{
		"model":  "nomic-embed-text",
		"prompt": text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/embeddings", ollamaURL), bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to generate embedding: %s", string(body))
	}

	var response struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Embedding, nil
}

// InitializeVectorSearch sets up the vector database for injection techniques
func InitializeVectorSearch() error {
	client := NewQdrantClient()

	// Create collection if it doesn't exist
	err := client.CreateCollection("injection_techniques", 768) // nomic-embed-text produces 768-dim vectors
	if err != nil {
		log.Printf("Collection might already exist or error creating: %v", err)
	}

	// Load existing techniques and generate embeddings
	query := `SELECT id, name, category, description, example_prompt FROM injection_techniques WHERE is_active = true`
	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("failed to load injection techniques: %v", err)
	}
	defer rows.Close()

	var points []VectorPoint
	for rows.Next() {
		var id, name, category, description, examplePrompt string
		if err := rows.Scan(&id, &name, &category, &description, &examplePrompt); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Generate embedding for the technique
		textToEmbed := fmt.Sprintf("%s %s %s %s", name, category, description, examplePrompt)
		embedding, err := GenerateEmbedding(textToEmbed)
		if err != nil {
			log.Printf("Error generating embedding for %s: %v", name, err)
			continue
		}

		point := VectorPoint{
			ID:     id,
			Vector: embedding,
			Payload: map[string]interface{}{
				"name":           name,
				"category":       category,
				"description":    description,
				"example_prompt": examplePrompt,
			},
		}
		points = append(points, point)
	}

	// Upsert points to Qdrant
	if len(points) > 0 {
		if err := client.UpsertPoints("injection_techniques", points); err != nil {
			return fmt.Errorf("failed to upsert points: %v", err)
		}
		log.Printf("Successfully indexed %d injection techniques", len(points))
	}

	return nil
}

// FindSimilarInjections finds injection techniques similar to the given text
func FindSimilarInjections(queryText string, limit int) ([]map[string]interface{}, error) {
	// Generate embedding for the query
	embedding, err := GenerateEmbedding(queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %v", err)
	}

	// Search for similar vectors
	client := NewQdrantClient()
	results, err := client.SearchSimilar("injection_techniques", embedding, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar: %v", err)
	}

	// Format results
	var similar []map[string]interface{}
	for _, result := range results {
		item := map[string]interface{}{
			"id":             result.ID,
			"score":          result.Score,
			"name":           result.Payload["name"],
			"category":       result.Payload["category"],
			"description":    result.Payload["description"],
			"example_prompt": result.Payload["example_prompt"],
		}
		similar = append(similar, item)
	}

	return similar, nil
}
