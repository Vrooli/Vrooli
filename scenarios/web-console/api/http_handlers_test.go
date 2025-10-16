package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestHealthzHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	mux := http.NewServeMux()
	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, ws := setupTestSessionManager(t, cfg)

	registerRoutes(mux, manager, metrics, ws)

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/healthz",
		})
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "ok",
		})
		if response != nil {
			if note, ok := response["note"].(string); ok {
				if note == "" {
					t.Error("Expected note field to be non-empty")
				}
			}
		}
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("SuccessfulWrite", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]string{"test": "value"}
		writeJSON(w, http.StatusOK, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
		}

		var result map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}
		if result["test"] != "value" {
			t.Errorf("Expected value 'value', got '%s'", result["test"])
		}
	})
}

func TestWriteJSONError(t *testing.T) {
	t.Run("ErrorWrite", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeJSONError(w, http.StatusBadRequest, "test error")

		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})
}

func TestHandleGetWorkspace(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	_, _, ws := setupTestSessionManager(t, cfg)

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/api/v1/workspace",
		})
		w := httptest.NewRecorder()

		handleGetWorkspace(w, req, ws)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Header().Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json")
		}
	})
}

func TestHandleUpdateWorkspace(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	_, _, ws := setupTestSessionManager(t, cfg)

	// Add a tab first
	tab := TestData.CreateTabRequest("tab-1", "Test Tab", "sky")
	_ = ws.addTab(tab)

	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"activeTabId": "tab-1",
			"tabs": []tabMeta{
				{ID: "tab-1", Label: "Updated Tab", ColorID: "emerald"},
			},
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPut,
			Path:   "/api/v1/workspace",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleUpdateWorkspace(w, req, ws)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPut,
			Path:   "/api/v1/workspace",
			Body:   `{"invalid": "json"`,
		})
		w := httptest.NewRecorder()

		handleUpdateWorkspace(w, req, ws)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})
}

func TestHandlePatchWorkspace(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, ws := setupTestSessionManager(t, cfg)

	t.Run("Success", func(t *testing.T) {
		mode := "floating"
		reqBody := map[string]interface{}{
			"keyboardToolbarMode": mode,
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handlePatchWorkspace(w, req, ws, manager)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})
	})

	t.Run("InvalidMode", func(t *testing.T) {
		mode := "invalid-mode"
		reqBody := map[string]interface{}{
			"keyboardToolbarMode": mode,
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handlePatchWorkspace(w, req, ws, manager)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace",
			Body:   `{"invalid"`,
		})
		w := httptest.NewRecorder()

		handlePatchWorkspace(w, req, ws, manager)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})

	t.Run("IdleTimeoutUpdate", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"idleTimeoutSeconds": 900,
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handlePatchWorkspace(w, req, ws, manager)

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})

		if got := ws.idleTimeoutDuration(); got != 15*time.Minute {
			t.Fatalf("Expected idle timeout duration 15m, got %v", got)
		}
		if manager.getIdleTimeout() != 15*time.Minute {
			t.Fatalf("Expected manager idle timeout 15m, got %v", manager.getIdleTimeout())
		}
	})
}

func TestHandleCreateTab(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	t.Run("Success", func(t *testing.T) {
		tab := TestData.CreateTabRequest("tab-1", "New Tab", "sky")
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/workspace/tabs",
			Body:   tab,
		})
		w := httptest.NewRecorder()

		handleCreateTab(w, req, ws)

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok {
				if id != "tab-1" {
					t.Errorf("Expected tab ID 'tab-1', got '%s'", id)
				}
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/workspace/tabs",
			Body:   `{"invalid"`,
		})
		w := httptest.NewRecorder()

		handleCreateTab(w, req, ws)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})
}

func TestHandleUpdateTab(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	// Add a tab first
	tab := TestData.CreateTabRequest("tab-1", "Test Tab", "sky")
	_ = ws.addTab(tab)

	t.Run("Success", func(t *testing.T) {
		reqBody := map[string]string{
			"label":   "Updated Label",
			"colorId": "emerald",
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace/tabs/tab-1",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleUpdateTab(w, req, ws, "tab-1")

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})
	})

	t.Run("TabNotFound", func(t *testing.T) {
		reqBody := map[string]string{
			"label":   "Updated Label",
			"colorId": "emerald",
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace/tabs/non-existent",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleUpdateTab(w, req, ws, "non-existent")

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace/tabs/tab-1",
			Body:   `{"invalid"`,
		})
		w := httptest.NewRecorder()

		handleUpdateTab(w, req, ws, "tab-1")

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})
}

func TestHandleDeleteTab(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	// Add a tab first
	tab := TestData.CreateTabRequest("tab-1", "Test Tab", "sky")
	_ = ws.addTab(tab)

	t.Run("Success", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodDelete,
			Path:   "/api/v1/workspace/tabs/tab-1",
		})
		w := httptest.NewRecorder()

		handleDeleteTab(w, req, ws, "tab-1")

		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "deleted",
		})
	})

	t.Run("TabNotFound", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodDelete,
			Path:   "/api/v1/workspace/tabs/non-existent",
		})
		w := httptest.NewRecorder()

		handleDeleteTab(w, req, ws, "non-existent")

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

func TestHandleCreateSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, ws := setupTestSessionManager(t, cfg)

	t.Run("Success", func(t *testing.T) {
		reqBody := TestData.CreateSessionRequest("", nil)
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok {
				if id == "" {
					t.Error("Expected non-empty session ID")
				}
				// Cleanup session
				if s, ok := manager.getSession(id); ok {
					cleanupSession(s)
				}
			}
		}
	})

	t.Run("SuccessWithTab", func(t *testing.T) {
		// Add tab first
		tab := TestData.CreateTabRequest("tab-1", "Test Tab", "sky")
		_ = ws.addTab(tab)

		reqBody := createSessionRequest{
			Operator: "test",
			TabID:    "tab-1",
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok {
				// Cleanup session
				if s, ok := manager.getSession(id); ok {
					cleanupSession(s)
				}
			}
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   `{"invalid"`,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})

	t.Run("CapacityReached", func(t *testing.T) {
		// Create sessions up to capacity with long-running command
		sessions := []*session{}
		for i := 0; i < cfg.maxConcurrent; i++ {
			s := createTestSession(t, manager, createSessionRequest{
				Command: "/bin/sleep",
				Args:    []string{"60"},
			})
			sessions = append(sessions, s)
			time.Sleep(10 * time.Millisecond)
		}

		// Cleanup sessions at end
		defer func() {
			for _, s := range sessions {
				cleanupSession(s)
			}
		}()

		// Try to create one more
		reqBody := TestData.CreateSessionRequest("", nil)
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		assertErrorResponse(t, w, http.StatusTooManyRequests, "")

		// Cleanup sessions
		for _, s := range sessions {
			cleanupSession(s)
		}
		time.Sleep(100 * time.Millisecond)
	})
}

func TestListSessionsIncludesCapacity(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, ws := setupTestSessionManager(t, cfg)
	mux := http.NewServeMux()
	registerRoutes(mux, manager, metrics, ws)

	// Seed one session so list has data
	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	headerValue := w.Header().Get("X-Session-Capacity")
	if headerValue != strconv.Itoa(cfg.maxConcurrent) {
		t.Fatalf("expected capacity header %d, got %q", cfg.maxConcurrent, headerValue)
	}

	var payload struct {
		Sessions []sessionSummary `json:"sessions"`
		Capacity int              `json:"capacity"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if payload.Capacity != cfg.maxConcurrent {
		t.Fatalf("expected payload capacity %d, got %d", cfg.maxConcurrent, payload.Capacity)
	}

	if len(payload.Sessions) == 0 {
		t.Fatal("expected at least one session entry in payload")
	}
}

func TestHandleGenerateCommand(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingPrompt", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"context": []string{"test"},
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/generate-command",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleGenerateCommand(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "prompt is required")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/generate-command",
			Body:   `{"invalid"`,
		})
		w := httptest.NewRecorder()

		handleGenerateCommand(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid payload")
	})
}
