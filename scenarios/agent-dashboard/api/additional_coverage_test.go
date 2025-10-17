package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestGetAgentDetails tests the getAgentDetails function
func TestGetAgentDetails(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonExistentAgent", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/agents/nonexistent", nil)
		if err != nil {
			t.Fatal(err)
		}

		w := httptest.NewRecorder()
		getAgentDetails(w, req, "nonexistent")

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("ValidAgent", func(t *testing.T) {
		// Create a mock agent in the manager
		tempDir, _ := os.MkdirTemp("", "test-agent-details")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		// Manually add a test agent
		testAgent := &Agent{
			ID:       "test:123",
			Name:     "Test Agent",
			Type:     "codex",
			Status:   "completed",
			PID:      12345,
			StartTime: time.Now(),
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}
		close(managed.done)

		manager.mu.Lock()
		manager.agents["test:123"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("GET", "/api/v1/agents/test:123", nil)
		w := httptest.NewRecorder()
		getAgentDetails(w, req, "test:123")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response Agent
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.ID != "test:123" {
			t.Errorf("Expected ID test:123, got %s", response.ID)
		}
	})
}

// TestGetAgentLogs tests the getAgentLogs function
func TestGetAgentLogs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidLineCount", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agents/test:123/logs?lines=invalid", nil)
		w := httptest.NewRecorder()
		getAgentLogs(w, req, "test:123")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("LineCountTooHigh", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agents/test:123/logs?lines=99999", nil)
		w := httptest.NewRecorder()
		getAgentLogs(w, req, "test:123")

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("ValidRequest", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-agent-logs")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID:     "test:456",
			Status: "running",
		}

		managed := &managedAgent{
			agent: testAgent,
			logs:  []string{"log line 1", "log line 2", "log line 3"},
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:456"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("GET", "/api/v1/agents/test:456/logs?lines=2", nil)
		w := httptest.NewRecorder()
		getAgentLogs(w, req, "test:456")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var response APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}
	})
}

// TestGetAgentMetrics tests the getAgentMetrics function
func TestGetAgentMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonExistentAgent", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/agents/nonexistent/metrics", nil)
		w := httptest.NewRecorder()
		getAgentMetrics(w, req, "nonexistent")

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("ValidAgent", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-agent-metrics")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID:      "test:789",
			Status:  "completed",
			Metrics: getDefaultMetrics(),
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}
		close(managed.done)

		manager.mu.Lock()
		manager.agents["test:789"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("GET", "/api/v1/agents/test:789/metrics", nil)
		w := httptest.NewRecorder()
		getAgentMetrics(w, req, "test:789")

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var metrics map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &metrics); err != nil {
			t.Fatalf("Failed to unmarshal metrics: %v", err)
		}

		// Verify expected metric keys exist
		expectedKeys := []string{"cpu_percent", "memory_mb", "io_read_bytes", "io_write_bytes"}
		for _, key := range expectedKeys {
			if _, exists := metrics[key]; !exists {
				t.Errorf("Expected metric key %s not found", key)
			}
		}
	})
}

// TestAppendLog tests the appendLog function
func TestAppendLog(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BasicAppend", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-append-log")
		defer os.RemoveAll(tempDir)

		logPath := filepath.Join(tempDir, "test.log")
		logFile, _ := os.Create(logPath)

		managed := &managedAgent{
			agent: &Agent{
				LastSeen: time.Now(),
			},
			logFile: logFile,
			logs:    []string{},
		}

		managed.appendLog("stdout", "test log line")

		if len(managed.logs) != 1 {
			t.Errorf("Expected 1 log line, got %d", len(managed.logs))
		}

		if !strings.Contains(managed.logs[0], "[STDOUT] test log line") {
			t.Errorf("Log line format incorrect: %s", managed.logs[0])
		}

		logFile.Close()
	})

	t.Run("LogBufferLimit", func(t *testing.T) {
		managed := &managedAgent{
			agent: &Agent{
				LastSeen: time.Now(),
			},
			logs: []string{},
		}

		// Add more than maxLogBufferLines
		for i := 0; i < maxLogBufferLines+100; i++ {
			managed.appendLog("stdout", "test line")
		}

		if len(managed.logs) > maxLogBufferLines {
			t.Errorf("Expected log buffer to be limited to %d, got %d", maxLogBufferLines, len(managed.logs))
		}
	})
}

// TestCodexManagerLogs tests the Logs method
func TestCodexManagerLogs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonExistentAgent", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-manager-logs")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")

		_, err := manager.Logs("nonexistent", 100)
		if err == nil {
			t.Error("Expected error for nonexistent agent")
		}
	})

	t.Run("ValidAgent", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-manager-logs")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")

		testAgent := &Agent{
			ID: "test:logs",
		}

		managed := &managedAgent{
			agent: testAgent,
			logs:  []string{"line1", "line2", "line3", "line4", "line5"},
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:logs"] = managed
		manager.mu.Unlock()

		logs, err := manager.Logs("test:logs", 3)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(logs) != 3 {
			t.Errorf("Expected 3 logs, got %d", len(logs))
		}

		// Should return last 3 logs
		if logs[0] != "line3" {
			t.Errorf("Expected first log to be line3, got %s", logs[0])
		}
	})
}

// TestCodexManagerMetrics tests the Metrics method
func TestCodexManagerMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonExistentAgent", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-manager-metrics")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")

		_, err := manager.Metrics("nonexistent")
		if err == nil {
			t.Error("Expected error for nonexistent agent")
		}
	})

	t.Run("CompletedAgent", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-manager-metrics")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")

		testAgent := &Agent{
			ID:      "test:metrics",
			Status:  "completed",
			Metrics: getDefaultMetrics(),
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:metrics"] = managed
		manager.mu.Unlock()

		metrics, err := manager.Metrics("test:metrics")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if metrics == nil {
			t.Error("Expected metrics to be returned")
		}
	})
}

// TestIndividualAgentHandlerAdditionalCases tests additional edge cases for individualAgentHandler
func TestIndividualAgentHandlerAdditionalCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AgentIDTooLong", func(t *testing.T) {
		longID := strings.Repeat("a", 101)
		req, _ := http.NewRequest("GET", "/api/v1/agents/"+longID, nil)
		w := httptest.NewRecorder()
		individualAgentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("UnknownEndpoint", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-unknown-endpoint")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID:   "test:endpoint",
			Name: "Test",
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:endpoint"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("GET", "/api/v1/agents/test:endpoint/unknown", nil)
		w := httptest.NewRecorder()
		individualAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("StartEndpointPost", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-start-endpoint")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID: "test:start",
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:start"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("POST", "/api/v1/agents/test:start/start", nil)
		w := httptest.NewRecorder()
		individualAgentHandler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("InvalidPath", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-invalid-path")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID: "test:path",
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:path"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("POST", "/api/v1/agents/test:path", nil)
		w := httptest.NewRecorder()
		individualAgentHandler(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		tempDir, _ := os.MkdirTemp("", "test-method-not-allowed")
		defer os.RemoveAll(tempDir)

		manager, _ := newCodexAgentManager(tempDir, defaultAgentTimeout, "")
		codexManager = manager

		testAgent := &Agent{
			ID: "test:method",
		}

		managed := &managedAgent{
			agent: testAgent,
			done:  make(chan struct{}),
		}

		manager.mu.Lock()
		manager.agents["test:method"] = managed
		manager.mu.Unlock()

		req, _ := http.NewRequest("PUT", "/api/v1/agents/test:method", nil)
		w := httptest.NewRecorder()
		individualAgentHandler(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405, got %d", w.Code)
		}
	})
}

// TestRateLimitMiddlewareNilCase tests the nil limiter case
func TestRateLimitMiddlewareNilCase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NilLimiter", func(t *testing.T) {
		handler := rateLimitMiddleware(nil, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with nil limiter, got %d", w.Code)
		}
	})
}

// TestSetupTestDirectory tests the test helper function
func TestSetupTestDirectory(t *testing.T) {
	env := setupTestDirectory(t)
	defer env.Cleanup()

	if env.TempDir == "" {
		t.Error("Expected temp directory to be created")
	}

	if _, err := os.Stat(env.TempDir); os.IsNotExist(err) {
		t.Error("Temp directory does not exist")
	}

	// Test cleanup
	env.Cleanup()
	if _, err := os.Stat(env.TempDir); !os.IsNotExist(err) {
		t.Error("Temp directory should be removed after cleanup")
	}
}
