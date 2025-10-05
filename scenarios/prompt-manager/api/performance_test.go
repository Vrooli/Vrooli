package main

import (
	"sync"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestPerformance_HealthCheck benchmarks health check performance
func TestPerformance_HealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	pattern := PerformanceTestPattern{
		Name:        "HealthCheck",
		Description: "Test health check endpoint performance",
		MaxDuration: 50 * time.Millisecond,
		Iterations:  100,
		Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}
		},
		Validate: func(t *testing.T, avgDuration time.Duration, setupData interface{}) {
			t.Logf("Health check average duration: %v", avgDuration)
		},
	}

	RunPerformanceTest(t, server, router, pattern)
}

// TestPerformance_ListCampaigns benchmarks campaign listing
func TestPerformance_ListCampaigns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	// Create multiple campaigns for realistic testing
	for i := 0; i < 20; i++ {
		createTestCampaign(t, db, "Performance Test Campaign "+string(rune(i)))
	}

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/campaigns", server.getCampaigns).Methods("GET")

	pattern := PerformanceTestPattern{
		Name:        "ListCampaigns",
		Description: "Test campaign listing performance",
		MaxDuration: 100 * time.Millisecond,
		Iterations:  100,
		Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/campaigns",
			}
		},
		Validate: func(t *testing.T, avgDuration time.Duration, setupData interface{}) {
			if avgDuration > 100*time.Millisecond {
				t.Errorf("Campaign listing too slow: %v", avgDuration)
			}
		},
	}

	RunPerformanceTest(t, server, router, pattern)
}

// TestPerformance_SearchPrompts benchmarks search performance
func TestPerformance_SearchPrompts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Search Performance Campaign")

	// Create multiple prompts for realistic search testing
	for i := 0; i < 50; i++ {
		createTestPrompt(t, db, campaign.ID,
			"Performance Test Prompt "+string(rune(i)),
			"This is test content for performance testing of search functionality")
	}

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/search/prompts", server.searchPrompts).Methods("GET")

	pattern := PerformanceTestPattern{
		Name:        "SearchPrompts",
		Description: "Test prompt search performance",
		MaxDuration: 200 * time.Millisecond,
		Iterations:  50,
		Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
			return HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/search/prompts?q=performance",
			}
		},
		Validate: func(t *testing.T, avgDuration time.Duration, setupData interface{}) {
			if avgDuration > 200*time.Millisecond {
				t.Errorf("Search too slow: %v", avgDuration)
			}
		},
	}

	RunPerformanceTest(t, server, router, pattern)
}

// TestPerformance_ConcurrentPromptCreation tests concurrent write performance
func TestPerformance_ConcurrentPromptCreation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	campaign := createTestCampaign(t, db, "Concurrent Test Campaign")

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/prompts", server.createPrompt).Methods("POST")

	t.Run("ConcurrentWrites", func(t *testing.T) {
		concurrency := 10
		iterationsPerWorker := 5

		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < iterationsPerWorker; j++ {
					makeHTTPRequest(server, router, HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/prompts",
						Body: map[string]interface{}{
							"campaign_id": campaign.ID,
							"title":       "Concurrent Prompt",
							"content":     "Concurrent test content",
						},
					})
				}
			}(i)
		}

		wg.Wait()
		duration := time.Since(start)

		totalOps := concurrency * iterationsPerWorker
		avgPerOp := duration / time.Duration(totalOps)

		t.Logf("Concurrent creation: %d operations in %v (avg %v/op)",
			totalOps, duration, avgPerOp)

		if avgPerOp > 100*time.Millisecond {
			t.Errorf("Concurrent operations too slow: %v per operation", avgPerOp)
		}
	})
}

// TestPerformance_DatabaseConnectionPool tests connection pool efficiency
func TestPerformance_DatabaseConnectionPool(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	t.Run("ConnectionPoolEfficiency", func(t *testing.T) {
		iterations := 100
		var wg sync.WaitGroup
		start := time.Now()

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				var count int
				db.QueryRow("SELECT COUNT(*) FROM campaigns").Scan(&count)
			}()
		}

		wg.Wait()
		duration := time.Since(start)

		avgPerQuery := duration / time.Duration(iterations)
		t.Logf("Connection pool: %d queries in %v (avg %v/query)",
			iterations, duration, avgPerQuery)

		stats := db.Stats()
		t.Logf("Pool stats: Open=%d, InUse=%d, Idle=%d, MaxOpen=%d",
			stats.OpenConnections, stats.InUse, stats.Idle, stats.MaxOpenConnections)

		if avgPerQuery > 10*time.Millisecond {
			t.Errorf("Query too slow: %v", avgPerQuery)
		}
	})
}

// TestPerformance_ExportLargeDataset tests export performance with large datasets
func TestPerformance_ExportLargeDataset(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	db, dbCleanup := setupTestDB(t)
	defer dbCleanup()

	setupTestTables(t, db)

	// Create larger dataset
	campaigns := make([]*Campaign, 10)
	for i := 0; i < 10; i++ {
		campaigns[i] = createTestCampaign(t, db, "Export Test Campaign "+string(rune(i)))

		// Add prompts to each campaign
		for j := 0; j < 10; j++ {
			createTestPrompt(t, db, campaigns[i].ID,
				"Prompt "+string(rune(j)),
				"Test content for export performance testing")
		}
	}

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/export", server.exportData).Methods("GET")

	t.Run("ExportPerformance", func(t *testing.T) {
		start := time.Now()

		w := makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export",
		})

		duration := time.Since(start)

		if w.Code != 200 {
			t.Fatalf("Export failed with status %d", w.Code)
		}

		t.Logf("Export of 10 campaigns and 100 prompts took: %v", duration)

		if duration > 1*time.Second {
			t.Errorf("Export too slow: %v", duration)
		}
	})
}

// BenchmarkHealthCheck provides Go benchmark for health check
func BenchmarkHealthCheck(b *testing.B) {
	db, dbCleanup := setupTestDB(&testing.T{})
	defer dbCleanup()

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", server.healthCheck).Methods("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
	}
}

// BenchmarkListCampaigns provides Go benchmark for campaign listing
func BenchmarkListCampaigns(b *testing.B) {
	db, dbCleanup := setupTestDB(&testing.T{})
	defer dbCleanup()

	setupTestTables(&testing.T{}, db)

	for i := 0; i < 20; i++ {
		createTestCampaign(&testing.T{}, db, "Bench Campaign "+string(rune(i)))
	}

	server := &APIServer{
		db:        db,
		qdrantURL: "",
		ollamaURL: "",
	}

	router := mux.NewRouter()
	v1 := router.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/campaigns", server.getCampaigns).Methods("GET")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(server, router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/campaigns",
		})
	}
}
