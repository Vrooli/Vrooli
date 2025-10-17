// +build testing

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Test HTTP request helper functions
func TestMakeHTTPRequest(t *testing.T) {
	t.Run("SimpleGET", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/test",
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if w == nil {
			t.Fatal("Expected response recorder")
		}

		if httpReq.Method != "GET" {
			t.Errorf("Expected GET method, got %s", httpReq.Method)
		}

		if httpReq.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", httpReq.URL.Path)
		}
	})

	t.Run("POSTWithJSONBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/test",
			Body: map[string]string{
				"key": "value",
			},
		}

		w, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type application/json")
		}

		if w == nil {
			t.Fatal("Expected response recorder")
		}
	})

	t.Run("WithStringBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/test",
			Body:   `{"test": "data"}`,
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("Content-Type") != "application/json" {
			t.Error("Expected Content-Type application/json")
		}
	})

	t.Run("WithURLVars", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/test/{id}",
			URLVars: map[string]string{
				"id": "123",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq == nil {
			t.Fatal("Expected HTTP request")
		}
	})

	t.Run("WithQueryParams", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/test",
			QueryParams: map[string]string{
				"filter": "active",
				"limit":  "10",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		query := httpReq.URL.Query()
		if query.Get("filter") != "active" {
			t.Error("Expected filter query param")
		}
		if query.Get("limit") != "10" {
			t.Error("Expected limit query param")
		}
	})

	t.Run("WithCustomHeaders", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/test",
			Headers: map[string]string{
				"Authorization": "Bearer token123",
				"X-Custom":      "value",
			},
		}

		_, httpReq, err := makeHTTPRequest(req)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		if httpReq.Header.Get("Authorization") != "Bearer token123" {
			t.Error("Expected Authorization header")
		}
		if httpReq.Header.Get("X-Custom") != "value" {
			t.Error("Expected X-Custom header")
		}
	})
}

// Test JSON response assertions
func TestAssertJSONResponse(t *testing.T) {
	t.Run("ValidResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "test"}`))

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"success": true,
		})

		if response == nil {
			t.Fatal("Expected response")
		}

		if success, ok := response["success"].(bool); !ok || !success {
			t.Error("Expected success field to be true")
		}
	})

	t.Run("FieldValidation", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "123", "name": "test", "count": 5}`))

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"id":   "123",
			"name": "test",
		})

		if response == nil {
			t.Fatal("Expected response")
		}
	})
}

// Test error response assertions
func TestAssertErrorResponse(t *testing.T) {
	t.Run("JSONError", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid input"}`))

		assertErrorResponse(t, w, http.StatusBadRequest, "invalid")
	})

	t.Run("PlainTextError", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error occurred"))

		assertErrorResponse(t, w, http.StatusInternalServerError, "server error")
	})

	t.Run("EmptySubstring", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": "not found"}`))

		assertErrorResponse(t, w, http.StatusNotFound, "")
	})
}

// Test array assertions
func TestAssertJSONArray(t *testing.T) {
	t.Run("ValidArray", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items": [{"id": 1}, {"id": 2}]}`))

		array := assertJSONArray(t, w, http.StatusOK, "items")
		if len(array) != 2 {
			t.Errorf("Expected 2 items, got %d", len(array))
		}
	})

	t.Run("EmptyArray", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"items": []}`))

		array := assertJSONArray(t, w, http.StatusOK, "items")
		if len(array) != 0 {
			t.Errorf("Expected 0 items, got %d", len(array))
		}
	})

}

// Test data generators
func TestTestDataGenerator(t *testing.T) {
	gen := &TestDataGenerator{}

	t.Run("SchemaConnectRequest", func(t *testing.T) {
		req := gen.SchemaConnectRequest("test_db")
		if req["database_name"] != "test_db" {
			t.Error("Expected database_name to be test_db")
		}
	})

	t.Run("QueryGenerateRequest", func(t *testing.T) {
		req := gen.QueryGenerateRequest("show users", "main", true)
		if req.NaturalLanguage != "show users" {
			t.Error("Expected natural_language to be 'show users'")
		}
		if !req.IncludeExplanation {
			t.Error("Expected include_explanation to be true")
		}
	})

	t.Run("QueryExecuteRequest", func(t *testing.T) {
		req := gen.QueryExecuteRequest("SELECT 1", "test", 100)
		if req.SQL != "SELECT 1" {
			t.Error("Expected SQL to be 'SELECT 1'")
		}
		if req.Limit != 100 {
			t.Error("Expected limit to be 100")
		}
	})

	t.Run("LayoutSaveRequest", func(t *testing.T) {
		req := gen.LayoutSaveRequest("My Layout", "test_db", "graph", true)
		if req["name"] != "My Layout" {
			t.Error("Expected name to be 'My Layout'")
		}
		if req["is_shared"] != true {
			t.Error("Expected is_shared to be true")
		}
	})
}

// Test pattern builders
func TestTestScenarioBuilder(t *testing.T) {
	t.Run("BuildEmptyPatterns", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.Build()

		if len(patterns) != 0 {
			t.Errorf("Expected 0 patterns, got %d", len(patterns))
		}
	})

	t.Run("AddInvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddInvalidJSON("POST", "/api/test").Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "InvalidJSON" {
			t.Errorf("Expected pattern name InvalidJSON, got %s", patterns[0].Name)
		}
	})

	t.Run("AddMissingRequiredField", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddMissingRequiredField("POST", "/api/test", map[string]string{}).Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("AddEmptyBody", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddEmptyBody("POST", "/api/test").Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("AddInvalidQueryParameter", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.AddInvalidQueryParameter("/api/test", map[string]string{"invalid": "param"}).Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}
	})

	t.Run("ChainedPatterns", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		patterns := builder.
			AddInvalidJSON("POST", "/api/test").
			AddEmptyBody("POST", "/api/test").
			AddMissingRequiredField("POST", "/api/test", map[string]string{}).
			Build()

		if len(patterns) != 3 {
			t.Errorf("Expected 3 patterns, got %d", len(patterns))
		}
	})

	t.Run("AddCustomPattern", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusBadRequest,
		}

		patterns := builder.AddCustom(customPattern).Build()

		if len(patterns) != 1 {
			t.Errorf("Expected 1 pattern, got %d", len(patterns))
		}

		if patterns[0].Name != "CustomTest" {
			t.Errorf("Expected pattern name CustomTest, got %s", patterns[0].Name)
		}
	})
}

// Test predefined error patterns
func TestPredefinedErrorPatterns(t *testing.T) {
	t.Run("SchemaConnectErrorPatterns", func(t *testing.T) {
		patterns := SchemaConnectErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("QueryGenerateErrorPatterns", func(t *testing.T) {
		patterns := QueryGenerateErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("QueryExecuteErrorPatterns", func(t *testing.T) {
		patterns := QueryExecuteErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("SchemaExportErrorPatterns", func(t *testing.T) {
		patterns := SchemaExportErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("SchemaDiffErrorPatterns", func(t *testing.T) {
		patterns := SchemaDiffErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("LayoutSaveErrorPatterns", func(t *testing.T) {
		patterns := LayoutSaveErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("QueryOptimizeErrorPatterns", func(t *testing.T) {
		patterns := QueryOptimizeErrorPatterns()
		if len(patterns) < 2 {
			t.Errorf("Expected at least 2 patterns, got %d", len(patterns))
		}
	})
}

// Test edge case patterns
func TestEdgeCasePatterns(t *testing.T) {
	t.Run("LargeLimitEdgeCase", func(t *testing.T) {
		pattern := LargeLimitEdgeCase()
		if pattern.Name != "LargeLimit" {
			t.Errorf("Expected name LargeLimit, got %s", pattern.Name)
		}
	})

	t.Run("EmptyDatabaseNameEdgeCase", func(t *testing.T) {
		pattern := EmptyDatabaseNameEdgeCase()
		if pattern.Name != "EmptyDatabaseName" {
			t.Errorf("Expected name EmptyDatabaseName, got %s", pattern.Name)
		}
	})

	t.Run("SpecialCharactersEdgeCase", func(t *testing.T) {
		pattern := SpecialCharactersEdgeCase()
		if pattern.Name != "SpecialCharacters" {
			t.Errorf("Expected name SpecialCharacters, got %s", pattern.Name)
		}
	})
}
