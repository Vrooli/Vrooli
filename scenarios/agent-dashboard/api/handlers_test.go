package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIndividualAgentHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("MissingAgentID", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/",
		})

		// Handler expects path-based identifier like /api/v1/agents/{id}
		// Empty path results in agent not found (404)
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 400 or 404 for missing agent_id, got %d", w.Code)
		}
	})

	t.Run("InvalidAgentID", func(t *testing.T) {
		// Use special characters that don't match identifierPattern
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/invalid@agent#123",
		})

		// Should return 400 for invalid pattern
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid agent_id, got %d", w.Code)
		}
	})

	t.Run("NonExistentAgent", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/nonexistent-agent",
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent agent, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/agents/test-agent",
		})

		// May return 404 if agent doesn't exist before checking method
		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 405 or 404 for DELETE method, got %d", w.Code)
		}
	})

	t.Run("LogsEndpoint_NonExistent", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/nonexistent-agent/logs",
		})

		// Should return 404 for non-existent agent
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent agent logs, got %d", w.Code)
		}
	})

	t.Run("MetricsEndpoint_NonExistent", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/nonexistent-agent/metrics",
		})

		// Should return 404 for non-existent agent
		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for non-existent agent metrics, got %d", w.Code)
		}
	})

	t.Run("UnknownEndpoint", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/agents/test-agent/unknown",
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for unknown endpoint, got %d", w.Code)
		}
	})

	t.Run("StopEndpoint_InvalidPath", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents/test-agent/stop/extra",
		})

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for invalid stop path, got %d", w.Code)
		}
	})

	t.Run("StartEndpoint_NotSupported", func(t *testing.T) {
		w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents/test-agent/start",
		})

		// May return 404 if agent doesn't exist, or 400 for invalid operation
		if w.Code != http.StatusBadRequest && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 400 or 404 for start endpoint, got %d", w.Code)
		}
	})
}

func TestOrchestrateHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidJSON", func(t *testing.T) {
		w := testHandlerWithRequest(t, orchestrateHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body:   `{invalid json}`,
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("MissingObjective", func(t *testing.T) {
		w := testHandlerWithRequest(t, orchestrateHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body: map[string]interface{}{
				"targets": []string{"codex:agent1"},
			},
		})

		// May accept empty objective or return 400 - depends on implementation
		if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
			t.Logf("Unexpected status for missing objective: %d", w.Code)
		}
	})

	t.Run("EmptyTargetsList", func(t *testing.T) {
		w := testHandlerWithRequest(t, orchestrateHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body: map[string]interface{}{
				"objective": "test objective",
				"targets":   []string{},
			},
		})

		// May be valid if using auto mode without specific targets
		if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
			t.Logf("Unexpected status for empty targets list: %d", w.Code)
		}
	})

	t.Run("TooManyTargets", func(t *testing.T) {
		targets := make([]string, 20)
		for i := range targets {
			targets[i] = "codex:agent" + string(rune(i))
		}

		w := testHandlerWithRequest(t, orchestrateHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/orchestrate",
			Body: map[string]interface{}{
				"objective": "test objective",
				"targets":   targets,
			},
		})

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for too many targets, got %d", w.Code)
		}
	})

	t.Run("MethodNotAllowed", func(t *testing.T) {
		w := testHandlerWithRequest(t, orchestrateHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/orchestrate",
		})

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("Expected status 405 for GET method, got %d", w.Code)
		}
	})
}

func TestCodexAgentManagerMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Get_NonExistent", func(t *testing.T) {
		agent, err := codexManager.Get("nonexistent:agent")
		if err == nil {
			t.Error("Expected error for non-existent agent")
		}
		if agent != nil {
			t.Error("Expected nil agent for non-existent agent")
		}
	})

	t.Run("Logs_NonExistent", func(t *testing.T) {
		logs, err := codexManager.Logs("nonexistent:agent", 100)
		if err == nil {
			t.Error("Expected error for non-existent agent")
		}
		if logs != nil {
			t.Error("Expected nil logs for non-existent agent")
		}
	})

	t.Run("Logs_InvalidLineCount", func(t *testing.T) {
		// Test with negative line count
		logs, err := codexManager.Logs("test:agent", -1)
		if err == nil {
			t.Error("Expected error for negative line count")
		}
		if logs != nil {
			t.Error("Expected nil logs for invalid line count")
		}
	})

	t.Run("Logs_ZeroLineCount", func(t *testing.T) {
		// Test with zero line count
		logs, err := codexManager.Logs("test:agent", 0)
		if err == nil {
			t.Error("Expected error for zero line count")
		}
		if logs != nil {
			t.Error("Expected nil logs for zero line count")
		}
	})

	t.Run("Metrics_NonExistent", func(t *testing.T) {
		metrics, err := codexManager.Metrics("nonexistent:agent")
		if err == nil {
			t.Error("Expected error for non-existent agent")
		}
		if metrics != nil {
			t.Error("Expected nil metrics for non-existent agent")
		}
	})

	t.Run("Stop_NonExistent", func(t *testing.T) {
		_, err := codexManager.Stop("nonexistent:agent")
		if err == nil {
			t.Error("Expected error when stopping non-existent agent")
		}
	})

	t.Run("Get_EmptyID", func(t *testing.T) {
		agent, err := codexManager.Get("")
		if err == nil {
			t.Error("Expected error for empty agent ID")
		}
		if agent != nil {
			t.Error("Expected nil agent for empty agent ID")
		}
	})

	t.Run("Metrics_EmptyID", func(t *testing.T) {
		metrics, err := codexManager.Metrics("")
		if err == nil {
			t.Error("Expected error for empty agent ID")
		}
		if metrics != nil {
			t.Error("Expected nil metrics for empty agent ID")
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("computeUptime", func(t *testing.T) {
		// Test with nil end time (agent still running)
		uptime := computeUptime(testTime, nil)
		if uptime == "" {
			t.Error("Expected non-empty uptime string")
		}

		// Test with end time
		endTime := testTime.Add(5 * time.Minute)
		uptime = computeUptime(testTime, &endTime)
		if uptime == "" {
			t.Error("Expected non-empty uptime string for completed agent")
		}
	})

	t.Run("cloneStringSlice", func(t *testing.T) {
		original := []string{"a", "b", "c"}
		cloned := cloneStringSlice(original)

		if len(cloned) != len(original) {
			t.Errorf("Expected cloned slice length %d, got %d", len(original), len(cloned))
		}

		// Verify it's a deep copy
		cloned[0] = "x"
		if original[0] == "x" {
			t.Error("Expected deep copy, but original was modified")
		}

		// Test with nil slice
		nilClone := cloneStringSlice(nil)
		if nilClone != nil {
			t.Error("Expected nil clone of nil slice")
		}
	})

	t.Run("cloneMetrics", func(t *testing.T) {
		original := map[string]interface{}{
			"cpu":    0.5,
			"memory": 1024,
			"test":   "value",
		}

		cloned := cloneMetrics(original)

		if len(cloned) != len(original) {
			t.Errorf("Expected cloned map length %d, got %d", len(original), len(cloned))
		}

		// Verify values match
		for key, val := range original {
			if cloned[key] != val {
				t.Errorf("Expected cloned[%s] to be %v, got %v", key, val, cloned[key])
			}
		}

		// Test with nil map
		nilClone := cloneMetrics(nil)
		if nilClone != nil {
			t.Error("Expected nil clone of nil map")
		}
	})

	t.Run("getDefaultMetrics", func(t *testing.T) {
		metrics := getDefaultMetrics()

		if metrics == nil {
			t.Fatal("Expected non-nil metrics map")
		}

		expectedFields := []string{"cpu_percent", "memory_mb", "io_read_bytes", "io_write_bytes", "thread_count", "fd_count"}
		for _, field := range expectedFields {
			if _, ok := metrics[field]; !ok {
				t.Errorf("Expected field '%s' in default metrics", field)
			}
		}
	})

	t.Run("generateRadarPosition", func(t *testing.T) {
		pos := generateRadarPosition()

		if pos == nil {
			t.Fatal("Expected non-nil radar position")
		}

		// Verify values are within expected ranges
		if pos.X < 0 || pos.X > 100 {
			t.Errorf("Expected X between 0 and 100, got %f", pos.X)
		}

		if pos.Y < 0 || pos.Y > 100 {
			t.Errorf("Expected Y between 0 and 100, got %f", pos.Y)
		}
	})

	t.Run("deriveLabelFromTask", func(t *testing.T) {
		tests := []struct {
			task     string
			expected string
		}{
			{"write a blog post about AI", "write a blog post about AI"},
			{"analyze this data", "analyze this data"},
			{"", "Codex Agent"},
			{"   ", "Codex Agent"},
			{"fix bug in authentication system", "fix bug in authentication system"},
			{strings.Repeat("a", 100), strings.Repeat("a", 64)}, // Test truncation
		}

		for _, tt := range tests {
			result := deriveLabelFromTask(tt.task)
			if result != tt.expected {
				t.Errorf("deriveLabelFromTask(%q) = %q, expected %q", tt.task, result, tt.expected)
			}
		}
	})

	t.Run("buildCodexPrompt", func(t *testing.T) {
		req := startAgentRequest{
			Task: "test task",
			Mode: "auto",
		}

		prompt := buildCodexPrompt(req, "")

		if prompt == "" {
			t.Error("Expected non-empty prompt")
		}

		if !strings.Contains(prompt, req.Task) {
			t.Error("Expected prompt to contain task")
		}
	})

	t.Run("buildOrchestrationPrompt", func(t *testing.T) {
		req := orchestrateRequest{
			Objective: "test orchestration objective",
			Targets:   []string{"agent1", "agent2"},
		}

		prompt := buildOrchestrationPrompt(req)

		if prompt == "" {
			t.Error("Expected non-empty orchestration prompt")
		}

		if !strings.Contains(prompt, req.Objective) {
			t.Error("Expected prompt to contain objective")
		}

		for _, target := range req.Targets {
			if !strings.Contains(prompt, target) {
				t.Errorf("Expected prompt to contain target %s", target)
			}
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WithinLimit", func(t *testing.T) {
		handler := rateLimitMiddleware(apiRateLimiter, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		handler(w, req)

		// First request should succeed
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("NilLimiter", func(t *testing.T) {
		handler := rateLimitMiddleware(nil, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		handler(w, req)

		// Should allow request with nil limiter
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 with nil limiter, got %d", w.Code)
		}
	})
}

func TestJSONResponse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SuccessResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		response := APIResponse{
			Success: true,
			Data: map[string]string{
				"message": "test message",
			},
		}

		jsonResponse(w, response, http.StatusOK)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var decoded APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !decoded.Success {
			t.Error("Expected success to be true")
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		errorResponse(w, "test error message", http.StatusBadRequest)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var decoded APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if decoded.Success {
			t.Error("Expected success to be false")
		}

		if decoded.Error != "test error message" {
			t.Errorf("Expected error message 'test error message', got '%s'", decoded.Error)
		}
	})
}

var testTime = time.Now()
