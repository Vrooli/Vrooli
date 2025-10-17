package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TestDialogGenerationPerformance tests dialog generation performance
func TestDialogGenerationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testChar := createTestCharacter(t, "PerfChar")
	mockDB := NewMockDatabase()
	mockDB.SaveCharacter(testChar.Character)

	// Setup route
	setupDialogGenerationRoute(env.Router, mockDB)

	t.Run("SingleDialogGenerationSpeed", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "SingleDialogGeneration",
			Description: "Measure time to generate a single dialog",
			MaxDuration: 500 * time.Millisecond,
			Setup: func(t *testing.T) interface{} {
				return testChar.Character.ID.String()
			},
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				characterID := setupData.(string)
				dialogReq := TestData.DialogRequest(characterID, "Performance test scene")

				start := time.Now()
				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/dialog/generate",
					Body:   dialogReq,
				})
				duration := time.Since(start)

				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				return duration
			},
		}

		RunPerformanceTest(t, pattern)
	})

	t.Run("BatchDialogGenerationSpeed", func(t *testing.T) {
		// Create multiple characters
		chars := make([]*TestCharacter, 10)
		for i := 0; i < 10; i++ {
			chars[i] = createTestCharacter(t, fmt.Sprintf("BatchChar%d", i))
			mockDB.SaveCharacter(chars[i].Character)
		}

		pattern := PerformanceTestPattern{
			Name:        "BatchDialogGeneration",
			Description: "Measure time to generate batch dialogs",
			MaxDuration: 2 * time.Second,
			Setup: func(t *testing.T) interface{} {
				return chars
			},
			Execute: func(t *testing.T, setupData interface{}) time.Duration {
				characters := setupData.([]*TestCharacter)

				var dialogReqs []DialogGenerationRequest
				for _, char := range characters {
					dialogReqs = append(dialogReqs, TestData.DialogRequest(
						char.Character.ID.String(),
						fmt.Sprintf("Scene for %s", char.Character.Name),
					))
				}

				batchReq := TestData.BatchDialogRequest(uuid.New().String(), dialogReqs)

				start := time.Now()
				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/dialog/batch",
					Body:   batchReq,
				})
				duration := time.Since(start)

				if w.Code != 200 {
					t.Errorf("Expected status 200, got %d", w.Code)
				}

				return duration
			},
		}

		RunPerformanceTest(t, pattern)
	})
}

// TestConcurrentDialogGeneration tests concurrent dialog generation
func TestConcurrentDialogGeneration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	testChar := createTestCharacter(t, "ConcurrentChar")
	mockDB := NewMockDatabase()
	mockDB.SaveCharacter(testChar.Character)

	setupDialogGenerationRoute(env.Router, mockDB)

	t.Run("ConcurrentRequests", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make([]error, concurrency)
		durations := make([]time.Duration, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				dialogReq := TestData.DialogRequest(testChar.Character.ID.String(), fmt.Sprintf("Concurrent scene %d", index))

				reqStart := time.Now()
				w := makeHTTPRequest(env.Router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/dialog/generate",
					Body:   dialogReq,
				})
				durations[index] = time.Since(reqStart)

				if w.Code != 200 {
					errors[index] = fmt.Errorf("request %d failed with status %d", index, w.Code)
				}
			}(i)
		}

		wg.Wait()
		totalDuration := time.Since(start)

		// Validate results
		failCount := 0
		for i, err := range errors {
			if err != nil {
				t.Errorf("Concurrent request %d failed: %v", i, err)
				failCount++
			}
		}

		if failCount > 0 {
			t.Errorf("%d out of %d concurrent requests failed", failCount, concurrency)
		}

		// Calculate average response time
		var totalReqTime time.Duration
		for _, d := range durations {
			totalReqTime += d
		}
		avgReqTime := totalReqTime / time.Duration(concurrency)

		t.Logf("Concurrent test completed in %v", totalDuration)
		t.Logf("Average request time: %v", avgReqTime)

		// Ensure total time is reasonable (should be much less than sequential)
		maxExpectedTime := 5 * time.Second
		if totalDuration > maxExpectedTime {
			t.Errorf("Concurrent execution took too long: %v > %v", totalDuration, maxExpectedTime)
		}
	})
}

// TestCharacterCreationPerformance tests character creation performance
func TestCharacterCreationPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	mockDB := NewMockDatabase()

	// Setup route
	env.Router.POST("/api/v1/characters", func(c *gin.Context) {
		var character Character
		if err := c.ShouldBindJSON(&character); err != nil {
			c.JSON(400, gin.H{"error": "Invalid character data"})
			return
		}
		character.ID = uuid.New()
		character.CreatedAt = time.Now().UTC()
		character.UpdatedAt = time.Now().UTC()
		mockDB.SaveCharacter(character)
		c.JSON(201, gin.H{
			"success":      true,
			"character_id": character.ID.String(),
		})
	})

	t.Run("BulkCharacterCreation", func(t *testing.T) {
		numCharacters := 50
		start := time.Now()

		for i := 0; i < numCharacters; i++ {
			charData := TestData.CharacterRequest(fmt.Sprintf("BulkChar%d", i))
			w := makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/characters",
				Body:   charData,
			})

			if w.Code != 201 {
				t.Errorf("Character creation %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(numCharacters)

		t.Logf("Created %d characters in %v", numCharacters, duration)
		t.Logf("Average time per character: %v", avgTime)

		// Should complete in reasonable time
		maxTime := 10 * time.Second
		if duration > maxTime {
			t.Errorf("Bulk creation took too long: %v > %v", duration, maxTime)
		}
	})
}

// TestDatabaseQueryPerformance tests database query performance patterns
func TestDatabaseQueryPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	mockDB := NewMockDatabase()

	// Pre-populate with test data
	for i := 0; i < 100; i++ {
		char := createTestCharacter(t, fmt.Sprintf("QueryChar%d", i))
		mockDB.SaveCharacter(char.Character)
	}

	// Setup list route
	env.Router.GET("/api/v1/characters", func(c *gin.Context) {
		characters := []Character{}
		for _, char := range mockDB.characters {
			characters = append(characters, char)
		}
		c.JSON(200, gin.H{
			"characters": characters,
			"count":      len(characters),
		})
	})

	t.Run("ListLargeDataset", func(t *testing.T) {
		start := time.Now()
		w := makeHTTPRequest(env.Router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/characters",
		})
		duration := time.Since(start)

		if w.Code != 200 {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		t.Logf("Listed 100 characters in %v", duration)

		// Should be fast for in-memory operations
		maxTime := 500 * time.Millisecond
		if duration > maxTime {
			t.Errorf("List operation took too long: %v > %v", duration, maxTime)
		}
	})
}

// TestConsistencyCalculationPerformance tests character consistency scoring performance
func TestConsistencyCalculationPerformance(t *testing.T) {
	testChar := createTestCharacter(t, "ConsistencyPerfChar")

	t.Run("ConsistencyScoring", func(t *testing.T) {
		dialogs := []string{
			"I will fight with all my courage!",
			"This battle requires bravery and wisdom",
			"Let me think about this carefully...",
			"Haha, what a funny situation!",
			"We must be strategic in our approach",
		}

		start := time.Now()

		for i := 0; i < 1000; i++ {
			for _, dialog := range dialogs {
				_ = calculateCharacterConsistency(testChar.Character, dialog)
			}
		}

		duration := time.Since(start)
		iterations := 1000 * len(dialogs)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Calculated consistency %d times in %v", iterations, duration)
		t.Logf("Average time per calculation: %v", avgTime)

		// Should be very fast
		maxTime := 5 * time.Second
		if duration > maxTime {
			t.Errorf("Consistency calculation took too long: %v > %v", duration, maxTime)
		}
	})
}

// TestEmotionExtractionPerformance tests emotion extraction performance
func TestEmotionExtractionPerformance(t *testing.T) {
	t.Run("EmotionExtraction", func(t *testing.T) {
		dialogs := []string{
			"That's great! Awesome!",
			"What is happening here?",
			"Hmm... I wonder about this...",
			"No! Stop that right now!",
			"Hello there, friend",
		}

		start := time.Now()

		for i := 0; i < 10000; i++ {
			for _, dialog := range dialogs {
				_ = extractEmotionFromDialog(dialog, "")
			}
		}

		duration := time.Since(start)
		iterations := 10000 * len(dialogs)
		avgTime := duration / time.Duration(iterations)

		t.Logf("Extracted emotion %d times in %v", iterations, duration)
		t.Logf("Average time per extraction: %v", avgTime)

		// Should be extremely fast
		maxTime := 2 * time.Second
		if duration > maxTime {
			t.Errorf("Emotion extraction took too long: %v > %v", duration, maxTime)
		}
	})
}

// TestMemoryUsageUnderLoad tests memory usage patterns
func TestMemoryUsageUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	mockDB := NewMockDatabase()

	// Setup routes
	setupWorkflowRoutes(env.Router, mockDB)

	t.Run("SustainedLoad", func(t *testing.T) {
		duration := 5 * time.Second
		requestCount := 0
		errors := 0

		deadline := time.Now().Add(duration)

		for time.Now().Before(deadline) {
			// Create character
			charData := TestData.CharacterRequest(fmt.Sprintf("LoadChar%d", requestCount))
			w := makeHTTPRequest(env.Router, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/characters",
				Body:   charData,
			})

			if w.Code == 201 {
				requestCount++
			} else {
				errors++
			}

			// Small delay to prevent overwhelming
			time.Sleep(10 * time.Millisecond)
		}

		t.Logf("Processed %d requests in %v with %d errors", requestCount, duration, errors)

		if errors > requestCount/10 {
			t.Errorf("Too many errors under load: %d/%d", errors, requestCount)
		}
	})
}

// Helper function to setup dialog generation route for performance tests
func setupDialogGenerationRoute(router *gin.Engine, mockDB *MockDatabase) {
	router.POST("/api/v1/dialog/generate", func(c *gin.Context) {
		var req DialogGenerationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid dialog request"})
			return
		}

		characterID, err := uuid.Parse(req.CharacterID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Invalid character ID"})
			return
		}

		char, err := mockDB.GetCharacter(characterID.String())
		if err != nil {
			c.JSON(404, gin.H{"error": "Character not found"})
			return
		}

		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)

		response := DialogGenerationResponse{
			Dialog:                   fmt.Sprintf("Dialog for %s in %s", char.Name, req.SceneContext),
			Emotion:                  extractEmotionFromDialog("test", req.EmotionState),
			CharacterConsistencyScore: calculateCharacterConsistency(char, "test dialog"),
		}

		c.JSON(200, response)
	})

	router.POST("/api/v1/dialog/batch", func(c *gin.Context) {
		var req BatchDialogRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": "Invalid batch request"})
			return
		}

		var dialogSet []DialogGenerationResponse
		startTime := time.Now()

		for _, dialogReq := range req.DialogRequests {
			characterID, err := uuid.Parse(dialogReq.CharacterID)
			if err != nil {
				continue
			}

			char, err := mockDB.GetCharacter(characterID.String())
			if err != nil {
				continue
			}

			// Simulate processing
			time.Sleep(5 * time.Millisecond)

			dialogSet = append(dialogSet, DialogGenerationResponse{
				Dialog:                   fmt.Sprintf("Batch dialog for %s", char.Name),
				Emotion:                  "neutral",
				CharacterConsistencyScore: 0.85,
			})
		}

		metrics := map[string]interface{}{
			"total_requested": len(req.DialogRequests),
			"successful":      len(dialogSet),
			"duration_ms":     time.Since(startTime).Milliseconds(),
			"avg_consistency": calculateAverageConsistency(dialogSet),
		}

		response := BatchDialogResponse{
			DialogSet:         dialogSet,
			GenerationMetrics: metrics,
		}

		c.JSON(200, response)
	})
}
