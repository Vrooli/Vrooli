package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestAPIServerPerformance tests API server performance under load
func TestAPIServerPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestServer(t)
	defer env.Cleanup()

	t.Run("ConcurrentExperimentCreation", func(t *testing.T) {
		numRequests := 50
		results := make(chan error, numRequests)
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				req := ExperimentRequest{
					Name:           fmt.Sprintf("Perf Test %d", index),
					Description:    "Performance test experiment",
					Prompt:         "Test prompt",
					TargetScenario: "test-scenario",
					NewResource:    "test-resource",
				}

				_, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/experiments",
					Body:   req,
				})
				results <- err
			}(i)
		}

		wg.Wait()
		close(results)
		elapsed := time.Since(start)

		failedRequests := 0
		for err := range results {
			if err != nil {
				failedRequests++
			}
		}

		if failedRequests > numRequests/10 { // Allow up to 10% failure
			t.Errorf("Too many failed requests: %d/%d", failedRequests, numRequests)
		}

		avgTime := elapsed / time.Duration(numRequests)
		t.Logf("Created %d experiments in %v (avg: %v per request)", numRequests-failedRequests, elapsed, avgTime)

		if avgTime > 500*time.Millisecond {
			t.Logf("Warning: Average request time %v exceeds 500ms", avgTime)
		}
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Create test data
		exp := createTestExperiment(t, env, "perf-read-test")

		numRequests := 100
		results := make(chan error, numRequests)
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < numRequests; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := makeHTTPRequest(env, HTTPTestRequest{
					Method:  "GET",
					Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
					URLVars: map[string]string{"id": exp.ID},
				})
				results <- err
			}()
		}

		wg.Wait()
		close(results)
		elapsed := time.Since(start)

		failedRequests := 0
		for err := range results {
			if err != nil {
				failedRequests++
			}
		}

		if failedRequests > 0 {
			t.Errorf("Failed requests: %d/%d", failedRequests, numRequests)
		}

		avgTime := elapsed / time.Duration(numRequests)
		t.Logf("Completed %d read requests in %v (avg: %v per request)", numRequests, elapsed, avgTime)

		if avgTime > 100*time.Millisecond {
			t.Logf("Warning: Average read time %v exceeds 100ms", avgTime)
		}
	})

	t.Run("LargeDatasetRetrieval", func(t *testing.T) {
		// Create large dataset
		numExperiments := 100
		for i := 0; i < numExperiments; i++ {
			createTestExperiment(t, env, fmt.Sprintf("large-dataset-%d", i))
		}

		start := time.Now()
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments",
		})
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to retrieve large dataset: %v", err)
		}

		var experiments []Experiment
		assertJSONResponse(t, w, 200, &experiments)

		if len(experiments) < numExperiments {
			t.Errorf("Expected at least %d experiments, got %d", numExperiments, len(experiments))
		}

		t.Logf("Retrieved %d experiments in %v", len(experiments), elapsed)

		if elapsed > 2*time.Second {
			t.Logf("Warning: Large dataset retrieval took %v (>2s)", elapsed)
		}
	})

	t.Run("DatabaseConnectionPool", func(t *testing.T) {
		// Test connection pool under heavy concurrent load
		numConcurrent := 30 // Exceeds default pool size of 25
		results := make(chan error, numConcurrent)
		start := time.Now()

		var wg sync.WaitGroup
		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// Create experiment
				req := ExperimentRequest{
					Name:           fmt.Sprintf("Pool Test %d", index),
					Description:    "Connection pool test",
					Prompt:         "Test",
					TargetScenario: "test",
					NewResource:    "test",
				}

				w, err := makeHTTPRequest(env, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/experiments",
					Body:   req,
				})

				if err != nil {
					results <- err
					return
				}

				if w.Code != 200 {
					results <- fmt.Errorf("unexpected status: %d", w.Code)
					return
				}

				results <- nil
			}(i)
		}

		wg.Wait()
		close(results)
		elapsed := time.Since(start)

		failedRequests := 0
		for err := range results {
			if err != nil {
				failedRequests++
			}
		}

		if failedRequests > numConcurrent/5 { // Allow up to 20% failure under stress
			t.Errorf("Too many connection pool failures: %d/%d", failedRequests, numConcurrent)
		}

		t.Logf("Connection pool handled %d concurrent requests in %v (%d failures)",
			numConcurrent, elapsed, failedRequests)
	})

	t.Run("MemoryUsageUnderLoad", func(t *testing.T) {
		// Test for memory leaks by creating/deleting many experiments
		numIterations := 50

		start := time.Now()
		for i := 0; i < numIterations; i++ {
			exp := createTestExperiment(t, env, fmt.Sprintf("memory-test-%d", i))

			// Immediately delete
			_, err := makeHTTPRequest(env, HTTPTestRequest{
				Method:  "DELETE",
				Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
				URLVars: map[string]string{"id": exp.ID},
			})

			if err != nil {
				t.Logf("Delete failed in iteration %d: %v", i, err)
			}
		}
		elapsed := time.Since(start)

		t.Logf("Created and deleted %d experiments in %v", numIterations, elapsed)

		// Verify cleanup
		var count int
		env.DB.QueryRow("SELECT COUNT(*) FROM experiments WHERE name LIKE 'memory-test-%'").Scan(&count)
		if count > 0 {
			t.Logf("Warning: %d experiments not deleted", count)
		}
	})

	t.Run("TemplateQueryPerformance", func(t *testing.T) {
		// Create many templates
		numTemplates := 50
		for i := 0; i < numTemplates; i++ {
			createTestTemplate(t, env, fmt.Sprintf("perf-template-%d", i))
		}

		// Measure query performance
		iterations := 10
		totalTime := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/templates",
			})
			elapsed := time.Since(start)
			totalTime += elapsed

			if err != nil {
				t.Fatalf("Template query failed: %v", err)
			}

			var templates []ExperimentTemplate
			assertJSONResponse(t, w, 200, &templates)

			if len(templates) < numTemplates {
				t.Errorf("Expected at least %d templates, got %d", numTemplates, len(templates))
			}
		}

		avgTime := totalTime / time.Duration(iterations)
		t.Logf("Average template query time over %d iterations: %v", iterations, avgTime)

		if avgTime > 500*time.Millisecond {
			t.Logf("Warning: Template query avg time %v exceeds 500ms", avgTime)
		}
	})

	t.Run("ScenarioQueryPerformance", func(t *testing.T) {
		// Create many scenarios
		numScenarios := 50
		for i := 0; i < numScenarios; i++ {
			createTestScenario(t, env, fmt.Sprintf("perf-scenario-%d", i))
		}

		// Measure query performance with JSON unmarshaling
		iterations := 10
		totalTime := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/scenarios",
			})
			elapsed := time.Since(start)
			totalTime += elapsed

			if err != nil {
				t.Fatalf("Scenario query failed: %v", err)
			}

			var scenarios []AvailableScenario
			assertJSONResponse(t, w, 200, &scenarios)

			if len(scenarios) < numScenarios {
				t.Errorf("Expected at least %d scenarios, got %d", numScenarios, len(scenarios))
			}
		}

		avgTime := totalTime / time.Duration(iterations)
		t.Logf("Average scenario query time over %d iterations: %v", iterations, avgTime)

		if avgTime > 500*time.Millisecond {
			t.Logf("Warning: Scenario query avg time %v exceeds 500ms", avgTime)
		}
	})

	t.Run("ExperimentLogPerformance", func(t *testing.T) {
		exp := createTestExperiment(t, env, "log-perf-test")

		// Create many logs
		numLogs := 100
		for i := 0; i < numLogs; i++ {
			logID := uuid.New().String()
			query := `INSERT INTO experiment_logs (id, experiment_id, step, prompt, success, started_at)
			          VALUES ($1, $2, $3, $4, $5, NOW())`
			env.DB.Exec(query, logID, exp.ID, fmt.Sprintf("step-%d", i), "test prompt", i%2 == 0)
		}

		// Measure retrieval performance
		start := time.Now()
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s/logs", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
		elapsed := time.Since(start)

		if err != nil {
			t.Fatalf("Log retrieval failed: %v", err)
		}

		var logs []ExperimentLog
		assertJSONResponse(t, w, 200, &logs)

		if len(logs) != numLogs {
			t.Errorf("Expected %d logs, got %d", numLogs, len(logs))
		}

		t.Logf("Retrieved %d logs in %v", numLogs, elapsed)

		if elapsed > 1*time.Second {
			t.Logf("Warning: Log retrieval took %v (>1s)", elapsed)
		}
	})

	t.Run("UpdateOperationPerformance", func(t *testing.T) {
		// Test update operation performance
		numExperiments := 20
		experiments := make([]*Experiment, numExperiments)

		for i := 0; i < numExperiments; i++ {
			experiments[i] = createTestExperiment(t, env, fmt.Sprintf("update-perf-%d", i))
		}

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < numExperiments; i++ {
			wg.Add(1)
			go func(exp *Experiment) {
				defer wg.Done()

				updates := map[string]interface{}{
					"status": "completed",
				}

				makeHTTPRequest(env, HTTPTestRequest{
					Method:  "PUT",
					Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
					URLVars: map[string]string{"id": exp.ID},
					Body:    updates,
				})
			}(experiments[i])
		}

		wg.Wait()
		elapsed := time.Since(start)

		t.Logf("Updated %d experiments concurrently in %v", numExperiments, elapsed)

		avgTime := elapsed / time.Duration(numExperiments)
		if avgTime > 200*time.Millisecond {
			t.Logf("Warning: Average update time %v exceeds 200ms", avgTime)
		}
	})
}

// BenchmarkExperimentCreation benchmarks experiment creation
func BenchmarkExperimentCreation(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	req := ExperimentRequest{
		Name:           "Benchmark Experiment",
		Description:    "Benchmark test",
		Prompt:         "Test prompt",
		TargetScenario: "test-scenario",
		NewResource:    "test-resource",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/experiments",
			Body:   req,
		})
	}
}

// BenchmarkExperimentRetrieval benchmarks experiment retrieval
func BenchmarkExperimentRetrieval(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	exp := createTestExperiment(&testing.T{}, env, "benchmark-retrieve")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method:  "GET",
			Path:    fmt.Sprintf("/api/experiments/%s", exp.ID),
			URLVars: map[string]string{"id": exp.ID},
		})
	}
}

// BenchmarkListExperiments benchmarks listing experiments
func BenchmarkListExperiments(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	// Create test data
	for i := 0; i < 10; i++ {
		createTestExperiment(&testing.T{}, env, fmt.Sprintf("bench-list-%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/experiments",
		})
	}
}

// BenchmarkJSONUnmarshal benchmarks JSON unmarshaling performance
func BenchmarkJSONUnmarshal(b *testing.B) {
	env := setupTestServer(&testing.T{})
	defer env.Cleanup()

	createTestScenario(&testing.T{}, env, "bench-json")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w, _ := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/scenarios",
		})

		var scenarios []AvailableScenario
		assertJSONResponse(&testing.T{}, w, 200, &scenarios)
	}
}
