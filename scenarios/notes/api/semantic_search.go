package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type VectorSearchRequest struct {
	Query  string `json:"query"`
	UserID string `json:"user_id"`
	Limit  int    `json:"limit"`
}

type VectorSearchResponse struct {
	Results []VectorSearchResult `json:"results"`
	Query   string               `json:"query"`
	Count   int                  `json:"count"`
}

type VectorSearchResult struct {
	ID       string  `json:"id"`
	Score    float64 `json:"score"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Summary  string  `json:"summary,omitempty"`
	FolderID string  `json:"folder_id,omitempty"`
}

type QdrantSearchRequest struct {
	Vector []float64              `json:"vector"`
	Limit  int                    `json:"limit"`
	Filter map[string]interface{} `json:"filter,omitempty"`
	With   []string               `json:"with_payload"`
}

type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
}

type QdrantSearchResult struct {
	ID      string                 `json:"id"`
	Score   float64                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Embeddings [][]float64 `json:"embeddings"`
}

func getEmbedding(text string) ([]float64, error) {
	// OLLAMA_HOST: Configurable via environment variable (default: localhost:11434)
	// Override in service.json or .env to use a different Ollama instance
	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "localhost:11434" // Standard Ollama default port
	}

	// Use nomic-embed-text model for embeddings
	reqBody := EmbeddingRequest{
		Model: "nomic-embed-text",
		Input: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/api/embed", ollamaHost),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Ollama returns a different format
	var ollamaResp map[string]interface{}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, err
	}

	// Extract embeddings from Ollama response
	if embeddings, ok := ollamaResp["embeddings"].([]interface{}); ok && len(embeddings) > 0 {
		if embedding, ok := embeddings[0].([]interface{}); ok {
			result := make([]float64, len(embedding))
			for i, v := range embedding {
				if val, ok := v.(float64); ok {
					result[i] = val
				}
			}
			return result, nil
		}
	}

	return nil, fmt.Errorf("failed to parse embedding response")
}

func searchQdrant(vector []float64, userID string, limit int) ([]QdrantSearchResult, error) {
	// QDRANT_HOST: Configurable via environment variable (default: localhost:6333)
	// Override in service.json or .env to use a different Qdrant instance
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost:6333" // Standard Qdrant default port
	}

	searchReq := QdrantSearchRequest{
		Vector: vector,
		Limit:  limit,
		With:   []string{"payload"},
	}

	if userID != "" {
		searchReq.Filter = map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key":   "user_id",
					"match": map[string]string{"value": userID},
				},
			},
		}
	}

	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(
		fmt.Sprintf("http://%s/collections/notes/points/search", qdrantHost),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var searchResp QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	return searchResp.Result, nil
}

func semanticSearchHandler(w http.ResponseWriter, r *http.Request) {
	var req VectorSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		req.UserID = getDefaultUserID()
	}
	if req.Limit == 0 {
		req.Limit = 20
	}

	// Get embedding for the query
	vector, err := getEmbedding(req.Query)
	if err != nil {
		logger.Warn("Could not generate embedding - falling back to text search", "error", err.Error())
		// Reconstruct request body for text search fallback
		fallbackBody := map[string]interface{}{
			"query":   req.Query,
			"user_id": req.UserID,
			"limit":   req.Limit,
		}
		jsonData, _ := json.Marshal(fallbackBody)
		r.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		searchHandler(w, r)
		return
	}

	// Search Qdrant
	qdrantResults, err := searchQdrant(vector, req.UserID, req.Limit)
	if err != nil {
		logger.Warn("Qdrant search failed - falling back to text search", "error", err.Error())
		// Reconstruct request body for text search fallback
		fallbackBody := map[string]interface{}{
			"query":   req.Query,
			"user_id": req.UserID,
			"limit":   req.Limit,
		}
		jsonData, _ := json.Marshal(fallbackBody)
		r.Body = io.NopCloser(bytes.NewBuffer(jsonData))
		searchHandler(w, r)
		return
	}

	// Convert Qdrant results to our response format
	var results []VectorSearchResult
	for _, qr := range qdrantResults {
		result := VectorSearchResult{
			ID:    qr.ID,
			Score: qr.Score,
		}

		if title, ok := qr.Payload["title"].(string); ok {
			result.Title = title
		}
		if content, ok := qr.Payload["content"].(string); ok {
			result.Content = content
		}
		if summary, ok := qr.Payload["summary"].(string); ok {
			result.Summary = summary
		}
		if folderID, ok := qr.Payload["folder_id"].(string); ok {
			result.FolderID = folderID
		}

		results = append(results, result)
	}

	response := VectorSearchResponse{
		Results: results,
		Query:   req.Query,
		Count:   len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Function to index a note in Qdrant
func indexNoteInQdrant(note Note) error {
	// Combine title and content for embedding
	textToEmbed := fmt.Sprintf("%s\n%s", note.Title, note.Content)

	vector, err := getEmbedding(textToEmbed)
	if err != nil {
		// Log error but don't fail the note creation
		logger.Warn("Could not generate embedding for note", "note_id", note.ID, "error", err.Error())
		return nil
	}

	// QDRANT_HOST: Configurable via environment variable (default: localhost:6333)
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost:6333" // Standard Qdrant default port
	}

	// Prepare payload
	payload := map[string]interface{}{
		"note_id":     note.ID,
		"user_id":     note.UserID,
		"title":       note.Title,
		"content":     note.Content,
		"folder_id":   note.FolderID,
		"is_pinned":   note.IsPinned,
		"is_favorite": note.IsFavorite,
		"created_at":  note.CreatedAt,
		"updated_at":  note.UpdatedAt,
	}

	if note.Summary != nil {
		payload["summary"] = *note.Summary
	}

	// Prepare point for Qdrant
	point := map[string]interface{}{
		"id":      note.ID,
		"vector":  vector,
		"payload": payload,
	}

	points := map[string]interface{}{
		"points": []interface{}{point},
	}

	jsonData, err := json.Marshal(points)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"PUT",
		fmt.Sprintf("http://%s/collections/notes/points", qdrantHost),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Could not index note in Qdrant", "error", err.Error())
		return nil
	}
	defer resp.Body.Close()

	return nil
}

// Function to delete a note from Qdrant
func deleteNoteFromQdrant(noteID string) error {
	// QDRANT_HOST: Configurable via environment variable (default: localhost:6333)
	qdrantHost := os.Getenv("QDRANT_HOST")
	if qdrantHost == "" {
		qdrantHost = "localhost:6333" // Standard Qdrant default port
	}

	deleteReq := map[string]interface{}{
		"points": []string{noteID},
	}

	jsonData, err := json.Marshal(deleteReq)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("http://%s/collections/notes/points/delete", qdrantHost),
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warn("Could not delete note from Qdrant", "error", err.Error())
		return nil
	}
	defer resp.Body.Close()

	return nil
}
