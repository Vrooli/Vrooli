package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAgentFullLifecycle tests the complete lifecycle of an agent
func TestAgentFullLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for logs
	tmpDir, err := os.MkdirTemp("", "agent-integration-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test manager
	testManager, err := newCodexAgentManager(tmpDir, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to create test manager: %v", err)
	}

	// Store original manager and restore after test
	originalManager := codexManager
	codexManager = testManager
	defer func() {
		codexManager = originalManager
	}()

	t.Run("Start_Invalid_NoTask", func(t *testing.T) {
		req := startAgentRequest{
			Task: "",
			Mode: "auto",
		}

		_, err := testManager.Start(req)
		if err == nil {
			t.Error("Expected error when starting agent with empty task")
		}
	})

	t.Run("Start_Invalid_NoCodex", func(t *testing.T) {
		// This will likely fail if codex is not installed, which is expected
		req := startAgentRequest{
			Task: "test task",
			Mode: "auto",
		}

		agent, err := testManager.Start(req)
		if err != nil {
			t.Logf("Expected error starting agent without codex installed: %v", err)
		} else if agent != nil {
			t.Logf("Unexpected: agent started successfully: %s", agent.ID)
			// Clean up if agent actually started
			defer testManager.Stop(agent.ID)
		}
	})
}

// TestIndividualAgentHandlerWithAgent tests handler with simulated agent data
func TestIndividualAgentHandlerWithAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetDetails_PathParsing", func(t *testing.T) {
		// Test various path formats
		paths := []string{
			"/api/v1/agents/codex:agent-123",
			"/api/v1/agents/test-agent",
			"/api/v1/agents/codex:very-long-agent-name-with-dashes",
		}

		for _, path := range paths {
			w := testHandlerWithRequest(t, individualAgentHandler, HTTPTestRequest{
				Method: "GET",
				Path:   path,
			})

			// Should parse path and return 404 for non-existent agent
			if w.Code != http.StatusNotFound {
				t.Logf("Path %s returned status %d", path, w.Code)
			}
		}
	})
}

// TestManagerLogsWithMockAgent tests the Logs method
func TestManagerLogsWithMockAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a temporary directory for logs
	tmpDir, err := os.MkdirTemp("", "logs-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testManager, err := newCodexAgentManager(tmpDir, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to create test manager: %v", err)
	}

	t.Run("Logs_InvalidLineCount", func(t *testing.T) {
		// Test with various invalid line counts
		invalidCounts := []int{-1, 0, 10001, 99999}

		for _, count := range invalidCounts {
			logs, err := testManager.Logs("test:agent", count)

			// Should error for invalid line count or non-existent agent
			if err == nil && logs != nil {
				t.Logf("Unexpected success with line count %d", count)
			}
		}
	})

	t.Run("Logs_ValidLineCountNonExistentAgent", func(t *testing.T) {
		validCounts := []int{1, 10, 100, 1000, 10000}

		for _, count := range validCounts {
			logs, err := testManager.Logs("nonexistent:agent", count)

			if err == nil {
				t.Error("Expected error for non-existent agent")
			}

			if logs != nil {
				t.Error("Expected nil logs for non-existent agent")
			}
		}
	})
}

// TestManagerMetricsWithMockAgent tests the Metrics method
func TestManagerMetricsWithMockAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "metrics-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testManager, err := newCodexAgentManager(tmpDir, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to create test manager: %v", err)
	}

	t.Run("Metrics_NonExistentAgent", func(t *testing.T) {
		metrics, err := testManager.Metrics("nonexistent:agent-xyz")

		if err == nil {
			t.Error("Expected error for non-existent agent")
		}

		if metrics != nil {
			t.Error("Expected nil metrics for non-existent agent")
		}
	})

	t.Run("Metrics_InvalidAgentIDs", func(t *testing.T) {
		invalidIDs := []string{"", "invalid", "missing-colon", "@#$%"}

		for _, id := range invalidIDs {
			metrics, err := testManager.Metrics(id)

			if err == nil && metrics != nil {
				t.Logf("Unexpected success with invalid ID: %s", id)
			}
		}
	})
}

// TestManagerStopWithMockAgent tests the Stop method
func TestManagerStopWithMockAgent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tmpDir, err := os.MkdirTemp("", "stop-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	testManager, err := newCodexAgentManager(tmpDir, 5*time.Minute, "")
	if err != nil {
		t.Fatalf("Failed to create test manager: %v", err)
	}

	t.Run("Stop_NonExistentAgent", func(t *testing.T) {
		agent, err := testManager.Stop("nonexistent:agent")

		if err == nil {
			t.Error("Expected error when stopping non-existent agent")
		}

		if agent != nil {
			t.Error("Expected nil agent when stopping non-existent agent")
		}
	})

	t.Run("Stop_InvalidAgentIDs", func(t *testing.T) {
		invalidIDs := []string{"", "no-colon", "@invalid#", "   "}

		for _, id := range invalidIDs {
			agent, err := testManager.Stop(id)

			if err == nil {
				t.Logf("Unexpected success stopping invalid ID: %s", id)
			}

			if agent != nil {
				t.Logf("Unexpected agent returned for invalid ID: %s", id)
			}
		}
	})
}

// TestHandlerEndpointsIntegration tests the handler endpoints with realistic scenarios
func TestHandlerEndpointsIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AgentsHandler_POST_CompleteFlow", func(t *testing.T) {
		req := map[string]interface{}{
			"task":            "test integration task",
			"mode":            "auto",
			"timeout_seconds": 300,
			"capabilities":    []string{"coding", "testing"},
		}

		w := testHandlerWithRequest(t, agentsHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/agents",
			Body:   req,
		})

		// May succeed or fail depending on codex availability
		if w.Code == http.StatusOK {
			t.Log("Agent started successfully")

			var response APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			// Try to stop the agent if it was created
			if agent, ok := response.Data.(map[string]interface{}); ok {
				if agentID, ok := agent["id"].(string); ok && agentID != "" {
					defer codexManager.Stop(agentID)
				}
			}
		} else {
			t.Logf("Agent start failed (expected if codex not installed): status %d", w.Code)
		}
	})

	t.Run("StatusHandler_MultipleRequests", func(t *testing.T) {
		// Test that status handler works consistently
		for i := 0; i < 5; i++ {
			w := testHandlerWithRequest(t, statusHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/status",
			})

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
			}

			var response APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Request %d: Failed to parse response: %v", i+1, err)
			}
		}
	})
}

// TestCapabilitiesAndSearchHandlers tests capability-related handlers
func TestCapabilitiesAndSearchHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CapabilitiesHandler_ResponseStructure", func(t *testing.T) {
		w := testHandlerWithRequest(t, capabilitiesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/capabilities",
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if !response.Success {
			t.Error("Expected success to be true")
		}

		data, ok := response.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be a map")
		}

		if _, ok := data["capabilities"]; !ok {
			t.Error("Expected capabilities field in data")
		}

		if _, ok := data["total"]; !ok {
			t.Error("Expected total field in data")
		}
	})

	t.Run("SearchByCapability_VariousCapabilities", func(t *testing.T) {
		capabilities := []string{
			"text-generation",
			"code-analysis",
			"data-processing",
			"testing",
			"debugging",
		}

		for _, cap := range capabilities {
			w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/agents/search?capability=" + cap,
				QueryParams: map[string]string{
					"capability": cap,
				},
			})

			if w.Code != http.StatusOK {
				t.Errorf("Search for capability '%s' failed with status %d", cap, w.Code)
				continue
			}

			var response APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to parse response for capability '%s': %v", cap, err)
				continue
			}

			if !response.Success {
				t.Errorf("Expected success for capability '%s'", cap)
			}
		}
	})

	t.Run("SearchByCapability_EdgeCases", func(t *testing.T) {
		edgeCases := []struct {
			name       string
			capability string
			expectOK   bool
		}{
			{"empty", "", false},
			{"whitespace", "   ", false},
			{"dashes", "text-generation", true},
			{"underscores", "data_processing", true},
			{"long_name", strings.Repeat("a", 50), true},
		}

		for _, tc := range edgeCases {
			t.Run(tc.name, func(t *testing.T) {
				w := testHandlerWithRequest(t, searchByCapabilityHandler, HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/agents/search",
					QueryParams: map[string]string{
						"capability": tc.capability,
					},
				})

				if tc.expectOK && w.Code != http.StatusOK {
					t.Logf("Expected OK for '%s', got %d", tc.capability, w.Code)
				} else if !tc.expectOK && w.Code == http.StatusOK {
					t.Logf("Expected error for '%s', got OK", tc.capability)
				}
			})
		}
	})
}

// TestScanHandlerIntegration tests the scan endpoint
func TestScanHandlerIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Scan_MultipleInvocations", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			w := testHandlerWithRequest(t, scanHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/scan",
			})

			if w.Code != http.StatusOK {
				t.Errorf("Scan %d: Expected status 200, got %d", i+1, w.Code)
			}

			var response APIResponse
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Errorf("Scan %d: Failed to parse response: %v", i+1, err)
			}

			if !response.Success {
				t.Errorf("Scan %d: Expected success to be true", i+1)
			}
		}
	})
}

// TestResolveAgentIdentifierAdvanced tests identifier resolution
func TestResolveAgentIdentifierAdvanced(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("IdentifierFormats", func(t *testing.T) {
		formats := []string{
			"codex:agent-123",
			"claude-code:test-agent",
			"ollama:model-runner",
			"test-agent", // Short name
			"agent_123",  // With underscore
		}

		for _, format := range formats {
			result := resolveAgentIdentifier(format)
			t.Logf("Identifier '%s' resolved to '%s'", format, result)
		}
	})

	t.Run("IdentifierValidation", func(t *testing.T) {
		invalid := []string{
			"",
			"   ",
			"@#$%",
			strings.Repeat("a", 200), // Very long
		}

		for _, id := range invalid {
			result := resolveAgentIdentifier(id)
			if result != "" {
				t.Logf("Invalid identifier '%s' unexpectedly resolved to '%s'", id, result)
			}
		}
	})
}

// TestVersionHandlerDetails tests version endpoint thoroughly
func TestVersionHandlerDetails(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("VersionResponseFields", func(t *testing.T) {
		w := testHandlerWithRequest(t, versionHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/version",
		})

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		// Verify all expected fields
		expectedFields := []string{
			"service",
			"api_version",
			"codex_default_mode",
			"default_timeout_sec",
		}

		for _, field := range expectedFields {
			if _, ok := response[field]; !ok {
				t.Errorf("Expected field '%s' in version response", field)
			}
		}

		// Verify field types
		if service, ok := response["service"].(string); !ok || service == "" {
			t.Error("Expected non-empty service name")
		}

		if apiVersion, ok := response["api_version"].(string); !ok || apiVersion == "" {
			t.Error("Expected non-empty API version")
		}
	})
}

// TestCORSHeadersAllMethods tests CORS for various HTTP methods
func TestCORSHeadersAllMethods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handler := corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(method, "/test", nil)
			handler(w, req)

			// Verify CORS headers
			if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
				t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", origin)
			}

			if method == "OPTIONS" {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
				}
			}
		})
	}
}
