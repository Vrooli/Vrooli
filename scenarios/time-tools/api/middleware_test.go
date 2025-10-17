package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("WithAllowedOrigin", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:3000" {
			t.Errorf("Expected origin http://localhost:3000, got %s", origin)
		}

		if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("Expected Access-Control-Allow-Credentials to be true")
		}
	})

	t.Run("WithDisallowedOrigin", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://evil.com")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin == "http://evil.com" {
			t.Error("Should not allow disallowed origin")
		}
	})

	t.Run("WithNoOrigin", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test", nil)

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Should still set CORS headers
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("OptionsRequest", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}

		methods := w.Header().Get("Access-Control-Allow-Methods")
		if methods == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("CheckAllowedMethods", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		methods := w.Header().Get("Access-Control-Allow-Methods")
		if methods != "GET, POST, PUT, DELETE, OPTIONS" {
			t.Errorf("Expected specific allowed methods, got %s", methods)
		}
	})

	t.Run("CheckAllowedHeaders", func(t *testing.T) {
		handler := corsMiddleware(testHandler)

		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("Origin", "http://localhost:8080")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		headers := w.Header().Get("Access-Control-Allow-Headers")
		if headers != "Content-Type, Authorization" {
			t.Errorf("Expected specific allowed headers, got %s", headers)
		}
	})
}

// TestLoggingMiddleware tests logging middleware
func TestLoggingMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	t.Run("LogsRequest", func(t *testing.T) {
		handler := loggingMiddleware(testHandler)

		req := httptest.NewRequest("GET", "/test/path", nil)
		req.RemoteAddr = "127.0.0.1:12345"

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("LogsPOSTRequest", func(t *testing.T) {
		handler := loggingMiddleware(testHandler)

		req := httptest.NewRequest("POST", "/api/v1/endpoint", nil)
		req.RemoteAddr = "192.168.1.1:54321"

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}

// TestMiddlewareChaining tests that middleware can be chained
func TestMiddlewareChaining(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	t.Run("CORSThenLogging", func(t *testing.T) {
		handler := loggingMiddleware(corsMiddleware(testHandler))

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CORS header is still set
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:3000" {
			t.Errorf("Expected origin header after chaining, got %s", origin)
		}
	})
}
