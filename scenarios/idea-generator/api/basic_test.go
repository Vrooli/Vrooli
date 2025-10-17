package main

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestInputValidation tests input validation logic without database
func TestInputValidation(t *testing.T) {
	// Mock processor with nil DB for validation-only tests
	processor := &IdeaProcessor{
		db:        nil,
		ollamaURL: "http://localhost:11434",
		qdrantURL: "http://localhost:6333",
	}

	ctx := context.Background()

	t.Run("GenerateIdeas_MissingCampaignID", func(t *testing.T) {
		req := GenerateIdeasRequest{
			CampaignID: "",
			Prompt:     "test",
			Count:      1,
		}
		resp := processor.GenerateIdeas(ctx, req)
		if resp.Success {
			t.Error("Expected validation failure for empty campaign ID")
		}
		if !strings.Contains(resp.Error, "required") {
			t.Errorf("Expected required error, got: %s", resp.Error)
		}
	})

	t.Run("GenerateIdeas_CountTooLow", func(t *testing.T) {
		req := GenerateIdeasRequest{
			CampaignID: uuid.New().String(),
			Prompt:     "test",
			Count:      0,
		}
		resp := processor.GenerateIdeas(ctx, req)
		if resp.Success {
			t.Error("Expected validation failure for count 0")
		}
		if !strings.Contains(resp.Error, "between 1 and 10") {
			t.Errorf("Expected count range error, got: %s", resp.Error)
		}
	})

	t.Run("GenerateIdeas_CountTooHigh", func(t *testing.T) {
		req := GenerateIdeasRequest{
			CampaignID: uuid.New().String(),
			Prompt:     "test",
			Count:      11,
		}
		resp := processor.GenerateIdeas(ctx, req)
		if resp.Success {
			t.Error("Expected validation failure for count 11")
		}
		if !strings.Contains(resp.Error, "between 1 and 10") {
			t.Errorf("Expected count range error, got: %s", resp.Error)
		}
	})

	t.Run("SemanticSearch_EmptyQuery", func(t *testing.T) {
		req := SemanticSearchRequest{
			Query: "",
			Limit: 10,
		}
		_, err := processor.SemanticSearch(ctx, req)
		if err == nil {
			t.Error("Expected validation error for empty query")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected empty query error, got: %v", err)
		}
	})

	t.Run("RefineIdea_MissingID", func(t *testing.T) {
		req := RefinementRequest{
			IdeaID:     "",
			Refinement: "test",
		}
		err := processor.RefineIdea(ctx, req)
		if err == nil {
			t.Error("Expected validation error for missing ID")
		}
		if !strings.Contains(err.Error(), "required") {
			t.Errorf("Expected required error, got: %v", err)
		}
	})

	t.Run("RefineIdea_EmptyRefinement", func(t *testing.T) {
		req := RefinementRequest{
			IdeaID:     uuid.New().String(),
			Refinement: "",
		}
		err := processor.RefineIdea(ctx, req)
		if err == nil {
			t.Error("Expected validation error for empty refinement")
		}
		if !strings.Contains(err.Error(), "empty") {
			t.Errorf("Expected empty error, got: %v", err)
		}
	})

	t.Run("RefineIdea_TooLong", func(t *testing.T) {
		req := RefinementRequest{
			IdeaID:     uuid.New().String(),
			Refinement: strings.Repeat("a", 2001),
		}
		err := processor.RefineIdea(ctx, req)
		if err == nil {
			t.Error("Expected validation error for too long refinement")
		}
		if !strings.Contains(err.Error(), "too long") {
			t.Errorf("Expected too long error, got: %v", err)
		}
	})
}

// TestBasicFunctions tests basic utility functions without database
func TestBasicFunctions(t *testing.T) {
	t.Run("GetString", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		result := getString(m, "key")
		if result != "value" {
			t.Errorf("Expected 'value', got '%s'", result)
		}
	})

	t.Run("GetString_MissingKey", func(t *testing.T) {
		m := map[string]interface{}{
			"key": "value",
		}
		result := getString(m, "missing")
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})

	t.Run("GetString_WrongType", func(t *testing.T) {
		m := map[string]interface{}{
			"key": 123,
		}
		result := getString(m, "key")
		if result != "" {
			t.Errorf("Expected empty string for wrong type, got '%s'", result)
		}
	})
}

// TestStructDefinitions verifies struct definitions
func TestStructDefinitions(t *testing.T) {
	t.Run("Campaign", func(t *testing.T) {
		c := Campaign{
			ID:          "test-id",
			Name:        "Test Campaign",
			Description: "Test Description",
		}
		if c.ID != "test-id" {
			t.Errorf("Expected ID 'test-id', got '%s'", c.ID)
		}
	})

	t.Run("Idea", func(t *testing.T) {
		i := Idea{
			ID:         "idea-id",
			CampaignID: "campaign-id",
			Title:      "Test Idea",
		}
		if i.Title != "Test Idea" {
			t.Errorf("Expected title 'Test Idea', got '%s'", i.Title)
		}
	})

	t.Run("GeneratedIdea", func(t *testing.T) {
		gi := GeneratedIdea{
			Title:       "Generated",
			Description: "Desc",
			Category:    "innovation",
			Tags:        []string{"tag1"},
		}
		if len(gi.Tags) != 1 {
			t.Errorf("Expected 1 tag, got %d", len(gi.Tags))
		}
	})

	t.Run("GenerateIdeasRequest", func(t *testing.T) {
		req := GenerateIdeasRequest{
			CampaignID: "campaign-id",
			Prompt:     "test prompt",
			Count:      5,
		}
		if req.Count != 5 {
			t.Errorf("Expected count 5, got %d", req.Count)
		}
	})

	t.Run("SemanticSearchRequest", func(t *testing.T) {
		req := SemanticSearchRequest{
			Query:      "search query",
			CampaignID: "campaign-id",
			Limit:      10,
		}
		if req.Limit != 10 {
			t.Errorf("Expected limit 10, got %d", req.Limit)
		}
	})
}

// TestHealthResponse tests health response structure
func TestHealthResponse(t *testing.T) {
	hr := HealthResponse{
		Status:  "healthy",
		Service: "idea-generator",
		Dependencies: map[string]interface{}{
			"postgres": map[string]interface{}{"status": "healthy"},
			"redis":    map[string]interface{}{"status": "healthy"},
		},
	}

	if hr.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", hr.Status)
	}

	if len(hr.Dependencies) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(hr.Dependencies))
	}
}

// TestWorkflow tests workflow structure
func TestWorkflow(t *testing.T) {
	wf := Workflow{
		ID:          "workflow-id",
		Name:        "Test Workflow",
		Description: "Test Description",
		Status:      "active",
	}

	if wf.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", wf.Status)
	}
}
