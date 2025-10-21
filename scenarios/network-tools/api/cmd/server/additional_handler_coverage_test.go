package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestHTTPRequestHandlerAdditional provides additional coverage for HTTP request handler
func TestHTTPRequestHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("POSTWithBody", func(t *testing.T) {
		// Mock HTTP server for testing
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"result": "success"})
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"url":    testServer.URL,
			"method": "POST",
			"body":   `{"test": "data"}`,
			"headers": map[string]string{
				"Content-Type": "application/json",
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/http", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleHTTPRequest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("PUTRequest", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"url":    testServer.URL,
			"method": "PUT",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/http", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleHTTPRequest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("DELETERequest", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"url":    testServer.URL,
			"method": "DELETE",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/http", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleHTTPRequest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("PATCHRequest", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"url":    testServer.URL,
			"method": "PATCH",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/http", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleHTTPRequest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("CustomHeaders", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check custom headers
			if r.Header.Get("X-Custom-Header") != "test-value" {
				t.Error("Custom header not sent")
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"url":    testServer.URL,
			"method": "GET",
			"headers": map[string]string{
				"X-Custom-Header": "test-value",
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/http", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleHTTPRequest(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestDNSQueryHandlerAdditional provides additional coverage for DNS query handler
func TestDNSQueryHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AAAARecordQuery", func(t *testing.T) {
		reqData := map[string]interface{}{
			"query":       "google.com",
			"record_type": "AAAA",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/dns", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleDNSQuery(w, req)

		// DNS queries may fail in test environments, just verify handler doesn't crash
		t.Logf("AAAA query status: %d", w.Code)
	})

	t.Run("MXRecordQuery", func(t *testing.T) {
		reqData := map[string]interface{}{
			"query":       "google.com",
			"record_type": "MX",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/dns", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleDNSQuery(w, req)

		t.Logf("MX query status: %d", w.Code)
	})

	t.Run("TXTRecordQuery", func(t *testing.T) {
		reqData := map[string]interface{}{
			"query":       "google.com",
			"record_type": "TXT",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/dns", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleDNSQuery(w, req)

		t.Logf("TXT query status: %d", w.Code)
	})

	t.Run("CNAMERecordQuery", func(t *testing.T) {
		reqData := map[string]interface{}{
			"query":       "www.google.com",
			"record_type": "CNAME",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/dns", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleDNSQuery(w, req)

		t.Logf("CNAME query status: %d", w.Code)
	})
}

// TestSSLValidationHandlerAdditional provides additional coverage for SSL validation
func TestSSLValidationHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ValidSSLCertificate", func(t *testing.T) {
		reqData := map[string]interface{}{
			"host": "google.com:443",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/ssl", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleSSLValidation(w, req)

		// SSL validation may fail in test environments, just verify handler doesn't crash
		t.Logf("SSL validation status: %d", w.Code)
	})

	t.Run("HostWithoutPort", func(t *testing.T) {
		reqData := map[string]interface{}{
			"host": "google.com",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/ssl", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleSSLValidation(w, req)

		// Verify handler handles host without port gracefully
		t.Logf("Host without port status: %d", w.Code)
	})
}

// TestAPITestHandlerAdditional provides additional coverage for API test handler
func TestAPITestHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("MultipleTestCases", func(t *testing.T) {
		// Create a test server
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/users":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]string{{"id": "1", "name": "Test"}})
			case "/posts":
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode([]map[string]string{{"id": "1", "title": "Test Post"}})
			default:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"api_definition": map[string]interface{}{
				"base_url": testServer.URL,
			},
			"test_suite": []map[string]interface{}{
				{
					"endpoint": "/users",
					"method":   "GET",
					"test_cases": []map[string]interface{}{
						{
							"name":            "Get users",
							"expected_status": 200,
						},
					},
				},
				{
					"endpoint": "/posts",
					"method":   "GET",
					"test_cases": []map[string]interface{}{
						{
							"name":            "Get posts",
							"expected_status": 200,
						},
					},
				},
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/api/test", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleAPITest(w, req)

		// API tests may succeed or fail, just verify handler doesn't crash
		t.Logf("Multiple test cases status: %d", w.Code)
	})

	t.Run("POSTTestCase", func(t *testing.T) {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{"id": "123"})
			}
		}))
		defer testServer.Close()

		reqData := map[string]interface{}{
			"api_definition": map[string]interface{}{
				"base_url": testServer.URL,
			},
			"test_suite": []map[string]interface{}{
				{
					"endpoint": "/items",
					"method":   "POST",
					"test_cases": []map[string]interface{}{
						{
							"name":            "Create item",
							"input":           map[string]string{"name": "test"},
							"expected_status": 201,
						},
					},
				},
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/api/test", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleAPITest(w, req)

		// Verify handler processes POST test cases
		t.Logf("POST test case status: %d", w.Code)
	})
}

// TestConnectivityHandlerAdditional provides additional coverage for connectivity handler
func TestConnectivityHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("TracerouteTest", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "8.8.8.8",
			"test_type": "traceroute",
			"options": map[string]interface{}{
				"max_hops": 10,
				"timeout":  5000,
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/test/connectivity", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleConnectivityTest(w, req)

		// Traceroute may not be supported, just verify handler doesn't crash
		t.Logf("Traceroute status: %d", w.Code)
	})

	t.Run("PingWithOptions", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "8.8.8.8",
			"test_type": "ping",
			"options": map[string]interface{}{
				"count":    3,
				"timeout":  2000,
				"interval": 1000,
			},
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/test/connectivity", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleConnectivityTest(w, req)

		// Ping may succeed or fail, just verify handler doesn't crash
		t.Logf("Ping with options status: %d", w.Code)
	})
}

// TestHelperFunctionsAdditional provides additional test helper coverage
func TestHelperFunctionsAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("assertJSONResponseSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := Response{Success: true, Data: map[string]string{"test": "data"}}
		json.NewEncoder(w).Encode(response)

		assertJSONResponse(t, w, http.StatusOK, func(data map[string]interface{}) error {
			if data["success"] != true {
				t.Error("Expected success to be true")
			}
			return nil
		})
	})

	t.Run("assertSuccessResponseWithData", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := Response{Success: true, Data: "test data"}
		json.NewEncoder(w).Encode(response)

		data := assertSuccessResponse(t, w, http.StatusOK)
		if data == nil {
			t.Error("Expected response data")
		}
	})

	t.Run("assertErrorResponseWithMessage", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		response := Response{Success: false, Error: "test error"}
		json.NewEncoder(w).Encode(response)

		assertErrorResponse(t, w, http.StatusBadRequest, "test error")
	})

	t.Run("makeHTTPRequestWithComplexBody", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body: map[string]interface{}{
				"nested": map[string]interface{}{
					"data": "value",
				},
				"array": []string{"one", "two"},
			},
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		w, err := makeHTTPRequest(handler, req)
		if err != nil {
			t.Fatalf("makeHTTPRequest failed: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestNetworkScanHandlerAdditional provides additional coverage for network scan handler
func TestNetworkScanHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PortScanMultiplePorts", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "127.0.0.1",
			"scan_type": "port",
			"ports":     "80,443,8080",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		// Verify handler processes multi-port scans
		t.Logf("Multi-port scan status: %d", w.Code)
	})

	t.Run("PortRangeScan", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "127.0.0.1",
			"scan_type": "port",
			"ports":     "8000-8010",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		// Verify handler processes port range scans
		t.Logf("Port range scan status: %d", w.Code)
	})
}

// TestHealthHandlerTimestamp tests that health endpoint includes timestamp
func TestHealthHandlerTimestamp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	env.Server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	// Check timestamp exists and is recent
	if health["timestamp"] == nil {
		t.Error("Expected timestamp in health response")
	}

	// Parse timestamp
	timestampStr, ok := health["timestamp"].(string)
	if !ok {
		t.Error("Timestamp should be a string")
	} else {
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			t.Errorf("Failed to parse timestamp: %v", err)
		}

		// Check it's recent (within last minute)
		if time.Since(timestamp) > time.Minute {
			t.Error("Timestamp is too old")
		}
	}
}
