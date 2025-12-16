package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestCalculateCoherence validates coherence score calculation [REQ:KO-QM-001]
func TestCalculateCoherence(t *testing.T) {
	t.Run("perfect coherence for single vector [REQ:KO-QM-001]", func(t *testing.T) {
		vectors := [][]float64{
			{0.1, 0.2, 0.3},
		}

		coherence := calculateCoherence(vectors)

		if coherence != 1.0 {
			t.Errorf("expected coherence 1.0, got %f", coherence)
		}
	})

	t.Run("high coherence for similar vectors [REQ:KO-QM-001]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{0.9, 0.1, 0.0},
			{0.8, 0.2, 0.0},
		}

		coherence := calculateCoherence(vectors)

		if coherence < 0.8 {
			t.Errorf("expected high coherence (>0.8), got %f", coherence)
		}
	})

	t.Run("low coherence for dissimilar vectors [REQ:KO-QM-001]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{0.0, 1.0, 0.0},
			{0.0, 0.0, 1.0},
		}

		coherence := calculateCoherence(vectors)

		if coherence > 0.6 {
			t.Errorf("expected low coherence (<0.6), got %f", coherence)
		}
	})

	t.Run("handles empty input [REQ:KO-QM-001]", func(t *testing.T) {
		vectors := [][]float64{}

		coherence := calculateCoherence(vectors)

		if coherence != 1.0 {
			t.Errorf("expected default coherence 1.0 for empty input, got %f", coherence)
		}
	})
}

// TestCalculateFreshness validates freshness score calculation [REQ:KO-QM-002]
func TestCalculateFreshness(t *testing.T) {
	t.Run("perfect freshness for recent timestamps [REQ:KO-QM-002]", func(t *testing.T) {
		now := time.Now()
		timestamps := []time.Time{
			now.Add(-1 * time.Hour),
			now.Add(-2 * time.Hour),
			now.Add(-5 * time.Hour),
		}

		freshness := calculateFreshness(timestamps)

		if freshness < 0.95 {
			t.Errorf("expected high freshness (>0.95) for recent timestamps, got %f", freshness)
		}
	})

	t.Run("lower freshness for old timestamps [REQ:KO-QM-002]", func(t *testing.T) {
		now := time.Now()
		timestamps := []time.Time{
			now.Add(-90 * 24 * time.Hour), // 90 days old
			now.Add(-120 * 24 * time.Hour), // 120 days old
		}

		freshness := calculateFreshness(timestamps)

		if freshness > 0.3 {
			t.Errorf("expected low freshness (<0.3) for old timestamps, got %f", freshness)
		}
	})

	t.Run("medium freshness for mixed timestamps [REQ:KO-QM-002]", func(t *testing.T) {
		now := time.Now()
		timestamps := []time.Time{
			now.Add(-1 * 24 * time.Hour),  // 1 day
			now.Add(-30 * 24 * time.Hour), // 30 days (half-life)
			now.Add(-60 * 24 * time.Hour), // 60 days
		}

		freshness := calculateFreshness(timestamps)

		if freshness < 0.3 || freshness > 0.8 {
			t.Errorf("expected medium freshness (0.3-0.8) for mixed timestamps, got %f", freshness)
		}
	})

	t.Run("handles empty timestamps [REQ:KO-QM-002]", func(t *testing.T) {
		timestamps := []time.Time{}

		freshness := calculateFreshness(timestamps)

		if freshness != 0.0 {
			t.Errorf("expected freshness 0.0 for empty input, got %f", freshness)
		}
	})
}

// TestDetectRedundancy validates duplicate detection [REQ:KO-QM-003]
func TestDetectRedundancy(t *testing.T) {
	t.Run("no redundancy for dissimilar vectors [REQ:KO-QM-003]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{0.0, 1.0, 0.0},
			{0.0, 0.0, 1.0},
		}

		redundancy := detectRedundancy(vectors, 0.95)

		if redundancy > 0.01 {
			t.Errorf("expected zero redundancy, got %f", redundancy)
		}
	})

	t.Run("high redundancy for duplicate vectors [REQ:KO-QM-003]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{1.0, 0.0, 0.0},
			{1.0, 0.0, 0.0},
		}

		redundancy := detectRedundancy(vectors, 0.95)

		if redundancy < 0.9 {
			t.Errorf("expected high redundancy (>0.9), got %f", redundancy)
		}
	})

	t.Run("partial redundancy for mixed vectors [REQ:KO-QM-003]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{1.0, 0.0, 0.0}, // duplicate
			{0.0, 1.0, 0.0}, // unique
		}

		redundancy := detectRedundancy(vectors, 0.95)

		if redundancy < 0.2 || redundancy > 0.6 {
			t.Errorf("expected partial redundancy (0.2-0.6), got %f", redundancy)
		}
	})

	t.Run("handles single vector [REQ:KO-QM-003]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
		}

		redundancy := detectRedundancy(vectors, 0.95)

		if redundancy != 0.0 {
			t.Errorf("expected zero redundancy for single vector, got %f", redundancy)
		}
	})
}

// TestCosineSimilarity validates cosine similarity calculation
func TestCosineSimilarity(t *testing.T) {
	t.Run("identical vectors have similarity 1.0", func(t *testing.T) {
		a := []float64{1.0, 2.0, 3.0}
		b := []float64{1.0, 2.0, 3.0}

		similarity := cosineSimilarity(a, b)

		if math.Abs(similarity-1.0) > 0.001 {
			t.Errorf("expected similarity 1.0, got %f", similarity)
		}
	})

	t.Run("orthogonal vectors have similarity 0.0", func(t *testing.T) {
		a := []float64{1.0, 0.0, 0.0}
		b := []float64{0.0, 1.0, 0.0}

		similarity := cosineSimilarity(a, b)

		if math.Abs(similarity) > 0.001 {
			t.Errorf("expected similarity 0.0, got %f", similarity)
		}
	})

	t.Run("opposite vectors have similarity -1.0", func(t *testing.T) {
		a := []float64{1.0, 0.0, 0.0}
		b := []float64{-1.0, 0.0, 0.0}

		similarity := cosineSimilarity(a, b)

		if math.Abs(similarity-(-1.0)) > 0.001 {
			t.Errorf("expected similarity -1.0, got %f", similarity)
		}
	})

	t.Run("handles zero vectors", func(t *testing.T) {
		a := []float64{0.0, 0.0, 0.0}
		b := []float64{1.0, 2.0, 3.0}

		similarity := cosineSimilarity(a, b)

		if similarity != 0.0 {
			t.Errorf("expected similarity 0.0 for zero vector, got %f", similarity)
		}
	})

	t.Run("handles mismatched dimensions", func(t *testing.T) {
		a := []float64{1.0, 2.0}
		b := []float64{1.0, 2.0, 3.0}

		similarity := cosineSimilarity(a, b)

		if similarity != 0.0 {
			t.Errorf("expected similarity 0.0 for mismatched dimensions, got %f", similarity)
		}
	})
}

// TestCalculateQualityMetrics validates combined metrics calculation [REQ:KO-QM-001,KO-QM-002,KO-QM-003]
func TestCalculateQualityMetrics(t *testing.T) {
	t.Run("calculates all metrics together [REQ:KO-QM-001,KO-QM-002,KO-QM-003]", func(t *testing.T) {
		vectors := [][]float64{
			{1.0, 0.0, 0.0},
			{0.9, 0.1, 0.0},
		}

		now := time.Now()
		timestamps := []time.Time{
			now.Add(-1 * 24 * time.Hour),
			now.Add(-2 * 24 * time.Hour),
		}

		metrics := calculateQualityMetrics(vectors, timestamps)

		if metrics.Coherence == nil || *metrics.Coherence < 0 || *metrics.Coherence > 1 {
			t.Errorf("coherence out of range [0,1]: %v", metrics.Coherence)
		}

		if metrics.Freshness == nil || *metrics.Freshness < 0 || *metrics.Freshness > 1 {
			t.Errorf("freshness out of range [0,1]: %v", metrics.Freshness)
		}

		if metrics.Redundancy == nil || *metrics.Redundancy < 0 || *metrics.Redundancy > 1 {
			t.Errorf("redundancy out of range [0,1]: %v", metrics.Redundancy)
		}

		if metrics.Coverage == nil || *metrics.Coverage < 0 || *metrics.Coverage > 1 {
			t.Errorf("coverage out of range [0,1]: %v", metrics.Coverage)
		}
	})
}

// TestHandleHealthEndpoint validates health API endpoint [REQ:KO-QM-004]
func TestHandleHealthEndpoint(t *testing.T) {
	server := &Server{
		config: &Config{Port: "8080"},
	}

	t.Run("returns valid health response [REQ:KO-QM-004]", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/knowledge/health", nil)
		w := httptest.NewRecorder()

		server.handleHealthEndpoint(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Type") != "application/json" {
			t.Error("expected JSON content type")
		}

		var response HealthResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Errorf("failed to decode response: %v", err)
		}

		// Verify response structure
		if response.OverallHealth == "" {
			t.Error("expected overall health status")
		}

		if response.Timestamp.IsZero() {
			t.Error("expected valid timestamp")
		}

		// Metrics are optional; when present they must be in range.
		if response.OverallMetrics != nil {
			metrics := response.OverallMetrics
			if metrics.Coherence != nil && (*metrics.Coherence < 0 || *metrics.Coherence > 1) {
				t.Errorf("coherence out of range [0,1]: %f", *metrics.Coherence)
			}
			if metrics.Freshness != nil && (*metrics.Freshness < 0 || *metrics.Freshness > 1) {
				t.Errorf("freshness out of range [0,1]: %f", *metrics.Freshness)
			}
			if metrics.Redundancy != nil && (*metrics.Redundancy < 0 || *metrics.Redundancy > 1) {
				t.Errorf("redundancy out of range [0,1]: %f", *metrics.Redundancy)
			}
		}
	})
}

// TestFormatHealthStatus validates health status formatting
func TestFormatHealthStatus(t *testing.T) {
	tests := []struct {
		score    *float64
		expected string
	}{
		{nil, "unknown"},
		{floatPtr(0.9), "excellent"},
		{floatPtr(0.8), "excellent"},
		{floatPtr(0.7), "good"},
		{floatPtr(0.6), "good"},
		{floatPtr(0.5), "fair"},
		{floatPtr(0.4), "fair"},
		{floatPtr(0.3), "poor"},
		{floatPtr(0.1), "poor"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			status := formatHealthStatus(tt.score)
			if status != tt.expected {
				t.Errorf("score %v: expected %s, got %s", tt.score, tt.expected, status)
			}
		})
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

// BenchmarkCosineSimilarity benchmarks similarity calculation [REQ:KO-QM-005]
func BenchmarkCosineSimilarity(b *testing.B) {
	// Simulate typical embedding dimension (768 for many models)
	vec1 := make([]float64, 768)
	vec2 := make([]float64, 768)
	for i := range vec1 {
		vec1[i] = float64(i) / 768.0
		vec2[i] = float64(i+1) / 768.0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cosineSimilarity(vec1, vec2)
	}
}

// BenchmarkCalculateCoherence benchmarks coherence calculation [REQ:KO-QM-005]
func BenchmarkCalculateCoherence(b *testing.B) {
	// Simulate collection with 100 vectors
	vectors := make([][]float64, 100)
	for i := range vectors {
		vectors[i] = make([]float64, 768)
		for j := range vectors[i] {
			vectors[i][j] = float64(i+j) / 1000.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateCoherence(vectors)
	}
}

// BenchmarkDetectRedundancy benchmarks redundancy detection [REQ:KO-QM-005]
func BenchmarkDetectRedundancy(b *testing.B) {
	vectors := make([][]float64, 100)
	for i := range vectors {
		vectors[i] = make([]float64, 768)
		for j := range vectors[i] {
			vectors[i][j] = float64(i+j) / 1000.0
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detectRedundancy(vectors, 0.95)
	}
}
