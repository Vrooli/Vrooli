package main

import (
	"encoding/json"
	"testing"
	"time"
)

// TestCampaignSerialization tests Campaign JSON serialization
func TestCampaignSerialization(t *testing.T) {
	t.Run("MarshalCampaign", func(t *testing.T) {
		now := time.Now()
		desc := "Test description"
		parentID := "parent-123"
		campaign := Campaign{
			ID:          "test-123",
			Name:        "Test Campaign",
			Description: &desc,
			Color:       "#ff0000",
			Icon:        "folder",
			ParentID:    &parentID,
			SortOrder:   1,
			IsFavorite:  true,
			PromptCount: 5,
			LastUsed:    &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		data, err := json.Marshal(campaign)
		if err != nil {
			t.Fatalf("Failed to marshal campaign: %v", err)
		}

		var unmarshaled Campaign
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal campaign: %v", err)
		}

		if unmarshaled.ID != campaign.ID {
			t.Errorf("Expected ID %s, got %s", campaign.ID, unmarshaled.ID)
		}
		if unmarshaled.Name != campaign.Name {
			t.Errorf("Expected Name %s, got %s", campaign.Name, unmarshaled.Name)
		}
		if *unmarshaled.Description != *campaign.Description {
			t.Errorf("Expected Description %s, got %s", *campaign.Description, *unmarshaled.Description)
		}
		if unmarshaled.PromptCount != campaign.PromptCount {
			t.Errorf("Expected PromptCount %d, got %d", campaign.PromptCount, unmarshaled.PromptCount)
		}
	})

	t.Run("MarshalCampaignWithNullFields", func(t *testing.T) {
		campaign := Campaign{
			ID:          "test-456",
			Name:        "Minimal Campaign",
			Description: nil,
			Color:       "#00ff00",
			Icon:        "folder",
			ParentID:    nil,
			SortOrder:   0,
			IsFavorite:  false,
			PromptCount: 0,
			LastUsed:    nil,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		data, err := json.Marshal(campaign)
		if err != nil {
			t.Fatalf("Failed to marshal campaign with null fields: %v", err)
		}

		var unmarshaled Campaign
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal campaign: %v", err)
		}

		if unmarshaled.Description != nil {
			t.Errorf("Expected nil Description, got %v", unmarshaled.Description)
		}
		if unmarshaled.ParentID != nil {
			t.Errorf("Expected nil ParentID, got %v", unmarshaled.ParentID)
		}
		if unmarshaled.LastUsed != nil {
			t.Errorf("Expected nil LastUsed, got %v", unmarshaled.LastUsed)
		}
	})
}

// TestPromptSerialization tests Prompt JSON serialization
func TestPromptSerialization(t *testing.T) {
	t.Run("MarshalPrompt", func(t *testing.T) {
		now := time.Now()
		desc := "Test prompt description"
		quickKey := "quick1"
		parentVersion := "parent-123"
		wordCount := 10
		tokens := 8
		rating := 5
		notes := "Test notes"
		campaignName := "Test Campaign"

		prompt := Prompt{
			ID:                  "prompt-123",
			CampaignID:          "campaign-123",
			Title:               "Test Prompt",
			Content:             "This is a test prompt",
			Description:         &desc,
			Variables:           []string{"var1", "var2"},
			UsageCount:          3,
			LastUsed:            &now,
			IsFavorite:          true,
			IsArchived:          false,
			QuickAccessKey:      &quickKey,
			Version:             1,
			ParentVersionID:     &parentVersion,
			WordCount:           &wordCount,
			EstimatedTokens:     &tokens,
			EffectivenessRating: &rating,
			Notes:               &notes,
			CreatedAt:           now,
			UpdatedAt:           now,
			CampaignName:        &campaignName,
			Tags:                []string{"tag1", "tag2"},
		}

		data, err := json.Marshal(prompt)
		if err != nil {
			t.Fatalf("Failed to marshal prompt: %v", err)
		}

		var unmarshaled Prompt
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal prompt: %v", err)
		}

		if unmarshaled.ID != prompt.ID {
			t.Errorf("Expected ID %s, got %s", prompt.ID, unmarshaled.ID)
		}
		if unmarshaled.Title != prompt.Title {
			t.Errorf("Expected Title %s, got %s", prompt.Title, unmarshaled.Title)
		}
		if len(unmarshaled.Variables) != len(prompt.Variables) {
			t.Errorf("Expected %d variables, got %d", len(prompt.Variables), len(unmarshaled.Variables))
		}
		if *unmarshaled.WordCount != *prompt.WordCount {
			t.Errorf("Expected WordCount %d, got %d", *prompt.WordCount, *unmarshaled.WordCount)
		}
	})

	t.Run("MarshalPromptMinimal", func(t *testing.T) {
		prompt := Prompt{
			ID:         "prompt-456",
			CampaignID: "campaign-456",
			Title:      "Minimal Prompt",
			Content:    "Minimal content",
			Variables:  []string{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		data, err := json.Marshal(prompt)
		if err != nil {
			t.Fatalf("Failed to marshal minimal prompt: %v", err)
		}

		var unmarshaled Prompt
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal prompt: %v", err)
		}

		if unmarshaled.Description != nil {
			t.Errorf("Expected nil Description, got %v", unmarshaled.Description)
		}
		if unmarshaled.QuickAccessKey != nil {
			t.Errorf("Expected nil QuickAccessKey, got %v", unmarshaled.QuickAccessKey)
		}
		if len(unmarshaled.Variables) != 0 {
			t.Errorf("Expected 0 variables, got %d", len(unmarshaled.Variables))
		}
	})
}

// TestTagSerialization tests Tag JSON serialization
func TestTagSerialization(t *testing.T) {
	t.Run("MarshalTag", func(t *testing.T) {
		color := "#ff0000"
		desc := "Test tag description"
		tag := Tag{
			ID:          "tag-123",
			Name:        "test-tag",
			Color:       &color,
			Description: &desc,
		}

		data, err := json.Marshal(tag)
		if err != nil {
			t.Fatalf("Failed to marshal tag: %v", err)
		}

		var unmarshaled Tag
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal tag: %v", err)
		}

		if unmarshaled.ID != tag.ID {
			t.Errorf("Expected ID %s, got %s", tag.ID, unmarshaled.ID)
		}
		if unmarshaled.Name != tag.Name {
			t.Errorf("Expected Name %s, got %s", tag.Name, unmarshaled.Name)
		}
		if *unmarshaled.Color != *tag.Color {
			t.Errorf("Expected Color %s, got %s", *tag.Color, *unmarshaled.Color)
		}
	})

	t.Run("MarshalTagMinimal", func(t *testing.T) {
		tag := Tag{
			ID:   "tag-456",
			Name: "minimal-tag",
		}

		data, err := json.Marshal(tag)
		if err != nil {
			t.Fatalf("Failed to marshal minimal tag: %v", err)
		}

		var unmarshaled Tag
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal tag: %v", err)
		}

		if unmarshaled.Color != nil {
			t.Errorf("Expected nil Color, got %v", unmarshaled.Color)
		}
		if unmarshaled.Description != nil {
			t.Errorf("Expected nil Description, got %v", unmarshaled.Description)
		}
	})
}

// TestTemplateSerialization tests Template JSON serialization
func TestTemplateSerialization(t *testing.T) {
	t.Run("MarshalTemplate", func(t *testing.T) {
		desc := "Template description"
		category := "testing"
		template := Template{
			ID:          "template-123",
			Name:        "Test Template",
			Description: &desc,
			Content:     "Template content with {{var}}",
			Variables:   []string{"var"},
			Category:    &category,
			UsageCount:  5,
			CreatedAt:   time.Now(),
		}

		data, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("Failed to marshal template: %v", err)
		}

		var unmarshaled Template
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal template: %v", err)
		}

		if unmarshaled.ID != template.ID {
			t.Errorf("Expected ID %s, got %s", template.ID, unmarshaled.ID)
		}
		if unmarshaled.Name != template.Name {
			t.Errorf("Expected Name %s, got %s", template.Name, unmarshaled.Name)
		}
		if len(unmarshaled.Variables) != len(template.Variables) {
			t.Errorf("Expected %d variables, got %d", len(template.Variables), len(unmarshaled.Variables))
		}
	})
}

// TestTestResultSerialization tests TestResult JSON serialization
func TestTestResultSerialization(t *testing.T) {
	t.Run("MarshalTestResult", func(t *testing.T) {
		inputVars := `{"var1": "value1"}`
		response := "Test response"
		responseTime := 1.23
		tokenCount := 150
		rating := 4
		notes := "Good result"

		result := TestResult{
			ID:           "result-123",
			PromptID:     "prompt-123",
			Model:        "gpt-4",
			InputVars:    &inputVars,
			Response:     &response,
			ResponseTime: &responseTime,
			TokenCount:   &tokenCount,
			Rating:       &rating,
			Notes:        &notes,
			TestedAt:     time.Now(),
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal test result: %v", err)
		}

		var unmarshaled TestResult
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal test result: %v", err)
		}

		if unmarshaled.ID != result.ID {
			t.Errorf("Expected ID %s, got %s", result.ID, unmarshaled.ID)
		}
		if unmarshaled.Model != result.Model {
			t.Errorf("Expected Model %s, got %s", result.Model, unmarshaled.Model)
		}
		if *unmarshaled.ResponseTime != *result.ResponseTime {
			t.Errorf("Expected ResponseTime %f, got %f", *result.ResponseTime, *unmarshaled.ResponseTime)
		}
	})
}

// TestRequestTypeSerialization tests request/response type serialization
func TestRequestTypeSerialization(t *testing.T) {
	t.Run("CreatePromptRequest", func(t *testing.T) {
		desc := "Test description"
		req := CreatePromptRequest{
			CampaignID:  "campaign-123",
			Title:       "Test Prompt",
			Content:     "Test content",
			Description: &desc,
			Variables:   []string{"var1", "var2"},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		var unmarshaled CreatePromptRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.CampaignID != req.CampaignID {
			t.Errorf("Expected CampaignID %s, got %s", req.CampaignID, unmarshaled.CampaignID)
		}
		if unmarshaled.Title != req.Title {
			t.Errorf("Expected Title %s, got %s", req.Title, unmarshaled.Title)
		}
		if len(unmarshaled.Variables) != len(req.Variables) {
			t.Errorf("Expected %d variables, got %d", len(req.Variables), len(unmarshaled.Variables))
		}
	})
}

// TestJSONEdgeCases tests edge cases in JSON handling
func TestJSONEdgeCases(t *testing.T) {
	t.Run("EmptyArrays", func(t *testing.T) {
		prompt := Prompt{
			ID:         "test-1",
			CampaignID: "campaign-1",
			Title:      "Test",
			Content:    "Content",
			Variables:  []string{},
			Tags:       []string{},
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		data, err := json.Marshal(prompt)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled Prompt
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Variables == nil {
			t.Error("Expected empty array, got nil for Variables")
		}
		if len(unmarshaled.Variables) != 0 {
			t.Errorf("Expected 0 variables, got %d", len(unmarshaled.Variables))
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		campaign := Campaign{
			ID:        "test-2",
			Name:      "Test <>&\" Campaign",
			Color:     "#ff0000",
			Icon:      "🚀",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		data, err := json.Marshal(campaign)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var unmarshaled Campaign
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Name != campaign.Name {
			t.Errorf("Expected Name %s, got %s", campaign.Name, unmarshaled.Name)
		}
		if unmarshaled.Icon != campaign.Icon {
			t.Errorf("Expected Icon %s, got %s", campaign.Icon, unmarshaled.Icon)
		}
	})
}
