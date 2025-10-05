package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("HeadersAreSet", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(SecurityHeaders())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check security headers (HSTS only set when TLS is enabled)
		headers := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Content-Security-Policy",
		}

		for _, h := range headers {
			if w.Header().Get(h) == "" {
				t.Errorf("Expected header %s to be set", h)
			}
		}

		// HSTS should NOT be set without TLS
		if w.Header().Get("Strict-Transport-Security") != "" {
			t.Error("Expected Strict-Transport-Security to NOT be set without TLS")
		}
	})

	t.Run("HeadersCorrectValues", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(SecurityHeaders())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("Expected X-Content-Type-Options to be 'nosniff', got '%s'", w.Header().Get("X-Content-Type-Options"))
		}

		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Errorf("Expected X-Frame-Options to be 'DENY', got '%s'", w.Header().Get("X-Frame-Options"))
		}
	})
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("DefaultNoOrigins", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(CORS(nil))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Default config has no allowed origins, so header should NOT be set
		if w.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("Expected Access-Control-Allow-Origin header to NOT be set for unconfigured origin")
		}
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		config := &CORSConfig{
			AllowedOrigins:   []string{"http://example.com"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           3600,
		}
		router := setupTestRouter()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header to be set for OPTIONS request")
		}
	})

	t.Run("WithAllowedOrigins", func(t *testing.T) {
		config := &CORSConfig{
			AllowedOrigins:   []string{"http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST"},
			AllowedHeaders:   []string{"Content-Type"},
			AllowCredentials: true,
			MaxAge:           3600,
		}
		router := setupTestRouter()
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:3000" {
			t.Errorf("Expected Access-Control-Allow-Origin to be 'http://localhost:3000', got '%s'", origin)
		}
	})
}

func TestValidateAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("NoAPIKeySet", func(t *testing.T) {
		os.Unsetenv("API_KEY")

		router := setupTestRouter()
		router.Use(ValidateAPIKey())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should allow request when no API key is configured
		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 when no API key is configured, got %d", w.Code)
		}
	})

	t.Run("ValidAPIKey", func(t *testing.T) {
		os.Setenv("API_KEY", "test-api-key-123")
		defer os.Unsetenv("API_KEY")

		router := setupTestRouter()
		router.Use(ValidateAPIKey())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "test-api-key-123")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid API key, got %d", w.Code)
		}
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		os.Setenv("API_KEY", "correct-api-key")
		defer os.Unsetenv("API_KEY")

		router := setupTestRouter()
		router.Use(ValidateAPIKey())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("X-API-Key", "wrong-api-key")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 with invalid API key, got %d", w.Code)
		}
	})

	t.Run("MissingAPIKey", func(t *testing.T) {
		os.Setenv("API_KEY", "required-api-key")
		defer os.Unsetenv("API_KEY")

		router := setupTestRouter()
		router.Use(ValidateAPIKey())
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		// Don't set API key header
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 with missing API key, got %d", w.Code)
		}
	})
}

func TestRateLimiting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("AllowsRequestsUnderLimit", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(RateLimiting(10)) // 10 requests per minute
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		// Make 5 requests (under limit)
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected 200, got %d", i+1, w.Code)
			}
		}
	})

	t.Run("DifferentIPsNotAffected", func(t *testing.T) {
		router := setupTestRouter()
		router.Use(RateLimiting(5))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		// Make requests from different IPs
		ips := []string{"192.168.1.1:12345", "192.168.1.2:12345", "192.168.1.3:12345"}
		for _, ip := range ips {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = ip
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request from %s: Expected 200, got %d", ip, w.Code)
			}
		}
	})
}

func TestSecureWebSocketUpgrader(t *testing.T) {
	t.Run("Returns Upgrader", func(t *testing.T) {
		upgrader := SecureWebSocketUpgrader()
		if upgrader == nil {
			t.Fatal("Expected non-nil WebSocket upgrader")
		}
	})

	t.Run("HasSecureSettings", func(t *testing.T) {
		upgrader := SecureWebSocketUpgrader()

		// Check that it has appropriate buffer sizes
		if upgrader.ReadBufferSize == 0 {
			t.Error("Expected ReadBufferSize to be set")
		}
		if upgrader.WriteBufferSize == 0 {
			t.Error("Expected WriteBufferSize to be set")
		}
	})
}

func TestMiddlewareChaining(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("MultipleMiddlewares", func(t *testing.T) {
		config := &CORSConfig{
			AllowedOrigins: []string{"http://example.com"},
		}
		router := setupTestRouter()
		router.Use(SecurityHeaders())
		router.Use(CORS(config))
		router.GET("/test", func(c *gin.Context) {
			c.String(http.StatusOK, "OK")
		})

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Both security headers and CORS headers should be present
		if w.Header().Get("X-Content-Type-Options") == "" {
			t.Error("Expected security headers to be set")
		}
		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Expected CORS headers to be set for allowed origin")
		}
	})
}
