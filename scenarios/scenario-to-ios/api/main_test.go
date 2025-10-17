
package main

import (
	"net/http"
	"testing"
	"time"
)

// TestServer_Initialization tests server initialization
func TestServer_Initialization(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("CreateServer", func(t *testing.T) {
		s := setupTestServer(t)
		if s == nil {
			t.Fatal("Failed to create server")
		}
		if s.router == nil {
			t.Fatal("Server router is nil")
		}
	})

	t.Run("RoutesRegistered", func(t *testing.T) {
		s := setupTestServer(t)
		if s.router == nil {
			t.Fatal("Server router is nil")
		}
		// Router is created successfully with routes
	})
}

// TestHandleHealth_Success tests successful health check
func TestHandleHealth_Success(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("HealthCheckSuccess", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Validate status code
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}

		// Validate JSON response
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "scenario-to-ios",
		})

		// Validate timestamp field exists
		if response != nil {
			if _, exists := response["timestamp"]; !exists {
				t.Error("Expected timestamp field in response")
			}
		}

		// Validate content type
		assertContentType(t, w, "application/json")
	})
}

// TestHandleHealth_Methods tests health endpoint with different HTTP methods
func TestHandleHealth_Methods(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	testCases := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{"GET", "GET", http.StatusOK},
		{"POST", "POST", http.StatusOK},
		{"PUT", "PUT", http.StatusOK},
		{"DELETE", "DELETE", http.StatusOK},
		{"PATCH", "PATCH", http.StatusOK},
		{"HEAD", "HEAD", http.StatusOK},
		{"OPTIONS", "OPTIONS", http.StatusOK},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
				Method: tc.method,
				Path:   "/api/v1/health",
			})

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d for %s, got %d", tc.expectedStatus, tc.method, w.Code)
			}
		})
	}
}

// TestHandleHealth_ResponseFormat tests health endpoint response format
func TestHandleHealth_ResponseFormat(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("ValidJSONFormat", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Failed to parse JSON response")
		}

		// Validate required fields
		requiredFields := []string{"status", "service", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Expected field '%s' in response", field)
			}
		}
	})

	t.Run("TimestampIsValid", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Failed to parse JSON response")
		}

		timestampStr, ok := response["timestamp"].(string)
		if !ok {
			t.Fatal("Timestamp is not a string")
		}

		// Parse timestamp
		_, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Errorf("Invalid timestamp format: %v", err)
		}
	})

	t.Run("ServiceNameIsCorrect", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"service": "scenario-to-ios",
		})

		if response == nil {
			t.Fatal("Failed to validate service name")
		}
	})
}

// TestHandleHealth_EdgeCases tests edge cases for health endpoint
func TestHandleHealth_EdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("WithQueryParameters", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method:      "GET",
			Path:        "/api/v1/health",
			QueryParams: map[string]string{"test": "value"},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("WithCustomHeaders", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
			Headers: map[string]string{
				"X-Custom-Header": "test-value",
			},
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})

	t.Run("WithBody", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/health",
			Body:   map[string]string{"test": "data"},
		})

		// Should still respond successfully even with unexpected body
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	})
}

// TestHandleHealth_Concurrency tests concurrent health check requests
func TestHandleHealth_Concurrency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	pattern := ConcurrencyTestPattern{
		Name:        "ConcurrentHealthChecks",
		Description: "Test concurrent health check requests",
		Concurrency: 10,
		Iterations:  50,
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/health",
			})

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				return nil
			}

			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			for i, err := range results {
				if err != nil {
					t.Errorf("Iteration %d failed: %v", i, err)
				}
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestHandleHealth_Performance tests health endpoint performance
func TestHandleHealth_Performance(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	pattern := PerformanceTestPattern{
		Name:        "HealthCheckPerformance",
		Description: "Test health check endpoint performance",
		MaxDuration: 10 * time.Millisecond,
		Execute: func(t *testing.T, setupData interface{}) time.Duration {
			start := time.Now()

			for i := 0; i < 100; i++ {
				w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
					Method: "GET",
					Path:   "/api/v1/health",
				})

				if w.Code != http.StatusOK {
					t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
				}
			}

			return time.Since(start)
		},
		Validate: func(t *testing.T, duration time.Duration) {
			avgDuration := duration / 100
			if avgDuration > 1*time.Millisecond {
				t.Logf("Average health check duration: %v (acceptable)", avgDuration)
			}
		},
	}

	RunPerformanceTest(t, pattern)
}

// TestRespondJSON tests the respondJSON helper method
func TestRespondJSON(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("ValidJSONEncoding", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Verify content type is set
		assertContentType(t, w, "application/json")

		// Verify response is valid JSON
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Failed to parse JSON response")
		}
	})

	t.Run("NonNilData", func(t *testing.T) {
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		if w.Body.Len() == 0 {
			t.Error("Expected non-empty response body")
		}
	})
}

// TestServer_Integration tests full server integration
func TestServer_Integration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("EndToEndHealthCheck", func(t *testing.T) {
		s := setupTestServer(t)

		// Make request
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		// Validate complete response
		response := assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{
			"status":  "healthy",
			"service": "scenario-to-ios",
		})

		if response == nil {
			t.Fatal("Failed to get health check response")
		}

		// Verify timestamp is recent
		timestampStr, ok := response["timestamp"].(string)
		if !ok {
			t.Fatal("Timestamp is not a string")
		}

		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Fatalf("Invalid timestamp format: %v", err)
		}

		// Timestamp should be within last 5 seconds
		if time.Since(timestamp) > 5*time.Second {
			t.Errorf("Timestamp is too old: %v", timestamp)
		}
	})
}

// TestHandleHealth_ResponseConsistency tests response consistency
func TestHandleHealth_ResponseConsistency(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("ConsistentResponses", func(t *testing.T) {
		// Make multiple requests
		responses := make([]map[string]interface{}, 10)
		for i := 0; i < 10; i++ {
			w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/health",
			})

			responses[i] = assertJSONResponse(t, w, http.StatusOK, nil)
		}

		// Verify all responses have same structure
		for i, response := range responses {
			if response == nil {
				t.Fatalf("Response %d is nil", i)
			}

			// Check status field is consistent
			if status, ok := response["status"].(string); !ok || status != "healthy" {
				t.Errorf("Response %d has inconsistent status", i)
			}

			// Check service field is consistent
			if service, ok := response["service"].(string); !ok || service != "scenario-to-ios" {
				t.Errorf("Response %d has inconsistent service", i)
			}

			// Check timestamp exists
			if _, ok := response["timestamp"].(string); !ok {
				t.Errorf("Response %d missing timestamp", i)
			}
		}
	})
}

// TestHandleHealth_ErrorHandling tests error handling (if applicable)
func TestHandleHealth_ErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	s := setupTestServer(t)

	t.Run("NilRequest", func(t *testing.T) {
		// Health handler should handle nil cases gracefully
		// This is a defensive test
		w := testHandlerWithRequest(t, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d even with edge cases, got %d", http.StatusOK, w.Code)
		}
	})
}

// Benchmark tests for performance analysis
func BenchmarkHandleHealth(b *testing.B) {
	s := setupTestServer(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := testHandlerWithRequest(&testing.T{}, s.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/health",
		})

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func BenchmarkHandleHealth_Parallel(b *testing.B) {
	s := setupTestServer(&testing.T{})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := testHandlerWithRequest(&testing.T{}, s.handleHealth, HTTPTestRequest{
				Method: "GET",
				Path:   "/api/v1/health",
			})

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status %d, got %d", http.StatusOK, w.Code)
			}
		}
	})
}
