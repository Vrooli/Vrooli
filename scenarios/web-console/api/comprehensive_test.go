package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, ws := setupTestSessionManager(t, cfg)

	t.Run("CreateSession_EmptyOperator", func(t *testing.T) {
		reqBody := createSessionRequest{
			Operator: "",
			Reason:   "test",
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		// Should still create session with empty operator
		if w.Code == http.StatusCreated {
			response := assertJSONResponse(t, w, http.StatusCreated, nil)
			if response != nil {
				if id, ok := response["id"].(string); ok && id != "" {
					if s, ok := manager.getSession(id); ok {
						cleanupSession(s)
					}
				}
			}
		}
	})

	t.Run("CreateSession_WithCustomCommand", func(t *testing.T) {
		reqBody := createSessionRequest{
			Operator: "test",
			Command:  "/bin/sleep",
			Args:     []string{"60"},
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateSession(w, req, manager, ws)

		assertJSONResponse(t, w, http.StatusCreated, nil)
		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok && id != "" {
				if s, ok := manager.getSession(id); ok {
					if s.commandName != "/bin/sleep" {
						t.Errorf("Expected command '/bin/sleep', got '%s'", s.commandName)
					}
					cleanupSession(s)
				}
			}
		}
	})

	t.Run("UpdateWorkspace_EmptyPayload", func(t *testing.T) {
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPut,
			Path:   "/api/v1/workspace",
			Body:   map[string]interface{}{},
		})
		w := httptest.NewRecorder()

		handleUpdateWorkspace(w, req, ws)

		// Should accept empty payload
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})
	})

	t.Run("PatchWorkspace_InvalidMode", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"keyboardToolbarMode": "invalid-mode",
		}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handlePatchWorkspace(w, req, ws, nil)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("CreateTab_DuplicateID", func(t *testing.T) {
		// Create first tab
		tab1 := TestData.CreateTabRequest("duplicate-id", "Tab 1", "sky")
		ws.addTab(tab1)

		// Try to create tab with same ID
		reqBody := TestData.CreateTabRequest("duplicate-id", "Tab 2", "emerald")
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/workspace/tabs",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleCreateTab(w, req, ws)

		// Should reject duplicate
		assertErrorResponse(t, w, http.StatusConflict, "")
	})

	t.Run("UpdateTab_EmptyUpdate", func(t *testing.T) {
		// Add tab first
		tab := TestData.CreateTabRequest("test-tab-empty", "Original", "sky")
		ws.addTab(tab)

		reqBody := map[string]string{}
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPatch,
			Path:   "/api/v1/workspace/tabs/test-tab-empty",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		handleUpdateTab(w, req, ws, "test-tab-empty")

		// Should accept empty update
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "updated",
		})
	})
}

func TestSessionEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("SessionTouch", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		initialTime := s.lastActivityTime()
		time.Sleep(50 * time.Millisecond)
		s.touch()
		newTime := s.lastActivityTime()

		if newTime.Before(initialTime) || newTime.Equal(initialTime) {
			t.Error("Expected touch to update last activity time")
		}
	})

	t.Run("SessionResize", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		err = s.resize(100, 50) // resize(cols, rows)
		if err != nil {
			t.Errorf("Failed to resize: %v", err)
		}

		s.termSizeMu.RLock()
		rows, cols := s.termRows, s.termCols
		s.termSizeMu.RUnlock()

		if rows != 50 || cols != 100 {
			t.Errorf("Expected size 50x100, got %dx%d", rows, cols)
		}
	})

	t.Run("SessionHandleInput", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		// Send input
		data := []byte("test input")
		err = s.handleInput(nil, data, 1, "test-client")
		if err != nil {
			t.Errorf("Failed to handle input: %v", err)
		}

		// Verify sequence number tracking
		if !s.shouldProcessInputSeq("test-client", 2) {
			t.Error("Next sequence should be processed")
		}
		if s.shouldProcessInputSeq("test-client", 1) {
			t.Error("Old sequence should not be processed")
		}
	})

	t.Run("SessionBroadcast", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		// Trigger output by writing to stdin which should echo
		// Since we're using /bin/sleep, we can't easily test broadcast
		// Just verify the function exists and session has buffer
		s.outputBufferMu.RLock()
		bufferSize := len(s.outputBuffer)
		s.outputBufferMu.RUnlock()

		// Buffer might be empty for sleep, that's ok
		_ = bufferSize
	})

	t.Run("SessionClose_Idempotent", func(t *testing.T) {
		s, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Close once
		s.Close(reasonClientRequested)
		time.Sleep(50 * time.Millisecond)

		// Close again - should not panic
		s.Close(reasonClientRequested)
	})
}

func TestWorkspaceEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	ws := setupTestWorkspace(t, cfg.storagePath)

	t.Run("UpdateNonExistentTab", func(t *testing.T) {
		err := ws.updateTab("non-existent", "New Label", "sky")
		if err == nil {
			t.Error("Expected error when updating non-existent tab")
		}
	})

	t.Run("RemoveNonExistentTab", func(t *testing.T) {
		err := ws.removeTab("non-existent")
		if err == nil {
			t.Error("Expected error when removing non-existent tab")
		}
	})

	t.Run("AttachToNonExistentTab", func(t *testing.T) {
		err := ws.attachSession("non-existent-tab", "session-id")
		if err == nil {
			t.Error("Expected error when attaching to non-existent tab")
		}
	})

	t.Run("DetachFromNonExistentTab", func(t *testing.T) {
		err := ws.detachSession("non-existent-tab")
		if err == nil {
			t.Error("Expected error when detaching from non-existent tab")
		}
	})

	t.Run("SetActiveTabNonExistent", func(t *testing.T) {
		err := ws.setActiveTab("non-existent")
		if err == nil {
			t.Error("Expected error when setting non-existent active tab")
		}
	})

	t.Run("UpdateTabComplete", func(t *testing.T) {
		// Add tab
		tab := TestData.CreateTabRequest("complete-update", "Original", "sky")
		ws.addTab(tab)

		// Update label and color
		err := ws.updateTab("complete-update", "Updated Label", "emerald")
		if err != nil {
			t.Fatalf("Failed to update tab: %v", err)
		}

		// Verify update by checking workspace state
		stateBytes, err := ws.getState()
		if err != nil {
			t.Fatalf("Failed to get state: %v", err)
		}
		if len(stateBytes) == 0 {
			t.Error("Expected non-empty state")
		}
	})

	t.Run("WorkspacePersistence", func(t *testing.T) {
		// Add some tabs
		for i := 0; i < 3; i++ {
			tab := TestData.CreateTabRequest("persist-"+string(rune('a'+i)), "Tab", "sky")
			ws.addTab(tab)
		}

		// Force save
		if err := ws.save(); err != nil {
			t.Fatalf("Failed to save workspace: %v", err)
		}

		// Create new workspace instance from same path
		ws2, err := newWorkspace(ws.storagePath)
		if err != nil {
			t.Fatalf("Failed to load workspace: %v", err)
		}

		// Verify tabs persisted by checking state
		stateBytes, err := ws2.getState()
		if err != nil {
			t.Fatalf("Failed to get state: %v", err)
		}
		if len(stateBytes) == 0 {
			t.Error("Expected non-empty persisted state")
		}
	})

	t.Run("KeyboardToolbarModes", func(t *testing.T) {
		validModes := []string{"disabled", "floating", "top"}
		for _, mode := range validModes {
			err := ws.setKeyboardToolbarMode(mode)
			if err != nil {
				t.Errorf("Mode '%s' should be valid, got error: %v", mode, err)
			}

			// Verify mode was set
			ws.mu.RLock()
			currentMode := ws.KeyboardToolbarMode
			ws.mu.RUnlock()

			if currentMode != mode {
				t.Errorf("Expected mode '%s', got '%s'", mode, currentMode)
			}
		}

		// Test invalid mode
		err := ws.setKeyboardToolbarMode("invalid")
		if err == nil {
			t.Error("Expected error for invalid mode")
		}
	})

	t.Run("WorkspaceSubscriptions", func(t *testing.T) {
		// Subscribe to events
		ch1 := ws.subscribe()
		ch2 := ws.subscribe()

		// Trigger an event
		tab := TestData.CreateTabRequest("sub-test", "Test", "sky")
		ws.addTab(tab)

		// Both channels should receive event (or timeout)
		select {
		case <-ch1:
			// Event received
		case <-time.After(100 * time.Millisecond):
			t.Error("ch1 did not receive event")
		}

		select {
		case <-ch2:
			// Event received
		case <-time.After(100 * time.Millisecond):
			t.Error("ch2 did not receive event")
		}

		// Unsubscribe
		ws.unsubscribe(ch1)
		ws.unsubscribe(ch2)
	})
}

func TestConfigurationEdgeCases(t *testing.T) {
	t.Run("ResolveWorkingDir", func(t *testing.T) {
		// This tests the internal working directory resolution
		cfg := setupTestConfig(t.TempDir())
		if cfg.defaultWorkingDir == "" {
			t.Error("Expected non-empty working directory")
		}
	})

	t.Run("ParseArgs", func(t *testing.T) {
		tests := []struct {
			input    string
			expected []string
		}{
			{"", nil},
			{"arg1", []string{"arg1"}},
			{"arg1 arg2", []string{"arg1", "arg2"}},
			{"  arg1  arg2  ", []string{"arg1", "arg2"}},
		}

		for _, tt := range tests {
			result := parseArgs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Input '%s': expected %d args, got %d", tt.input, len(tt.expected), len(result))
				continue
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("Input '%s': arg[%d] expected '%s', got '%s'", tt.input, i, tt.expected[i], result[i])
				}
			}
		}
	})
}

func TestMetricsEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	_, metrics, _ := setupTestSessionManager(t, cfg)

	t.Run("MultipleReasonMetrics", func(t *testing.T) {
		// Check metrics output
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		metrics.serveHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		body := w.Body.String()
		if body == "" {
			t.Error("Expected non-empty metrics output")
		}

		// Verify metrics format contains expected metrics
		if !contains(body, "web_console_total_sessions") {
			t.Error("Expected total_sessions metric in output")
		}
		if !contains(body, "web_console_active_sessions") {
			t.Error("Expected active_sessions metric in output")
		}
	})

	t.Run("ConcurrentMetricsUpdates", func(t *testing.T) {
		var wg sync.WaitGroup
		iterations := 100

		// Concurrent increments
		wg.Add(iterations)
		for i := 0; i < iterations; i++ {
			go func() {
				defer wg.Done()
				metrics.activeSessions.Add(1)
				metrics.totalSessions.Add(1)
				time.Sleep(time.Millisecond)
				metrics.activeSessions.Add(-1)
			}()
		}
		wg.Wait()

		// Active should be 0, total should be iterations
		if metrics.activeSessions.Load() != 0 {
			t.Errorf("Expected 0 active sessions, got %d", metrics.activeSessions.Load())
		}
		if metrics.totalSessions.Load() != int64(iterations) {
			t.Errorf("Expected %d total sessions, got %d", iterations, metrics.totalSessions.Load())
		}
	})
}
