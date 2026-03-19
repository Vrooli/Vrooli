package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// requireDB opens a database connection or skips the test.
func requireDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("requires database: DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Ensure the table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS onboarding_progress (
			id SERIAL PRIMARY KEY,
			user_id TEXT UNIQUE NOT NULL,
			current_step INTEGER NOT NULL DEFAULT 0,
			completed_steps JSONB NOT NULL DEFAULT '[]',
			config_data JSONB NOT NULL DEFAULT '{}',
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		t.Fatalf("failed to ensure table exists: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})
	return db
}

// TestProgressUpdate verifies PUT then GET of onboarding progress.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressUpdate(t *testing.T) {
	db := requireDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	// Clean up test user before and after
	testUserID := "test-progress-update"
	db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID)
	t.Cleanup(func() {
		db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID)
	})

	// PUT progress
	putBody := `{"user_id": "test-progress-update", "current_step": 3, "completed_steps": [1, 2, 3], "config_data": {"resources": ["postgres"]}}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body: %s", putW.Code, http.StatusOK, putW.Body.String())
	}

	var putResp OnboardingProgress
	if err := json.Unmarshal(putW.Body.Bytes(), &putResp); err != nil {
		t.Fatalf("failed to decode PUT response: %v", err)
	}
	if putResp.CurrentStep != 3 {
		t.Errorf("PUT CurrentStep = %d, want 3", putResp.CurrentStep)
	}
	if putResp.UserID != testUserID {
		t.Errorf("PUT UserID = %q, want %q", putResp.UserID, testUserID)
	}

	// GET progress back
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/progress?user_id=test-progress-update", nil)
	getW := httptest.NewRecorder()
	srv.router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var getResp OnboardingProgress
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}
	if getResp.CurrentStep != 3 {
		t.Errorf("GET CurrentStep = %d, want 3", getResp.CurrentStep)
	}
}

// TestProgressDefaultUser verifies that empty user_id defaults to "default".
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressDefaultUser(t *testing.T) {
	db := requireDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	// Clean up default user before and after
	db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", "default")
	t.Cleanup(func() {
		db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", "default")
	})

	// PUT without user_id - should default to "default"
	putBody := `{"current_step": 1, "completed_steps": [1]}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body: %s", putW.Code, http.StatusOK, putW.Body.String())
	}

	var putResp OnboardingProgress
	if err := json.Unmarshal(putW.Body.Bytes(), &putResp); err != nil {
		t.Fatalf("failed to decode PUT response: %v", err)
	}
	if putResp.UserID != "default" {
		t.Errorf("UserID = %q, want %q", putResp.UserID, "default")
	}

	// GET without user_id - should also default to "default"
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/progress", nil)
	getW := httptest.NewRecorder()
	srv.router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d; body: %s", getW.Code, http.StatusOK, getW.Body.String())
	}

	var getResp OnboardingProgress
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("failed to decode GET response: %v", err)
	}
	if getResp.UserID != "default" {
		t.Errorf("GET UserID = %q, want %q", getResp.UserID, "default")
	}
}

// TestProgressGetNotFound verifies 404 when no progress exists for a user.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressGetNotFound(t *testing.T) {
	db := requireDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/progress?user_id=nonexistent-user-xyz", nil)
	getW := httptest.NewRecorder()
	srv.router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusNotFound {
		t.Fatalf("GET status = %d, want %d; body: %s", getW.Code, http.StatusNotFound, getW.Body.String())
	}
}

// newRouterForTest builds a mux.Router with all routes registered, bypassing
// the health check which requires a non-nil db for the health.DB check.
func newRouterForTest(srv *Server) *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/api/v1/progress", srv.handleGetProgress).Methods("GET")
	r.HandleFunc("/api/v1/progress", srv.handleUpdateProgress).Methods("PUT")
	return r
}
