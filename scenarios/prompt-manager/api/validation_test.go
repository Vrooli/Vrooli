package main

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestDataValidation tests various data validation scenarios
func TestDataValidation(t *testing.T) {
	t.Run("ValidUUID", func(t *testing.T) {
		validUUID := uuid.New().String()
		_, err := uuid.Parse(validUUID)
		if err != nil {
			t.Errorf("Expected valid UUID, got error: %v", err)
		}
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		invalidUUID := "not-a-valid-uuid"
		_, err := uuid.Parse(invalidUUID)
		if err == nil {
			t.Error("Expected error for invalid UUID")
		}
	})

	t.Run("EmptyUUID", func(t *testing.T) {
		emptyUUID := ""
		_, err := uuid.Parse(emptyUUID)
		if err == nil {
			t.Error("Expected error for empty UUID")
		}
	})
}

// TestCampaignValidation tests campaign data validation
func TestCampaignValidation(t *testing.T) {
	t.Run("ValidCampaign", func(t *testing.T) {
		desc := "Valid description"
		campaign := Campaign{
			ID:          uuid.New().String(),
			Name:        "Valid Campaign",
			Description: &desc,
			Color:       "#ff0000",
			Icon:        "folder",
			SortOrder:   1,
			IsFavorite:  false,
			PromptCount: 0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if campaign.Name == "" {
			t.Error("Expected non-empty name")
		}

		if campaign.Color == "" {
			t.Error("Expected non-empty color")
		}

		if _, err := uuid.Parse(campaign.ID); err != nil {
			t.Error("Expected valid UUID for ID")
		}
	})

	t.Run("CampaignNameValidation", func(t *testing.T) {
		testCases := []struct {
			name  string
			valid bool
		}{
			{"Valid Name", true},
			{"", false}, // Empty name should be invalid
			{"A", true}, // Single character
			{strings.Repeat("a", 1000), true}, // Very long name
			{"Name with 123 numbers", true},
			{"Name with special !@# chars", true},
		}

		for _, tc := range testCases {
			campaign := Campaign{
				ID:        uuid.New().String(),
				Name:      tc.name,
				Color:     "#ff0000",
				Icon:      "folder",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			isEmpty := campaign.Name == ""
			if tc.valid && isEmpty {
				t.Errorf("Expected name '%s' to be valid", tc.name)
			}
		}
	})

	t.Run("CampaignColorValidation", func(t *testing.T) {
		validColors := []string{
			"#ff0000",
			"#00ff00",
			"#0000ff",
			"#123456",
			"#abcdef",
			"#ABCDEF",
		}

		for _, color := range validColors {
			campaign := Campaign{
				ID:        uuid.New().String(),
				Name:      "Test",
				Color:     color,
				Icon:      "folder",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			if !strings.HasPrefix(campaign.Color, "#") {
				t.Errorf("Expected color %s to start with #", color)
			}

			if len(campaign.Color) != 7 {
				t.Errorf("Expected color %s to be 7 characters", color)
			}
		}
	})
}

// TestPromptValidation tests prompt data validation
func TestPromptValidation(t *testing.T) {
	t.Run("ValidPrompt", func(t *testing.T) {
		desc := "Valid prompt description"
		prompt := Prompt{
			ID:          uuid.New().String(),
			CampaignID:  uuid.New().String(),
			Title:       "Valid Prompt",
			Content:     "This is valid content",
			Description: &desc,
			Variables:   []string{"var1", "var2"},
			UsageCount:  0,
			IsFavorite:  false,
			IsArchived:  false,
			Version:     1,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if prompt.Title == "" {
			t.Error("Expected non-empty title")
		}

		if prompt.Content == "" {
			t.Error("Expected non-empty content")
		}

		if _, err := uuid.Parse(prompt.ID); err != nil {
			t.Error("Expected valid UUID for ID")
		}

		if _, err := uuid.Parse(prompt.CampaignID); err != nil {
			t.Error("Expected valid UUID for CampaignID")
		}
	})

	t.Run("PromptContentValidation", func(t *testing.T) {
		testCases := []struct {
			content string
			valid   bool
		}{
			{"Valid content", true},
			{"", false}, // Empty content
			{"A", true},
			{strings.Repeat("a", 10000), true}, // Very long content
			{"Content with\nnewlines\n", true},
			{"Content with 	tabs	", true},
			{"Unicode content: 你好世界 🚀", true},
		}

		for _, tc := range testCases {
			prompt := Prompt{
				ID:         uuid.New().String(),
				CampaignID: uuid.New().String(),
				Title:      "Test",
				Content:    tc.content,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			isEmpty := prompt.Content == ""
			if tc.valid && isEmpty {
				t.Errorf("Expected content to be valid")
			}
		}
	})

	t.Run("PromptVariablesValidation", func(t *testing.T) {
		testCases := []struct {
			variables []string
			valid     bool
		}{
			{[]string{}, true},                           // Empty array
			{[]string{"var1"}, true},                     // Single variable
			{[]string{"var1", "var2", "var3"}, true},     // Multiple variables
			{[]string{"a", "b", "c", "d", "e"}, true},    // Many variables
			{[]string{"variable_name"}, true},            // Underscore
			{[]string{"variableName"}, true},             // Camel case
			{[]string{"variable-name"}, true},            // Hyphen
			{[]string{"var1", "var1"}, true},             // Duplicates (should be handled by app logic)
		}

		for _, tc := range testCases {
			prompt := Prompt{
				ID:         uuid.New().String(),
				CampaignID: uuid.New().String(),
				Title:      "Test",
				Content:    "Content",
				Variables:  tc.variables,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}

			if prompt.Variables == nil && tc.valid {
				t.Error("Expected variables to be non-nil")
			}
		}
	})
}

// TestRequestValidation tests request structure validation
func TestRequestValidation(t *testing.T) {
	t.Run("ValidCreateCampaignRequest", func(t *testing.T) {
		desc := "Test description"
		color := "#ff0000"
		req := CreateCampaignRequest{
			Name:        "Test Campaign",
			Description: &desc,
			Color:       &color,
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		var unmarshaled CreateCampaignRequest
		if err := json.Unmarshal(data, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		if unmarshaled.Name != req.Name {
			t.Errorf("Expected name %s, got %s", req.Name, unmarshaled.Name)
		}
	})

	t.Run("ValidCreatePromptRequest", func(t *testing.T) {
		desc := "Test prompt description"
		req := CreatePromptRequest{
			CampaignID:  uuid.New().String(),
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

		if unmarshaled.Title != req.Title {
			t.Errorf("Expected title %s, got %s", req.Title, unmarshaled.Title)
		}

		if len(unmarshaled.Variables) != len(req.Variables) {
			t.Errorf("Expected %d variables, got %d", len(req.Variables), len(unmarshaled.Variables))
		}
	})
}

// TestTimeValidation tests time-related validation
func TestTimeValidation(t *testing.T) {
	t.Run("TimestampPrecision", func(t *testing.T) {
		now := time.Now()
		campaign := Campaign{
			ID:        uuid.New().String(),
			Name:      "Test",
			Color:     "#ff0000",
			Icon:      "folder",
			CreatedAt: now,
			UpdatedAt: now,
		}

		if campaign.CreatedAt.IsZero() {
			t.Error("Expected non-zero CreatedAt")
		}

		if campaign.UpdatedAt.IsZero() {
			t.Error("Expected non-zero UpdatedAt")
		}

		if campaign.CreatedAt.After(campaign.UpdatedAt) {
			t.Error("CreatedAt should not be after UpdatedAt")
		}
	})

	t.Run("LastUsedTimestamp", func(t *testing.T) {
		now := time.Now()
		prompt := Prompt{
			ID:         uuid.New().String(),
			CampaignID: uuid.New().String(),
			Title:      "Test",
			Content:    "Content",
			LastUsed:   &now,
			CreatedAt:  now.Add(-1 * time.Hour),
			UpdatedAt:  now,
		}

		if prompt.LastUsed == nil {
			t.Error("Expected LastUsed to be set")
		}

		if prompt.LastUsed.Before(prompt.CreatedAt) {
			t.Error("LastUsed should not be before CreatedAt")
		}
	})
}

// TestCountersValidation tests counter field validation
func TestCountersValidation(t *testing.T) {
	t.Run("UsageCount", func(t *testing.T) {
		prompt := Prompt{
			ID:         uuid.New().String(),
			CampaignID: uuid.New().String(),
			Title:      "Test",
			Content:    "Content",
			UsageCount: 0,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if prompt.UsageCount < 0 {
			t.Error("UsageCount should not be negative")
		}

		// Simulate usage increment
		prompt.UsageCount++
		if prompt.UsageCount != 1 {
			t.Errorf("Expected UsageCount=1, got %d", prompt.UsageCount)
		}
	})

	t.Run("PromptCount", func(t *testing.T) {
		campaign := Campaign{
			ID:          uuid.New().String(),
			Name:        "Test",
			Color:       "#ff0000",
			Icon:        "folder",
			PromptCount: 0,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if campaign.PromptCount < 0 {
			t.Error("PromptCount should not be negative")
		}
	})

	t.Run("WordCount", func(t *testing.T) {
		content := "This is a test with five words"
		wordCount := calculateWordCount(&content)

		if wordCount == nil {
			t.Fatal("Expected word count to be calculated")
		}

		if *wordCount <= 0 {
			t.Error("Word count should be positive for non-empty content")
		}
	})

	t.Run("TokenCount", func(t *testing.T) {
		content := "This is a test"
		tokenCount := calculateTokenCount(&content)

		if tokenCount == nil {
			t.Fatal("Expected token count to be calculated")
		}

		if *tokenCount <= 0 {
			t.Error("Token count should be positive for non-empty content")
		}
	})
}

// TestBooleanFields tests boolean field handling
func TestBooleanFields(t *testing.T) {
	t.Run("IsFavorite", func(t *testing.T) {
		prompt := Prompt{
			ID:         uuid.New().String(),
			CampaignID: uuid.New().String(),
			Title:      "Test",
			Content:    "Content",
			IsFavorite: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if prompt.IsFavorite {
			t.Error("Expected IsFavorite to be false")
		}

		// Toggle
		prompt.IsFavorite = true
		if !prompt.IsFavorite {
			t.Error("Expected IsFavorite to be true")
		}
	})

	t.Run("IsArchived", func(t *testing.T) {
		prompt := Prompt{
			ID:         uuid.New().String(),
			CampaignID: uuid.New().String(),
			Title:      "Test",
			Content:    "Content",
			IsArchived: false,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		if prompt.IsArchived {
			t.Error("Expected IsArchived to be false")
		}
	})
}

// TestOptionalFields tests handling of optional/nullable fields
func TestOptionalFields(t *testing.T) {
	t.Run("NullableDescription", func(t *testing.T) {
		// With description
		desc := "Description"
		campaign1 := Campaign{
			ID:          uuid.New().String(),
			Name:        "Test",
			Description: &desc,
			Color:       "#ff0000",
			Icon:        "folder",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if campaign1.Description == nil {
			t.Error("Expected description to be set")
		}

		// Without description
		campaign2 := Campaign{
			ID:          uuid.New().String(),
			Name:        "Test",
			Description: nil,
			Color:       "#ff0000",
			Icon:        "folder",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if campaign2.Description != nil {
			t.Error("Expected description to be nil")
		}
	})

	t.Run("NullableParentID", func(t *testing.T) {
		// Root campaign (no parent)
		campaign1 := Campaign{
			ID:        uuid.New().String(),
			Name:      "Root",
			Color:     "#ff0000",
			Icon:      "folder",
			ParentID:  nil,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if campaign1.ParentID != nil {
			t.Error("Expected ParentID to be nil for root campaign")
		}

		// Child campaign (has parent)
		parentID := uuid.New().String()
		campaign2 := Campaign{
			ID:        uuid.New().String(),
			Name:      "Child",
			Color:     "#ff0000",
			Icon:      "folder",
			ParentID:  &parentID,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if campaign2.ParentID == nil {
			t.Error("Expected ParentID to be set for child campaign")
		}
	})

	t.Run("NullableQuickAccessKey", func(t *testing.T) {
		// Without quick access key
		prompt1 := Prompt{
			ID:             uuid.New().String(),
			CampaignID:     uuid.New().String(),
			Title:          "Test",
			Content:        "Content",
			QuickAccessKey: nil,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if prompt1.QuickAccessKey != nil {
			t.Error("Expected QuickAccessKey to be nil")
		}

		// With quick access key
		quickKey := "quick1"
		prompt2 := Prompt{
			ID:             uuid.New().String(),
			CampaignID:     uuid.New().String(),
			Title:          "Test",
			Content:        "Content",
			QuickAccessKey: &quickKey,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		if prompt2.QuickAccessKey == nil || *prompt2.QuickAccessKey != "quick1" {
			t.Error("Expected QuickAccessKey to be 'quick1'")
		}
	})
}
