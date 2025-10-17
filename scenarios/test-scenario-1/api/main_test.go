package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestHealthHandler tests the health endpoint
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		healthHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "test-scenario-1",
		})

		if response != nil {
			if _, exists := response["time"]; !exists {
				t.Error("Expected 'time' field in health response")
			}
		}
	})
}

// TestCreateTaskHandler tests task creation
func TestCreateTaskHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		req := TestData.CreateTaskRequest("Test Task", "This is a test task")
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"title":  "Test Task",
			"status": "pending",
		})

		if response != nil {
			if _, exists := response["id"]; !exists {
				t.Error("Expected 'id' field in response")
			}
			if _, exists := response["created_at"]; !exists {
				t.Error("Expected 'created_at' field in response")
			}
		}
	})

	t.Run("MissingTitle", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   map[string]interface{}{"description": "No title provided"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest, "Title is required")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})

	t.Run("EmptyBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   "",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for empty body, got %d", w.Code)
		}
	})

	t.Run("EmptyTitle", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   map[string]interface{}{"title": ""},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest, "Title is required")
	})
}

// TestGetTaskHandler tests retrieving a single task
func TestGetTaskHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testTask := setupTestTask(t, "Get Test Task")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		getTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"title": "Get Test Task",
			"id":    testTask.Task.ID.String(),
		})

		if response != nil {
			if status, ok := response["status"].(string); !ok || status == "" {
				t.Error("Expected valid status in response")
			}
		}
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		Scenarios.InvalidUUID(t, getTaskHandler, "GET", "/api/v1/tasks/{id}")
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		Scenarios.TaskNotFound(t, getTaskHandler, "GET", "/api/v1/tasks/{id}")
	})
}

// TestListTasksHandler tests listing all tasks
func TestListTasksHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("EmptyList", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listTasksHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"count": float64(0),
		})

		if response != nil {
			tasks, ok := response["tasks"].([]interface{})
			if !ok {
				t.Error("Expected 'tasks' to be an array")
			} else if len(tasks) != 0 {
				t.Errorf("Expected 0 tasks, got %d", len(tasks))
			}
		}
	})

	t.Run("MultipleTasksok", func(t *testing.T) {
		testTasks := setupMultipleTestTasks(t, 5)
		defer func() {
			for _, task := range testTasks {
				task.Cleanup()
			}
		}()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listTasksHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"count": float64(5),
		})

		if response != nil {
			tasks, ok := response["tasks"].([]interface{})
			if !ok {
				t.Error("Expected 'tasks' to be an array")
			} else if len(tasks) != 5 {
				t.Errorf("Expected 5 tasks, got %d", len(tasks))
			}
		}
	})

	t.Run("SingleTask", func(t *testing.T) {
		testTask := setupTestTask(t, "Single Task")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listTasksHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"count": float64(1),
		})

		if response != nil {
			tasks, ok := response["tasks"].([]interface{})
			if !ok {
				t.Error("Expected 'tasks' to be an array")
			} else if len(tasks) != 1 {
				t.Errorf("Expected 1 task, got %d", len(tasks))
			}
		}
	})
}

// TestUpdateTaskHandler tests updating tasks
func TestUpdateTaskHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("UpdateTitle", func(t *testing.T) {
		testTask := setupTestTask(t, "Original Title")
		defer testTask.Cleanup()

		newTitle := "Updated Title"
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body:    map[string]interface{}{"title": newTitle},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"title": newTitle,
			"id":    testTask.Task.ID.String(),
		})

		if response != nil {
			if updatedAt, ok := response["updated_at"]; !ok || updatedAt == nil {
				t.Error("Expected 'updated_at' field in response")
			}
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		testTask := setupTestTask(t, "Task for Status Update")
		defer testTask.Cleanup()

		newStatus := "completed"
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body:    map[string]interface{}{"status": newStatus},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": newStatus,
		})
	})

	t.Run("UpdateMultipleFields", func(t *testing.T) {
		testTask := setupTestTask(t, "Multi-field Update")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body: map[string]interface{}{
				"title":       "New Title",
				"description": "New Description",
				"status":      "in_progress",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"title":       "New Title",
			"description": "New Description",
			"status":      "in_progress",
		})
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		Scenarios.InvalidUUID(t, updateTaskHandler, "PUT", "/api/v1/tasks/{id}")
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		Scenarios.TaskNotFound(t, updateTaskHandler, "PUT", "/api/v1/tasks/{id}")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		testTask := setupTestTask(t, "Invalid JSON Test")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body:    `{"invalid": json}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid JSON")
	})

	t.Run("EmptyUpdate", func(t *testing.T) {
		testTask := setupTestTask(t, "Empty Update Test")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body:    map[string]interface{}{},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		// Empty update should succeed but not change anything
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for empty update, got %d", w.Code)
		}
	})
}

// TestDeleteTaskHandler tests deleting tasks
func TestDeleteTaskHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		testTask := setupTestTask(t, "Task to Delete")

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		deleteTaskHandler(w, httpReq)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204, got %d", w.Code)
		}

		// Verify task is actually deleted
		_, err = store.Get(testTask.Task.ID)
		if err == nil {
			t.Error("Expected task to be deleted, but it still exists")
		}
	})

	t.Run("InvalidUUID", func(t *testing.T) {
		Scenarios.InvalidUUID(t, deleteTaskHandler, "DELETE", "/api/v1/tasks/{id}")
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		Scenarios.TaskNotFound(t, deleteTaskHandler, "DELETE", "/api/v1/tasks/{id}")
	})

	t.Run("DeleteTwice", func(t *testing.T) {
		testTask := setupTestTask(t, "Double Delete Test")

		// First deletion
		w1, httpReq1, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
		})
		deleteTaskHandler(w1, httpReq1)

		if w1.Code != http.StatusNoContent {
			t.Errorf("First delete: expected status 204, got %d", w1.Code)
		}

		// Second deletion should fail
		w2, httpReq2, _ := makeHTTPRequest(HTTPTestRequest{
			Method:  "DELETE",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
		})
		deleteTaskHandler(w2, httpReq2)

		assertErrorResponse(t, w2, http.StatusNotFound, "Task not found")
	})
}

// TestTaskStore tests the task store directly
func TestTaskStore(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateAndGet", func(t *testing.T) {
		task := &Task{
			ID:     uuid.New(),
			Title:  "Store Test",
			Status: "pending",
		}

		err := store.Create(task)
		if err != nil {
			t.Fatalf("Failed to create task: %v", err)
		}

		retrieved, err := store.Get(task.ID)
		if err != nil {
			t.Fatalf("Failed to get task: %v", err)
		}

		if retrieved.ID != task.ID {
			t.Errorf("Expected ID %s, got %s", task.ID, retrieved.ID)
		}
		if retrieved.Title != task.Title {
			t.Errorf("Expected title %s, got %s", task.Title, retrieved.Title)
		}
	})

	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := store.Get(uuid.New())
		if err == nil {
			t.Error("Expected error for non-existent task")
		}
	})

	t.Run("Update", func(t *testing.T) {
		task := &Task{
			ID:     uuid.New(),
			Title:  "Original",
			Status: "pending",
		}
		store.Create(task)

		updates := map[string]interface{}{
			"title":  "Updated",
			"status": "completed",
		}

		updated, err := store.Update(task.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update task: %v", err)
		}

		if updated.Title != "Updated" {
			t.Errorf("Expected title 'Updated', got '%s'", updated.Title)
		}
		if updated.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", updated.Status)
		}
	})

	t.Run("UpdateNonExistent", func(t *testing.T) {
		_, err := store.Update(uuid.New(), map[string]interface{}{"title": "Test"})
		if err == nil {
			t.Error("Expected error for updating non-existent task")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		task := &Task{
			ID:     uuid.New(),
			Title:  "To Delete",
			Status: "pending",
		}
		store.Create(task)

		err := store.Delete(task.ID)
		if err != nil {
			t.Fatalf("Failed to delete task: %v", err)
		}

		_, err = store.Get(task.ID)
		if err == nil {
			t.Error("Expected error when getting deleted task")
		}
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		err := store.Delete(uuid.New())
		if err == nil {
			t.Error("Expected error for deleting non-existent task")
		}
	})

	t.Run("List", func(t *testing.T) {
		// Create multiple tasks
		for i := 0; i < 3; i++ {
			task := &Task{
				ID:     uuid.New(),
				Title:  "List Test " + string(rune(i)),
				Status: "pending",
			}
			store.Create(task)
		}

		tasks := store.List()
		if len(tasks) < 3 {
			t.Errorf("Expected at least 3 tasks, got %d", len(tasks))
		}
	})
}

// TestErrorPatterns tests using the error pattern builder
func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateTaskErrors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "createTask",
			Handler:     createTaskHandler,
			BaseURL:     "/api/v1/tasks",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/tasks", "POST").
			AddMissingRequiredField("/api/v1/tasks", "POST").
			AddEmptyBody("/api/v1/tasks", "POST").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("GetTaskErrors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "getTask",
			Handler:     getTaskHandler,
			BaseURL:     "/api/v1/tasks/{id}",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidUUID("/api/v1/tasks/{id}", "GET").
			AddNonExistentTask("/api/v1/tasks/{id}", "GET").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("UpdateTaskErrors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "updateTask",
			Handler:     updateTaskHandler,
			BaseURL:     "/api/v1/tasks/{id}",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidUUID("/api/v1/tasks/{id}", "PUT").
			AddNonExistentTask("/api/v1/tasks/{id}", "PUT").
			AddInvalidJSON("/api/v1/tasks/{id}", "PUT").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("DeleteTaskErrors", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "deleteTask",
			Handler:     deleteTaskHandler,
			BaseURL:     "/api/v1/tasks/{id}",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidUUID("/api/v1/tasks/{id}", "DELETE").
			AddNonExistentTask("/api/v1/tasks/{id}", "DELETE").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestJSONSerialization tests that tasks serialize correctly
func TestJSONSerialization(t *testing.T) {
	task := &Task{
		ID:          uuid.New(),
		Title:       "Serialization Test",
		Description: "Testing JSON",
		Status:      "pending",
		Metadata:    map[string]interface{}{"key": "value"},
	}

	data, err := json.Marshal(task)
	if err != nil {
		t.Fatalf("Failed to marshal task: %v", err)
	}

	var unmarshaled Task
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal task: %v", err)
	}

	if unmarshaled.ID != task.ID {
		t.Errorf("ID mismatch: expected %s, got %s", task.ID, unmarshaled.ID)
	}
	if unmarshaled.Title != task.Title {
		t.Errorf("Title mismatch: expected %s, got %s", task.Title, unmarshaled.Title)
	}
}
