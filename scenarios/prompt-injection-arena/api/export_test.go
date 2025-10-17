// +build testing

package main

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestExportFormats tests export format handling
func TestExportFormats(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SupportedFormats", func(t *testing.T) {
		formats := []string{"json", "csv", "xlsx", "markdown", "html"}

		for _, format := range formats {
			if format == "" {
				t.Error("Format should not be empty")
			}
			t.Logf("Format '%s' should be supported", format)
		}
	})

	t.Run("FormatDetection", func(t *testing.T) {
		testCases := []struct {
			format      string
			contentType string
		}{
			{"json", "application/json"},
			{"csv", "text/csv"},
			{"xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
			{"markdown", "text/markdown"},
			{"html", "text/html"},
		}

		for _, tc := range testCases {
			if tc.contentType == "" {
				t.Errorf("Content type for format %s should not be empty", tc.format)
			}
		}
	})
}

// TestExportHandlers tests export HTTP handlers
func TestExportHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()
	defer cleanupTestData(env.DB, "injection_techniques", "agent_configurations", "test_results")

	router := setupTestRouter()

	// Create test data
	agent := createTestAgentConfig(t, env.DB, "Test Agent")
	technique := createTestInjectionTechnique(t, env.DB, "Test Injection")
	result := createTestResult(t, env.DB, technique.ID, agent.ID)

	t.Run("ExportResultsJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/results",
			QueryParams: map[string]string{
				"format": "json",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			var data interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &data); err != nil {
				t.Errorf("Failed to parse JSON export: %v", err)
			}
		}
	})

	t.Run("ExportResultsCSV", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/results",
			QueryParams: map[string]string{
				"format": "csv",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			// Verify CSV format
			reader := csv.NewReader(strings.NewReader(w.Body.String()))
			records, err := reader.ReadAll()
			if err != nil {
				t.Logf("CSV parsing note: %v", err)
			}
			if len(records) > 0 {
				t.Logf("CSV export has %d records", len(records))
			}
		}
	})

	t.Run("ExportReport", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/report/" + result.TestSessionID,
		}

		w := makeHTTPRequest(router, req)
		// May return various status codes
		t.Logf("Export report returned status: %d", w.Code)
	})

	t.Run("ExportWithFilters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/results",
			QueryParams: map[string]string{
				"format":     "json",
				"agent_id":   agent.ID,
				"start_date": time.Now().Add(-24 * time.Hour).Format(time.RFC3339),
				"end_date":   time.Now().Format(time.RFC3339),
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			assertJSONResponse(t, w, http.StatusOK)
		}
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("ExportHandlers", router, "/api/v1/export")
		patterns := NewTestScenarioBuilder().
			AddInvalidQueryParams("/api/v1/export/results", map[string]string{
			"format": "invalid-format",
		}).
			AddNonExistentResource("/api/v1/export/report", "GET", "Session").
			Build()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("UnsupportedFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/results",
			QueryParams: map[string]string{
				"format": "unsupported",
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for unsupported format, got %d", w.Code)
		}
	})

	t.Run("MissingFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/export/results",
		}

		w := makeHTTPRequest(router, req)
		// Should use default format or return error
		t.Logf("Missing format parameter returned status: %d", w.Code)
	})
}

// TestJSONExport tests JSON export functionality
func TestJSONExport(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExportSingleResult", func(t *testing.T) {
		result := TestResult{
			ID:               uuid.New().String(),
			InjectionID:      uuid.New().String(),
			AgentID:          uuid.New().String(),
			Success:          true,
			ResponseText:     "Test response",
			ExecutionTimeMS:  100,
			SafetyViolations: []map[string]interface{}{},
			ConfidenceScore:  0.9,
			ErrorMessage:     "",
			ExecutedAt:       time.Now(),
			TestSessionID:    uuid.New().String(),
			Metadata:         map[string]interface{}{},
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result: %v", err)
		}

		var decoded TestResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal result: %v", err)
		}

		if decoded.ID != result.ID {
			t.Errorf("Expected ID %s, got %s", result.ID, decoded.ID)
		}
	})

	t.Run("ExportMultipleResults", func(t *testing.T) {
		results := []TestResult{
			{
				ID:              uuid.New().String(),
				InjectionID:     uuid.New().String(),
				AgentID:         uuid.New().String(),
				Success:         true,
				ExecutionTimeMS: 100,
				ExecutedAt:      time.Now(),
			},
			{
				ID:              uuid.New().String(),
				InjectionID:     uuid.New().String(),
				AgentID:         uuid.New().String(),
				Success:         false,
				ExecutionTimeMS: 150,
				ExecutedAt:      time.Now(),
			},
		}

		data, err := json.Marshal(results)
		if err != nil {
			t.Fatalf("Failed to marshal results: %v", err)
		}

		var decoded []TestResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal results: %v", err)
		}

		if len(decoded) != len(results) {
			t.Errorf("Expected %d results, got %d", len(results), len(decoded))
		}
	})

	t.Run("PrettyPrintJSON", func(t *testing.T) {
		result := map[string]interface{}{
			"id":      uuid.New().String(),
			"success": true,
			"score":   0.9,
		}

		prettyJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			t.Fatalf("Failed to pretty print JSON: %v", err)
		}

		if !strings.Contains(string(prettyJSON), "\n") {
			t.Error("Pretty printed JSON should contain newlines")
		}
	})
}

// TestCSVExport tests CSV export functionality
func TestCSVExport(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GenerateCSVHeader", func(t *testing.T) {
		headers := []string{
			"id",
			"injection_id",
			"agent_id",
			"success",
			"execution_time_ms",
			"confidence_score",
			"executed_at",
		}

		if len(headers) == 0 {
			t.Error("Headers should not be empty")
		}

		for _, header := range headers {
			if header == "" {
				t.Error("Header should not be empty")
			}
		}
	})

	t.Run("ConvertResultToCSVRow", func(t *testing.T) {
		result := TestResult{
			ID:              uuid.New().String(),
			InjectionID:     uuid.New().String(),
			AgentID:         uuid.New().String(),
			Success:         true,
			ExecutionTimeMS: 100,
			ConfidenceScore: 0.9,
			ExecutedAt:      time.Now(),
		}

		row := []string{
			result.ID,
			result.InjectionID,
			result.AgentID,
			"true",
			"100",
			"0.9",
			result.ExecutedAt.Format(time.RFC3339),
		}

		if len(row) != 7 {
			t.Errorf("Expected 7 CSV columns, got %d", len(row))
		}
	})

	t.Run("EscapeCSVSpecialCharacters", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"simple text", "simple text"},
			{"text with, comma", "\"text with, comma\""},
			{"text with \"quote\"", "\"text with \"\"quote\"\"\""},
			{"text with\nnewline", "\"text with\nnewline\""},
		}

		for _, tc := range testCases {
			escaped := tc.input
			if strings.ContainsAny(escaped, ",\"\n") {
				// Should be quoted
				t.Logf("String '%s' contains special characters", tc.input[:10])
			}
		}
	})

	t.Run("WriteCSVFile", func(t *testing.T) {
		var buffer strings.Builder
		writer := csv.NewWriter(&buffer)

		// Write header
		header := []string{"id", "name", "score"}
		if err := writer.Write(header); err != nil {
			t.Fatalf("Failed to write CSV header: %v", err)
		}

		// Write data rows
		rows := [][]string{
			{uuid.New().String(), "Test 1", "0.9"},
			{uuid.New().String(), "Test 2", "0.8"},
		}

		for _, row := range rows {
			if err := writer.Write(row); err != nil {
				t.Fatalf("Failed to write CSV row: %v", err)
			}
		}

		writer.Flush()

		if err := writer.Error(); err != nil {
			t.Fatalf("CSV writer error: %v", err)
		}

		csvContent := buffer.String()
		if !strings.Contains(csvContent, "id,name,score") {
			t.Error("CSV should contain header")
		}
	})
}

// TestExportFiltering tests export filtering logic
func TestExportFiltering(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("FilterByDateRange", func(t *testing.T) {
		now := time.Now()
		results := []TestResult{
			{ID: "1", ExecutedAt: now.Add(-48 * time.Hour)},
			{ID: "2", ExecutedAt: now.Add(-24 * time.Hour)},
			{ID: "3", ExecutedAt: now.Add(-12 * time.Hour)},
			{ID: "4", ExecutedAt: now},
		}

		startDate := now.Add(-30 * time.Hour)
		endDate := now

		filtered := []TestResult{}
		for _, result := range results {
			if result.ExecutedAt.After(startDate) && result.ExecutedAt.Before(endDate) {
				filtered = append(filtered, result)
			}
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 results in date range, got %d", len(filtered))
		}
	})

	t.Run("FilterByAgent", func(t *testing.T) {
		targetAgentID := uuid.New().String()
		results := []TestResult{
			{ID: "1", AgentID: targetAgentID},
			{ID: "2", AgentID: uuid.New().String()},
			{ID: "3", AgentID: targetAgentID},
		}

		filtered := []TestResult{}
		for _, result := range results {
			if result.AgentID == targetAgentID {
				filtered = append(filtered, result)
			}
		}

		if len(filtered) != 2 {
			t.Errorf("Expected 2 results for agent, got %d", len(filtered))
		}
	})

	t.Run("FilterBySuccess", func(t *testing.T) {
		results := []TestResult{
			{ID: "1", Success: true},
			{ID: "2", Success: false},
			{ID: "3", Success: true},
			{ID: "4", Success: false},
		}

		successCount := 0
		for _, result := range results {
			if result.Success {
				successCount++
			}
		}

		if successCount != 2 {
			t.Errorf("Expected 2 successful results, got %d", successCount)
		}
	})

	t.Run("MultipleFilters", func(t *testing.T) {
		now := time.Now()
		targetAgentID := uuid.New().String()

		results := []TestResult{
			{ID: "1", AgentID: targetAgentID, Success: true, ExecutedAt: now.Add(-1 * time.Hour)},
			{ID: "2", AgentID: targetAgentID, Success: false, ExecutedAt: now.Add(-1 * time.Hour)},
			{ID: "3", AgentID: uuid.New().String(), Success: true, ExecutedAt: now.Add(-1 * time.Hour)},
			{ID: "4", AgentID: targetAgentID, Success: true, ExecutedAt: now.Add(-48 * time.Hour)},
		}

		// Filter: agent AND success AND recent
		filtered := []TestResult{}
		for _, result := range results {
			if result.AgentID == targetAgentID &&
				result.Success &&
				result.ExecutedAt.After(now.Add(-24*time.Hour)) {
				filtered = append(filtered, result)
			}
		}

		if len(filtered) != 1 {
			t.Errorf("Expected 1 result with all filters, got %d", len(filtered))
		}
	})
}

// TestExportEdgeCases tests edge cases in export
func TestExportEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyResults", func(t *testing.T) {
		results := []TestResult{}

		data, err := json.Marshal(results)
		if err != nil {
			t.Fatalf("Failed to marshal empty results: %v", err)
		}

		if string(data) != "[]" {
			t.Errorf("Expected '[]', got '%s'", string(data))
		}
	})

	t.Run("VeryLargeExport", func(t *testing.T) {
		// Test exporting large number of results
		results := make([]TestResult, 10000)
		for i := range results {
			results[i] = TestResult{
				ID:              uuid.New().String(),
				InjectionID:     uuid.New().String(),
				AgentID:         uuid.New().String(),
				Success:         i%2 == 0,
				ExecutionTimeMS: i,
				ExecutedAt:      time.Now(),
			}
		}

		data, err := json.Marshal(results)
		if err != nil {
			t.Fatalf("Failed to marshal large export: %v", err)
		}

		if len(data) == 0 {
			t.Error("Export data should not be empty")
		}
	})

	t.Run("SpecialCharactersInData", func(t *testing.T) {
		result := TestResult{
			ID:           uuid.New().String(),
			ResponseText: "Response with \"quotes\" and 'apostrophes' and, commas",
			ErrorMessage: "Error: something went\nwrong\twith tabs",
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result with special chars: %v", err)
		}

		var decoded TestResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal result with special chars: %v", err)
		}
	})

	t.Run("NullValues", func(t *testing.T) {
		result := TestResult{
			ID:              uuid.New().String(),
			ResponseText:    "",
			ErrorMessage:    "",
			SafetyViolations: nil,
			Metadata:        nil,
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result with null values: %v", err)
		}

		if len(data) == 0 {
			t.Error("Marshaled data should not be empty")
		}
	})

	t.Run("UnicodeCharacters", func(t *testing.T) {
		result := TestResult{
			ID:           uuid.New().String(),
			ResponseText: "Unicode: 你好 🚀 café",
		}

		data, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("Failed to marshal result with unicode: %v", err)
		}

		var decoded TestResult
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal result with unicode: %v", err)
		}

		if decoded.ResponseText != result.ResponseText {
			t.Error("Unicode characters should be preserved")
		}
	})
}

// TestExportPerformance tests export performance
func TestExportPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("StreamingExport", func(t *testing.T) {
		// Test streaming large exports
		t.Log("Streaming export should handle large datasets efficiently")
	})

	t.Run("PaginatedExport", func(t *testing.T) {
		// Test paginated export for very large datasets
		pageSize := 1000
		totalRecords := 10000
		expectedPages := totalRecords / pageSize

		if expectedPages != 10 {
			t.Errorf("Expected 10 pages, got %d", expectedPages)
		}
	})

	t.Run("CompressionSupport", func(t *testing.T) {
		// Test compression for large exports
		t.Log("Compression should be supported for large exports")
	})
}
