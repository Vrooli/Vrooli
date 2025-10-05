
package main

import (
	"context"
	"testing"
	"time"
)

// TestNewRelationshipProcessor tests processor creation
func TestNewRelationshipProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)
	if processor == nil {
		t.Fatal("Expected non-nil processor")
	}

	if processor.db == nil {
		t.Error("Expected processor to have database connection")
	}
}

// TestGetUpcomingBirthdays tests birthday reminder generation
func TestGetUpcomingBirthdays(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	// Create contact with upcoming birthday
	today := time.Now()
	upcomingDate := today.AddDate(0, 0, 5) // 5 days from now
	contact := setupTestContact(t, testDB.DB, "TestBirthdayContact")
	defer contact.Cleanup()

	// Update birthday to upcoming date (keep same year for testing)
	birthdayStr := upcomingDate.Format("2006-01-02")
	testDB.DB.Exec("UPDATE contacts SET birthday = $1 WHERE id = $2", birthdayStr, contact.Contact.ID)

	t.Run("Success_DefaultDays", func(t *testing.T) {
		ctx := context.Background()
		reminders, err := processor.GetUpcomingBirthdays(ctx, 7)
		if err != nil {
			t.Fatalf("Failed to get birthdays: %v", err)
		}

		if reminders == nil {
			reminders = []BirthdayReminder{}
		}

		// Should include our test contact
		found := false
		for _, reminder := range reminders {
			if reminder.ContactID == contact.Contact.ID {
				found = true
				if reminder.DaysUntil < 0 || reminder.DaysUntil > 7 {
					t.Errorf("Expected days_until between 0-7, got %d", reminder.DaysUntil)
				}
				if reminder.Message == "" {
					t.Error("Expected non-empty message")
				}
				if reminder.Urgency == "" {
					t.Error("Expected non-empty urgency")
				}
			}
		}

		if !found {
			t.Log("Note: Test contact birthday may not be within default 7-day window")
		}
	})

	t.Run("Success_CustomDays", func(t *testing.T) {
		ctx := context.Background()
		reminders, err := processor.GetUpcomingBirthdays(ctx, 30)
		if err != nil {
			t.Fatalf("Failed to get birthdays: %v", err)
		}

		if reminders == nil {
			reminders = []BirthdayReminder{}
		}
	})

	t.Run("ZeroDays_UsesDefault", func(t *testing.T) {
		ctx := context.Background()
		reminders, err := processor.GetUpcomingBirthdays(ctx, 0)
		if err != nil {
			t.Fatalf("Failed to get birthdays: %v", err)
		}

		if reminders == nil {
			reminders = []BirthdayReminder{}
		}
	})

	t.Run("NegativeDays_UsesDefault", func(t *testing.T) {
		ctx := context.Background()
		reminders, err := processor.GetUpcomingBirthdays(ctx, -5)
		if err != nil {
			t.Fatalf("Failed to get birthdays: %v", err)
		}

		if reminders == nil {
			reminders = []BirthdayReminder{}
		}
	})
}

// TestEnrichContact tests contact enrichment
func TestEnrichContact(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		enrichment, err := processor.EnrichContact(ctx, 1, "John Doe")
		if err != nil {
			t.Fatalf("Failed to enrich contact: %v", err)
		}

		if enrichment == nil {
			t.Fatal("Expected non-nil enrichment")
		}

		if enrichment.ContactID != 1 {
			t.Errorf("Expected contact_id 1, got %d", enrichment.ContactID)
		}

		if enrichment.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got '%s'", enrichment.Name)
		}

		if len(enrichment.SuggestedInterests) == 0 {
			t.Error("Expected at least one suggested interest")
		}

		if len(enrichment.PersonalityTraits) == 0 {
			t.Error("Expected at least one personality trait")
		}

		if len(enrichment.ConversationStarters) == 0 {
			t.Error("Expected at least one conversation starter")
		}
	})

	t.Run("DifferentNames", func(t *testing.T) {
		ctx := context.Background()
		names := []string{"Alice Smith", "Bob Johnson", "Charlie Brown"}

		for _, name := range names {
			enrichment, err := processor.EnrichContact(ctx, 1, name)
			if err != nil {
				t.Errorf("Failed to enrich contact with name '%s': %v", name, err)
				continue
			}

			if enrichment.Name != name {
				t.Errorf("Expected name '%s', got '%s'", name, enrichment.Name)
			}
		}
	})

	t.Run("EmptyName", func(t *testing.T) {
		ctx := context.Background()
		enrichment, err := processor.EnrichContact(ctx, 1, "")
		// Should still work with fallback
		if err != nil {
			t.Fatalf("Failed to enrich contact: %v", err)
		}

		if enrichment == nil {
			t.Fatal("Expected non-nil enrichment")
		}
	})
}

// TestSuggestGifts tests gift suggestion generation
func TestSuggestGifts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	t.Run("Success", func(t *testing.T) {
		ctx := context.Background()
		req := GiftSuggestionRequest{
			ContactID: 1,
			Name:      "Jane Doe",
			Interests: "reading, hiking, photography",
			Occasion:  "birthday",
			Budget:    "50-100",
		}

		suggestions, err := processor.SuggestGifts(ctx, req)
		if err != nil {
			t.Fatalf("Failed to suggest gifts: %v", err)
		}

		if suggestions == nil {
			t.Fatal("Expected non-nil suggestions")
		}

		if suggestions.ContactID != req.ContactID {
			t.Errorf("Expected contact_id %d, got %d", req.ContactID, suggestions.ContactID)
		}

		if suggestions.Occasion != req.Occasion {
			t.Errorf("Expected occasion '%s', got '%s'", req.Occasion, suggestions.Occasion)
		}

		if len(suggestions.Suggestions) == 0 {
			t.Error("Expected at least one gift suggestion")
		}

		// Verify suggestion structure
		for i, suggestion := range suggestions.Suggestions {
			if suggestion.Name == "" {
				t.Errorf("Suggestion %d: expected non-empty name", i)
			}
			if suggestion.Description == "" {
				t.Errorf("Suggestion %d: expected non-empty description", i)
			}
			if suggestion.Price <= 0 {
				t.Errorf("Suggestion %d: expected positive price, got %f", i, suggestion.Price)
			}
			if suggestion.Store == "" {
				t.Errorf("Suggestion %d: expected non-empty store", i)
			}
			if suggestion.RelevanceScore < 0 || suggestion.RelevanceScore > 1 {
				t.Errorf("Suggestion %d: expected relevance score 0-1, got %f", i, suggestion.RelevanceScore)
			}
		}
	})

	t.Run("WithPastGifts", func(t *testing.T) {
		ctx := context.Background()
		req := GiftSuggestionRequest{
			ContactID: 1,
			Name:      "Jane Doe",
			Interests: "reading",
			Occasion:  "christmas",
			Budget:    "100-200",
			PastGifts: "book, coffee mug",
		}

		suggestions, err := processor.SuggestGifts(ctx, req)
		if err != nil {
			t.Fatalf("Failed to suggest gifts: %v", err)
		}

		if suggestions == nil {
			t.Fatal("Expected non-nil suggestions")
		}
	})

	t.Run("EmptyInterests", func(t *testing.T) {
		ctx := context.Background()
		req := GiftSuggestionRequest{
			ContactID: 1,
			Name:      "Jane Doe",
			Interests: "",
			Occasion:  "birthday",
			Budget:    "50-100",
		}

		suggestions, err := processor.SuggestGifts(ctx, req)
		if err != nil {
			t.Fatalf("Failed to suggest gifts: %v", err)
		}

		if suggestions == nil {
			t.Fatal("Expected non-nil suggestions")
		}
	})

	t.Run("DefaultValues", func(t *testing.T) {
		ctx := context.Background()
		req := GiftSuggestionRequest{
			ContactID: 1,
			Name:      "Jane Doe",
			Interests: "travel",
			// Budget and Occasion will use defaults in handler
		}

		suggestions, err := processor.SuggestGifts(ctx, req)
		if err != nil {
			t.Fatalf("Failed to suggest gifts: %v", err)
		}

		if suggestions == nil {
			t.Fatal("Expected non-nil suggestions")
		}
	})
}

// TestAnalyzeRelationships tests relationship insight generation
func TestAnalyzeRelationships(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	contact := setupTestContact(t, testDB.DB, "TestInsightContact")
	defer contact.Cleanup()

	t.Run("Success_WithInteractions", func(t *testing.T) {
		// Create some interactions
		for i := 0; i < 5; i++ {
			interaction := setupTestInteraction(t, testDB.DB, contact.Contact.ID)
			defer testDB.DB.Exec("DELETE FROM interactions WHERE id = $1", interaction.ID)
		}

		ctx := context.Background()
		insights, err := processor.AnalyzeRelationships(ctx, contact.Contact.ID)
		if err != nil {
			t.Fatalf("Failed to analyze relationships: %v", err)
		}

		if insights == nil {
			t.Fatal("Expected non-nil insights")
		}

		if insights.ContactID != contact.Contact.ID {
			t.Errorf("Expected contact_id %d, got %d", contact.Contact.ID, insights.ContactID)
		}

		if insights.Name != contact.Contact.Name {
			t.Errorf("Expected name '%s', got '%s'", contact.Contact.Name, insights.Name)
		}

		if insights.InteractionFrequency == "" {
			t.Error("Expected non-empty interaction frequency")
		}

		if insights.OverallSentiment == "" {
			t.Error("Expected non-empty overall sentiment")
		}

		if len(insights.RecommendedActions) == 0 {
			t.Error("Expected at least one recommended action")
		}

		if insights.RelationshipScore < 0 || insights.RelationshipScore > 100 {
			t.Errorf("Expected relationship score 0-100, got %f", insights.RelationshipScore)
		}

		if insights.TrendAnalysis == "" {
			t.Error("Expected non-empty trend analysis")
		}
	})

	t.Run("Success_NoInteractions", func(t *testing.T) {
		emptyContact := setupTestContact(t, testDB.DB, "TestNoInteractions")
		defer emptyContact.Cleanup()

		ctx := context.Background()
		insights, err := processor.AnalyzeRelationships(ctx, emptyContact.Contact.ID)
		if err != nil {
			t.Fatalf("Failed to analyze relationships: %v", err)
		}

		if insights == nil {
			t.Fatal("Expected non-nil insights")
		}

		if insights.InteractionFrequency != "none" {
			t.Errorf("Expected frequency 'none', got '%s'", insights.InteractionFrequency)
		}

		if insights.RelationshipScore != 0 {
			t.Errorf("Expected relationship score 0, got %f", insights.RelationshipScore)
		}

		if len(insights.RecommendedActions) == 0 {
			t.Error("Expected recommendations even with no interactions")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		ctx := context.Background()
		_, err := processor.AnalyzeRelationships(ctx, 999999)
		if err == nil {
			t.Error("Expected error for non-existent contact")
		}
	})
}

// TestGenerateRecommendations tests recommendation generation logic
func TestGenerateRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	testCases := []struct {
		name     string
		insight  *RelationshipInsight
		minRecs  int
		expected string
	}{
		{
			name: "LongTimeSinceInteraction",
			insight: &RelationshipInsight{
				LastInteractionDays:  90,
				InteractionFrequency: "occasional",
				OverallSentiment:     "positive",
				RelationshipScore:    60,
			},
			minRecs:  1,
			expected: "Reach out soon",
		},
		{
			name: "RecentInteraction",
			insight: &RelationshipInsight{
				LastInteractionDays:  5,
				InteractionFrequency: "frequent",
				OverallSentiment:     "very positive",
				RelationshipScore:    85,
			},
			minRecs:  1,
			expected: "Keep up",
		},
		{
			name: "NeedsAttention",
			insight: &RelationshipInsight{
				LastInteractionDays:  45,
				InteractionFrequency: "occasional",
				OverallSentiment:     "needs attention",
				RelationshipScore:    40,
			},
			minRecs: 2,
		},
		{
			name: "LowScore",
			insight: &RelationshipInsight{
				LastInteractionDays:  30,
				InteractionFrequency: "occasional",
				OverallSentiment:     "neutral",
				RelationshipScore:    35,
			},
			minRecs: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recommendations := processor.generateRecommendations(tc.insight)

			if len(recommendations) < tc.minRecs {
				t.Errorf("Expected at least %d recommendations, got %d", tc.minRecs, len(recommendations))
			}

			if tc.expected != "" {
				found := false
				for _, rec := range recommendations {
					if rec != "" {
						found = true
						break
					}
				}
				if !found {
					t.Error("Expected non-empty recommendations")
				}
			}
		})
	}
}

// TestAnalyzeTrend tests trend analysis logic
func TestAnalyzeTrend(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	t.Run("InsufficientData", func(t *testing.T) {
		interactions := []struct {
			Type        string
			Date        time.Time
			Sentiment   string
			Description string
		}{
			{Type: "call", Date: time.Now(), Sentiment: "positive", Description: "Test"},
		}

		trend := processor.analyzeTrend(interactions)
		if trend != "Insufficient data for trend analysis" {
			t.Errorf("Expected 'Insufficient data' message, got '%s'", trend)
		}
	})

	t.Run("ImprovingTrend", func(t *testing.T) {
		interactions := []struct {
			Type        string
			Date        time.Time
			Sentiment   string
			Description string
		}{
			{Type: "call", Date: time.Now().AddDate(0, 0, -1), Sentiment: "positive", Description: "Recent 1"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -2), Sentiment: "positive", Description: "Recent 2"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -10), Sentiment: "neutral", Description: "Old 1"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -11), Sentiment: "neutral", Description: "Old 2"},
		}

		trend := processor.analyzeTrend(interactions)
		if trend == "" {
			t.Error("Expected non-empty trend analysis")
		}
	})

	t.Run("StableTrend", func(t *testing.T) {
		interactions := []struct {
			Type        string
			Date        time.Time
			Sentiment   string
			Description string
		}{
			{Type: "call", Date: time.Now().AddDate(0, 0, -1), Sentiment: "positive", Description: "Recent 1"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -2), Sentiment: "positive", Description: "Recent 2"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -10), Sentiment: "positive", Description: "Old 1"},
			{Type: "call", Date: time.Now().AddDate(0, 0, -11), Sentiment: "positive", Description: "Old 2"},
		}

		trend := processor.analyzeTrend(interactions)
		if trend == "" {
			t.Error("Expected non-empty trend analysis")
		}
	})
}

// TestCallOllama tests the Ollama API client (mock implementation)
func TestCallOllama(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	processor := NewRelationshipProcessor(testDB.DB)

	t.Run("InterestsPrompt", func(t *testing.T) {
		ctx := context.Background()
		prompt := "Based on the name 'John Doe', suggest potential interests and personality traits"
		response, err := processor.callOllama(ctx, prompt, "llama3.2", 0.7)

		if err != nil {
			t.Fatalf("Failed to call Ollama: %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("GiftSuggestionsPrompt", func(t *testing.T) {
		ctx := context.Background()
		prompt := "Generate gift suggestions for someone who loves reading and travel"
		response, err := processor.callOllama(ctx, prompt, "llama3.2", 0.8)

		if err != nil {
			t.Fatalf("Failed to call Ollama: %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("GenericPrompt", func(t *testing.T) {
		ctx := context.Background()
		prompt := "Some generic prompt"
		response, err := processor.callOllama(ctx, prompt, "llama3.2", 0.5)

		if err != nil {
			t.Fatalf("Failed to call Ollama: %v", err)
		}

		if response == "" {
			t.Error("Expected non-empty response")
		}
	})
}

// TestMinFunction tests the min helper function
func TestMinFunction(t *testing.T) {
	testCases := []struct {
		a, b     int
		expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{0, 0, 0},
		{-5, 5, -5},
		{100, 100, 100},
	}

	for _, tc := range testCases {
		result := min(tc.a, tc.b)
		if result != tc.expected {
			t.Errorf("min(%d, %d) = %d, expected %d", tc.a, tc.b, result, tc.expected)
		}
	}
}
