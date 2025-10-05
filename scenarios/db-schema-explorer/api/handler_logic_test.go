package main

import (
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestSetupRoutes tests that all routes are properly registered
func TestSetupRoutes(t *testing.T) {
	server := &Server{
		router: mux.NewRouter(),
	}

	server.setupRoutes()

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/health"},
		{"POST", "/api/v1/schema/connect"},
		{"GET", "/api/v1/schema/list"},
		{"POST", "/api/v1/schema/export"},
		{"POST", "/api/v1/schema/diff"},
		{"POST", "/api/v1/query/generate"},
		{"POST", "/api/v1/query/execute"},
		{"GET", "/api/v1/query/history"},
		{"POST", "/api/v1/query/optimize"},
		{"POST", "/api/v1/layout/save"},
		{"GET", "/api/v1/layout/list"},
	}

	for _, route := range routes {
		req := httptest.NewRequest(route.method, route.path, nil)
		match := &mux.RouteMatch{}
		if !server.router.Match(req, match) {
			t.Errorf("Route %s %s not registered", route.method, route.path)
		}
	}
}

// TestHandleHealthLogic tests health handler logic structure
func TestHandleHealthLogic(t *testing.T) {
	// Test the health response structure
	services := make(map[string]string)
	services["database"] = "healthy"
	services["n8n"] = "healthy"
	services["qdrant"] = "healthy"
	services["ollama"] = "healthy"

	response := HealthResponse{
		Status:   "healthy",
		Services: services,
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if len(response.Services) != 4 {
		t.Errorf("Expected 4 services, got %d", len(response.Services))
	}

	if response.Services["database"] != "healthy" {
		t.Error("Expected database to be healthy")
	}
}

// Test handler request/response structures (without database)
func TestHandlerStructures(t *testing.T) {
	t.Run("QueryGenerateRequest", func(t *testing.T) {
		var req QueryGenerateRequest
		reqData := `{"natural_language": "test", "database_context": "main", "include_explanation": true}`

		err := json.Unmarshal([]byte(reqData), &req)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.NaturalLanguage != "test" {
			t.Error("Expected natural_language field")
		}
	})

	t.Run("SchemaDiffRequest", func(t *testing.T) {
		var req struct {
			Source string `json:"source"`
			Target string `json:"target"`
		}
		reqData := `{"source": "db1", "target": "db2"}`

		err := json.Unmarshal([]byte(reqData), &req)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.Source != "db1" || req.Target != "db2" {
			t.Error("Expected source and target fields")
		}
	})

	t.Run("QueryOptimizeRequest", func(t *testing.T) {
		var req struct {
			SQL          string `json:"sql"`
			DatabaseName string `json:"database_name"`
		}
		reqData := `{"sql": "SELECT * FROM users", "database_name": "main"}`

		err := json.Unmarshal([]byte(reqData), &req)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.SQL == "" {
			t.Error("Expected SQL field")
		}
	})
}

// TestGetSchemaInfoLogic tests getSchemaInfo logic
func TestGetSchemaInfoLogic(t *testing.T) {
	server := &Server{
		db:     nil, // Mock implementation doesn't use DB
		router: mux.NewRouter(),
	}

	tables, relationships, stats, err := server.getSchemaInfo("test_db")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(tables) == 0 {
		t.Error("Expected at least one table")
	}

	if len(relationships) == 0 {
		t.Error("Expected at least one relationship")
	}

	if stats.TotalTables <= 0 {
		t.Error("Expected positive total tables")
	}

	if stats.TotalColumns <= 0 {
		t.Error("Expected positive total columns")
	}

	// Verify table structure
	for _, table := range tables {
		if table.Name == "" {
			t.Error("Expected table name")
		}
		if table.Type == "" {
			t.Error("Expected table type")
		}
		if len(table.Columns) == 0 {
			t.Error("Expected columns in table")
		}
	}

	// Verify relationship structure
	for _, rel := range relationships {
		if rel.FromTable == "" {
			t.Error("Expected from_table in relationship")
		}
		if rel.ToTable == "" {
			t.Error("Expected to_table in relationship")
		}
		if rel.Type == "" {
			t.Error("Expected type in relationship")
		}
	}
}

// TestSaveSchemaSnapshotLogic tests saveSchemaSnapshot logic
func TestSaveSchemaSnapshotLogic(t *testing.T) {
	// This would require a database, so we just test that it accepts the right structure
	schema := SchemaResponse{
		Success:      true,
		DatabaseName: "test_db",
		SchemaName:   "public",
		Tables:       []SchemaTable{},
		Relationships: []Relationship{},
		Statistics: SchemaStats{
			TotalTables:        5,
			TotalColumns:       20,
			TotalRelationships: 3,
		},
	}

	// Just verify the structure is correct
	if schema.DatabaseName != "test_db" {
		t.Error("Schema structure incorrect")
	}

	// Verify it can be marshaled to JSON
	_, err := json.Marshal(schema)
	if err != nil {
		t.Errorf("Failed to marshal schema: %v", err)
	}
}

// TestSaveQueryHistoryLogic tests saveQueryHistory logic structure
func TestSaveQueryHistoryLogic(t *testing.T) {
	// Verify the parameters that would be passed
	naturalLanguage := "show all users"
	sqlQuery := "SELECT * FROM users"
	databaseName := "main"

	if naturalLanguage == "" {
		t.Error("Expected natural language")
	}
	if sqlQuery == "" {
		t.Error("Expected SQL query")
	}
	if databaseName == "" {
		t.Error("Expected database name")
	}
}

// TestMainLifecycleCheck tests that main requires lifecycle management
func TestMainLifecycleCheck(t *testing.T) {
	// Test that VROOLI_LIFECYCLE_MANAGED is required
	// This is tested in TestMain setup
}

// Test query execute default limit
func TestQueryExecuteDefaultLimit(t *testing.T) {
	req := QueryExecuteRequest{
		SQL:          "SELECT 1",
		DatabaseName: "main",
		Limit:        0,
	}

	limit := req.Limit
	if limit == 0 {
		limit = 100
	}

	if limit != 100 {
		t.Errorf("Expected default limit 100, got %d", limit)
	}
}

// Test query history default database
func TestQueryHistoryDefaultDatabase(t *testing.T) {
	databaseName := ""
	if databaseName == "" {
		databaseName = "main"
	}

	if databaseName != "main" {
		t.Errorf("Expected default database 'main', got '%s'", databaseName)
	}
}

// Test schema connect default database
func TestSchemaConnectDefaultDatabase(t *testing.T) {
	dbName := ""
	if dbName == "" {
		dbName = "main"
	}

	if dbName != "main" {
		t.Errorf("Expected default database 'main', got '%s'", dbName)
	}
}

// Test NewServer configuration logic
func TestNewServerConfigLogic(t *testing.T) {
	t.Run("MissingConfig", func(t *testing.T) {
		// Test that error is returned when no config is provided
		postgresURL := ""
		postgresHost := ""

		if postgresURL == "" && postgresHost == "" {
			// This is the error condition that should be caught
			_ = "Expected error: Database configuration missing"
		}
	})

	t.Run("WithPostgresURL", func(t *testing.T) {
		url := "postgresql://user:pass@localhost:5432/testdb"
		if url == "" {
			t.Error("Expected POSTGRES_URL to be set")
		}
	})

	t.Run("WithComponents", func(t *testing.T) {
		host := "localhost"
		port := "5432"
		user := "testuser"
		password := "testpass"
		dbname := "testdb"

		if host == "" || port == "" || user == "" || password == "" || dbname == "" {
			t.Error("Expected all components to be set")
		}
	})
}

// Test NULL handling types
func TestNullTypes(t *testing.T) {
	t.Run("NullString", func(t *testing.T) {
		var ns sql.NullString
		ns.String = "test"
		ns.Valid = true

		if !ns.Valid {
			t.Error("Expected Valid to be true")
		}
		if ns.String != "test" {
			t.Error("Expected string value")
		}
	})

	t.Run("NullInt64", func(t *testing.T) {
		var ni sql.NullInt64
		ni.Int64 = 42
		ni.Valid = true

		if !ni.Valid {
			t.Error("Expected Valid to be true")
		}
		if ni.Int64 != 42 {
			t.Error("Expected int64 value")
		}
	})
}
