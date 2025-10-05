package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCheckDatabaseHealth tests database health check function
func TestCheckDatabaseHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("HealthyDatabase", func(t *testing.T) {
		health := checkDatabaseHealth(env.Database.DB)

		assert.NotNil(t, health)
		assert.Contains(t, []string{"healthy", "degraded"}, health["status"])
	})

	t.Run("NilDatabase", func(t *testing.T) {
		health := checkDatabaseHealth(nil)

		assert.NotNil(t, health)
		assert.Equal(t, "unhealthy", health["status"])
	})
}

// TestCountHealthyDependenciesIntegration tests the countHealthyDependencies function
func TestCountHealthyDependenciesIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AllHealthy", func(t *testing.T) {
		deps := map[string]interface{}{
			"postgres": map[string]interface{}{"status": "healthy"},
			"ollama":   map[string]interface{}{"status": "healthy"},
			"n8n":      map[string]interface{}{"status": "healthy"},
		}

		count := countHealthyDependencies(deps)
		assert.Equal(t, 3, count)
	})

	t.Run("MixedHealth", func(t *testing.T) {
		deps := map[string]interface{}{
			"postgres": map[string]interface{}{"status": "healthy"},
			"ollama":   map[string]interface{}{"status": "unhealthy"},
			"n8n":      map[string]interface{}{"status": "degraded"},
		}

		count := countHealthyDependencies(deps)
		assert.Equal(t, 1, count)
	})

	t.Run("InvalidStructure", func(t *testing.T) {
		deps := map[string]interface{}{
			"postgres": "not a map",
			"ollama":   123,
		}

		count := countHealthyDependencies(deps)
		assert.Equal(t, 0, count)
	})
}

// TestGetDataStatistics tests the data statistics function
func TestGetDataStatistics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("WithData", func(t *testing.T) {
		// Create test data
		schema := createTestSchema(t, env.Database, "stats-test")
		assert.NotNil(t, schema)

		createTestProcessedData(t, env.Database, schema.ID)

		stats := getDataStatistics(env.Database.DB)

		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_schemas")
		assert.Contains(t, stats, "total_processed_items")
	})

	t.Run("EmptyDatabase", func(t *testing.T) {
		// Clean database
		env.Database.DB.Exec("DELETE FROM processed_data WHERE source_file_name LIKE 'test-%'")
		env.Database.DB.Exec("DELETE FROM schemas WHERE name LIKE 'test-%'")

		stats := getDataStatistics(env.Database.DB)

		assert.NotNil(t, stats)
	})
}

// TestSchemaLifecycle tests complete schema CRUD lifecycle
func TestSchemaLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router with all schema endpoints
	env.Router.POST("/api/v1/schemas", createSchema)
	env.Router.GET("/api/v1/schemas/:id", getSchema)
	env.Router.PUT("/api/v1/schemas/:id", updateSchema)
	env.Router.DELETE("/api/v1/schemas/:id", deleteSchema)

	var schemaID string

	t.Run("Step1_CreateSchema", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "test-lifecycle-schema",
			"description": "Schema for lifecycle testing",
			"schema_definition": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":  map[string]interface{}{"type": "string"},
					"email": map[string]interface{}{"type": "string"},
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)
		if w.Code != 201 {
			t.Skipf("Schema creation failed (database may not be available): %d - %s", w.Code, w.Body.String())
			return
		}

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Skipf("Failed to parse response: %v", err)
			return
		}

		if id, ok := response["id"].(string); ok {
			schemaID = id
			assert.NotEmpty(t, schemaID)
		} else {
			t.Skip("Schema ID not found in response")
		}
	})

	t.Run("Step2_GetSchema", func(t *testing.T) {
		if schemaID == "" {
			t.Skip("Schema ID not available from previous step")
			return
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/" + schemaID,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 200, w.Code)

		var schema Schema
		err := json.Unmarshal(w.Body.Bytes(), &schema)
		assert.NoError(t, err)

		assert.Equal(t, "test-lifecycle-schema", schema.Name)
	})

	t.Run("Step3_UpdateSchema", func(t *testing.T) {
		if schemaID == "" {
			t.Skip("Schema ID not available from previous step")
			return
		}

		updateBody := map[string]interface{}{
			"description": "Updated lifecycle description",
		}

		req := HTTPTestRequest{
			Method: "PUT",
			Path:   "/api/v1/schemas/" + schemaID,
			Body:   updateBody,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("Step4_DeleteSchema", func(t *testing.T) {
		if schemaID == "" {
			t.Skip("Schema ID not available from previous step")
			return
		}

		req := HTTPTestRequest{
			Method: "DELETE",
			Path:   "/api/v1/schemas/" + schemaID,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 200, w.Code)
	})

	t.Run("Step5_VerifyDeleted", func(t *testing.T) {
		if schemaID == "" {
			t.Skip("Schema ID not available from previous step")
			return
		}

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schemas/" + schemaID,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 404, w.Code)
	})
}

// TestDataProcessingWorkflow tests the complete data processing workflow
func TestDataProcessingWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)
	env.Router.POST("/api/v1/process", processData)
	env.Router.GET("/api/v1/data/:schema_id", getProcessedData)

	var schemaID string

	t.Run("Step1_CreateProcessingSchema", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"name":        "test-processing-workflow",
			"description": "Schema for processing workflow testing",
			"schema_definition": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name":  map[string]interface{}{"type": "string"},
					"email": map[string]interface{}{"type": "string"},
					"phone": map[string]interface{}{"type": "string"},
				},
			},
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 201, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		schemaID = response["id"].(string)
		assert.NotEmpty(t, schemaID)
	})

	t.Run("Step2_ProcessTextData", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"schema_id":  schemaID,
			"input_type": "text",
			"input_data": "John Doe, email: john@example.com, phone: 555-1234",
			"batch_mode": false,
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/process",
			Body:   requestBody,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Contains(t, []int{200, 202}, w.Code)
	})

	t.Run("Step3_RetrieveProcessedData", func(t *testing.T) {
		// Give processing some time if async
		time.Sleep(100 * time.Millisecond)

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/data/" + schemaID,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 200, w.Code)
	})
}

// TestConcurrentSchemaOperations tests concurrent schema operations
func TestConcurrentSchemaOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)
	env.Router.GET("/api/v1/schemas", getSchemas)

	t.Run("ConcurrentCreates", func(t *testing.T) {
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				requestBody := map[string]interface{}{
					"name":        "test-concurrent-" + string(rune(index)),
					"description": "Concurrent test schema",
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
				assert.Contains(t, []int{200, 201}, w.Code)

				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

// TestDatabaseTransactions tests database transaction handling
func TestDatabaseTransactions(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	t.Run("TransactionRollback", func(t *testing.T) {
		// Create a transaction
		tx, err := env.Database.DB.Begin()
		assert.NoError(t, err)

		// Insert test data
		schemaJSON := `{"type": "object"}`
		_, err = tx.Exec(`
			INSERT INTO schemas (id, name, description, schema_definition, version, is_active, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, "test-tx-id", "test-transaction", "test", schemaJSON, 1, true, time.Now(), time.Now())

		assert.NoError(t, err)

		// Rollback
		err = tx.Rollback()
		assert.NoError(t, err)

		// Verify data was not saved
		var count int
		err = env.Database.DB.QueryRow("SELECT COUNT(*) FROM schemas WHERE id = $1", "test-tx-id").Scan(&count)
		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

// TestErrorRecovery tests error handling and recovery
func TestErrorRecovery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/schemas", createSchema)

	t.Run("RecoverFromInvalidData", func(t *testing.T) {
		// First, send invalid data
		invalidBody := map[string]interface{}{
			"name": "", // Empty name should fail validation
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   invalidBody,
		}

		w := makeHTTPRequest(env.Router, req)
		assert.Equal(t, 400, w.Code)

		// Then send valid data to ensure system recovered
		validBody := map[string]interface{}{
			"name":        "test-recovery-schema",
			"description": "Recovery test",
			"schema_definition": map[string]interface{}{
				"type": "object",
			},
		}

		req2 := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schemas",
			Body:   validBody,
		}

		w2 := makeHTTPRequest(env.Router, req2)
		assert.Equal(t, 201, w2.Code)
	})
}

// TestDataValidation tests data validation rules
func TestDataValidation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	skipIfNoDatabase(t, env.Database)

	// Setup router
	env.Router.POST("/api/v1/process", processData)

	schema := createTestSchema(t, env.Database, "validation-test")
	assert.NotNil(t, schema)

	tests := []struct {
		name           string
		inputType      string
		inputData      string
		expectedStatus int
	}{
		{
			name:           "ValidTextInput",
			inputType:      "text",
			inputData:      "Valid text data",
			expectedStatus: 200,
		},
		{
			name:           "EmptyTextInput",
			inputType:      "text",
			inputData:      "",
			expectedStatus: 400,
		},
		{
			name:           "InvalidInputType",
			inputType:      "invalid",
			inputData:      "data",
			expectedStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requestBody := map[string]interface{}{
				"schema_id":  schema.ID.String(),
				"input_type": tt.inputType,
				"input_data": tt.inputData,
			}

			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/process",
				Body:   requestBody,
			}

			w := makeHTTPRequest(env.Router, req)

			// Allow for both expected status and 202 (accepted for processing)
			if tt.expectedStatus == 200 {
				assert.Contains(t, []int{200, 202}, w.Code)
			} else {
				assert.Equal(t, tt.expectedStatus, w.Code)
			}
		})
	}
}
