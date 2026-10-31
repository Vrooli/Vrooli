package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type middlewareRecordingLogger struct{ messages []string }

func (l *middlewareRecordingLogger) Info(message string, _ ...any) {
	l.messages = append(l.messages, message)
}

func TestMiddlewareAddsSecurityHeadersAndLogs(t *testing.T) {
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	logger := &middlewareRecordingLogger{}
	handler := LoggingMiddlewareWithLogger(logger, SecurityHeadersMiddleware(next))
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusNoContent {
		t.Fatalf("middleware response=%d called=%v", rec.Code, called)
	}
	for key, want := range map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-XSS-Protection":          "0",
		"Referrer-Policy":           "no-referrer",
		"X-Frame-Options":           "SAMEORIGIN",
		"Permissions-Policy":        "camera=(), microphone=(), geolocation=()",
		"Content-Security-Policy":   "default-src 'none'; frame-ancestors 'self'",
	} {
		if got := rec.Header().Get(key); got != want {
			t.Errorf("%s=%q want %q", key, got, want)
		}
	}
	if len(logger.messages) != 1 || logger.messages[0] != "request" {
		t.Fatalf("logger messages=%v", logger.messages)
	}
	LogStructuredWith(nil, "ignored", nil)
}
