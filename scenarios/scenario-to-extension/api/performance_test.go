package main

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// TestPerformanceExtensionGeneration tests performance of extension generation
func TestPerformanceExtensionGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("SingleExtensionGenerationTime", func(t *testing.T) {
		req := TestData.GenerateExtensionRequest(
			"perf-test-scenario",
			"Performance Test Extension",
			"http://localhost:3000",
		)

		start := time.Now()

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to generate extension: %v", err)
		}

		if w.Code != 201 {
			t.Fatalf("Expected status 201, got %d", w.Code)
		}

		// Extension generation should return immediately (async)
		// API response should be < 100ms
		maxDuration := 100 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("Extension generation API took too long: %v (max: %v)", duration, maxDuration)
		}

		t.Logf("Extension generation API responded in %v", duration)
	})

	t.Run("StatusCheckPerformance", func(t *testing.T) {
		// Create a build first
		req := TestData.GenerateExtensionRequest(
			"status-perf-test",
			"Status Perf Test",
			"http://localhost:3000",
		)

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
		if err != nil {
			t.Fatalf("Failed to generate extension: %v", err)
		}

		response := assertJSONResponse(t, w, 201, nil)
		buildID := response["build_id"].(string)

		// Now test status check performance
		const iterations = 100
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			_, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/extension/status/" + buildID,
				URLVars: map[string]string{"build_id": buildID},
			})

			if err != nil {
				t.Fatalf("Failed to check status: %v", err)
			}

			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / iterations
		maxAvgDuration := 10 * time.Millisecond

		if avgDuration > maxAvgDuration {
			t.Errorf("Average status check took too long: %v (max: %v)", avgDuration, maxAvgDuration)
		}

		t.Logf("Average status check time over %d iterations: %v", iterations, avgDuration)
	})

	t.Run("ConcurrentBuildPerformance", func(t *testing.T) {
		const concurrentBuilds = 10

		start := time.Now()
		done := make(chan bool, concurrentBuilds)

		for i := 0; i < concurrentBuilds; i++ {
			go func(index int) {
				req := TestData.GenerateExtensionRequest(
					fmt.Sprintf("concurrent-perf-%d", index),
					fmt.Sprintf("Concurrent Perf %d", index),
					"http://localhost:3000",
				)

				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})

				if err != nil {
					t.Errorf("Failed to generate extension %d: %v", index, err)
					done <- false
					return
				}

				if w.Code != 201 {
					t.Errorf("Expected status 201 for build %d, got %d", index, w.Code)
					done <- false
					return
				}

				done <- true
			}(i)
		}

		successCount := 0
		for i := 0; i < concurrentBuilds; i++ {
			if <-done {
				successCount++
			}
		}

		duration := time.Since(start)

		if successCount != concurrentBuilds {
			t.Errorf("Expected %d successful builds, got %d", concurrentBuilds, successCount)
		}

		// All concurrent requests should complete within reasonable time
		maxDuration := 1 * time.Second
		if duration > maxDuration {
			t.Errorf("Concurrent builds took too long: %v (max: %v)", duration, maxDuration)
		}

		t.Logf("%d concurrent builds completed in %v (avg: %v per build)",
			concurrentBuilds, duration, duration/time.Duration(concurrentBuilds))
	})

	t.Run("ListBuildsScalability", func(t *testing.T) {
		// Create multiple builds
		const buildCount = 50

		for i := 0; i < buildCount; i++ {
			req := TestData.GenerateExtensionRequest(
				fmt.Sprintf("scale-test-%d", i),
				fmt.Sprintf("Scale Test %d", i),
				"http://localhost:3000",
			)

			_, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})
			if err != nil {
				t.Fatalf("Failed to create build %d: %v", i, err)
			}
		}

		// Test list builds performance
		start := time.Now()

		w, err := executeRequest(listBuildsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/builds",
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to list builds: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		// Should list 50+ builds quickly
		maxDuration := 50 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("Listing %d builds took too long: %v (max: %v)", buildCount, duration, maxDuration)
		}

		t.Logf("Listed %d+ builds in %v", buildCount, duration)
	})

	t.Run("TemplateListingPerformance", func(t *testing.T) {
		const iterations = 1000
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			w, err := executeRequest(listTemplatesHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/extension/templates",
			})

			if err != nil {
				t.Fatalf("Failed to list templates: %v", err)
			}

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / iterations
		maxAvgDuration := 1 * time.Millisecond

		if avgDuration > maxAvgDuration {
			t.Errorf("Average template listing took too long: %v (max: %v)", avgDuration, maxAvgDuration)
		}

		t.Logf("Average template listing time over %d iterations: %v", iterations, avgDuration)
	})

	t.Run("ExtensionTestingPerformance", func(t *testing.T) {
		req := TestData.GenerateTestRequest("/tmp/perf-test", []string{
			"https://example.com",
		})

		start := time.Now()

		w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/test",
			Body:   req,
		})

		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Failed to test extension: %v", err)
		}

		if w.Code != 200 {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		// Extension testing should be reasonably fast for simulated tests
		maxDuration := 100 * time.Millisecond
		if duration > maxDuration {
			t.Errorf("Extension testing took too long: %v (max: %v)", duration, maxDuration)
		}

		t.Logf("Extension testing completed in %v", duration)
	})

	t.Run("HealthCheckPerformance", func(t *testing.T) {
		const iterations = 1000
		var totalDuration time.Duration

		for i := 0; i < iterations; i++ {
			start := time.Now()

			w, err := executeRequest(healthHandler, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/health",
			})

			if err != nil {
				t.Fatalf("Failed health check: %v", err)
			}

			if w.Code != 200 {
				t.Fatalf("Expected status 200, got %d", w.Code)
			}

			totalDuration += time.Since(start)
		}

		avgDuration := totalDuration / iterations
		maxAvgDuration := 5 * time.Millisecond

		if avgDuration > maxAvgDuration {
			t.Errorf("Average health check took too long: %v (max: %v)", avgDuration, maxAvgDuration)
		}

		t.Logf("Average health check time over %d iterations: %v", iterations, avgDuration)
	})

	t.Run("MemoryUsageUnderLoad", func(t *testing.T) {
		// Test that memory usage remains reasonable under load
		const buildCount = 100

		for i := 0; i < buildCount; i++ {
			req := TestData.GenerateExtensionRequest(
				fmt.Sprintf("memory-test-%d", i),
				fmt.Sprintf("Memory Test %d", i),
				"http://localhost:3000",
			)

			w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Failed to create build %d: %v", i, err)
			}

			if w.Code != 201 {
				t.Fatalf("Expected status 201 for build %d, got %d", i, w.Code)
			}
		}

		totalBuilds := buildManager.Count()

		if totalBuilds < buildCount {
			t.Errorf("Expected at least %d builds in memory, got %d", buildCount, totalBuilds)
		}

		t.Logf("Successfully maintained %d builds in memory", totalBuilds)
	})
}

// TestPerformanceBuildIDGeneration tests build ID generation performance
func TestPerformanceBuildIDGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("BuildIDGenerationSpeed", func(t *testing.T) {
		const iterations = 10000
		buildIDs := make(map[string]bool, iterations)

		start := time.Now()

		for i := 0; i < iterations; i++ {
			buildID := generateBuildID()
			buildIDs[buildID] = true
		}

		duration := time.Since(start)

		// Check uniqueness
		if len(buildIDs) != iterations {
			t.Errorf("Expected %d unique build IDs, got %d", iterations, len(buildIDs))
		}

		avgDuration := duration / iterations
		maxAvgDuration := 1 * time.Microsecond

		if avgDuration > maxAvgDuration {
			t.Errorf("Average build ID generation too slow: %v (max: %v)", avgDuration, maxAvgDuration)
		}

		t.Logf("Generated %d unique build IDs in %v (avg: %v)", iterations, duration, avgDuration)
	})
}

// Benchmark functions for detailed performance analysis
func BenchmarkGenerateExtension(b *testing.B) {
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

	env := &TestEnvironment{
		TempDir: b.TempDir(),
	}

	cfg := &Config{
		Port:           3201,
		TemplatesPath:  env.TempDir + "/templates",
		OutputPath:     env.TempDir + "/output",
		BrowserlessURL: "http://localhost:3000",
	}

	config = cfg
	buildManager = NewBuildManager()

	req := TestData.GenerateExtensionRequest(
		"benchmark-scenario",
		"Benchmark Extension",
		"http://localhost:3000",
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

	env := &TestEnvironment{
		TempDir: b.TempDir(),
	}

	cfg := &Config{
		Port:           3201,
		TemplatesPath:  env.TempDir + "/templates",
		OutputPath:     env.TempDir + "/output",
		BrowserlessURL: "http://localhost:3000",
	}

	config = cfg
	buildManager = NewBuildManager()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = executeRequest(healthHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})
	}
}

func BenchmarkListTemplates(b *testing.B) {
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	defer os.Unsetenv("VROOLI_LIFECYCLE_MANAGED")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = executeRequest(listTemplatesHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/extension/templates",
		})
	}
}

func BenchmarkBuildIDGeneration(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = generateBuildID()
	}
}

func BenchmarkCountBuildsByStatus(b *testing.B) {
	// Setup test builds
	buildManager = NewBuildManager()
	for i := 0; i < 1000; i++ {
		status := "building"
		if i%3 == 0 {
			status = "ready"
		} else if i%3 == 1 {
			status = "failed"
		}
		buildManager.Add(&ExtensionBuild{BuildID: fmt.Sprintf("build-%d", i), Status: status})
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = buildManager.CountByStatus("building")
	}
}
