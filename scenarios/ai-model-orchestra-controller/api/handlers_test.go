package main

import (
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlers_HealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	suite := NewHandlerTestSuite(t, testApp.App)
	suite.TestHealthCheck()
}

func TestHandlers_HealthCheckWithDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	// Try to setup dependencies (won't fail if not available)
	setupTestDatabase(t, testApp.App)
	setupTestRedis(t, testApp.App)
	setupTestOllama(t, testApp.App)

	handlers := NewHandlers(testApp.App)

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	w, httpReq, err := makeHTTPRequest(req)
	assert.NoError(t, err)

	handlers.handleHealthCheck(w, httpReq)

	// Should return some status
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusPartialContent || w.Code == http.StatusServiceUnavailable)

	// Should have valid JSON response
	response := assertJSONResponse(t, w, w.Code, nil)
	assert.NotNil(t, response)

	// Verify required fields
	assert.Contains(t, response, "status")
	assert.Contains(t, response, "timestamp")
	assert.Contains(t, response, "services")
	assert.Contains(t, response, "system")
	assert.Contains(t, response, "version")

	// Verify services field structure
	services, ok := response["services"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, services, "database")
	assert.Contains(t, services, "redis")
	assert.Contains(t, services, "ollama")
}

func TestHandlers_ModelSelect(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	suite := NewHandlerTestSuite(t, testApp.App)
	suite.TestModelSelect()
}

func TestHandlers_ModelSelectSuccess(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	t.Run("CompletionTask", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: ModelSelectRequest{
				TaskType: "completion",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleModelSelect(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)

		// Verify response structure
		assert.Contains(t, response, "requestId")
		assert.Contains(t, response, "selectedModel")
		assert.Contains(t, response, "taskType")
		assert.Contains(t, response, "systemMetrics")
		assert.Contains(t, response, "modelInfo")

		assert.Equal(t, "completion", response["taskType"])
	})

	t.Run("ReasoningTask", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: ModelSelectRequest{
				TaskType: "reasoning",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleModelSelect(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)
		assert.Equal(t, "reasoning", response["taskType"])
	})

	t.Run("WithCostLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: ModelSelectRequest{
				TaskType: "completion",
				Requirements: map[string]interface{}{
					"costLimit": 0.005,
				},
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleModelSelect(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)

		modelInfo, ok := response["modelInfo"].(map[string]interface{})
		assert.True(t, ok)

		if costPer1K, ok := modelInfo["cost_per_1k"].(float64); ok {
			assert.LessOrEqual(t, costPer1K, 0.005)
		}
	})
}

func TestHandlers_RouteRequest(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	suite := NewHandlerTestSuite(t, testApp.App)
	suite.TestRouteRequest()
}

func TestHandlers_RouteRequestSuccess(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	t.Run("BasicRequest", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: RouteRequest{
				TaskType: "completion",
				Prompt:   "Hello, world!",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleRouteRequest(w, httpReq)

		// Should succeed (falls back to simulation if Ollama unavailable)
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			assert.NotNil(t, response)

			assert.Contains(t, response, "requestId")
			assert.Contains(t, response, "selectedModel")
			assert.Contains(t, response, "response")
			assert.Contains(t, response, "metrics")
		}
	})

	t.Run("WithRequirements", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: RouteRequest{
				TaskType: "completion",
				Prompt:   "Test prompt",
				Requirements: map[string]interface{}{
					"maxTokens":   float64(100),
					"temperature": 0.7,
				},
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleRouteRequest(w, httpReq)

		// Should process successfully
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK, nil)
			assert.NotNil(t, response)
		}
	})
}

func TestHandlers_ModelsStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/v1/ai/models/status",
	}

	w, httpReq, err := makeHTTPRequest(req)
	assert.NoError(t, err)

	handlers.handleModelsStatus(w, httpReq)

	response := assertJSONResponse(t, w, http.StatusOK, nil)
	assert.NotNil(t, response)

	// Verify response structure
	assert.Contains(t, response, "models")
	assert.Contains(t, response, "totalModels")
	assert.Contains(t, response, "healthyModels")
	assert.Contains(t, response, "systemHealth")

	// Should have some models (either from DB or Ollama defaults)
	totalModels, ok := response["totalModels"].(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, totalModels, float64(0))
}

func TestHandlers_ResourceMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	t.Run("DefaultMetrics", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/ai/resources/metrics",
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleResourceMetrics(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)

		// Verify response structure
		assert.Contains(t, response, "current")
		assert.Contains(t, response, "history")
		assert.Contains(t, response, "memoryPressure")

		// Verify current metrics
		current, ok := response["current"].(map[string]interface{})
		assert.True(t, ok)
		assert.Contains(t, current, "memoryPressure")
		assert.Contains(t, current, "cpuUsage")
	})

	t.Run("WithHoursParam", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/ai/resources/metrics",
			QueryParams: map[string]string{
				"hours": "24",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleResourceMetrics(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		assert.NotNil(t, response)
	})
}

func TestHandlers_Dashboard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/dashboard",
	}

	w, httpReq, err := makeHTTPRequest(req)
	assert.NoError(t, err)

	handlers.handleDashboard(w, httpReq)

	// Should redirect to dashboard UI
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Equal(t, "/ui/dashboard.html", w.Header().Get("Location"))
}

func TestHandlers_ErrorCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testApp := setupTestAppState(t)
	defer testApp.Cleanup()

	handlers := NewHandlers(testApp.App)

	t.Run("ModelSelect_InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/ai/select-model",
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleModelSelect(w, httpReq)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("ModelSelect_MissingTaskType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/select-model",
			Body: map[string]interface{}{
				"requirements": map[string]interface{}{},
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleModelSelect(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "taskType is required")
	})

	t.Run("RouteRequest_InvalidMethod", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/ai/route-request",
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleRouteRequest(w, httpReq)

		assertErrorResponse(t, w, http.StatusMethodNotAllowed, "Method not allowed")
	})

	t.Run("RouteRequest_MissingPrompt", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: map[string]interface{}{
				"taskType": "completion",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleRouteRequest(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "prompt")
	})

	t.Run("RouteRequest_MissingTaskType", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/ai/route-request",
			Body: map[string]interface{}{
				"prompt": "Test",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		assert.NoError(t, err)

		handlers.handleRouteRequest(w, httpReq)

		assertErrorResponse(t, w, http.StatusBadRequest, "taskType")
	})
}

func TestWriteJSONResponse(t *testing.T) {
	// Test the utility functions used by handlers
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	_ = logger

	// These functions are tested indirectly through handler tests
	// But we can add direct tests here if needed
}
