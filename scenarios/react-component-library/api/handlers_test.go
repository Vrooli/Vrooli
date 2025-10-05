package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// ==============================================================================
// Handler Tests - Component Testing
// ==============================================================================

func TestComponentTestEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("TestComponentWithAccessibility", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestAccessibilityComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/test", componentID.String()),
			Body: map[string]interface{}{
				"test_types": []string{"accessibility"},
			},
		})

		// Endpoint should exist and handle request
		assert.NotEqual(t, http.StatusNotFound, w.Code)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
		}
	})

	t.Run("TestComponentWithPerformance", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestPerformanceComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/test", componentID.String()),
			Body: map[string]interface{}{
				"test_types": []string{"performance"},
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("TestNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/test", nonExistentID.String()),
			Body: map[string]interface{}{
				"test_types": []string{"accessibility"},
			},
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("TestWithInvalidTestType", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestInvalidTypeComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/test", componentID.String()),
			Body: map[string]interface{}{
				"test_types": []string{"invalid_type"},
			},
		})

		// Should return bad request or similar error
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("TestWithMultipleTypes", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestMultiTypeComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/test", componentID.String()),
			Body: map[string]interface{}{
				"test_types": []string{"accessibility", "performance"},
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

func TestComponentBenchmarkEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("BenchmarkExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "BenchmarkComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s/benchmark", componentID.String()),
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("BenchmarkNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s/benchmark", nonExistentID.String()),
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestComponentExportEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("ExportAsNpmPackage", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "ExportNpmComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/export", componentID.String()),
			Body: map[string]interface{}{
				"format": "npm-package",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("ExportAsRawCode", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "ExportRawComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/export", componentID.String()),
			Body: map[string]interface{}{
				"format": "raw-code",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("ExportWithInvalidFormat", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "ExportInvalidComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/export", componentID.String()),
			Body: map[string]interface{}{
				"format": "invalid-format",
			},
		})

		// Should handle gracefully
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})
}

func TestComponentVersionsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("GetVersionsForExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "VersionedComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s/versions", componentID.String()),
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
		}
	})

	t.Run("GetVersionsForNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s/versions", nonExistentID.String()),
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestGenerateComponentEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("GenerateSimpleButton", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components/generate",
			Body: map[string]interface{}{
				"description": "A simple button component",
				"requirements": []string{"accessible", "customizable"},
			},
		})

		// Endpoint should exist (may fail if AI service unavailable)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("GenerateWithEmptyDescription", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components/generate",
			Body: map[string]interface{}{
				"description": "",
			},
		})

		// Should return bad request
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("GenerateWithStylePreferences", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components/generate",
			Body: map[string]interface{}{
				"description": "A modal dialog",
				"style_preferences": map[string]interface{}{
					"theme": "minimal",
				},
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

func TestImproveComponentEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("ImproveExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "ImproveComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/improve", componentID.String()),
			Body: map[string]interface{}{
				"focus": "accessibility",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("ImproveNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/improve", nonExistentID.String()),
			Body: map[string]interface{}{
				"focus": "performance",
			},
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("ImproveWithMultipleFocusAreas", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "ImproveMultiComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   fmt.Sprintf("/api/v1/components/%s/improve", componentID.String()),
			Body: map[string]interface{}{
				"focus": []string{"accessibility", "performance", "code-quality"},
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// ==============================================================================
// Analytics Endpoint Tests
// ==============================================================================

func TestAnalyticsEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("UsageAnalyticsWithDateRange", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/usage",
			QueryParams: map[string]string{
				"start_date": "2025-01-01",
				"end_date":   "2025-12-31",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
		}
	})

	t.Run("PopularComponentsWithLimit", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/popular",
			QueryParams: map[string]string{
				"limit": "10",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.NotNil(t, response["components"])
		}
	})

	t.Run("PopularComponentsByCategory", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/popular",
			QueryParams: map[string]string{
				"category": "form",
				"limit":    "5",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// ==============================================================================
// Search Endpoint Tests
// ==============================================================================

func TestSearchEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	// Create some test components for searching
	createTestComponent(t, testDB.DB, "SearchableButton")
	createTestComponent(t, testDB.DB, "SearchableModal")

	t.Run("SearchWithQuery", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query": "button",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)

		if w.Code == http.StatusOK {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.NotNil(t, response["components"])
		}
	})

	t.Run("SearchWithCategoryFilter", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query":    "component",
				"category": "test",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("SearchWithMinAccessibilityScore", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query":                   "component",
				"min_accessibility_score": "0.8",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("SearchWithEmptyQuery", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query": "",
			},
		})

		// Should either return all or require query
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SearchWithTags", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query": "button",
				"tags":  "form,interactive",
			},
		})

		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// ==============================================================================
// Error Handling Tests
// ==============================================================================

func TestHandlerErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("InvalidJSONPayload", func(t *testing.T) {
		// Send malformed JSON to create endpoint
		req, _ := http.NewRequest("POST", "/api/v1/components", nil)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("UnsupportedHTTPMethod", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "PATCH",
			Path:   "/api/v1/components",
		})

		// Should return method not allowed
		assert.NotEqual(t, http.StatusOK, w.Code)
	})

	t.Run("ExtremelyLargePayload", func(t *testing.T) {
		// Create a very large component payload
		largeCode := string(make([]byte, 1024*1024)) // 1MB of code

		componentData := getValidComponentData()
		componentData.Code = largeCode

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		// Should handle gracefully (either accept or reject with proper error)
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SpecialCharactersInComponentName", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Name = "Test<>Component&\"'🚀"

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		// Should either sanitize or reject with proper error
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("SQLInjectionAttempt", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Name = "Test'; DROP TABLE components; --"

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		// Should handle safely (parameterized queries should prevent SQL injection)
		// Component might be created or rejected, but should not cause server error
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)

		// Verify database integrity - table should still exist
		var count int
		err := testDB.DB.QueryRow("SELECT COUNT(*) FROM components").Scan(&count)
		assert.NoError(t, err, "Components table should still exist")
	})
}
