package main

import (
	"testing"
)

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
		Status:   "healthy",
		Services: map[string]string{
			"postgres": "healthy",
			"redis":    "healthy",
		},
	}

	if hr.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", hr.Status)
	}

	if len(hr.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(hr.Services))
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
