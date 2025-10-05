// +build testing

package main

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// TestPerformanceTaskOperations tests performance of task operations
func TestPerformanceTaskOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	app := createTestApp()

	t.Run("CreateTask_Performance", func(t *testing.T) {
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			taskReq := TestData.CreateTaskRequest(
				fmt.Sprintf("Performance Task %d", i),
				"bug-fix",
				"test-target",
			)

			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "POST",
				Path:   "/api/tasks",
				Body:   taskReq,
			})
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			if w.Code != 201 {
				t.Errorf("Request %d: Expected 201, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Created %d tasks in %v (avg: %v per task)", iterations, duration, avgDuration)

		// Performance threshold: 100ms per task creation
		if avgDuration > 100*time.Millisecond {
			t.Errorf("Task creation too slow: %v per task (threshold: 100ms)", avgDuration)
		}
	})

	t.Run("ReadTask_Performance", func(t *testing.T) {
		// Create test tasks first
		taskCount := 50
		for i := 0; i < taskCount; i++ {
			testTask := setupTestTask(t, env, "feature", "backlog")
			defer testTask.Cleanup()
		}

		iterations := 200
		start := time.Now()

		for i := 0; i < iterations; i++ {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "GET",
				Path:   "/api/tasks",
			})
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			if w.Code != 200 {
				t.Errorf("Request %d: Expected 200, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Read tasks %d times in %v (avg: %v per read)", iterations, duration, avgDuration)

		// Performance threshold: 50ms per task read
		if avgDuration > 50*time.Millisecond {
			t.Errorf("Task read too slow: %v per read (threshold: 50ms)", avgDuration)
		}
	})

	t.Run("UpdateTask_Performance", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		defer testTask.Cleanup()

		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			updateReq := map[string]interface{}{
				"notes": fmt.Sprintf("Update iteration %d", i),
			}

			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "PUT",
				Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
				Body:   updateReq,
			})
			if err != nil {
				t.Fatalf("Request %d failed: %v", i, err)
			}

			if w.Code != 200 {
				t.Errorf("Request %d: Expected 200, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Updated task %d times in %v (avg: %v per update)", iterations, duration, avgDuration)

		// Performance threshold: 50ms per update
		if avgDuration > 50*time.Millisecond {
			t.Errorf("Task update too slow: %v per update (threshold: 50ms)", avgDuration)
		}
	})

	t.Run("DeleteTask_Performance", func(t *testing.T) {
		// Create tasks to delete
		taskCount := 100
		tasks := make([]*TestTask, taskCount)
		for i := 0; i < taskCount; i++ {
			tasks[i] = setupTestTask(t, env, "bug-fix", "completed")
		}

		start := time.Now()

		for i, task := range tasks {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "DELETE",
				Path:   fmt.Sprintf("/api/tasks/%s", task.Task.ID),
			})
			if err != nil {
				t.Fatalf("Delete %d failed: %v", i, err)
			}

			if w.Code != 200 {
				t.Errorf("Delete %d: Expected 200, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(taskCount)

		t.Logf("Deleted %d tasks in %v (avg: %v per delete)", taskCount, duration, avgDuration)

		// Performance threshold: 30ms per delete
		if avgDuration > 30*time.Millisecond {
			t.Errorf("Task delete too slow: %v per delete (threshold: 30ms)", avgDuration)
		}
	})
}

// TestPerformanceConcurrency tests concurrent operations
func TestPerformanceConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	app := createTestApp()

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Create test tasks
		for i := 0; i < 20; i++ {
			testTask := setupTestTask(t, env, "feature", "backlog")
			defer testTask.Cleanup()
		}

		concurrency := 10
		iterationsPerWorker := 20

		start := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterationsPerWorker)

		for w := 0; w < concurrency; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < iterationsPerWorker; i++ {
					w, err := makeFiberRequest(app, FiberTestRequest{
						Method: "GET",
						Path:   "/api/tasks",
					})
					if err != nil {
						errors <- fmt.Errorf("Worker %d iteration %d: %v", workerID, i, err)
						return
					}

					if w.Code != 200 {
						errors <- fmt.Errorf("Worker %d iteration %d: Expected 200, got %d", workerID, i, w.Code)
					}
				}
			}(w)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOps := concurrency * iterationsPerWorker
		avgDuration := duration / time.Duration(totalOps)

		t.Logf("Concurrent reads: %d workers × %d ops = %d total ops in %v (avg: %v per op)",
			concurrency, iterationsPerWorker, totalOps, duration, avgDuration)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed with %d errors", errorCount)
		}

		// Performance threshold: 100ms average with concurrency
		if avgDuration > 100*time.Millisecond {
			t.Errorf("Concurrent read too slow: %v per op (threshold: 100ms)", avgDuration)
		}
	})

	t.Run("ConcurrentWrites", func(t *testing.T) {
		concurrency := 5
		iterationsPerWorker := 10

		start := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterationsPerWorker)

		for w := 0; w < concurrency; w++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()

				for i := 0; i < iterationsPerWorker; i++ {
					taskReq := TestData.CreateTaskRequest(
						fmt.Sprintf("Concurrent Task W%d-I%d", workerID, i),
						"bug-fix",
						"test-target",
					)

					w, err := makeFiberRequest(app, FiberTestRequest{
						Method: "POST",
						Path:   "/api/tasks",
						Body:   taskReq,
					})
					if err != nil {
						errors <- fmt.Errorf("Worker %d iteration %d: %v", workerID, i, err)
						return
					}

					if w.Code != 201 {
						errors <- fmt.Errorf("Worker %d iteration %d: Expected 201, got %d", workerID, i, w.Code)
					}
				}
			}(w)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOps := concurrency * iterationsPerWorker
		avgDuration := duration / time.Duration(totalOps)

		t.Logf("Concurrent writes: %d workers × %d ops = %d total ops in %v (avg: %v per op)",
			concurrency, iterationsPerWorker, totalOps, duration, avgDuration)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed with %d errors", errorCount)
		}

		// Performance threshold: 200ms average with concurrency
		if avgDuration > 200*time.Millisecond {
			t.Errorf("Concurrent write too slow: %v per op (threshold: 200ms)", avgDuration)
		}
	})

	t.Run("MixedOperations", func(t *testing.T) {
		// Create initial tasks
		initialTasks := make([]*TestTask, 10)
		for i := 0; i < 10; i++ {
			initialTasks[i] = setupTestTask(t, env, "feature", "backlog")
			defer initialTasks[i].Cleanup()
		}

		concurrency := 6
		iterationsPerWorker := 10

		start := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, concurrency*iterationsPerWorker)

		// Different worker types
		for w := 0; w < concurrency; w++ {
			wg.Add(1)

			workerType := w % 3
			go func(workerID, wType int) {
				defer wg.Done()

				for i := 0; i < iterationsPerWorker; i++ {
					var w *httptest.ResponseRecorder
					var err error

					switch wType {
					case 0: // Read operations
						w, err = makeFiberRequest(app, FiberTestRequest{
							Method: "GET",
							Path:   "/api/tasks",
						})
					case 1: // Create operations
						taskReq := TestData.CreateTaskRequest(
							fmt.Sprintf("Mixed Task W%d-I%d", workerID, i),
							"bug-fix",
							"test-target",
						)
						w, err = makeFiberRequest(app, FiberTestRequest{
							Method: "POST",
							Path:   "/api/tasks",
							Body:   taskReq,
						})
					case 2: // Update operations
						taskID := initialTasks[i%len(initialTasks)].Task.ID
						updateReq := map[string]interface{}{
							"notes": fmt.Sprintf("Worker %d update %d", workerID, i),
						}
						w, err = makeFiberRequest(app, FiberTestRequest{
							Method: "PUT",
							Path:   fmt.Sprintf("/api/tasks/%s", taskID),
							Body:   updateReq,
						})
					}

					if err != nil {
						errors <- fmt.Errorf("Worker %d (type %d) iteration %d: %v", workerID, wType, i, err)
						return
					}

					expectedCodes := map[int]int{0: 200, 1: 201, 2: 200}
					if w.Code != expectedCodes[wType] {
						errors <- fmt.Errorf("Worker %d (type %d) iteration %d: Expected %d, got %d",
							workerID, wType, i, expectedCodes[wType], w.Code)
					}
				}
			}(w, workerType)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalOps := concurrency * iterationsPerWorker
		avgDuration := duration / time.Duration(totalOps)

		t.Logf("Mixed operations: %d workers × %d ops = %d total ops in %v (avg: %v per op)",
			concurrency, iterationsPerWorker, totalOps, duration, avgDuration)

		// Check for errors
		errorCount := 0
		for err := range errors {
			t.Error(err)
			errorCount++
		}

		if errorCount > 0 {
			t.Errorf("Failed with %d errors", errorCount)
		}

		// Performance threshold: 150ms average with mixed operations
		if avgDuration > 150*time.Millisecond {
			t.Errorf("Mixed operations too slow: %v per op (threshold: 150ms)", avgDuration)
		}
	})
}

// BenchmarkTaskCreation benchmarks task creation
func BenchmarkTaskCreation(b *testing.B) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	tasksDir = env.TasksDir
	app := createTestApp()

	taskReq := TestData.CreateTaskRequest("Benchmark Task", "bug-fix", "test-target")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   taskReq,
		})
	}
}

// BenchmarkTaskRetrieval benchmarks task retrieval
func BenchmarkTaskRetrieval(b *testing.B) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(&testing.T{})
	defer env.Cleanup()

	tasksDir = env.TasksDir
	app := createTestApp()

	// Create some test tasks
	for i := 0; i < 20; i++ {
		setupTestTask(&testing.T{}, env, "feature", "backlog")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
		})
	}
}

// BenchmarkPriorityCalculation benchmarks priority calculation
func BenchmarkPriorityCalculation(b *testing.B) {
	// Note: calculatePriorityScore would need to be exported or tested indirectly
	// For now, we benchmark the conversion functions which are the core of priority calc
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		convertUrgencyToFloat("high")
		convertResourceCostToFloat("moderate")
		getFloat(8.0, 5.0)
		getFloat(0.8, 0.5)
	}
}
