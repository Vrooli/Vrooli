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
		t.Skipf("database unavailable (open): %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("database unavailable (ping): %v", err)
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
	if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
		t.Fatalf("setup cleanup: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
			t.Logf("cleanup: %v", err)
		}
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
	if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", "default"); err != nil {
		t.Fatalf("setup cleanup: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", "default"); err != nil {
			t.Logf("cleanup: %v", err)
		}
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

// closedDB returns a *sql.DB that is already closed, triggering DB errors on use.
func closedDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", "postgres://invalid:invalid@localhost:1/invalid?sslmode=disable")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}
	db.Close()
	return db
}

// TestProgressGetDBError verifies 500 when database returns a generic error.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressGetDBError(t *testing.T) {
	db := closedDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/progress?user_id=test-db-error", nil)
	getW := httptest.NewRecorder()
	srv.router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusInternalServerError {
		t.Fatalf("GET status = %d, want %d; body: %s", getW.Code, http.StatusInternalServerError, getW.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(getW.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(body["error"], "database error") {
		t.Errorf("expected 'database error' in response, got %q", body["error"])
	}
}

// TestProgressUpdateDBError verifies 500 when database returns a generic error on update.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressUpdateDBError(t *testing.T) {
	db := closedDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	putBody := `{"user_id": "test-db-error", "current_step": 1}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusInternalServerError {
		t.Fatalf("PUT status = %d, want %d; body: %s", putW.Code, http.StatusInternalServerError, putW.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(putW.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !strings.Contains(body["error"], "database error") {
		t.Errorf("expected 'database error' in response, got %q", body["error"])
	}
}

// TestProgressUpdateInvalidJSON verifies 400 for malformed JSON body.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressUpdateInvalidJSON(t *testing.T) {
	db := closedDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader("{bad json"))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusBadRequest {
		t.Fatalf("PUT status = %d, want %d; body: %s", putW.Code, http.StatusBadRequest, putW.Body.String())
	}
}

// TestProgressUpdateMinimalBody verifies defaults are applied for missing fields.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressUpdateMinimalBody(t *testing.T) {
	db := requireDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	testUserID := "test-minimal-body"
	if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
		t.Fatalf("setup cleanup: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	// Only provide user_id — step, completed_steps, config_data should all default
	putBody := `{"user_id": "test-minimal-body"}`
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putW := httptest.NewRecorder()
	srv.router.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT status = %d, want %d; body: %s", putW.Code, http.StatusOK, putW.Body.String())
	}

	var resp OnboardingProgress
	if err := json.Unmarshal(putW.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CurrentStep != 0 {
		t.Errorf("CurrentStep = %d, want 0 (default)", resp.CurrentStep)
	}
	if string(resp.CompletedSteps) != "[]" {
		t.Errorf("CompletedSteps = %s, want []", resp.CompletedSteps)
	}
	if string(resp.ConfigData) != "{}" {
		t.Errorf("ConfigData = %s, want {}", resp.ConfigData)
	}
}

// TestProgressUpsertOverwrite verifies that updating an existing user's progress replaces old values.
// [REQ:REQ-P1-003] - Progress Storage Backend
func TestProgressUpsertOverwrite(t *testing.T) {
	db := requireDB(t)
	srv := &Server{db: db, router: nil}
	srv.router = newRouterForTest(srv)

	testUserID := "test-upsert-overwrite"
	if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
		t.Fatalf("setup cleanup: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM onboarding_progress WHERE user_id = $1", testUserID); err != nil {
			t.Logf("cleanup: %v", err)
		}
	})

	// First insert
	body1 := `{"user_id": "test-upsert-overwrite", "current_step": 1, "completed_steps": [1]}`
	req1 := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first PUT status = %d; body: %s", w1.Code, w1.Body.String())
	}

	// Second update — should overwrite
	body2 := `{"user_id": "test-upsert-overwrite", "current_step": 5, "completed_steps": [1,2,3,4,5], "config_data": {"step": 5}}`
	req2 := httptest.NewRequest(http.MethodPut, "/api/v1/progress", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second PUT status = %d; body: %s", w2.Code, w2.Body.String())
	}

	var resp OnboardingProgress
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.CurrentStep != 5 {
		t.Errorf("CurrentStep = %d, want 5 after upsert", resp.CurrentStep)
	}

	// Verify GET returns the updated values
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/progress?user_id=test-upsert-overwrite", nil)
	getW := httptest.NewRecorder()
	srv.router.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET status = %d; body: %s", getW.Code, getW.Body.String())
	}

	var getResp OnboardingProgress
	if err := json.Unmarshal(getW.Body.Bytes(), &getResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if getResp.CurrentStep != 5 {
		t.Errorf("GET CurrentStep = %d, want 5", getResp.CurrentStep)
	}
}
