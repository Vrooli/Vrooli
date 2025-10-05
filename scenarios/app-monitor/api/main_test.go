// +build testing

package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	// Setup
	gin.SetMode(gin.TestMode)
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	os.Setenv("API_PORT", "8080")

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestNewServer(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		cfg := setupTestConfig()
		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("Expected NewServer to succeed, got error: %v", err)
		}
		if server == nil {
			t.Fatal("Expected non-nil server")
		}
		if server.router == nil {
			t.Error("Expected server to have router initialized")
		}
		if server.handlers == nil {
			t.Error("Expected server to have handlers initialized")
		}
	})

	t.Run("HandlersInitialized", func(t *testing.T) {
		cfg := setupTestConfig()
		server, err := NewServer(cfg)
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		if server.handlers.health == nil {
			t.Error("Expected health handler to be initialized")
		}
		if server.handlers.app == nil {
			t.Error("Expected app handler to be initialized")
		}
		if server.handlers.system == nil {
			t.Error("Expected system handler to be initialized")
		}
		if server.handlers.docker == nil {
			t.Error("Expected docker handler to be initialized")
		}
		if server.handlers.websocket == nil {
			t.Error("Expected websocket handler to be initialized")
		}
	})
}

func TestSetupRouter(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RoutesRegistered", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Test health endpoints
		routes := []struct {
			method string
			path   string
		}{
			{"GET", "/health"},
			{"GET", "/api/health"},
			{"GET", "/api/v1/system/metrics"},
			{"GET", "/api/v1/apps/summary"},
			{"GET", "/api/v1/apps"},
			{"GET", "/api/v1/apps/:id"},
			{"POST", "/api/v1/apps/:id/start"},
			{"POST", "/api/v1/apps/:id/stop"},
			{"POST", "/api/v1/apps/:id/restart"},
			{"GET", "/api/v1/resources"},
			{"GET", "/api/v1/docker/info"},
		}

		for _, route := range routes {
			req := httptest.NewRequest(route.method, route.path, nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			// We're just checking the route exists, not that it returns 200
			// 404 would mean the route doesn't exist
			if w.Code == http.StatusNotFound && !strings.Contains(route.path, ":") {
				t.Errorf("Route %s %s not registered (got 404)", route.method, route.path)
			}
		}
	})

	t.Run("MiddlewareApplied", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Check for CORS headers (from middleware)
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Expected CORS middleware to set Access-Control-Allow-Origin header")
		}

		// Check for security headers
		if w.Header().Get("X-Content-Type-Options") == "" {
			t.Error("Expected security middleware to set X-Content-Type-Options header")
		}
	})
}

func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("BasicHealth", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		assertContentType(t, w, "application/json")
	})

	t.Run("APIHealth", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		assertContentType(t, w, "application/json")
		assertResponseContains(t, w, "timestamp")
	})
}

func TestSystemMetricsEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetSystemMetrics", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/system/metrics", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		assertContentType(t, w, "application/json")
	})
}

func TestAppsEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetAppsSummary", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/apps/summary", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// May return 200 or 500 depending on whether orchestrator is available
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		assertContentType(t, w, "application/json")
	})

	t.Run("GetApps", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/apps", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// May return 200 or 500 depending on whether orchestrator is available
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}

		assertContentType(t, w, "application/json")
	})

	t.Run("GetAppNotFound", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/apps/nonexistent-app", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestResourceEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetResources", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/resources", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// May succeed or fail depending on environment
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestDockerEndpoints(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("GetDockerInfo", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/docker/info", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Docker may not be available in test environment
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("GetContainers", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/docker/containers", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Docker may not be available in test environment
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d. Body: %s", w.Code, w.Body.String())
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("OptionsRequest", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("OPTIONS", "/api/v1/apps", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "GET")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Expected Access-Control-Allow-Origin header to be set")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set")
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SecurityHeadersPresent", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		headers := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
		}

		for _, header := range headers {
			if w.Header().Get(header) == "" {
				t.Errorf("Expected security header '%s' to be set", header)
			}
		}
	})
}

func TestInvalidRoutes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NotFoundRoute", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404 for nonexistent route, got %d", w.Code)
		}
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		// Try POST on a GET-only endpoint
		req := httptest.NewRequest("POST", "/health", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusMethodNotAllowed && w.Code != http.StatusNotFound {
			t.Errorf("Expected status 405 or 404 for invalid method, got %d", w.Code)
		}
	})
}

func TestWebSocketEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WebSocketWithoutUpgrade", func(t *testing.T) {
		server, cleanup := setupTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/ws", nil)
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		// Should fail without proper WebSocket upgrade headers
		if w.Code == http.StatusOK {
			t.Error("Expected WebSocket endpoint to fail without upgrade headers")
		}
	})
}
