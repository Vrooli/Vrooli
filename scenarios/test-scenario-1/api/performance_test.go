package main

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPerformanceCreateTasks tests task creation performance
func TestPerformanceCreateTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	patterns := []PerformanceTestPattern{
		CreateTaskPerformancePattern(100, 100*time.Millisecond),
		CreateTaskPerformancePattern(1000, 1*time.Second),
	}

	RunPerformanceTests(t, patterns)
}

// TestPerformanceListTasks tests task listing performance
func TestPerformanceListTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	patterns := []PerformanceTestPattern{
		ListTasksPerformancePattern(100, 10*time.Millisecond),
		ListTasksPerformancePattern(1000, 50*time.Millisecond),
	}

	RunPerformanceTests(t, patterns)
}

// TestPerformanceGetTask tests task retrieval performance
func TestPerformanceGetTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SingleTaskRetrieval", func(t *testing.T) {
		// Create a task
		testTask := setupTestTask(t, "Performance Test Task")
		defer testTask.Cleanup()

		iterations := 10000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			_, err := store.Get(testTask.Task.ID)
			if err != nil {
				t.Fatalf("Failed to get task: %v", err)
			}
		}

		duration := time.Since(start)
		avgPerOp := duration / time.Duration(iterations)

		t.Logf("Retrieved task %d times in %v (avg: %v per operation)", iterations, duration, avgPerOp)

		if avgPerOp > time.Microsecond*100 {
			t.Errorf("Task retrieval too slow: %v per operation (max: 100µs)", avgPerOp)
		}
	})

	t.Run("TaskRetrievalWithManyTasks", func(t *testing.T) {
		// Create many tasks
		taskCount := 1000
		tasks := setupMultipleTestTasks(t, taskCount)
		defer func() {
			for _, task := range tasks {
				task.Cleanup()
			}
		}()

		// Measure retrieval time
		iterations := 100
		start := time.Now()

		for i := 0; i < iterations; i++ {
			// Get a random task
			idx := i % taskCount
			_, err := store.Get(tasks[idx].Task.ID)
			if err != nil {
				t.Fatalf("Failed to get task: %v", err)
			}
		}

		duration := time.Since(start)
		avgPerOp := duration / time.Duration(iterations)

		t.Logf("Retrieved tasks %d times with %d total tasks in %v (avg: %v per operation)",
			iterations, taskCount, duration, avgPerOp)

		if avgPerOp > time.Millisecond {
			t.Errorf("Task retrieval too slow with many tasks: %v per operation (max: 1ms)", avgPerOp)
		}
	})
}

// TestPerformanceUpdateTask tests task update performance
func TestPerformanceUpdateTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("UpdateOperations", func(t *testing.T) {
		testTask := setupTestTask(t, "Update Performance Test")
		defer testTask.Cleanup()

		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			updates := map[string]interface{}{
				"title":  fmt.Sprintf("Updated Title %d", i),
				"status": "in_progress",
			}
			_, err := store.Update(testTask.Task.ID, updates)
			if err != nil {
				t.Fatalf("Failed to update task: %v", err)
			}
		}

		duration := time.Since(start)
		avgPerOp := duration / time.Duration(iterations)

		t.Logf("Updated task %d times in %v (avg: %v per operation)", iterations, duration, avgPerOp)

		if avgPerOp > time.Microsecond*500 {
			t.Errorf("Task update too slow: %v per operation (max: 500µs)", avgPerOp)
		}
	})
}

// TestPerformanceDeleteTask tests task deletion performance
func TestPerformanceDeleteTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("BulkDeletion", func(t *testing.T) {
		taskCount := 1000
		tasks := setupMultipleTestTasks(t, taskCount)

		start := time.Now()

		for _, task := range tasks {
			err := store.Delete(task.Task.ID)
			if err != nil {
				t.Fatalf("Failed to delete task: %v", err)
			}
		}

		duration := time.Since(start)
		avgPerOp := duration / time.Duration(taskCount)

		t.Logf("Deleted %d tasks in %v (avg: %v per operation)", taskCount, duration, avgPerOp)

		if avgPerOp > time.Microsecond*500 {
			t.Errorf("Task deletion too slow: %v per operation (max: 500µs)", avgPerOp)
		}
	})
}

// TestConcurrentAccess tests concurrent task operations
func TestConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ConcurrentCreates", func(t *testing.T) {
		goroutines := 10
		tasksPerGoroutine := 100
		var wg sync.WaitGroup
		errors := make(chan error, goroutines*tasksPerGoroutine)

		start := time.Now()

		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < tasksPerGoroutine; i++ {
					task := &Task{
						ID:     uuid.New(),
						Title:  fmt.Sprintf("Concurrent Task %d-%d", id, i),
						Status: "pending",
					}
					if err := store.Create(task); err != nil {
						errors <- err
					}
				}
			}(g)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Error during concurrent create: %v", err)
				errorCount++
			}
		}

		totalTasks := goroutines * tasksPerGoroutine
		t.Logf("Created %d tasks concurrently (%d goroutines) in %v", totalTasks, goroutines, duration)

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent creation", errorCount)
		}

		// Verify all tasks were created
		tasks := store.List()
		if len(tasks) != totalTasks {
			t.Errorf("Expected %d tasks, got %d", totalTasks, len(tasks))
		}
	})

	t.Run("ConcurrentReads", func(t *testing.T) {
		// Create some tasks first
		testTasks := setupMultipleTestTasks(t, 10)
		defer func() {
			for _, task := range testTasks {
				task.Cleanup()
			}
		}()

		goroutines := 20
		readsPerGoroutine := 500
		var wg sync.WaitGroup
		errors := make(chan error, goroutines*readsPerGoroutine)

		start := time.Now()

		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for i := 0; i < readsPerGoroutine; i++ {
					taskIdx := i % len(testTasks)
					_, err := store.Get(testTasks[taskIdx].Task.ID)
					if err != nil {
						errors <- err
					}
				}
			}(g)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)
		totalReads := goroutines * readsPerGoroutine

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Error during concurrent read: %v", err)
				errorCount++
			}
		}

		t.Logf("Performed %d concurrent reads (%d goroutines) in %v", totalReads, goroutines, duration)

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent reads", errorCount)
		}
	})

	t.Run("ConcurrentMixedOperations", func(t *testing.T) {
		goroutines := 10
		opsPerGoroutine := 50
		var wg sync.WaitGroup
		errors := make(chan error, goroutines*opsPerGoroutine)

		start := time.Now()

		for g := 0; g < goroutines; g++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Create tasks
				taskIDs := make([]uuid.UUID, opsPerGoroutine/5)
				for i := 0; i < len(taskIDs); i++ {
					task := &Task{
						ID:     uuid.New(),
						Title:  fmt.Sprintf("Mixed Op Task %d-%d", id, i),
						Status: "pending",
					}
					if err := store.Create(task); err != nil {
						errors <- err
					}
					taskIDs[i] = task.ID
				}

				// Read tasks
				for _, taskID := range taskIDs {
					if _, err := store.Get(taskID); err != nil {
						errors <- err
					}
				}

				// Update tasks
				for _, taskID := range taskIDs {
					updates := map[string]interface{}{"status": "completed"}
					if _, err := store.Update(taskID, updates); err != nil {
						errors <- err
					}
				}

				// List tasks
				store.List()

				// Delete tasks
				for _, taskID := range taskIDs {
					if err := store.Delete(taskID); err != nil {
						errors <- err
					}
				}
			}(g)
		}

		wg.Wait()
		close(errors)

		duration := time.Since(start)

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				t.Errorf("Error during mixed operations: %v", err)
				errorCount++
			}
		}

		t.Logf("Performed mixed operations with %d goroutines in %v", goroutines, duration)

		if errorCount > 0 {
			t.Errorf("Had %d errors during mixed operations", errorCount)
		}
	})
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("LargeNumberOfTasks", func(t *testing.T) {
		taskCount := 10000

		start := time.Now()

		for i := 0; i < taskCount; i++ {
			task := &Task{
				ID:          uuid.New(),
				Title:       fmt.Sprintf("Memory Test Task %d", i),
				Description: "This is a test task for memory usage testing",
				Status:      "pending",
				Metadata:    map[string]interface{}{"index": i},
			}
			store.Create(task)
		}

		duration := time.Since(start)

		t.Logf("Created %d tasks in %v", taskCount, duration)

		// Verify count
		tasks := store.List()
		if len(tasks) != taskCount {
			t.Errorf("Expected %d tasks, got %d", taskCount, len(tasks))
		}
	})
}
