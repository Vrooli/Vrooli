package main

import (
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
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

// Benchmarks retain comparable performance data without machine-specific pass/fail limits.
func BenchmarkHealthHandler(b *testing.B) {
	db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()
	handler := newHealthHandler(db)
	request := HTTPTestRequest{Method: http.MethodGet, Path: "/health"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mock.ExpectPing()
		w := serveRequest(b, handler, request)
		if w.Code != http.StatusOK {
			b.Fatalf("health status = %d, want %d", w.Code, http.StatusOK)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkCachedStoryHandler(b *testing.B) {
	story := &Story{ID: "benchmark-story", Title: "Benchmark Story", AgeGroup: "6-8"}
	storyCache.Set(story.ID, story)
	defer storyCache.Delete(story.ID)
	handler := getStoryHandler
	request := HTTPTestRequest{
		Method:  http.MethodGet,
		Path:    "/api/v1/stories/" + story.ID,
		URLVars: map[string]string{"id": story.ID},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := serveRequest(b, handler, request)
		if w.Code != http.StatusOK {
			b.Fatalf("cached story status = %d, want %d", w.Code, http.StatusOK)
		}
	}
}

// TestHelperFunctionPerformance duplicated these inputs with fixed 100µs ceilings.
// Benchmarks preserve the workload variants while allowing comparison across hosts.
func BenchmarkHelperFunctions(b *testing.B) {
	for _, tc := range []struct {
		name string
		call func()
	}{
		{name: "validate-age-group", call: func() { validateAgeGroup("6-8") }},
		{name: "age-description", call: func() { getAgeDescription("6-8") }},
		{name: "reading-time", call: func() { calculateReadingTime("medium") }},
	} {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				tc.call()
			}
		})
	}
	for _, tc := range []struct {
		name string
		text string
	}{
		{name: "page-count/small", text: "Once upon a time..."},
		{name: "page-count/medium", text: string(make([]byte, 1000))},
		{name: "page-count/large", text: string(make([]byte, 10000))},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				calculatePageCount(tc.text)
			}
		})
	}
	for _, tc := range []struct {
		name     string
		response string
	}{
		{name: "parse-response/title", response: "Title: The Magic Forest\n\nOnce upon a time in a magical forest..."},
		{name: "parse-response/no-title", response: "A story without a title marker"},
		{name: "parse-response/large", response: "Title: Adventure Time\n\n" + string(make([]byte, 5000))},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				parseStoryResponse(tc.response)
			}
		})
	}
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
