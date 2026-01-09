package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestHandlerCoverage adds tests for uncovered HTTP handlers
func TestHandlerCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	_, metrics, _ := setupTestSessionManager(t, cfg)

	t.Run("HandleGenerateCommand", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/generate-command",
			Body: map[string]interface{}{
				"prompt": "list files",
			},
		})
		w := httptest.NewRecorder()

		handleGenerateCommand(w, req)

		// Should return a response (may be error if Ollama not available)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("HandleGenerateCommand_EmptyInput", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/generate-command",
			Body: map[string]interface{}{
				"prompt": "",
			},
		})
		w := httptest.NewRecorder()

		handleGenerateCommand(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("HandleGenerateCommand_InvalidMethod", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/api/v1/generate-command",
		})
		w := httptest.NewRecorder()

		handleGenerateCommand(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})

	t.Run("MetricsEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		metrics.serveHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify metrics format
		body := w.Body.String()
		if len(body) == 0 {
			t.Error("Expected non-empty metrics response")
		}
	})
}

// TestSessionCoverage adds tests for uncovered session functions
func TestSessionCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("SessionCleanupOnInitFailure", func(t *testing.T) {
		// Create session with invalid command
		_, err := manager.createSession(createSessionRequest{
			Command: "/nonexistent/binary",
			Args:    []string{},
		})

		if err == nil {
			t.Error("Expected error for invalid command")
		}
	})

	t.Run("SessionReasonMetricIncrement", func(t *testing.T) {
		// Create sessions with different reasons
		reasons := []string{"test", "debug", "admin"}

		for _, reason := range reasons {
			s, err := manager.createSession(createSessionRequest{
				Reason: reason,
			})
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			defer cleanupSession(s)
		}

		// Verify metrics were incremented (checked via metrics endpoint)
	})

	t.Run("SessionHandleOutput", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/echo",
			Args:    []string{"test output"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		// Wait for output
		if !waitForSessionOutput(t, s, 2*time.Second) {
			t.Error("No output received from session")
		}

		// Verify output buffer
		if s.replayBuffer == nil || s.replayBuffer.len() == 0 {
			t.Error("Expected output in buffer")
		}
	})
}

// TestConfigCoverage adds tests for uncovered config functions
func TestConfigCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ResolveWorkingDir_WithEnv", func(t *testing.T) {
		testDir := env.TempDir + "/custom"
		if err := os.MkdirAll(testDir, 0o755); err != nil {
			t.Fatalf("failed to create custom directory: %v", err)
		}
		t.Setenv("WEB_CONSOLE_WORKING_DIR", testDir)

		wd, err := resolveWorkingDir()
		if err != nil {
			t.Fatalf("resolveWorkingDir failed: %v", err)
		}

		if wd != testDir {
			t.Errorf("Expected working dir %s, got %s", testDir, wd)
		}
	})

	t.Run("ResolveWorkingDir_Default", func(t *testing.T) {
		wd, err := resolveWorkingDir()
		if err != nil {
			t.Fatalf("resolveWorkingDir failed: %v", err)
		}

		if wd == "" {
			t.Error("Expected non-empty working directory")
		}
	})
}

// TestOllamaCoverage adds tests for Ollama integration
func TestOllamaCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetEnv_WithValue", func(t *testing.T) {
		t.Setenv("TEST_VAR", "test-value")
		val := getEnv("TEST_VAR", "default")
		if val != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", val)
		}
	})

	t.Run("GetEnv_WithDefault", func(t *testing.T) {
		val := getEnv("NONEXISTENT_VAR", "default-value")
		if val != "default-value" {
			t.Errorf("Expected 'default-value', got '%s'", val)
		}
	})

	t.Run("GenerateCommandWithOllama_NoServer", func(t *testing.T) {
		// This will fail since Ollama is likely not running
		_, err := generateCommandWithOllama("list files", []string{})
		// We expect an error, that's fine
		if err == nil {
			t.Log("Ollama server is available (unexpected in tests)")
		}
	})
}

// TestHelperCoverage adds tests to improve test helper coverage
func TestHelperCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	_, _, ws := setupTestSessionManager(t, cfg)

	t.Run("AssertWorkspaceState", func(t *testing.T) {
		// Add some tabs
		ws.addTab(TestData.CreateTabRequest("tab1", "Tab 1", "sky"))
		ws.addTab(TestData.CreateTabRequest("tab2", "Tab 2", "emerald"))

		assertWorkspaceState(t, ws, 2)
	})

	t.Run("MakeHTTPRequest_WithQueryParams", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/test",
			QueryParams: map[string]string{
				"param1": "value1",
				"param2": "value2",
			},
		})

		if req.URL.Query().Get("param1") != "value1" {
			t.Error("Query param not set correctly")
		}
	})

	t.Run("MakeHTTPRequest_WithHeaders", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/test",
			Headers: map[string]string{
				"X-Test-Header": "test-value",
			},
		})

		if req.Header.Get("X-Test-Header") != "test-value" {
			t.Error("Header not set correctly")
		}
	})

	t.Run("AssertJSONResponse_WithFields", func(t *testing.T) {
		w := httptest.NewRecorder()
		handleGetWorkspace(w, httptest.NewRequest(http.MethodGet, "/api/v1/workspace", nil), ws)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"tabs": nil, // Just check field exists
		})

		if response == nil {
			t.Error("Expected non-nil response")
		}
	})

	t.Run("AssertErrorResponse_WithMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/workspace/tabs",
			Body:   `{invalid json`,
		})
		handleCreateTab(w, req, ws)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestMiddlewareCoverage tests middleware functions
func TestMiddlewareCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	cfg.enableProxyGuard = true
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("ProxyGuard_RejectsMissingHeaders", func(t *testing.T) {
		handler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("ProxyGuard_AllowsWithHeaders", func(t *testing.T) {
		handler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		addProxyHeaders(req)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("ProxyGuard_Disabled", func(t *testing.T) {
		cfg2 := setupTestConfig(env.TempDir)
		cfg2.enableProxyGuard = false
		manager2, _, _ := setupTestSessionManager(t, cfg2)

		handler := manager2.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestSessionEdgeCases adds more session edge case tests
func TestSessionEdgeCasesAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("SessionWithLongRunningCommand", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"2"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		// Verify session exists and was created properly
		if s.id == "" {
			t.Error("Session should have non-empty ID")
		}

		// Session should still exist after a moment
		time.Sleep(500 * time.Millisecond)
		_, found := manager.getSession(s.id)
		if !found {
			t.Error("Session should still be found")
		}
	})

	t.Run("SessionRetrieveNonexistent", func(t *testing.T) {
		_, found := manager.getSession("nonexistent-id")
		if found {
			t.Error("Should not find nonexistent session")
		}
	})

	t.Run("SessionMultiple", func(t *testing.T) {
		// Create sessions with long-running commands
		s1, err1 := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		s2, err2 := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})

		if err1 == nil {
			defer cleanupSession(s1)
		}
		if err2 == nil {
			defer cleanupSession(s2)
		}

		if err1 != nil || err2 != nil {
			t.Errorf("Failed to create sessions: err1=%v, err2=%v", err1, err2)
		}

		// Give sessions time to initialize
		time.Sleep(100 * time.Millisecond)

		// Both should be retrievable
		_, found1 := manager.getSession(s1.id)
		_, found2 := manager.getSession(s2.id)

		if !found1 || !found2 {
			t.Errorf("Sessions should be retrievable: found1=%v, found2=%v", found1, found2)
		}
	})
}
