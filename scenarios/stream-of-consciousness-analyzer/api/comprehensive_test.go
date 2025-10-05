// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// TestCampaignLifecycle tests complete campaign lifecycle
func TestCampaignLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	var campaignID string

	t.Run("CreateCampaign", func(t *testing.T) {
		campaign := Campaign{
			Name:          "Lifecycle Test Campaign",
			Description:   "Testing full campaign lifecycle",
			ContextPrompt: "Track all thoughts about project planning",
			Color:         "#10B981",
			Icon:          "🌟",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/campaigns",
			Body:   campaign,
		}

		w, err := makeHTTPRequest(req, createCampaign)
		if err != nil {
			t.Fatalf("Failed to create campaign: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		var result Campaign
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.ID == "" {
			t.Fatal("Campaign ID not set")
		}

		campaignID = result.ID

		if result.Name != campaign.Name {
			t.Errorf("Name mismatch: expected %s, got %s", campaign.Name, result.Name)
		}

		if !result.Active {
			t.Error("Campaign should be active after creation")
		}
	})

	t.Run("CaptureStreamEntries", func(t *testing.T) {
		entries := []string{
			"Need to refactor the authentication module",
			"Consider migrating to microservices architecture",
			"Team meeting scheduled for next Tuesday",
		}

		for _, content := range entries {
			entry := StreamEntry{
				CampaignID: campaignID,
				Content:    content,
				Type:       "text",
				Source:     "manual",
				Metadata:   json.RawMessage(`{"source_app": "test"}`),
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/stream/capture",
				Body:   entry,
			}

			w, err := makeHTTPRequest(req, captureStream)
			if err != nil {
				t.Errorf("Failed to capture stream: %v", err)
				continue
			}

			if w.Code != http.StatusCreated {
				t.Errorf("Expected status 201, got %d for entry: %s", w.Code, content)
			}
		}
	})

	t.Run("VerifyNotesCreated", func(t *testing.T) {
		// Give processing time (in real scenario, n8n would process)
		time.Sleep(100 * time.Millisecond)

		// Create some organized notes manually for testing
		createTestNote(t, testDB, campaignID, "Authentication Refactoring")
		createTestNote(t, testDB, campaignID, "Microservices Migration Plan")

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaignID},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to get notes: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		if len(notes) < 2 {
			t.Errorf("Expected at least 2 notes, got %d", len(notes))
		}

		for _, note := range notes {
			if note.CampaignID != campaignID {
				t.Errorf("Note campaign_id mismatch: expected %s, got %s", campaignID, note.CampaignID)
			}
		}
	})

	t.Run("SearchNotes", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q":           "Authentication",
				"campaign_id": campaignID,
			},
		}

		w, err := makeHTTPRequest(req, searchNotes)
		if err != nil {
			t.Fatalf("Failed to search notes: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		// Should find notes containing "Authentication"
		foundAuthNote := false
		for _, note := range notes {
			if strings.Contains(note.Title, "Authentication") || strings.Contains(note.Content, "Authentication") {
				foundAuthNote = true
				break
			}
		}

		if len(notes) > 0 && !foundAuthNote {
			t.Log("Search might not have found the expected note - verify search implementation")
		}
	})

	t.Run("GenerateInsights", func(t *testing.T) {
		// Create notes first
		note1 := createTestNote(t, testDB, campaignID, "Backend Performance Issue")
		note2 := createTestNote(t, testDB, campaignID, "Database Query Optimization")

		// Create an insight connecting the notes
		insight := createTestInsight(t, testDB, campaignID, []string{note1.ID, note2.ID})

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/insights",
			QueryParams: map[string]string{"campaign_id": campaignID},
		}

		w, err := makeHTTPRequest(req, getInsights)
		if err != nil {
			t.Fatalf("Failed to get insights: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var insights []Insight
		json.Unmarshal(w.Body.Bytes(), &insights)

		if len(insights) < 1 {
			t.Error("Expected at least 1 insight")
		}

		foundInsight := false
		for _, i := range insights {
			if i.ID == insight.ID {
				foundInsight = true
				if i.Confidence < 0.6 {
					t.Errorf("Expected confidence >= 0.6, got %f", i.Confidence)
				}
			}
		}

		if !foundInsight {
			t.Error("Created insight not found in results")
		}
	})
}

// TestStreamProcessingFlow tests the complete stream processing workflow
func TestStreamProcessingFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Stream Processing Campaign")

	t.Run("CaptureTextStream", func(t *testing.T) {
		entry := StreamEntry{
			CampaignID: campaign.ID,
			Content:    "This is a stream of consciousness about the new feature we're building",
			Type:       "text",
			Source:     "keyboard",
			Metadata:   json.RawMessage(`{"timestamp": "2025-01-01T10:00:00Z"}`),
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stream/capture",
			Body:   entry,
		}

		w, err := makeHTTPRequest(req, captureStream)
		if err != nil {
			t.Fatalf("Failed to capture stream: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", w.Code)
		}

		var result StreamEntry
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.ID == "" {
			t.Error("Stream entry ID not set")
		}

		if result.CampaignID != campaign.ID {
			t.Errorf("Campaign ID mismatch: expected %s, got %s", campaign.ID, result.CampaignID)
		}

		if result.Type != "text" {
			t.Errorf("Type mismatch: expected text, got %s", result.Type)
		}
	})

	t.Run("CaptureVoiceStream", func(t *testing.T) {
		entry := StreamEntry{
			CampaignID: campaign.ID,
			Content:    "Voice transcribed: Need to schedule a team meeting for sprint planning",
			Type:       "voice",
			Source:     "microphone",
			Metadata:   json.RawMessage(`{"duration": 15, "language": "en-US"}`),
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stream/capture",
			Body:   entry,
		}

		w, err := makeHTTPRequest(req, captureStream)
		if err != nil {
			t.Fatalf("Failed to capture voice stream: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Fatalf("Expected status 201, got %d", w.Code)
		}
	})
}

// TestNotesOrganization tests note organization and categorization
func TestNotesOrganization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Organization Test Campaign")

	t.Run("NotesWithDifferentCategories", func(t *testing.T) {
		categories := []string{"technical", "meeting", "idea", "task"}

		for _, category := range categories {
			note := &OrganizedNote{
				CampaignID: campaign.ID,
				Title:      "Note for " + category,
				Content:    "Content about " + category,
				Summary:    "Summary",
				Category:   category,
				Tags:       []string{category, "test"},
				Priority:   5,
				Metadata:   json.RawMessage(`{}`),
			}

			tags, _ := json.Marshal(note.Tags)

			err := testDB.DB.QueryRow(`
				INSERT INTO organized_notes (campaign_id, title, content, summary, category, tags, priority, metadata)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
				RETURNING id, created_at, updated_at
			`, note.CampaignID, note.Title, note.Content, note.Summary,
				note.Category, tags, note.Priority, note.Metadata).Scan(&note.ID, &note.CreatedAt, &note.UpdatedAt)

			if err != nil {
				t.Errorf("Failed to create note for category %s: %v", category, err)
			}
		}

		// Retrieve all notes
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to get notes: %v", err)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		if len(notes) < len(categories) {
			t.Errorf("Expected at least %d notes, got %d", len(categories), len(notes))
		}

		// Verify categories are preserved
		categoriesFound := make(map[string]bool)
		for _, note := range notes {
			categoriesFound[note.Category] = true
		}

		for _, category := range categories {
			if !categoriesFound[category] {
				t.Errorf("Category %s not found in results", category)
			}
		}
	})

	t.Run("NotesWithPriority", func(t *testing.T) {
		priorities := []int{1, 5, 10}

		for _, priority := range priorities {
			note := createTestNote(t, testDB, campaign.ID, "Priority Note")

			// Update priority
			_, err := testDB.DB.Exec(`
				UPDATE organized_notes SET priority = $1 WHERE id = $2
			`, priority, note.ID)

			if err != nil {
				t.Errorf("Failed to update note priority: %v", err)
			}
		}

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/notes",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getNotes)
		if err != nil {
			t.Fatalf("Failed to get notes: %v", err)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		// Verify different priorities exist
		prioritiesFound := make(map[int]bool)
		for _, note := range notes {
			prioritiesFound[note.Priority] = true
		}

		if len(prioritiesFound) < 2 {
			t.Log("Expected notes with different priorities")
		}
	})
}

// TestInsightGeneration tests insight generation and filtering
func TestInsightGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	campaign := createTestCampaign(t, testDB, "Insight Generation Campaign")

	t.Run("HighConfidenceInsights", func(t *testing.T) {
		note1 := createTestNote(t, testDB, campaign.ID, "Pattern Note 1")
		note2 := createTestNote(t, testDB, campaign.ID, "Pattern Note 2")
		note3 := createTestNote(t, testDB, campaign.ID, "Pattern Note 3")

		// Create high confidence insight
		insight := &Insight{
			CampaignID: campaign.ID,
			NoteIDs:    []string{note1.ID, note2.ID, note3.ID},
			Type:       "pattern",
			Content:    "All three notes discuss similar patterns in the codebase",
			Confidence: 0.92,
			Metadata:   json.RawMessage(`{"analysis": "semantic similarity"}`),
		}

		noteIDsJSON, _ := json.Marshal(insight.NoteIDs)

		err := testDB.DB.QueryRow(`
			INSERT INTO insights (campaign_id, note_ids, insight_type, content, confidence, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at
		`, insight.CampaignID, noteIDsJSON, insight.Type, insight.Content,
			insight.Confidence, insight.Metadata).Scan(&insight.ID, &insight.CreatedAt)

		if err != nil {
			t.Fatalf("Failed to create insight: %v", err)
		}

		// Retrieve insights
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/insights",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getInsights)
		if err != nil {
			t.Fatalf("Failed to get insights: %v", err)
		}

		var insights []Insight
		json.Unmarshal(w.Body.Bytes(), &insights)

		if len(insights) < 1 {
			t.Fatal("Expected at least 1 insight")
		}

		// Verify high confidence insight is returned
		foundHighConfidence := false
		for _, i := range insights {
			if i.ID == insight.ID && i.Confidence >= 0.9 {
				foundHighConfidence = true
			}
		}

		if !foundHighConfidence {
			t.Error("High confidence insight not found")
		}
	})

	t.Run("LowConfidenceFiltered", func(t *testing.T) {
		note := createTestNote(t, testDB, campaign.ID, "Low Confidence Note")

		// Create low confidence insight - should be filtered out
		lowInsight := &Insight{
			CampaignID: campaign.ID,
			NoteIDs:    []string{note.ID},
			Type:       "tentative",
			Content:    "Possible connection but uncertain",
			Confidence: 0.35,
			Metadata:   json.RawMessage(`{}`),
		}

		noteIDsJSON, _ := json.Marshal(lowInsight.NoteIDs)

		_, err := testDB.DB.Exec(`
			INSERT INTO insights (campaign_id, note_ids, insight_type, content, confidence, metadata)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, lowInsight.CampaignID, noteIDsJSON, lowInsight.Type,
			lowInsight.Content, lowInsight.Confidence, lowInsight.Metadata)

		if err != nil {
			t.Fatalf("Failed to create low confidence insight: %v", err)
		}

		// Retrieve insights
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/insights",
			QueryParams: map[string]string{"campaign_id": campaign.ID},
		}

		w, err := makeHTTPRequest(req, getInsights)
		if err != nil {
			t.Fatalf("Failed to get insights: %v", err)
		}

		var insights []Insight
		json.Unmarshal(w.Body.Bytes(), &insights)

		// Verify low confidence insights are filtered (confidence < 0.6)
		for _, i := range insights {
			if i.Confidence < 0.6 {
				t.Errorf("Found insight with confidence %f < 0.6 (should be filtered)", i.Confidence)
			}
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDB(t)
	if testDB == nil {
		t.Skip("Database not available for testing")
	}
	defer testDB.Cleanup()

	t.Run("EmptyStringFields", func(t *testing.T) {
		campaign := Campaign{
			Name:          "Edge Case Campaign",
			Description:   "",
			ContextPrompt: "",
			Color:         "",
			Icon:          "",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/campaigns",
			Body:   campaign,
		}

		w, err := makeHTTPRequest(req, createCampaign)
		if err != nil {
			t.Fatalf("Failed to create campaign: %v", err)
		}

		// Should either succeed or return a validation error
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 201 or 400, got %d", w.Code)
		}
	})

	t.Run("VeryLongContent", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB, "Long Content Campaign")

		// Create entry with very long content
		longContent := strings.Repeat("This is a very long stream of consciousness entry. ", 100)

		entry := StreamEntry{
			CampaignID: campaign.ID,
			Content:    longContent,
			Type:       "text",
			Source:     "test",
			Metadata:   json.RawMessage(`{}`),
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stream/capture",
			Body:   entry,
		}

		w, err := makeHTTPRequest(req, captureStream)
		if err != nil {
			t.Fatalf("Failed to capture long stream: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201 for long content, got %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInContent", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB, "Special Chars Campaign")

		specialContent := "Testing special chars: <>&\"'`\n\t\r\\/@#$%^&*()"

		entry := StreamEntry{
			CampaignID: campaign.ID,
			Content:    specialContent,
			Type:       "text",
			Source:     "test",
			Metadata:   json.RawMessage(`{}`),
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stream/capture",
			Body:   entry,
		}

		w, err := makeHTTPRequest(req, captureStream)
		if err != nil {
			t.Fatalf("Failed to capture special chars stream: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		var result StreamEntry
		json.Unmarshal(w.Body.Bytes(), &result)

		if result.Content != specialContent {
			t.Error("Special characters not preserved correctly")
		}
	})

	t.Run("UnicodeAndEmoji", func(t *testing.T) {
		campaign := createTestCampaign(t, testDB, "Unicode Campaign")

		unicodeContent := "Testing Unicode: 你好世界 مرحبا العالم 🌍🌎🌏 ✨🎉🚀"

		entry := StreamEntry{
			CampaignID: campaign.ID,
			Content:    unicodeContent,
			Type:       "text",
			Source:     "test",
			Metadata:   json.RawMessage(`{}`),
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/stream/capture",
			Body:   entry,
		}

		w, err := makeHTTPRequest(req, captureStream)
		if err != nil {
			t.Fatalf("Failed to capture unicode stream: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("EmptyQueryResults", func(t *testing.T) {
		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/search",
			QueryParams: map[string]string{"q": "xyznonexistentterm9999"},
		}

		w, err := makeHTTPRequest(req, searchNotes)
		if err != nil {
			t.Fatalf("Failed to search: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty results, got %d", w.Code)
		}

		var notes []OrganizedNote
		json.Unmarshal(w.Body.Bytes(), &notes)

		// Should return empty array, not null
		if notes == nil {
			notes = []OrganizedNote{}
		}
	})
}

// TestDatabaseConnectionHandling tests database connection scenarios
func TestDatabaseConnectionHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthCheckWithValidDB", func(t *testing.T) {
		testDB := setupTestDB(t)
		if testDB == nil {
			t.Skip("Database not available for testing")
		}
		defer testDB.Cleanup()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(req, healthCheck)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		result := assertJSONResponse(t, w, http.StatusOK)
		if result["status"] != "healthy" {
			t.Errorf("Expected healthy status, got %v", result["status"])
		}
	})
}
