package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

type MindMapProcessor struct {
	db        *sql.DB
	ollamaURL string
	qdrantURL string
}

type CreateMindMapRequest struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	UserID      string                 `json:"userId"`
	InitialNode *Node                  `json:"initialNode,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	IsPublic    bool                   `json:"isPublic,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type SemanticSearchRequest struct {
	Query      string `json:"query"`
	Collection string `json:"collection,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

type MindMapSearchResult struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Score       float64                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type OrganizeRequest struct {
	MindMapID string `json:"mind_map_id"`
	Method    string `json:"method"` // "basic", "enhanced", "advanced"
}

type DocumentToMindMapRequest struct {
	DocumentContent string                 `json:"document_content"`
	DocumentType    string                 `json:"document_type"` // "text", "markdown", "pdf"
	Title           string                 `json:"title"`
	UserID          string                 `json:"user_id"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func NewMindMapProcessor(db *sql.DB, ollamaURL, qdrantURL string) *MindMapProcessor {
	return &MindMapProcessor{
		db:        db,
		ollamaURL: ollamaURL,
		qdrantURL: qdrantURL,
	}
}

func (mp *MindMapProcessor) CreateMindMap(ctx context.Context, req CreateMindMapRequest) (*MindMap, error) {
	// Generate unique ID
	mapID := uuid.New().String()
	
	// Create initial node if not provided
	initialNode := req.InitialNode
	if initialNode == nil {
		initialNode = &Node{
			ID:        uuid.New().String(),
			MindMapID: mapID,
			Content:   req.Title,
			Type:      "root",
			PositionX: 0,
			PositionY: 0,
			Metadata:  make(map[string]interface{}),
		}
	}

	// Create mind map structure
	mindMapData := map[string]interface{}{
		"nodes": []Node{*initialNode},
		"edges": []interface{}{},
	}

	// Insert into database
	query := `
		INSERT INTO mind_maps (id, title, description, user_id, data, tags, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING *`

	tagsJSON, _ := json.Marshal(req.Tags)
	dataJSON, _ := json.Marshal(mindMapData)
	now := time.Now()

	var mindMap MindMap
	var createdAt, updatedAt time.Time
	var tagsStr, dataStr string

	err := mp.db.QueryRow(query, mapID, req.Title, req.Description, req.UserID,
		dataJSON, tagsJSON, req.IsPublic, now).Scan(
		&mindMap.ID, &mindMap.Title, &mindMap.Description, &mindMap.OwnerID,
		&dataStr, &tagsStr, &req.IsPublic, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mind map: %w", err)
	}

	mindMap.CreatedAt = createdAt.Format(time.RFC3339)
	mindMap.UpdatedAt = updatedAt.Format(time.RFC3339)

	// Generate and store embedding for semantic search
	go mp.generateAndStoreEmbedding(context.Background(), mapID, req.Title, req.Description, req.Tags)

	return &mindMap, nil
}

func (mp *MindMapProcessor) UpdateMindMap(ctx context.Context, mapID string, updates map[string]interface{}) error {
	// Build dynamic update query
	setParts := []string{}
	args := []interface{}{}
	argIndex := 1

	for field, value := range updates {
		switch field {
		case "title", "description", "data":
			setParts = append(setParts, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no valid fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	args = append(args, mapID)

	query := fmt.Sprintf(`
		UPDATE mind_maps 
		SET %s 
		WHERE id = $%d`,
		strings.Join(setParts, ", "), argIndex)

	_, err := mp.db.Exec(query, args...)
	return err
}

func (mp *MindMapProcessor) SemanticSearch(ctx context.Context, req SemanticSearchRequest) ([]MindMapSearchResult, error) {
	if req.Collection == "" {
		req.Collection = "mind_maps"
	}
	if req.Limit == 0 {
		req.Limit = 10
	}

	// Generate query embedding
	embedding, err := mp.generateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Search in Qdrant (simplified - in real implementation, use Qdrant client)
	results, err := mp.searchInQdrant(ctx, req.Collection, embedding, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search in Qdrant: %w", err)
	}

	return results, nil
}

func (mp *MindMapProcessor) AutoOrganize(ctx context.Context, req OrganizeRequest) error {
	// Get mind map data
	mindMap, err := mp.getMindMapByID(req.MindMapID)
	if err != nil {
		return fmt.Errorf("failed to get mind map: %w", err)
	}

	// Parse existing data
	var mapData map[string]interface{}
	if err := json.Unmarshal([]byte(mindMap.Metadata["data"].(string)), &mapData); err != nil {
		return fmt.Errorf("failed to parse mind map data: %w", err)
	}

	// Organize based on method
	switch req.Method {
	case "basic":
		return mp.organizeBasic(ctx, req.MindMapID, mapData)
	case "enhanced":
		return mp.organizeEnhanced(ctx, req.MindMapID, mapData)
	case "advanced":
		return mp.organizeAdvanced(ctx, req.MindMapID, mapData)
	default:
		return mp.organizeBasic(ctx, req.MindMapID, mapData)
	}
}

func (mp *MindMapProcessor) DocumentToMindMap(ctx context.Context, req DocumentToMindMapRequest) (*MindMap, error) {
	// Process document with AI to extract structure
	prompt := fmt.Sprintf(`Convert this document into a mind map structure. Extract key concepts, relationships, and hierarchies.

Document Title: %s
Document Type: %s
Content:
%s

Create a JSON structure with nodes and connections suitable for a mind map. Return only the JSON with this format:
{
  "title": "Mind Map Title",
  "nodes": [
    {
      "id": "unique_id",
      "content": "Node content",
      "type": "root|branch|leaf",
      "position_x": 0,
      "position_y": 0,
      "parent_id": null,
      "metadata": {}
    }
  ],
  "edges": [
    {
      "from": "node_id",
      "to": "node_id",
      "relationship": "describes|contains|relates_to"
    }
  ]
}`, req.Title, req.DocumentType, req.DocumentContent)

	// Generate mind map structure with AI
	response, err := mp.callOllamaGenerate(ctx, prompt, "llama3.2", "analysis")
	if err != nil {
		return nil, fmt.Errorf("failed to generate mind map structure: %w", err)
	}

	// Parse AI response
	mindMapStructure, err := mp.parseAIMindMapResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	// Create mind map in database
	mapID := uuid.New().String()
	query := `
		INSERT INTO mind_maps (id, title, description, user_id, data, tags, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
		RETURNING *`

	dataJSON, _ := json.Marshal(mindMapStructure)
	now := time.Now()
	description := fmt.Sprintf("Generated from %s document", req.DocumentType)

	var mindMap MindMap
	var createdAt, updatedAt time.Time
	var tagsStr, dataStr string

	err = mp.db.QueryRow(query, mapID, mindMapStructure["title"], description, req.UserID,
		dataJSON, "[]", false, now).Scan(
		&mindMap.ID, &mindMap.Title, &mindMap.Description, &mindMap.OwnerID,
		&dataStr, &tagsStr, &req.IsPublic, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create mind map from document: %w", err)
	}

	mindMap.CreatedAt = createdAt.Format(time.RFC3339)
	mindMap.UpdatedAt = updatedAt.Format(time.RFC3339)

	return &mindMap, nil
}

func (mp *MindMapProcessor) ExportMindMap(ctx context.Context, mapID string, format string) (interface{}, error) {
	mindMap, err := mp.getMindMapByID(mapID)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return mindMap, nil
	case "markdown":
		return mp.exportToMarkdown(mindMap)
	case "svg":
		return mp.exportToSVG(mindMap)
	case "pdf":
		return mp.exportToPDF(mindMap)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// Helper functions

func (mp *MindMapProcessor) callOllamaGenerate(ctx context.Context, prompt, model, taskType string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "/vrooli/cli/vrooli", "resource", "ollama", "generate", prompt, "--model", model, "--type", taskType, "--quiet")
	
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("ollama command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

func (mp *MindMapProcessor) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	cmd := exec.CommandContext(ctx, "bash", "/vrooli/cli/vrooli", "resource", "ollama", "embed", text, "--model", "nomic-embed-text", "--quiet")
	
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Parse embedding result (this would need to be adapted based on actual output format)
	var embedding []float64
	if err := json.Unmarshal(stdout.Bytes(), &embedding); err != nil {
		// Fallback parsing if needed
		return nil, fmt.Errorf("failed to parse embedding: %w", err)
	}

	return embedding, nil
}

func (mp *MindMapProcessor) generateAndStoreEmbedding(ctx context.Context, mapID, title, description string, tags []string) {
	// Combine text for embedding
	text := fmt.Sprintf("%s %s %s", title, description, strings.Join(tags, " "))
	
	embedding, err := mp.generateEmbedding(ctx, text)
	if err != nil {
		// Log error but don't fail the operation
		return
	}

	// Store in Qdrant (simplified implementation)
	mp.storeInQdrant(ctx, "mind_maps", mapID, embedding, map[string]interface{}{
		"map_id":      mapID,
		"title":       title,
		"description": description,
		"tags":        tags,
		"created_at":  time.Now().Unix(),
	})
}

func (mp *MindMapProcessor) searchInQdrant(ctx context.Context, collection string, embedding []float64, limit int) ([]MindMapSearchResult, error) {
	// This would be implemented using Qdrant client
	// For now, return empty results
	return []MindMapSearchResult{}, nil
}

func (mp *MindMapProcessor) storeInQdrant(ctx context.Context, collection, pointID string, embedding []float64, payload map[string]interface{}) error {
	// This would be implemented using Qdrant client
	return nil
}

func (mp *MindMapProcessor) getMindMapByID(mapID string) (*MindMap, error) {
	query := `SELECT id, title, description, user_id, data, tags, is_public, created_at, updated_at FROM mind_maps WHERE id = $1`
	
	var mindMap MindMap
	var createdAt, updatedAt time.Time
	var dataStr, tagsStr string
	var isPublic bool

	err := mp.db.QueryRow(query, mapID).Scan(
		&mindMap.ID, &mindMap.Title, &mindMap.Description, &mindMap.OwnerID,
		&dataStr, &tagsStr, &isPublic, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	mindMap.CreatedAt = createdAt.Format(time.RFC3339)
	mindMap.UpdatedAt = updatedAt.Format(time.RFC3339)
	
	// Parse metadata
	mindMap.Metadata = map[string]interface{}{
		"data":      dataStr,
		"tags":      tagsStr,
		"is_public": isPublic,
	}

	return &mindMap, nil
}

func (mp *MindMapProcessor) parseAIMindMapResponse(response string) (map[string]interface{}, error) {
	// Clean response and extract JSON
	cleanResponse := strings.TrimSpace(response)
	cleanResponse = regexp.MustCompile("```json\\n?|```\\n?").ReplaceAllString(cleanResponse, "")
	
	// Try to find JSON object
	jsonMatch := regexp.MustCompile(`\{[\s\S]*\}`).FindString(cleanResponse)
	if jsonMatch == "" {
		return nil, fmt.Errorf("no JSON object found in AI response")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(jsonMatch), &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Validate required fields
	if result["title"] == nil || result["nodes"] == nil {
		return nil, fmt.Errorf("invalid mind map structure: missing title or nodes")
	}

	return result, nil
}

func (mp *MindMapProcessor) organizeBasic(ctx context.Context, mapID string, mapData map[string]interface{}) error {
	// Basic organization: arrange nodes in a simple tree structure
	// This would implement basic node positioning algorithms
	return mp.updateNodePositions(mapID, mapData, "basic")
}

func (mp *MindMapProcessor) organizeEnhanced(ctx context.Context, mapID string, mapData map[string]interface{}) error {
	// Enhanced organization with AI-powered grouping
	prompt := "Analyze this mind map structure and suggest optimal node groupings and positioning for better visual organization."
	// Implementation would use AI to suggest better organization
	return mp.updateNodePositions(mapID, mapData, "enhanced")
}

func (mp *MindMapProcessor) organizeAdvanced(ctx context.Context, mapID string, mapData map[string]interface{}) error {
	// Advanced organization with semantic analysis and relationship detection
	return mp.updateNodePositions(mapID, mapData, "advanced")
}

func (mp *MindMapProcessor) updateNodePositions(mapID string, mapData map[string]interface{}, method string) error {
	// Update the mind map with new node positions
	dataJSON, _ := json.Marshal(mapData)
	
	query := `UPDATE mind_maps SET data = $1, updated_at = $2 WHERE id = $3`
	_, err := mp.db.Exec(query, dataJSON, time.Now(), mapID)
	return err
}

func (mp *MindMapProcessor) exportToMarkdown(mindMap *MindMap) (string, error) {
	// Convert mind map to markdown format
	markdown := fmt.Sprintf("# %s\n\n%s\n\n", mindMap.Title, mindMap.Description)
	// Implementation would traverse nodes and create markdown structure
	return markdown, nil
}

func (mp *MindMapProcessor) exportToSVG(mindMap *MindMap) (string, error) {
	// Generate SVG representation
	return "<svg><!-- Mind map SVG --></svg>", nil
}

func (mp *MindMapProcessor) exportToPDF(mindMap *MindMap) ([]byte, error) {
	// Generate PDF representation
	return []byte("PDF content"), nil
}