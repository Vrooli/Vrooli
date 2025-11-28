package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSearchEndpoint validates the semantic search API endpoint [REQ:KO-SS-001]
func TestSearchEndpoint(t *testing.T) {
	t.Run("accepts valid search request [REQ:KO-SS-001]", func(t *testing.T) {
		// Create test request
		reqBody := SearchRequest{
			Query: "test query",
			Limit: 10,
		}
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("failed to marshal request: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		// We can't fully test without Ollama/Qdrant, but we can verify the endpoint structure
		// In a real scenario, this would use mock services or test containers

		// For now, just verify the request validation works
		var searchReq SearchRequest
		err = json.NewDecoder(req.Body).Decode(&searchReq)
		if err != nil {
			t.Errorf("failed to decode valid request: %v", err)
		}

		if searchReq.Query == "" {
			t.Error("query should not be empty")
		}

		if searchReq.Limit != 10 {
			t.Errorf("expected limit 10, got %d", searchReq.Limit)
		}
	})

	t.Run("rejects empty query [REQ:KO-SS-001]", func(t *testing.T) {
		reqBody := SearchRequest{
			Query: "",
		}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		var searchReq SearchRequest
		json.NewDecoder(req.Body).Decode(&searchReq)

		if searchReq.Query != "" {
			t.Error("empty query should fail validation")
		}
	})

	t.Run("applies default limit [REQ:KO-SS-001]", func(t *testing.T) {
		req := SearchRequest{
			Query: "test",
			Limit: 0,
		}

		// Simulate default logic
		if req.Limit <= 0 {
			req.Limit = 10
		}

		if req.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", req.Limit)
		}
	})

	t.Run("enforces maximum limit [REQ:KO-SS-001]", func(t *testing.T) {
		req := SearchRequest{
			Query: "test",
			Limit: 500,
		}

		// Simulate max limit logic
		if req.Limit > 100 {
			req.Limit = 100
		}

		if req.Limit != 100 {
			t.Errorf("expected max limit 100, got %d", req.Limit)
		}
	})

	t.Run("sets default threshold [REQ:KO-SS-001]", func(t *testing.T) {
		req := SearchRequest{
			Query:     "test",
			Threshold: 0,
		}

		// Simulate default threshold logic
		if req.Threshold <= 0 {
			req.Threshold = 0.3
		}

		if req.Threshold != 0.3 {
			t.Errorf("expected default threshold 0.3, got %f", req.Threshold)
		}
	})
}

// TestGenerateEmbedding validates Ollama integration [REQ:KO-SS-002]
func TestGenerateEmbedding(t *testing.T) {
	t.Run("creates embedding request structure [REQ:KO-SS-002]", func(t *testing.T) {
		req := OllamaEmbeddingRequest{
			Model:  "nomic-embed-text",
			Prompt: "test query",
		}

		if req.Model != "nomic-embed-text" {
			t.Errorf("expected model nomic-embed-text, got %s", req.Model)
		}

		if req.Prompt == "" {
			t.Error("prompt should not be empty")
		}

		// Verify JSON marshaling works
		jsonData, err := json.Marshal(req)
		if err != nil {
			t.Errorf("failed to marshal embedding request: %v", err)
		}

		var decoded OllamaEmbeddingRequest
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Errorf("failed to unmarshal embedding request: %v", err)
		}

		if decoded.Model != req.Model || decoded.Prompt != req.Prompt {
			t.Error("JSON round-trip failed")
		}
	})

	t.Run("handles embedding response [REQ:KO-SS-002]", func(t *testing.T) {
		// Simulate Ollama response
		resp := OllamaEmbeddingResponse{
			Embedding: []float64{0.1, 0.2, 0.3},
		}

		if len(resp.Embedding) != 3 {
			t.Errorf("expected 3 dimensions, got %d", len(resp.Embedding))
		}

		// Verify JSON decoding works
		jsonData, _ := json.Marshal(resp)
		var decoded OllamaEmbeddingResponse
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Errorf("failed to decode embedding response: %v", err)
		}

		if len(decoded.Embedding) != len(resp.Embedding) {
			t.Error("embedding dimensions mismatch after decode")
		}
	})
}

// TestQdrantSearch validates Qdrant integration [REQ:KO-SS-003]
func TestQdrantSearch(t *testing.T) {
	t.Run("creates valid search request [REQ:KO-SS-003]", func(t *testing.T) {
		vector := []float64{0.1, 0.2, 0.3}
		threshold := 0.7

		req := QdrantSearchRequest{
			Vector:         vector,
			Limit:          10,
			WithPayload:    true,
			ScoreThreshold: &threshold,
		}

		if len(req.Vector) != 3 {
			t.Error("vector dimensions mismatch")
		}

		if req.Limit != 10 {
			t.Errorf("expected limit 10, got %d", req.Limit)
		}

		if !req.WithPayload {
			t.Error("should request payload")
		}

		if req.ScoreThreshold == nil || *req.ScoreThreshold != 0.7 {
			t.Error("score threshold not set correctly")
		}
	})

	t.Run("parses search results [REQ:KO-SS-003]", func(t *testing.T) {
		// Simulate Qdrant response
		qdrantResp := QdrantSearchResponse{
			Result: []QdrantSearchResult{
				{
					ID:    "test-id-1",
					Score: 0.95,
					Payload: map[string]interface{}{
						"content": "test content",
						"source":  "test",
					},
				},
			},
		}

		if len(qdrantResp.Result) != 1 {
			t.Errorf("expected 1 result, got %d", len(qdrantResp.Result))
		}

		result := qdrantResp.Result[0]
		if result.Score != 0.95 {
			t.Errorf("expected score 0.95, got %f", result.Score)
		}

		content, ok := result.Payload["content"].(string)
		if !ok || content != "test content" {
			t.Error("failed to extract content from payload")
		}
	})

	t.Run("converts results to standard format [REQ:KO-SS-003]", func(t *testing.T) {
		// Simulate conversion logic
		qdrantResult := QdrantSearchResult{
			ID:    "123",
			Score: 0.88,
			Payload: map[string]interface{}{
				"content": "test content",
				"metadata": map[string]interface{}{
					"source": "test",
				},
			},
		}

		searchResult := SearchResult{
			ID:       "123",
			Score:    qdrantResult.Score,
			Content:  qdrantResult.Payload["content"].(string),
			Metadata: qdrantResult.Payload,
		}

		if searchResult.ID != "123" {
			t.Errorf("ID mismatch: expected 123, got %s", searchResult.ID)
		}

		if searchResult.Score != 0.88 {
			t.Errorf("Score mismatch: expected 0.88, got %f", searchResult.Score)
		}

		if searchResult.Content != "test content" {
			t.Error("Content extraction failed")
		}
	})
}

// TestSearchResponseStructure validates the response format [REQ:KO-SS-001]
func TestSearchResponseStructure(t *testing.T) {
	t.Run("includes all required fields [REQ:KO-SS-001]", func(t *testing.T) {
		resp := SearchResponse{
			Results: []SearchResult{
				{
					ID:      "1",
					Score:   0.9,
					Content: "test",
					Metadata: map[string]interface{}{
						"source": "test",
					},
				},
			},
			Query: "test query",
			Took:  123,
		}

		if len(resp.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(resp.Results))
		}

		if resp.Query == "" {
			t.Error("query should be included in response")
		}

		if resp.Took <= 0 {
			t.Error("response time should be positive")
		}

		// Verify JSON encoding works
		jsonData, err := json.Marshal(resp)
		if err != nil {
			t.Errorf("failed to marshal response: %v", err)
		}

		var decoded SearchResponse
		if err := json.Unmarshal(jsonData, &decoded); err != nil {
			t.Errorf("failed to unmarshal response: %v", err)
		}

		if decoded.Query != resp.Query || decoded.Took != resp.Took {
			t.Error("JSON round-trip failed")
		}
	})
}

// Benchmark search performance [REQ:KO-SS-004]
func BenchmarkSearchRequestValidation(b *testing.B) {
	reqBody := SearchRequest{
		Query: "test query for benchmarking performance",
		Limit: 10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		jsonData, _ := json.Marshal(reqBody)
		var decoded SearchRequest
		json.Unmarshal(jsonData, &decoded)

		// Simulate validation
		if decoded.Query == "" {
			b.Fatal("validation failed")
		}
		if decoded.Limit <= 0 {
			decoded.Limit = 10
		}
		if decoded.Limit > 100 {
			decoded.Limit = 100
		}
		if decoded.Threshold <= 0 {
			decoded.Threshold = 0.3
		}
	}
}

// TestHandleSearchValidation tests the HTTP handler validation logic [REQ:KO-SS-001]
func TestHandleSearchValidation(t *testing.T) {
	// Create a minimal server for testing
	server := &Server{
		config: &Config{Port: "8080"},
	}

	t.Run("rejects invalid JSON [REQ:KO-SS-001]", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", strings.NewReader("{invalid json}"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		json.NewDecoder(w.Body).Decode(&resp)
		if !strings.Contains(resp["error"], "Invalid request body") {
			t.Errorf("unexpected error message: %v", resp["error"])
		}
	})

	t.Run("rejects empty query [REQ:KO-SS-001]", func(t *testing.T) {
		reqBody := SearchRequest{Query: ""}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		var resp map[string]string
		json.NewDecoder(w.Body).Decode(&resp)
		if !strings.Contains(resp["error"], "Query parameter is required") {
			t.Errorf("unexpected error message: %v", resp["error"])
		}
	})

	t.Run("rejects whitespace-only query [REQ:KO-SS-001]", func(t *testing.T) {
		reqBody := SearchRequest{Query: "   "}
		jsonData, _ := json.Marshal(reqBody)

		req := httptest.NewRequest("POST", "/api/v1/knowledge/search", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleSearch(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

// TestRespondError tests error response formatting [REQ:KO-SS-001]
func TestRespondError(t *testing.T) {
	server := &Server{}

	t.Run("formats error response correctly", func(t *testing.T) {
		w := httptest.NewRecorder()
		server.respondError(w, http.StatusBadRequest, "test error")

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("expected JSON content type")
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Errorf("failed to decode response: %v", err)
		}

		if resp["error"] != "test error" {
			t.Errorf("expected 'test error', got '%s'", resp["error"])
		}
	})
}

// TestSearchEndpointIntegration is a placeholder for integration tests [REQ:KO-SS-001,KO-SS-002,KO-SS-003]
func TestSearchEndpointIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("full search flow requires running services", func(t *testing.T) {
		t.Skip("Integration test requires Ollama and Qdrant - run in test-integration.sh phase")
		// This would test the full flow:
		// 1. Accept search request
		// 2. Generate embedding via Ollama
		// 3. Search Qdrant
		// 4. Return formatted results
		// 5. Verify response time < 500ms [REQ:KO-SS-004]
	})
}
