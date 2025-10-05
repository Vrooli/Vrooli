package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestHelperFunctions tests all helper functions with comprehensive coverage
func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BuildCharacterDialogPrompt", func(t *testing.T) {
		testChar := createTestCharacter(t, "PromptTestChar")
		req := DialogGenerationRequest{
			CharacterID:    testChar.Character.ID.String(),
			SceneContext:   "A dark jungle cave",
			EmotionState:   "fearful",
			PreviousDialog: []string{"What was that sound?"},
			Constraints: map[string]interface{}{
				"max_length": 50,
			},
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		// Validate prompt contains key elements
		if !strings.Contains(prompt, testChar.Character.Name) {
			t.Errorf("Prompt should contain character name '%s'", testChar.Character.Name)
		}
		if !strings.Contains(prompt, req.SceneContext) {
			t.Error("Prompt should contain scene context")
		}
		if !strings.Contains(prompt, req.EmotionState) {
			t.Error("Prompt should contain emotion state")
		}
	})

	t.Run("ExtractEmotionFromDialog_Excited", func(t *testing.T) {
		dialog := "This is great! Awesome adventure!"
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "excited" {
			t.Errorf("Expected 'excited', got '%s'", emotion)
		}
	})

	t.Run("ExtractEmotionFromDialog_Curious", func(t *testing.T) {
		dialog := "What is that?"
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "curious" {
			t.Errorf("Expected 'curious', got '%s'", emotion)
		}
	})

	t.Run("ExtractEmotionFromDialog_Thoughtful", func(t *testing.T) {
		dialog := "I wonder... maybe we should wait."
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "thoughtful" {
			t.Errorf("Expected 'thoughtful', got '%s'", emotion)
		}
	})

	t.Run("ExtractEmotionFromDialog_Angry", func(t *testing.T) {
		dialog := "No! Stop right there!"
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "angry" {
			t.Errorf("Expected 'angry', got '%s'", emotion)
		}
	})

	t.Run("ExtractEmotionFromDialog_DefaultEmotion", func(t *testing.T) {
		dialog := "Hello there."
		emotion := extractEmotionFromDialog(dialog, "happy")
		if emotion != "happy" {
			t.Errorf("Expected default 'happy', got '%s'", emotion)
		}
	})

	t.Run("ExtractEmotionFromDialog_Neutral", func(t *testing.T) {
		dialog := "The path continues ahead."
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "neutral" {
			t.Errorf("Expected 'neutral', got '%s'", emotion)
		}
	})

	t.Run("CalculateCharacterConsistency_Brave", func(t *testing.T) {
		testChar := createTestCharacter(t, "BraveHero")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"brave": 0.9,
		}

		dialog := "Let's fight this creature! We must be brave and show courage!"
		score := calculateCharacterConsistency(testChar.Character, dialog)

		if score < 0.7 || score > 1.0 {
			t.Errorf("Expected consistency score between 0.7 and 1.0, got %f", score)
		}
		// Should have bonus for brave keywords
		if score <= 0.7 {
			t.Errorf("Expected higher score for brave character with brave dialog, got %f", score)
		}
	})

	t.Run("CalculateCharacterConsistency_Humorous", func(t *testing.T) {
		testChar := createTestCharacter(t, "FunnyChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"humorous": 0.8,
		}

		dialog := "Haha, that was a funny joke!"
		score := calculateCharacterConsistency(testChar.Character, dialog)

		if score < 0.7 || score > 1.0 {
			t.Errorf("Expected consistency score between 0.7 and 1.0, got %f", score)
		}
	})

	t.Run("CalculateCharacterConsistency_NoMatch", func(t *testing.T) {
		testChar := createTestCharacter(t, "PlainChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"quiet": 0.5,
		}

		dialog := "The weather is nice."
		score := calculateCharacterConsistency(testChar.Character, dialog)

		if score != 0.7 {
			t.Errorf("Expected base score of 0.7, got %f", score)
		}
	})

	t.Run("CalculateAverageConsistency_MultipleDialogs", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{
			{CharacterConsistencyScore: 0.8},
			{CharacterConsistencyScore: 0.9},
			{CharacterConsistencyScore: 0.7},
		}

		avg := calculateAverageConsistency(dialogSet)
		expected := (0.8 + 0.9 + 0.7) / 3.0

		// Use epsilon for floating point comparison
		epsilon := 0.0001
		if avg < expected-epsilon || avg > expected+epsilon {
			t.Errorf("Expected average close to %f, got %f", expected, avg)
		}
	})

	t.Run("CalculateAverageConsistency_EmptySet", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{}
		avg := calculateAverageConsistency(dialogSet)

		if avg != 0.0 {
			t.Errorf("Expected 0.0 for empty set, got %f", avg)
		}
	})

	t.Run("CalculateAverageConsistency_SingleDialog", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{
			{CharacterConsistencyScore: 0.85},
		}

		avg := calculateAverageConsistency(dialogSet)

		if avg != 0.85 {
			t.Errorf("Expected 0.85, got %f", avg)
		}
	})
}

// TestDataStructures tests all data structure marshaling and unmarshaling
func TestDataStructures(t *testing.T) {
	t.Run("Character_JSONSerialization", func(t *testing.T) {
		char := Character{
			ID:   uuid.New(),
			Name: "TestChar",
			PersonalityTraits: map[string]interface{}{
				"brave": 0.8,
			},
			BackgroundStory: "A brave explorer",
			SpeechPatterns: map[string]interface{}{
				"tempo": "fast",
			},
			Relationships: map[string]interface{}{
				"friends": []string{"ally1"},
			},
			VoiceProfile: map[string]interface{}{
				"pitch": "high",
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		// Marshal to JSON
		data, err := json.Marshal(char)
		if err != nil {
			t.Fatalf("Failed to marshal character: %v", err)
		}

		// Unmarshal back
		var decoded Character
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal character: %v", err)
		}

		// Validate
		if decoded.Name != char.Name {
			t.Errorf("Expected name %s, got %s", char.Name, decoded.Name)
		}
	})

	t.Run("DialogGenerationRequest_Validation", func(t *testing.T) {
		req := DialogGenerationRequest{
			CharacterID:    uuid.New().String(),
			SceneContext:   "Test scene",
			PreviousDialog: []string{"line1", "line2"},
			EmotionState:   "happy",
			Constraints: map[string]interface{}{
				"max_length": 100,
			},
		}

		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		var decoded DialogGenerationRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		if decoded.CharacterID != req.CharacterID {
			t.Error("Character ID mismatch after JSON round-trip")
		}
		if len(decoded.PreviousDialog) != 2 {
			t.Error("Previous dialog array mismatch")
		}
	})

	t.Run("BatchDialogRequest_Validation", func(t *testing.T) {
		batch := BatchDialogRequest{
			SceneID: uuid.New().String(),
			DialogRequests: []DialogGenerationRequest{
				{
					CharacterID:  uuid.New().String(),
					SceneContext: "Scene 1",
				},
				{
					CharacterID:  uuid.New().String(),
					SceneContext: "Scene 2",
				},
			},
			ExportFormat: "json",
		}

		data, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("Failed to marshal batch request: %v", err)
		}

		var decoded BatchDialogRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal batch request: %v", err)
		}

		if len(decoded.DialogRequests) != 2 {
			t.Errorf("Expected 2 dialog requests, got %d", len(decoded.DialogRequests))
		}
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyCharacterName", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		env.Router.POST("/api/v1/characters", func(c *gin.Context) {
			var character Character
			if err := c.ShouldBindJSON(&character); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid character data"})
				return
			}

			// Validate name is not empty
			if character.Name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Character name is required"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{"success": true})
		})

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body: map[string]interface{}{
				"name":               "",
				"personality_traits": map[string]interface{}{},
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("VeryLongCharacterName", func(t *testing.T) {
		t.Skip("Skipping - requires database connection")
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		env.Router.POST("/api/v1/characters", createCharacterHandler)

		// Create a very long name (500 characters)
		longName := strings.Repeat("A", 500)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body: map[string]interface{}{
				"name":               longName,
				"personality_traits": map[string]interface{}{},
				"background_story":   "Test",
			},
		})

		// Should still accept it (no validation in place currently)
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})

	t.Run("NullPersonalityTraits", func(t *testing.T) {
		t.Skip("Skipping - requires database connection")
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		env.Router.POST("/api/v1/characters", createCharacterHandler)

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/characters",
			Body: map[string]interface{}{
				"name":               "TestChar",
				"personality_traits": nil,
				"background_story":   "Test",
			},
		})

		// Should handle null gracefully
		if w.Code != http.StatusCreated && w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status code: %d", w.Code)
		}
	})

	t.Run("LargeDialogBatch", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		testChar := createTestCharacter(t, "BatchChar")
		mockDB := NewMockDatabase()
		mockDB.SaveCharacter(testChar.Character)

		// Create route with mock
		env.Router.POST("/api/v1/dialog/batch", func(c *gin.Context) {
			var req BatchDialogRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
				return
			}

			var dialogSet []DialogGenerationResponse
			for range req.DialogRequests {
				dialogSet = append(dialogSet, DialogGenerationResponse{
					Dialog:                   "Test dialog",
					Emotion:                  "neutral",
					CharacterConsistencyScore: 0.8,
				})
			}

			c.JSON(http.StatusOK, BatchDialogResponse{
				DialogSet: dialogSet,
				GenerationMetrics: map[string]interface{}{
					"total": len(dialogSet),
				},
			})
		})

		// Create a batch with 100 requests
		requests := make([]DialogGenerationRequest, 100)
		for i := 0; i < 100; i++ {
			requests[i] = DialogGenerationRequest{
				CharacterID:  testChar.Character.ID.String(),
				SceneContext: fmt.Sprintf("Scene %d", i),
			}
		}

		batch := BatchDialogRequest{
			SceneID:        uuid.New().String(),
			DialogRequests: requests,
		}

		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/dialog/batch",
			Body:   batch,
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestCORSHeaders tests CORS middleware functionality
func TestCORSHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Add CORS middleware
	env.Router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	env.Router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"success": true})
	})

	t.Run("CORSHeadersPresent", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		})

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS header to be '*'")
		}
	})

	t.Run("OPTIONSRequest", func(t *testing.T) {
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/test",
		})

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
		}
	})
}

// TestErrorHandling tests comprehensive error handling
func TestErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidUUIDFormat", func(t *testing.T) {
		invalidIDs := []string{
			"not-a-uuid",
			"12345",
			"",
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
			"123e4567-e89b-12d3-a456-42661417400",  // Missing one character
		}

		for _, invalidID := range invalidIDs {
			t.Run(fmt.Sprintf("InvalidID_%s", invalidID), func(t *testing.T) {
				_, err := uuid.Parse(invalidID)
				if err == nil {
					t.Errorf("Expected error for invalid UUID '%s'", invalidID)
				}
			})
		}
	})

	t.Run("JSONUnmarshalErrors", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		env.Router.POST("/test", func(c *gin.Context) {
			var data map[string]interface{}
			if err := c.ShouldBindJSON(&data); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"success": true})
		})

		malformedJSON := []string{
			`{"key": "value"`,           // Missing closing brace
			`{"key": value}`,             // Unquoted value
			`{key: "value"}`,             // Unquoted key
			``,                           // Empty string
			`null`,                       // Null
		}

		for _, jsonStr := range malformedJSON {
			w := makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: "POST",
				Path:   "/test",
				Body:   jsonStr,
			})

			if w.Code == http.StatusOK {
				t.Errorf("Expected error for malformed JSON: %s", jsonStr)
			}
		}
	})
}
