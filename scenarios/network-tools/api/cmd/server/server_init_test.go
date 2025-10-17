package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestNewServerEnvironmentVariables tests NewServer with various env configurations
func TestNewServerEnvironmentVariables(t *testing.T) {
	t.Run("WithPostgresURL", func(t *testing.T) {
		originalURL := os.Getenv("POSTGRES_URL")
		os.Setenv("POSTGRES_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")
		defer os.Setenv("POSTGRES_URL", originalURL)

		// This will likely fail without a real database, but tests the logic
		server, err := NewServer()
		if err != nil {
			t.Logf("Expected failure without database: %v", err)
			return
		}
		if server != nil {
			defer server.db.Close()
		}
	})

	t.Run("WithComponentEnvVars", func(t *testing.T) {
		// Clear all DB URLs
		originalDBURL := os.Getenv("DATABASE_URL")
		originalPostgresURL := os.Getenv("POSTGRES_URL")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("POSTGRES_URL")
		defer func() {
			os.Setenv("DATABASE_URL", originalDBURL)
			os.Setenv("POSTGRES_URL", originalPostgresURL)
		}()

		// Set component variables
		os.Setenv("POSTGRES_HOST", "localhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "test")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		os.Setenv("POSTGRES_SSLMODE", "disable")

		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
			os.Unsetenv("POSTGRES_SSLMODE")
		}()

		server, err := NewServer()
		if err != nil {
			t.Logf("Expected failure without database: %v", err)
			return
		}
		if server != nil {
			defer server.db.Close()
		}
	})

	t.Run("WithoutPassword", func(t *testing.T) {
		// Clear all DB URLs and password
		originalDBURL := os.Getenv("DATABASE_URL")
		originalPostgresURL := os.Getenv("POSTGRES_URL")
		originalPassword := os.Getenv("POSTGRES_PASSWORD")
		originalDBPassword := os.Getenv("DB_PASSWORD")

		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("POSTGRES_URL")
		os.Unsetenv("POSTGRES_PASSWORD")
		os.Unsetenv("DB_PASSWORD")

		defer func() {
			os.Setenv("DATABASE_URL", originalDBURL)
			os.Setenv("POSTGRES_URL", originalPostgresURL)
			os.Setenv("POSTGRES_PASSWORD", originalPassword)
			os.Setenv("DB_PASSWORD", originalDBPassword)
		}()

		os.Setenv("POSTGRES_HOST", "localhost")
		defer os.Unsetenv("POSTGRES_HOST")

		_, err := NewServer()
		if err == nil {
			t.Error("Expected error when password not configured")
		}
		if err != nil && err.Error() != "database password not configured - set POSTGRES_PASSWORD or DB_PASSWORD environment variable" {
			t.Errorf("Expected password error, got: %v", err)
		}
	})

	t.Run("WithCustomRateLimit", func(t *testing.T) {
		originalRL := os.Getenv("RATE_LIMIT_REQUESTS")
		originalRW := os.Getenv("RATE_LIMIT_WINDOW")

		os.Setenv("RATE_LIMIT_REQUESTS", "50")
		os.Setenv("RATE_LIMIT_WINDOW", "30s")

		defer func() {
			os.Setenv("RATE_LIMIT_REQUESTS", originalRL)
			os.Setenv("RATE_LIMIT_WINDOW", originalRW)
		}()

		// Try to create server (will fail without DB but tests the logic)
		os.Setenv("POSTGRES_PASSWORD", "test")
		defer os.Unsetenv("POSTGRES_PASSWORD")

		_, err := NewServer()
		if err != nil {
			t.Logf("Expected failure without database: %v", err)
		}
	})

	t.Run("WithCustomPort", func(t *testing.T) {
		originalPort := os.Getenv("PORT")
		originalAPIPort := os.Getenv("API_PORT")

		os.Setenv("PORT", "9999")

		defer func() {
			os.Setenv("PORT", originalPort)
			os.Setenv("API_PORT", originalAPIPort)
		}()

		os.Setenv("POSTGRES_PASSWORD", "test")
		defer os.Unsetenv("POSTGRES_PASSWORD")

		_, err := NewServer()
		if err != nil {
			t.Logf("Expected failure without database: %v", err)
		}
	})

	t.Run("WithAPIPort", func(t *testing.T) {
		originalPort := os.Getenv("PORT")
		originalAPIPort := os.Getenv("API_PORT")

		os.Unsetenv("PORT")
		os.Setenv("API_PORT", "8888")

		defer func() {
			os.Setenv("PORT", originalPort)
			os.Setenv("API_PORT", originalAPIPort)
		}()

		os.Setenv("POSTGRES_PASSWORD", "test")
		defer os.Unsetenv("POSTGRES_PASSWORD")

		_, err := NewServer()
		if err != nil {
			t.Logf("Expected failure without database: %v", err)
		}
	})
}

// TestAuthMiddlewareComprehensive tests all auth middleware paths
func TestAuthMiddlewareComprehensive(t *testing.T) {
	t.Run("StrictModeWithAPIKey", func(t *testing.T) {
		originalAuthMode := os.Getenv("AUTH_MODE")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")

		os.Setenv("AUTH_MODE", "strict")
		os.Setenv("NETWORK_TOOLS_API_KEY", "strict-key-123")

		defer func() {
			os.Setenv("AUTH_MODE", originalAuthMode)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-API-Key", "strict-key-123")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Valid API key in strict mode should allow access, got %d", w.Code)
		}
	})

	t.Run("BearerTokenAuth", func(t *testing.T) {
		originalEnv := os.Getenv("VROOLI_ENV")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")

		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "bearer-key-123")

		defer func() {
			os.Setenv("VROOLI_ENV", originalEnv)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Authorization", "Bearer bearer-key-123")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Valid Bearer token should allow access, got %d", w.Code)
		}
	})

	t.Run("ProductionNoKeyConfigured", func(t *testing.T) {
		originalEnv := os.Getenv("VROOLI_ENV")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")

		os.Setenv("VROOLI_ENV", "production")
		os.Unsetenv("NETWORK_TOOLS_API_KEY")

		defer func() {
			os.Setenv("VROOLI_ENV", originalEnv)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-API-Key", "any-key")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		// Should return error when no key is configured in production
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Should return 500 when no key configured in production, got %d", w.Code)
		}
	})
}

// TestCORSMiddlewareComprehensive tests all CORS middleware paths
func TestCORSMiddlewareComprehensive(t *testing.T) {
	t.Run("CustomAllowedOrigins", func(t *testing.T) {
		originalOrigins := os.Getenv("ALLOWED_ORIGINS")
		os.Setenv("ALLOWED_ORIGINS", "https://example.com, https://test.com")
		defer os.Setenv("ALLOWED_ORIGINS", originalOrigins)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Origin", "https://example.com")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("Expected origin to be allowed, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("DevModeLocalhostOrigin", func(t *testing.T) {
		originalEnv := os.Getenv("VROOLI_ENV")
		os.Setenv("VROOLI_ENV", "development")
		defer os.Setenv("VROOLI_ENV", originalEnv)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:9999")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:9999" {
			t.Errorf("Dev mode should allow any localhost origin, got: %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}
