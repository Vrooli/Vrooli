package main

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCacheConcurrency tests cache operations under concurrent access
func TestCacheConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ConcurrentReads", func(t *testing.T) {
		cache := &StoryCache{
			stories: make(map[string]*Story),
			maxSize: 100,
		}

		// Populate cache
		stories := make([]*TestStory, 10)
		for i := 0; i < 10; i++ {
			story := setupTestStory(t, "Concurrent Story", "6-8")
			defer story.Cleanup()
			stories[i] = story
			cache.Set(story.Story.ID, story.Story)
		}

		// Concurrent reads
		const goroutines = 50
		const iterations = 100

		var wg sync.WaitGroup
		wg.Add(goroutines)

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				storyIdx := idx % len(stories)
				for j := 0; j < iterations; j++ {
					_, ok := cache.Get(stories[storyIdx].Story.ID)
					assert.True(t, ok)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Concurrent reads completed in %v", duration)
		t.Logf("Operations per second: %.2f", float64(goroutines*iterations)/duration.Seconds())
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		cache := &StoryCache{
			stories: make(map[string]*Story),
			maxSize: 100,
		}

		const goroutines = 20
		const iterations = 50

		var wg sync.WaitGroup
		wg.Add(goroutines)

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					story := setupTestStory(t, "Write Story", "6-8")
					defer story.Cleanup()
					cache.Set(story.Story.ID, story.Story)
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Concurrent writes completed in %v", duration)
		t.Logf("Operations per second: %.2f", float64(goroutines*iterations)/duration.Seconds())

		// Verify cache size limit
		assert.LessOrEqual(t, len(cache.stories), cache.maxSize)
	})

	t.Run("MixedOperations", func(t *testing.T) {
		cache := &StoryCache{
			stories: make(map[string]*Story),
			maxSize: 100,
		}

		// Pre-populate some stories
		stories := make([]*TestStory, 10)
		for i := 0; i < 10; i++ {
			story := setupTestStory(t, "Mixed Story", "6-8")
			defer story.Cleanup()
			stories[i] = story
			cache.Set(story.Story.ID, story.Story)
		}

		const goroutines = 30
		const iterations = 50

		var wg sync.WaitGroup
		wg.Add(goroutines)

		start := time.Now()

		for i := 0; i < goroutines; i++ {
			go func(idx int) {
				defer wg.Done()
				for j := 0; j < iterations; j++ {
					switch idx % 3 {
					case 0: // Read
						storyIdx := (idx + j) % len(stories)
						cache.Get(stories[storyIdx].Story.ID)
					case 1: // Write
						story := setupTestStory(t, "New Story", "6-8")
						defer story.Cleanup()
						cache.Set(story.Story.ID, story.Story)
					case 2: // Delete
						storyIdx := (idx + j) % len(stories)
						cache.Delete(stories[storyIdx].Story.ID)
					}
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		t.Logf("Mixed operations completed in %v", duration)
		t.Logf("Operations per second: %.2f", float64(goroutines*iterations)/duration.Seconds())
	})
}

// TestHandlerPerformance tests handler response times
func TestHandlerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthCheckLatency", func(t *testing.T) {
		const iterations = 1000

		start := time.Now()

		for i := 0; i < iterations; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			healthHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		duration := time.Since(start)
		avgLatency := duration / iterations

		t.Logf("Average health check latency: %v", avgLatency)
		t.Logf("Requests per second: %.2f", float64(iterations)/duration.Seconds())

		// Assert reasonable performance (adjust threshold as needed)
		assert.Less(t, avgLatency, 10*time.Millisecond, "Health check should be fast")
	})

	t.Run("CachedStoryRetrievalLatency", func(t *testing.T) {
		story := setupTestStory(t, "Performance Story", "6-8")
		defer story.Cleanup()

		// Warm up cache
		storyCache.Set(story.Story.ID, story.Story)

		const iterations = 1000

		start := time.Now()

		for i := 0; i < iterations; i++ {
			w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/stories/" + story.Story.ID,
				URLVars: map[string]string{"id": story.Story.ID},
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			getStoryHandler(w, httpReq)

			if w.Code != 200 {
				t.Errorf("Expected status 200, got %d", w.Code)
			}
		}

		duration := time.Since(start)
		avgLatency := duration / iterations

		t.Logf("Average cached story retrieval latency: %v", avgLatency)
		t.Logf("Requests per second: %.2f", float64(iterations)/duration.Seconds())

		// Cached reads should be very fast
		assert.Less(t, avgLatency, 5*time.Millisecond, "Cached story retrieval should be very fast")
	})
}

// TestHelperFunctionPerformance tests performance of utility functions
func TestHelperFunctionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("calculatePageCountPerformance", func(t *testing.T) {
		// Generate test content of varying sizes
		testCases := []struct {
			name    string
			content string
		}{
			{"Small", "Once upon a time..."},
			{"Medium", string(make([]byte, 1000))},
			{"Large", string(make([]byte, 10000))},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				const iterations = 10000

				start := time.Now()

				for i := 0; i < iterations; i++ {
					calculatePageCount(tc.content)
				}

				duration := time.Since(start)
				avgLatency := duration / iterations

				t.Logf("%s: Average calculatePageCount latency: %v", tc.name, avgLatency)
				assert.Less(t, avgLatency, 100*time.Microsecond, "calculatePageCount should be fast")
			})
		}
	})

	t.Run("parseStoryResponsePerformance", func(t *testing.T) {
		responses := []string{
			"Title: The Magic Forest\n\nOnce upon a time in a magical forest...",
			"A story without a title marker",
			"Title: Adventure Time\n\n" + string(make([]byte, 5000)),
		}

		for i, response := range responses {
			t.Run(string(rune('A'+i)), func(t *testing.T) {
				const iterations = 10000

				start := time.Now()

				for j := 0; j < iterations; j++ {
					parseStoryResponse(response)
				}

				duration := time.Since(start)
				avgLatency := duration / iterations

				t.Logf("Average parseStoryResponse latency: %v", avgLatency)
				assert.Less(t, avgLatency, 100*time.Microsecond, "parseStoryResponse should be fast")
			})
		}
	})
}

// BenchmarkStoryCache benchmarks cache operations
func BenchmarkStoryCache(b *testing.B) {
	cache := &StoryCache{
		stories: make(map[string]*Story),
		maxSize: 100,
	}

	story := &Story{
		ID:      "test-id",
		Title:   "Benchmark Story",
		Content: "Benchmark content",
	}

	b.Run("Get", func(b *testing.B) {
		cache.Set(story.ID, story)
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			cache.Get(story.ID)
		}
	})

	b.Run("Set", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set(story.ID, story)
		}
	})

	b.Run("Delete", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cache.Set(story.ID, story)
			cache.Delete(story.ID)
		}
	})
}

// BenchmarkHelperFunctions benchmarks utility functions
func BenchmarkHelperFunctions(b *testing.B) {
	b.Run("validateAgeGroup", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			validateAgeGroup("6-8")
		}
	})

	b.Run("getAgeDescription", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			getAgeDescription("6-8")
		}
	})

	b.Run("calculateReadingTime", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			calculateReadingTime("medium")
		}
	})

	b.Run("calculatePageCount", func(b *testing.B) {
		content := string(make([]byte, 1000))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			calculatePageCount(content)
		}
	})

	b.Run("parseStoryResponse", func(b *testing.B) {
		response := "Title: Test Story\n\nOnce upon a time..."
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			parseStoryResponse(response)
		}
	})
}
