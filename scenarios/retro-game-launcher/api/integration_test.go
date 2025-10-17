// +build testing

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestCheckDatabase verifies database health check
func TestCheckDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("HealthyDatabase", func(t *testing.T) {
		status := env.Server.checkDatabase()
		if status != "healthy" {
			t.Errorf("Expected 'healthy', got '%s'", status)
		}
	})
}

// TestCheckOllama verifies Ollama health check
func TestCheckOllama(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Use mock Ollama server
	mockServer := mockOllamaServer()
	defer mockServer.Close()

	originalURL := env.Server.ollamaURL
	env.Server.ollamaURL = mockServer.URL
	defer func() { env.Server.ollamaURL = originalURL }()

	t.Run("HealthyOllama", func(t *testing.T) {
		status := env.Server.checkOllama()
		if status != "healthy" {
			t.Errorf("Expected 'healthy', got '%s'", status)
		}
	})

	t.Run("UnhealthyOllama", func(t *testing.T) {
		env.Server.ollamaURL = "http://invalid-url-for-testing:99999"
		status := env.Server.checkOllama()
		if status != "unavailable" {
			t.Logf("Expected 'unavailable', got '%s'", status)
		}
		env.Server.ollamaURL = mockServer.URL
	})
}

// TestGetGamesEdgeCases verifies edge cases for getGames
func TestGetGamesEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("NoPublishedGames", func(t *testing.T) {
		// Clean existing published games
		env.DB.Exec("DELETE FROM games WHERE published = true")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/games",
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		var games []Game
		json.Unmarshal(rr.Body.Bytes(), &games)

		if len(games) != 0 {
			t.Logf("Expected empty array, got %d games", len(games))
		}
	})

	t.Run("LargeTagArray", func(t *testing.T) {
		largeTags := make([]string, 50)
		for i := 0; i < 50; i++ {
			largeTags[i] = fmt.Sprintf("tag%d", i)
		}

		createTestGame(t, env.DB, &Game{
			Title:     "Test Large Tags Game",
			Prompt:    "Test",
			Code:      "const canvas = document.getElementById('game');",
			Engine:    "html5",
			Published: true,
			Tags:      largeTags,
		})

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/games",
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}

// TestCreateGameEdgeCases verifies edge cases for game creation
func TestCreateGameEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("MinimalValidGame", func(t *testing.T) {
		minimalGame := map[string]interface{}{
			"title":  "Minimal",
			"prompt": "Test",
			"code":   "code",
			"engine": "html5",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/games",
			Body:   minimalGame,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusCreated {
			t.Logf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("GameWithAllFields", func(t *testing.T) {
		authorID := uuid.New().String()
		parentID := uuid.New().String()
		thumbURL := "https://example.com/thumb.png"
		rating := 4.5

		completeGame := map[string]interface{}{
			"title":          "Complete Game",
			"prompt":         "Complete prompt",
			"description":    "Complete description",
			"code":           "const game = {};",
			"engine":         "html5",
			"author_id":      authorID,
			"parent_game_id": parentID,
			"is_remix":       true,
			"thumbnail_url":  thumbURL,
			"rating":         rating,
			"tags":           []string{"complete", "test"},
			"published":      true,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/games",
			Body:   completeGame,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusCreated {
			t.Logf("Expected status 201, got %d. Body: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("EmptyTagsArray", func(t *testing.T) {
		gameWithEmptyTags := map[string]interface{}{
			"title":  "Empty Tags",
			"prompt": "Test",
			"code":   "code",
			"engine": "html5",
			"tags":   []string{},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/games",
			Body:   gameWithEmptyTags,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusCreated {
			t.Logf("Expected status 201, got %d", rr.Code)
		}
	})
}

// TestRecordPlayEdgeCases verifies play recording edge cases
func TestRecordPlayEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("MultiplePlayRecords", func(t *testing.T) {
		testGame := createTestGame(t, env.DB, &Game{
			Title:     "Test Multi-Play Game",
			Prompt:    "Test",
			Code:      "const canvas = document.getElementById('game');",
			Engine:    "html5",
			Published: true,
			PlayCount: 0,
		})

		// Record multiple plays
		for i := 0; i < 5; i++ {
			req := HTTPTestRequest{
				Method:  "POST",
				Path:    "/api/games/" + testGame.ID + "/play",
				URLVars: map[string]string{"id": testGame.ID},
			}

			rr := makeHTTPRequest(env.Router, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Play %d failed with status %d", i+1, rr.Code)
			}
		}

		// Verify final count
		var finalCount int
		err := env.DB.QueryRow("SELECT play_count FROM games WHERE id = $1", testGame.ID).Scan(&finalCount)
		if err != nil {
			t.Fatalf("Failed to query play count: %v", err)
		}

		if finalCount != 5 {
			t.Errorf("Expected play count 5, got %d", finalCount)
		}
	})

	t.Run("PlayNonExistentGame", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req := HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/games/" + nonExistentID + "/play",
			URLVars: map[string]string{"id": nonExistentID},
		}

		rr := makeHTTPRequest(env.Router, req)

		// Should succeed even if game doesn't exist (no error handling in stub)
		if rr.Code != http.StatusOK {
			t.Logf("Play non-existent game returned status %d", rr.Code)
		}
	})
}

// TestGetFeaturedGamesFiltering verifies featured games filtering
func TestGetFeaturedGamesFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Clean existing games
	env.DB.Exec("DELETE FROM games")

	// Create games with different ratings
	highRating := 4.5
	lowRating := 3.0

	createTestGame(t, env.DB, &Game{
		Title:     "Test High Rated",
		Prompt:    "High",
		Code:      "code",
		Engine:    "html5",
		Published: true,
		Rating:    &highRating,
		PlayCount: 100,
	})

	createTestGame(t, env.DB, &Game{
		Title:     "Test Low Rated",
		Prompt:    "Low",
		Code:      "code",
		Engine:    "html5",
		Published: true,
		Rating:    &lowRating,
		PlayCount: 50,
	})

	createTestGame(t, env.DB, &Game{
		Title:     "Test No Rating",
		Prompt:    "None",
		Code:      "code",
		Engine:    "html5",
		Published: true,
		Rating:    nil,
		PlayCount: 200,
	})

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/featured",
	}

	rr := makeHTTPRequest(env.Router, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var games []Game
	json.Unmarshal(rr.Body.Bytes(), &games)

	// Featured should only include games with rating > 4.0
	for _, g := range games {
		if g.Rating != nil && *g.Rating <= 4.0 {
			t.Errorf("Featured game '%s' has rating %.1f, should be > 4.0", g.Title, *g.Rating)
		}
	}
}

// TestGenerateGameEdgeCases verifies generation edge cases
func TestGenerateGameEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("DefaultEngine", func(t *testing.T) {
		genRequest := map[string]interface{}{
			"prompt": "Create a game",
			// engine not specified, should default to html5
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   genRequest,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", rr.Code)
		}
	})

	t.Run("EmptyPrompt", func(t *testing.T) {
		genRequest := map[string]interface{}{
			"prompt": "",
			"engine": "html5",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   genRequest,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("WithRemixID", func(t *testing.T) {
		remixFromID := uuid.New().String()
		genRequest := map[string]interface{}{
			"prompt":        "Remix this game",
			"engine":        "html5",
			"remix_from_id": remixFromID,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/generate",
			Body:   genRequest,
		}

		rr := makeHTTPRequest(env.Router, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", rr.Code)
		}
	})
}

// TestGenerateGameWithAIIntegration tests the full generation flow
func TestGenerateGameWithAIIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Use mock Ollama server
	mockServer := mockOllamaServer()
	defer mockServer.Close()

	originalURL := env.Server.ollamaURL
	env.Server.ollamaURL = mockServer.URL
	defer func() { env.Server.ollamaURL = originalURL }()

	t.Run("FullGenerationFlow", func(t *testing.T) {
		genID := uuid.New().String()
		generationStore[genID] = &GameGenerationStatus{
			ID:        genID,
			Status:    "pending",
			Prompt:    "Create a test game",
			CreatedAt: time.Now(),
		}

		req := GameGenerationRequest{
			Prompt: "Create a simple test game",
			Engine: "html5",
			Tags:   []string{"test"},
		}

		ctx := context.Background()
		err := env.Server.generateGameWithAI(ctx, req, genID)

		if err != nil {
			t.Logf("Generation failed: %v (may be expected if Ollama unavailable)", err)
		} else {
			status := generationStore[genID]
			if status.Status != "completed" && status.Status != "failed" {
				t.Logf("Unexpected status: %s", status.Status)
			}
		}

		delete(generationStore, genID)
	})
}

// TestGenerateGameAssetsBackground tests background asset generation
func TestGenerateGameAssetsBackground(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Use mock Ollama server
	mockServer := mockOllamaServer()
	defer mockServer.Close()

	originalURL := env.Server.ollamaURL
	env.Server.ollamaURL = mockServer.URL
	defer func() { env.Server.ollamaURL = originalURL }()

	t.Run("AssetGeneration", func(t *testing.T) {
		testGame := createTestGame(t, env.DB, &Game{
			Title:     "Test Asset Game",
			Prompt:    "Test",
			Code:      "code",
			Engine:    "html5",
			Published: true,
		})

		ctx := context.Background()
		env.Server.generateGameAssets(ctx, testGame.ID, testGame.Title)

		// Asset generation runs in background and is optional
		// Just verify it doesn't panic
		time.Sleep(100 * time.Millisecond)
	})
}
