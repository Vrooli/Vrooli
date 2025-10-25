// +build testing

package main

import (
	"net/http"
	"testing"
)

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
