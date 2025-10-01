package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type VectorSearchManager struct {
	db             *sql.DB
	qdrantURL      string
	collectionName string
	dimensions     int
}

type QdrantPoint struct {
	ID      string                 `json:"id"`
	Vector  []float64              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type QdrantSearchRequest struct {
	Vector      []float64 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
	WithVector  bool      `json:"with_vector"`
}

type QdrantSearchResponse struct {
	Result []struct {
		ID      string                 `json:"id"`
		Version int                    `json:"version"`
		Score   float64                `json:"score"`
		Payload map[string]interface{} `json:"payload"`
		Vector  []float64              `json:"vector,omitempty"`
	} `json:"result"`
}

type EventEmbedding struct {
	ID               string    `json:"id"`
	EventID          string    `json:"event_id"`
	QdrantPointID    string    `json:"qdrant_point_id"`
	EmbeddingVersion string    `json:"embedding_version"`
	ContentHash      string    `json:"content_hash"`
	Keywords         []string  `json:"keywords"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

func NewVectorSearchManager(db *sql.DB, qdrantURL string) *VectorSearchManager {
	return &VectorSearchManager{
		db:             db,
		qdrantURL:      qdrantURL,
		collectionName: "calendar_events",
		dimensions:     1536, // Standard OpenAI embedding dimensions
	}
}

// CreateEmbeddingForEvent generates and stores an embedding for an event
func (vsm *VectorSearchManager) CreateEmbeddingForEvent(ctx context.Context, event Event) error {
	if vsm.qdrantURL == "" {
		// Skip if Qdrant not configured
		return nil
	}

	// Generate content for embedding
	content := vsm.generateEventContent(event)

	// Calculate content hash
	hash := sha256.Sum256([]byte(content))
	contentHash := fmt.Sprintf("%x", hash)

	// Check if embedding already exists with same content
	var existingHash string
	checkQuery := `SELECT content_hash FROM event_embeddings WHERE event_id = $1`
	err := vsm.db.QueryRowContext(ctx, checkQuery, event.ID).Scan(&existingHash)

	if err == nil && existingHash == contentHash {
		// Content hasn't changed, no need to update embedding
		return nil
	}

	// Generate embedding vector (placeholder - in real implementation, this would call OpenAI or similar)
	vector := vsm.generateMockEmbedding(content)

	// Generate keywords for better searchability
	keywords := vsm.extractKeywords(content)

	// Create Qdrant point
	pointID := uuid.New().String()
	point := QdrantPoint{
		ID:     pointID,
		Vector: vector,
		Payload: map[string]interface{}{
			"event_id":    event.ID,
			"user_id":     event.UserID,
			"title":       event.Title,
			"description": event.Description,
			"event_type":  event.EventType,
			"location":    event.Location,
			"start_time":  event.StartTime.Unix(),
			"keywords":    keywords,
			"content":     content,
		},
	}

	// Store in Qdrant
	if err := vsm.storePointInQdrant(ctx, point); err != nil {
		return fmt.Errorf("failed to store point in Qdrant: %w", err)
	}

	// Store embedding metadata in PostgreSQL
	if err == sql.ErrNoRows {
		// Insert new embedding record
		insertQuery := `
			INSERT INTO event_embeddings (id, event_id, qdrant_point_id, embedding_version, content_hash, keywords, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`

		keywordsJSON, _ := json.Marshal(keywords)
		_, err = vsm.db.ExecContext(ctx, insertQuery, uuid.New().String(), event.ID, pointID, "v1.0", contentHash, keywordsJSON)
	} else {
		// Update existing embedding record
		updateQuery := `
			UPDATE event_embeddings 
			SET qdrant_point_id = $2, content_hash = $3, keywords = $4, updated_at = NOW()
			WHERE event_id = $1`

		keywordsJSON, _ := json.Marshal(keywords)
		_, err = vsm.db.ExecContext(ctx, updateQuery, event.ID, pointID, contentHash, keywordsJSON)
	}

	if err != nil {
		return fmt.Errorf("failed to store embedding metadata: %w", err)
	}

	return nil
}

// SearchEventsBySemantic performs semantic search using Qdrant
func (vsm *VectorSearchManager) SearchEventsBySemantic(ctx context.Context, query string, userID string, limit int) ([]Event, error) {
	if vsm.qdrantURL == "" {
		// Fallback to basic text search if Qdrant not configured
		return vsm.fallbackTextSearch(ctx, query, userID, limit)
	}

	// Generate embedding for search query (mock implementation)
	queryVector := vsm.generateMockEmbedding(query)

	// Search in Qdrant
	searchReq := QdrantSearchRequest{
		Vector:      queryVector,
		Limit:       limit,
		WithPayload: true,
		WithVector:  false,
	}

	searchURL := fmt.Sprintf("%s/collections/%s/points/search", vsm.qdrantURL, vsm.collectionName)
	jsonData, err := json.Marshal(searchReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal search request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Fallback to text search on error
		return vsm.fallbackTextSearch(ctx, query, userID, limit)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Fallback to text search on error
		return vsm.fallbackTextSearch(ctx, query, userID, limit)
	}

	var searchResponse QdrantSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Extract event IDs from search results
	var eventIDs []string
	for _, result := range searchResponse.Result {
		if eventID, ok := result.Payload["event_id"].(string); ok {
			eventIDs = append(eventIDs, eventID)
		}
	}

	if len(eventIDs) == 0 {
		return []Event{}, nil
	}

	// Fetch full event details from PostgreSQL
	return vsm.getEventsByIDs(ctx, eventIDs, userID)
}

// generateEventContent creates searchable content from an event
func (vsm *VectorSearchManager) generateEventContent(event Event) string {
	content := []string{event.Title}

	if event.Description != "" {
		content = append(content, event.Description)
	}

	if event.Location != "" {
		content = append(content, event.Location)
	}

	content = append(content, event.EventType)
	content = append(content, event.StartTime.Format("January 2006"))

	return strings.Join(content, " ")
}

// generateMockEmbedding creates a mock embedding vector
// In a real implementation, this would call OpenAI's embedding API or similar
func (vsm *VectorSearchManager) generateMockEmbedding(text string) []float64 {
	// This is a placeholder implementation
	// In production, you would call OpenAI's embedding API or use a local model
	vector := make([]float64, vsm.dimensions)

	// Generate deterministic but varied embeddings based on text content
	textBytes := []byte(strings.ToLower(text))
	for i := 0; i < vsm.dimensions; i++ {
		if i < len(textBytes) {
			vector[i] = float64(int(textBytes[i])) / 256.0
		} else {
			vector[i] = float64((i*37+len(textBytes))%256) / 256.0
		}
	}

	return vector
}

// extractKeywords extracts searchable keywords from content
func (vsm *VectorSearchManager) extractKeywords(content string) []string {
	words := strings.Fields(strings.ToLower(content))
	keywords := make([]string, 0)

	for _, word := range words {
		// Basic keyword extraction - remove common words and short words
		word = strings.Trim(word, ".,!?;:")
		if len(word) > 3 && !vsm.isStopWord(word) {
			keywords = append(keywords, word)
		}
	}

	return keywords
}

// isStopWord checks if a word is a common stop word
func (vsm *VectorSearchManager) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "or": true, "but": true, "in": true, "on": true, "at": true, "to": true, "for": true,
		"of": true, "with": true, "by": true, "from": true, "up": true, "about": true, "into": true, "through": true,
		"during": true, "before": true, "after": true, "above": true, "below": true, "between": true, "among": true,
		"this": true, "that": true, "these": true, "those": true, "a": true, "an": true, "is": true, "was": true,
		"are": true, "were": true, "will": true, "would": true, "could": true, "should": true, "may": true, "might": true,
	}

	return stopWords[word]
}

// storePointInQdrant stores a vector point in Qdrant
func (vsm *VectorSearchManager) storePointInQdrant(ctx context.Context, point QdrantPoint) error {
	upsertURL := fmt.Sprintf("%s/collections/%s/points", vsm.qdrantURL, vsm.collectionName)

	payload := map[string]interface{}{
		"points": []QdrantPoint{point},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal point: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", upsertURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create upsert request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to store point: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Qdrant returned status %d", resp.StatusCode)
	}

	return nil
}

// fallbackTextSearch provides basic text search when Qdrant is unavailable
func (vsm *VectorSearchManager) fallbackTextSearch(ctx context.Context, query string, userID string, limit int) ([]Event, error) {
	searchQuery := `
		SELECT id, user_id, title, description, start_time, end_time, timezone, 
		       location, event_type, status, created_at, updated_at
		FROM events 
		WHERE user_id = $1 AND status = 'active'
		  AND (title ILIKE $2 OR description ILIKE $2)
		ORDER BY start_time ASC 
		LIMIT $3`

	rows, err := vsm.db.QueryContext(ctx, searchQuery, userID, "%"+query+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute fallback search: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// getEventsByIDs fetches events by their IDs
func (vsm *VectorSearchManager) getEventsByIDs(ctx context.Context, eventIDs []string, userID string) ([]Event, error) {
	if len(eventIDs) == 0 {
		return []Event{}, nil
	}

	// Build IN clause for SQL query
	placeholders := make([]string, len(eventIDs))
	args := []interface{}{userID}
	for i, id := range eventIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT id, user_id, title, description, start_time, end_time, timezone, 
		       location, event_type, status, created_at, updated_at
		FROM events 
		WHERE user_id = $1 AND id IN (%s) AND status != 'deleted'
		ORDER BY start_time ASC`,
		strings.Join(placeholders, ","))

	rows, err := vsm.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events by IDs: %w", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err := rows.Scan(
			&event.ID, &event.UserID, &event.Title, &event.Description,
			&event.StartTime, &event.EndTime, &event.Timezone, &event.Location,
			&event.EventType, &event.Status, &event.CreatedAt, &event.UpdatedAt,
		)
		if err != nil {
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// DeleteEmbeddingForEvent removes an event's embedding from Qdrant and PostgreSQL
func (vsm *VectorSearchManager) DeleteEmbeddingForEvent(ctx context.Context, eventID string) error {
	if vsm.qdrantURL == "" {
		return nil
	}

	// Get Qdrant point ID
	var pointID string
	getQuery := `SELECT qdrant_point_id FROM event_embeddings WHERE event_id = $1`
	err := vsm.db.QueryRowContext(ctx, getQuery, eventID).Scan(&pointID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil // No embedding exists
		}
		return fmt.Errorf("failed to get embedding point ID: %w", err)
	}

	// Delete from Qdrant
	deleteURL := fmt.Sprintf("%s/collections/%s/points/delete", vsm.qdrantURL, vsm.collectionName)
	payload := map[string]interface{}{
		"points": []string{pointID},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal delete request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", deleteURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete from Qdrant: %w", err)
	}
	defer resp.Body.Close()

	// Delete from PostgreSQL
	deleteQuery := `DELETE FROM event_embeddings WHERE event_id = $1`
	if _, err := vsm.db.ExecContext(ctx, deleteQuery, eventID); err != nil {
		return fmt.Errorf("failed to delete embedding metadata: %w", err)
	}

	return nil
}
