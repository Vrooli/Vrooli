// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if status, ok := response["status"].(string); !ok || status == "" {
			t.Error("Expected status field in response")
		}
	})
}

// TestGetTasks tests the GET /api/tasks endpoint
func TestGetTasks(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Set global tasksDir for the test
	tasksDir = env.TasksDir

	app := createTestApp()

	t.Run("Success_EmptyTasks", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Check that response contains tasks array and count
		if _, ok := response["tasks"]; !ok {
			t.Error("Expected 'tasks' field in response")
		}
		if _, ok := response["count"]; !ok {
			t.Error("Expected 'count' field in response")
		}
	})

	t.Run("Success_WithTasks", func(t *testing.T) {
		// Create test tasks
		testTask1 := setupTestTask(t, env, "bug-fix", "active")
		defer testTask1.Cleanup()

		testTask2 := setupTestTask(t, env, "feature", "backlog")
		defer testTask2.Cleanup()

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify tasks are returned
		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatal("Expected 'tasks' to be an array")
		}
		if len(tasks) == 0 {
			t.Error("Expected at least one task")
		}

		// Verify count field
		count, ok := response["count"].(float64)
		if !ok {
			t.Error("Expected 'count' to be a number")
		}
		if count == 0 {
			t.Error("Expected count to be greater than 0")
		}
	})

	t.Run("QueryParams_Status", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
			QueryParams: map[string]string{
				"status": "active",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestCreateTask tests the POST /api/tasks endpoint
func TestCreateTask(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	// Setup test database or skip database-dependent tests
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		taskReq := TestData.CreateTaskRequest("Test Task", "bug-fix", "test-scenario")

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   taskReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := response["id"]; !ok {
			t.Error("Expected 'id' field in response")
		}
		if title, ok := response["title"].(string); !ok || title != "Test Task" {
			t.Errorf("Expected title 'Test Task', got %v", response["title"])
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   `{"invalid": "json"`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Error_MissingTitle", func(t *testing.T) {
		taskReq := map[string]interface{}{
			"type":   "bug-fix",
			"target": "test",
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   taskReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   nil,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestUpdateTask tests the PUT /api/tasks/:id endpoint
func TestUpdateTask(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		defer testTask.Cleanup()

		updateReq := map[string]interface{}{
			"notes": "Updated notes",
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
			Body:   updateReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if notes, ok := response["notes"].(string); !ok || notes != "Updated notes" {
			t.Errorf("Expected notes 'Updated notes', got %v", response["notes"])
		}
	})

	t.Run("Error_TaskNotFound", func(t *testing.T) {
		updateReq := map[string]interface{}{
			"notes": "Updated notes",
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/non-existent-task",
			Body:   updateReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		defer testTask.Cleanup()

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
			Body:   `{"invalid": "json"`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestDeleteTask tests the DELETE /api/tasks/:id endpoint
func TestDeleteTask(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		defer testTask.Cleanup()

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		// Verify task file was deleted
		if _, err := os.Stat(testTask.Path); !os.IsNotExist(err) {
			t.Error("Expected task file to be deleted")
		}
	})

	t.Run("Error_TaskNotFound", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "DELETE",
			Path:   "/api/tasks/non-existent-task",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestGetAgents tests the GET /api/agents endpoint
func TestGetAgents(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	// Setup test database or skip
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/agents",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := response["agents"]; !ok {
			t.Error("Expected 'agents' field in response")
		}
	})
}

// TestGetMetrics tests the GET /api/metrics endpoint
func TestGetMetrics(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	// Setup test database or skip
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/metrics",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		expectedFields := []string{"uptime", "total_tasks", "active_agents"}
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected '%s' field in response", field)
			}
		}
	})
}

// TestGetConfig tests the GET /api/config endpoint
func TestGetConfig(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	// Setup test database or skip
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/config",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		expectedFields := []string{"max_concurrent_tasks", "yolo_mode"}
		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected '%s' field in response", field)
			}
		}
	})
}

// TestCalculatePriority tests the POST /api/calculate-priority endpoint
func TestCalculatePriority(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	// Setup test database or skip
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		priorityReq := map[string]interface{}{
			"impact":        8.0,
			"urgency":       "high",
			"success_prob":  0.8,
			"resource_cost": "moderate",
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/calculate-priority",
			Body:   priorityReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := response["priority_score"]; !ok {
			t.Error("Expected 'priority_score' field in response")
		}
	})

	t.Run("Error_MissingFields", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		priorityReq := map[string]interface{}{
			"impact": 8.0,
			// Missing other required fields
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/calculate-priority",
			Body:   priorityReq,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		// Should still work with defaults
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHelperFunctions tests utility functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getFloat", func(t *testing.T) {
		tests := []struct {
			name         string
			input        interface{}
			defaultVal   float64
			expectedVal  float64
		}{
			{"ValidFloat64", float64(5.5), 0.0, 5.5},
			{"ValidInt", 10, 0.0, 10.0},
			{"NilValue", nil, 7.5, 7.5},
			{"StringValue", "invalid", 3.0, 3.0},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := getFloat(tt.input, tt.defaultVal)
				if result != tt.expectedVal {
					t.Errorf("Expected %f, got %f", tt.expectedVal, result)
				}
			})
		}
	})

	t.Run("getString", func(t *testing.T) {
		tests := []struct {
			name         string
			input        interface{}
			defaultVal   string
			expectedVal  string
		}{
			{"ValidString", "test", "default", "test"},
			{"NilValue", nil, "default", "default"},
			{"IntValue", 123, "default", "default"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := getString(tt.input, tt.defaultVal)
				if result != tt.expectedVal {
					t.Errorf("Expected %s, got %s", tt.expectedVal, result)
				}
			})
		}
	})

	t.Run("convertUrgencyToFloat", func(t *testing.T) {
		tests := []struct {
			urgency     interface{}
			expectedVal float64
		}{
			{"critical", 4.0},
			{"high", 3.0},
			{"medium", 2.0},
			{"low", 1.0},
			{"unknown", 2.0}, // default
			{nil, 2.0},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("Urgency_%v", tt.urgency), func(t *testing.T) {
				result := convertUrgencyToFloat(tt.urgency)
				if result != tt.expectedVal {
					t.Errorf("Expected %f, got %f", tt.expectedVal, result)
				}
			})
		}
	})

	t.Run("convertResourceCostToFloat", func(t *testing.T) {
		tests := []struct {
			cost        interface{}
			expectedVal float64
		}{
			{"minimal", 1.0},
			{"moderate", 2.0},
			{"heavy", 3.0},
			{"unknown", 2.0}, // default
			{nil, 2.0},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("Cost_%v", tt.cost), func(t *testing.T) {
				result := convertResourceCostToFloat(tt.cost)
				if result != tt.expectedVal {
					t.Errorf("Expected %f, got %f", tt.expectedVal, result)
				}
			})
		}
	})
}


// TestProblemEndpoints tests problem scanning and management
func TestProblemEndpoints(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	// Setup test database or skip
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	t.Run("GetProblems_Success", func(t *testing.T) {
		if db == nil {
			t.Skip("Skipping test: database not available")
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/problems",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if _, ok := response["problems"]; !ok {
			t.Error("Expected 'problems' field in response")
		}
	})

	t.Run("ScanProblems_InvalidJSON", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/problems/scan",
			Body:   `{"invalid": "json"`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

// TestPerformance tests performance of key operations
func TestPerformance(t *testing.T) {
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

	t.Run("CreateMultipleTasks_Performance", func(t *testing.T) {
		start := time.Now()
		iterations := 50

		for i := 0; i < iterations; i++ {
			taskReq := TestData.CreateTaskRequest(
				fmt.Sprintf("Task %d", i),
				"bug-fix",
				"test-target",
			)

			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "POST",
				Path:   "/api/tasks",
				Body:   taskReq,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if w.Code != http.StatusCreated {
				t.Errorf("Task %d: Expected status 201, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Created %d tasks in %v (avg: %v per task)", iterations, duration, avgDuration)

		if avgDuration > 100*time.Millisecond {
			t.Errorf("Task creation too slow: %v per task (threshold: 100ms)", avgDuration)
		}
	})

	t.Run("GetTasks_Performance", func(t *testing.T) {
		start := time.Now()
		iterations := 100

		for i := 0; i < iterations; i++ {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "GET",
				Path:   "/api/tasks",
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Iteration %d: Expected status 200, got %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / time.Duration(iterations)

		t.Logf("Retrieved tasks %d times in %v (avg: %v per request)", iterations, duration, avgDuration)

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Task retrieval too slow: %v per request (threshold: 50ms)", avgDuration)
		}
	})
}
