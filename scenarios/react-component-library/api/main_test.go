package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vrooli/scenarios/react-component-library/handlers"
	"github.com/vrooli/scenarios/react-component-library/services"
)

// TestMain sets up and tears down test environment
func TestMain(m *testing.M) {
	// Set test mode
	gin.SetMode(gin.TestMode)
	os.Setenv("GIN_MODE", "test")

	// Disable lifecycle check for tests
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

// setupTestRouter creates a test router with all dependencies
func setupTestRouter(t *testing.T) (*gin.Engine, *TestDatabase) {
	testDB := setupTestDatabase(t)

	// Initialize services
	componentService := services.NewComponentService(testDB.DB)
	testingService := services.NewTestingService()
	aiService := services.NewAIService()
	searchService := services.NewSearchService()

	// Initialize handlers
	componentHandler := handlers.NewComponentHandler(componentService, testingService, aiService, searchService)
	healthHandler := handlers.NewHealthHandler(testDB.DB)

	// Setup router (simplified version of setupRouter from main.go)
	router := gin.New()
	router.Use(gin.Recovery())

	// Health endpoints
	router.GET("/health", healthHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.GET("/components", componentHandler.ListComponents)
		v1.POST("/components", componentHandler.CreateComponent)
		v1.GET("/components/:id", componentHandler.GetComponent)
		v1.PUT("/components/:id", componentHandler.UpdateComponent)
		v1.DELETE("/components/:id", componentHandler.DeleteComponent)
		v1.GET("/components/search", componentHandler.SearchComponents)
		v1.POST("/components/generate", componentHandler.GenerateComponent)
		v1.POST("/components/:id/improve", componentHandler.ImproveComponent)
		v1.POST("/components/:id/test", componentHandler.TestComponent)
		v1.GET("/components/:id/benchmark", componentHandler.BenchmarkComponent)
		v1.POST("/components/:id/export", componentHandler.ExportComponent)
		v1.GET("/components/:id/versions", componentHandler.GetComponentVersions)
		v1.GET("/analytics/usage", componentHandler.GetUsageAnalytics)
		v1.GET("/analytics/popular", componentHandler.GetPopularComponents)
	}

	return router, testDB
}

// ==============================================================================
// Health Endpoint Tests
// ==============================================================================

func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("HealthEndpointReturnsOK", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})
}

// ==============================================================================
// Component CRUD Tests
// ==============================================================================

func TestListComponents(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("ListEmptyComponents", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["components"])
		assert.Equal(t, float64(0), response["total"])
	})

	t.Run("ListComponentsAfterCreation", func(t *testing.T) {
		// Create test component
		createTestComponent(t, testDB.DB, "TestListComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, int(response["total"].(float64)), 1)
	})

	t.Run("ListComponentsWithCategoryFilter", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
			QueryParams: map[string]string{
				"category": "test",
			},
		})

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ListComponentsWithPagination", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components",
			QueryParams: map[string]string{
				"limit":  "10",
				"offset": "0",
			},
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, float64(10), response["limit"])
		assert.Equal(t, float64(0), response["offset"])
	})
}

func TestCreateComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("CreateValidComponent", func(t *testing.T) {
		componentData := getValidComponentData()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["id"])
		assert.Equal(t, componentData.Name, response["name"])
		assert.Equal(t, componentData.Category, response["category"])
	})

	t.Run("CreateComponentWithoutName", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Name = ""

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("CreateComponentWithInvalidCode", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Code = ""

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		// Should fail validation
		assert.NotEqual(t, http.StatusCreated, w.Code)
	})

	t.Run("CreateComponentWithMalformedJSON", func(t *testing.T) {
		// Send invalid JSON
		req, _ := http.NewRequest("POST", "/api/v1/components", nil)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("GetExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestGetComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
		})

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, componentID.String(), response["id"])
	})

	t.Run("GetNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", nonExistentID.String()),
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
		assertErrorResponse(t, w, http.StatusNotFound, "Component not found")
	})

	t.Run("GetComponentWithInvalidUUID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/invalid-uuid",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assertErrorResponse(t, w, http.StatusBadRequest, "Invalid component ID")
	})
}

func TestUpdateComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("UpdateExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestUpdateComponent")

		updateData := map[string]interface{}{
			"description": "Updated description",
			"tags":        []string{"updated", "test"},
		}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
			Body:   updateData,
		})

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("UpdateNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		updateData := map[string]interface{}{
			"description": "Updated description",
		}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/v1/components/%s", nonExistentID.String()),
			Body:   updateData,
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("UpdateComponentWithInvalidUUID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/components/invalid-uuid",
			Body:   map[string]interface{}{"description": "test"},
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestDeleteComponent(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("DeleteExistingComponent", func(t *testing.T) {
		componentID := createTestComponent(t, testDB.DB, "TestDeleteComponent")

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
		})

		assert.Equal(t, http.StatusOK, w.Code)

		// Verify component is deleted
		w = makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID.String()),
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DeleteNonExistentComponent", func(t *testing.T) {
		nonExistentID := uuid.New()

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/v1/components/%s", nonExistentID.String()),
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("DeleteComponentWithInvalidUUID", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/components/invalid-uuid",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// ==============================================================================
// Error Pattern Tests
// ==============================================================================

func TestComponentErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	// Build error test scenarios
	scenarios := NewTestScenarioBuilder().
		AddInvalidUUID("/api/v1/components/%s").
		AddNonExistentComponent("/api/v1/components/%s").
		AddMissingRequiredFields("/api/v1/components", "POST").
		Build()

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var setupData interface{}
			if scenario.Setup != nil {
				setupData = scenario.Setup(t)
			}

			if scenario.Cleanup != nil {
				defer scenario.Cleanup(setupData)
			}

			w := scenario.Execute(t, router, setupData)

			if scenario.Validate != nil {
				scenario.Validate(t, w, setupData)
			}
		})
	}
}

// ==============================================================================
// Integration Tests
// ==============================================================================

func TestComponentLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("FullComponentLifecycle", func(t *testing.T) {
		// Create
		componentData := getValidComponentData()
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		assert.NoError(t, err)

		componentID := createResponse["id"].(string)

		// Read
		w = makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
		})
		assert.Equal(t, http.StatusOK, w.Code)

		// Update
		w = makeHTTPRequest(router, HTTPTestRequest{
			Method: "PUT",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
			Body: map[string]interface{}{
				"description": "Updated description",
			},
		})
		assert.Equal(t, http.StatusOK, w.Code)

		// Delete
		w = makeHTTPRequest(router, HTTPTestRequest{
			Method: "DELETE",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
		})
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify deletion
		w = makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   fmt.Sprintf("/api/v1/components/%s", componentID),
		})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// ==============================================================================
// Business Logic Tests
// ==============================================================================

func TestComponentSearch(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("SearchComponentsEndpointExists", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/components/search",
			QueryParams: map[string]string{
				"query": "button",
			},
		})

		// Endpoint should exist (may return empty results)
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

func TestComponentAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("GetUsageAnalytics", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/usage",
		})

		// Endpoint should exist
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})

	t.Run("GetPopularComponents", func(t *testing.T) {
		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/analytics/popular",
		})

		// Endpoint should exist
		assert.NotEqual(t, http.StatusNotFound, w.Code)
	})
}

// ==============================================================================
// Edge Cases Tests
// ==============================================================================

func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, testDB := setupTestRouter(t)
	defer testDB.Cleanup()

	t.Run("ComponentWithEmptyTags", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Tags = []string{}

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("ComponentWithLongDescription", func(t *testing.T) {
		componentData := getValidComponentData()
		componentData.Description = string(make([]byte, 10000)) // 10KB description

		w := makeHTTPRequest(router, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/components",
			Body:   componentData,
		})

		// Should handle long descriptions
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("ConcurrentComponentCreation", func(t *testing.T) {
		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				componentData := getValidComponentData()
				componentData.Name = fmt.Sprintf("ConcurrentTest%d", index)

				w := makeHTTPRequest(router, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/components",
					Body:   componentData,
				})

				assert.Equal(t, http.StatusCreated, w.Code)
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}
