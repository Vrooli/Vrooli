package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewHealthHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		handler := NewHealthHandler(nil, nil, nil)

		if handler == nil {
			t.Fatal("Expected non-nil handler")
		}
	})
}

func TestHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, nil, nil)

	t.Run("Success", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/health", handler.Check)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}

		// Should return JSON
		contentType := w.Header().Get("Content-Type")
		if contentType == "" {
			t.Error("Expected Content-Type header to be set")
		}
	})

	t.Run("ResponseFormat", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/health", handler.Check)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()
		if body == "" {
			t.Error("Expected non-empty response body")
		}

		// Should contain status information
		if !containsAny(body, []string{"status", "timestamp", "healthy"}) {
			t.Error("Expected health check response to contain status information")
		}
	})
}

func TestAPIHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, nil, nil)

	t.Run("Success", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/api/health", handler.APIHealth)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d. Body: %s", w.Code, w.Body.String())
		}
	})

	t.Run("DetailedResponse", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/api/health", handler.APIHealth)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		body := w.Body.String()
		// API health should provide more detailed information
		if !containsAny(body, []string{"database", "redis", "docker", "status"}) {
			t.Error("Expected detailed health information in API health response")
		}
	})
}

func TestHealthWithDatabaseConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// With nil DB connection
	handler := NewHealthHandler(nil, nil, nil)

	t.Run("WithoutDatabase", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/api/health", handler.APIHealth)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 even without DB
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 even without DB, got %d", w.Code)
		}
	})
}

func TestHealthWithRedisConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// With nil Redis connection
	handler := NewHealthHandler(nil, nil, nil)

	t.Run("WithoutRedis", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/api/health", handler.APIHealth)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 even without Redis
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 even without Redis, got %d", w.Code)
		}
	})
}

func TestHealthWithDockerConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// With nil Docker connection
	handler := NewHealthHandler(nil, nil, nil)

	t.Run("WithoutDocker", func(t *testing.T) {
		router := setupTestRouter()
		router.GET("/api/health", handler.APIHealth)

		req := httptest.NewRequest("GET", "/api/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still return 200 even without Docker
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 even without Docker, got %d", w.Code)
		}
	})
}

func TestHealthConcurrency(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewHealthHandler(nil, nil, nil)
	router := setupTestRouter()
	router.GET("/health", handler.Check)

	t.Run("ConcurrentRequests", func(t *testing.T) {
		// Make multiple concurrent requests
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func() {
				req := httptest.NewRequest("GET", "/health", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if w.Code != http.StatusOK {
					t.Errorf("Expected 200 in concurrent request, got %d", w.Code)
				}
				done <- true
			}()
		}

		// Wait for all requests to complete
		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// Helper function to check if string contains any of the substrings
func containsAny(s string, substrings []string) bool {
	for _, sub := range substrings {
		if contains(s, sub) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
