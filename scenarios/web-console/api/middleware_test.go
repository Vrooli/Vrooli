package main

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	// Create a logger (required by loggingMiddleware)
	logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	middleware := loggingMiddleware(logger, handler)

	t.Run("LogsRequest", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "OK" {
			t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
		}
	})

	t.Run("LogsErrorStatus", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelInfo}))
		errorHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		middleware := loggingMiddleware(logger, errorHandler)

		req := httptest.NewRequest(http.MethodPost, "/error", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("LogsDifferentMethods", func(t *testing.T) {
		methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/test", nil)
			w := httptest.NewRecorder()

			middleware.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Method %s: Expected status 200, got %d", method, w.Code)
			}
		}
	})
}

func TestLoggingResponseWriter(t *testing.T) {
	t.Run("WriteHeader", func(t *testing.T) {
		w := httptest.NewRecorder()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}

		lrw.WriteHeader(http.StatusCreated)

		if lrw.status != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, lrw.status)
		}
		if w.Code != http.StatusCreated {
			t.Errorf("Expected underlying status %d, got %d", http.StatusCreated, w.Code)
		}
	})

	t.Run("DefaultStatus", func(t *testing.T) {
		w := httptest.NewRecorder()
		lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}

		// Write without calling WriteHeader
		lrw.Write([]byte("test"))

		if lrw.status != http.StatusOK {
			t.Errorf("Expected default status 200, got %d", lrw.status)
		}
	})
}
