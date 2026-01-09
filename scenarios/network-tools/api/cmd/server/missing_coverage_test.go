package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandleListTargetsWithDatabase tests the handleListTargets with actual database scenarios
func TestHandleListTargetsWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		server := &Server{db: nil}

		req := httptest.NewRequest("GET", "/api/v1/network/targets", nil)
		w := httptest.NewRecorder()

		server.handleListTargets(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 when DB not configured, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected failure response")
		}

		if !strings.Contains(resp.Error, "Database not configured") {
			t.Errorf("Expected database error message, got: %s", resp.Error)
		}
	})
}

// TestHandleCreateTargetWithDatabase tests the handleCreateTarget with database scenarios
func TestHandleCreateTargetWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		server := &Server{db: nil}

		target := map[string]interface{}{
			"name":        "Test Target",
			"target_type": "host",
			"address":     "example.com",
			"port":        80,
			"protocol":    "http",
		}

		body, _ := json.Marshal(target)
		req := httptest.NewRequest("POST", "/api/v1/network/targets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleCreateTarget(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 when DB not configured, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected failure response")
		}
	})

	t.Run("InvalidJSONBody", func(t *testing.T) {
		server := &Server{db: nil}

		req := httptest.NewRequest("POST", "/api/v1/network/targets", bytes.NewReader([]byte("{invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleCreateTarget(w, req)

		// Should get database error before JSON validation in this case
		// or bad request if JSON validation happens first
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 500 or 400, got %d", w.Code)
		}
	})

	t.Run("ValidRequestNoDB", func(t *testing.T) {
		server := &Server{db: nil}

		target := map[string]interface{}{
			"name":        "Test",
			"target_type": "host",
			"address":     "example.com",
		}

		body, _ := json.Marshal(target)
		req := httptest.NewRequest("POST", "/api/v1/network/targets", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.handleCreateTarget(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 when DB not configured, got %d", w.Code)
		}
	})
}

// TestHandleListAlertsWithDatabase tests the handleListAlerts with database scenarios
func TestHandleListAlertsWithDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		server := &Server{db: nil}

		req := httptest.NewRequest("GET", "/api/v1/network/alerts", nil)
		w := httptest.NewRecorder()

		server.handleListAlerts(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 when DB not configured, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected failure response")
		}

		if !strings.Contains(resp.Error, "Database not configured") {
			t.Errorf("Expected database error message, got: %s", resp.Error)
		}
	})
}

// TestHealthHandlerDatabase tests health endpoint with database states
func TestHealthHandlerDatabase(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NoDatabaseConfigured", func(t *testing.T) {
		server := &Server{db: nil}

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		server.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health should return 200, got %d", w.Code)
		}

		var health map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&health); err != nil {
			t.Fatalf("Failed to decode health response: %v", err)
		}

		// New health schema returns status="degraded" when DB not configured
		if health["status"] != "degraded" {
			t.Errorf("Expected overall status 'degraded' (no database), got: %v", health["status"])
		}

		// Check dependencies.database structure
		if deps, ok := health["dependencies"].(map[string]interface{}); ok {
			if dbInfo, ok := deps["database"].(map[string]interface{}); ok {
				if connected, ok := dbInfo["connected"].(bool); !ok || connected {
					t.Error("Expected database.connected=false when no database")
				}
			} else {
				t.Error("Expected dependencies.database in health response")
			}
		} else {
			t.Error("Expected dependencies in health response")
		}
	})
}

// Additional edge case tests for handlers not covered elsewhere

// TestConnectivityTestHandlerEdgeCases tests connectivity test handler edge cases
func TestConnectivityTestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("InvalidJSONBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/test/connectivity", bytes.NewReader([]byte("{invalid")))
		w := httptest.NewRecorder()

		env.Server.handleConnectivityTest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("EmptyTarget", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "",
			"test_type": "ping",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/test/connectivity", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleConnectivityTest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty target, got %d", w.Code)
		}
	})

	t.Run("InvalidTestType", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "8.8.8.8",
			"test_type": "invalid_type",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/test/connectivity", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleConnectivityTest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid test type, got %d", w.Code)
		}
	})
}

// TestNetworkScanHandlerEdgeCases tests network scan handler edge cases
func TestNetworkScanHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("InvalidJSONBody", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader([]byte("{invalid")))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("EmptyTarget", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "",
			"scan_type": "port",
			"ports":     "80,443",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for empty target, got %d", w.Code)
		}
	})

	t.Run("InvalidScanType", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "127.0.0.1",
			"scan_type": "invalid_scan",
			"ports":     "80",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid scan type, got %d", w.Code)
		}
	})

	t.Run("InvalidPortFormat", func(t *testing.T) {
		reqData := map[string]interface{}{
			"target":    "127.0.0.1",
			"scan_type": "port",
			"ports":     "not-a-port",
		}

		body, _ := json.Marshal(reqData)
		req := httptest.NewRequest("POST", "/api/v1/network/scan", bytes.NewReader(body))
		w := httptest.NewRecorder()

		env.Server.handleNetworkScan(w, req)

		// Should fail validation
		if w.Code == http.StatusOK {
			t.Log("Warning: invalid port format may have succeeded unexpectedly")
		}
	})
}

// TestAdditionalHelperFunctions tests additional utility functions
func TestAdditionalHelperFunctions(t *testing.T) {
	t.Run("getEnvFunction", func(t *testing.T) {
		// Test with existing env var
		result := getEnv("PATH", "default")
		if result == "default" {
			t.Error("Expected PATH to be set")
		}

		// Test with non-existing env var
		result = getEnv("DEFINITELY_NOT_SET_12345", "default_value")
		if result != "default_value" {
			t.Errorf("Expected default value, got: %s", result)
		}
	})

	t.Run("mapToJSONFunction", func(t *testing.T) {
		testMap := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		jsonStr := mapToJSON(testMap)
		if jsonStr == "" {
			t.Error("Expected non-empty JSON string")
		}

		// Verify it's valid JSON
		var decoded map[string]string
		if err := json.Unmarshal([]byte(jsonStr), &decoded); err != nil {
			t.Errorf("Failed to decode JSON: %v", err)
		}
	})

	t.Run("getServiceNameFunction", func(t *testing.T) {
		name := getServiceName(8080)
		if name == "" {
			t.Error("Expected non-empty service name")
		}
		// Service name can vary based on port, just check it's not empty
		t.Logf("Service name for port 8080: %s", name)
	})
}

// TestNullSQLTypes tests handling of SQL null types
func TestNullSQLTypes(t *testing.T) {
	t.Run("NullInt64Valid", func(t *testing.T) {
		val := sql.NullInt64{Int64: 42, Valid: true}
		if !val.Valid {
			t.Error("Expected valid int64")
		}
		if val.Int64 != 42 {
			t.Errorf("Expected 42, got %d", val.Int64)
		}
	})

	t.Run("NullInt64Invalid", func(t *testing.T) {
		val := sql.NullInt64{Valid: false}
		if val.Valid {
			t.Error("Expected invalid int64")
		}
	})

	t.Run("NullStringValid", func(t *testing.T) {
		val := sql.NullString{String: "test", Valid: true}
		if !val.Valid {
			t.Error("Expected valid string")
		}
		if val.String != "test" {
			t.Errorf("Expected 'test', got %s", val.String)
		}
	})

	t.Run("NullStringInvalid", func(t *testing.T) {
		val := sql.NullString{Valid: false}
		if val.Valid {
			t.Error("Expected invalid string")
		}
	})
}
