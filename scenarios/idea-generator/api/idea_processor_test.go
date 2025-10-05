// +build testing

package main

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestNewIdeaProcessor tests IdeaProcessor initialization
func TestNewIdeaProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("DefaultURLs", func(t *testing.T) {
		// Clear env vars to test defaults
		oldOllama := os.Getenv("OLLAMA_URL")
		oldQdrant := os.Getenv("QDRANT_URL")
		os.Unsetenv("OLLAMA_URL")
		os.Unsetenv("QDRANT_URL")

		defer func() {
			if oldOllama != "" {
				os.Setenv("OLLAMA_URL", oldOllama)
			}
			if oldQdrant != "" {
				os.Setenv("QDRANT_URL", oldQdrant)
			}
		}()

		processor := NewIdeaProcessor(env.DB)

		if processor.ollamaURL != "http://localhost:11434" {
			t.Errorf("Expected default Ollama URL, got %s", processor.ollamaURL)
		}
		if processor.qdrantURL != "http://localhost:6333" {
			t.Errorf("Expected default Qdrant URL, got %s", processor.qdrantURL)
		}
	})

	t.Run("CustomURLs", func(t *testing.T) {
		os.Setenv("OLLAMA_URL", "http://custom-ollama:8080")
		os.Setenv("QDRANT_URL", "http://custom-qdrant:9090")

		defer func() {
			os.Unsetenv("OLLAMA_URL")
			os.Unsetenv("QDRANT_URL")
		}()

		processor := NewIdeaProcessor(env.DB)

		if processor.ollamaURL != "http://custom-ollama:8080" {
			t.Errorf("Expected custom Ollama URL, got %s", processor.ollamaURL)
		}
		if processor.qdrantURL != "http://custom-qdrant:9090" {
			t.Errorf("Expected custom Qdrant URL, got %s", processor.qdrantURL)
		}
	})
}

// TestGetCampaignData tests campaign data retrieval
func TestGetCampaignData(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)

	t.Run("ValidCampaign", func(t *testing.T) {
		campaign := createTestCampaign(t, env.DB, "data-test")
		defer campaign.Cleanup()

		ctx := context.Background()
		data, err := processor.getCampaignData(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get campaign data: %v", err)
		}

		if data["id"] != campaign.ID {
			t.Errorf("Expected campaign ID %s, got %v", campaign.ID, data["id"])
		}
		if data["name"] != campaign.Name {
			t.Errorf("Expected campaign name %s, got %v", campaign.Name, data["name"])
		}
	})

	t.Run("NonExistentCampaign", func(t *testing.T) {
		ctx := context.Background()
		_, err := processor.getCampaignData(ctx, uuid.New().String())

		if err != sql.ErrNoRows && err == nil {
			t.Error("Expected error for non-existent campaign")
		}
	})

	t.Run("InvalidCampaignID", func(t *testing.T) {
		ctx := context.Background()
		_, err := processor.getCampaignData(ctx, "invalid-id")

		if err == nil {
			t.Error("Expected error for invalid campaign ID")
		}
	})
}

// TestGetCampaignDocuments tests document retrieval
func TestGetCampaignDocuments(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "docs-test")
	defer campaign.Cleanup()

	t.Run("NoDocuments", func(t *testing.T) {
		ctx := context.Background()
		docs, err := processor.getCampaignDocuments(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get documents: %v", err)
		}

		if len(docs) != 0 {
			t.Errorf("Expected 0 documents, got %d", len(docs))
		}
	})

	t.Run("WithDocuments", func(t *testing.T) {
		// Create test documents
		docID := uuid.New().String()
		query := `INSERT INTO documents (id, campaign_id, original_name, extracted_text, processing_status)
		          VALUES ($1, $2, $3, $4, $5)`
		_, err := env.DB.Exec(query, docID, campaign.ID, "test.pdf", "Extracted text content", "completed")
		if err != nil {
			t.Fatalf("Failed to create test document: %v", err)
		}

		ctx := context.Background()
		docs, err := processor.getCampaignDocuments(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get documents: %v", err)
		}

		if len(docs) != 1 {
			t.Errorf("Expected 1 document, got %d", len(docs))
		}

		if docs[0]["original_name"] != "test.pdf" {
			t.Errorf("Expected document name 'test.pdf', got %v", docs[0]["original_name"])
		}
	})

	t.Run("LimitToFive", func(t *testing.T) {
		// Create 10 documents
		for i := 0; i < 10; i++ {
			docID := uuid.New().String()
			query := `INSERT INTO documents (id, campaign_id, original_name, processing_status)
			          VALUES ($1, $2, $3, $4)`
			env.DB.Exec(query, docID, campaign.ID, "doc"+string(rune(i))+".pdf", "completed")
		}

		ctx := context.Background()
		docs, err := processor.getCampaignDocuments(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get documents: %v", err)
		}

		// Should limit to 5 documents
		if len(docs) > 5 {
			t.Errorf("Expected max 5 documents, got %d", len(docs))
		}
	})
}

// TestGetRecentIdeas tests recent ideas retrieval
func TestGetRecentIdeas(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "recent-ideas-test")
	defer campaign.Cleanup()

	t.Run("NoIdeas", func(t *testing.T) {
		ctx := context.Background()
		ideas, err := processor.getRecentIdeas(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get ideas: %v", err)
		}

		if len(ideas) != 0 {
			t.Errorf("Expected 0 ideas, got %d", len(ideas))
		}
	})

	t.Run("WithIdeas", func(t *testing.T) {
		// Create test ideas with refined status
		ideaID := uuid.New().String()
		query := `INSERT INTO ideas (id, campaign_id, title, content, category, status)
		          VALUES ($1, $2, $3, $4, $5, $6)`
		_, err := env.DB.Exec(query, ideaID, campaign.ID, "Test Idea", "Test content", "innovation", "refined")
		if err != nil {
			t.Fatalf("Failed to create test idea: %v", err)
		}

		ctx := context.Background()
		ideas, err := processor.getRecentIdeas(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get ideas: %v", err)
		}

		if len(ideas) != 1 {
			t.Errorf("Expected 1 idea, got %d", len(ideas))
		}
	})

	t.Run("LimitToThree", func(t *testing.T) {
		// Create 5 ideas with refined status
		for i := 0; i < 5; i++ {
			ideaID := uuid.New().String()
			query := `INSERT INTO ideas (id, campaign_id, title, content, category, status)
			          VALUES ($1, $2, $3, $4, $5, $6)`
			env.DB.Exec(query, ideaID, campaign.ID, "Idea "+string(rune(i)), "Content", "innovation", "finalized")
		}

		ctx := context.Background()
		ideas, err := processor.getRecentIdeas(ctx, campaign.ID)

		if err != nil {
			t.Fatalf("Failed to get ideas: %v", err)
		}

		// Should limit to 3 ideas
		if len(ideas) > 3 {
			t.Errorf("Expected max 3 ideas, got %d", len(ideas))
		}
	})
}

// TestBuildEnrichedPrompt tests prompt building
func TestBuildEnrichedPrompt(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)

	t.Run("BasicPrompt", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Test Campaign",
			"description": "Test Description",
		}
		documents := []map[string]interface{}{}
		existingIdeas := []map[string]interface{}{}
		req := GenerateIdeasRequest{
			Prompt: "Generate test ideas",
			Count:  1,
		}

		prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)

		if !strings.Contains(prompt, "Test Campaign") {
			t.Error("Prompt should contain campaign name")
		}
		if !strings.Contains(prompt, "Generate test ideas") {
			t.Error("Prompt should contain user request")
		}
	})

	t.Run("WithDocuments", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Test Campaign",
			"description": "Test Description",
		}
		documents := []map[string]interface{}{
			{
				"original_name":  "doc1.pdf",
				"extracted_text": "Document content here",
			},
		}
		existingIdeas := []map[string]interface{}{}
		req := GenerateIdeasRequest{
			Prompt: "Generate ideas",
			Count:  1,
		}

		prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)

		if !strings.Contains(prompt, "doc1.pdf") {
			t.Error("Prompt should contain document name")
		}
		if !strings.Contains(prompt, "Document content") {
			t.Error("Prompt should contain document content")
		}
	})

	t.Run("WithExistingIdeas", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Test Campaign",
			"description": "Test Description",
		}
		documents := []map[string]interface{}{}
		existingIdeas := []map[string]interface{}{
			{
				"title":    "Existing Idea",
				"content":  "Existing content",
				"category": "innovation",
			},
		}
		req := GenerateIdeasRequest{
			Prompt: "Generate ideas",
			Count:  1,
		}

		prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)

		if !strings.Contains(prompt, "Existing Idea") {
			t.Error("Prompt should contain existing idea")
		}
		if !strings.Contains(prompt, "avoid duplication") {
			t.Error("Prompt should mention avoiding duplication")
		}
	})

	t.Run("EmptyPrompt", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Test Campaign",
			"description": "Test Description",
		}
		documents := []map[string]interface{}{}
		existingIdeas := []map[string]interface{}{}
		req := GenerateIdeasRequest{
			Prompt: "",
			Count:  1,
		}

		prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)

		if !strings.Contains(prompt, "Generate innovative ideas") {
			t.Error("Should use default prompt when empty")
		}
	})

	t.Run("MultipleIdeas", func(t *testing.T) {
		campaign := map[string]interface{}{
			"name":        "Test Campaign",
			"description": "Test Description",
		}
		documents := []map[string]interface{}{}
		existingIdeas := []map[string]interface{}{}
		req := GenerateIdeasRequest{
			Prompt: "Generate ideas",
			Count:  5,
		}

		prompt := processor.buildEnrichedPrompt(campaign, documents, existingIdeas, req)

		if !strings.Contains(prompt, "Generate 5 innovative") {
			t.Error("Prompt should specify count")
		}
	})
}

// TestStoreIdea tests idea storage
func TestStoreIdea(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "store-test")
	defer campaign.Cleanup()

	t.Run("BasicIdea", func(t *testing.T) {
		ctx := context.Background()
		ideaID := uuid.New().String()
		idea := GeneratedIdea{
			Title:              "Test Idea",
			Description:        "Test Description",
			Category:           "innovation",
			Tags:               []string{"test", "demo"},
			ImplementationNotes: "Test notes",
		}

		err := processor.storeIdea(ctx, ideaID, campaign.ID, idea, "user-123")

		if err != nil {
			t.Fatalf("Failed to store idea: %v", err)
		}

		// Verify idea was stored
		var storedTitle string
		query := `SELECT title FROM ideas WHERE id = $1`
		err = env.DB.QueryRow(query, ideaID).Scan(&storedTitle)

		if err != nil {
			t.Fatalf("Failed to retrieve stored idea: %v", err)
		}

		if storedTitle != "Test Idea" {
			t.Errorf("Expected title 'Test Idea', got %s", storedTitle)
		}
	})

	t.Run("IdeaWithTags", func(t *testing.T) {
		ctx := context.Background()
		ideaID := uuid.New().String()
		idea := GeneratedIdea{
			Title:       "Tagged Idea",
			Description: "Description",
			Category:    "innovation",
			Tags:        []string{"tag1", "tag2", "tag3"},
		}

		err := processor.storeIdea(ctx, ideaID, campaign.ID, idea, "user-123")

		if err != nil {
			t.Fatalf("Failed to store idea with tags: %v", err)
		}
	})

	t.Run("IdeaWithoutTags", func(t *testing.T) {
		ctx := context.Background()
		ideaID := uuid.New().String()
		idea := GeneratedIdea{
			Title:       "Untagged Idea",
			Description: "Description",
			Category:    "innovation",
			Tags:        []string{},
		}

		err := processor.storeIdea(ctx, ideaID, campaign.ID, idea, "user-123")

		if err != nil {
			t.Fatalf("Failed to store idea without tags: %v", err)
		}
	})
}

// TestProcessDocument tests document processing
func TestProcessDocument(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "doc-process-test")
	defer campaign.Cleanup()

	t.Run("ProcessDocument", func(t *testing.T) {
		// Create a test document
		docID := uuid.New().String()
		query := `INSERT INTO documents (id, campaign_id, original_name, processing_status)
		          VALUES ($1, $2, $3, $4)`
		_, err := env.DB.Exec(query, docID, campaign.ID, "test.pdf", "pending")
		if err != nil {
			t.Fatalf("Failed to create test document: %v", err)
		}

		ctx := context.Background()
		req := DocumentProcessingRequest{
			DocumentID: docID,
			CampaignID: campaign.ID,
			FilePath:   "/tmp/test.pdf",
		}

		err = processor.ProcessDocument(ctx, req)

		if err != nil {
			t.Fatalf("Failed to process document: %v", err)
		}

		// Verify document status was updated
		var status string
		query = `SELECT processing_status FROM documents WHERE id = $1`
		err = env.DB.QueryRow(query, docID).Scan(&status)

		if err != nil {
			t.Fatalf("Failed to retrieve document status: %v", err)
		}

		if status != "completed" {
			t.Errorf("Expected status 'completed', got %s", status)
		}
	})

	t.Run("NonExistentDocument", func(t *testing.T) {
		ctx := context.Background()
		req := DocumentProcessingRequest{
			DocumentID: uuid.New().String(),
			CampaignID: campaign.ID,
			FilePath:   "/tmp/test.pdf",
		}

		err := processor.ProcessDocument(ctx, req)

		if err == nil {
			t.Error("Expected error for non-existent document")
		}
	})
}

// TestGetString tests the getString helper
func TestGetString(t *testing.T) {
	t.Run("ValidString", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		result := getString(m, "key")
		if result != "value" {
			t.Errorf("Expected 'value', got '%s'", result)
		}
	})

	t.Run("MissingKey", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		result := getString(m, "missing")
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("WrongType", func(t *testing.T) {
		m := map[string]interface{}{
			"key": 123,
		}
		result := getString(m, "key")
		if result != "" {
			t.Errorf("Expected empty string for wrong type, got '%s'", result)
		}
	})
}

// TestGenerateIdeasIntegration tests the full idea generation flow
func TestGenerateIdeasIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	processor := NewIdeaProcessor(env.DB)
	campaign := createTestCampaign(t, env.DB, "integration-test")
	defer campaign.Cleanup()

	t.Run("GenerateIdeas_WithoutExternalServices", func(t *testing.T) {
		ctx, cancel := createContextWithTimeout(5 * time.Second)
		defer cancel()

		req := GenerateIdeasRequest{
			CampaignID: campaign.ID,
			Prompt:     "Test prompt",
			Count:      1,
			UserID:     "test-user",
		}

		response := processor.GenerateIdeas(ctx, req)

		// Without real Ollama, this should fail gracefully
		if response.Success {
			t.Log("Idea generation succeeded (likely has access to real Ollama)")
		} else {
			t.Logf("Idea generation failed as expected without external services: %s", response.Error)
		}
	})
}
