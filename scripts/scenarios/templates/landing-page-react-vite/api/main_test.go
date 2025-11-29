package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
)

func TestHealthEndpoint(t *testing.T) {
	// Set required environment variables for test
	os.Setenv("API_PORT", "15000")
	os.Setenv("POSTGRES_HOST", "localhost")
	os.Setenv("POSTGRES_PORT", "5432")
	os.Setenv("POSTGRES_USER", "test")
	os.Setenv("POSTGRES_PASSWORD", "test")
	os.Setenv("POSTGRES_DB", "test")

	// Note: This test requires a database connection
	// In a real scenario, you'd use a test database or mock
	srv, err := NewServer()
	if err != nil {
		t.Skip("Skipping test - database not available:", err)
	}
	defer srv.db.Close()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestRequireEnv(t *testing.T) {
	// Test valid environment variable
	t.Run("valid environment variable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := requireEnv("TEST_VAR")
		if result != "test_value" {
			t.Errorf("Expected test_value, got %s", result)
		}
	})

	// Test whitespace trimming
	t.Run("whitespace trimming", func(t *testing.T) {
		os.Setenv("TEST_VAR_WS", "  trimmed  ")
		defer os.Unsetenv("TEST_VAR_WS")

		result := requireEnv("TEST_VAR_WS")
		if result != "trimmed" {
			t.Errorf("Expected trimmed, got '%s'", result)
		}
	})
}

func TestHandleTemplateList(t *testing.T) {
	// Create a temporary templates directory with test data
	tmpDir := t.TempDir()

	// Create test server with mock template service
	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{templatesDir: tmpDir},
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()

	srv.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestHandleTemplateShow(t *testing.T) {
	tmpDir := t.TempDir()

	srv := &Server{
		router:          mux.NewRouter(),
		templateService: &TemplateService{templatesDir: tmpDir},
	}
	srv.setupRoutes()

	t.Run("non-existing template returns error", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/templates/non-existing", nil)
		w := httptest.NewRecorder()

		srv.router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("Expected status 404, got %d", w.Code)
		}
	})
}

func TestResolveDatabaseURL(t *testing.T) {
	t.Run("explicit DATABASE_URL", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://explicit:5432/db")
		defer os.Unsetenv("DATABASE_URL")

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if url != "postgres://explicit:5432/db" {
			t.Errorf("Expected explicit URL, got %s", url)
		}
	})

	t.Run("constructed from components", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("POSTGRES_HOST", "testhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
		}()

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "postgres://testuser:testpass@testhost:5432/testdb?sslmode=disable"
		if url != expected {
			t.Errorf("Expected %s, got %s", expected, url)
		}
	})
}

func TestLogStructured(t *testing.T) {
	// Test structured logging (output validation is complex, just ensure no panic)
	logStructured("test_event", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	logStructured("test_event_no_fields", nil)
}
