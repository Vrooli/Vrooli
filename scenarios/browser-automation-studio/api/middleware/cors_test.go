package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestCORSMiddlewareAllowsAppMonitorByDefault(t *testing.T) {
	ResetCorsConfigForTesting()
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_ORIGIN", "")
	t.Setenv("UI_PORT", "36221")

	log := logrus.New()
	middleware := CorsMiddleware(log)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/projects/test", nil)
	req.Header.Set("Origin", "https://app-monitor.itsagitime.com")

	called := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if !called {
		t.Fatalf("expected next handler to be called")
	}

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status OK, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://app-monitor.itsagitime.com" {
		t.Fatalf("expected Access-Control-Allow-Origin to match origin, got %q", got)
	}

	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("expected credentials header to be true, got %q", got)
	}
}

func TestCORSMiddlewareAllowsOriginlessLocalProbe(t *testing.T) {
	ResetCorsConfigForTesting()
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://allowed.example")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_ORIGIN", "")

	log := logrus.New()
	middleware := CorsMiddleware(log)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	called := false
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(rr, req)

	if !called || rr.Code != http.StatusOK {
		t.Fatalf("expected originless probe to reach handler with 200, called=%t status=%d", called, rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("originless probe must not receive an allow-origin header, got %q", got)
	}
}

func TestCORSMiddlewareRejectsUnauthorizedOrigin(t *testing.T) {
	ResetCorsConfigForTesting()
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://allowed.example")
	t.Setenv("ALLOWED_ORIGINS", "")
	t.Setenv("CORS_ALLOWED_ORIGIN", "")

	log := logrus.New()
	middleware := CorsMiddleware(log)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Origin", "https://not-allowed.example")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("expected request to be rejected before reaching next handler")
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status forbidden, got %d", rr.Code)
	}

	if rr.Body.String() == "" {
		t.Fatalf("expected error message in response body")
	}
}

func TestCORSMiddlewareRejectsWildcardConfiguration(t *testing.T) {
	ResetCorsConfigForTesting()
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	log := logrus.New()
	middleware := CorsMiddleware(log)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil)
	req.Header.Set("Origin", "https://any.example")

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("wildcard CORS configuration must not reach the handler")
	}))

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected status forbidden, got %d", rr.Code)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no wildcard origin header, got %q", got)
	}
}
