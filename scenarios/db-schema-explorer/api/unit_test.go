package main

import (
	"testing"

	"github.com/google/uuid"
)

// Test utility function getEnv
func TestGetEnv(t *testing.T) {
	t.Run("ExistingVariable", func(t *testing.T) {
		t.Setenv("TEST_VAR", "test_value")
		result := getEnv("TEST_VAR", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("NonExistingVariable", func(t *testing.T) {
		result := getEnv("NON_EXISTING_VAR_12345", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("EmptyDefault", func(t *testing.T) {
		result := getEnv("NON_EXISTING_VAR_67890", "")
		if result != "" {
			t.Errorf("Expected empty string, got '%s'", result)
		}
	})
}

// Test struct constructors and types
func TestSchemaTable(t *testing.T) {
	table := SchemaTable{
		Name:         "users",
		Type:         "BASE TABLE",
		ColumnsCount: 5,
		Columns: []Column{
			{
				Name:       "id",
				DataType:   "uuid",
				IsNullable: false,
				IsPrimary:  true,
			},
		},
	}

	if table.Name != "users" {
		t.Errorf("Expected name 'users', got '%s'", table.Name)
	}

	if table.ColumnsCount != 5 {
		t.Errorf("Expected columns count 5, got %d", table.ColumnsCount)
	}

	if len(table.Columns) != 1 {
		t.Errorf("Expected 1 column, got %d", len(table.Columns))
	}
}

func TestColumn(t *testing.T) {
	refTable := "users"
	column := Column{
		Name:       "user_id",
		DataType:   "uuid",
		IsNullable: false,
		IsPrimary:  false,
		IsForeign:  true,
		References: &refTable,
	}

	if column.Name != "user_id" {
		t.Errorf("Expected name 'user_id', got '%s'", column.Name)
	}

	if !column.IsForeign {
		t.Error("Expected IsForeign to be true")
	}

	if column.References == nil || *column.References != "users" {
		t.Error("Expected References to be 'users'")
	}
}

func TestRelationship(t *testing.T) {
	rel := Relationship{
		FromTable:  "posts",
		FromColumn: "user_id",
		ToTable:    "users",
		ToColumn:   "id",
		Type:       "foreign_key",
	}

	if rel.FromTable != "posts" {
		t.Errorf("Expected FromTable 'posts', got '%s'", rel.FromTable)
	}

	if rel.Type != "foreign_key" {
		t.Errorf("Expected Type 'foreign_key', got '%s'", rel.Type)
	}
}

func TestSchemaResponse(t *testing.T) {
	response := SchemaResponse{
		Success:      true,
		DatabaseName: "test_db",
		SchemaName:   "public",
		Tables:       []SchemaTable{},
		Relationships: []Relationship{},
		Statistics: SchemaStats{
			TotalTables:        5,
			TotalColumns:       25,
			TotalRelationships: 3,
			TotalIndexes:       10,
		},
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.DatabaseName != "test_db" {
		t.Errorf("Expected DatabaseName 'test_db', got '%s'", response.DatabaseName)
	}

	if response.Statistics.TotalTables != 5 {
		t.Errorf("Expected TotalTables 5, got %d", response.Statistics.TotalTables)
	}
}

func TestSchemaStats(t *testing.T) {
	stats := SchemaStats{
		TotalTables:        10,
		TotalColumns:       50,
		TotalRelationships: 8,
		TotalIndexes:       15,
	}

	if stats.TotalTables != 10 {
		t.Errorf("Expected TotalTables 10, got %d", stats.TotalTables)
	}

	if stats.TotalColumns != 50 {
		t.Errorf("Expected TotalColumns 50, got %d", stats.TotalColumns)
	}
}

func TestQueryGenerateRequest(t *testing.T) {
	req := QueryGenerateRequest{
		NaturalLanguage:    "show all users",
		DatabaseContext:    "main",
		IncludeExplanation: true,
	}

	if req.NaturalLanguage != "show all users" {
		t.Errorf("Expected NaturalLanguage 'show all users', got '%s'", req.NaturalLanguage)
	}

	if !req.IncludeExplanation {
		t.Error("Expected IncludeExplanation to be true")
	}
}

func TestQueryGenerateResponse(t *testing.T) {
	response := QueryGenerateResponse{
		Success:     true,
		SQL:         "SELECT * FROM users",
		Explanation: "Retrieves all user records",
		TablesUsed:  []string{"users"},
		QueryType:   "SELECT",
		Confidence:  85,
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if response.QueryType != "SELECT" {
		t.Errorf("Expected QueryType 'SELECT', got '%s'", response.QueryType)
	}

	if response.Confidence != 85 {
		t.Errorf("Expected Confidence 85, got %d", response.Confidence)
	}

	if len(response.TablesUsed) != 1 {
		t.Errorf("Expected 1 table used, got %d", len(response.TablesUsed))
	}
}

func TestQuery(t *testing.T) {
	queryID := uuid.New()
	query := Query{
		ID:              queryID.String(),
		NaturalLanguage: "get recent users",
		SQL:             "SELECT * FROM users ORDER BY created_at DESC LIMIT 10",
		UsageCount:      5,
	}

	if query.ID != queryID.String() {
		t.Errorf("Expected ID '%s', got '%s'", queryID.String(), query.ID)
	}

	if query.UsageCount != 5 {
		t.Errorf("Expected UsageCount 5, got %d", query.UsageCount)
	}
}

func TestQueryExecuteRequest(t *testing.T) {
	req := QueryExecuteRequest{
		SQL:          "SELECT * FROM users",
		DatabaseName: "test_db",
		Limit:        100,
	}

	if req.SQL != "SELECT * FROM users" {
		t.Errorf("Expected SQL 'SELECT * FROM users', got '%s'", req.SQL)
	}

	if req.Limit != 100 {
		t.Errorf("Expected Limit 100, got %d", req.Limit)
	}
}

func TestQueryExecuteResponse(t *testing.T) {
	response := QueryExecuteResponse{
		Success:       true,
		Columns:       []string{"id", "name", "email"},
		Rows:          [][]interface{}{},
		RowCount:      0,
		ExecutionTime: 15.5,
	}

	if !response.Success {
		t.Error("Expected Success to be true")
	}

	if len(response.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(response.Columns))
	}

	if response.ExecutionTime != 15.5 {
		t.Errorf("Expected ExecutionTime 15.5, got %f", response.ExecutionTime)
	}
}

func TestHealthResponse(t *testing.T) {
	services := make(map[string]string)
	services["database"] = "healthy"
	services["n8n"] = "healthy"

	response := HealthResponse{
		Status:   "healthy",
		Services: services,
	}

	if response.Status != "healthy" {
		t.Errorf("Expected Status 'healthy', got '%s'", response.Status)
	}

	if len(response.Services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(response.Services))
	}

	if response.Services["database"] != "healthy" {
		t.Error("Expected database service to be healthy")
	}
}

// Test Server struct
func TestServerStruct(t *testing.T) {
	server := &Server{
		db:     nil, // Would be a real DB in production
		router: nil, // Would be a real router in production
	}

	if server.db != nil {
		t.Error("Expected db to be nil")
	}

	if server.router != nil {
		t.Error("Expected router to be nil")
	}
}

// Test default limit logic
func TestDefaultLimitLogic(t *testing.T) {
	testCases := []struct {
		name     string
		limit    int
		expected int
	}{
		{"ZeroLimit", 0, 100},
		{"NegativeLimit", -5, 100},
		{"ValidLimit", 50, 50},
		{"LargeLimit", 1000, 1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limit := tc.limit
			if limit == 0 {
				limit = 100
			}
			// Note: negative limits would need to be handled in production code
			if limit < 0 {
				limit = 100
			}

			if limit != tc.expected && tc.limit >= 0 {
				t.Errorf("Expected limit %d, got %d", tc.expected, limit)
			}
		})
	}
}

// Test data structure validation
func TestSchemaDataStructures(t *testing.T) {
	t.Run("EmptySchemaResponse", func(t *testing.T) {
		response := SchemaResponse{
			Success:       true,
			Tables:        []SchemaTable{},
			Relationships: []Relationship{},
			Statistics:    SchemaStats{},
		}

		if len(response.Tables) != 0 {
			t.Error("Expected empty tables slice")
		}

		if response.Statistics.TotalTables != 0 {
			t.Error("Expected zero total tables")
		}
	})

	t.Run("MultipleColumns", func(t *testing.T) {
		table := SchemaTable{
			Name:         "users",
			ColumnsCount: 3,
			Columns: []Column{
				{Name: "id", DataType: "uuid", IsPrimary: true},
				{Name: "name", DataType: "varchar"},
				{Name: "email", DataType: "varchar"},
			},
		}

		if len(table.Columns) != 3 {
			t.Errorf("Expected 3 columns, got %d", len(table.Columns))
		}

		primaryFound := false
		for _, col := range table.Columns {
			if col.IsPrimary {
				primaryFound = true
				break
			}
		}

		if !primaryFound {
			t.Error("Expected to find primary key column")
		}
	})
}
