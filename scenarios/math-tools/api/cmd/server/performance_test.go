package main

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestPerformanceCalculations tests calculation performance
func TestPerformanceCalculations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("BasicOperationsSpeed", func(t *testing.T) {
		operations := []string{"add", "subtract", "multiply", "divide"}
		data := []float64{100.5, 50.25, 25.125, 12.5}

		for _, op := range operations {
			t.Run(op, func(t *testing.T) {
				start := time.Now()
				iterations := 100

				for i := 0; i < iterations; i++ {
					body := createCalculationRequest(op, data)
					testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
				}

				duration := time.Since(start)
				avgTime := duration / time.Duration(iterations)

				t.Logf("%s: %d operations in %v (avg: %v per operation)",
					op, iterations, duration, avgTime)

				// Performance target: < 10ms per operation
				if avgTime > 10*time.Millisecond {
					t.Logf("Warning: Average time %v exceeds 10ms target", avgTime)
				}
			})
		}
	})

	t.Run("StatisticalOperationsSpeed", func(t *testing.T) {
		// Test with increasing dataset sizes
		sizes := []int{100, 1000, 10000}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("DataSize_%d", size), func(t *testing.T) {
				data := make([]float64, size)
				for i := 0; i < size; i++ {
					data[i] = float64(i)
				}

				operations := []string{"mean", "median", "mode", "stddev", "variance"}
				for _, op := range operations {
					start := time.Now()
					body := CalculationRequest{
						Operation: op,
						Data:      data,
					}
					testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
					duration := time.Since(start)

					t.Logf("%s with %d points: %v", op, size, duration)

					// Performance target: < 500ms for 10K points
					if size == 10000 && duration > 500*time.Millisecond {
						t.Logf("Warning: %s with 10K points took %v (target: <500ms)", op, duration)
					}
				}
			})
		}
	})

	t.Run("MatrixOperationsSpeed", func(t *testing.T) {
		sizes := []int{10, 50, 100}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("MatrixSize_%dx%d", size, size), func(t *testing.T) {
				// Create square matrix
				matrix := make([][]float64, size)
				for i := 0; i < size; i++ {
					matrix[i] = make([]float64, size)
					for j := 0; j < size; j++ {
						matrix[i][j] = float64(i*size + j + 1)
					}
				}

				operations := []struct {
					name string
					body interface{}
				}{
					{
						name: "transpose",
						body: createMatrixRequest("matrix_transpose", matrix, nil, nil),
					},
					{
						name: "determinant",
						body: createMatrixRequest("matrix_determinant", matrix, nil, nil),
					},
				}

				for _, op := range operations {
					start := time.Now()
					testEndpoint(t, server, "POST", "/api/v1/math/calculate", op.body, testToken, http.StatusOK, nil)
					duration := time.Since(start)

					t.Logf("Matrix %s (%dx%d): %v", op.name, size, size, duration)

					// Performance target: < 100ms for 100x100 matrix
					if size == 100 && duration > 100*time.Millisecond {
						t.Logf("Warning: Matrix %s took %v (target: <100ms)", op.name, duration)
					}
				}
			})
		}
	})

	t.Run("OptimizationPerformance", func(t *testing.T) {
		iterations := []int{10, 50, 100, 500}

		for _, maxIter := range iterations {
			t.Run(fmt.Sprintf("MaxIterations_%d", maxIter), func(t *testing.T) {
				body := OptimizeRequest{
					ObjectiveFunction: "x^2 + y^2",
					Variables:         []string{"x", "y"},
					OptimizationType:  "minimize",
					Algorithm:         "gradient_descent",
				}
				body.Options.MaxIterations = maxIter
				body.Options.Tolerance = 1e-6

				start := time.Now()
				testEndpoint(t, server, "POST", "/api/v1/math/optimize", body, testToken, http.StatusOK, nil)
				duration := time.Since(start)

				t.Logf("Optimization with %d max iterations: %v", maxIter, duration)

				// Performance target: roughly linear with iterations
				perIterTime := duration / time.Duration(maxIter)
				if perIterTime > 1*time.Millisecond {
					t.Logf("Warning: %v per iteration (target: <1ms)", perIterTime)
				}
			})
		}
	})

	t.Run("ForecastingPerformance", func(t *testing.T) {
		dataSize := []int{10, 100, 1000}

		for _, size := range dataSize {
			t.Run(fmt.Sprintf("TimeSeries_%d_points", size), func(t *testing.T) {
				timeSeries := make([]interface{}, size)
				for i := 0; i < size; i++ {
					timeSeries[i] = float64(100 + i)
				}

				methods := []string{"linear_trend", "exponential_smoothing", "moving_average"}
				for _, method := range methods {
					body := ForecastRequest{
						TimeSeries:      timeSeries,
						ForecastHorizon: 10,
						Method:          method,
					}

					start := time.Now()
					testEndpoint(t, server, "POST", "/api/v1/math/forecast", body, testToken, http.StatusOK, nil)
					duration := time.Since(start)

					t.Logf("%s with %d points: %v", method, size, duration)

					// Performance target: < 1s for 1000 points
					if size == 1000 && duration > 1*time.Second {
						t.Logf("Warning: %s with 1000 points took %v (target: <1s)", method, duration)
					}
				}
			})
		}
	})
}

// TestConcurrentRequests tests handling of concurrent requests
func TestConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrency test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("ConcurrentCalculations", func(t *testing.T) {
		concurrency := 10
		requestsPerWorker := 20

		start := time.Now()
		errors := make(chan error, concurrency*requestsPerWorker)
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < requestsPerWorker; j++ {
					body := createCalculationRequest("add", []float64{float64(workerID), float64(j)})
					// testEndpoint handles success/failure internally
					testEndpoint(t, server, "POST", "/api/v1/math/calculate", body, testToken, http.StatusOK, nil)
				}
				done <- true
			}(i)
		}

		// Wait for all workers
		for i := 0; i < concurrency; i++ {
			<-done
		}
		duration := time.Since(start)

		close(errors)
		errorCount := len(errors)

		totalRequests := concurrency * requestsPerWorker
		t.Logf("Concurrent test: %d requests across %d workers in %v", totalRequests, concurrency, duration)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/duration.Seconds())

		if errorCount > 0 {
			t.Errorf("Had %d errors during concurrent execution", errorCount)
			for err := range errors {
				t.Logf("Error: %v", err)
			}
		}
	})

	t.Run("MixedOperationsConcurrent", func(t *testing.T) {
		concurrency := 5
		requestsPerWorker := 10

		operations := []func() (string, string, interface{}){
			func() (string, string, interface{}) {
				return "POST", "/api/v1/math/calculate", createCalculationRequest("add", []float64{1, 2, 3})
			},
			func() (string, string, interface{}) {
				return "POST", "/api/v1/math/statistics", createStatisticsRequest([]float64{1, 2, 3, 4, 5}, []string{"descriptive"})
			},
			func() (string, string, interface{}) {
				return "POST", "/api/v1/math/solve", createSolveRequest("x^2 - 4 = 0", []string{"x"}, "numerical")
			},
			func() (string, string, interface{}) {
				return "POST", "/api/v1/math/optimize", createOptimizeRequest("x^2 + y^2", []string{"x", "y"}, "minimize")
			},
			func() (string, string, interface{}) {
				return "POST", "/api/v1/math/forecast", createForecastRequest([]float64{100, 102, 104, 106}, 3, "linear_trend")
			},
		}

		start := time.Now()
		done := make(chan bool, concurrency)

		for i := 0; i < concurrency; i++ {
			go func(workerID int) {
				for j := 0; j < requestsPerWorker; j++ {
					opFunc := operations[j%len(operations)]
					method, path, body := opFunc()
					testEndpoint(t, server, method, path, body, testToken, http.StatusOK, nil)
				}
				done <- true
			}(i)
		}

		for i := 0; i < concurrency; i++ {
			<-done
		}
		duration := time.Since(start)

		totalRequests := concurrency * requestsPerWorker
		t.Logf("Mixed operations: %d requests in %v", totalRequests, duration)
		t.Logf("Throughput: %.2f requests/second", float64(totalRequests)/duration.Seconds())
	})
}

// TestMemoryUsage provides basic memory usage information
func TestMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(t, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	t.Run("LargeDatasetHandling", func(t *testing.T) {
		// Test with progressively larger datasets
		sizes := []int{1000, 10000, 100000}

		for _, size := range sizes {
			t.Run(fmt.Sprintf("Dataset_%d", size), func(t *testing.T) {
				data := make([]float64, size)
				for i := 0; i < size; i++ {
					data[i] = float64(i % 1000) // Repeating pattern
				}

				body := createStatisticsRequest(data, []string{"descriptive"})

				start := time.Now()
				testEndpoint(t, server, "POST", "/api/v1/math/statistics", body, testToken, http.StatusOK, func(resp map[string]interface{}) error {
					// Verify we got valid results
					if resp["results"] == nil {
						t.Error("Should have results for large dataset")
					}
					return nil
				})
				duration := time.Since(start)

				t.Logf("Dataset size %d: processed in %v", size, duration)

				// Memory efficiency target: should handle 100K points in < 5s
				if size == 100000 && duration > 5*time.Second {
					t.Logf("Warning: 100K points took %v (target: <5s)", duration)
				}
			})
		}
	})
}

// BenchmarkCalculations provides benchmark data
func BenchmarkCalculations(b *testing.B) {
	cleanup := setupTestLogger()
	defer cleanup()

	server, cleanup := setupTestServer(&testing.T{}, &TestServerConfig{APIToken: testToken})
	defer cleanup()

	b.Run("Addition", func(b *testing.B) {
		body := createCalculationRequest("add", []float64{5, 3, 2})
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := makeHTTPRequest(&testing.T{}, "POST", "/api/v1/math/calculate", body, testToken)
			executeRequest(server, req)
		}
	})

	b.Run("Mean", func(b *testing.B) {
		data := make([]float64, 1000)
		for i := 0; i < 1000; i++ {
			data[i] = float64(i)
		}
		body := CalculationRequest{Operation: "mean", Data: data}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := makeHTTPRequest(&testing.T{}, "POST", "/api/v1/math/calculate", body, testToken)
			executeRequest(server, req)
		}
	})

	b.Run("MatrixTranspose", func(b *testing.B) {
		matrix := make([][]float64, 50)
		for i := 0; i < 50; i++ {
			matrix[i] = make([]float64, 50)
			for j := 0; j < 50; j++ {
				matrix[i][j] = float64(i*50 + j)
			}
		}
		body := createMatrixRequest("matrix_transpose", matrix, nil, nil)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := makeHTTPRequest(&testing.T{}, "POST", "/api/v1/math/calculate", body, testToken)
			executeRequest(server, req)
		}
	})
}
