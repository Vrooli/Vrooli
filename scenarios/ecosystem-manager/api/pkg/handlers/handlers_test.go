package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
	"github.com/gorilla/mux"
)

// Test helpers

func setupTestEnv(t *testing.T) (string, func()) {
	t.Helper()

	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")

	for _, status := range []string{"pending", "in-progress", "completed", "failed"} {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("Failed to create queue dir %s: %v", status, err)
		}
	}

	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("Failed to create prompts dir: %v", err)
	}

	// Create minimal sections.yaml with proper operations configuration
	sectionsYAML := `name: test-ecosystem
type: unified
operations:
  resource-generator:
    name: resource-generator
    type: generator
    target: resources
    description: Test resource generator
    additional_sections: []
  resource-improver:
    name: resource-improver
    type: improver
    target: resources
    description: Test resource improver
    additional_sections: []
  scenario-generator:
    name: scenario-generator
    type: generator
    target: scenarios
    description: Test scenario generator
    additional_sections: []
  scenario-improver:
    name: scenario-improver
    type: improver
    target: scenarios
    description: Test scenario improver
    additional_sections: []
global_config:
  time_allocations:
    generators:
      research: 30
      prd_creation: 50
      scaffolding: 20
    improvers:
      assessment: 20
      prioritization: 10
      implementation: 60
      validation: 10`
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), []byte(sectionsYAML), 0o644); err != nil {
		t.Fatalf("Failed to create sections.yaml: %v", err)
	}

	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	cleanup := func() {
		os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")
	}

	return tempDir, cleanup
}

func createTestHandlers(t *testing.T, tempDir string) (*TaskHandlers, *QueueHandlers, *HealthHandlers, *DiscoveryHandlers, *SettingsHandlers) {
	t.Helper()

	queueDir := filepath.Join(tempDir, "queue")
	promptsDir := filepath.Join(tempDir, "prompts")

	storage := tasks.NewStorage(queueDir)
	assembler, err := prompts.NewAssembler(promptsDir, tempDir)
	if err != nil {
		t.Fatalf("Failed to create assembler: %v", err)
	}

	broadcast := make(chan any, 10)
	processor := queue.NewProcessor(storage, assembler, broadcast, nil)
	wsManager := websocket.NewManager()
	testRecycler := &recycler.Recycler{}

	taskHandlers := NewTaskHandlers(storage, assembler, processor, wsManager, nil)
	queueHandlers := NewQueueHandlers(processor, wsManager, storage)
	healthHandlers := NewHealthHandlers(processor, nil)
	discoveryHandlers := NewDiscoveryHandlers(assembler)
	settingsHandlers := NewSettingsHandlers(processor, wsManager, testRecycler)

	return taskHandlers, queueHandlers, healthHandlers, discoveryHandlers, settingsHandlers
}

func mustSaveTask(t *testing.T, storage *tasks.Storage, status string, task tasks.TaskItem) tasks.TaskItem {
	t.Helper()
	now := time.Now().Format(time.RFC3339)

	if task.ID == "" {
		task.ID = "task-" + status
	}
	if task.CreatedAt == "" {
		task.CreatedAt = now
	}
	if task.UpdatedAt == "" {
		task.UpdatedAt = now
	}
	if task.Status == "" {
		task.Status = status
	}

	if err := storage.SaveQueueItem(task, status); err != nil {
		t.Fatalf("Failed to save task %s: %v", task.ID, err)
	}
	return task
}

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _, healthHandlers, _, _ := createTestHandlers(t, tempDir)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandlers.HealthCheckHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["service"] != "ecosystem-manager-api" {
			t.Errorf("Expected service name, got %v", response["service"])
		}

		if response["status"] == nil {
			t.Error("Expected status field")
		}
	})
}

// TestTaskHandlers_GetTasks tests the get tasks endpoint
func TestTaskHandlers_GetTasks(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	t.Run("EmptyQueue", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tasks", nil)
		w := httptest.NewRecorder()

		taskHandlers.GetTasksHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		tasksField, ok := response["tasks"].([]any)
		if !ok {
			t.Fatal("Expected tasks array")
		}

		if len(tasksField) != 0 {
			t.Errorf("Expected empty tasks array, got %d items", len(tasksField))
		}
	})

	t.Run("WithTasks", func(t *testing.T) {
		task := tasks.TaskItem{
			ID:        "test-task-1",
			Type:      "resource",
			Operation: "generator",
			Target:    "test-resource",
			Status:    "pending",
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}

		if err := taskHandlers.storage.SaveQueueItem(task, "pending"); err != nil {
			t.Fatalf("Failed to save test task: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/tasks", nil)
		w := httptest.NewRecorder()

		taskHandlers.GetTasksHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("FilterByStatus", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tasks?status=pending", nil)
		w := httptest.NewRecorder()

		taskHandlers.GetTasksHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestTaskHandlers_CreateTask tests the create task endpoint
func TestTaskHandlers_CreateTask(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	t.Run("ValidSingleTarget", func(t *testing.T) {
		taskBody := map[string]any{
			"type":      "resource",
			"operation": "generator",
			"target":    "test-resource",
			"title":     "Test Resource Generation",
		}

		bodyBytes, _ := json.Marshal(taskBody)
		req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		taskHandlers.CreateTaskHandler(w, req)

		if w.Code != http.StatusCreated && w.Code != http.StatusOK && w.Code != http.StatusConflict {
			t.Errorf("Expected status 201, 200, or 409, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("MissingType", func(t *testing.T) {
		taskBody := map[string]any{
			"operation": "generator",
			"target":    "test-resource",
		}

		bodyBytes, _ := json.Marshal(taskBody)
		req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		taskHandlers.CreateTaskHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader([]byte(`{"invalid": json`)))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		taskHandlers.CreateTaskHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})
}

func TestTaskHandlers_CreateTask_MultiTarget(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	body := map[string]any{
		"type":      "resource",
		"operation": "generator",
		"targets":   []string{"alpha", "beta"},
		"title":     "Generate resources",
	}

	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	taskHandlers.CreateTaskHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool             `json:"success"`
		Created []tasks.TaskItem `json:"created"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(resp.Created) != 2 {
		t.Fatalf("expected 2 created tasks, got %d", len(resp.Created))
	}
}

// TestTaskHandlers_GetTask tests the get single task endpoint
func TestTaskHandlers_GetTask(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	t.Run("ExistingTask", func(t *testing.T) {
		task := tasks.TaskItem{
			ID:        "test-task-get",
			Type:      "resource",
			Operation: "generator",
			Target:    "test-resource",
			Status:    "pending",
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}

		if err := taskHandlers.storage.SaveQueueItem(task, "pending"); err != nil {
			t.Fatalf("Failed to save test task: %v", err)
		}

		req := httptest.NewRequest("GET", "/api/tasks/test-task-get", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "test-task-get"})
		w := httptest.NewRecorder()

		taskHandlers.GetTaskHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("NonExistentTask", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tasks/non-existent", nil)
		req = mux.SetURLVars(req, map[string]string{"id": "non-existent"})
		w := httptest.NewRecorder()

		taskHandlers.GetTaskHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

// TestQueueHandlers_GetQueueStatus tests the queue status endpoint
func TestQueueHandlers_GetQueueStatus(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, queueHandlers, _, _, _ := createTestHandlers(t, tempDir)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/queue/status", nil)
		w := httptest.NewRecorder()

		queueHandlers.GetQueueStatusHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["processor_active"] == nil {
			t.Error("Expected processor_active field")
		}
	})
}

// TestDiscoveryHandlers tests discovery endpoints
func TestDiscoveryHandlers_GetResources(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _, _, discoveryHandlers, _ := createTestHandlers(t, tempDir)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/resources", nil)
		w := httptest.NewRecorder()

		discoveryHandlers.GetResourcesHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestDiscoveryHandlers_GetScenarios(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	_, _, _, discoveryHandlers, _ := createTestHandlers(t, tempDir)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/discovery/scenarios", nil)
		w := httptest.NewRecorder()

		discoveryHandlers.GetScenariosHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

func TestTaskHandlers_GetActiveTargets(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	mustSaveTask(t, taskHandlers.storage, "pending", tasks.TaskItem{
		ID:        "t-pending",
		Type:      "resource",
		Operation: "generator",
		Target:    "alpha",
	})
	mustSaveTask(t, taskHandlers.storage, "in-progress", tasks.TaskItem{
		ID:        "t-progress",
		Type:      "resource",
		Operation: "generator",
		Target:    "beta",
	})
	mustSaveTask(t, taskHandlers.storage, "pending", tasks.TaskItem{
		ID:        "t-ignore",
		Type:      "scenario",
		Operation: "generator",
		Target:    "gamma",
	})

	t.Run("RequiresParams", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tasks/active-targets", nil)
		w := httptest.NewRecorder()

		taskHandlers.GetActiveTargetsHandler(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing params, got %d", w.Code)
		}
	})

	t.Run("ReturnsActiveTargets", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/tasks/active-targets?type=resource&operation=generator", nil)
		w := httptest.NewRecorder()

		taskHandlers.GetActiveTargetsHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var resp []map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected two active targets, got %d", len(resp))
		}
	})
}

func TestTaskHandlers_UpdateTask(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	current := mustSaveTask(t, taskHandlers.storage, "pending", tasks.TaskItem{
		ID:        "update-task",
		Type:      "resource",
		Operation: "generator",
		Target:    "delta",
		Title:     "Original Title",
		Priority:  "P1",
	})

	updateBody := map[string]any{
		"title": "Refined Title",
		// omit priority to ensure preserveUnsetFields keeps the original value
	}
	bodyBytes, _ := json.Marshal(updateBody)

	req := httptest.NewRequest("PUT", "/api/tasks/update-task", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": current.ID})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	taskHandlers.UpdateTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool           `json:"success"`
		Task    tasks.TaskItem `json:"task"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Task.Priority != "P1" {
		t.Fatalf("expected priority to be preserved, got %s", resp.Task.Priority)
	}

	expectedTitle := deriveTaskTitle("", current.Operation, current.Type, "")
	if resp.Task.Title != expectedTitle {
		t.Fatalf("expected derived title %q, got %q", expectedTitle, resp.Task.Title)
	}
}

func TestTaskHandlers_UpdateTaskStatus_BackwardsTransition(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	completed := mustSaveTask(t, taskHandlers.storage, "completed", tasks.TaskItem{
		ID:        "status-task",
		Type:      "resource",
		Operation: "generator",
		Target:    "epsilon",
		Status:    "completed",
		Results: map[string]any{
			"success": true,
		},
		CompletedAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	})

	body := map[string]any{
		"status": "pending",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("PUT", "/api/tasks/status-task/status", bytes.NewReader(bodyBytes))
	req = mux.SetURLVars(req, map[string]string{"id": completed.ID})
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	taskHandlers.UpdateTaskStatusHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
	}

	var updated tasks.TaskItem
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if updated.Status != "pending" {
		t.Fatalf("expected status pending, got %s", updated.Status)
	}
	if updated.Results != nil {
		t.Fatalf("expected results to be cleared on backwards transition, got %v", updated.Results)
	}
	if updated.CurrentPhase == "archived" {
		t.Fatalf("current phase should not be archived on backwards transition")
	}
}

func TestTaskHandlers_DeleteTask(t *testing.T) {
	tempDir, cleanup := setupTestEnv(t)
	defer cleanup()

	taskHandlers, _, _, _, _ := createTestHandlers(t, tempDir)

	task := mustSaveTask(t, taskHandlers.storage, "pending", tasks.TaskItem{
		ID:        "delete-me",
		Type:      "resource",
		Operation: "generator",
		Target:    "zeta",
	})

	req := httptest.NewRequest("DELETE", "/api/tasks/delete-me", nil)
	req = mux.SetURLVars(req, map[string]string{"id": task.ID})
	w := httptest.NewRecorder()

	taskHandlers.DeleteTaskHandler(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 response, got %d", w.Code)
	}

	if _, _, err := taskHandlers.storage.GetTaskByID(task.ID); err == nil {
		t.Fatalf("expected task %s to be deleted", task.ID)
	}
}
