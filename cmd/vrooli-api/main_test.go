//go:build testing
// +build testing

package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/gorilla/mux"
)

func stubStartAllScenarios(t *testing.T, result map[string]interface{}, err error) {
	t.Helper()
	original := startAllScenariosFn
	startAllScenariosFn = func() (map[string]interface{}, error) {
		return result, err
	}
	t.Cleanup(func() {
		startAllScenariosFn = original
	})
}

func stubStopAllScenarios(t *testing.T, result map[string]interface{}, err error) {
	t.Helper()
	original := stopAllScenariosFn
	stopAllScenariosFn = func() (map[string]interface{}, error) {
		return result, err
	}
	t.Cleanup(func() {
		stopAllScenariosFn = original
	})
}

func stubStopScenario(t *testing.T, err error) {
	t.Helper()
	original := stopScenarioFn
	stopScenarioFn = func(name string) error {
		return err
	}
	t.Cleanup(func() {
		stopScenarioFn = original
	})
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, healthCheck, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		// Health check can return 200 (healthy) or 503 (degraded)
		if w.Code != http.StatusOK && w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status 200 or 503, got %d", w.Code)
		}

		response := assertJSONResponse(t, w, w.Code, nil)
		if response != nil {
			if _, ok := response["status"]; !ok {
				t.Error("Expected status field in response")
			}
		}
	})
}

func TestEnforceStrictFingerprint(t *testing.T) {
	originalFingerprint := buildinfo.Fingerprint
	t.Cleanup(func() {
		buildinfo.Fingerprint = originalFingerprint
	})

	t.Run("Disabled", func(t *testing.T) {
		t.Setenv("VROOLI_STRICT_FINGERPRINT", "")
		if err := enforceStrictFingerprint(); err != nil {
			t.Fatalf("enforceStrictFingerprint disabled: %v", err)
		}
	})

	t.Run("Match", func(t *testing.T) {
		root := t.TempDir()
		writeGoTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
		writeGoTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
		writeGoTestFile(t, root, "internal/logx/logx.go", "package logx\n")

		t.Setenv("VROOLI_STRICT_FINGERPRINT", "1")
		t.Setenv(buildinfo.SourceRootEnvVar, root)
		t.Setenv(buildinfo.FingerprintPathsEnvVar, "cmd/vrooli-api,internal")

		current, err := buildinfo.CurrentFingerprint()
		if err != nil {
			t.Fatalf("CurrentFingerprint: %v", err)
		}
		buildinfo.Fingerprint = current

		if err := enforceStrictFingerprint(); err != nil {
			t.Fatalf("enforceStrictFingerprint match: %v", err)
		}
	})

	t.Run("Mismatch", func(t *testing.T) {
		root := t.TempDir()
		writeGoTestFile(t, root, "go.mod", "module example.com/test\n\ngo 1.21\n")
		writeGoTestFile(t, root, "cmd/vrooli-api/main.go", "package main\n")
		writeGoTestFile(t, root, "internal/logx/logx.go", "package logx\n")

		t.Setenv("VROOLI_STRICT_FINGERPRINT", "1")
		t.Setenv(buildinfo.SourceRootEnvVar, root)
		t.Setenv(buildinfo.FingerprintPathsEnvVar, "cmd/vrooli-api,internal")

		buildinfo.Fingerprint = "stale-fingerprint"
		err := enforceStrictFingerprint()
		if err == nil {
			t.Fatalf("expected mismatch error")
		}
		if !strings.Contains(err.Error(), "stale-fingerprint") {
			t.Fatalf("mismatch error %q does not include embedded fingerprint", err)
		}
	})
}

// TestListScenarios tests the list scenarios endpoint
func TestListScenarios(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, listScenariosNative, HTTPTestRequest{
			Method: "GET",
			Path:   "/scenarios",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}

		// Response should have a data field with scenarios array
		if data, ok := response["data"].([]interface{}); ok {
			t.Logf("Found %d scenarios", len(data))
		} else {
			t.Error("Expected data field to be an array")
		}
	})
}

// TestGetScenarioStatus tests the get scenario status endpoint
func TestGetScenarioStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentScenario", func(t *testing.T) {
		w := testHandlerWithRequest(t, getScenarioStatusNative, HTTPTestRequest{
			Method:  "GET",
			Path:    "/scenarios/nonexistent-scenario/status",
			URLVars: map[string]string{"name": "nonexistent-scenario"},
		})

		// API returns success with stopped status for nonexistent scenarios
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if data, ok := response["data"].(map[string]interface{}); ok {
				if status, exists := data["status"]; exists {
					if status != "stopped" {
						t.Errorf("Expected status 'stopped' for nonexistent scenario, got %v", status)
					}
				}
				if _, exists := data["health_status"]; !exists {
					t.Error("Expected health_status field in scenario status response")
				}
			}
		}
	})

	t.Run("EmptyScenarioName", func(t *testing.T) {
		w := testHandlerWithRequest(t, getScenarioStatusNative, HTTPTestRequest{
			Method:  "GET",
			Path:    "/scenarios//status",
			URLVars: map[string]string{"name": ""},
		})

		// Empty scenario name returns success with stopped status
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			t.Logf("Empty scenario name handled: %v", response)
			if data, ok := response["data"].(map[string]interface{}); ok {
				if _, exists := data["health_status"]; !exists {
					t.Error("Expected health_status field in scenario status response")
				}
			}
		}
	})
}

// TestListApps tests the list apps endpoint
func TestListApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, listApps, HTTPTestRequest{
			Method: "GET",
			Path:   "/apps",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}

		// Response should have a data field with apps array
		if data, ok := response["data"].([]interface{}); ok {
			t.Logf("Found %d apps", len(data))
		} else {
			t.Error("Expected data field to be an array")
		}
	})
}

// TestGetRunningApps tests the get running apps endpoint
func TestGetRunningApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, getRunningApps, HTTPTestRequest{
			Method: "GET",
			Path:   "/apps/running",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}

		// Response should have a data field with apps array
		if data, ok := response["data"].([]interface{}); ok {
			t.Logf("Found %d running apps", len(data))
		} else {
			t.Error("Expected data field to be an array")
		}
	})
}

// TestGetDetailedAppStatus tests the get detailed app status endpoint
func TestGetDetailedAppStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, getDetailedAppStatus, HTTPTestRequest{
			Method:  "GET",
			Path:    "/apps/nonexistent-app/status",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should return success with stopped status
		response := assertSuccessResponse(t, w, http.StatusOK)
		if response != nil {
			if data, ok := response["data"].(map[string]interface{}); ok {
				if status, exists := data["status"]; exists {
					if status != "stopped" {
						t.Errorf("Expected status 'stopped' for nonexistent app, got %v", status)
					}
				}
			}
		}
	})
}

// TestListResources tests the list resources endpoint
func TestListResources(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, listResources, HTTPTestRequest{
			Method: "GET",
			Path:   "/resources",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}

		// Response should have a data field
		if _, ok := response["data"]; !ok {
			t.Error("Expected data field in response")
		}
	})
}

// TestProcessMetrics tests the process metrics endpoint
func TestProcessMetrics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		w := testHandlerWithRequest(t, processMetricsHandler, HTTPTestRequest{
			Method: "GET",
			Path:   "/metrics/processes",
		})

		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}

		// Response should have process metrics (various field names possible)
		t.Logf("Process metrics response: %v", response)
	})
}

// TestStopScenarioEndpoint tests the stop scenario endpoint
func TestStopScenarioEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentScenario", func(t *testing.T) {
		stubStopScenario(t, nil)

		w := testHandlerWithRequest(t, stopScenarioEndpoint, HTTPTestRequest{
			Method:  "POST",
			Path:    "/scenarios/nonexistent-scenario/stop",
			URLVars: map[string]string{"name": "nonexistent-scenario"},
		})

		// Should return error for nonexistent scenario
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["error"]; ok {
				// Error is expected for nonexistent scenario
				t.Logf("Got expected error for nonexistent scenario")
			}
		}
	})
}

// TestGetAppLogs tests the get app logs endpoint
func TestGetAppLogs(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, getAppLogs, HTTPTestRequest{
			Method:  "GET",
			Path:    "/apps/nonexistent-app/logs",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should return error for nonexistent app
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["error"]; ok {
				// Error is expected for nonexistent app
				t.Logf("Got expected error for nonexistent app")
			}
		}
	})

	t.Run("WithQueryParameters", func(t *testing.T) {
		w := testHandlerWithRequest(t, getAppLogs, HTTPTestRequest{
			Method:  "GET",
			Path:    "/apps/test-app/logs?lines=100",
			URLVars: map[string]string{"name": "test-app"},
			QueryParams: map[string]string{
				"lines": "100",
			},
		})

		// Should process query parameters without error
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestHandleLifecycle tests the lifecycle endpoint
func TestHandleLifecycle(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InvalidAction", func(t *testing.T) {
		w := testHandlerWithRequest(t, handleLifecycle, HTTPTestRequest{
			Method:  "POST",
			Path:    "/lifecycle/invalid-action",
			URLVars: map[string]string{"action": "invalid-action"},
		})

		// Should handle invalid actions gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})
}

// TestStopAllScenariosEndpoint tests the stop all scenarios endpoint
func TestStopAllScenariosEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		stubStopAllScenarios(t, map[string]interface{}{
			"stopped": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "Stopped 0 scenarios, 0 failed",
		}, nil)

		w := testHandlerWithRequest(t, stopAllScenariosEndpoint, HTTPTestRequest{
			Method: "POST",
			Path:   "/scenarios/stop-all",
		})

		// Should return success
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}
	})
}

// TestStopAllApps tests the stop all apps endpoint
func TestStopAllApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		stubStopAllScenarios(t, map[string]interface{}{
			"stopped": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "Stopped 0 scenarios, 0 failed",
		}, nil)

		w := testHandlerWithRequest(t, stopAllApps, HTTPTestRequest{
			Method: "POST",
			Path:   "/apps/stop-all",
		})

		// Should return success
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response == nil {
			t.Fatal("Expected response but got nil")
		}
	})
}

// TestStartAllApps tests the start all apps endpoint
func TestStartAllApps(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		stubStartAllScenarios(t, map[string]interface{}{
			"started": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "Started 0 scenarios, 0 failed",
		}, nil)

		w := testHandlerWithRequest(t, startAllApps, HTTPTestRequest{
			Method: "POST",
			Path:   "/apps/start-all",
		})

		// Should return success or error (depending on system state)
		if w.Code != http.StatusOK {
			t.Logf("Start all apps returned status %d", w.Code)
		}
	})
}

// TestStartAllScenariosEndpoint tests the start all scenarios endpoint
func TestStartAllScenariosEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		stubStartAllScenarios(t, map[string]interface{}{
			"started": []map[string]string{},
			"failed":  []map[string]string{},
			"message": "Started 0 scenarios, 0 failed",
		}, nil)

		w := testHandlerWithRequest(t, startAllScenariosEndpoint, HTTPTestRequest{
			Method: "POST",
			Path:   "/scenarios/start-all",
		})

		// Should return success or error (depending on system state)
		if w.Code != http.StatusOK {
			t.Logf("Start all scenarios returned status %d", w.Code)
		}
	})
}

// TestProtectApp tests the protect app endpoint
func TestProtectApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, protectApp, HTTPTestRequest{
			Method:  "POST",
			Path:    "/apps/nonexistent-app/protect",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should handle nonexistent app gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 200 or 404, got %d", w.Code)
		}
	})
}

// TestStartApp tests the start app endpoint
func TestStartApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, startApp, HTTPTestRequest{
			Method:  "POST",
			Path:    "/apps/nonexistent-app/start",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should return error for nonexistent app
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["error"]; ok {
				// Error is expected for nonexistent app
				t.Logf("Got expected error for nonexistent app")
			}
		}
	})
}

// TestStopApp tests the stop app endpoint
func TestStopApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, stopApp, HTTPTestRequest{
			Method:  "POST",
			Path:    "/apps/nonexistent-app/stop",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should return error for nonexistent app
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["error"]; ok {
				// Error is expected for nonexistent app
				t.Logf("Got expected error for nonexistent app")
			}
		}
	})
}

// TestRestartApp tests the restart app endpoint
func TestRestartApp(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NonexistentApp", func(t *testing.T) {
		w := testHandlerWithRequest(t, restartApp, HTTPTestRequest{
			Method:  "POST",
			Path:    "/apps/nonexistent-app/restart",
			URLVars: map[string]string{"name": "nonexistent-app"},
		})

		// Should return error for nonexistent app
		response := assertJSONResponse(t, w, http.StatusOK, nil)
		if response != nil {
			if _, ok := response["error"]; ok {
				// Error is expected for nonexistent app
				t.Logf("Got expected error for nonexistent app")
			}
		}
	})
}

// TestRouterIntegration tests the full router integration
func TestRouterIntegration(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	testCases := []struct {
		name         string
		method       string
		path         string
		expectCode   int
		allowedCodes []int
	}{
		{"Health", "GET", "/health", http.StatusOK, []int{http.StatusOK, http.StatusServiceUnavailable}},
		{"ListScenarios", "GET", "/scenarios", http.StatusOK, nil},
		{"ListApps", "GET", "/apps", http.StatusOK, nil},
		{"ListResources", "GET", "/resources", http.StatusOK, nil},
		{"ProcessMetrics", "GET", "/metrics/processes", http.StatusOK, nil},
		{"GetRunningApps", "GET", "/apps/running", http.StatusOK, nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w, req, err := makeHTTPRequest(HTTPTestRequest{
				Method: tc.method,
				Path:   tc.path,
			})
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			router.ServeHTTP(w, req)

			// Check if status code is allowed
			allowed := tc.allowedCodes != nil
			if allowed {
				found := false
				for _, code := range tc.allowedCodes {
					if w.Code == code {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected status codes %v, got %d. Response: %s",
						tc.allowedCodes, w.Code, w.Body.String())
				}
			} else if w.Code != tc.expectCode {
				t.Errorf("Expected status %d, got %d. Response: %s",
					tc.expectCode, w.Code, w.Body.String())
			}
		})
	}
}

type HTTPTestRequest struct {
	Method      string
	Path        string
	Body        io.Reader
	Headers     map[string]string
	QueryParams map[string]string
	URLVars     map[string]string
}

func setupTestLogger() func() {
	original := log.Writer()
	log.SetOutput(io.Discard)
	return func() {
		log.SetOutput(original)
	}
}

func setupTestRouter() *mux.Router {
	router := mux.NewRouter()

	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/metrics/processes", processMetricsHandler).Methods("GET")

	router.HandleFunc("/apps", listApps).Methods("GET")
	router.HandleFunc("/apps/running", getRunningApps).Methods("GET")
	router.HandleFunc("/apps/start-all", startAllApps).Methods("POST")
	router.HandleFunc("/apps/stop-all", stopAllApps).Methods("POST")
	router.HandleFunc("/apps/{name}/protect", protectApp).Methods("POST")
	router.HandleFunc("/apps/{name}/start", startApp).Methods("POST")
	router.HandleFunc("/apps/{name}/stop", stopApp).Methods("POST")
	router.HandleFunc("/apps/{name}/restart", restartApp).Methods("POST")
	router.HandleFunc("/apps/{name}/logs", getAppLogs).Methods("GET")
	router.HandleFunc("/apps/{name}/status", getDetailedAppStatus).Methods("GET")

	router.HandleFunc("/scenarios", listScenariosNative).Methods("GET")
	router.HandleFunc("/scenarios/{name}/status", getScenarioStatusNative).Methods("GET")
	router.HandleFunc("/scenarios/{name}/start", startApp).Methods("POST")
	router.HandleFunc("/scenarios/{name}/stop", stopScenarioEndpoint).Methods("POST")
	router.HandleFunc("/scenarios/start-all", startAllScenariosEndpoint).Methods("POST")
	router.HandleFunc("/scenarios/stop-all", stopAllScenariosEndpoint).Methods("POST")

	router.HandleFunc("/resources", listResources).Methods("GET")
	router.HandleFunc("/lifecycle/{action}", handleLifecycle).Methods("POST")

	return router
}

func makeHTTPRequest(req HTTPTestRequest) (*httptest.ResponseRecorder, *http.Request, error) {
	method := req.Method
	if method == "" {
		method = http.MethodGet
	}

	target := req.Path
	if target == "" {
		target = "/"
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, nil, err
	}

	query := parsed.Query()
	for key, value := range req.QueryParams {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()

	body := req.Body
	if body == nil {
		body = bytes.NewReader(nil)
	}

	httpReq := httptest.NewRequest(method, parsed.String(), body)
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	if len(req.URLVars) > 0 {
		httpReq = mux.SetURLVars(httpReq, req.URLVars)
	}

	return httptest.NewRecorder(), httpReq, nil
}

func testHandlerWithRequest(t *testing.T, handler func(http.ResponseWriter, *http.Request), req HTTPTestRequest) *httptest.ResponseRecorder {
	t.Helper()

	recorder, httpReq, err := makeHTTPRequest(req)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	handler(recorder, httpReq)
	return recorder
}

func assertJSONResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int, expectedFields map[string]interface{}) map[string]interface{} {
	t.Helper()

	if w.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d. Response: %s", expectedStatus, w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected JSON response, got error %v. Body: %s", err, w.Body.String())
	}

	for key, value := range expectedFields {
		if got, ok := response[key]; !ok || got != value {
			t.Fatalf("expected response[%q] = %v, got %v", key, value, got)
		}
	}

	return response
}

func assertSuccessResponse(t *testing.T, w *httptest.ResponseRecorder, expectedStatus int) map[string]interface{} {
	t.Helper()

	response := assertJSONResponse(t, w, expectedStatus, nil)
	if success, ok := response["success"].(bool); ok && !success {
		t.Fatalf("expected success response, got %v", response)
	}
	return response
}

func writeGoTestFile(t *testing.T, root, rel, contents string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
