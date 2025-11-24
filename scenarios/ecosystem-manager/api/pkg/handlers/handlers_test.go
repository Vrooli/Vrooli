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
	processor := queue.NewProcessor(30*time.Second, storage, assembler, broadcast, nil)
	wsManager := websocket.NewManager()
	testRecycler := &recycler.Recycler{}

	taskHandlers := NewTaskHandlers(storage, assembler, processor, wsManager, nil)
	queueHandlers := NewQueueHandlers(processor, wsManager, storage)
	healthHandlers := NewHealthHandlers(processor, nil)
	discoveryHandlers := NewDiscoveryHandlers(assembler)
	settingsHandlers := NewSettingsHandlers(processor, wsManager, testRecycler)

	return taskHandlers, queueHandlers, healthHandlers, discoveryHandlers, settingsHandlers
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
