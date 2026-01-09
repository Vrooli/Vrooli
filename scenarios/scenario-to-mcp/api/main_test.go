//go:build testing
// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	suite := &HandlerTestSuite{
		Name:       "HealthEndpoint",
		TestServer: ts,
		SuccessTests: []SuccessTestCase{
			{
				Name: "BasicHealth",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				},
				ExpectedStatus: http.StatusOK,
				Validate: func(t *testing.T, resp map[string]interface{}) {
					if status, ok := resp["status"].(string); !ok || status != "healthy" {
						t.Errorf("Expected status 'healthy', got %v", resp["status"])
					}
					if _, ok := resp["timestamp"]; !ok {
						t.Error("Expected timestamp in response")
					}
					if service, ok := resp["service"].(string); !ok || service != "scenario-to-mcp" {
						t.Errorf("Expected service 'scenario-to-mcp', got %v", resp["service"])
					}
				},
			},
		},
	}

	suite.RunAllTests(t)
}

// TestGetEndpoints tests the MCP endpoints listing
func TestGetEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("Success", func(t *testing.T) {
		// Create mock detector output
		mockOutput := `[
			{"name": "test-scenario-1", "hasMCP": true, "confidence": "high", "status": "active"},
			{"name": "test-scenario-2", "hasMCP": false, "confidence": "low", "status": "inactive"}
		]`

		libPath := filepath.Join(env.TempDir, "scenarios", "scenario-to-mcp", "lib")
		createMockDetectorScript(t, libPath, mockOutput)

		// Create mock endpoints in DB
		createMockEndpoint(t, db, "test-scenario-1", 4000)

		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/endpoints",
		})

		assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
			if _, ok := resp["scenarios"]; !ok {
				t.Error("Expected 'scenarios' in response")
			}
			if _, ok := resp["summary"]; !ok {
				t.Error("Expected 'summary' in response")
			}
			return nil
		})
	})

	t.Run("DetectorError", func(t *testing.T) {
		// Don't create detector script - should cause error
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/endpoints",
		})

		// Should return error when detector fails
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})
}

// TestAddMCP tests the MCP addition endpoint
func TestAddMCP(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	suite := &HandlerTestSuite{
		Name:       "AddMCP",
		TestServer: ts,
		SuccessTests: []SuccessTestCase{
			{
				Name: "ValidRequest",
				Request: HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body: AddMCPRequest{
						ScenarioName: "test-scenario",
						AgentConfig: AgentConfig{
							AutoDetect: true,
						},
					},
				},
				ExpectedStatus: http.StatusOK,
				Validate: func(t *testing.T, resp map[string]interface{}) {
					if success, ok := resp["success"].(bool); !ok || !success {
						t.Error("Expected success to be true")
					}
					if _, ok := resp["agent_session_id"]; !ok {
						t.Error("Expected agent_session_id in response")
					}
					if _, ok := resp["estimated_time"]; !ok {
						t.Error("Expected estimated_time in response")
					}
				},
			},
		},
		ErrorTests: []ErrorTestPattern{
			{
				Name:        "MissingScenarioName",
				Description: "Should reject request without scenario name",
				Request: HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body: AddMCPRequest{
						AgentConfig: AgentConfig{AutoDetect: true},
					},
				},
				ExpectedStatus: http.StatusBadRequest,
				ValidateError: func(t *testing.T, resp map[string]interface{}) {
					if success, ok := resp["success"].(bool); ok && success {
						t.Error("Expected success to be false")
					}
				},
			},
			{
				Name:        "InvalidJSON",
				Description: "Should reject malformed JSON",
				Request: HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body:   `{"invalid": json}`,
				},
				ExpectedStatus: http.StatusBadRequest,
			},
		},
	}

	suite.RunAllTests(t)
}

// TestGetRegistry tests the registry endpoint
func TestGetRegistry(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("EmptyRegistry", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/registry",
		})

		assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
			if version, ok := resp["version"].(string); !ok || version != "1.0" {
				t.Errorf("Expected version '1.0', got %v", resp["version"])
			}
			if endpoints, ok := resp["endpoints"].([]interface{}); !ok {
				t.Error("Expected 'endpoints' array in response")
			} else if len(endpoints) != 0 {
				t.Errorf("Expected 0 endpoints, got %d", len(endpoints))
			}
			return nil
		})
	})

	t.Run("WithEndpoints", func(t *testing.T) {
		// Create active endpoints
		createMockEndpoint(t, db, "scenario-1", 4001)
		createMockEndpoint(t, db, "scenario-2", 4002)

		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/registry",
		})

		assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
			endpoints, ok := resp["endpoints"].([]interface{})
			if !ok {
				t.Fatal("Expected 'endpoints' array in response")
			}

			if len(endpoints) < 1 {
				t.Errorf("Expected at least 1 endpoint, got %d", len(endpoints))
			}

			// Validate endpoint structure
			for i, ep := range endpoints {
				endpoint, ok := ep.(map[string]interface{})
				if !ok {
					t.Errorf("Endpoint %d is not an object", i)
					continue
				}

				requiredFields := []string{"name", "transport", "url", "manifest_url"}
				for _, field := range requiredFields {
					if _, exists := endpoint[field]; !exists {
						t.Errorf("Endpoint %d missing field: %s", i, field)
					}
				}
			}
			return nil
		})
	})
}

// TestGetScenarioDetails tests the scenario details endpoint
func TestGetScenarioDetails(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("ValidScenario", func(t *testing.T) {
		// Create mock detector output
		mockOutput := `{"name": "test-scenario", "hasMCP": true, "confidence": "high"}`
		libPath := filepath.Join(env.TempDir, "scenarios", "scenario-to-mcp", "lib")
		createMockDetectorScript(t, libPath, mockOutput)

		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/mcp/scenarios/test-scenario",
			URLVars: map[string]string{"name": "test-scenario"},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		var result map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if name, ok := result["name"].(string); !ok || name != "test-scenario" {
			t.Errorf("Expected name 'test-scenario', got %v", result["name"])
		}
	})
}

// TestGetSession tests the session retrieval endpoint
func TestGetSession(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("ExistingSession", func(t *testing.T) {
		sessionID := "test-session-123"
		createMockSession(t, db, sessionID, "test-scenario", "completed")

		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/mcp/sessions/" + sessionID,
			URLVars: map[string]string{"id": sessionID},
		})

		assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
			if id, ok := resp["id"].(string); !ok || id != sessionID {
				t.Errorf("Expected id '%s', got %v", sessionID, resp["id"])
			}
			if status, ok := resp["status"].(string); !ok || status != "completed" {
				t.Errorf("Expected status 'completed', got %v", resp["status"])
			}
			return nil
		})
	})

	t.Run("NonExistentSession", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method:  "GET",
			Path:    "/api/v1/mcp/sessions/nonexistent",
			URLVars: map[string]string{"id": "nonexistent"},
		})

		assertErrorResponse(t, w, http.StatusNotFound, "Session not found")
	})
}

func TestDocumentationEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	createTestDocFile(t, env.TempDir, filepath.Join("docs", "intro.md"), "# Intro\nScenario documentation sample")
	createTestDocFile(t, env.TempDir, "README.md", "# Scenario Overview\nOverview details")

	ts := setupTestServer(t, nil, env.TempDir)

	w := makeHTTPRequest(ts, HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/docs",
	})

	var docID string

	assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
		docsValue, ok := resp["docs"].([]interface{})
		if !ok {
			t.Fatalf("Expected docs array in response, got: %T", resp["docs"])
		}

		if len(docsValue) == 0 {
			t.Fatal("Expected at least one documentation entry")
		}

		for _, item := range docsValue {
			entry, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("Invalid documentation entry type: %T", item)
			}

			if path, ok := entry["relativePath"].(string); ok && path == "docs/intro.md" {
				if id, ok := entry["id"].(string); ok {
					docID = id
				}
			}
		}

		return nil
	})

	if docID == "" {
		t.Fatal("Expected to capture document id for docs/intro.md")
	}

	encodedID := url.QueryEscape(docID)
	w = makeHTTPRequest(ts, HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/docs/content?id=" + encodedID,
	})

	assertJSONResponse(t, w, http.StatusOK, func(resp map[string]interface{}) error {
		content, ok := resp["content"].(string)
		if !ok {
			t.Fatal("Expected content string in response")
		}

		if content == "" {
			t.Error("Expected document content to be returned")
		}

		return nil
	})

	w = makeHTTPRequest(ts, HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/docs/content?id=invalid",
	})

	assertErrorResponse(t, w, http.StatusNotFound, "")
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("OptionsRequest", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", origin)
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected CORS methods header to be set")
		}
	})

	t.Run("CORSHeadersOnGet", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", origin)
		}
	})
}

// TestServerInitialization tests server initialization
func TestServerInitialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ValidConfig", func(t *testing.T) {
		_, dbCleanup := setupTestDatabase(t)
		defer dbCleanup()

		config := &Config{
			APIPort:       3290,
			RegistryPort:  3292,
			DatabaseURL:   "test-url",
			ScenariosPath: "/tmp/scenarios",
		}

		server := NewServer(config)
		if server == nil {
			t.Fatal("Expected server to be created")
		}

		if server.config != config {
			t.Error("Server config not set correctly")
		}

		if server.router == nil {
			t.Error("Router should be initialized")
		}
	})
}

// TestGetEnvHelpers tests environment variable helper functions
func TestGetEnvHelpers(t *testing.T) {
	t.Run("GetEnvString", func(t *testing.T) {
		os.Setenv("TEST_STRING_VAR", "test-value")
		defer os.Unsetenv("TEST_STRING_VAR")

		if val := getEnvString("TEST_STRING_VAR", "default"); val != "test-value" {
			t.Errorf("Expected 'test-value', got '%s'", val)
		}

		if val := getEnvString("NONEXISTENT_VAR", "default"); val != "default" {
			t.Errorf("Expected 'default', got '%s'", val)
		}
	})

	t.Run("GetEnvInt", func(t *testing.T) {
		os.Setenv("TEST_INT_VAR", "42")
		defer os.Unsetenv("TEST_INT_VAR")

		if val := getEnvInt("TEST_INT_VAR", 0); val != 42 {
			t.Errorf("Expected 42, got %d", val)
		}

		if val := getEnvInt("NONEXISTENT_VAR", 10); val != 10 {
			t.Errorf("Expected 10, got %d", val)
		}

		os.Setenv("TEST_INVALID_INT", "not-a-number")
		defer os.Unsetenv("TEST_INVALID_INT")

		if val := getEnvInt("TEST_INVALID_INT", 5); val != 5 {
			t.Errorf("Expected default 5 for invalid int, got %d", val)
		}
	})
}

// TestPerformance tests API performance
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	RunPerformanceTest(t, ts, PerformanceTestPattern{
		Name:        "HealthCheckPerformance",
		Description: "Health check should respond quickly",
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		},
		MaxDuration: 100 * time.Millisecond,
	})

	RunPerformanceTest(t, ts, PerformanceTestPattern{
		Name:        "RegistryPerformance",
		Description: "Registry should respond within 50ms",
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/registry",
		},
		MaxDuration: 50 * time.Millisecond,
	})
}

// TestIntegrationWorkflow tests complete MCP addition workflow
func TestIntegrationWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	RunIntegrationTest(t, ts, IntegrationTestPattern{
		Name:        "CompleteMCPAdditionWorkflow",
		Description: "Test complete flow from adding MCP to checking status",
		Steps: []IntegrationStep{
			{
				Name: "CheckInitialRegistry",
				Request: HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/mcp/registry",
				},
				Validate: func(t *testing.T, resp map[string]interface{}) interface{} {
					if version, ok := resp["version"].(string); !ok || version != "1.0" {
						t.Errorf("Expected version 1.0, got %v", resp["version"])
					}
					return nil
				},
			},
			{
				Name: "AddMCPToScenario",
				Request: HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body: AddMCPRequest{
						ScenarioName: "integration-test-scenario",
						AgentConfig:  AgentConfig{AutoDetect: true},
					},
				},
				Validate: func(t *testing.T, resp map[string]interface{}) interface{} {
					sessionID, ok := resp["agent_session_id"].(string)
					if !ok || sessionID == "" {
						t.Fatal("Expected session ID in response")
					}
					return sessionID
				},
			},
		},
	})
}

// TestEdgeCases tests edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("EmptyScenarioName", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/mcp/add",
			Body: AddMCPRequest{
				ScenarioName: "",
				AgentConfig:  AgentConfig{AutoDetect: true},
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Scenario name is required")
	})

	t.Run("VeryLongScenarioName", func(t *testing.T) {
		longName := string(make([]byte, 1000))
		for i := range longName {
			longName = string(append([]byte(longName[:i]), 'a'))
		}

		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/mcp/add",
			Body: AddMCPRequest{
				ScenarioName: longName,
				AgentConfig:  AgentConfig{AutoDetect: true},
			},
		})

		// Should either accept or reject gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Unexpected status for long name: %d", w.Code)
		}
	})

	t.Run("SpecialCharactersInScenarioName", func(t *testing.T) {
		specialNames := []string{
			"scenario/with/slashes",
			"scenario-with-dashes",
			"scenario_with_underscores",
			"scenario.with.dots",
			"scenario with spaces",
		}

		for _, name := range specialNames {
			t.Run(name, func(t *testing.T) {
				w := makeHTTPRequest(ts, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body: AddMCPRequest{
						ScenarioName: name,
						AgentConfig:  AgentConfig{AutoDetect: true},
					},
				})

				// Should handle gracefully (either accept or reject with proper error)
				if w.Code >= 500 {
					t.Errorf("Server error for special chars '%s': %d", name, w.Code)
				}
			})
		}
	})

	t.Run("ConcurrentSessionCreation", func(t *testing.T) {
		done := make(chan bool, 3)
		sessionIDs := make(chan string, 3)

		for i := 0; i < 3; i++ {
			go func(index int) {
				w := makeHTTPRequest(ts, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/mcp/add",
					Body: AddMCPRequest{
						ScenarioName: fmt.Sprintf("concurrent-scenario-%d", index),
						AgentConfig:  AgentConfig{AutoDetect: true},
					},
				})

				if w.Code == http.StatusOK {
					var resp map[string]interface{}
					json.Unmarshal(w.Body.Bytes(), &resp)
					if sessionID, ok := resp["agent_session_id"].(string); ok {
						sessionIDs <- sessionID
					}
				}
				done <- true
			}(i)
		}

		for i := 0; i < 3; i++ {
			<-done
		}
		close(sessionIDs)

		// Verify all session IDs are unique
		seen := make(map[string]bool)
		for id := range sessionIDs {
			if seen[id] {
				t.Errorf("Duplicate session ID created: %s", id)
			}
			seen[id] = true
		}
	})

	t.Run("NullFieldsInJSON", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/mcp/add",
			Body:   `{"scenario_name": null, "agent_config": null}`,
		})

		// Should handle null values gracefully
		if w.Code == http.StatusOK {
			t.Error("Should not accept null scenario_name")
		}
	})
}

// TestDatabaseErrors tests database error handling
func TestDatabaseErrors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("DatabaseConnectionFailure", func(t *testing.T) {
		// Try to create server with invalid database URL
		config := &Config{
			APIPort:       3290,
			DatabaseURL:   "invalid://connection",
			ScenariosPath: env.TempDir,
		}

		server := NewServer(config)
		err := server.Initialize()

		if err == nil {
			t.Error("Expected error with invalid database URL")
		}
	})
}

// TestAPIResponseFormats tests various response formats
func TestAPIResponseFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("ContentTypeHeaders", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
	})

	t.Run("ErrorResponseFormat", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/mcp/add",
			Body:   `invalid json`,
		})

		var resp APIResponse
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Error response not in correct format: %v", err)
		}

		if resp.Success {
			t.Error("Error response should have success=false")
		}

		if resp.Error == "" {
			t.Error("Error response should have error message")
		}
	})
}

// TestLoadScenarios tests load handling
func TestLoadScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	RunLoadTest(t, ts, LoadTestPattern{
		Name:        "HealthCheckLoad",
		Description: "Health check should handle concurrent requests",
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		},
		Concurrency:    10,
		TotalRequests:  100,
		MaxAvgDuration: 50 * time.Millisecond,
	})

	RunLoadTest(t, ts, LoadTestPattern{
		Name:        "RegistryLoad",
		Description: "Registry should handle concurrent requests",
		Request: HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/mcp/registry",
		},
		Concurrency:    5,
		TotalRequests:  50,
		MaxAvgDuration: 100 * time.Millisecond,
	})
}

// TestSecurityHeaders tests security-related headers and validations
func TestSecurityHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	db, dbCleanup := setupTestDatabase(t)
	defer dbCleanup()

	ts := setupTestServer(t, db, env.TempDir)

	t.Run("CORSPreflight", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "OPTIONS",
			Path:   "/api/v1/mcp/add",
		})

		if w.Code != http.StatusOK {
			t.Errorf("OPTIONS request should return 200, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("CORS origin header not set correctly")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("CORS methods header not set")
		}
	})

	t.Run("InvalidMethodRejection", func(t *testing.T) {
		w := makeHTTPRequest(ts, HTTPTestRequest{
			Method: "PATCH",
			Path:   "/api/v1/health",
		})

		// Should reject or handle gracefully
		if w.Code == http.StatusOK {
			t.Error("Should not accept PATCH on GET-only endpoint")
		}
	})
}
