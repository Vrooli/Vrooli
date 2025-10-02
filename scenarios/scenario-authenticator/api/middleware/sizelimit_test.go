package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRequestSizeLimitMiddleware(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read the body to trigger MaxBytesReader limit check
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	tests := []struct {
		name          string
		bodySize      int
		envMaxSizeMB  string
		expectSuccess bool
	}{
		{
			name:          "small request within default 1MB limit",
			bodySize:      1024, // 1KB
			envMaxSizeMB:  "",
			expectSuccess: true,
		},
		{
			name:          "request exceeds default 1MB limit",
			bodySize:      2 * 1024 * 1024, // 2MB
			envMaxSizeMB:  "",
			expectSuccess: false,
		},
		{
			name:          "request within custom 5MB limit",
			bodySize:      3 * 1024 * 1024, // 3MB
			envMaxSizeMB:  "5",
			expectSuccess: true,
		},
		{
			name:          "request exceeds custom 5MB limit",
			bodySize:      6 * 1024 * 1024, // 6MB
			envMaxSizeMB:  "5",
			expectSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable if specified
			if tt.envMaxSizeMB != "" {
				os.Setenv("MAX_REQUEST_SIZE_MB", tt.envMaxSizeMB)
				defer os.Unsetenv("MAX_REQUEST_SIZE_MB")
			}

			// Create request with specified body size
			body := bytes.NewReader(bytes.Repeat([]byte("a"), tt.bodySize))
			req := httptest.NewRequest("POST", "/test", body)
			rec := httptest.NewRecorder()

			// Apply middleware
			handler := RequestSizeLimitMiddleware(testHandler)
			handler.ServeHTTP(rec, req)

			// Check result
			if tt.expectSuccess {
				if rec.Code != http.StatusOK {
					t.Errorf("Expected status 200 for small request, got %d", rec.Code)
				}
			} else {
				// MaxBytesReader returns 413 (Request Entity Too Large) when limit exceeded
				if rec.Code != http.StatusRequestEntityTooLarge {
					t.Errorf("Expected status 413 for oversized request, got %d", rec.Code)
				}
			}
		})
	}
}

func TestRequestSizeLimitMiddleware_InvalidEnvVar(t *testing.T) {
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read the body to trigger MaxBytesReader limit check
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		envValue   string
		bodySize   int
		shouldFail bool
	}{
		{
			name:       "invalid non-numeric env var falls back to 1MB",
			envValue:   "invalid",
			bodySize:   2 * 1024 * 1024,
			shouldFail: true,
		},
		{
			name:       "negative env var falls back to 1MB",
			envValue:   "-1",
			bodySize:   2 * 1024 * 1024,
			shouldFail: true,
		},
		{
			name:       "zero env var falls back to 1MB",
			envValue:   "0",
			bodySize:   2 * 1024 * 1024,
			shouldFail: true,
		},
		{
			name:       "oversized env var (>100MB) falls back to 1MB",
			envValue:   "150",
			bodySize:   2 * 1024 * 1024,
			shouldFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("MAX_REQUEST_SIZE_MB", tt.envValue)
			defer os.Unsetenv("MAX_REQUEST_SIZE_MB")

			body := strings.NewReader(strings.Repeat("a", tt.bodySize))
			req := httptest.NewRequest("POST", "/test", body)
			rec := httptest.NewRecorder()

			handler := RequestSizeLimitMiddleware(testHandler)
			handler.ServeHTTP(rec, req)

			if tt.shouldFail && rec.Code != http.StatusRequestEntityTooLarge {
				t.Errorf("Expected request to fail with status 413, but got status %d", rec.Code)
			}
		})
	}
}
