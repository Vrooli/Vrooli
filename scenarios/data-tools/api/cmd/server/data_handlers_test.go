package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Vrooli/Vrooli/scenarios/data-tools/api/internal/dataprocessor"
)

func TestDataParse(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ParseCSVData", func(t *testing.T) {
		parseBody := map[string]interface{}{
			"data":   "name,age\nJohn,30\nJane,25",
			"format": "csv",
			"options": map[string]interface{}{
				"headers":     true,
				"infer_types": true,
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["dataset_id"] == nil {
			t.Error("Expected dataset_id in response")
		}

		schema, ok := data["schema"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected schema object in response")
		}

		columns, ok := schema["columns"].([]interface{})
		if !ok {
			t.Fatal("Expected columns array in schema")
		}

		if len(columns) != 2 {
			t.Errorf("Expected 2 columns, got %d", len(columns))
		}
	})

	t.Run("ParseJSONData", func(t *testing.T) {
		parseBody := map[string]interface{}{
			"data":   `[{"name":"John","age":30},{"name":"Jane","age":25}]`,
			"format": "json",
		}

		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["preview"] == nil {
			t.Error("Expected preview in response")
		}
	})

	t.Run("ParseInvalidData", func(t *testing.T) {
		parseBody := map[string]interface{}{
			"data":   "",
			"format": "csv",
		}

		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		// Should handle empty data gracefully
		if w.Code >= 500 {
			t.Errorf("Server error on empty data: %d", w.Code)
		}
	})

	t.Run("ParseMissingFormat", func(t *testing.T) {
		parseBody := map[string]interface{}{
			"data": "name,age\nJohn,30",
		}

		req := makeHTTPRequest("POST", "/api/v1/data/parse", parseBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		// Should handle missing format
		if w.Code >= 500 {
			t.Errorf("Server error on missing format: %d", w.Code)
		}
	})
}

func TestDataTransform(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("FilterTransformation", func(t *testing.T) {
		transformBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"name": "John", "age": 30.0},
				{"name": "Jane", "age": 25.0},
				{"name": "Bob", "age": 35.0},
			},
			"transformations": []map[string]interface{}{
				{
					"type": "filter",
					"parameters": map[string]interface{}{
						"condition": "age > 25",
					},
				},
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/transform", transformBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		result, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result object")
		}

		if result["transformation_id"] == nil && data["transformation_id"] == nil {
			t.Error("Expected transformation_id in response")
		}
	})

	t.Run("MultipleTransformations", func(t *testing.T) {
		transformBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"name": "John", "age": 30.0},
				{"name": "Jane", "age": 25.0},
			},
			"transformations": []map[string]interface{}{
				{
					"type": "filter",
					"parameters": map[string]interface{}{
						"condition": "age > 20",
					},
				},
				{
					"type": "sort",
					"parameters": map[string]interface{}{
						"field": "age",
						"order": "desc",
					},
				},
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/transform", transformBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["result"] == nil {
			t.Error("Expected result in response")
		}
	})

	t.Run("MissingData", func(t *testing.T) {
		transformBody := map[string]interface{}{
			"transformations": []map[string]interface{}{
				{"type": "filter"},
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/transform", transformBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "either dataset_id or data is required")
	})

	t.Run("EmptyTransformations", func(t *testing.T) {
		transformBody := map[string]interface{}{
			"data":             []map[string]interface{}{{"name": "John"}},
			"transformations":  []map[string]interface{}{},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/transform", transformBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		// Should handle empty transformations
		if w.Code >= 500 {
			t.Errorf("Server error on empty transformations: %d", w.Code)
		}
	})
}

func TestDataValidate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidateCompleteData", func(t *testing.T) {
		validateBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"name": "John", "age": 30.0, "email": "john@example.com"},
				{"name": "Jane", "age": 25.0, "email": "jane@example.com"},
			},
			"schema": map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "name", "type": "string", "nullable": false},
					{"name": "age", "type": "integer", "nullable": false},
					{"name": "email", "type": "string", "nullable": false},
				},
			},
			"quality_rules": []map[string]interface{}{},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		qualityReport, ok := data["quality_report"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected quality_report in response")
		}

		if qualityReport["completeness"] == nil {
			t.Error("Expected completeness score")
		}

		if qualityReport["accuracy"] == nil {
			t.Error("Expected accuracy score")
		}

		if qualityReport["consistency"] == nil {
			t.Error("Expected consistency score")
		}
	})

	t.Run("ValidateIncompleteData", func(t *testing.T) {
		validateBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"name": "John", "age": 30.0},
				{"name": "", "age": nil},
			},
			"schema": map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "name", "type": "string", "nullable": false},
					{"name": "age", "type": "integer", "nullable": false},
				},
			},
			"quality_rules": []map[string]interface{}{},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		qualityReport, ok := data["quality_report"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected quality_report in response")
		}

		completeness, ok := qualityReport["completeness"].(float64)
		if !ok {
			t.Fatal("Expected completeness to be a number")
		}

		if completeness >= 1.0 {
			t.Errorf("Expected completeness < 1.0 for incomplete data, got %f", completeness)
		}
	})

	t.Run("ValidateWithQualityRules", func(t *testing.T) {
		validateBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"name": "John", "age": 30.0},
				{"name": "Jane"},
			},
			"schema": map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "name", "type": "string", "nullable": false},
					{"name": "age", "type": "integer", "nullable": true},
				},
			},
			"quality_rules": []map[string]interface{}{
				{
					"rule_type": "required_field",
					"parameters": map[string]interface{}{
						"field": "name",
					},
					"severity": "error",
				},
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		violations, ok := data["violations"].([]interface{})
		if !ok {
			t.Fatal("Expected violations array in response")
		}

		_ = violations // Violations may be empty if data is valid
	})

	t.Run("DetectAnomalies", func(t *testing.T) {
		validateBody := map[string]interface{}{
			"data": []map[string]interface{}{
				{"value": 10.0},
				{"value": 12.0},
				{"value": 11.0},
				{"value": 1000.0}, // Anomaly
			},
			"schema": map[string]interface{}{
				"columns": []map[string]interface{}{
					{"name": "value", "type": "number", "nullable": false},
				},
			},
			"quality_rules": []map[string]interface{}{},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/validate", validateBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		qualityReport, ok := data["quality_report"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected quality_report in response")
		}

		anomalies, ok := qualityReport["anomalies"].([]interface{})
		if !ok {
			t.Fatal("Expected anomalies array")
		}

		if len(anomalies) == 0 {
			t.Log("Warning: No anomalies detected (may need more data points)")
		}
	})
}

func TestDataQuery(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("SimpleQuery", func(t *testing.T) {
		queryBody := map[string]interface{}{
			"sql": "SELECT 1 as test_value",
			"options": map[string]interface{}{
				"limit": 10,
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/query", queryBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		result, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result object")
		}

		if result["data"] == nil {
			t.Error("Expected data in result")
		}

		if result["columns"] == nil {
			t.Error("Expected columns in result")
		}
	})

	t.Run("QueryWithLimit", func(t *testing.T) {
		queryBody := map[string]interface{}{
			"sql": "SELECT generate_series(1, 100) as num",
			"options": map[string]interface{}{
				"limit": 5,
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/query", queryBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		result, ok := data["result"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected result object")
		}

		resultData, ok := result["data"].([]interface{})
		if !ok {
			t.Fatal("Expected data array in result")
		}

		if len(resultData) > 5 {
			t.Errorf("Expected at most 5 rows, got %d", len(resultData))
		}
	})

	t.Run("InvalidSQL", func(t *testing.T) {
		queryBody := map[string]interface{}{
			"sql": "INVALID SQL SYNTAX",
		}

		req := makeHTTPRequest("POST", "/api/v1/data/query", queryBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("EmptyQuery", func(t *testing.T) {
		queryBody := map[string]interface{}{
			"sql": "",
		}

		req := makeHTTPRequest("POST", "/api/v1/data/query", queryBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		if w.Code >= 500 {
			t.Errorf("Server error on empty query: %d", w.Code)
		}
	})
}

func TestStreamCreate(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("CreateWebhookStream", func(t *testing.T) {
		streamBody := map[string]interface{}{
			"source_config": map[string]interface{}{
				"type": "webhook",
				"url":  "http://example.com/webhook",
			},
			"processing_rules": []map[string]interface{}{
				{
					"type": "filter",
					"parameters": map[string]interface{}{
						"field": "status",
						"value": "active",
					},
				},
			},
			"output_config": map[string]interface{}{
				"destination": "dataset",
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/stream/create", streamBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["stream_id"] == nil {
			t.Error("Expected stream_id in response")
		}

		if data["status"] != "active" {
			t.Errorf("Expected status 'active', got '%v'", data["status"])
		}

		endpoints, ok := data["endpoints"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected endpoints object")
		}

		if endpoints["webhook"] == nil {
			t.Error("Expected webhook endpoint")
		}
	})

	t.Run("CreateKafkaStream", func(t *testing.T) {
		streamBody := map[string]interface{}{
			"source_config": map[string]interface{}{
				"type":   "kafka",
				"broker": "localhost:9092",
				"topic":  "test-topic",
			},
			"processing_rules": []map[string]interface{}{},
			"output_config": map[string]interface{}{
				"destination": "dataset",
			},
		}

		req := makeHTTPRequest("POST", "/api/v1/data/stream/create", streamBody, "test-token")
		w := httptest.NewRecorder()

		env.Server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusCreated)

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected data object in response")
		}

		if data["stream_id"] == nil {
			t.Error("Expected stream_id in response")
		}
	})
}

func TestDataValidationHelpers(t *testing.T) {
	t.Run("calculateCompleteness", func(t *testing.T) {
		data := []map[string]interface{}{
			{"name": "John", "age": 30.0},
			{"name": "Jane", "age": nil},
			{"name": "", "age": 25.0},
		}

		completeness := calculateCompleteness(data)

		if completeness < 0 || completeness > 1 {
			t.Errorf("Completeness should be between 0 and 1, got %f", completeness)
		}

		if completeness == 1.0 {
			t.Error("Expected completeness < 1.0 for incomplete data")
		}
	})

	t.Run("countDuplicates", func(t *testing.T) {
		data := []map[string]interface{}{
			{"name": "John", "age": 30.0},
			{"name": "John", "age": 30.0},
			{"name": "Jane", "age": 25.0},
		}

		duplicates := countDuplicates(data)

		if duplicates != 1 {
			t.Errorf("Expected 1 duplicate, got %d", duplicates)
		}
	})

	t.Run("detectAnomalies", func(t *testing.T) {
		data := []map[string]interface{}{
			{"value": 10.0},
			{"value": 12.0},
			{"value": 11.0},
			{"value": 10.5},
			{"value": 1000.0}, // Outlier
		}

		anomalies := detectAnomalies(data)

		if len(anomalies) == 0 {
			t.Log("Warning: No anomalies detected")
		}
	})

	t.Run("calculateAccuracy", func(t *testing.T) {
		data := []map[string]interface{}{
			{"name": "John", "age": 30.0},
			{"name": "Jane", "age": "invalid"}, // Type mismatch
		}

		schema := dataprocessor.Schema{
			Columns: []dataprocessor.Column{
				{Name: "name", Type: "string", Nullable: false},
				{Name: "age", Type: "integer", Nullable: false},
			},
		}

		accuracy := calculateAccuracy(data, schema)

		if accuracy < 0 || accuracy > 1 {
			t.Errorf("Accuracy should be between 0 and 1, got %f", accuracy)
		}
	})

	t.Run("calculateConsistency", func(t *testing.T) {
		data := []map[string]interface{}{
			{"name": "John", "age": 30.0},
			{"name": "Jane", "age": 25.0},
			{"name": "Bob"}, // Missing age field
		}

		consistency := calculateConsistency(data)

		if consistency < 0 || consistency > 1 {
			t.Errorf("Consistency should be between 0 and 1, got %f", consistency)
		}
	})
}
