package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Benchmark tests for API performance

func BenchmarkHealthHandler(b *testing.B) {
	handler, _ := setupTestHandler()
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		handler.HealthHandler(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkWriteJSON(b *testing.B) {
	data := map[string]interface{}{
		"id":      uuid.New(),
		"name":    "Test Workflow",
		"status":  "active",
		"count":   42,
		"enabled": true,
		"tags":    []string{"test", "benchmark"},
		"metadata": map[string]interface{}{
			"created": time.Now(),
			"version": 1.0,
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		writeJSON(w, http.StatusOK, data)
	}
}

func BenchmarkGenerateWorkflowHandler(b *testing.B) {
	handler, mockService := setupTestHandler()
	
	// Setup mock response
	mockService.generateResp = &WorkflowCreate{
		Name:        "Generated Workflow",
		Description: "AI generated workflow",
		Type:        "automation",
		Platform:    "n8n",
		Config:      json.RawMessage(`{"nodes": []}`),
	}
	
	requestBody := GenerateRequest{
		Prompt:      "Create a simple data processing workflow",
		Platform:    "n8n",
		Model:       "llama2",
		Temperature: 0.7,
	}
	
	body, _ := json.Marshal(requestBody)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/workflows/generate", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		handler.GenerateWorkflowHandler(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkGetModelsHandler(b *testing.B) {
	handler, mockService := setupTestHandler()
	
	mockService.models = []string{"llama2", "codellama", "mistral", "gpt-4"}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/models", nil)
		w := httptest.NewRecorder()
		
		handler.GetModelsHandler(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkGetPlatformsHandler(b *testing.B) {
	handler, mockService := setupTestHandler()
	
	mockService.platforms = []PlatformInfo{
		{Name: "n8n", Description: "n8n platform", Status: true, URL: "http://localhost:5678"},
		{Name: "windmill", Description: "windmill platform", Status: true, URL: "http://localhost:8000"},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/platforms", nil)
		w := httptest.NewRecorder()
		
		handler.GetPlatformsHandler(w, req)
		
		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

func BenchmarkImportWorkflowHandler(b *testing.B) {
	handler, _ := setupTestHandler()
	
	requestBody := ImportRequest{
		Platform: "n8n",
		Name:     "Imported Workflow",
		Data:     json.RawMessage(`{"nodes": [{"id": "1", "type": "webhook"}]}`),
	}
	
	body, _ := json.Marshal(requestBody)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/workflows/import", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		
		handler.ImportWorkflowHandler(w, req)
		
		if w.Code != http.StatusCreated {
			b.Fatalf("Expected status 201, got %d", w.Code)
		}
	}
}

func BenchmarkPerformanceMiddleware(b *testing.B) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	wrapped := PerformanceMiddleware(testHandler)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)
	}
}

func BenchmarkCacheMiddleware(b *testing.B) {
	// Clear cache before benchmark
	responseCache = make(map[string]CacheableResponse)
	
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate database lookup
		time.Sleep(10 * time.Millisecond)
		writeJSON(w, http.StatusOK, map[string]string{"data": "test"})
	})
	
	cached := CacheMiddleware(300)(testHandler) // 5 minute cache
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/cached-endpoint", nil)
		w := httptest.NewRecorder()
		cached.ServeHTTP(w, req)
	}
}

// Benchmark JSON marshaling performance
func BenchmarkJSONMarshal(b *testing.B) {
	workflow := Workflow{
		ID:                uuid.New(),
		Name:              "Performance Test Workflow",
		Description:       "Used for performance benchmarking",
		Type:              "automation",
		Platform:          "n8n",
		Config:            json.RawMessage(`{"nodes": [{"id": "1", "type": "webhook"}]}`),
		EstimatedDuration: 1000,
		Version:           1,
		IsActive:          true,
		IsBuiltin:         false,
		Tags:              []string{"performance", "test", "benchmark"},
		UsageCount:        100,
		SuccessCount:      95,
		FailureCount:      5,
		CreatedBy:         "benchmark",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(workflow)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark UUID generation
func BenchmarkUUIDGeneration(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}

// Benchmark query parameter parsing
func BenchmarkParseQueryInt(b *testing.B) {
	req := httptest.NewRequest("GET", "/?page=10&size=50&offset=100", nil)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = parseQueryInt(req, "page", 1)
		_ = parseQueryInt(req, "size", 20)
		_ = parseQueryInt(req, "offset", 0)
	}
}

// Benchmark configuration loading
func BenchmarkLoadConfig(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = LoadConfig()
	}
}

// Performance regression test
func TestResponseTimeRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}
	
	handler, _ := setupTestHandler()
	
	// Test health endpoint should be very fast
	start := time.Now()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.HealthHandler(w, req)
	duration := time.Since(start)
	
	if duration > 50*time.Millisecond {
		t.Errorf("Health endpoint too slow: %v (should be <50ms)", duration)
	}
	
	// Test models endpoint should be reasonably fast
	start = time.Now()
	req = httptest.NewRequest("GET", "/models", nil)
	w = httptest.NewRecorder()
	handler.GetModelsHandler(w, req)
	duration = time.Since(start)
	
	if duration > 100*time.Millisecond {
		t.Errorf("Models endpoint too slow: %v (should be <100ms)", duration)
	}
}

// Memory usage test
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}
	
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	// Simulate handling many requests
	handler, _ := setupTestHandler()
	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		handler.HealthHandler(w, req)
	}
	
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	allocated := m2.Alloc - m1.Alloc
	
	// Memory usage should be reasonable (less than 10MB for 1000 requests)
	if allocated > 10*1024*1024 {
		t.Errorf("Memory usage too high: %d bytes allocated for 1000 requests", allocated)
	}
	
	t.Logf("Memory allocated for 1000 requests: %d bytes", allocated)
}

// Imports are already added at the top

// Benchmark with various payload sizes
func BenchmarkVariousPayloadSizes(b *testing.B) {
	sizes := []int{100, 1000, 10000, 100000}
	
	for _, size := range sizes {
		b.Run(fmt.Sprintf("PayloadSize-%d", size), func(b *testing.B) {
			// Create payload of specified size
			data := make(map[string]interface{})
			for i := 0; i < size; i++ {
				data[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				writeJSON(w, http.StatusOK, data)
			}
		})
	}
}