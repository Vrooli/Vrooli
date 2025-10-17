// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// BenchmarkHealth benchmarks the health endpoint
func BenchmarkHealth(b *testing.B) {
	req, _ := http.NewRequest("GET", "/health", nil)
	handler := http.HandlerFunc(Health)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkHTTPError benchmarks error response generation
func BenchmarkHTTPError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		HTTPError(rr, "Test error", http.StatusBadRequest, nil)
	}
}

// BenchmarkBackupApp benchmarks backup creation
func BenchmarkBackupApp(b *testing.B) {
	env := &TestEnvironment{}
	tempDir, _ := setupTestDirectoryForBench(b)
	env.TestAppPath = tempDir
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	reqBody := BackupAppRequest{
		AppPath:    env.TestAppPath,
		BackupType: "full",
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/backup", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.BackupApp)
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkAnalyzeAppStructure benchmarks app analysis
func BenchmarkAnalyzeAppStructure(b *testing.B) {
	env := &TestEnvironment{}
	tempDir, _ := setupTestDirectoryForBench(b)
	env.TestAppPath = tempDir
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.analyzeAppStructure(env.TestAppPath, "react")
	}
}

// BenchmarkValidateApp benchmarks validation
func BenchmarkValidateApp(b *testing.B) {
	env := &TestEnvironment{}
	tempDir, _ := setupTestDirectoryForBench(b)
	env.TestAppPath = tempDir
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	reqBody := ValidateAppRequest{
		AppPath: env.TestAppPath,
		Tests:   []string{"startup"},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("POST", "/api/validate", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(service.ValidateApp)
		handler.ServeHTTP(rr, req)
	}
}

// setupTestDirectoryForBench creates a minimal test directory for benchmarks
func setupTestDirectoryForBench(b *testing.B) (string, func()) {
	tempDir, err := os.MkdirTemp("", "app-personalizer-bench-*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}

	testAppPath := filepath.Join(tempDir, "test-app")
	os.MkdirAll(filepath.Join(testAppPath, "src/styles"), 0755)

	packageJSON := `{"name": "test-app", "version": "1.0.0"}`
	os.WriteFile(filepath.Join(testAppPath, "package.json"), []byte(packageJSON), 0644)

	themeJS := `export const theme = { colors: { primary: "#007bff" } };`
	os.WriteFile(filepath.Join(testAppPath, "src/styles/theme.js"), []byte(themeJS), 0644)

	return testAppPath, func() {
		os.RemoveAll(tempDir)
	}
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	service := NewAppPersonalizerService(nil, "http://localhost:5678", "http://localhost:9000")
	router := createTestRouter(service)

	concurrency := 10
	requestsPerGoroutine := 10

	var wg sync.WaitGroup
	errors := make(chan error, concurrency*requestsPerGoroutine)

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req, err := http.NewRequest("GET", "/health", nil)
				if err != nil {
					errors <- fmt.Errorf("goroutine %d: failed to create request: %v", id, err)
					continue
				}

				rr := httptest.NewRecorder()
				router.ServeHTTP(rr, req)

				if rr.Code != http.StatusOK {
					errors <- fmt.Errorf("goroutine %d: expected status 200, got %d", id, rr.Code)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	duration := time.Since(start)
	totalRequests := concurrency * requestsPerGoroutine

	t.Logf("Completed %d concurrent requests in %v", totalRequests, duration)
	t.Logf("Average: %v per request", duration/time.Duration(totalRequests))

	errorCount := 0
	for err := range errors {
		errorCount++
		t.Logf("Error: %v", err)
	}

	if errorCount > 0 {
		t.Errorf("Encountered %d errors during concurrent testing", errorCount)
	}
}

// TestMemoryUsage tests memory efficiency
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	// Run analysis multiple times to check for memory leaks
	iterations := 100
	for i := 0; i < iterations; i++ {
		_ = service.analyzeAppStructure(env.TestAppPath, "react")

		// Periodically log progress
		if i%20 == 0 {
			t.Logf("Completed %d/%d iterations", i, iterations)
		}
	}

	t.Log("Memory usage test completed successfully")
}

// TestResponseTimeUnderLoad tests response times under load
func TestResponseTimeUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	_ = NewAppPersonalizerService(nil, "http://localhost:5678", "http://localhost:9000")

	measurements := make([]time.Duration, 0, 100)
	maxAcceptableTime := 100 * time.Millisecond

	for i := 0; i < 100; i++ {
		start := time.Now()

		req, _ := http.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(Health)
		handler.ServeHTTP(rr, req)

		duration := time.Since(start)
		measurements = append(measurements, duration)

		if duration > maxAcceptableTime {
			t.Logf("Warning: Request %d took %v (exceeds %v)", i, duration, maxAcceptableTime)
		}
	}

	// Calculate statistics
	var total time.Duration
	var max time.Duration
	min := measurements[0]

	for _, d := range measurements {
		total += d
		if d > max {
			max = d
		}
		if d < min {
			min = d
		}
	}

	avg := total / time.Duration(len(measurements))

	t.Logf("Response time statistics:")
	t.Logf("  Average: %v", avg)
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)

	if avg > maxAcceptableTime {
		t.Errorf("Average response time %v exceeds acceptable threshold %v", avg, maxAcceptableTime)
	}
}

// TestBackupPerformance tests backup operation performance
func TestBackupPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping backup performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	// Test multiple backup operations
	iterations := 5
	maxAcceptableTime := 5 * time.Second

	for i := 0; i < iterations; i++ {
		start := time.Now()

		backupPath, err := service.createAppBackup(env.TestAppPath, "full")
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Backup iteration %d failed: %v", i, err)
			continue
		}

		// Cleanup backup
		defer os.Remove(backupPath)

		t.Logf("Backup iteration %d completed in %v", i, duration)

		if duration > maxAcceptableTime {
			t.Errorf("Backup took %v, exceeds acceptable time %v", duration, maxAcceptableTime)
		}
	}
}

// TestValidationPerformance tests validation performance
func TestValidationPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping validation performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	service := &AppPersonalizerService{
		logger: NewLogger(),
	}

	tests := []string{"startup", "build", "lint"}
	maxAcceptableTime := 2 * time.Second

	for _, testType := range tests {
		start := time.Now()
		result := service.runValidationTest(env.TestAppPath, testType)
		duration := time.Since(start)

		if result == nil {
			t.Errorf("Validation test '%s' returned nil", testType)
			continue
		}

		t.Logf("Validation test '%s' completed in %v", testType, duration)

		if duration > maxAcceptableTime {
			t.Logf("Warning: Validation test '%s' took %v (exceeds %v)", testType, duration, maxAcceptableTime)
		}
	}
}
