package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	metrics := newMetricsRegistry()
	ws := setupTestWorkspace(t, cfg.storagePath)

	manager := newSessionManager(cfg, metrics, ws)

	if manager == nil {
		t.Fatal("Expected non-nil session manager")
	}
	if manager.sessions == nil {
		t.Error("Expected sessions map to be initialized")
	}
	if manager.slots == nil {
		t.Error("Expected slots channel to be initialized")
	}
	if cap(manager.slots) != cfg.maxConcurrent {
		t.Errorf("Expected slots capacity %d, got %d", cfg.maxConcurrent, cap(manager.slots))
	}
}

func TestSessionManagerCreateSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, _ := setupTestSessionManager(t, cfg)

	t.Run("Success", func(t *testing.T) {
		req := createSessionRequest{
			Operator: "test-operator",
			Reason:   "test",
		}

		s, err := manager.createSession(req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		if s.id == "" {
			t.Error("Expected non-empty session ID")
		}
		if s.commandName != cfg.defaultCommand {
			t.Errorf("Expected command '%s', got '%s'", cfg.defaultCommand, s.commandName)
		}

		// Verify session is in manager
		retrieved, ok := manager.getSession(s.id)
		if !ok {
			t.Error("Session not found in manager")
		}
		if retrieved != s {
			t.Error("Retrieved session doesn't match created session")
		}

		// Verify metrics
		if metrics.activeSessions.Load() != 1 {
			t.Errorf("Expected active sessions 1, got %d", metrics.activeSessions.Load())
		}
		if metrics.totalSessions.Load() != 1 {
			t.Errorf("Expected total sessions 1, got %d", metrics.totalSessions.Load())
		}
	})

	t.Run("CustomCommand", func(t *testing.T) {
		req := createSessionRequest{
			Command: "/bin/echo",
			Args:    []string{"hello", "world"},
		}

		s, err := manager.createSession(req)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		defer cleanupSession(s)

		if s.commandName != "/bin/echo" {
			t.Errorf("Expected command '/bin/echo', got '%s'", s.commandName)
		}
		if len(s.commandArgs) != 2 {
			t.Errorf("Expected 2 args, got %d", len(s.commandArgs))
		}
	})

	t.Run("CapacityLimit", func(t *testing.T) {
		sessions := []*session{}
		defer func() {
			for _, s := range sessions {
				if s != nil {
					cleanupSession(s)
				}
			}
		}()

		// Fill up capacity with long-running sessions
		for i := 0; i < cfg.maxConcurrent; i++ {
			s, err := manager.createSession(createSessionRequest{
				Command: "/bin/sleep",
				Args:    []string{"60"},
			})
			if err != nil {
				t.Fatalf("Failed to create session %d: %v", i, err)
			}
			sessions = append(sessions, s)
			// Small delay to ensure session is fully started
			time.Sleep(10 * time.Millisecond)
		}

		// Try to create one more - should fail
		_, err := manager.createSession(createSessionRequest{
			Command: "/bin/sleep",
			Args:    []string{"60"},
		})
		if err == nil {
			t.Error("Expected error when exceeding capacity")
		} else if err.Error() != "session capacity reached" {
			t.Errorf("Expected 'session capacity reached', got '%s'", err.Error())
		}
	})
}

func TestSessionManagerGetSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})
	defer cleanupSession(s)

	t.Run("ExistingSession", func(t *testing.T) {
		retrieved, ok := manager.getSession(s.id)
		if !ok {
			t.Error("Expected to find session")
		}
		if retrieved != s {
			t.Error("Retrieved session doesn't match")
		}
	})

	t.Run("NonExistentSession", func(t *testing.T) {
		_, ok := manager.getSession("non-existent-id")
		if ok {
			t.Error("Expected not to find non-existent session")
		}
	})
}

func TestSessionManagerDeleteSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, _ := setupTestSessionManager(t, cfg)

	s := createTestSession(t, manager, createSessionRequest{})

	t.Run("DeleteExisting", func(t *testing.T) {
		initialActive := metrics.activeSessions.Load()

		manager.deleteSession(s.id, reasonClientRequested)
		time.Sleep(50 * time.Millisecond) // Allow cleanup to complete

		// Verify session removed from manager
		_, ok := manager.getSession(s.id)
		if ok {
			t.Error("Session should be removed from manager")
		}

		// Verify metrics updated
		if metrics.activeSessions.Load() >= initialActive {
			t.Error("Active sessions count should decrease")
		}
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		// Should not panic or error
		manager.deleteSession("non-existent-id", reasonClientRequested)
	})
}

func TestSessionManagerDeleteAllSessions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("DeleteAll", func(t *testing.T) {
		// Create fresh sessions for this test with long-running command
		testSessions := []*session{}
		for i := 0; i < 3; i++ {
			s, err := manager.createSession(createSessionRequest{
				Command: "/bin/sleep",
				Args:    []string{"60"},
			})
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			testSessions = append(testSessions, s)
			time.Sleep(10 * time.Millisecond)
		}

		count := manager.deleteAllSessions(reasonClientRequested)
		if count != 3 {
			t.Errorf("Expected 3 sessions deleted, got %d", count)
		}

		time.Sleep(100 * time.Millisecond) // Allow cleanup to complete

		// Verify all sessions removed
		for _, s := range testSessions {
			if _, ok := manager.getSession(s.id); ok {
				t.Error("Session should be removed")
			}
		}
	})
}

func TestSessionManagerListSummaries(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, _ := setupTestSessionManager(t, cfg)

	t.Run("EmptyList", func(t *testing.T) {
		summaries := manager.listSummaries()
		if len(summaries) != 0 {
			t.Errorf("Expected 0 summaries, got %d", len(summaries))
		}
	})

	t.Run("WithSessions", func(t *testing.T) {
		sessions := []*session{}
		defer func() {
			for _, s := range sessions {
				if s != nil {
					cleanupSession(s)
				}
			}
		}()

		// Create sessions with long-running command
		for i := 0; i < 3; i++ {
			s, err := manager.createSession(createSessionRequest{
				Command: "/bin/sleep",
				Args:    []string{"60"},
			})
			if err != nil {
				t.Fatalf("Failed to create session: %v", err)
			}
			sessions = append(sessions, s)
			time.Sleep(10 * time.Millisecond)
		}

		summaries := manager.listSummaries()
		if len(summaries) != 3 {
			t.Errorf("Expected 3 summaries, got %d", len(summaries))
		}

		// Verify summary fields
		for _, summary := range summaries {
			if summary.ID == "" {
				t.Error("Summary ID should not be empty")
			}
			if summary.State != "active" {
				t.Errorf("Expected state 'active', got '%s'", summary.State)
			}
			if summary.CreatedAt.IsZero() {
				t.Error("CreatedAt should not be zero")
			}
		}
	})
}

func TestSessionManagerProxyGuard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ProxyGuardEnabled", func(t *testing.T) {
		cfg := setupTestConfig(env.TempDir)
		cfg.enableProxyGuard = true
		manager, _, _ := setupTestSessionManager(t, cfg)

		handler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Without proxy headers - should fail
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusForbidden, "")

		// With proxy headers - should succeed
		req2 := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})
		addProxyHeaders(req2)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)

		if w2.Code != http.StatusOK {
			t.Errorf("Expected status 200 with proxy headers, got %d", w2.Code)
		}
	})

	t.Run("ProxyGuardDisabled", func(t *testing.T) {
		cfg := setupTestConfig(env.TempDir)
		cfg.enableProxyGuard = false
		manager, _, _ := setupTestSessionManager(t, cfg)

		handler := manager.makeProxyGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Without proxy headers - should still succeed
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodGet,
			Path:   "/test",
		})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 without guard, got %d", w.Code)
		}
	})
}

func TestSessionManagerOnSessionClosed(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, _, ws := setupTestSessionManager(t, cfg)

	t.Run("DetachesFromWorkspace", func(t *testing.T) {
		// Create tab
		tab := TestData.CreateTabRequest("tab-1", "Test Tab", "sky")
		_ = ws.addTab(tab)

		// Create session and attach to tab
		s := createTestSession(t, manager, createSessionRequest{})
		_ = ws.attachSession("tab-1", s.id)

		// Verify session attached
		ws.mu.RLock()
		attached := ws.Tabs[0].SessionID != nil && *ws.Tabs[0].SessionID == s.id
		ws.mu.RUnlock()
		if !attached {
			t.Error("Session should be attached to tab")
		}

		// Close session
		s.Close(reasonClientRequested)
		time.Sleep(50 * time.Millisecond) // Allow cleanup

		// Verify session detached
		ws.mu.RLock()
		detached := ws.Tabs[0].SessionID == nil
		ws.mu.RUnlock()
		if !detached {
			t.Error("Session should be detached from tab after close")
		}
	})
}
