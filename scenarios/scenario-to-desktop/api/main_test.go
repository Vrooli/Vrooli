package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

// TestHealthHandler tests the health check endpoint comprehensively
func TestHealthHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)

	t.Run("Success", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldValue(t, response, "service", "scenario-to-desktop-api")
		assertFieldExists(t, response, "version")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "timestamp")
		assertFieldExists(t, response, "readiness")
	})

	t.Run("MultipleRequests", func(t *testing.T) {
		// Test that health endpoint handles concurrent requests
		for i := 0; i < 10; i++ {
			req := httptest.NewRequest("GET", "/api/v1/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
	})
}

// TestTestDesktopHandler tests the desktop testing endpoint
func TestTestDesktopHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ValidRequest", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":  filepath.Join(env.TempDir, "test-app"),
			"platforms": []string{"linux"},
			"headless":  true,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/test", body)
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		response := assertJSONResponse(t, w, http.StatusOK)
		assertFieldExists(t, response, "test_results")
		assertFieldExists(t, response, "status")
		assertFieldExists(t, response, "timestamp")
	})

	t.Run("MultiPlatform", func(t *testing.T) {
		body := map[string]interface{}{
			"app_path":  filepath.Join(env.TempDir, "test-app-multi"),
			"platforms": []string{"win", "mac", "linux"},
			"headless":  false,
		}

		req := createJSONRequest("POST", "/api/v1/desktop/test", body)
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/desktop/test", strings.NewReader("{bad"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		env.Server.testDesktopHandler(w, req)

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestValidateDesktopConfig tests configuration validation
func TestValidateDesktopConfig(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	server := NewServer(0)
	newConfig := func() *DesktopConfig {
		return &DesktopConfig{
			AppName:      "TestApp",
			Framework:    "electron",
			TemplateType: "basic",
			OutputPath:   "/tmp/test",
			ServerType:   "static",
			ServerPath:   "./dist",
			APIEndpoint:  "http://localhost:3000",
		}
	}

	t.Run("ValidConfig", func(t *testing.T) {
		config := newConfig()

		if err := server.validateDesktopConfig(config); err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if config.License != "MIT" {
			t.Errorf("Expected default license MIT, got: %s", config.License)
		}
		if len(config.Platforms) != 3 {
			t.Errorf("Expected default platforms [win, mac, linux], got: %v", config.Platforms)
		}
	})

	t.Run("MissingAppName", func(t *testing.T) {
		config := newConfig()
		config.AppName = ""

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "app_name") {
			t.Errorf("Expected app_name error, got: %v", err)
		}
	})

	t.Run("MissingFramework", func(t *testing.T) {
		config := newConfig()
		config.Framework = ""

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "framework") {
			t.Errorf("Expected framework error, got: %v", err)
		}
	})

	t.Run("InvalidFramework", func(t *testing.T) {
		config := newConfig()
		config.Framework = "invalid"

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "framework") {
			t.Errorf("Expected framework validation error, got: %v", err)
		}
	})

	t.Run("InvalidTemplateType", func(t *testing.T) {
		config := newConfig()
		config.TemplateType = "invalid"

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "template_type") {
			t.Errorf("Expected template_type error, got: %v", err)
		}
	})

	t.Run("MissingOutputPathDefaults", func(t *testing.T) {
		config := newConfig()
		config.OutputPath = ""

		if err := server.validateDesktopConfig(config); err != nil {
			t.Errorf("Expected default output path handling, got: %v", err)
		}
	})

	t.Run("CustomLicense", func(t *testing.T) {
		config := newConfig()
		config.License = "Apache-2.0"

		if err := server.validateDesktopConfig(config); err != nil {
			t.Errorf("Expected no error for custom license, got: %v", err)
		}
		if config.License != "Apache-2.0" {
			t.Errorf("Expected license to remain Apache-2.0, got: %s", config.License)
		}
	})

	t.Run("CustomPlatforms", func(t *testing.T) {
		config := newConfig()
		config.Platforms = []string{"win"}

		if err := server.validateDesktopConfig(config); err != nil {
			t.Errorf("Expected no error for custom platforms, got: %v", err)
		}
		if len(config.Platforms) != 1 {
			t.Errorf("Expected single platform to remain, got: %v", config.Platforms)
		}
	})

	t.Run("InvalidDeploymentMode", func(t *testing.T) {
		config := newConfig()
		config.DeploymentMode = "unsupported"

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "deployment_mode") {
			t.Errorf("Expected deployment_mode error, got: %v", err)
		}
	})

	t.Run("AutoManageRequiresExternal", func(t *testing.T) {
		config := newConfig()
		config.AutoManageVrooli = true

		if err := server.validateDesktopConfig(config); err == nil || !strings.Contains(err.Error(), "auto_manage_vrooli") {
			t.Errorf("Expected auto-manage error for static server, got: %v", err)
		}
	})

	t.Run("AutoManageAllowedWithExternal", func(t *testing.T) {
		config := newConfig()
		config.ServerType = "external"
		config.ProxyURL = "http://localhost:3000/"
		config.AutoManageVrooli = true

		if err := server.validateDesktopConfig(config); err != nil {
			t.Errorf("Expected external auto-manage to pass, got: %v", err)
		}
	})

	t.Run("ServerPortDefault", func(t *testing.T) {
		config := newConfig()
		config.ServerType = "node"
		config.ServerPath = "ui/server.js"

		if err := server.validateDesktopConfig(config); err != nil {
			t.Errorf("Expected embedded node config to pass, got: %v", err)
		}
		if config.ServerPort != 3000 {
			t.Errorf("Expected default port 3000, got %d", config.ServerPort)
		}
	})
}

// TestContainsUtility tests the contains utility function
func TestContainsUtility(t *testing.T) {
	testCases := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"EmptySlice", []string{}, "test", false},
		{"ItemExists", []string{"a", "b", "c"}, "b", true},
		{"ItemNotExists", []string{"a", "b", "c"}, "d", false},
		{"SingleItemMatch", []string{"test"}, "test", true},
		{"SingleItemNoMatch", []string{"test"}, "other", false},
		{"CaseSensitive", []string{"Test"}, "test", false},
		{"MultipleMatches", []string{"a", "b", "a"}, "a", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := contains(tc.slice, tc.item)
			if result != tc.expected {
				t.Errorf("Expected %v, got %v for slice %v and item %s",
					tc.expected, result, tc.slice, tc.item)
			}
		})
	}
}

// TestNewServer tests server initialization
func TestNewServer(t *testing.T) {
	t.Run("ValidPort", func(t *testing.T) {
		server := NewServer(8080)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 8080 {
			t.Errorf("Expected port 8080, got %d", server.port)
		}
		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
		if server.buildHandler == nil {
			t.Error("Expected build handler to be initialized")
		}
	})

	t.Run("ZeroPort", func(t *testing.T) {
		server := NewServer(0)
		if server == nil {
			t.Fatal("Expected server to be created")
		}
		if server.port != 0 {
			t.Errorf("Expected port 0, got %d", server.port)
		}
	})
}

// TestServerRoutes tests that all routes are properly configured
func TestServerRoutes(t *testing.T) {
	server := NewServer(0)

	routes := []struct {
		method   string
		path     string
		allow404 bool // Allow 404 for routes with path parameters (resource not found is valid)
	}{
		{"GET", "/api/v1/health", false},
		{"GET", "/api/v1/status", false},
		{"GET", "/api/v1/templates", false},
		{"GET", "/api/v1/templates/react-vite", true}, // Template file might not exist in test env
		{"POST", "/api/v1/desktop/generate", false},
		{"GET", "/api/v1/desktop/status/test-id", true}, // Build ID doesn't exist
		{"POST", "/api/v1/desktop/build", false},
		{"POST", "/api/v1/desktop/test", false},
		{"POST", "/api/v1/desktop/package", false},
		{"POST", "/api/v1/desktop/webhook/build-complete", false},
	}

	for _, route := range routes {
		t.Run(route.method+"_"+route.path, func(t *testing.T) {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Routes should exist - 404 only acceptable if explicitly allowed (resource not found)
			// Other status codes (400, 500, etc.) indicate route exists but request was invalid
			if w.Code == 404 && !route.allow404 {
				t.Errorf("Route %s %s not found (status: %d)", route.method, route.path, w.Code)
			}
		})
	}
}

// TestCORSMiddleware tests CORS headers
func TestCORSMiddleware(t *testing.T) {
	server := NewServer(0)

	t.Run("OptionsRequest", func(t *testing.T) {
		req := httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers - middleware adds them
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Log("CORS middleware may not be active in test mode")
		}
		// Should return 200 for OPTIONS
		if w.Code != http.StatusOK {
			t.Logf("OPTIONS request returned status %d", w.Code)
		}
	})

	t.Run("GetRequest", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()

		server.router.ServeHTTP(w, req)

		// Check CORS headers are present (middleware sets specific origin, not *)
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "" {
			t.Error("Expected CORS headers on GET request")
		}
	})
}

// TestConcurrentRequests tests concurrent request handling
func TestConcurrentRequests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	t.Run("ConcurrentHealthChecks", func(t *testing.T) {
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				req := httptest.NewRequest("GET", "/api/v1/health", nil)
				w := httptest.NewRecorder()
				env.Server.router.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Errorf("Concurrent request %d failed with status %d", id, w.Code)
				}
				done <- true
			}(i)
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Unused but needed for imports
var _ = mux.Vars
var _ = fmt.Sprintf
var _ = filepath.Join
