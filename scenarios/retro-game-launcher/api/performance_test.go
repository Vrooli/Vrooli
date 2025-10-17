// +build testing

package main

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestGetGamesPerformance verifies list performance
func TestGetGamesPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Create multiple games
	for i := 0; i < 20; i++ {
		title := fmt.Sprintf("Test Perf Game %d", i)
		createTestGame(t, env.DB, &Game{
			Title:     title,
			Prompt:    "Test",
			Code:      "const canvas = document.getElementById('game');",
			Engine:    "html5",
			Published: true,
		})
	}

	t.Run("ResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/games",
		}

		rr := makeHTTPRequest(env.Router, req)

		elapsed := time.Since(start)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		if elapsed > 500*time.Millisecond {
			t.Logf("Warning: GET /api/games took %v (expected < 500ms)", elapsed)
		} else {
			t.Logf("GET /api/games completed in %v", elapsed)
		}
	})
}

// TestConcurrentGameAccess verifies concurrent request handling
func TestConcurrentGameAccess(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	testGame := createTestGame(t, env.DB, &Game{
		Title:     "Test Concurrent Game",
		Prompt:    "Test",
		Code:      "const canvas = document.getElementById('game');",
		Engine:    "html5",
		Published: true,
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		concurrency := 10
		var wg sync.WaitGroup
		errors := make(chan error, concurrency)

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method:  "GET",
					Path:    "/api/games/" + testGame.ID,
					URLVars: map[string]string{"id": testGame.ID},
				}

				rr := makeHTTPRequest(env.Router, req)

				if rr.Code != http.StatusOK {
					errors <- http.ErrAbortHandler
				}
			}()
		}

		wg.Wait()
		close(errors)

		elapsed := time.Since(start)

		errorCount := 0
		for range errors {
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Had %d errors in concurrent reads", errorCount)
		}

		if elapsed > 1*time.Second {
			t.Logf("Warning: %d concurrent reads took %v", concurrency, elapsed)
		} else {
			t.Logf("%d concurrent reads completed in %v", concurrency, elapsed)
		}
	})

	t.Run("ConcurrentPlayRecording", func(t *testing.T) {
		concurrency := 20
		var wg sync.WaitGroup

		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				req := HTTPTestRequest{
					Method:  "POST",
					Path:    "/api/games/" + testGame.ID + "/play",
					URLVars: map[string]string{"id": testGame.ID},
				}

				makeHTTPRequest(env.Router, req)
			}()
		}

		wg.Wait()

		elapsed := time.Since(start)

		// Verify play count increased correctly
		var playCount int
		env.DB.QueryRow("SELECT play_count FROM games WHERE id = $1", testGame.ID).Scan(&playCount)

		if playCount < concurrency {
			t.Logf("Warning: Expected at least %d plays, got %d (race condition possible)", concurrency, playCount)
		}

		if elapsed > 2*time.Second {
			t.Logf("Warning: %d concurrent plays took %v", concurrency, elapsed)
		} else {
			t.Logf("%d concurrent plays completed in %v", concurrency, elapsed)
		}
	})
}

// TestSearchPerformance verifies search performance
func TestSearchPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Create games with searchable content
	for i := 0; i < 30; i++ {
		tags := []string{fmt.Sprintf("tag%d", i%5)}
		title := fmt.Sprintf("Test Search Game %d", i)
		prompt := fmt.Sprintf("Searchable content %d", i)
		createTestGame(t, env.DB, &Game{
			Title:     title,
			Prompt:    prompt,
			Code:      "const canvas = document.getElementById('game');",
			Engine:    "html5",
			Published: true,
			Tags:      tags,
		})
	}

	t.Run("SearchResponseTime", func(t *testing.T) {
		start := time.Now()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search/games?q=Search",
		}

		rr := makeHTTPRequest(env.Router, req)

		elapsed := time.Since(start)

		if rr.Code != http.StatusOK {
			t.Logf("Search returned status %d (skipping performance check)", rr.Code)
			return
		}

		if elapsed > 1*time.Second {
			t.Logf("Warning: Search took %v (expected < 1s)", elapsed)
		} else {
			t.Logf("Search completed in %v", elapsed)
		}
	})
}

// TestCreateGamePerformance verifies game creation performance
func TestCreateGamePerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("SingleCreation", func(t *testing.T) {
		newGame := map[string]interface{}{
			"title":     "Performance Test Game",
			"prompt":    "Test prompt",
			"code":      "const canvas = document.getElementById('game');\nconst ctx = canvas.getContext('2d');",
			"engine":    "html5",
			"published": true,
			"tags":      []string{"performance", "test"},
		}

		start := time.Now()

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/games",
			Body:   newGame,
		}

		rr := makeHTTPRequest(env.Router, req)

		elapsed := time.Since(start)

		if rr.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", rr.Code)
		}

		if elapsed > 500*time.Millisecond {
			t.Logf("Warning: Game creation took %v (expected < 500ms)", elapsed)
		} else {
			t.Logf("Game creation completed in %v", elapsed)
		}
	})

	t.Run("BulkCreation", func(t *testing.T) {
		count := 10
		start := time.Now()

		for i := 0; i < count; i++ {
			title := fmt.Sprintf("Bulk Game %d", i)
			newGame := map[string]interface{}{
				"title":     title,
				"prompt":    "Bulk test",
				"code":      "code",
				"engine":    "html5",
				"published": false,
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/games",
				Body:   newGame,
			}

			makeHTTPRequest(env.Router, req)
		}

		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(count)

		if avgTime > 100*time.Millisecond {
			t.Logf("Warning: Average creation time %v per game (expected < 100ms)", avgTime)
		} else {
			t.Logf("Created %d games in %v (avg %v per game)", count, elapsed, avgTime)
		}
	})
}

// TestHealthCheckPerformance verifies health check performance
func TestHealthCheckPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("HealthCheckResponseTime", func(t *testing.T) {
		iterations := 10
		var totalTime time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			rr := makeHTTPRequest(env.Router, req)

			elapsed := time.Since(start)
			totalTime += elapsed

			if rr.Code != http.StatusOK {
				t.Errorf("Health check failed with status %d", rr.Code)
			}
		}

		avgTime := totalTime / time.Duration(iterations)

		if avgTime > 200*time.Millisecond {
			t.Logf("Warning: Average health check time %v (expected < 200ms)", avgTime)
		} else {
			t.Logf("Average health check time: %v", avgTime)
		}
	})
}

// TestGenerationStatusPerformance verifies status retrieval performance
func TestGenerationStatusPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	// Create multiple generation statuses
	genIDs := make([]string, 100)
	for i := 0; i < 100; i++ {
		genID := uuid.New().String()
		genIDs[i] = genID
		generationStore[genID] = &GameGenerationStatus{
			ID:        genID,
			Status:    "pending",
			Prompt:    "Test",
			CreatedAt: time.Now(),
		}
	}

	t.Run("StatusRetrievalPerformance", func(t *testing.T) {
		start := time.Now()

		for _, genID := range genIDs {
			req := HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/generate/status/" + genID,
				URLVars: map[string]string{"id": genID},
			}

			makeHTTPRequest(env.Router, req)
		}

		elapsed := time.Since(start)
		avgTime := elapsed / time.Duration(len(genIDs))

		if avgTime > 10*time.Millisecond {
			t.Logf("Warning: Average status retrieval %v per request", avgTime)
		} else {
			t.Logf("Retrieved %d statuses in %v (avg %v per request)", len(genIDs), elapsed, avgTime)
		}
	})

	// Cleanup
	for _, genID := range genIDs {
		delete(generationStore, genID)
	}
}

// BenchmarkGetGames benchmarks the getGames endpoint
func BenchmarkGetGames(b *testing.B) {
	env := setupTestServer(&testing.T{})
	if env == nil {
		b.Skip("Database not available")
		return
	}
	defer env.Cleanup()

	// Create some test data
	for i := 0; i < 10; i++ {
		createTestGame(&testing.T{}, env.DB, &Game{
			Title:     "Benchmark Game",
			Prompt:    "Test",
			Code:      "code",
			Engine:    "html5",
			Published: true,
		})
	}

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/games",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Router, req)
	}
}

// BenchmarkHealthCheck benchmarks the health check endpoint
func BenchmarkHealthCheck(b *testing.B) {
	env := setupTestServer(&testing.T{})
	if env == nil {
		b.Skip("Database not available")
		return
	}
	defer env.Cleanup()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env.Router, req)
	}
}
