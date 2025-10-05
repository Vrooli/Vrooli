// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// TestComprehensiveHandlers tests all HTTP handlers comprehensively
func TestComprehensiveHandlers(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	tasksDir = env.TasksDir

	// Setup test database
	testDB := setupTestDB(t)
	if testDB != nil {
		db = testDB
		defer db.Close()
	}

	app := createTestApp()

	// Test all handlers
	t.Run("HealthCheck_Comprehensive", func(t *testing.T) {
		testHealthCheckComprehensive(t, app)
	})

	t.Run("Tasks_Comprehensive", func(t *testing.T) {
		testTasksComprehensive(t, app, env)
	})

	t.Run("Agents_Comprehensive", func(t *testing.T) {
		testAgentsComprehensive(t, app)
	})

	t.Run("Problems_Comprehensive", func(t *testing.T) {
		testProblemsComprehensive(t, app, env)
	})

	t.Run("Metrics_Comprehensive", func(t *testing.T) {
		testMetricsComprehensive(t, app)
	})

	t.Run("Config_Comprehensive", func(t *testing.T) {
		testConfigComprehensive(t, app)
	})
}

func testHealthCheckComprehensive(t *testing.T, app *fiber.App) {
	t.Run("Success_StatusOK", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/health",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": nil,
		})

		if response != nil {
			if status, ok := response["status"].(string); !ok || status == "" {
				t.Error("Expected non-empty status field")
			}
		}
	})

	t.Run("Methods_OnlyGET", func(t *testing.T) {
		methods := []string{"POST", "PUT", "DELETE", "PATCH"}
		for _, method := range methods {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: method,
				Path:   "/health",
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code == http.StatusOK {
				t.Errorf("Method %s should not be allowed on /health", method)
			}
		}
	})
}

func testTasksComprehensive(t *testing.T, app *fiber.App, env *TestEnvironment) {
	t.Run("GET_Tasks_WithFilters", func(t *testing.T) {
		// Create test tasks with different statuses
		testTask1 := setupTestTask(t, env, "bug-fix", "active")
		defer testTask1.Cleanup()

		testTask2 := setupTestTask(t, env, "feature", "backlog")
		defer testTask2.Cleanup()

		// Test status filter
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/tasks",
			QueryParams: map[string]string{
				"status": "active",
			},
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if tasks, ok := response["tasks"].([]interface{}); ok {
			for _, task := range tasks {
				if taskMap, ok := task.(map[string]interface{}); ok {
					if status, ok := taskMap["status"].(string); ok && status != "active" {
						t.Errorf("Expected only active tasks, got status: %s", status)
					}
				}
			}
		}
	})

	t.Run("POST_Tasks_ValidData", func(t *testing.T) {
		taskReq := map[string]interface{}{
			"title":       "Comprehensive Test Task",
			"description": "Testing task creation",
			"type":        "bug-fix",
			"target":      "test-scenario",
			"priority_estimates": map[string]interface{}{
				"impact":        8.0,
				"urgency":       "high",
				"success_prob":  0.85,
				"resource_cost": "moderate",
			},
			"created_by": "test-suite",
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/tasks",
			Body:   taskReq,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("POST_Tasks_ErrorCases", func(t *testing.T) {
		// Use error patterns
		suite := &HandlerTestSuite{
			HandlerName: "CreateTask",
			BaseURL:     "/api/tasks",
		}

		patterns := []ErrorTestPattern{
			invalidJSONPattern("POST", "/api/tasks"),
			emptyBodyPattern("POST", "/api/tasks"),
			missingRequiredFieldPattern("POST", "/api/tasks", "title"),
		}

		suite.RunErrorTests(t, app, patterns)
	})

	t.Run("PUT_Tasks_UpdateFields", func(t *testing.T) {
		testTask := setupTestTask(t, env, "feature", "backlog")
		defer testTask.Cleanup()

		updates := []map[string]interface{}{
			{"notes": "Updated notes"},
			{"status": "active"},
			{"assigned_agent": "test-agent"},
		}

		for i, update := range updates {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "PUT",
				Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
				Body:   update,
			})
			if err != nil {
				t.Fatalf("Update %d failed: %v", i, err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Update %d: Expected status 200, got %d", i, w.Code)
			}
		}
	})

	t.Run("DELETE_Tasks_Cleanup", func(t *testing.T) {
		testTask := setupTestTask(t, env, "bug-fix", "backlog")
		taskPath := testTask.Path
		defer testTask.Cleanup()

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/tasks/%s", testTask.Task.ID),
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Verify file was deleted
		if fileExists(taskPath) {
			t.Error("Expected task file to be deleted")
		}
	})
}

func testAgentsComprehensive(t *testing.T, app *fiber.App) {
	if db == nil {
		t.Skip("Skipping agent tests: database not available")
	}

	t.Run("GET_Agents_EmptyList", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/agents",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"agents": nil,
		})

		if response != nil {
			if agents, ok := response["agents"].([]interface{}); !ok {
				t.Error("Expected agents to be an array")
			} else if len(agents) < 0 {
				t.Error("Agents array should not be nil")
			}
		}
	})

	t.Run("POST_AgentHeartbeat_Valid", func(t *testing.T) {
		agentName := "test-agent-" + uuid.New().String()[:8]
		heartbeatReq := map[string]interface{}{
			"status": "working",
			"resource_usage": map[string]interface{}{
				"cpu":    50.5,
				"memory": 1024,
			},
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/agents/%s/heartbeat", agentName),
			Body:   heartbeatReq,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("POST_AgentHeartbeat_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "AgentHeartbeat",
			BaseURL:     "/api/agents",
		}

		patterns := buildAgentErrorPatterns()
		suite.RunErrorTests(t, app, patterns)
	})
}

func testProblemsComprehensive(t *testing.T, app *fiber.App, env *TestEnvironment) {
	if db == nil {
		t.Skip("Skipping problem tests: database not available")
	}

	t.Run("GET_Problems_EmptyList", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/problems",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"problems": nil,
		})

		if response != nil {
			if problems, ok := response["problems"].([]interface{}); !ok {
				t.Error("Expected problems to be an array")
			} else if len(problems) < 0 {
				t.Error("Problems array should not be nil")
			}
		}
	})

	t.Run("POST_ScanProblems_ValidPath", func(t *testing.T) {
		scanReq := map[string]interface{}{
			"scan_path": env.TempDir,
			"force":     false,
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "POST",
			Path:   "/api/problems/scan",
			Body:   scanReq,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return OK even if no problems found
		if w.Code != http.StatusOK && w.Code != http.StatusAccepted {
			t.Errorf("Expected status 200 or 202, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("POST_ScanProblems_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "ScanProblems",
			BaseURL:     "/api/problems/scan",
		}

		patterns := buildProblemErrorPatterns()
		suite.RunErrorTests(t, app, patterns)
	})
}

func testMetricsComprehensive(t *testing.T, app *fiber.App) {
	if db == nil {
		t.Skip("Skipping metrics tests: database not available")
	}

	t.Run("GET_Metrics_AllFields", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/metrics",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		expectedFields := map[string]interface{}{
			"uptime":        nil,
			"total_tasks":   nil,
			"active_agents": nil,
		}

		response := assertJSONResponse(t, w, http.StatusOK, expectedFields)

		if response != nil {
			// Validate field types
			if _, ok := response["uptime"].(float64); !ok {
				t.Error("Expected uptime to be a number")
			}
			if _, ok := response["total_tasks"].(float64); !ok {
				t.Error("Expected total_tasks to be a number")
			}
			if _, ok := response["active_agents"].(float64); !ok {
				t.Error("Expected active_agents to be a number")
			}
		}
	})

	t.Run("GET_SuccessRate_Valid", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/success-rate",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})
}

func testConfigComprehensive(t *testing.T, app *fiber.App) {
	if db == nil {
		t.Skip("Skipping config tests: database not available")
	}

	t.Run("GET_Config_AllFields", func(t *testing.T) {
		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "GET",
			Path:   "/api/config",
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		expectedFields := map[string]interface{}{
			"max_concurrent_tasks": nil,
			"yolo_mode":            nil,
		}

		assertJSONResponse(t, w, http.StatusOK, expectedFields)
	})

	t.Run("PUT_Config_UpdateFields", func(t *testing.T) {
		configReq := map[string]interface{}{
			"max_concurrent_tasks": 10,
			"yolo_mode":            false,
		}

		w, err := makeFiberRequest(app, FiberTestRequest{
			Method: "PUT",
			Path:   "/api/config",
			Body:   configReq,
		})
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Response: %s", w.Code, w.Body.String())
		}
	})

	t.Run("PUT_Config_ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "UpdateConfig",
			BaseURL:     "/api/config",
		}

		patterns := buildConfigErrorPatterns()
		suite.RunErrorTests(t, app, patterns)
	})
}

// TestCalculatePriorityComprehensive tests priority calculation with various inputs
func TestCalculatePriorityComprehensive(t *testing.T) {
	if db == nil {
		t.Skip("Skipping priority tests: database not available")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	envCleanup := setTestEnvironmentVars()
	defer envCleanup()

	app := createTestApp()

	testCases := []struct {
		name     string
		input    map[string]interface{}
		wantCode int
	}{
		{
			name: "AllFieldsValid",
			input: map[string]interface{}{
				"impact":        8.0,
				"urgency":       "high",
				"success_prob":  0.8,
				"resource_cost": "moderate",
			},
			wantCode: http.StatusOK,
		},
		{
			name: "MinimalFields",
			input: map[string]interface{}{
				"impact": 5.0,
			},
			wantCode: http.StatusOK,
		},
		{
			name: "CriticalUrgency",
			input: map[string]interface{}{
				"impact":        10.0,
				"urgency":       "critical",
				"success_prob":  0.9,
				"resource_cost": "minimal",
			},
			wantCode: http.StatusOK,
		},
		{
			name: "LowPriority",
			input: map[string]interface{}{
				"impact":        2.0,
				"urgency":       "low",
				"success_prob":  0.3,
				"resource_cost": "heavy",
			},
			wantCode: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := makeFiberRequest(app, FiberTestRequest{
				Method: "POST",
				Path:   "/api/calculate-priority",
				Body:   tc.input,
			})
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != tc.wantCode {
				t.Errorf("Expected status %d, got %d. Response: %s", tc.wantCode, w.Code, w.Body.String())
			}

			if w.Code == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if _, ok := response["priority_score"]; !ok {
					t.Error("Expected priority_score in response")
				}
			}
		})
	}
}

// TestUtilityFunctionsComprehensive tests all utility functions
func TestUtilityFunctionsComprehensive(t *testing.T) {
	t.Run("ConversionFunctions", func(t *testing.T) {
		t.Run("UrgencyConversion_AllValues", func(t *testing.T) {
			testCases := []struct {
				input    interface{}
				expected float64
			}{
				{"critical", 4.0},
				{"high", 3.0},
				{"medium", 2.0},
				{"low", 1.0},
				{"CRITICAL", 4.0}, // Case insensitive with ToLower
				{nil, 2.0},
				{123, 123.0}, // Numeric values passed through getFloat
				{"", 2.0},
			}

			for _, tc := range testCases {
				result := convertUrgencyToFloat(tc.input)
				if result != tc.expected {
					t.Errorf("Input %v: expected %f, got %f", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("ResourceCostConversion_AllValues", func(t *testing.T) {
			testCases := []struct {
				input    interface{}
				expected float64
			}{
				{"minimal", 1.0},
				{"moderate", 2.0},
				{"heavy", 3.0},
				{"MINIMAL", 1.0}, // Case insensitive with ToLower
				{nil, 2.0},
				{456, 456.0}, // Numeric values passed through getFloat
				{"", 2.0},
			}

			for _, tc := range testCases {
				result := convertResourceCostToFloat(tc.input)
				if result != tc.expected {
					t.Errorf("Input %v: expected %f, got %f", tc.input, tc.expected, result)
				}
			}
		})
	})

	t.Run("TypeConversionHelpers", func(t *testing.T) {
		t.Run("GetFloat_EdgeCases", func(t *testing.T) {
			testCases := []struct {
				name     string
				input    interface{}
				defVal   float64
				expected float64
			}{
				{"NilValue", nil, 10.0, 10.0},
				{"ValidFloat", 5.5, 10.0, 5.5},
				{"ValidInt", 7, 10.0, 7.0},
				{"String", "invalid", 10.0, 10.0},
				{"ZeroFloat", 0.0, 10.0, 0.0},
				{"NegativeFloat", -5.5, 10.0, -5.5},
				{"LargeFloat", 999999.9, 10.0, 999999.9},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					result := getFloat(tc.input, tc.defVal)
					if result != tc.expected {
						t.Errorf("Expected %f, got %f", tc.expected, result)
					}
				})
			}
		})

		t.Run("GetString_EdgeCases", func(t *testing.T) {
			testCases := []struct {
				name     string
				input    interface{}
				defVal   string
				expected string
			}{
				{"NilValue", nil, "default", "default"},
				{"ValidString", "test", "default", "test"},
				{"EmptyString", "", "default", ""},
				{"IntValue", 123, "default", "default"},
				{"BoolValue", true, "default", "default"},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					result := getString(tc.input, tc.defVal)
					if result != tc.expected {
						t.Errorf("Expected %s, got %s", tc.expected, result)
					}
				})
			}
		})
	})
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
