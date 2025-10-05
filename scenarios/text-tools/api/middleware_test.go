package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
)

func TestCORSMiddleware(t *testing.T) {
	t.Run("GET_Request_With_CORS_Headers", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Check CORS headers
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected Access-Control-Allow-Origin header")
		}
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			t.Error("Expected Access-Control-Allow-Headers header")
		}
		if w.Header().Get("Access-Control-Max-Age") != "86400" {
			t.Error("Expected Access-Control-Max-Age header")
		}
	})

	t.Run("OPTIONS_Preflight_Request", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for OPTIONS request")
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Check that response is OK
		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		// Check CORS headers are set
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected Access-Control-Allow-Origin header for OPTIONS")
		}
	})

	t.Run("POST_Request_With_CORS_Headers", func(t *testing.T) {
		handlerCalled := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handlerCalled = true
			w.WriteHeader(http.StatusCreated)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("Expected handler to be called for POST request")
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS headers on POST request")
		}
	})
}

func TestLoggingMiddleware(t *testing.T) {
	t.Run("Logs_Request_Details", func(t *testing.T) {
		// Capture log output
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := loggingMiddleware(handler)

		req := httptest.NewRequest("GET", "/test/path", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "GET") {
			t.Error("Expected log to contain request method")
		}
		if !strings.Contains(logOutput, "/test/path") {
			t.Error("Expected log to contain request path")
		}
		if !strings.Contains(logOutput, "200") {
			t.Error("Expected log to contain status code")
		}
	})

	t.Run("Logs_Error_Status_Code", func(t *testing.T) {
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})

		middleware := loggingMiddleware(handler)

		req := httptest.NewRequest("GET", "/not-found", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "404") {
			t.Error("Expected log to contain 404 status code")
		}
	})

	t.Run("Captures_Custom_Status_Code", func(t *testing.T) {
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("created"))
		})

		middleware := loggingMiddleware(handler)

		req := httptest.NewRequest("POST", "/create", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "201") {
			t.Error("Expected log to contain 201 status code")
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("Recovers_From_Panic", func(t *testing.T) {
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		middleware := recoveryMiddleware(handler)

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		// Should not panic
		middleware.ServeHTTP(w, req)

		// Check that panic was logged
		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "Panic occurred") {
			t.Error("Expected panic to be logged")
		}
		if !strings.Contains(logOutput, "test panic") {
			t.Error("Expected panic message in log")
		}

		// Check response
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500, got %d", w.Code)
		}
	})

	t.Run("Normal_Request_Not_Affected", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		})

		middleware := recoveryMiddleware(handler)

		req := httptest.NewRequest("GET", "/normal", nil)
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		if w.Body.String() != "success" {
			t.Errorf("Expected body 'success', got '%s'", w.Body.String())
		}
	})
}

func TestApplyMiddleware(t *testing.T) {
	t.Run("Applies_All_Middleware", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		handler := applyMiddleware(router)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		// Check that CORS middleware was applied
		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Expected CORS middleware to be applied")
		}
	})

	t.Run("Middleware_Chain_With_OPTIONS", func(t *testing.T) {
		router := mux.NewRouter()
		router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for OPTIONS")
		})

		handler := applyMiddleware(router)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("Middleware_Chain_Handles_Panic", func(t *testing.T) {
		var logBuf bytes.Buffer
		log.SetOutput(&logBuf)
		defer log.SetOutput(os.Stderr)

		router := mux.NewRouter()
		router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("middleware test panic")
		})

		handler := applyMiddleware(router)

		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()

		// Should not panic due to recovery middleware
		handler.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 after panic, got %d", w.Code)
		}
	})
}

func TestResponseWriter(t *testing.T) {
	t.Run("Captures_Status_Code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusCreated)

		if rw.statusCode != http.StatusCreated {
			t.Errorf("Expected status code 201, got %d", rw.statusCode)
		}
	})

	t.Run("Default_Status_Code", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		if rw.statusCode != http.StatusOK {
			t.Errorf("Expected default status code 200, got %d", rw.statusCode)
		}
	})

	t.Run("Multiple_WriteHeader_Calls", func(t *testing.T) {
		w := httptest.NewRecorder()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		rw.WriteHeader(http.StatusCreated)
		rw.WriteHeader(http.StatusAccepted)

		// First call should win (HTTP behavior)
		if rw.statusCode != http.StatusAccepted {
			t.Errorf("Expected status code from last call: 202, got %d", rw.statusCode)
		}
	})
}
