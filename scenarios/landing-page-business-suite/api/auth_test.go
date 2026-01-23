package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

func TestHandleAdminLogin_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminUsers(t, db)
	cleanupAdminSessions(t, db)

	// Create test admin user
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("validpassword123"), bcrypt.DefaultCost)
	_, err := db.Exec(`
		INSERT INTO admin_users (email, password_hash)
		VALUES ('admin@test.com', $1)
	`, string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to insert test admin: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	reqBody := `{"email":"admin@test.com","password":"validpassword123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleAdminLogin(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if !resp.Authenticated {
		t.Error("Expected authenticated=true")
	}
	if resp.Email != "admin@test.com" {
		t.Errorf("Expected email 'admin@test.com', got '%s'", resp.Email)
	}
}

func TestHandleAdminLogin_InvalidBody(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleAdminLogin(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestHandleAdminLogin_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminUsers(t, db)

	// Create test admin user
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.DefaultCost)
	_, err := db.Exec(`
		INSERT INTO admin_users (email, password_hash)
		VALUES ('admin@test.com', $1)
	`, string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to insert test admin: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	reqBody := `{"email":"admin@test.com","password":"wrongpassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleAdminLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleAdminLogin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminUsers(t, db)

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	reqBody := `{"email":"nonexistent@test.com","password":"anypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleAdminLogin(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleAdminLogout_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminSessions(t, db)

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "admin@test.com",
		"session_id": "test_session_id",
	})

	// Insert server-side session
	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at)
		VALUES ('test_session_id', 'admin@test.com', $1)
	`, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil)
	rr := httptest.NewRecorder()

	server.handleAdminLogout(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}

	// Verify session was deleted from database
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE id = 'test_session_id'`).Scan(&count)
	if count != 0 {
		t.Error("Expected session to be deleted from database")
	}
}

func TestHandleAdminLogout_NoSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil)
	rr := httptest.NewRecorder()

	server.handleAdminLogout(rr, req)

	// Should still return success even without session
	if rr.Code != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", rr.Code)
	}
}

func TestHandleAdminSession_ValidSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminSessions(t, db)

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "valid@test.com",
		"session_id": "valid_session_id",
	})

	// Insert valid server-side session
	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at)
		VALUES ('valid_session_id', 'valid@test.com', $1)
	`, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	rr := httptest.NewRecorder()

	server.handleAdminSession(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp LoginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if !resp.Authenticated {
		t.Error("Expected authenticated=true")
	}
}

func TestHandleAdminSession_NoSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	rr := httptest.NewRecorder()

	server.handleAdminSession(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestHandleAdminSession_ExpiredSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminSessions(t, db)

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "expired@test.com",
		"session_id": "expired_session_id",
	})

	// Insert expired server-side session
	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at)
		VALUES ('expired_session_id', 'expired@test.com', $1)
	`, time.Now().Add(-1*time.Hour)) // Expired 1 hour ago
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	rr := httptest.NewRecorder()

	server.handleAdminSession(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestRequireAdmin_ValidSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminSessions(t, db)

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "admin@test.com",
		"session_id": "valid_session",
	})

	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at)
		VALUES ('valid_session', 'admin@test.com', $1)
	`, time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	called := false
	handler := server.requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Expected handler to be called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestRequireAdmin_NoSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	called := false
	handler := server.requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Error("Expected handler NOT to be called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestRequireAdmin_ExpiredSession(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminSessions(t, db)

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "admin@test.com",
		"session_id": "expired_session",
	})

	_, err := db.Exec(`
		INSERT INTO admin_sessions (id, admin_email, expires_at)
		VALUES ('expired_session', 'admin@test.com', $1)
	`, time.Now().Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	called := false
	handler := server.requireAdmin(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if called {
		t.Error("Expected handler NOT to be called for expired session")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rr.Code)
	}
}

func TestValidateAdminPasswordUpdate_Valid(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := validateAdminPasswordUpdate("newpassword456", string(currentHash))
	if err != nil {
		t.Errorf("Expected no error for valid password, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_TooShort(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := validateAdminPasswordUpdate("short1", string(currentHash))
	if err == nil {
		t.Error("Expected error for short password")
	}
	if !strings.Contains(err.Error(), "12 characters") {
		t.Errorf("Expected error about length, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_NoLetters(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := validateAdminPasswordUpdate("123456789012", string(currentHash))
	if err == nil {
		t.Error("Expected error for password without letters")
	}
	if !strings.Contains(err.Error(), "letters and numbers") {
		t.Errorf("Expected error about letters, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_NoDigits(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := validateAdminPasswordUpdate("onlylettershere", string(currentHash))
	if err == nil {
		t.Error("Expected error for password without digits")
	}
	if !strings.Contains(err.Error(), "letters and numbers") {
		t.Errorf("Expected error about digits, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_SameAsOld(t *testing.T) {
	password := "samepassword123"
	currentHash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	err := validateAdminPasswordUpdate(password, string(currentHash))
	if err == nil {
		t.Error("Expected error for same password as current")
	}
	if !strings.Contains(err.Error(), "different from the current") {
		t.Errorf("Expected error about same password, got: %v", err)
	}
}

func TestBuildLoginResponse(t *testing.T) {
	tests := []struct {
		name          string
		email         string
		authenticated bool
		wantEmail     string
	}{
		{"authenticated with email", "test@example.com", true, "test@example.com"},
		{"authenticated no email", "", true, ""},
		{"not authenticated", "ignored@example.com", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := buildLoginResponse(tt.email, tt.authenticated)
			if resp.Authenticated != tt.authenticated {
				t.Errorf("Expected authenticated=%v, got %v", tt.authenticated, resp.Authenticated)
			}
			if resp.Email != tt.wantEmail {
				t.Errorf("Expected email='%s', got '%s'", tt.wantEmail, resp.Email)
			}
			if !resp.ResetEnabled {
				t.Error("Expected ResetEnabled=true")
			}
		})
	}
}

func TestGenerateSessionID(t *testing.T) {
	id1, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID failed: %v", err)
	}
	if len(id1) != 64 { // 32 bytes = 64 hex chars
		t.Errorf("Expected 64 char session ID, got %d chars", len(id1))
	}

	id2, err := generateSessionID()
	if err != nil {
		t.Fatalf("generateSessionID failed: %v", err)
	}

	if id1 == id2 {
		t.Error("Expected unique session IDs")
	}
}

func TestIsSecureCookiesEnabled(t *testing.T) {
	// This tests the function based on environment
	// Result depends on current env, so we just verify it doesn't panic
	_ = isSecureCookiesEnabled()
}

func TestResolveSecret(t *testing.T) {
	// Test that it doesn't panic and returns empty for non-existent keys
	result := resolveSecret("NON_EXISTENT_KEY_12345")
	if result != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", result)
	}
}

func TestSessionManager_Interface(t *testing.T) {
	// Test that MockSessionManager implements SessionManager
	var _ SessionManager = &MockSessionManager{}
	var _ SessionManager = &cookieSessionManager{}
}

func TestMockSessionManager_SetAndGetValues(t *testing.T) {
	mock := NewMockSessionManager()

	mock.SetSessionValues("test_session", map[interface{}]interface{}{
		"key1": "value1",
		"key2": 42,
	})

	session, err := mock.GetSession(nil, "test_session")
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if session.Values["key1"] != "value1" {
		t.Errorf("Expected key1='value1', got '%v'", session.Values["key1"])
	}
	if session.Values["key2"] != 42 {
		t.Errorf("Expected key2=42, got '%v'", session.Values["key2"])
	}
}

func TestMockSessionManager_SaveError(t *testing.T) {
	mock := NewMockSessionManager()
	mock.SaveError = sql.ErrConnDone

	session := sessions.NewSession(nil, "test")
	err := mock.SaveSession(nil, nil, session)

	if err != sql.ErrConnDone {
		t.Errorf("Expected SaveError to be returned, got %v", err)
	}
}

func TestHandleAdminProfile_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminUsers(t, db)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	_, err := db.Exec(`
		INSERT INTO admin_users (email, password_hash)
		VALUES ('profile@test.com', $1)
	`, string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to insert test admin: %v", err)
	}

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email": "profile@test.com",
	})

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
	rr := httptest.NewRecorder()

	server.handleAdminProfile(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp AdminProfileResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if resp.Email != "profile@test.com" {
		t.Errorf("Expected email 'profile@test.com', got '%s'", resp.Email)
	}
}

func TestHandleAdminProfileUpdate_SessionInvalidation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	cleanupAdminUsers(t, db)
	cleanupAdminSessions(t, db)

	passwordHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)
	_, err := db.Exec(`
		INSERT INTO admin_users (email, password_hash)
		VALUES ('update@test.com', $1)
	`, string(passwordHash))
	if err != nil {
		t.Fatalf("Failed to insert test admin: %v", err)
	}

	// Insert multiple sessions
	for i, sessionID := range []string{"current_session", "other_session_1", "other_session_2"} {
		_, err := db.Exec(`
			INSERT INTO admin_sessions (id, admin_email, expires_at)
			VALUES ($1, 'update@test.com', $2)
		`, sessionID, time.Now().Add(24*time.Hour))
		if err != nil {
			t.Fatalf("Failed to insert session %d: %v", i, err)
		}
	}

	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email":      "update@test.com",
		"session_id": "current_session",
	})

	server := &Server{
		db:             db,
		sessionManager: mockSession,
	}

	reqBody, _ := json.Marshal(map[string]string{
		"current_password": "oldpassword123",
		"new_password":     "newpassword456",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/profile", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	server.handleAdminProfileUpdate(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify other sessions were invalidated
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE admin_email = 'update@test.com'`).Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 session remaining (current), got %d", count)
	}

	// Verify current session still exists
	var exists bool
	db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_sessions WHERE id = 'current_session')`).Scan(&exists)
	if !exists {
		t.Error("Current session should not have been invalidated")
	}
}

// cleanupAdminUsers removes all admin users from the test database
func cleanupAdminUsers(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'"); err != nil {
		t.Fatalf("Failed to cleanup admin_users: %v", err)
	}
}

// cleanupAdminSessions removes all admin sessions from the test database
func cleanupAdminSessions(t *testing.T, db *sql.DB) {
	t.Helper()
	if _, err := db.Exec("DELETE FROM admin_sessions"); err != nil {
		t.Fatalf("Failed to cleanup admin_sessions: %v", err)
	}
}
