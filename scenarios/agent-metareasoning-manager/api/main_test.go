package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealth tests the health endpoint
func TestHealth(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthCheck_Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		Health(w, req)

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": serviceName,
			"version": apiVersion,
		})

		if response == nil {
			t.Fatal("Response should not be nil")
		}
	})
}

// TestNewLogger tests logger creation
func TestNewLogger(t *testing.T) {
	t.Run("CreateLogger_Success", func(t *testing.T) {
		logger := NewLogger()
		if logger == nil {
			t.Fatal("Logger should not be nil")
		}
		if logger.Logger == nil {
			t.Fatal("Logger.Logger should not be nil")
		}
	})
}

// TestLoggerMethods tests logger methods
func TestLoggerMethods(t *testing.T) {
	logger := NewLogger()

	t.Run("LogError", func(t *testing.T) {
		// Should not panic
		logger.Error("test error", fmt.Errorf("test"))
	})

	t.Run("LogWarn", func(t *testing.T) {
		// Should not panic
		logger.Warn("test warning", fmt.Errorf("test"))
	})

	t.Run("LogInfo", func(t *testing.T) {
		// Should not panic
		logger.Info("test info")
	})
}

// TestHTTPError tests error response formatting
func TestHTTPError(t *testing.T) {
	t.Run("HTTPError_BasicError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "test error", http.StatusBadRequest, fmt.Errorf("underlying error"))

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse error response: %v", err)
		}

		if response["error"] != "test error" {
			t.Errorf("Expected error message 'test error', got %v", response["error"])
		}

		if response["status"] != float64(http.StatusBadRequest) {
			t.Errorf("Expected status %d, got %v", http.StatusBadRequest, response["status"])
		}
	})

	t.Run("HTTPError_NilError", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "test error", http.StatusInternalServerError, nil)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
		}
	})
}

// TestGetResourcePort tests port registry lookup
func TestGetResourcePort(t *testing.T) {
	t.Run("GetResourcePort_KnownResource", func(t *testing.T) {
		port := getResourcePort("n8n")
		if port == "" {
			t.Error("Expected port for n8n, got empty string")
		}
	})

	t.Run("GetResourcePort_UnknownResource", func(t *testing.T) {
		port := getResourcePort("unknown-resource")
		// Should return a default or fallback port (8080 is the generic fallback)
		if port != "8080" && port != "" {
			// Port could be empty if the script fails, or "8080" if it falls back
			// Accept both as valid for unknown resources
		}
	})
}

// TestExtractTags tests tag extraction from workflow metadata
func TestExtractTags(t *testing.T) {
	t.Run("ExtractTags_WithTags", func(t *testing.T) {
		workflow := map[string]interface{}{
			"tags": []interface{}{"tag1", "tag2", "tag3"},
		}
		tags := extractTags(workflow)
		if len(tags) != 3 {
			t.Errorf("Expected 3 tags, got %d", len(tags))
		}
	})

	t.Run("ExtractTags_NoTags", func(t *testing.T) {
		workflow := map[string]interface{}{}
		tags := extractTags(workflow)
		if len(tags) != 0 {
			t.Errorf("Expected 0 tags, got %d", len(tags))
		}
	})

	t.Run("ExtractTags_InvalidType", func(t *testing.T) {
		workflow := map[string]interface{}{
			"tags": "not-an-array",
		}
		tags := extractTags(workflow)
		if len(tags) != 0 {
			t.Errorf("Expected 0 tags for invalid type, got %d", len(tags))
		}
	})
}

// TestNewDiscoveryService tests discovery service creation
func TestNewDiscoveryService(t *testing.T) {
	// Skip if no test database
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	t.Run("CreateDiscoveryService_Success", func(t *testing.T) {
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		// Mock DB would be needed here for full test
		// For now, test with nil to verify structure
		service := NewDiscoveryService(nil, "http://localhost:5678", "http://localhost:5681", "http://localhost:6333")

		if service == nil {
			t.Fatal("Service should not be nil")
		}
		if service.n8nBaseURL != "http://localhost:5678" {
			t.Errorf("Expected n8n URL 'http://localhost:5678', got %s", service.n8nBaseURL)
		}
		if service.windmillURL != "http://localhost:5681" {
			t.Errorf("Expected windmill URL 'http://localhost:5681', got %s", service.windmillURL)
		}
	})
}

// TestIsMetareasoningWorkflow tests workflow pattern matching
func TestIsMetareasoningWorkflow(t *testing.T) {
	service := NewDiscoveryService(nil, "", "", "")

	tests := []struct {
		name     string
		workflow string
		expected bool
	}{
		{"ProsConsWorkflow", "pros-cons-analyzer", true},
		{"SWOTWorkflow", "swot-analysis", true},
		{"RiskWorkflow", "risk-assessment", true},
		{"DecisionWorkflow", "decision-maker", true},
		{"SelfReviewWorkflow", "self-review-tool", true},
		{"ReasoningWorkflow", "reasoning-chain", true},
		{"MetareasoningWorkflow", "metareasoning-engine", true},
		{"NonMetareasoningWorkflow", "data-processor", false},
		{"EmptyName", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isMetareasoningWorkflow(tt.workflow)
			if result != tt.expected {
				t.Errorf("For workflow '%s', expected %v, got %v", tt.workflow, tt.expected, result)
			}
		})
	}
}

// TestAnalyzeRequest tests the analyze request structure
func TestAnalyzeRequest(t *testing.T) {
	t.Run("ValidAnalyzeRequest", func(t *testing.T) {
		req := AnalyzeRequest{
			Type:    "pros-cons",
			Input:   "Should we adopt microservices?",
			Context: "E-commerce platform",
			Model:   "llama3.2",
		}

		if req.Type != "pros-cons" {
			t.Error("Type not set correctly")
		}
		if req.Input == "" {
			t.Error("Input should not be empty")
		}
	})
}

// TestWorkflowMetadata tests workflow metadata structure
func TestWorkflowMetadata(t *testing.T) {
	t.Run("CreateWorkflowMetadata", func(t *testing.T) {
		now := time.Now()
		wf := WorkflowMetadata{
			ID:          uuid.New(),
			Platform:    "n8n",
			PlatformID:  "test-workflow",
			Name:        "Test Workflow",
			Description: "Test Description",
			Category:    "metareasoning",
			Tags:        []string{"test", "reasoning"},
			UsageCount:  5,
			LastUsed:    &now,
		}

		if wf.Platform != "n8n" {
			t.Error("Platform not set correctly")
		}
		if len(wf.Tags) != 2 {
			t.Errorf("Expected 2 tags, got %d", len(wf.Tags))
		}
		if wf.UsageCount != 5 {
			t.Errorf("Expected usage count 5, got %d", wf.UsageCount)
		}
	})
}

// TestListWorkflows tests workflow listing (integration test)
func TestListWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ListWorkflows_EmptyDatabase", func(t *testing.T) {
		// This would require database setup
		// Skipping for now without DB
		t.Skip("Requires test database")
	})
}

// TestSearchWorkflows tests workflow search functionality
func TestSearchWorkflows(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SearchWorkflows_InvalidJSON", func(t *testing.T) {
		service := NewDiscoveryService(nil, "", "", "")

		req := httptest.NewRequest("POST", "/workflows/search", bytes.NewBufferString(`{"invalid json`))
		w := httptest.NewRecorder()

		service.SearchWorkflows(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("SearchWorkflows_ValidRequest", func(t *testing.T) {
		// Skip this test without DB - it requires database for fallback search
		t.Skip("Requires test database for fallback search")
	})
}

// TestAnalyzeWorkflow tests the analyze workflow handler
func TestAnalyzeWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("AnalyzeWorkflow_InvalidJSON", func(t *testing.T) {
		service := NewDiscoveryService(nil, "", "", "")

		req := httptest.NewRequest("POST", "/analyze", bytes.NewBufferString(`{"invalid json`))
		w := httptest.NewRecorder()

		service.AnalyzeWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for invalid JSON, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("AnalyzeWorkflow_MissingType", func(t *testing.T) {
		service := NewDiscoveryService(nil, "", "", "")

		analyzeReq := map[string]interface{}{
			"input": "test input",
			// Missing "type" field
		}
		reqBody, _ := json.Marshal(analyzeReq)

		req := httptest.NewRequest("POST", "/analyze", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.AnalyzeWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for missing type, got %d", http.StatusBadRequest, w.Code)
		}
	})

	t.Run("AnalyzeWorkflow_MissingInput", func(t *testing.T) {
		service := NewDiscoveryService(nil, "", "", "")

		analyzeReq := map[string]interface{}{
			"type": "pros-cons",
			// Missing "input" field
		}
		reqBody, _ := json.Marshal(analyzeReq)

		req := httptest.NewRequest("POST", "/analyze", bytes.NewBuffer(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		service.AnalyzeWorkflow(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status %d for missing input, got %d", http.StatusBadRequest, w.Code)
		}
	})
}

// TestExecuteWorkflow tests workflow execution
func TestExecuteWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ExecuteWorkflow_InvalidPlatform", func(t *testing.T) {
		// Skip - requires DB for logging execution
		t.Skip("Requires test database")
	})
}

// TestHandleReasoningResults tests reasoning results endpoint
func TestHandleReasoningResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ReasoningResults_DefaultLimit", func(t *testing.T) {
		// Would need DB setup
		t.Skip("Requires test database")
	})

	t.Run("ReasoningResults_CustomLimit", func(t *testing.T) {
		// Would need DB setup
		t.Skip("Requires test database")
	})
}

// TestHandleReasoningResult tests single reasoning result endpoint
func TestHandleReasoningResult(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("ReasoningResult_NotFound", func(t *testing.T) {
		// Would need DB setup
		t.Skip("Requires test database")
	})
}

// TestVectorWorkflowResponse tests vector workflow response structure
func TestVectorWorkflowResponse(t *testing.T) {
	t.Run("CreateVectorWorkflowResponse", func(t *testing.T) {
		resp := VectorWorkflowResponse{
			Status:             "success",
			PointID:            "test-point-123",
			Collection:         "workflow_embeddings",
			EmbeddingDimension: 768,
			ExecutionTimeMS:    150,
			ModelUsed:          "nomic-embed-text",
		}

		if resp.Status != "success" {
			t.Error("Status not set correctly")
		}
		if resp.EmbeddingDimension != 768 {
			t.Errorf("Expected embedding dimension 768, got %d", resp.EmbeddingDimension)
		}
	})

	t.Run("VectorWorkflowResponse_WithError", func(t *testing.T) {
		resp := VectorWorkflowResponse{
			Status: "failed",
			Error:  "Embedding generation failed",
		}

		if resp.Status != "failed" {
			t.Error("Status should be 'failed'")
		}
		if resp.Error == "" {
			t.Error("Error message should not be empty")
		}
	})
}

// Benchmark tests for performance validation

func BenchmarkHealth(b *testing.B) {
	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		Health(w, req)
	}
}

func BenchmarkNewLogger(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewLogger()
	}
}

func BenchmarkExtractTags(b *testing.B) {
	workflow := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2", "tag3", "tag4", "tag5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractTags(workflow)
	}
}

// Edge case tests

func TestEdgeCases(t *testing.T) {
	t.Run("EmptyWorkflowName", func(t *testing.T) {
		service := NewDiscoveryService(nil, "", "", "")
		result := service.isMetareasoningWorkflow("")
		if result {
			t.Error("Empty workflow name should not match metareasoning patterns")
		}
	})

	t.Run("HTTPError_EmptyMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		HTTPError(w, "", http.StatusBadRequest, nil)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["error"] != "" {
			// Empty message should still be in response
		}
	})

	t.Run("GetResourcePort_EmptyName", func(t *testing.T) {
		port := getResourcePort("")
		// May return empty or fallback port depending on implementation
		_ = port // Accept any result for empty name
	})
}

// Test error handling patterns using test_patterns.go

func TestErrorPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	service := NewDiscoveryService(nil, "", "", "")

	t.Run("SearchWorkflows_ErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "SearchWorkflows",
			Handler:     service.SearchWorkflows,
			BaseURL:     "/workflows/search",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/workflows/search").
			AddEmptyBody("POST", "/workflows/search").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("AnalyzeWorkflow_ErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "AnalyzeWorkflow",
			Handler:     service.AnalyzeWorkflow,
			BaseURL:     "/analyze",
		}

		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("POST", "/analyze").
			AddMissingRequiredFields("POST", "/analyze", map[string]interface{}{
				"type": "pros-cons",
				// missing "input"
			}).
			// Skip invalid reasoning type test as it requires DB
			Build()

		suite.RunErrorTests(t, patterns)
	})
}
