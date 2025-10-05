package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestComprehensiveEdgeCases tests additional edge cases for better coverage
func TestComprehensiveEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateTaskWithMetadata", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body: map[string]interface{}{
				"title":       "Task with metadata",
				"description": "This has metadata",
				"metadata": map[string]interface{}{
					"priority": "high",
					"tags":     []string{"urgent", "important"},
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"title": "Task with metadata",
		})

		if response != nil {
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if metadata["priority"] != "high" {
					t.Error("Expected metadata to be preserved")
				}
			}
		}
	})

	t.Run("UpdateTaskMetadata", func(t *testing.T) {
		testTask := setupTestTask(t, "Metadata Update Test")
		defer testTask.Cleanup()

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body: map[string]interface{}{
				"metadata": map[string]interface{}{
					"updated": true,
					"version": 2,
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if metadata, ok := response["metadata"].(map[string]interface{}); ok {
				if metadata["updated"] != true {
					t.Error("Expected metadata to be updated")
				}
			}
		}
	})

	t.Run("TaskWithLongDescription", func(t *testing.T) {
		longDesc := ""
		for i := 0; i < 1000; i++ {
			longDesc += "This is a very long description. "
		}

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body: map[string]interface{}{
				"title":       "Task with long description",
				"description": longDesc,
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}
	})

	t.Run("MultipleFieldUpdates", func(t *testing.T) {
		testTask := setupTestTask(t, "Multi-field Test")
		defer testTask.Cleanup()

		newTitle := "Completely New Title"
		newDesc := "Completely New Description"
		newStatus := "completed"

		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method:  "PUT",
			Path:    "/api/v1/tasks/" + testTask.Task.ID.String(),
			URLVars: map[string]string{"id": testTask.Task.ID.String()},
			Body: map[string]interface{}{
				"title":       newTitle,
				"description": newDesc,
				"status":      newStatus,
				"metadata": map[string]interface{}{
					"updated": true,
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		updateTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"title":       newTitle,
			"description": newDesc,
			"status":      newStatus,
		})

		if response != nil {
			if _, ok := response["updated_at"]; !ok {
				t.Error("Expected updated_at to be set")
			}
		}
	})

	t.Run("CreateTaskMinimalFields", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   map[string]interface{}{"title": "Minimal Task"},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{
			"title":  "Minimal Task",
			"status": "pending",
		})

		if response != nil {
			if desc, ok := response["description"]; ok && desc != "" {
				t.Error("Expected empty description for minimal task")
			}
		}
	})

	t.Run("ListTasksWithManyTasks", func(t *testing.T) {
		// Create 50 tasks
		tasks := setupMultipleTestTasks(t, 50)
		defer func() {
			for _, task := range tasks {
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

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			count, ok := response["count"].(float64)
			if !ok || count < 50 {
				t.Errorf("Expected at least 50 tasks, got %v", count)
			}
		}
	})

	t.Run("TestHelperFunctions", func(t *testing.T) {
		// Test assertJSONArray
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tasks",
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		listTasksHandler(w, httpReq)
		array := assertJSONArray(t, w, http.StatusOK, "tasks")
		if array == nil {
			t.Error("Expected array to be returned")
		}

		// Test UpdateTaskRequest from TestData
		updateReq := TestData.UpdateTaskRequest(
			strPtr("New Title"),
			strPtr("New Description"),
			strPtr("completed"),
		)
		if updateReq.Title == nil || *updateReq.Title != "New Title" {
			t.Error("UpdateTaskRequest helper failed")
		}
	})

	t.Run("StoreEdgeCases", func(t *testing.T) {
		// Test updating with nil metadata
		task := &Task{
			ID:     uuid.New(),
			Title:  "Test",
			Status: "pending",
		}
		store.Create(task)

		updates := map[string]interface{}{
			"title": "Updated",
		}
		updated, err := store.Update(task.ID, updates)
		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}
		if updated.Title != "Updated" {
			t.Error("Title was not updated")
		}

		// Clean up
		store.Delete(task.ID)
	})

	t.Run("TimestampValidation", func(t *testing.T) {
		beforeCreate := time.Now()
		testTask := setupTestTask(t, "Timestamp Test")
		afterCreate := time.Now()
		defer testTask.Cleanup()

		if testTask.Task.CreatedAt.Before(beforeCreate) || testTask.Task.CreatedAt.After(afterCreate) {
			t.Error("CreatedAt timestamp is out of expected range")
		}

		// Wait a tiny bit to ensure UpdatedAt changes
		time.Sleep(time.Millisecond)

		beforeUpdate := time.Now()
		updates := map[string]interface{}{"title": "Updated"}
		updated, err := store.Update(testTask.Task.ID, updates)
		afterUpdate := time.Now()

		if err != nil {
			t.Fatalf("Failed to update: %v", err)
		}

		if updated.UpdatedAt.Before(beforeUpdate) || updated.UpdatedAt.After(afterUpdate) {
			t.Error("UpdatedAt timestamp is out of expected range")
		}

		if !updated.UpdatedAt.After(updated.CreatedAt) {
			t.Error("UpdatedAt should be after CreatedAt")
		}
	})
}

// Helper function for string pointers
func strPtr(s string) *string {
	return &s
}

// TestHTTPRequestVariations tests different HTTP request patterns
func TestHTTPRequestVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("RequestWithCustomHeaders", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("X-Custom-Header") != "test-value" {
			t.Error("Custom header not set correctly")
		}

		healthHandler(w, httpReq)
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("RequestWithQueryParams", func(t *testing.T) {
		_, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/tasks",
			QueryParams: map[string]string{
				"filter": "active",
				"limit":  "10",
			},
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.URL.Query().Get("filter") != "active" {
			t.Error("Query parameter 'filter' not set correctly")
		}
		if httpReq.URL.Query().Get("limit") != "10" {
			t.Error("Query parameter 'limit' not set correctly")
		}
	})

	t.Run("RequestWithByteBody", func(t *testing.T) {
		bodyBytes := []byte(`{"title":"Byte Body Test"}`)
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   bodyBytes,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})

	t.Run("RequestWithStringBody", func(t *testing.T) {
		w, httpReq, err := makeHTTPRequest(HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/tasks",
			Body:   `{"title":"String Body Test"}`,
		})
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		createTaskHandler(w, httpReq)
		if w.Code != http.StatusCreated {
			t.Errorf("Expected 201, got %d", w.Code)
		}
	})
}
