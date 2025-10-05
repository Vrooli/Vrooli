package main

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestMain sets up the test environment
func TestMain(m *testing.M) {
	// Set required environment variables for testing
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "15999")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

// Test Health Endpoint
func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status": "healthy",
		})

		if response != nil {
			assertHealthyServices(t, response)
		}
	})
}

// Test Schema Connect Endpoint
func TestSchemaConnectEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schema/connect",
			Body:   TestData.SchemaConnectRequest("test_db"),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			// Verify response structure
			if dbName, ok := response["database_name"].(string); !ok || dbName == "" {
				t.Error("Expected database_name in response")
			}
			if schemaName, ok := response["schema_name"].(string); !ok || schemaName == "" {
				t.Error("Expected schema_name in response")
			}
			if tables, ok := response["tables"].([]interface{}); !ok || tables == nil {
				t.Error("Expected tables array in response")
			}
			if stats, ok := response["statistics"].(map[string]interface{}); !ok || stats == nil {
				t.Error("Expected statistics in response")
			}
		}
	})

	t.Run("DefaultDatabaseName", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schema/connect",
			Body: map[string]string{
				"connection_string": "",
				"database_name":     "",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if dbName, ok := response["database_name"].(string); !ok || dbName != "main" {
				t.Errorf("Expected default database_name 'main', got '%s'", dbName)
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleSchemaConnect",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, SchemaConnectErrorPatterns())
	})
}

// Test Schema List Endpoint
func TestSchemaListEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test data
		createTestSchemaSnapshot(t, testDB.DB, "test_db")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schema/list",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			schemas := assertJSONArray(t, w, http.StatusOK, "schemas")
			if len(schemas) == 0 {
				t.Error("Expected at least one schema in response")
			}
		}
	})

	t.Run("EmptyList", func(t *testing.T) {
		// Clean test data
		testDB.DB.Exec("TRUNCATE TABLE db_explorer.schema_snapshots")

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/schema/list",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			schemas, _ := response["schemas"].([]interface{})
			if schemas != nil && len(schemas) > 0 {
				t.Error("Expected empty schemas list")
			}
		}
	})
}

// Test Schema Export Endpoint
func TestSchemaExportEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("JSONExport", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schema/export",
			Body: map[string]string{
				"database_name": "test_db",
				"format":        "json",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if database, ok := response["database"].(string); !ok || database != "test_db" {
				t.Error("Expected database field in export")
			}
			if format, ok := response["format"].(string); !ok || format != "json" {
				t.Error("Expected format field in export")
			}
			if _, ok := response["tables"]; !ok {
				t.Error("Expected tables in export")
			}
			if _, ok := response["relationships"]; !ok {
				t.Error("Expected relationships in export")
			}
			if _, ok := response["statistics"]; !ok {
				t.Error("Expected statistics in export")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleSchemaExport",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, SchemaExportErrorPatterns())
	})
}

// Test Schema Diff Endpoint
func TestSchemaDiffEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schema/diff",
			Body: map[string]string{
				"source": "db1",
				"target": "db2",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if source, ok := response["source"].(string); !ok || source != "db1" {
				t.Error("Expected source database in diff")
			}
			if target, ok := response["target"].(string); !ok || target != "db2" {
				t.Error("Expected target database in diff")
			}
			if _, ok := response["differences"]; !ok {
				t.Error("Expected differences in diff response")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleSchemaDiff",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, SchemaDiffErrorPatterns())
	})
}

// Test Query Generate Endpoint
func TestQueryGenerateEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/generate",
			Body:   TestData.QueryGenerateRequest("show all users", "main", true),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if sql, ok := response["sql"].(string); !ok || sql == "" {
				t.Error("Expected sql field in response")
			}
			if queryType, ok := response["query_type"].(string); !ok || queryType == "" {
				t.Error("Expected query_type field in response")
			}
			if confidence, ok := response["confidence"].(float64); !ok || confidence <= 0 {
				t.Error("Expected confidence field in response")
			}
			if tablesUsed, ok := response["tables_used"].([]interface{}); !ok || len(tablesUsed) == 0 {
				t.Error("Expected tables_used array in response")
			}
		}
	})

	t.Run("WithoutExplanation", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/generate",
			Body:   TestData.QueryGenerateRequest("count all records", "main", false),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if _, ok := response["explanation"]; ok {
				// Explanation may or may not be present when include_explanation is false
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleQueryGenerate",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, QueryGenerateErrorPatterns())
	})
}

// Test Query Execute Endpoint
func TestQueryExecuteEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("SimpleQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/execute",
			Body:   TestData.QueryExecuteRequest("SELECT 1 as num", "main", 10),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if columns, ok := response["columns"].([]interface{}); !ok || len(columns) == 0 {
				t.Error("Expected columns in response")
			}
			if rows, ok := response["rows"].([]interface{}); !ok || len(rows) == 0 {
				t.Error("Expected rows in response")
			}
			if rowCount, ok := response["row_count"].(float64); !ok || rowCount != 1 {
				t.Error("Expected row_count to be 1")
			}
			if execTime, ok := response["execution_time_ms"].(float64); !ok || execTime < 0 {
				t.Error("Expected execution_time_ms in response")
			}
		}
	})

	t.Run("DefaultLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/execute",
			Body: map[string]interface{}{
				"sql":           "SELECT 1",
				"database_name": "main",
				// limit not specified
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Fatal("Expected response")
		}
	})

	t.Run("InvalidSQL", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/execute",
			Body:   TestData.QueryExecuteRequest("INVALID SQL", "main", 10),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, nil)

		if response != nil {
			if success, ok := response["success"].(bool); !ok || success {
				t.Error("Expected success to be false for invalid SQL")
			}
			if _, ok := response["error"].(string); !ok {
				t.Error("Expected error message for invalid SQL")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleQueryExecute",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, QueryExecuteErrorPatterns())
	})
}

// Test Query History Endpoint
func TestQueryHistoryEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test data
		createTestQueryHistory(t, testDB.DB, "test_db")

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/query/history",
			QueryParams: map[string]string{"database": "test_db"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			history := assertJSONArray(t, w, http.StatusOK, "history")
			if len(history) == 0 {
				t.Error("Expected at least one history entry")
			}
		}
	})

	t.Run("DefaultDatabase", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/query/history",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Fatal("Expected response")
		}
	})
}

// Test Query Optimize Endpoint
func TestQueryOptimizeEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/optimize",
			Body: map[string]string{
				"sql":           "SELECT * FROM users WHERE created_at > NOW() - INTERVAL '7 days'",
				"database_name": "main",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if originalSQL, ok := response["original_sql"].(string); !ok || originalSQL == "" {
				t.Error("Expected original_sql in response")
			}
			if optimizedSQL, ok := response["optimized_sql"].(string); !ok || optimizedSQL == "" {
				t.Error("Expected optimized_sql in response")
			}
			if optimizations, ok := response["optimizations"].([]interface{}); !ok || len(optimizations) == 0 {
				t.Error("Expected optimizations array in response")
			}
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleQueryOptimize",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, QueryOptimizeErrorPatterns())
	})
}

// Test Layout Save Endpoint
func TestLayoutSaveEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/layout/save",
			Body:   TestData.LayoutSaveRequest("My Layout", "test_db", "graph", false),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			if id, ok := response["id"].(string); !ok || id == "" {
				t.Error("Expected id in response")
			} else {
				// Verify it's a valid UUID
				if _, err := uuid.Parse(id); err != nil {
					t.Errorf("Expected valid UUID for id, got %s", id)
				}
			}
			if message, ok := response["message"].(string); !ok || message == "" {
				t.Error("Expected message in response")
			}
		}
	})

	t.Run("SharedLayout", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/layout/save",
			Body:   TestData.LayoutSaveRequest("Shared Layout", "test_db", "tree", true),
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Fatal("Expected response")
		}
	})

	t.Run("ErrorCases", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "handleLayoutSave",
			Router:      server.Router,
		}
		suite.RunErrorTests(t, LayoutSaveErrorPatterns())
	})
}

// Test Layout List Endpoint
func TestLayoutListEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test data
		createTestLayout(t, testDB.DB, "test_db")

		req := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/layout/list",
			QueryParams: map[string]string{"database": "test_db"},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response != nil {
			layouts := assertJSONArray(t, w, http.StatusOK, "layouts")
			if len(layouts) == 0 {
				t.Error("Expected at least one layout")
			}
		}
	})

	t.Run("AllLayouts", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/layout/list",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		server.Router.ServeHTTP(w, httpReq)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Fatal("Expected response")
		}
	})
}

// Test Helper Functions
func TestGetSchemaInfo(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("Success", func(t *testing.T) {
		tables, relationships, stats, err := server.Server.getSchemaInfo("test_db")

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(tables) == 0 {
			t.Error("Expected at least one table")
		}

		if len(relationships) == 0 {
			t.Error("Expected at least one relationship")
		}

		if stats.TotalTables <= 0 {
			t.Error("Expected positive total_tables")
		}

		if stats.TotalColumns <= 0 {
			t.Error("Expected positive total_columns")
		}
	})
}

// Test Server Creation
func TestNewServer(t *testing.T) {
	t.Run("WithoutDatabaseConfig", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_HOST")
		os.Unsetenv("POSTGRES_PORT")
		os.Unsetenv("POSTGRES_USER")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("POSTGRES_DB")

		_, err := NewServer()

		if err == nil {
			t.Error("Expected error when database config is missing")
		}
	})
}

// Performance Tests
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("HealthCheckPerformance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			req := HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			}

			w, httpReq, _ := makeHTTPRequest(req)
			server.Router.ServeHTTP(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 100

		if avgDuration > 50*time.Millisecond {
			t.Errorf("Health check too slow: avg %v per request", avgDuration)
		}
	})

	t.Run("QueryGenerationPerformance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 50; i++ {
			req := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/query/generate",
				Body:   TestData.QueryGenerateRequest("show users", "main", false),
			}

			w, httpReq, _ := makeHTTPRequest(req)
			server.Router.ServeHTTP(w, httpReq)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}

		duration := time.Since(start)
		avgDuration := duration / 50

		if avgDuration > 100*time.Millisecond {
			t.Errorf("Query generation too slow: avg %v per request", avgDuration)
		}
	})
}

// Edge Case Tests
func TestEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	edgeCases := []EdgeCaseTestPattern{
		LargeLimitEdgeCase(),
		EmptyDatabaseNameEdgeCase(),
		SpecialCharactersEdgeCase(),
	}

	RunEdgeCaseTests(t, server, edgeCases)
}

// Integration Tests
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	testDB := setupTestDatabase(t)
	if testDB == nil {
		return
	}
	defer testDB.Cleanup()

	server := setupTestServer(t, testDB.DB)

	t.Run("CompleteWorkflow", func(t *testing.T) {
		// 1. Connect to schema
		connectReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/schema/connect",
			Body:   TestData.SchemaConnectRequest("workflow_test"),
		}

		w, httpReq, _ := makeHTTPRequest(connectReq)
		server.Router.ServeHTTP(w, httpReq)
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"success": true})

		// 2. Generate query
		genReq := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/query/generate",
			Body:   TestData.QueryGenerateRequest("show all data", "workflow_test", true),
		}

		w, httpReq, _ = makeHTTPRequest(genReq)
		server.Router.ServeHTTP(w, httpReq)
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"success": true})

		// 3. Execute query
		if response != nil {
			sql, _ := response["sql"].(string)
			execReq := HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/query/execute",
				Body:   TestData.QueryExecuteRequest(sql, "workflow_test", 10),
			}

			w, httpReq, _ = makeHTTPRequest(execReq)
			server.Router.ServeHTTP(w, httpReq)
			assertJSONResponse(t, w, http.StatusOK, nil)
		}

		// 4. Check history
		histReq := HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/query/history",
			QueryParams: map[string]string{"database": "workflow_test"},
		}

		w, httpReq, _ = makeHTTPRequest(histReq)
		server.Router.ServeHTTP(w, httpReq)
		assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"success": true})
	})
}
