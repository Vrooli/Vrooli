package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterRoutes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	manager, metrics, ws := setupTestSessionManager(t, cfg)

	mux := http.NewServeMux()
	registerRoutes(mux, manager, metrics, ws)

	t.Run("HealthzEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("MetricsEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("WorkspaceEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/workspace", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("SessionsEndpoint", func(t *testing.T) {
		reqBody := TestData.CreateSessionRequest("", nil)
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should create session
		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Cleanup created session
		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok && id != "" {
				if s, ok := manager.getSession(id); ok {
					cleanupSession(s)
				}
			}
		}
	})

	t.Run("NotFoundEndpoint", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/nonexistent", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestRegisterRoutesWithProxyGuard(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cfg := setupTestConfig(env.TempDir)
	cfg.enableProxyGuard = true // Enable proxy guard
	manager, metrics, ws := setupTestSessionManager(t, cfg)

	mux := http.NewServeMux()
	registerRoutes(mux, manager, metrics, ws)

	t.Run("RejectsWithoutProxyHeaders", func(t *testing.T) {
		reqBody := TestData.CreateSessionRequest("", nil)
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}
	})

	t.Run("AllowsWithProxyHeaders", func(t *testing.T) {
		reqBody := TestData.CreateSessionRequest("", nil)
		req := makeHTTPRequest(HTTPTestRequest{
			Method: http.MethodPost,
			Path:   "/api/v1/sessions",
			Body:   reqBody,
		})
		addProxyHeaders(req)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("Expected status 201, got %d", w.Code)
		}

		// Cleanup created session
		response := assertJSONResponse(t, w, http.StatusCreated, nil)
		if response != nil {
			if id, ok := response["id"].(string); ok && id != "" {
				if s, ok := manager.getSession(id); ok {
					cleanupSession(s)
				}
			}
		}
	})

	t.Run("HealthzBypassesProxyGuard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected healthz to bypass proxy guard, got status %d", w.Code)
		}
	})

	t.Run("MetricsBypassesProxyGuard", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected metrics to bypass proxy guard, got status %d", w.Code)
		}
	})
}
