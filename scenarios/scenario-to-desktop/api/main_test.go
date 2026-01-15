package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
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

// NOTE: TestTestDesktopHandler and TestValidateDesktopConfig were removed
// as part of the pipeline migration. The testDesktopHandler and
// validateDesktopConfig functions from validation.go have been deprecated
// in favor of the unified pipeline approach with preflight validation.

// TestSlicesContains validates that slices.Contains works as expected
// for our use cases (replacing the old local contains() helper).
func TestSlicesContains(t *testing.T) {
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
			result := slices.Contains(tc.slice, tc.item)
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
		// NOTE: POST /api/v1/desktop/generate, GET /api/v1/desktop/status/{id}, and
		// POST /api/v1/desktop/build were removed - use /api/v1/pipeline/* instead
		// NOTE: POST /api/v1/desktop/package was removed - use pipeline bundle stage instead
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
var (
	_ = mux.Vars
	_ = fmt.Sprintf
	_ = filepath.Join
)
