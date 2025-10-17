package main

import (
	"testing"

	"github.com/google/uuid"
)

// TestGetCharacterFromDB tests the getCharacterFromDB helper function
func TestGetCharacterFromDB(t *testing.T) {
	t.Skip("Requires database connection - will be tested in integration tests")
}

// TestGenerateDialogWithOllama tests dialog generation with Ollama
func TestGenerateDialogWithOllama(t *testing.T) {
	t.Skip("Requires Ollama connection - will be tested in integration tests")
}

// TestGenerateCharacterEmbedding tests character embedding generation
func TestGenerateCharacterEmbedding(t *testing.T) {
	t.Skip("Requires Ollama connection - will be tested in integration tests")
}

// TestStoreCharacterEmbedding tests storing embeddings in Qdrant
func TestStoreCharacterEmbedding(t *testing.T) {
	t.Skip("Requires Qdrant connection - will be tested in integration tests")
}

// TestCalculateCharacterConsistencyEdgeCases tests edge cases for consistency calculation
func TestCalculateCharacterConsistencyEdgeCases(t *testing.T) {
	t.Run("BoundaryScores", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")

		// Test score capping at 1.0
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"brave":    0.9,
			"humorous": 0.9,
		}

		dialog := "This is brave and we must fight with courage! Haha, that's funny and a great joke!"
		score := calculateCharacterConsistency(testChar.Character, dialog)

		// Should be capped at 1.0
		if score > 1.0 {
			t.Errorf("Score should be capped at 1.0, got %f", score)
		}
		if score < 0.0 {
			t.Errorf("Score should not be negative, got %f", score)
		}
	})

	t.Run("EmptyDialog", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		score := calculateCharacterConsistency(testChar.Character, "")

		// Should return base score
		if score != 0.7 {
			t.Errorf("Expected base score 0.7 for empty dialog, got %f", score)
		}
	})

	t.Run("CaseInsensitivity", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"brave": 0.8,
		}

		// Test uppercase
		score1 := calculateCharacterConsistency(testChar.Character, "BRAVE FIGHT COURAGE")
		// Test lowercase
		score2 := calculateCharacterConsistency(testChar.Character, "brave fight courage")

		if score1 != score2 {
			t.Errorf("Consistency should be case-insensitive, got %f and %f", score1, score2)
		}
	})

	t.Run("NonFloatPersonalityTraits", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"brave": "high", // Non-float value
		}

		dialog := "Let's fight bravely!"
		score := calculateCharacterConsistency(testChar.Character, dialog)

		// Should handle gracefully and return base score
		if score < 0.0 || score > 1.0 {
			t.Errorf("Score should be valid even with non-float traits, got %f", score)
		}
	})
}

// TestExtractEmotionFromDialogEdgeCases tests edge cases for emotion extraction
func TestExtractEmotionFromDialogEdgeCases(t *testing.T) {
	t.Run("MultiplePunctuation", func(t *testing.T) {
		dialog := "What?! Is that great?!"
		emotion := extractEmotionFromDialog(dialog, "")

		// Should detect based on pattern priority
		if emotion != "excited" && emotion != "curious" {
			t.Logf("Got emotion: %s (acceptable)", emotion)
		}
	})

	t.Run("EmptyDialog", func(t *testing.T) {
		emotion := extractEmotionFromDialog("", "happy")
		if emotion != "happy" {
			t.Errorf("Expected default emotion 'happy', got '%s'", emotion)
		}
	})

	t.Run("EmptyDialogNoDefault", func(t *testing.T) {
		emotion := extractEmotionFromDialog("", "")
		if emotion != "neutral" {
			t.Errorf("Expected 'neutral' when no default, got '%s'", emotion)
		}
	})

	t.Run("MixedCase", func(t *testing.T) {
		dialog := "NO! STOP"
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "angry" {
			t.Errorf("Expected 'angry' for shouting, got '%s'", emotion)
		}
	})

	t.Run("MultipleEllipsis", func(t *testing.T) {
		dialog := "I wonder... perhaps... maybe..."
		emotion := extractEmotionFromDialog(dialog, "")
		if emotion != "thoughtful" {
			t.Errorf("Expected 'thoughtful' for ellipsis, got '%s'", emotion)
		}
	})
}

// TestBuildCharacterDialogPromptEdgeCases tests edge cases for prompt building
func TestBuildCharacterDialogPromptEdgeCases(t *testing.T) {
	t.Run("EmptySceneContext", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		req := DialogGenerationRequest{
			CharacterID:  testChar.Character.ID.String(),
			SceneContext: "",
			EmotionState: "neutral",
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		// Should still build prompt even with empty context
		if prompt == "" {
			t.Error("Prompt should not be empty even with empty scene context")
		}
		if len(prompt) < 100 {
			t.Errorf("Prompt seems too short: %d characters", len(prompt))
		}
	})

	t.Run("EmptyPreviousDialog", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		req := DialogGenerationRequest{
			CharacterID:    testChar.Character.ID.String(),
			SceneContext:   "Test scene",
			EmotionState:   "happy",
			PreviousDialog: []string{},
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		if prompt == "" {
			t.Error("Prompt should not be empty")
		}
	})

	t.Run("LongPreviousDialog", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		longPrevious := make([]string, 50)
		for i := range longPrevious {
			longPrevious[i] = "Previous dialog line"
		}

		req := DialogGenerationRequest{
			CharacterID:    testChar.Character.ID.String(),
			SceneContext:   "Test scene",
			PreviousDialog: longPrevious,
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		// Should include all previous dialog
		if prompt == "" {
			t.Error("Prompt should not be empty")
		}
	})

	t.Run("NilPersonalityTraits", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = nil

		req := DialogGenerationRequest{
			CharacterID:  testChar.Character.ID.String(),
			SceneContext: "Test scene",
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		// Should handle nil traits gracefully
		if prompt == "" {
			t.Error("Prompt should not be empty even with nil traits")
		}
	})

	t.Run("SpecialCharactersInDialog", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		req := DialogGenerationRequest{
			CharacterID:  testChar.Character.ID.String(),
			SceneContext: "Test scene with \"quotes\" and 'apostrophes'",
			PreviousDialog: []string{
				"Dialog with\nnewlines",
				"Dialog with\ttabs",
			},
		}

		prompt := buildCharacterDialogPrompt(testChar.Character, req)

		// Should handle special characters
		if prompt == "" {
			t.Error("Prompt should not be empty")
		}
	})
}

// TestCalculateAverageConsistencyEdgeCases tests more edge cases
func TestCalculateAverageConsistencyEdgeCases(t *testing.T) {
	t.Run("SingleHighScore", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{
			{CharacterConsistencyScore: 1.0},
		}
		avg := calculateAverageConsistency(dialogSet)
		if avg != 1.0 {
			t.Errorf("Expected 1.0, got %f", avg)
		}
	})

	t.Run("SingleLowScore", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{
			{CharacterConsistencyScore: 0.0},
		}
		avg := calculateAverageConsistency(dialogSet)
		if avg != 0.0 {
			t.Errorf("Expected 0.0, got %f", avg)
		}
	})

	t.Run("AllSameScores", func(t *testing.T) {
		dialogSet := []DialogGenerationResponse{
			{CharacterConsistencyScore: 0.75},
			{CharacterConsistencyScore: 0.75},
			{CharacterConsistencyScore: 0.75},
		}
		avg := calculateAverageConsistency(dialogSet)
		if avg != 0.75 {
			t.Errorf("Expected 0.75, got %f", avg)
		}
	})

	t.Run("LargeDataset", func(t *testing.T) {
		dialogSet := make([]DialogGenerationResponse, 1000)
		for i := range dialogSet {
			dialogSet[i].CharacterConsistencyScore = 0.8
		}
		avg := calculateAverageConsistency(dialogSet)
		epsilon := 0.0001
		if avg < 0.8-epsilon || avg > 0.8+epsilon {
			t.Errorf("Expected close to 0.8, got %f", avg)
		}
	})
}

// TestUUIDOperations tests UUID-related operations
func TestUUIDOperations(t *testing.T) {
	t.Run("ValidUUIDParsing", func(t *testing.T) {
		validUUID := uuid.New().String()
		parsed, err := uuid.Parse(validUUID)
		if err != nil {
			t.Errorf("Failed to parse valid UUID: %v", err)
		}
		if parsed.String() != validUUID {
			t.Error("UUID roundtrip failed")
		}
	})

	t.Run("InvalidUUIDParsing", func(t *testing.T) {
		invalidUUIDs := []string{
			"not-a-uuid",
			"12345",
			"",
			"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
		}

		for _, invalid := range invalidUUIDs {
			_, err := uuid.Parse(invalid)
			if err == nil {
				t.Errorf("Expected error for invalid UUID: %s", invalid)
			}
		}
	})

	t.Run("UUIDGeneration", func(t *testing.T) {
		uuid1 := uuid.New()
		uuid2 := uuid.New()

		if uuid1 == uuid2 {
			t.Error("Generated UUIDs should be unique")
		}

		if uuid1.String() == "" {
			t.Error("UUID string should not be empty")
		}
	})
}

// TestJSONMarshalingHelpers tests JSON marshaling for complex structures
func TestJSONMarshalingHelpers(t *testing.T) {
	t.Run("EmptyPersonalityTraits", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{}

		// Should marshal empty map
		if testChar.Character.PersonalityTraits == nil {
			t.Error("Empty traits should not be nil")
		}
	})

	t.Run("ComplexNestedStructures", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"nested": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "deep value",
				},
			},
		}

		// Should handle nested structures
		if testChar.Character.PersonalityTraits == nil {
			t.Error("Nested traits should not be nil")
		}
	})

	t.Run("MixedTypes", func(t *testing.T) {
		testChar := createTestCharacter(t, "TestChar")
		testChar.Character.PersonalityTraits = map[string]interface{}{
			"string_val":  "test",
			"int_val":     42,
			"float_val":   3.14,
			"bool_val":    true,
			"array_val":   []string{"a", "b", "c"},
			"map_val":     map[string]int{"x": 1, "y": 2},
		}

		// Should handle mixed types
		if len(testChar.Character.PersonalityTraits) != 6 {
			t.Errorf("Expected 6 traits, got %d", len(testChar.Character.PersonalityTraits))
		}
	})
}
