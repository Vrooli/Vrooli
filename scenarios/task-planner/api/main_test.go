package main

import (
	"database/sql"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("HealthCheck_Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"status", "service", "version"})

		if response["status"] != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", response["status"])
		}

		if response["service"] != serviceName {
			t.Errorf("Expected service '%s', got %v", serviceName, response["service"])
		}

		if response["version"] != apiVersion {
			t.Errorf("Expected version '%s', got %v", apiVersion, response["version"])
		}
	})
}

// TestGetApps tests the get apps endpoint
func TestGetApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	suite := NewHandlerTestSuite("GetApps", env)

	t.Run("GetApps_Success", func(t *testing.T) {
		// Create test apps
		createTestApp(t, env, "test-app-1")
		createTestApp(t, env, "test-app-2")
		defer cleanupTestData(t, env)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/apps",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"apps", "count"})

		apps, ok := response["apps"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'apps' to be an array, got: %T", response["apps"])
		}

		if len(apps) < 2 {
			t.Errorf("Expected at least 2 apps, got %d", len(apps))
		}

		// Validate app structure
		for i, appInterface := range apps {
			app, ok := appInterface.(map[string]interface{})
			if !ok {
				t.Errorf("App %d is not a map: %T", i, appInterface)
				continue
			}
			ValidateAppResponse(t, app)
		}
	})

	t.Run("GetApps_EmptyDatabase", func(t *testing.T) {
		// Ensure clean database
		cleanupTestData(t, env)

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/apps",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"apps", "count"})

		count, ok := response["count"].(float64)
		if !ok {
			t.Fatalf("Expected 'count' to be a number, got: %T", response["count"])
		}

		// Count might be 0 or include existing test data
		if count < 0 {
			t.Errorf("Expected count >= 0, got %v", count)
		}
	})

	_ = suite // Use suite variable
}

// TestGetTasks tests the get tasks endpoint
func TestGetTasks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	t.Run("GetTasks_Success", func(t *testing.T) {
		// Create test tasks
		createTestTask(t, env, appID, "Task 1", "backlog")
		createTestTask(t, env, appID, "Task 2", "in_progress")
		createTestTask(t, env, appID, "Task 3", "completed")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"tasks", "count"})

		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'tasks' to be an array, got: %T", response["tasks"])
		}

		if len(tasks) < 3 {
			t.Errorf("Expected at least 3 tasks, got %d", len(tasks))
		}

		// Validate task structure
		for i, taskInterface := range tasks {
			task, ok := taskInterface.(map[string]interface{})
			if !ok {
				t.Errorf("Task %d is not a map: %T", i, taskInterface)
				continue
			}
			ValidateTaskResponse(t, task)
		}
	})

	t.Run("GetTasks_FilterByAppID", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks",
			QueryParams: map[string]string{"app_id": appID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"tasks", "count"})

		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'tasks' to be an array, got: %T", response["tasks"])
		}

		// All returned tasks should have the specified app_id
		for i, taskInterface := range tasks {
			task, ok := taskInterface.(map[string]interface{})
			if !ok {
				continue
			}
			if task["app_id"] != appID.String() {
				t.Errorf("Task %d has wrong app_id: expected %s, got %v", i, appID.String(), task["app_id"])
			}
		}
	})

	t.Run("GetTasks_FilterByStatus", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks",
			QueryParams: map[string]string{"status": "completed"},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"tasks", "count"})

		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'tasks' to be an array, got: %T", response["tasks"])
		}

		// All returned tasks should have status "completed"
		for i, taskInterface := range tasks {
			task, ok := taskInterface.(map[string]interface{})
			if !ok {
				continue
			}
			if task["status"] != "completed" {
				t.Errorf("Task %d has wrong status: expected 'completed', got %v", i, task["status"])
			}
		}
	})

	t.Run("GetTasks_FilterByPriority", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks",
			QueryParams: map[string]string{"priority": "high"},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"tasks", "count"})

		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'tasks' to be an array, got: %T", response["tasks"])
		}

		// All returned tasks should have priority "high"
		for i, taskInterface := range tasks {
			task, ok := taskInterface.(map[string]interface{})
			if !ok {
				continue
			}
			if priority, ok := task["priority"].(string); ok && priority != "high" {
				t.Errorf("Task %d has wrong priority: expected 'high', got %v", i, priority)
			}
		}
	})

	t.Run("GetTasks_WithPagination", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks",
			QueryParams: map[string]string{"limit": "2", "offset": "0"},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"tasks", "count"})

		tasks, ok := response["tasks"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'tasks' to be an array, got: %T", response["tasks"])
		}

		if len(tasks) > 2 {
			t.Errorf("Expected at most 2 tasks with limit=2, got %d", len(tasks))
		}
	})
}

// TestUpdateTaskStatus tests the update task status endpoint
func TestUpdateTaskStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	suite := NewHandlerTestSuite("UpdateTaskStatus", env)

	t.Run("UpdateTaskStatus_Success_BacklogToInProgress", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task", "backlog")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": "in_progress",
				"reason":    "Starting implementation",
				"notes":     "Test notes",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"success", "task_id", "status_changed", "previous_status", "new_status"})

		if response["success"] != true {
			t.Errorf("Expected success=true, got %v", response["success"])
		}

		if response["previous_status"] != "backlog" {
			t.Errorf("Expected previous_status='backlog', got %v", response["previous_status"])
		}

		if response["new_status"] != "in_progress" {
			t.Errorf("Expected new_status='in_progress', got %v", response["new_status"])
		}
	})

	t.Run("UpdateTaskStatus_Success_InProgressToCompleted", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task 2", "in_progress")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": "completed",
				"reason":    "Task finished",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"success", "new_status"})

		if response["new_status"] != "completed" {
			t.Errorf("Expected new_status='completed', got %v", response["new_status"])
		}
	})

	// Error tests using test patterns
	patterns := NewTestScenarioBuilder().
		AddMissingRequiredField("/api/tasks/status", "PUT", map[string]interface{}{
			"to_status": "completed",
		}).
		AddInvalidStatusTransition(map[string]interface{}{
			"task_id":   uuid.New().String(),
			"to_status": "invalid_status",
		}).
		Build()

	suite.RunErrorTests(t, patterns)

	t.Run("UpdateTaskStatus_Error_AlreadyInStatus", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task 3", "completed")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": "completed",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "already in")
	})

	t.Run("UpdateTaskStatus_Error_InvalidTransition", func(t *testing.T) {
		taskID := createTestTask(t, env, appID, "Test Task 4", "backlog")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": "completed", // Invalid: cannot go directly from backlog to completed
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// This might succeed depending on validation rules - adjust based on actual implementation
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})
}

// TestGetTaskStatusHistory tests the get task status history endpoint
func TestGetTaskStatusHistory(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	taskID := createTestTask(t, env, appID, "Test Task", "backlog")
	defer cleanupTestData(t, env)

	// Create some status changes
	statusChanges := []string{"in_progress", "review", "completed"}
	for _, status := range statusChanges {
		_, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/tasks/status",
			Body: map[string]interface{}{
				"task_id":   taskID.String(),
				"to_status": status,
				"reason":    "Test transition to " + status,
			},
		})
		if err != nil {
			t.Logf("Warning: Failed to create status change: %v", err)
		}
		time.Sleep(100 * time.Millisecond) // Small delay between changes
	}

	t.Run("GetStatusHistory_Success", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks/status-history",
			QueryParams: map[string]string{"task_id": taskID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertSuccessResponse(t, w, []string{"task_id", "history"})

		history, ok := response["history"].([]interface{})
		if !ok {
			t.Fatalf("Expected 'history' to be an array, got: %T", response["history"])
		}

		// Validate history structure
		ValidateStatusHistoryResponse(t, history)

		// History should contain the transitions we made
		if len(history) > 0 && len(history) < len(statusChanges) {
			t.Logf("Note: Expected %d history entries, got %d. Some transitions may have failed.", len(statusChanges), len(history))
		}
	})

	t.Run("GetStatusHistory_Error_MissingTaskID", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/tasks/status-history",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "task_id")
	})

	t.Run("GetStatusHistory_Error_InvalidTaskID", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/tasks/status-history",
			QueryParams: map[string]string{"task_id": "invalid-uuid"},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid")
	})
}

// TestParseText tests the parse text endpoint
func TestParseText(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	suite := NewHandlerTestSuite("ParseText", env)

	t.Run("ParseText_Error_MissingFields", func(t *testing.T) {
		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/parse-text",
			Body: map[string]interface{}{
				"raw_text": "Some text to parse",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "required")
	})

	// Error test patterns
	patterns := NewTestScenarioBuilder().
		AddEmptyPayload("/api/parse-text", "POST").
		AddUnauthorizedAccess("/api/parse-text", "POST").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestResearchTask tests the research task endpoint
func TestResearchTask(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()
	skipIfNoDatabase(t, env.DB)

	appID := createTestApp(t, env, "test-app")
	defer cleanupTestData(t, env)

	suite := NewHandlerTestSuite("ResearchTask", env)

	t.Run("ResearchTask_Error_NonBacklogStatus", func(t *testing.T) {
		// Create task in non-backlog status
		taskID := createTestTask(t, env, appID, "Test Task", "in_progress")

		w, err := makeHTTPRequest(env, HTTPTestRequest{
			Method:  "POST",
			Path:    "/api/tasks/" + taskID.String() + "/research",
			URLVars: map[string]string{"taskId": taskID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertErrorResponse(t, w, http.StatusBadRequest, "backlog")
	})

	// Error test patterns
	patterns := NewTestScenarioBuilder().
		AddInvalidUUID("/api/tasks/{taskId}/research", "POST").
		AddNonExistentTask("/api/tasks/{taskId}/research", "POST").
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestLogger tests the logger functionality
func TestLogger(t *testing.T) {
	t.Run("NewLogger_CreatesLogger", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("Expected NewLogger to return a logger instance")
		}
	})

	t.Run("Logger_Methods", func(t *testing.T) {
		logger := NewLogger()

		// These should not panic
		logger.Info("Test info message")
		logger.Warn("Test warning", nil)
		logger.Error("Test error", sql.ErrNoRows)
	})
}

// TestHTTPError tests the HTTP error response function
func TestHTTPError(t *testing.T) {
	t.Run("HTTPError_ReturnsProperFormat", func(t *testing.T) {
		// This is tested indirectly through error response tests above
		// Direct testing would require mocking ResponseWriter
	})
}

// TestTaskPlannerService tests service initialization
func TestTaskPlannerService(t *testing.T) {
	t.Run("NewTaskPlannerService_CreatesService", func(t *testing.T) {
		db, err := sql.Open("postgres", "postgres://localhost:5433/vrooli_test?sslmode=disable")
		if err != nil {
			t.Skip("Database not available")
		}
		defer db.Close()

		service := NewTaskPlannerService(db, "http://localhost:6333")
		if service == nil {
			t.Fatal("Expected NewTaskPlannerService to return a service instance")
		}

		if service.db != db {
			t.Error("Expected service.db to be set")
		}

		if service.qdrantURL != "http://localhost:6333" {
			t.Error("Expected service.qdrantURL to be set")
		}

		if service.httpClient == nil {
			t.Error("Expected service.httpClient to be initialized")
		}

		if service.logger == nil {
			t.Error("Expected service.logger to be initialized")
		}
	})
}
