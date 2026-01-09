package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ecosystem-manager/api/pkg/autosteer"
	"github.com/ecosystem-manager/api/pkg/handlers"
	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/ecosystem-manager/api/pkg/queue"
	"github.com/ecosystem-manager/api/pkg/recycler"
	"github.com/ecosystem-manager/api/pkg/tasks"
	"github.com/ecosystem-manager/api/pkg/websocket"
)

func setupTestServer(t *testing.T) (http.Handler, string, func()) {
	t.Helper()

	resetTimings := queue.SetTimingScaleForTests(0.01)

	tempDir := t.TempDir()
	queueDir := filepath.Join(tempDir, "queue")
	for _, status := range tasks.GetValidStatuses() {
		if err := os.MkdirAll(filepath.Join(queueDir, status), 0o755); err != nil {
			t.Fatalf("failed to create queue dir: %v", err)
		}
	}

	promptsDir := filepath.Join(tempDir, "prompts")
	if err := os.MkdirAll(promptsDir, 0o755); err != nil {
		t.Fatalf("failed to create prompts dir: %v", err)
	}
	sections := []byte("base_sections: []\noperations:\n  resource-generator:\n    additional_sections: []\n")
	if err := os.WriteFile(filepath.Join(promptsDir, "sections.yaml"), sections, 0o644); err != nil {
		t.Fatalf("failed to write sections: %v", err)
	}

	storage := tasks.NewStorage(queueDir)
	assembler, err := prompts.NewAssembler(promptsDir, tempDir)
	if err != nil {
		t.Fatalf("failed to create assembler: %v", err)
	}

	wsManager := websocket.NewManager()
	recyclerSvc := recycler.New(storage, wsManager)
	broadcast := make(chan any, 1)
	processor := queue.NewProcessor(storage, assembler, broadcast, recyclerSvc)

	coord := &tasks.Coordinator{
		LC:          &tasks.Lifecycle{Store: storage},
		Store:       storage,
		Runtime:     processor,
		Broadcaster: wsManager,
	}
	processor.SetCoordinator(coord)
	recyclerSvc.SetCoordinator(coord)

	taskHandlers := handlers.NewTaskHandlers(storage, assembler, processor, wsManager, nil, coord)
	queueHandlers := handlers.NewQueueHandlers(processor, wsManager, storage, coord)
	discoveryHandlers := handlers.NewDiscoveryHandlers(assembler)
	healthHandlers := handlers.NewHealthHandlers(processor, recyclerSvc, queueDir, nil, "test-version")
	settingsHandlers := handlers.NewSettingsHandlers(processor, wsManager, recyclerSvc)
	promptsHandlers := handlers.NewPromptsHandlers(assembler)
	autoSteerHandlers := autosteer.NewAutoSteerHandlers(&autosteer.ProfileService{}, &autosteer.ExecutionEngine{}, &autosteer.HistoryService{})

	app := &Application{
		storage:           storage,
		assembler:         assembler,
		processor:         processor,
		wsManager:         wsManager,
		taskRecycler:      recyclerSvc,
		taskHandlers:      taskHandlers,
		queueHandlers:     queueHandlers,
		discoveryHandlers: discoveryHandlers,
		healthHandlers:    healthHandlers,
		settingsHandlers:  settingsHandlers,
		promptsHandlers:   promptsHandlers,
		autoSteerHandlers: autoSteerHandlers,
		port:              "12345",
		allowedOrigins:    []string{"http://allowed.test"},
	}

	router := app.setupRoutes()

	cleanup := func() {
		processor.Stop()
		processor.Shutdown()
		recyclerSvc.Stop()
		resetTimings()
	}

	return router, queueDir, cleanup
}

func TestSetupRoutesHealthUsesDynamicPath(t *testing.T) {
	router, queueDir, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response map[string]any
	if err := json.NewDecoder(bytes.NewReader(rr.Body.Bytes())).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["version"] != "test-version" {
		t.Fatalf("expected version test-version, got %v", response["version"])
	}

	deps, ok := response["dependencies"].(map[string]any)
	if !ok {
		t.Fatalf("dependencies missing")
	}
	storageDep, ok := deps["storage"].(map[string]any)
	if !ok {
		t.Fatalf("storage dependency missing")
	}
	if storageDep["path"] != queueDir {
		t.Fatalf("expected storage path %s, got %v", queueDir, storageDep["path"])
	}
}

func TestSetupRoutesCORSAllowsConfiguredOrigin(t *testing.T) {
	router, _, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	req.Header.Set("Origin", "http://allowed.test")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from tasks endpoint, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://allowed.test" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
}
