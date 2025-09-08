package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type IdeaProcessor struct {
	db        *sql.DB
	ollamaURL string
	qdrantURL string
}

type GenerateIdeasRequest struct {
	CampaignID string `json:"campaign_id"`
	Prompt     string `json:"prompt"`
	Count      int    `json:"count"`
	UserID     string `json:"user_id,omitempty"`
}

type GeneratedIdea struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Category           string   `json:"category"`
	Tags               []string `json:"tags"`
	ImplementationNotes string   `json:"implementation_notes"`
}

type GenerateIdeasResponse struct {
	Success bool            `json:"success"`
	Ideas   []GeneratedIdea `json:"ideas"`
	Message string          `json:"message,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type SemanticSearchRequest struct {
	Query      string `json:"query"`
	CampaignID string `json:"campaign_id,omitempty"`
	Limit      int    `json:"limit"`
}

type SemanticSearchResult struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Content  string  `json:"content"`
	Score    float64 `json:"score"`
	Category string  `json:"category"`
	Tags     []string `json:"tags"`
}

type DocumentProcessingRequest struct {
	DocumentID string `json:"document_id"`
	CampaignID string `json:"campaign_id"`
	FilePath   string `json:"file_path"`
}

type RefinementRequest struct {
	IdeaID     string `json:"idea_id"`
	Refinement string `json:"refinement"`
	UserID     string `json:"user_id,omitempty"`
}

func NewIdeaProcessor(db *sql.DB) *IdeaProcessor {
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = "http://localhost:6333"
	}
	
	return &IdeaProcessor{
		db:        db,
		ollamaURL: ollamaURL,
		qdrantURL: qdrantURL,
	}
}

// GenerateIdeas generates new ideas for a campaign (replaces n8n idea-generation-workflow)
func (ip *IdeaProcessor) GenerateIdeas(ctx context.Context, req GenerateIdeasRequest) GenerateIdeasResponse {
	response := GenerateIdeasResponse{
		Ideas: []GeneratedIdea{},
	}
	
	// Get campaign data
	campaign, err := ip.getCampaignData(ctx, req.CampaignID)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to get campaign data: %v", err)
		return response
	}
	
	// Get campaign documents
	documents, err := ip.getCampaignDocuments(ctx, req.CampaignID)
	if err != nil {
		// Continue without documents - they're optional
		documents = []map[string]interface{}{}
	}
	
	// Get existing ideas to avoid duplication
	existingIdeas, err := ip.getRecentIdeas(ctx, req.CampaignID)
	if err != nil {
		existingIdeas = []map[string]interface{}{}
	}
	
	// Build enriched prompt
	enrichedPrompt := ip.buildEnrichedPrompt(campaign, documents, existingIdeas, req)
	
	// Generate ideas using Ollama
	generatedIdeas, err := ip.generateWithOllama(ctx, enrichedPrompt, req.Count)
	if err != nil {
		response.Success = false
		response.Error = fmt.Sprintf("Failed to generate ideas: %v", err)
		return response
	}
	
	// Store generated ideas in database
	for _, idea := range generatedIdeas {
		ideaID := uuid.New().String()
		err := ip.storeIdea(ctx, ideaID, req.CampaignID, idea, req.UserID)
		if err != nil {
			fmt.Printf("Failed to store idea: %v\n", err)
			continue
		}
		
		// Store in vector database for semantic search
		ip.storeInVectorDB(ctx, ideaID, req.CampaignID, idea)
	}
	
	response.Success = true
	response.Ideas = generatedIdeas
	response.Message = fmt.Sprintf("Generated %d ideas successfully", len(generatedIdeas))
	
	return response
}

// SemanticSearch performs semantic search on ideas (replaces n8n semantic-search-workflow)
func (ip *IdeaProcessor) SemanticSearch(ctx context.Context, req SemanticSearchRequest) ([]SemanticSearchResult, error) {
	// Generate embedding for query
	embedding, err := ip.generateEmbedding(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	
	// Search in Qdrant
	searchPayload := map[string]interface{}{
		"vector": embedding,
		"limit":  req.Limit,
		"with_payload": true,
	}
	
	if req.CampaignID != "" {
		searchPayload["filter"] = map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "campaign_id",
					"match": map[string]interface{}{
						"value": req.CampaignID,
					},
				},
			},
		}
	}
	
	searchJSON, _ := json.Marshal(searchPayload)
	
	resp, err := http.Post(
		fmt.Sprintf("%s/collections/ideas/points/search", ip.qdrantURL),
		"application/json",
		bytes.NewBuffer(searchJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search Qdrant: %w", err)
	}
	defer resp.Body.Close()
	
	var searchResponse struct {
		Result []struct {
			ID      string                 `json:"id"`
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}
	
	// Convert to results
	results := []SemanticSearchResult{}
	for _, hit := range searchResponse.Result {
		result := SemanticSearchResult{
			ID:       hit.ID,
			Score:    hit.Score,
			Title:    getString(hit.Payload, "title"),
			Content:  getString(hit.Payload, "content"),
			Category: getString(hit.Payload, "category"),
		}
		
		if tags, ok := hit.Payload["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					result.Tags = append(result.Tags, tagStr)
				}
			}
		}
		
		results = append(results, result)
	}
	
	return results, nil
}

// RefineIdea refines an existing idea (replaces n8n chat-refinement-workflow)
func (ip *IdeaProcessor) RefineIdea(ctx context.Context, req RefinementRequest) error {
	// Get existing idea
	var idea struct {
		Title   string
		Content string
		Category string
	}
	
	query := `SELECT title, content, category FROM ideas WHERE id = $1`
	err := ip.db.QueryRowContext(ctx, query, req.IdeaID).Scan(&idea.Title, &idea.Content, &idea.Category)
	if err != nil {
		return fmt.Errorf("failed to get idea: %w", err)
	}
	
	// Build refinement prompt
	refinementPrompt := fmt.Sprintf(`
		Original Idea:
		Title: %s
		Category: %s
		Content: %s
		
		Refinement Request: %s
		
		Please provide an improved version of this idea that incorporates the refinement request.
		Maintain the core concept while enhancing based on the feedback.
		
		Return as JSON with format: {"title": "...", "content": "...", "category": "..."}
	`, idea.Title, idea.Category, idea.Content, req.Refinement)
	
	// Generate refined idea
	refinedJSON, err := ip.generateWithOllamaRaw(ctx, refinementPrompt)
	if err != nil {
		return fmt.Errorf("failed to generate refinement: %w", err)
	}
	
	var refined struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Category string `json:"category"`
	}
	
	if err := json.Unmarshal([]byte(refinedJSON), &refined); err != nil {
		return fmt.Errorf("failed to parse refined idea: %w", err)
	}
	
	// Update idea in database
	updateQuery := `
		UPDATE ideas 
		SET title = $1, content = $2, category = $3, status = 'refined', updated_at = $4
		WHERE id = $5`
	
	_, err = ip.db.ExecContext(ctx, updateQuery, 
		refined.Title, refined.Content, refined.Category, time.Now(), req.IdeaID)
	
	if err != nil {
		return fmt.Errorf("failed to update idea: %w", err)
	}
	
	// Update in vector database
	ip.updateVectorDB(ctx, req.IdeaID, refined.Title, refined.Content, refined.Category)
	
	return nil
}

// ProcessDocument processes a document for idea extraction (replaces n8n document-processing-pipeline)
func (ip *IdeaProcessor) ProcessDocument(ctx context.Context, req DocumentProcessingRequest) error {
	// This would integrate with document processing services
	// For now, we'll simulate the extraction
	
	// Update document status
	updateQuery := `
		UPDATE documents 
		SET processing_status = 'processing', updated_at = $1
		WHERE id = $2`
	
	_, err := ip.db.ExecContext(ctx, updateQuery, time.Now(), req.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to update document status: %w", err)
	}
	
	// Simulate document processing
	// In production, this would call Unstructured API or similar
	extractedText := "Extracted content from document..."
	
	// Update with extracted text
	completeQuery := `
		UPDATE documents 
		SET extracted_text = $1, processing_status = 'completed', updated_at = $2
		WHERE id = $3`
	
	_, err = ip.db.ExecContext(ctx, completeQuery, extractedText, time.Now(), req.DocumentID)
	if err != nil {
		return fmt.Errorf("failed to save extracted text: %w", err)
	}
	
	return nil
}

// Helper methods

func (ip *IdeaProcessor) getCampaignData(ctx context.Context, campaignID string) (map[string]interface{}, error) {
	var campaign struct {
		ID          string
		Name        string
		Description string
		Context     sql.NullString
	}
	
	query := `SELECT id, name, description, context FROM campaigns WHERE id = $1`
	err := ip.db.QueryRowContext(ctx, query, campaignID).Scan(
		&campaign.ID, &campaign.Name, &campaign.Description, &campaign.Context)
	
	if err != nil {
		return nil, err
	}
	
	result := map[string]interface{}{
		"id":          campaign.ID,
		"name":        campaign.Name,
		"description": campaign.Description,
	}
	
	if campaign.Context.Valid {
		result["context"] = campaign.Context.String
	}
	
	return result, nil
}

func (ip *IdeaProcessor) getCampaignDocuments(ctx context.Context, campaignID string) ([]map[string]interface{}, error) {
	query := `
		SELECT id, original_name, extracted_text 
		FROM documents 
		WHERE campaign_id = $1 AND processing_status = 'completed' 
		ORDER BY created_at DESC 
		LIMIT 5`
	
	rows, err := ip.db.QueryContext(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	documents := []map[string]interface{}{}
	for rows.Next() {
		var doc struct {
			ID           string
			OriginalName string
			ExtractedText sql.NullString
		}
		
		err := rows.Scan(&doc.ID, &doc.OriginalName, &doc.ExtractedText)
		if err != nil {
			continue
		}
		
		docMap := map[string]interface{}{
			"id":            doc.ID,
			"original_name": doc.OriginalName,
		}
		
		if doc.ExtractedText.Valid {
			docMap["extracted_text"] = doc.ExtractedText.String
		}
		
		documents = append(documents, docMap)
	}
	
	return documents, nil
}

func (ip *IdeaProcessor) getRecentIdeas(ctx context.Context, campaignID string) ([]map[string]interface{}, error) {
	query := `
		SELECT title, content, category 
		FROM ideas 
		WHERE campaign_id = $1 AND status IN ('refined', 'finalized') 
		ORDER BY created_at DESC 
		LIMIT 3`
	
	rows, err := ip.db.QueryContext(ctx, query, campaignID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	ideas := []map[string]interface{}{}
	for rows.Next() {
		var idea struct {
			Title    string
			Content  string
			Category string
		}
		
		err := rows.Scan(&idea.Title, &idea.Content, &idea.Category)
		if err != nil {
			continue
		}
		
		ideas = append(ideas, map[string]interface{}{
			"title":    idea.Title,
			"content":  idea.Content,
			"category": idea.Category,
		})
	}
	
	return ideas, nil
}

func (ip *IdeaProcessor) buildEnrichedPrompt(campaign, documents, existingIdeas map[string]interface{}, req GenerateIdeasRequest) string {
	var sb strings.Builder
	
	sb.WriteString("Campaign Context:\n")
	sb.WriteString(fmt.Sprintf("Name: %s\n", campaign["name"]))
	sb.WriteString(fmt.Sprintf("Description: %s\n", campaign["description"]))
	if context, ok := campaign["context"].(string); ok {
		sb.WriteString(fmt.Sprintf("Objective: %s\n", context))
	}
	sb.WriteString("\n")
	
	// Add document context if available
	if docs, ok := documents.([]map[string]interface{}); ok && len(docs) > 0 {
		sb.WriteString("Supporting Documents:\n")
		for _, doc := range docs {
			sb.WriteString(fmt.Sprintf("Document: %s\n", doc["original_name"]))
			if text, ok := doc["extracted_text"].(string); ok && text != "" {
				preview := text
				if len(preview) > 500 {
					preview = preview[:500] + "..."
				}
				sb.WriteString(fmt.Sprintf("Content Preview: %s\n", preview))
			}
			sb.WriteString("\n")
		}
	}
	
	// Add existing ideas to avoid duplication
	if ideas, ok := existingIdeas.([]map[string]interface{}); ok && len(ideas) > 0 {
		sb.WriteString("Recent Ideas for Reference (avoid duplication):\n")
		for _, idea := range ideas {
			sb.WriteString(fmt.Sprintf("Previous Idea: %s\n", idea["title"]))
			sb.WriteString(fmt.Sprintf("Category: %s\n", idea["category"]))
			if content, ok := idea["content"].(string); ok && content != "" {
				brief := content
				if len(brief) > 200 {
					brief = brief[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("Brief: %s\n", brief))
			}
			sb.WriteString("\n")
		}
	}
	
	// Add user request
	userPrompt := req.Prompt
	if userPrompt == "" {
		userPrompt = "Generate innovative ideas for this campaign"
	}
	sb.WriteString(fmt.Sprintf("User Request: %s\n\n", userPrompt))
	
	// Add generation instructions
	count := req.Count
	if count <= 0 {
		count = 1
	}
	
	sb.WriteString(fmt.Sprintf("Generate %d innovative, actionable ideas that:\n", count))
	sb.WriteString("1. Align with the campaign context and objectives\n")
	sb.WriteString("2. Leverage insights from available context\n")
	sb.WriteString("3. Are distinct from existing ideas\n")
	sb.WriteString("4. Include practical implementation considerations\n\n")
	sb.WriteString("Return as JSON array with format: [{\"title\": \"...\", \"description\": \"...\", \"category\": \"...\", \"tags\": [\"...\"], \"implementation_notes\": \"...\"}]\n")
	
	return sb.String()
}

func (ip *IdeaProcessor) generateWithOllama(ctx context.Context, prompt string, count int) ([]GeneratedIdea, error) {
	response, err := ip.generateWithOllamaRaw(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	// Parse the JSON response
	var ideas []GeneratedIdea
	if err := json.Unmarshal([]byte(response), &ideas); err != nil {
		// Try to extract JSON from the response if it's wrapped in text
		startIdx := strings.Index(response, "[")
		endIdx := strings.LastIndex(response, "]")
		if startIdx >= 0 && endIdx > startIdx {
			jsonStr := response[startIdx:endIdx+1]
			if err := json.Unmarshal([]byte(jsonStr), &ideas); err != nil {
				return nil, fmt.Errorf("failed to parse generated ideas: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse generated ideas: %w", err)
		}
	}
	
	// Ensure we have the requested number of ideas
	if len(ideas) > count {
		ideas = ideas[:count]
	}
	
	return ideas, nil
}

func (ip *IdeaProcessor) generateWithOllamaRaw(ctx context.Context, prompt string) (string, error) {
	payload := map[string]interface{}{
		"model":  "llama2",
		"prompt": prompt,
		"stream": false,
	}
	
	payloadJSON, _ := json.Marshal(payload)
	
	req, err := http.NewRequestWithContext(ctx, "POST", 
		fmt.Sprintf("%s/api/generate", ip.ollamaURL),
		bytes.NewBuffer(payloadJSON))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	
	var ollamaResp struct {
		Response string `json:"response"`
	}
	
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return "", err
	}
	
	return ollamaResp.Response, nil
}

func (ip *IdeaProcessor) generateEmbedding(ctx context.Context, text string) ([]float64, error) {
	payload := map[string]interface{}{
		"model":  "llama2",
		"prompt": text,
	}
	
	payloadJSON, _ := json.Marshal(payload)
	
	req, err := http.NewRequestWithContext(ctx, "POST",
		fmt.Sprintf("%s/api/embeddings", ip.ollamaURL),
		bytes.NewBuffer(payloadJSON))
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
	
	var embResp struct {
		Embedding []float64 `json:"embedding"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, err
	}
	
	return embResp.Embedding, nil
}

func (ip *IdeaProcessor) storeIdea(ctx context.Context, ideaID, campaignID string, idea GeneratedIdea, userID string) error {
	tagsJSON, _ := json.Marshal(idea.Tags)
	
	query := `
		INSERT INTO ideas (id, campaign_id, title, content, category, tags, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, 'generated', $7, $8)`
	
	content := idea.Description
	if idea.ImplementationNotes != "" {
		content += "\n\nImplementation Notes:\n" + idea.ImplementationNotes
	}
	
	_, err := ip.db.ExecContext(ctx, query,
		ideaID, campaignID, idea.Title, content, idea.Category, tagsJSON, time.Now(), time.Now())
	
	return err
}

func (ip *IdeaProcessor) storeInVectorDB(ctx context.Context, ideaID, campaignID string, idea GeneratedIdea) {
	// Generate embedding for the idea
	combinedText := fmt.Sprintf("%s %s %s", idea.Title, idea.Description, strings.Join(idea.Tags, " "))
	embedding, err := ip.generateEmbedding(ctx, combinedText)
	if err != nil {
		fmt.Printf("Failed to generate embedding for idea %s: %v\n", ideaID, err)
		return
	}
	
	// Store in Qdrant
	point := map[string]interface{}{
		"id": ideaID,
		"vector": embedding,
		"payload": map[string]interface{}{
			"campaign_id": campaignID,
			"title":       idea.Title,
			"content":     idea.Description,
			"category":    idea.Category,
			"tags":        idea.Tags,
		},
	}
	
	points := map[string]interface{}{
		"points": []interface{}{point},
	}
	
	pointsJSON, _ := json.Marshal(points)
	
	http.Post(
		fmt.Sprintf("%s/collections/ideas/points", ip.qdrantURL),
		"application/json",
		bytes.NewBuffer(pointsJSON),
	)
}

func (ip *IdeaProcessor) updateVectorDB(ctx context.Context, ideaID, title, content, category string) {
	// Similar to storeInVectorDB but updates existing point
	embedding, err := ip.generateEmbedding(ctx, title+" "+content)
	if err != nil {
		return
	}
	
	payload := map[string]interface{}{
		"points": []map[string]interface{}{
			{
				"id": ideaID,
				"vector": embedding,
				"payload": map[string]interface{}{
					"title":    title,
					"content":  content,
					"category": category,
				},
			},
		},
	}
	
	payloadJSON, _ := json.Marshal(payload)
	
	req, _ := http.NewRequest("PUT",
		fmt.Sprintf("%s/collections/ideas/points", ip.qdrantURL),
		bytes.NewBuffer(payloadJSON))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	client.Do(req)
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}