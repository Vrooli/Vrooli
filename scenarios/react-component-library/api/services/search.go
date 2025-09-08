package services

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/scenarios/react-component-library/models"
)

type SearchService struct {
	collectionName string
}

func NewSearchService() *SearchService {
	return &SearchService{
		collectionName: "react_components",
	}
}

// SearchComponents performs semantic search across components
func (s *SearchService) SearchComponents(req models.ComponentSearchRequest) (*models.ComponentSearchResponse, error) {
	// Use Qdrant for semantic search
	results, err := s.semanticSearch(req.Query, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("semantic search failed: %w", err)
	}

	// Convert Qdrant results to components
	components, err := s.convertSearchResults(results, req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert search results: %w", err)
	}

	response := &models.ComponentSearchResponse{
		Components: components,
		Total:      len(components),
		Query:      req.Query,
	}

	return response, nil
}

// IndexComponent adds or updates a component in the search index
func (s *SearchService) IndexComponent(component *models.Component) error {
	// Create embedding for component
	embedding, err := s.createComponentEmbedding(component)
	if err != nil {
		return fmt.Errorf("failed to create embedding: %w", err)
	}

	// Prepare metadata
	metadata := map[string]interface{}{
		"component_id":    component.ID.String(),
		"name":           component.Name,
		"category":       component.Category,
		"description":    component.Description,
		"tags":          component.Tags,
		"author":        component.Author,
		"accessibility_score": component.AccessibilityScore,
		"usage_count":   component.UsageCount,
		"created_at":    component.CreatedAt.Unix(),
	}

	// Insert into Qdrant
	return s.insertVector(component.ID.String(), embedding, metadata)
}

// UpdateComponentIndex updates an existing component in the search index
func (s *SearchService) UpdateComponentIndex(component *models.Component) error {
	// Remove existing entry
	s.RemoveFromIndex(component.ID)
	
	// Re-index with updated data
	return s.IndexComponent(component)
}

// RemoveFromIndex removes a component from the search index
func (s *SearchService) RemoveFromIndex(componentID uuid.UUID) error {
	// Remove from Qdrant
	cmd := exec.Command("resource-qdrant", "delete", "--collection", s.collectionName, "--id", componentID.String())
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove from index: %w", err)
	}
	return nil
}

// semanticSearch performs semantic search using Qdrant
func (s *SearchService) semanticSearch(query string, limit int) ([]map[string]interface{}, error) {
	// Create embedding for search query
	queryEmbedding, err := s.createQueryEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Search in Qdrant using resource-qdrant CLI
	cmd := exec.Command(
		"resource-qdrant", "search",
		"--collection", s.collectionName,
		"--vector", queryEmbedding,
		"--limit", strconv.Itoa(limit),
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("qdrant search failed: %w", err)
	}

	// Parse Qdrant response
	var results []map[string]interface{}
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	return results, nil
}

// createComponentEmbedding creates a vector embedding for a component
func (s *SearchService) createComponentEmbedding(component *models.Component) (string, error) {
	// Combine component text for embedding
	text := fmt.Sprintf("%s %s %s %s",
		component.Name,
		component.Category,
		component.Description,
		strings.Join(component.Tags, " "),
	)

	return s.generateEmbedding(text)
}

// createQueryEmbedding creates a vector embedding for a search query
func (s *SearchService) createQueryEmbedding(query string) (string, error) {
	return s.generateEmbedding(query)
}

// generateEmbedding generates a vector embedding using available embedding service
func (s *SearchService) generateEmbedding(text string) (string, error) {
	// Try to use Qdrant's embedding generation first
	cmd := exec.Command("resource-qdrant", "embed", "--text", text, "--model", "nomic-embed-text")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to mock embedding for development
		return s.generateMockEmbedding(text), nil
	}

	return strings.TrimSpace(string(output)), nil
}

// generateMockEmbedding generates a mock embedding vector (for development/testing)
func (s *SearchService) generateMockEmbedding(text string) string {
	// Generate a simple mock embedding based on text hash
	hash := s.simpleHash(text)
	embedding := make([]float64, 768) // Standard embedding dimension
	
	for i := range embedding {
		embedding[i] = float64((hash*31+i)%100) / 100.0 - 0.5
	}

	// Convert to JSON array string
	embeddingBytes, _ := json.Marshal(embedding)
	return string(embeddingBytes)
}

// simpleHash generates a simple hash for mock embeddings
func (s *SearchService) simpleHash(text string) int {
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}
	if hash < 0 {
		hash = -hash
	}
	return hash
}

// insertVector inserts a vector into Qdrant
func (s *SearchService) insertVector(id string, embedding string, metadata map[string]interface{}) error {
	// Prepare payload
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id":      id,
				"vector":  embedding,
				"payload": metadata,
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Insert into Qdrant
	cmd := exec.Command("resource-qdrant", "upsert", "--collection", s.collectionName)
	cmd.Stdin = strings.NewReader(string(payloadBytes))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to insert vector: %w", err)
	}

	return nil
}

// convertSearchResults converts Qdrant search results to component objects
func (s *SearchService) convertSearchResults(results []map[string]interface{}, req models.ComponentSearchRequest) ([]models.Component, error) {
	var components []models.Component

	for _, result := range results {
		// Extract payload from Qdrant result
		payload, ok := result["payload"].(map[string]interface{})
		if !ok {
			continue
		}

		// Convert to component
		component, err := s.payloadToComponent(payload)
		if err != nil {
			continue // Skip invalid results
		}

		// Apply additional filters
		if s.matchesFilters(component, req) {
			components = append(components, *component)
		}
	}

	return components, nil
}

// payloadToComponent converts Qdrant payload to Component struct
func (s *SearchService) payloadToComponent(payload map[string]interface{}) (*models.Component, error) {
	component := &models.Component{}

	// Extract and convert fields
	if idStr, ok := payload["component_id"].(string); ok {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, err
		}
		component.ID = id
	}

	if name, ok := payload["name"].(string); ok {
		component.Name = name
	}

	if category, ok := payload["category"].(string); ok {
		component.Category = category
	}

	if description, ok := payload["description"].(string); ok {
		component.Description = description
	}

	if author, ok := payload["author"].(string); ok {
		component.Author = author
	}

	if tags, ok := payload["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				component.Tags = append(component.Tags, tagStr)
			}
		}
	}

	if usageCount, ok := payload["usage_count"].(float64); ok {
		component.UsageCount = int(usageCount)
	}

	if accessibilityScore, ok := payload["accessibility_score"].(float64); ok {
		component.AccessibilityScore = &accessibilityScore
	}

	return component, nil
}

// matchesFilters checks if a component matches the search filters
func (s *SearchService) matchesFilters(component *models.Component, req models.ComponentSearchRequest) bool {
	// Category filter
	if req.Category != "" && component.Category != req.Category {
		return false
	}

	// Tags filter
	if len(req.Tags) > 0 {
		hasMatchingTag := false
		for _, reqTag := range req.Tags {
			for _, componentTag := range component.Tags {
				if strings.EqualFold(reqTag, componentTag) {
					hasMatchingTag = true
					break
				}
			}
			if hasMatchingTag {
				break
			}
		}
		if !hasMatchingTag {
			return false
		}
	}

	// Accessibility score filter
	if req.MinAccessibilityScore > 0 {
		if component.AccessibilityScore == nil || *component.AccessibilityScore < req.MinAccessibilityScore {
			return false
		}
	}

	return true
}

// EnsureCollection ensures the Qdrant collection exists for components
func (s *SearchService) EnsureCollection() error {
	// Check if collection exists
	cmd := exec.Command("resource-qdrant", "collection-info", s.collectionName)
	if err := cmd.Run(); err != nil {
		// Collection doesn't exist, create it
		createCmd := exec.Command(
			"resource-qdrant", "create-collection",
			"--name", s.collectionName,
			"--dimension", "768", // Standard embedding dimension
			"--distance", "Cosine",
		)
		if err := createCmd.Run(); err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
	}
	return nil
}

// BulkIndexComponents indexes multiple components at once
func (s *SearchService) BulkIndexComponents(components []models.Component) error {
	var points []map[string]interface{}

	for _, component := range components {
		embedding, err := s.createComponentEmbedding(&component)
		if err != nil {
			continue // Skip components that can't be embedded
		}

		metadata := map[string]interface{}{
			"component_id":       component.ID.String(),
			"name":              component.Name,
			"category":          component.Category,
			"description":       component.Description,
			"tags":             component.Tags,
			"author":           component.Author,
			"accessibility_score": component.AccessibilityScore,
			"usage_count":      component.UsageCount,
			"created_at":       component.CreatedAt.Unix(),
		}

		point := map[string]interface{}{
			"id":      component.ID.String(),
			"vector":  embedding,
			"payload": metadata,
		}

		points = append(points, point)
	}

	// Bulk insert
	payload := map[string]interface{}{
		"points": points,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal bulk payload: %w", err)
	}

	cmd := exec.Command("resource-qdrant", "upsert", "--collection", s.collectionName)
	cmd.Stdin = strings.NewReader(string(payloadBytes))
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to bulk insert vectors: %w", err)
	}

	return nil
}

// GetSimilarComponents finds components similar to a given component
func (s *SearchService) GetSimilarComponents(componentID uuid.UUID, limit int) ([]models.Component, error) {
	// Get the component's vector from Qdrant
	cmd := exec.Command("resource-qdrant", "get", "--collection", s.collectionName, "--id", componentID.String())
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get component vector: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse component data: %w", err)
	}

	vector, ok := result["vector"].(string)
	if !ok {
		return nil, fmt.Errorf("component vector not found")
	}

	// Search for similar components
	searchCmd := exec.Command(
		"resource-qdrant", "search",
		"--collection", s.collectionName,
		"--vector", vector,
		"--limit", strconv.Itoa(limit+1), // +1 to exclude self
	)

	searchOutput, err := searchCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("similarity search failed: %w", err)
	}

	var searchResults []map[string]interface{}
	if err := json.Unmarshal(searchOutput, &searchResults); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	// Convert results and filter out the original component
	var components []models.Component
	for _, result := range searchResults {
		if payload, ok := result["payload"].(map[string]interface{}); ok {
			if id, ok := payload["component_id"].(string); ok {
				if id != componentID.String() { // Exclude self
					if component, err := s.payloadToComponent(payload); err == nil {
						components = append(components, *component)
						if len(components) >= limit {
							break
						}
					}
				}
			}
		}
	}

	return components, nil
}