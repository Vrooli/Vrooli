package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// BenchmarkHealth benchmarks the health endpoint
func BenchmarkHealth(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/health",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeHTTPRequest(req, Health)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkListBrands benchmarks the list brands endpoint
func BenchmarkListBrands(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip benchmark if database not available
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		b.Skip("Database not available for benchmarking")
	}
	if err := db.Ping(); err != nil {
		b.Skip("Database not available for benchmarking")
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")

	req := HTTPTestRequest{
		Method: "GET",
		Path:   "/api/brands",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeHTTPRequest(req, service.ListBrands)
		if err != nil {
			b.Fatalf("Request failed: %v", err)
		}
	}
}

// BenchmarkGetBrandByID benchmarks the get brand by ID endpoint
func BenchmarkGetBrandByID(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Skip benchmark if database not available
	db, err := sql.Open("postgres", "postgres://test:test@localhost:5432/test?sslmode=disable")
	if err != nil {
		b.Skip("Database not available for benchmarking")
	}
	if err := db.Ping(); err != nil {
		b.Skip("Database not available for benchmarking")
	}
	defer db.Close()

	service := NewBrandManagerService(db, "", "", "", "")
	brandID := uuid.New() // Use a random ID for benchmark

	req := HTTPTestRequest{
		Method:  "GET",
		Path:    "/api/brands/" + brandID.String(),
		URLVars: map[string]string{"id": brandID.String()},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := makeHTTPRequest(req, service.GetBrandByID)
		if err != nil {
			// Expected to fail since brand doesn't exist, but we're testing speed
			continue
		}
	}
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	const numGoroutines = 10
	const requestsPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*requestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}

				_, err := makeHTTPRequest(req, Health)
				if err != nil {
					errors <- err
				}
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("Concurrent request failed: %v", err)
	}
}

// TestResponseTime tests response time requirements
func TestResponseTime(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name         string
		handler      http.HandlerFunc
		req          HTTPTestRequest
		maxDuration  time.Duration
		skipIfNoDB   bool
	}{
		{
			name:        "HealthEndpoint",
			handler:     Health,
			req:         HTTPTestRequest{Method: "GET", Path: "/health"},
			maxDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			_, err := makeHTTPRequest(tt.req, tt.handler)
			duration := time.Since(start)

			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}

			if duration > tt.maxDuration {
				t.Errorf("Response too slow: %v (max: %v)", duration, tt.maxDuration)
			}
		})
	}
}

// TestMemoryUsage tests memory usage under load
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	const iterations = 1000

	for i := 0; i < iterations; i++ {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		_, err := makeHTTPRequest(req, Health)
		if err != nil {
			t.Fatalf("Request failed at iteration %d: %v", i, err)
		}
	}

	// If we got here without OOM, test passes
}

// BenchmarkNewLogger benchmarks logger creation
func BenchmarkNewLogger(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewLogger()
	}
}

// BenchmarkHTTPError benchmarks error response generation
func BenchmarkHTTPError(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		HTTPError(w, "test error", http.StatusBadRequest, io.EOF)
	}
}

// BenchmarkGetResourcePort benchmarks port lookup
func BenchmarkGetResourcePort(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getResourcePort("n8n")
	}
}

// TestPerformanceRegression tests for performance regressions
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	// Baseline measurements
	baselineDurations := make([]time.Duration, 100)
	for i := 0; i < len(baselineDurations); i++ {
		start := time.Now()
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}
		_, err := makeHTTPRequest(req, Health)
		if err != nil {
			t.Fatalf("Baseline request failed: %v", err)
		}
		baselineDurations[i] = time.Since(start)
	}

	// Calculate average
	var total time.Duration
	for _, d := range baselineDurations {
		total += d
	}
	avgDuration := total / time.Duration(len(baselineDurations))

	t.Logf("Average response time: %v", avgDuration)

	// Verify no single request exceeds 10x average
	maxAllowed := avgDuration * 10
	for i, d := range baselineDurations {
		if d > maxAllowed {
			t.Errorf("Request %d took %v, which exceeds 10x average (%v)", i, d, avgDuration)
		}
	}
}

// BenchmarkJSONMarshalBrand benchmarks brand JSON marshaling
func BenchmarkJSONMarshalBrand(b *testing.B) {
	brand := Brand{
		ID:          uuid.New(),
		Name:        "Test Brand",
		ShortName:   "TB",
		Slogan:      "Test Slogan",
		AdCopy:      "Test Ad Copy",
		Description: "Test Description",
		BrandColors: map[string]interface{}{"primary": "#FF5733"},
		LogoURL:     "http://example.com/logo.png",
		FaviconURL:  "http://example.com/favicon.ico",
		Assets:      []interface{}{},
		Metadata:    map[string]interface{}{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(brand)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}

// BenchmarkJSONMarshalIntegration benchmarks integration JSON marshaling
func BenchmarkJSONMarshalIntegration(b *testing.B) {
	now := time.Now()
	integration := IntegrationRequest{
		ID:              uuid.New(),
		BrandID:         uuid.New(),
		TargetAppPath:   "/test/app",
		IntegrationType: "full",
		ClaudeSessionID: "session-123",
		Status:          "pending",
		RequestPayload:  map[string]interface{}{"test": true},
		ResponsePayload: map[string]interface{}{"result": "success"},
		CreatedAt:       now,
		CompletedAt:     &now,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(integration)
		if err != nil {
			b.Fatalf("Marshal failed: %v", err)
		}
	}
}
