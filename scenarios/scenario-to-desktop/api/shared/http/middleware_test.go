package http

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCORSMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("with explicit allowed origin", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			AllowedOrigin: "https://example.com",
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://example.com")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("expected origin 'https://example.com', got %q", rr.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("with UI port", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			UIPort: "3000",
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		origin := rr.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:3000" {
			t.Errorf("expected origin 'http://localhost:3000', got %q", origin)
		}
	})

	t.Run("127.0.0.1 origin matches", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			UIPort: "3000",
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://127.0.0.1:3000")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		origin := rr.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://127.0.0.1:3000" {
			t.Errorf("expected origin 'http://127.0.0.1:3000', got %q", origin)
		}
	})

	t.Run("OPTIONS request returns OK", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			AllowedOrigin: "https://example.com",
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodOptions, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200 for OPTIONS, got %d", rr.Code)
		}
	})

	t.Run("no config uses restrictive localhost", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			Logger: slog.Default(),
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "http://localhost:5000")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		origin := rr.Header().Get("Access-Control-Allow-Origin")
		// Should allow localhost origins
		if !strings.HasPrefix(origin, "http://localhost") && !strings.HasPrefix(origin, "http://127.0.0.1") {
			t.Errorf("expected localhost origin, got %q", origin)
		}
	})

	t.Run("non-localhost origin rejected when no config", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			Logger: slog.Default(),
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Origin", "https://evil.com")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		origin := rr.Header().Get("Access-Control-Allow-Origin")
		// Should default to localhost, not the evil origin
		if origin == "https://evil.com" {
			t.Error("should not allow non-localhost origin without config")
		}
	})

	t.Run("sets all expected headers", func(t *testing.T) {
		middleware := CORSMiddleware(CORSConfig{
			AllowedOrigin: "https://example.com",
		})

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		expectedHeaders := []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Methods",
			"Access-Control-Allow-Headers",
			"Access-Control-Allow-Credentials",
		}

		for _, header := range expectedHeaders {
			if rr.Header().Get(header) == "" {
				t.Errorf("expected header %q to be set", header)
			}
		}
	})
}

func TestCORSMiddlewareFromEnv(t *testing.T) {
	// This test just verifies the function doesn't panic
	middleware := CORSMiddlewareFromEnv(slog.Default())
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	var buf bytes.Buffer
	middleware := LoggingMiddleware(&buf)

	wrapped := middleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	// The gorilla handlers.LoggingHandler writes Apache-style logs
	// Just verify something was written
	if buf.Len() == 0 {
		t.Error("expected log output to be written")
	}
}

func TestLoggingMiddlewareStdout(t *testing.T) {
	// Just verify it doesn't panic
	middleware := LoggingMiddlewareStdout()
	if middleware == nil {
		t.Error("expected middleware to be created")
	}
}

func TestStructuredLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	middleware := StructuredLoggingMiddleware(logger)

	wrapped := middleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	if buf.Len() == 0 {
		t.Error("expected log output")
	}

	log := buf.String()
	if !strings.Contains(log, "HTTP request") {
		t.Error("expected log to contain 'HTTP request'")
	}
	if !strings.Contains(log, "method=GET") {
		t.Error("expected log to contain method")
	}
	if !strings.Contains(log, "path=/test") {
		t.Error("expected log to contain path")
	}
}

func TestStructuredLoggingMiddlewareNilLogger(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := StructuredLoggingMiddleware(nil)
	wrapped := middleware(handler)

	// Should not panic with nil logger
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}

func TestStructuredLoggingMiddlewareLogLevels(t *testing.T) {
	tests := []struct {
		name           string
		handlerStatus  int
		expectLogLevel string
	}{
		{"success logs info", http.StatusOK, "INFO"},
		{"client error logs warn", http.StatusNotFound, "WARN"},
		{"server error logs error", http.StatusInternalServerError, "ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.handlerStatus)
			})

			var buf bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&buf, nil))
			middleware := StructuredLoggingMiddleware(logger)

			wrapped := middleware(handler)
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rr := httptest.NewRecorder()

			wrapped.ServeHTTP(rr, req)

			log := buf.String()
			if !strings.Contains(log, tc.expectLogLevel) {
				t.Errorf("expected log level %q in output: %s", tc.expectLogLevel, log)
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		middleware := RecoveryMiddleware(logger)

		wrapped := middleware(handler)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		// Should not panic
		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}

		log := buf.String()
		if !strings.Contains(log, "panic recovered") {
			t.Error("expected panic to be logged")
		}
	})

	t.Run("passes through on no panic", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := RecoveryMiddleware(slog.Default())
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("nil logger", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		middleware := RecoveryMiddleware(nil)
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		// Should not panic
		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rr.Code)
		}
	})
}

func TestChain(t *testing.T) {
	var order []string

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	chained := Chain(middleware1, middleware2)
	wrapped := chained(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	wrapped.ServeHTTP(rr, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Errorf("expected %d calls, got %d", len(expected), len(order))
		return
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RequestIDMiddleware()
	wrapped := middleware(handler)

	t.Run("generates new ID when not present", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		requestID := rr.Header().Get("X-Request-ID")
		if requestID == "" {
			t.Error("expected X-Request-ID to be set")
		}
	})

	t.Run("preserves existing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", "existing-id")
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		requestID := rr.Header().Get("X-Request-ID")
		if requestID != "existing-id" {
			t.Errorf("expected X-Request-ID 'existing-id', got %q", requestID)
		}
	})
}

func TestTimeoutMiddleware(t *testing.T) {
	t.Run("request completes within timeout", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		middleware := TimeoutMiddleware(1 * time.Second)
		wrapped := middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		wrapped.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	// Note: Testing actual timeouts is tricky in unit tests due to goroutine timing.
	// The http.TimeoutHandler creates a separate goroutine which makes deterministic
	// testing difficult. In production, integration tests would verify timeout behavior.
}

func TestResponseWriterWrapper(t *testing.T) {
	t.Run("captures status code", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := &responseWriter{ResponseWriter: rr}

		wrapped.WriteHeader(http.StatusNotFound)

		if wrapped.status != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", wrapped.status)
		}
		if !wrapped.wroteHeader {
			t.Error("expected wroteHeader to be true")
		}
	})

	t.Run("only writes header once", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := &responseWriter{ResponseWriter: rr}

		wrapped.WriteHeader(http.StatusOK)
		wrapped.WriteHeader(http.StatusNotFound)

		if wrapped.status != http.StatusOK {
			t.Errorf("expected status 200 (first call), got %d", wrapped.status)
		}
	})

	t.Run("Write sets default status", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := &responseWriter{ResponseWriter: rr}

		wrapped.Write([]byte("hello"))

		if wrapped.status != http.StatusOK {
			t.Errorf("expected default status 200, got %d", wrapped.status)
		}
	})

	t.Run("tracks written bytes", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := &responseWriter{ResponseWriter: rr}

		wrapped.Write([]byte("hello"))
		wrapped.Write([]byte(" world"))

		if wrapped.size != 11 {
			t.Errorf("expected size 11, got %d", wrapped.size)
		}
	})

	t.Run("Unwrap returns underlying writer", func(t *testing.T) {
		rr := httptest.NewRecorder()
		wrapped := &responseWriter{ResponseWriter: rr}

		unwrapped := wrapped.Unwrap()
		if unwrapped != rr {
			t.Error("expected Unwrap to return original ResponseWriter")
		}
	})
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("expected non-empty request ID")
	}
	if id1 == id2 {
		t.Error("expected different request IDs")
	}
}
