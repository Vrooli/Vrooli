package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrapped.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Body.String() != "test response" {
		t.Errorf("Expected body 'test response', got '%s'", w.Body.String())
	}
}

func TestCORSMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := corsMiddleware(testHandler)

	tests := []struct {
		name           string
		method         string
		wantStatus     int
		checkHeaders   bool
	}{
		{
			name:         "GET request",
			method:       "GET",
			wantStatus:   http.StatusOK,
			checkHeaders: true,
		},
		{
			name:         "POST request",
			method:       "POST",
			wantStatus:   http.StatusOK,
			checkHeaders: true,
		},
		{
			name:         "OPTIONS request",
			method:       "OPTIONS",
			wantStatus:   http.StatusOK,
			checkHeaders: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.checkHeaders {
				// Check CORS headers
				origin := w.Header().Get("Access-Control-Allow-Origin")
				if origin != "*" {
					t.Errorf("Expected Access-Control-Allow-Origin '*', got '%s'", origin)
				}

				methods := w.Header().Get("Access-Control-Allow-Methods")
				if methods == "" {
					t.Error("Expected Access-Control-Allow-Methods header to be set")
				}

				headers := w.Header().Get("Access-Control-Allow-Headers")
				if headers == "" {
					t.Error("Expected Access-Control-Allow-Headers header to be set")
				}
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if user context is set
		user := r.Context().Value("user")
		if user == nil {
			t.Error("Expected user context to be set")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	})

	wrapped := authMiddleware(testHandler)

	tests := []struct {
		name       string
		path       string
		authHeader string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "health endpoint bypasses auth",
			path:       "/health",
			authHeader: "",
			wantStatus: http.StatusOK,
			wantBody:   "authenticated",
		},
		{
			name:       "missing auth header",
			path:       "/workflows",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "invalid auth header format",
			path:       "/workflows",
			authHeader: "InvalidFormat",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "invalid token",
			path:       "/workflows",
			authHeader: "Bearer invalid_token",
			wantStatus: http.StatusUnauthorized,
			wantBody:   "",
		},
		{
			name:       "valid token",
			path:       "/workflows",
			authHeader: "Bearer metareasoning_cli_default_2024",
			wantStatus: http.StatusOK,
			wantBody:   "authenticated",
		},
		{
			name:       "admin token",
			path:       "/workflows",
			authHeader: "Bearer admin_token_2024",
			wantStatus: http.StatusOK,
			wantBody:   "authenticated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantBody != "" && w.Body.String() != tt.wantBody {
				t.Errorf("Expected body '%s', got '%s'", tt.wantBody, w.Body.String())
			}
		})
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	// Handler that panics
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	// Handler that doesn't panic
	normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("normal response"))
	})

	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantPanic  bool
	}{
		{
			name:       "recovers from panic",
			handler:    panicHandler,
			wantStatus: http.StatusInternalServerError,
			wantPanic:  true,
		},
		{
			name:       "normal operation",
			handler:    normalHandler,
			wantStatus: http.StatusOK,
			wantPanic:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := recoveryMiddleware(tt.handler)

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()

			// Execute request (should not panic even if handler panics)
			wrapped.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantPanic {
				// Check error response
				body := w.Body.String()
				if body == "" {
					t.Error("Expected error response body")
				}
			}
		})
	}
}

func TestIsValidToken(t *testing.T) {
	tests := []struct {
		token string
		valid bool
	}{
		{"metareasoning_cli_default_2024", true},
		{"test_token_2024", true},
		{"admin_token_2024", true},
		{"invalid_token", false},
		{"", false},
		{"Bearer metareasoning_cli_default_2024", false}, // Should not include Bearer prefix
	}

	for _, tt := range tests {
		result := isValidToken(tt.token)
		if result != tt.valid {
			t.Errorf("Token '%s': expected %v, got %v", tt.token, tt.valid, result)
		}
	}
}

func TestGetUserFromToken(t *testing.T) {
	tests := []struct {
		token    string
		expected string
	}{
		{"admin_token_2024", "admin"},
		{"metareasoning_cli_default_2024", "cli"},
		{"test_token_2024", "user"},
		{"unknown_token", "user"},
		{"", "user"},
	}

	for _, tt := range tests {
		result := getUserFromToken(tt.token)
		if result != tt.expected {
			t.Errorf("Token '%s': expected user '%s', got '%s'", 
				tt.token, tt.expected, result)
		}
	}
}

func TestResponseWriter(t *testing.T) {
	// Test custom response writer
	baseWriter := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: baseWriter,
		statusCode:     http.StatusOK,
	}

	// Test WriteHeader
	rw.WriteHeader(http.StatusCreated)
	if rw.statusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", rw.statusCode)
	}

	// Test Write (should use underlying writer)
	data := []byte("test data")
	n, err := rw.Write(data)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}
}

func TestMiddlewareChain(t *testing.T) {
	// Test that multiple middleware can be chained
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value("user")
		if user != nil {
			w.Write([]byte("user:" + user.(string)))
		} else {
			w.Write([]byte("no user"))
		}
	})

	// Chain middlewares
	handler := recoveryMiddleware(
		loggingMiddleware(
			corsMiddleware(
				authMiddleware(testHandler),
			),
		),
	)

	// Test with valid auth
	req := httptest.NewRequest("GET", "/workflows", nil)
	req.Header.Set("Authorization", "Bearer admin_token_2024")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Check CORS headers are set
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("CORS headers not set")
	}

	// Check user context was set
	if w.Body.String() != "user:admin" {
		t.Errorf("Expected 'user:admin', got '%s'", w.Body.String())
	}
}

func TestAuthMiddlewareContextPropagation(t *testing.T) {
	// Test that user context is properly propagated
	var capturedUser string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user := r.Context().Value("user"); user != nil {
			capturedUser = user.(string)
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := authMiddleware(testHandler)

	tests := []struct {
		token        string
		expectedUser string
	}{
		{"admin_token_2024", "admin"},
		{"metareasoning_cli_default_2024", "cli"},
		{"test_token_2024", "user"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedUser, func(t *testing.T) {
			capturedUser = ""
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer "+tt.token)
			w := httptest.NewRecorder()

			wrapped.ServeHTTP(w, req)

			if capturedUser != tt.expectedUser {
				t.Errorf("Expected user '%s', got '%s'", tt.expectedUser, capturedUser)
			}
		})
	}
}