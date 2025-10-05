package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestHealthCheckHandler tests the health check endpoint
func TestHealthCheckHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	// Setup router with health check endpoint
	if env.Database != nil && env.Database.DB != nil {
		env.Router.GET("/health", func(c *gin.Context) {
			handleHealthCheck(c, env.Database.DB)
		})

		t.Run("Success", func(t *testing.T) {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w := makeHTTPRequest(env.Router, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var response HealthResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)

			assert.Equal(t, "data-structurer-api", response.Service)
			assert.NotEmpty(t, response.Timestamp)
			assert.Contains(t, []string{"healthy", "degraded", "unhealthy"}, response.Status)
			assert.NotNil(t, response.Dependencies)
		})
	}
}

// TestGetSchemasHandler tests the list schemas endpoint
func TestGetSchemasHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.GET("/api/v1/schemas", getSchemas)

	t.Run("Success_ListSchemas", func(t *testing.T) {
		// Create test schema
		schema := createTestSchema(t, env.Database, "list-test")
		assert.NotNil(t, schema)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		schemas, ok := response["schemas"].([]interface{})
		assert.True(t, ok)
		assert.NotEmpty(t, schemas)
	})

	t.Run("Success_EmptyList", func(t *testing.T) {
		// Clean up all test schemas first
		env.Database.DB.Exec("DELETE FROM schemas WHERE name LIKE 'test-%'")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestCreateSchemaHandler tests the create schema endpoint
func TestCreateSchemaHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)

	t.Run("Success_CreateSchema", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "test-create-schema",
			"description": "Test schema creation",
			"schema_definition": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.NotEmpty(t, response["id"])
		assert.Equal(t, "test-create-schema", response["name"])
	})

	t.Run("Error_MissingName", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"description": "Missing name field",
			"schema_definition": map[string]interface{}{
				"type": "object",
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_InvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   `{"invalid": "json"`,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_EmptyBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   "",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// TestGetSchemaHandler tests the get single schema endpoint
func TestGetSchemaHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.GET("/api/v1/schemas/:id", getSchema)

	t.Run("Success_GetSchema", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "get-test")
		assert.NotNil(t, schema)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/" + schema.ID.String(),
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response Schema
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, schema.ID, response.ID)
		assert.Equal(t, schema.Name, response.Name)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/invalid-uuid",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/" + nonExistentID.String(),
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestUpdateSchemaHandler tests the update schema endpoint
func TestUpdateSchemaHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.PUT("/api/v1/schemas/:id", updateSchema)

	t.Run("Success_UpdateSchema", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "update-test")
		assert.NotNil(t, schema)

		updateBody := map[string]interface{}{
			"description": "Updated description",
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/schemas/" + schema.ID.String(),
			Body:   updateBody,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/schemas/invalid-uuid",
			Body:   map[string]interface{}{"description": "test"},
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/schemas/" + nonExistentID.String(),
			Body:   map[string]interface{}{"description": "test"},
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestDeleteSchemaHandler tests the delete schema endpoint
func TestDeleteSchemaHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.DELETE("/api/v1/schemas/:id", deleteSchema)

	t.Run("Success_DeleteSchema", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "delete-test")
		assert.NotNil(t, schema)

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/schemas/" + schema.ID.String(),
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/schemas/invalid-uuid",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/schemas/" + nonExistentID.String(),
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestGetSchemaTemplatesHandler tests the list schema templates endpoint
func TestGetSchemaTemplatesHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.GET("/api/v1/templates", getSchemaTemplates)

	t.Run("Success_ListTemplates", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})

	t.Run("Success_FilterByCategory", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/templates?category=contacts",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestProcessDataHandler tests the data processing endpoint
func TestProcessDataHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/process", processData)

	t.Run("Success_ProcessText", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "process-test")
		assert.NotNil(t, schema)

		requestBody := map[string]interface{}{
			"schema_id":  schema.ID.String(),
			"input_type": "text",
			"input_data": "John Doe is 30 years old",
			"batch_mode": false,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/process",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		// Allow both 200 and 202 (processing may be async)
		assert.Contains(t, []int{http.StatusOK, http.StatusAccepted}, w.Code)
	})

	t.Run("Error_MissingSchemaID", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"input_type": "text",
			"input_data": "test data",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/process",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_InvalidInputType", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "invalid-input-test")
		assert.NotNil(t, schema)

		requestBody := map[string]interface{}{
			"schema_id":  schema.ID.String(),
			"input_type": "invalid-type",
			"input_data": "test data",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/process",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Error_NonExistentSchema", func(t *testing.T) {
		nonExistentID := uuid.New()

		requestBody := map[string]interface{}{
			"schema_id":  nonExistentID.String(),
			"input_type": "text",
			"input_data": "test data",
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/process",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)

		// May return 404 or 400 depending on validation order
		assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound}, w.Code)
	})
}

// TestGetProcessedDataHandler tests the get processed data endpoint
func TestGetProcessedDataHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.GET("/api/v1/data/:schema_id", getProcessedData)

	t.Run("Success_GetProcessedData", func(t *testing.T) {
		schema := createTestSchema(t, env.Database, "processed-data-test")
		assert.NotNil(t, schema)

		processedData := createTestProcessedData(t, env.Database, schema.ID)
		assert.NotNil(t, processedData)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/data/" + schema.ID.String(),
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
	})

	t.Run("Error_InvalidUUID", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/data/invalid-uuid",
		}

		w := makeHTTPRequest(env.Router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
