package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPerformance_GetTasks benchmarks the GetTasks endpoint
func TestPerformance_GetTasks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	// Setup: Create test data
	appID := createTestApp(t, env, "perf-test-app")
	defer cleanupTestData(t, env)

	// Create 100 test tasks
	for i := 0; i < 100; i++ {
		status := "backlog"
		if i%3 == 0 {
			status = "in_progress"
		} else if i%3 == 1 {
			status = "completed"
		}
		createTestTask(t, env, appID, fmt.Sprintf("Performance Test Task %d", i), status)
	}

	t.Run("GetTasks_ResponseTime", func(t *testing.T) {
		iterations := 50
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/tasks",
				QueryParams: map[string]string{
					"limit":  "50",
					"offset": "0",
				},
			})
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Fatalf("Expected 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time for GetTasks: %v", avgDuration)

		// Performance threshold: should complete in under 500ms on average
		if avgDuration > 500*time.Millisecond {
			t.Errorf("GetTasks too slow: average %v, expected < 500ms", avgDuration)
		}
	})

	t.Run("GetTasks_Throughput", func(t *testing.T) {
		concurrentRequests := 10
		requestsPerWorker := 10

		start := time.Now()
		var wg sync.WaitGroup

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for j := 0; j < requestsPerWorker; j++ {
					w, err := makeHTTPRequest(env, HTTPTestRequest{
						Method: "GET",
						Path:   "/api/tasks",
					})
					if err != nil {
						t.Logf("Worker %d request %d failed: %v", workerID, j, err)
						return
					}
					if w.Code != 200 {
						t.Logf("Worker %d request %d got status %d", workerID, j, w.Code)
					}
				}
			}(i)
		}

		wg.Wait()
		totalTime := time.Since(start)

		totalRequests := concurrentRequests * requestsPerWorker
		requestsPerSecond := float64(totalRequests) / totalTime.Seconds()

		t.Logf("Throughput: %.2f requests/second (%d requests in %v)",
			requestsPerSecond, totalRequests, totalTime)

		// Should handle at least 10 requests/second
		if requestsPerSecond < 10 {
			t.Errorf("Low throughput: %.2f req/s, expected > 10 req/s", requestsPerSecond)
		}
	})
}

// TestPerformance_GetApps benchmarks the GetApps endpoint
func TestPerformance_GetApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	t.Run("GetApps_ResponseTime", func(t *testing.T) {
		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/apps",
			})
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Fatalf("Expected 200, got %d", w.Code)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time for GetApps: %v", avgDuration)

		// Should be very fast - under 100ms
		if avgDuration > 100*time.Millisecond {
			t.Errorf("GetApps too slow: average %v, expected < 100ms", avgDuration)
		}
	})
}

// TestPerformance_UpdateTaskStatus benchmarks status updates
func TestPerformance_UpdateTaskStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "perf-test-app")
	defer cleanupTestData(t, env)

	t.Run("UpdateTaskStatus_ResponseTime", func(t *testing.T) {
		iterations := 20
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			// Create a fresh task for each update
			taskID := createTestTask(t, env, appID, fmt.Sprintf("Perf Task %d", i), "backlog")

			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "PUT",
				Path:   "/api/tasks/status",
				Body: map[string]interface{}{
					"task_id":   taskID.String(),
					"to_status": "in_progress",
					"reason":    "Performance test",
				},
			})
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Logf("Status update got status %d: %s", w.Code, w.Body.String())
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time for UpdateTaskStatus: %v", avgDuration)

		// Status updates should complete in under 500ms
		if avgDuration > 500*time.Millisecond {
			t.Errorf("UpdateTaskStatus too slow: average %v, expected < 500ms", avgDuration)
		}
	})
}

// TestPerformance_GetTaskStatusHistory benchmarks history retrieval
func TestPerformance_GetTaskStatusHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "perf-test-app")
	defer cleanupTestData(t, env)

	// Create a task with some history
	taskID := createTestTask(t, env, appID, "Task With History", "backlog")

	// Create some status transitions
	transitions := []string{"in_progress", "review", "completed"}
	for _, status := range transitions {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": status,
				"reason":    "Creating history",
			},
		})
	}

	t.Run("GetStatusHistory_ResponseTime", func(t *testing.T) {
		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			w, err := makeHTTPRequest(env, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/tasks/status-history",
				QueryParams: map[string]string{
					"task_id": taskID.String(),
				},
			})
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if w.Code != 200 {
				t.Logf("Got status %d: %s", w.Code, w.Body.String())
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average response time for GetStatusHistory: %v", avgDuration)

		// History retrieval should be fast - under 100ms
		if avgDuration > 100*time.Millisecond {
			t.Errorf("GetStatusHistory too slow: average %v, expected < 100ms", avgDuration)
		}
	})
}

// TestPerformance_ParseAITaskResponse benchmarks AI response parsing
func TestPerformance_ParseAITaskResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Create a realistic AI response with 10 tasks
	aiResponse := `[`
	for i := 0; i < 10; i++ {
		if i > 0 {
			aiResponse += ","
		}
		aiResponse += fmt.Sprintf(`{
			"title": "Test Task %d",
			"description": "This is a test task description with some detail about what needs to be done for task number %d",
			"priority": "medium",
			"tags": ["test", "automated", "task-%d"],
			"estimated_hours": 2.5
		}`, i, i, i)
	}
	aiResponse += `]`

	t.Run("ParseAITaskResponse_Performance", func(t *testing.T) {
		iterations := 1000
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			tasks, err := env.Service.parseAITaskResponse(aiResponse)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if len(tasks) != 10 {
				t.Fatalf("Expected 10 tasks, got %d", len(tasks))
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average parse time: %v", avgDuration)

		// Parsing should be very fast - under 1ms
		if avgDuration > 1*time.Millisecond {
			t.Errorf("Parsing too slow: average %v, expected < 1ms", avgDuration)
		}
	})
}

// TestPerformance_ParseResearchResults benchmarks research parsing
func TestPerformance_ParseResearchResults(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	taskID := uuid.New()
	aiResponse := `{
		"research_summary": "This task involves implementing a comprehensive new feature with multiple components and dependencies",
		"requirements": ["req1", "req2", "req3", "req4", "req5"],
		"dependencies": ["dep1", "dep2", "dep3"],
		"recommendations": ["rec1", "rec2", "rec3", "rec4"],
		"estimated_hours": 12.5,
		"complexity": "high",
		"technical_considerations": ["consideration1", "consideration2", "consideration3"],
		"potential_challenges": ["challenge1", "challenge2"],
		"success_criteria": ["criteria1", "criteria2", "criteria3"]
	}`

	t.Run("ParseResearchResults_Performance", func(t *testing.T) {
		iterations := 1000
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			result, err := env.Service.parseResearchResults(aiResponse, taskID)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if result == nil {
				t.Fatal("Expected result, got nil")
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average parse time: %v", avgDuration)

		// Parsing should be very fast - under 1ms
		if avgDuration > 1*time.Millisecond {
			t.Errorf("Parsing too slow: average %v, expected < 1ms", avgDuration)
		}
	})
}

// TestPerformance_DatabaseQueries benchmarks database operations
func TestPerformance_DatabaseQueries(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "perf-test-app")
	defer cleanupTestData(t, env)

	taskID := createTestTask(t, env, appID, "Performance Test Task", "backlog")

	t.Run("GetTaskByID_Performance", func(t *testing.T) {
		iterations := 500
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			task, err := env.Service.getTaskByID(taskID)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if task == nil {
				t.Fatal("Expected task, got nil")
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average query time for GetTaskByID: %v", avgDuration)

		// Database queries should be fast - under 10ms
		if avgDuration > 10*time.Millisecond {
			t.Errorf("GetTaskByID too slow: average %v, expected < 10ms", avgDuration)
		}
	})

	t.Run("GetAppByID_Performance", func(t *testing.T) {
		iterations := 500
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			app, err := env.Service.getAppByID(appID)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Fatalf("Query failed: %v", err)
			}

			if app == nil {
				t.Fatal("Expected app, got nil")
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average query time for GetAppByID: %v", avgDuration)

		// Database queries should be fast - under 10ms
		if avgDuration > 10*time.Millisecond {
			t.Errorf("GetAppByID too slow: average %v, expected < 10ms", avgDuration)
		}
	})

	t.Run("FindRelatedTasks_Performance", func(t *testing.T) {
		// Create some tasks for better results
		for i := 0; i < 10; i++ {
			createTestTask(t, env, appID, fmt.Sprintf("Related Task %d", i), "backlog")
		}

		task, err := env.Service.getTaskByID(taskID)
		if err != nil {
			t.Skipf("Failed to get task: %v", err)
		}

		iterations := 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, err := env.Service.findRelatedTasks(task)
			duration := time.Since(start)
			totalDuration += duration

			if err != nil {
				t.Logf("Query failed: %v", err)
			}
		}

		avgDuration := totalDuration / time.Duration(iterations)
		t.Logf("Average query time for FindRelatedTasks: %v", avgDuration)

		// More complex query - should still be under 50ms
		if avgDuration > 50*time.Millisecond {
			t.Errorf("FindRelatedTasks too slow: average %v, expected < 50ms", avgDuration)
		}
	})
}

// TestPerformance_MemoryUsage tests memory consumption
func TestPerformance_MemoryUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "perf-test-app")
	defer cleanupTestData(t, env)

	t.Run("LargeResultSet_Memory", func(t *testing.T) {
		// Create many tasks
		for i := 0; i < 200; i++ {
			createTestTask(t, env, appID, fmt.Sprintf("Memory Test Task %d", i), "backlog")
		}

		// Fetch all tasks
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
			QueryParams: map[string]string{
				"limit": "200",
			},
		})

		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		// Verify response size is reasonable
		responseSize := len(w.Body.Bytes())
		t.Logf("Response size for 200 tasks: %d bytes (%.2f KB)",
			responseSize, float64(responseSize)/1024)

		// Response should be under 1MB for 200 tasks
		if responseSize > 1024*1024 {
			t.Errorf("Response too large: %d bytes, expected < 1MB", responseSize)
		}
	})
}

// BenchmarkGetTasks provides standard Go benchmark
func BenchmarkGetTasks(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	appID := createTestApp(&testing.T{}, env, "bench-app")
	for i := 0; i < 50; i++ {
		createTestTask(&testing.T{}, env, appID, fmt.Sprintf("Bench Task %d", i), "backlog")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
		})
	}
}

// BenchmarkUpdateTaskStatus provides standard Go benchmark
func BenchmarkUpdateTaskStatus(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	appID := createTestApp(&testing.T{}, env, "bench-app")

	// Pre-create tasks
	taskIDs := make([]uuid.UUID, b.N)
	for i := 0; i < b.N; i++ {
		taskIDs[i] = createTestTask(&testing.T{}, env, appID, fmt.Sprintf("Bench Task %d", i), "backlog")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskIDs[i].String(),
				"to_status": "in_progress",
			},
		})
	}
}

// BenchmarkParseAITaskResponse provides standard Go benchmark
func BenchmarkParseAITaskResponse(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(&testing.T{})
	defer env.Cleanup()

	aiResponse := `[
		{"title": "Task 1", "description": "Desc", "priority": "high", "tags": ["test"], "estimated_hours": 2.0},
		{"title": "Task 2", "description": "Desc", "priority": "medium", "tags": ["test"], "estimated_hours": 1.5},
		{"title": "Task 3", "description": "Desc", "priority": "low", "tags": ["test"], "estimated_hours": 0.5}
	]`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		env.Service.parseAITaskResponse(aiResponse)
	}
}
