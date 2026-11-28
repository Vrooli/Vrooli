package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newMiddlewareTestRouter(path string, middleware gin.HandlerFunc) *gin.Engine {
	router := gin.New()
	router.Use(middleware)
	router.GET(path, func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	return router
}

func sendPreviewRequest(router http.Handler, target, token, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	if token != "" {
		req.Header.Set("X-Preview-Token", token)
	}
	if remoteAddr != "" {
		req.RemoteAddr = remoteAddr
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := newMiddlewareTestRouter("/test", SecurityHeadersMiddleware())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options should be set to DENY")
	}

	// Test X-Content-Type-Options
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options should be set to nosniff")
	}

	// Test X-XSS-Protection
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("X-XSS-Protection should be set")
	}

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header should be set")
	}

	expectedAncestors := "frame-ancestors 'self' http://localhost:* http://127.0.0.1:* http://[::1]:*"
	if !strings.Contains(csp, expectedAncestors) {
		t.Errorf("Content-Security-Policy missing expected frame-ancestors directive; got %q", csp)
	}
}

func TestSecurityHeadersMiddlewareWithExtraFrameAncestors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("FRAME_ANCESTORS", "https://preview.example.com https://*.customer.example")

	router := newMiddlewareTestRouter("/test", SecurityHeadersMiddleware())

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("Content-Security-Policy header should be set")
	}

	for _, ancestor := range []string{"https://preview.example.com", "https://*.customer.example"} {
		if !strings.Contains(csp, ancestor) {
			t.Errorf("Content-Security-Policy missing custom frame ancestor %q: %q", ancestor, csp)
		}
	}
}

func TestPreviewAccessMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, name := range []string{
		"VROOLI_LIFECYCLE",
		"VROOLI_LIFECYCLE_MANAGED",
		"SCENARIO_MODE",
		"VROOLI_PHASE",
		"GRAPH_STUDIO_DISABLE_PREVIEW_GUARD",
		"GRAPH_STUDIO_FORCE_PREVIEW_GUARD",
	} {
		t.Setenv(name, "")
	}
	for _, name := range []string{
		"PREVIEW_ACCESS_TOKEN",
		"GRAPH_STUDIO_PREVIEW_TOKEN",
		"VITE_PREVIEW_ACCESS_TOKEN",
		"SCENARIO_NAME",
		"VROOLI_SCENARIO",
		"VROOLI_SCENARIO_NAME",
		"APP_MONITOR_SCENARIO",
		"APP_MONITOR_PREVIEW_TOKEN",
		"APP_MONITOR_PREVIEW_TOKENS",
		"PREVIEW_TOKEN_FILE",
		"APP_MONITOR_PREVIEW_TOKEN_FILE",
		"GRAPH_STUDIO_PREVIEW_TOKEN_FILE",
	} {
		t.Setenv(name, "")
	}
	t.Setenv("SCENARIO_ROOT", t.TempDir())

	t.Run("no token configured", func(t *testing.T) {

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when token disabled, got %d", w.Code)
		}
	})

	t.Run("missing token denied", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "secret-token")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when preview token missing, got %d", w.Code)
		}

		if !strings.Contains(w.Body.String(), "Preview access denied") {
			t.Errorf("expected denial message in response, got %q", w.Body.String())
		}
	})

	t.Run("legacy env variable supported", func(t *testing.T) {
		t.Setenv("GRAPH_STUDIO_PREVIEW_TOKEN", "legacy-token")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when preview token missing, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/test", "legacy-token", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when legacy preview token matches, got %d", w.Code)
		}
	})

	t.Run("valid token accepted", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "secret-token")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "secret-token", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when preview token matches, got %d", w.Code)
		}
	})

	t.Run("multiple tokens supported", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "token-one, token-two")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "token-two", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when using one of multiple preview tokens, got %d", w.Code)
		}
	})

	t.Run("query parameter without header rejected", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "token-from-query")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test?preview_token=token-from-query", "", "")

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when preview token only provided via query, got %d", w.Code)
		}
	})

	t.Run("managed lifecycle without token", func(t *testing.T) {
		t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		t.Setenv("VROOLI_LIFECYCLE", "active")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when managed lifecycle runs without configured token, got %d", w.Code)
		}
	})

	t.Run("managed lifecycle respects configured token", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "managed-secret")
		t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		t.Setenv("VROOLI_LIFECYCLE", "active")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when header missing under managed lifecycle, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/test", "managed-secret", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when managed token matches, got %d", w.Code)
		}
	})

	t.Run("scenario mode implies managed lifecycle", func(t *testing.T) {
		t.Setenv("SCENARIO_MODE", "true")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when scenario mode enabled without token, got %d", w.Code)
		}
	})

	t.Run("disable guard env bypasses enforcement", func(t *testing.T) {
		t.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
		t.Setenv("GRAPH_STUDIO_DISABLE_PREVIEW_GUARD", "true")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when guard explicitly disabled, got %d", w.Code)
		}
	})

	t.Run("force guard env enforces even without lifecycle", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "forced-token")
		t.Setenv("GRAPH_STUDIO_FORCE_PREVIEW_GUARD", "true")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when guard forced and header missing, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/test", "forced-token", "")
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when guard forced and token provided, got %d", w.Code)
		}
	})

	t.Run("scenario name implies managed lifecycle", func(t *testing.T) {
		t.Setenv("SCENARIO_NAME", "graph-studio")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when scenario name set but token missing, got %d", w.Code)
		}
	})

	t.Run("scenario name with token", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "scenario-secret")
		t.Setenv("SCENARIO_NAME", "graph-studio")

		router := newMiddlewareTestRouter("/test", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/test", "", "")
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 when token missing from request, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/test", "scenario-secret", "")
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when scenario token matches, got %d", w.Code)
		}
	})

	t.Run("health endpoint allows local probes but enforces token otherwise", func(t *testing.T) {
		t.Setenv("PREVIEW_ACCESS_TOKEN", "health-token")

		router := newMiddlewareTestRouter("/health", PreviewAccessMiddleware())

		w := sendPreviewRequest(router, "/health", "", "203.0.113.10:54321")
		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403 for remote health access without token, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/health", "", "127.0.0.1:54321")
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 for local health probe without token, got %d", w.Code)
		}

		w = sendPreviewRequest(router, "/health", "health-token", "203.0.113.10:54321")
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200 when remote health probe includes header token, got %d", w.Code)
		}
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limit test in short mode")
	}

	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RateLimitMiddleware(50, 100)) // 50 req/sec, burst of 100
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Make a few requests that should succeed
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}

func TestRequestSizeLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	maxSize := int64(10 * 1024 * 1024) // 10 MB
	router := gin.New()
	router.Use(RequestSizeLimitMiddleware(maxSize))
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// Small request should succeed
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(LoggingMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestIDMiddleware())
	router.GET("/test", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		if requestID == "" {
			t.Error("Request ID should be set")
		}
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)
}

func TestUserContextMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(UserContextMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "test-user-123")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestErrorHandlerMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorHandlerMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(TimeoutMiddleware(5 * time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/test", func(c *gin.Context) {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	// Should recover and return 500
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 after panic, got %d", w.Code)
	}
}
