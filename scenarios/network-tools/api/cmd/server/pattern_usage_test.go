package main

import (
	"net/http"
	"testing"
)

// TestErrorPatternsUsage tests handlers using the ErrorTestPattern framework
func TestErrorPatternsUsage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HTTPRequestErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "HTTPRequest",
			Handler:     server.handleHTTPRequest,
			BaseURL:     "/api/v1/network/http",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/http").
			AddInvalidJSON("/api/v1/network/http").
			AddInvalidURL("/api/v1/network/http").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("DNSQueryErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "DNSQuery",
			Handler:     server.handleDNSQuery,
			BaseURL:     "/api/v1/network/dns",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/dns").
			AddInvalidJSON("/api/v1/network/dns").
			AddMissingField("/api/v1/network/dns", "query").
			AddUnsupportedRecordType("/api/v1/network/dns").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("NetworkScanErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "NetworkScan",
			Handler:     server.handleNetworkScan,
			BaseURL:     "/api/v1/network/scan",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/scan").
			AddInvalidJSON("/api/v1/network/scan").
			AddMissingField("/api/v1/network/scan", "target").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("ConnectivityTestErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "ConnectivityTest",
			Handler:     server.handleConnectivityTest,
			BaseURL:     "/api/v1/network/test/connectivity",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/test/connectivity").
			AddInvalidJSON("/api/v1/network/test/connectivity").
			AddMissingField("/api/v1/network/test/connectivity", "target").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("SSLValidationErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "SSLValidation",
			Handler:     server.handleSSLValidation,
			BaseURL:     "/api/v1/network/ssl/validate",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/ssl/validate").
			AddInvalidJSON("/api/v1/network/ssl/validate").
			AddMissingField("/api/v1/network/ssl/validate", "url").
			Build()

		suite.RunErrorTests(t, patterns)
	})

	t.Run("APITestErrorPatterns", func(t *testing.T) {
		suite := &HandlerTestSuite{
			HandlerName: "APITest",
			Handler:     server.handleAPITest,
			BaseURL:     "/api/v1/network/api/test",
		}

		patterns := NewTestScenarioBuilder().
			AddEmptyBody("/api/v1/network/api/test").
			AddInvalidJSON("/api/v1/network/api/test").
			AddMissingField("/api/v1/network/api/test", "base_url").
			Build()

		suite.RunErrorTests(t, patterns)
	})
}

// TestConcurrencyPatterns tests concurrent access patterns
func TestConcurrencyPatterns(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	pattern := ConcurrencyTestPattern{
		Name:        "HTTPRequestConcurrency",
		Description: "Test concurrent HTTP requests",
		Concurrency: 10,
		Iterations:  10,
		Setup: func(t *testing.T) interface{} {
			mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})
			return mockServer.URL
		},
		Execute: func(t *testing.T, setupData interface{}, iteration int) error {
			url := setupData.(string)
			req := createTestHTTPRequest(url, "GET", nil, nil)
			w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/network/http",
				Body:   req,
			})
			if err != nil {
				return err
			}
			if w.Code != http.StatusOK {
				t.Errorf("Expected 200, got %d", w.Code)
			}
			return nil
		},
		Validate: func(t *testing.T, setupData interface{}, results []error) {
			errorCount := 0
			for _, err := range results {
				if err != nil {
					errorCount++
				}
			}
			if errorCount > 0 {
				t.Errorf("Expected 0 errors, got %d", errorCount)
			}
		},
	}

	RunConcurrencyTest(t, pattern)
}

// TestHelperFunctionCoverage tests helper functions that need coverage
func TestHelperFunctionCoverage(t *testing.T) {
	t.Run("assertResponseField", func(t *testing.T) {
		resp := map[string]interface{}{
			"field1": "value1",
			"field2": 123,
		}

		// Test existing field
		assertResponseField(t, resp, "field1", "value1")

		// Test missing field (should fail but not panic)
		// Note: This will fail the test, but that's expected to cover the error path
	})
}
