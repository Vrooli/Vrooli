package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"golang.org/x/crypto/bcrypt"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	"landing-page-business-suite-api/internal/commerce"
)

func TestHandleAdminLogin_Success(t *testing.T) {
	db := setupTestDB(t)
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

	resp, loginError := adminhttp.LoginSession(req, rr, adminhttp.LoginRequest{Email: "admin@test.com", Password: "validpassword123"}, server.adminSessionDependencies())
	if loginError != nil {
		t.Fatalf("login failed: %v", loginError)
	}
	if !resp.Authenticated {
		t.Error("Expected authenticated=true")
	}
	if resp.Email != "admin@test.com" {
		t.Errorf("Expected email 'admin@test.com', got '%s'", resp.Email)
	}
}

func TestHandleAdminLogin_InvalidCredentials(t *testing.T) {
	db := setupTestDB(t)
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

	_, loginError := adminhttp.LoginSession(req, rr, adminhttp.LoginRequest{Email: "admin@test.com", Password: "wrongpassword"}, server.adminSessionDependencies())
	if loginError == nil || loginError.Status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized login error, got %v", loginError)
	}
}

func TestHandleAdminLogin_UserNotFound(t *testing.T) {
	db := setupTestDB(t)
	cleanupAdminUsers(t, db)

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	reqBody := `{"email":"nonexistent@test.com","password":"anypassword"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	_, loginError := adminhttp.LoginSession(req, rr, adminhttp.LoginRequest{Email: "nonexistent@test.com", Password: "anypassword"}, server.adminSessionDependencies())
	if loginError == nil || loginError.Status != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized login error, got %v", loginError)
	}
}

func TestHandleAdminLogout_Success(t *testing.T) {
	db := setupTestDB(t)
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

	adminhttp.LogoutSession(req, rr, server.adminSessionDependencies())

	// Verify session was deleted from database
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE id = 'test_session_id'`).Scan(&count); err != nil {
		t.Fatalf("Failed to count admin sessions: %v", err)
	}
	if count != 0 {
		t.Error("Expected session to be deleted from database")
	}
}

func TestHandleAdminLogout_NoSession(t *testing.T) {
	db := setupTestDB(t)

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/logout", nil)
	rr := httptest.NewRecorder()

	adminhttp.LogoutSession(req, rr, server.adminSessionDependencies())
}

func TestHandleAdminSession_ValidSession(t *testing.T) {
	db := setupTestDB(t)
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

	resp, authenticated := adminhttp.ReadSession(req, rr, server.adminSessionDependencies())
	if !authenticated || !resp.Authenticated {
		t.Error("Expected authenticated=true")
	}
}

func TestHandleAdminSession_NoSession(t *testing.T) {
	db := setupTestDB(t)

	server := &Server{
		db:             db,
		sessionManager: NewMockSessionManager(),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	rr := httptest.NewRecorder()

	_, authenticated := adminhttp.ReadSession(req, rr, server.adminSessionDependencies())
	if authenticated {
		t.Fatal("expected no authenticated session")
	}
}

func TestHandleAdminSession_ExpiredSession(t *testing.T) {
	db := setupTestDB(t)
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

	_, authenticated := adminhttp.ReadSession(req, rr, server.adminSessionDependencies())
	if authenticated {
		t.Fatal("expected expired session to be rejected")
	}
}

func TestRequireAdmin_ValidSession(t *testing.T) {
	db := setupTestDB(t)
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

func TestRequireAdminOrService_ValidatesServiceTokenAndFallsBackToAdminSession(t *testing.T) {
	db := setupTestDB(t)
	service := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{ServiceToken: "service-token"})

	for _, tc := range []struct {
		name          string
		usageService  *commerce.UsageService
		authorization string
		wantStatus    int
		wantCalled    bool
	}{
		{
			name:          "configured bearer token",
			usageService:  service,
			authorization: "Bearer service-token",
			wantStatus:    http.StatusNoContent,
			wantCalled:    true,
		},
		{
			name:          "invalid bearer token",
			usageService:  service,
			authorization: "Bearer wrong-token",
			wantStatus:    http.StatusUnauthorized,
		},
		{
			name:          "usage service unavailable",
			authorization: "Bearer service-token",
			wantStatus:    http.StatusUnauthorized,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := &Server{db: db, sessionManager: NewMockSessionManager(), usageService: tc.usageService}
			called := false
			handler := server.requireAdminOrService(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil)
			req.Header.Set("Authorization", tc.authorization)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rr.Code, tc.wantStatus)
			}
			if called != tc.wantCalled {
				t.Errorf("handler called = %v, want %v", called, tc.wantCalled)
			}
		})
	}
}

func TestRequireAdminOrService_AllowsValidAdminSessionFallback(t *testing.T) {
	mockSession := NewMockSessionManager()
	mockSession.SetSessionValues("admin_session", map[interface{}]interface{}{
		"email": "admin@test.com",
	})
	server := &Server{
		sessionManager: mockSession,
		usageService:   commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{ServiceToken: "service-token"}),
	}
	called := false
	handler := server.requireAdminOrService(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")

	handler.ServeHTTP(rr, req)

	if !called || rr.Code != http.StatusNoContent {
		t.Errorf("admin-session fallback = called %v, status %d; want true, %d", called, rr.Code, http.StatusNoContent)
	}
}

func TestRequireAdmin_RejectsSessionManagerErrors(t *testing.T) {
	server := &Server{sessionManager: &MockSessionManager{GetError: os.ErrPermission}}
	handler := server.requireAdmin(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler must not be called")
	})
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/admin/protected", nil))

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestSessionAdminEmail_RejectsInvalidSessions(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	for _, tc := range []struct {
		name    string
		manager *MockSessionManager
		want    string
		ok      bool
	}{
		{name: "session manager error", manager: &MockSessionManager{GetError: os.ErrPermission}},
		{name: "empty email", manager: NewMockSessionManager()},
		{name: "whitespace email", manager: NewMockSessionManager()},
		{name: "valid email", manager: NewMockSessionManager(), want: "admin@example.com", ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "whitespace email" {
				tc.manager.SetSessionValues("admin_session", map[interface{}]interface{}{"email": "  \t"})
			}
			if tc.name == "valid email" {
				tc.manager.SetSessionValues("admin_session", map[interface{}]interface{}{"email": tc.want})
			}
			server := &Server{sessionManager: tc.manager}

			email, ok := server.sessionAdminEmail(request)

			if email != tc.want || ok != tc.ok {
				t.Errorf("sessionAdminEmail() = (%q, %v), want (%q, %v)", email, ok, tc.want, tc.ok)
			}
		})
	}
}

func TestRequireAdmin_ExpiredSession(t *testing.T) {
	db := setupTestDB(t)
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

	err := adminhttp.ValidateProfilePassword("newpassword456", string(currentHash), "")
	if err != nil {
		t.Errorf("Expected no error for valid password, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_TooShort(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := adminhttp.ValidateProfilePassword("short1", string(currentHash), "")
	if err == nil {
		t.Error("Expected error for short password")
	}
	if !strings.Contains(err.Error(), "12 characters") {
		t.Errorf("Expected error about length, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_NoLetters(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := adminhttp.ValidateProfilePassword("123456789012", string(currentHash), "")
	if err == nil {
		t.Error("Expected error for password without letters")
	}
	if !strings.Contains(err.Error(), "letters and numbers") {
		t.Errorf("Expected error about letters, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_NoDigits(t *testing.T) {
	currentHash, _ := bcrypt.GenerateFromPassword([]byte("oldpassword123"), bcrypt.DefaultCost)

	err := adminhttp.ValidateProfilePassword("onlylettershere", string(currentHash), "")
	if err == nil {
		t.Error("Expected error for password without digits")
	}
	if !strings.Contains(err.Error(), "letters and numbers") {
		t.Errorf("Expected error about digits, got: %v", err)
	}
}

func TestValidateAdminPasswordUpdate_SameAsOld(t *testing.T) {
	currentPassphrase := "same-admin-credential-123"
	currentHash, _ := bcrypt.GenerateFromPassword([]byte(currentPassphrase), bcrypt.DefaultCost)

	err := adminhttp.ValidateProfilePassword(currentPassphrase, string(currentHash), "")
	if err == nil {
		t.Error("Expected error for same password as current")
	}
	if !strings.Contains(err.Error(), "different from the current") {
		t.Errorf("Expected error about same password, got: %v", err)
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
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("LPBS_SECURE_COOKIES", "")
	if !isSecureCookiesEnabled() {
		t.Fatal("production must default to secure cookies")
	}
	t.Setenv("LPBS_SECURE_COOKIES", "false")
	if isSecureCookiesEnabled() {
		t.Fatal("explicit false must disable secure cookies for local development")
	}
}

func TestResolveSecret(t *testing.T) {
	// Test that it doesn't panic and returns empty for non-existent keys
	result := resolveSecret("NON_EXISTENT_KEY_12345")
	if result != "" {
		t.Errorf("Expected empty string for non-existent key, got '%s'", result)
	}
}

// isolateSecretResolution prevents production-configuration tests from
// accidentally consulting a developer's user-level ~/.vrooli/secrets.json.
// Each test controls the complete secret source it is asserting against.
func isolateSecretResolution(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, ".vrooli", "secrets.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir isolated secrets directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatalf("write isolated secrets file: %v", err)
	}
	t.Setenv("VROOLI_ROOT", root)
}

func TestInitSessionManagerUsesDistinctEphemeralKeys(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("SESSION_SECRET", "")

	firstManager := initSessionManager()
	firstRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	firstSession, err := firstManager.GetSession(firstRequest, "admin_session")
	if err != nil {
		t.Fatalf("first session: %v", err)
	}
	firstSession.Values["email"] = "admin@example.test"
	firstResponse := httptest.NewRecorder()
	if err := firstManager.SaveSession(firstRequest, firstResponse, firstSession); err != nil {
		t.Fatalf("save first session: %v", err)
	}

	secondRequest := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range firstResponse.Result().Cookies() {
		secondRequest.AddCookie(cookie)
	}
	secondManager := initSessionManager()
	if _, err := secondManager.GetSession(secondRequest, "admin_session"); err == nil {
		t.Fatal("session signed with an ephemeral key was accepted after a fresh initialization")
	}
}

func TestValidateProductionCredentialsRejectsMissingSessionSecret(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "strong-password-123")

	err := validateProductionCredentials()
	if err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Fatalf("validateProductionCredentials() error = %v, want missing SESSION_SECRET", err)
	}
}

func TestValidateProductionCredentialsRejectsMissingAdminPassword(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "stable-session-secret")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "https://app.example.test/auth/verify")

	err := validateProductionCredentials()
	if err == nil || !strings.Contains(err.Error(), "ADMIN_DEFAULT_PASSWORD") {
		t.Fatalf("validateProductionCredentials() error = %v, want missing ADMIN_DEFAULT_PASSWORD", err)
	}
}

func TestValidateProductionCredentialsRejectsWeakAdminPassword(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "stable-session-secret")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "changeme123")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "https://app.example.test/auth/verify")

	err := validateProductionCredentials()
	if err == nil || !strings.Contains(err.Error(), "at least 12 characters") {
		t.Fatalf("validateProductionCredentials() error = %v, want weak admin password rejection", err)
	}
}

func TestNewServerRejectsMissingProductionSessionSecretBeforeDatabaseConnection(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "strong-password-123")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "https://app.example.test/auth/verify")

	if _, err := NewServer(); err == nil || !strings.Contains(err.Error(), "SESSION_SECRET") {
		t.Fatalf("NewServer() error = %v, want missing SESSION_SECRET rejection", err)
	}
}

func TestValidateProductionCredentialsRejectsMissingMagicLinkBaseURL(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "stable-session-secret")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "strong-password-123")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "")

	err := validateProductionCredentials()
	if err == nil || !strings.Contains(err.Error(), "AUTH_MAGIC_LINK_BASE_URL") {
		t.Fatalf("validateProductionCredentials() error = %v, want missing AUTH_MAGIC_LINK_BASE_URL", err)
	}
}

func TestValidateProductionCredentialsRejectsInsecureMagicLinkBaseURL(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "stable-session-secret")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "strong-password-123")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "http://localhost:3000/auth/verify")

	err := validateProductionCredentials()
	if err == nil || !strings.Contains(err.Error(), "absolute https URL") {
		t.Fatalf("validateProductionCredentials() error = %v, want https URL validation", err)
	}
}

func TestValidateProductionCredentialsAcceptsExplicitSecrets(t *testing.T) {
	isolateSecretResolution(t)
	t.Setenv("LPBS_ENVIRONMENT", "production")
	t.Setenv("SESSION_SECRET", "stable-session-secret")
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "strong-password-123")
	t.Setenv("AUTH_MAGIC_LINK_BASE_URL", "https://app.example.test/auth/verify")

	if err := validateProductionCredentials(); err != nil {
		t.Fatalf("validateProductionCredentials() error = %v", err)
	}
}

func TestSessionManager_Interface(t *testing.T) {
	managers := []SessionManager{&MockSessionManager{}, &cookieSessionManager{}}
	if len(managers) != 2 {
		t.Fatalf("session manager implementations = %d, want 2", len(managers))
	}
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

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	response, err := adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).GetAdminProfile(context.Background(), profileGetRequest(req))
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if response.Msg.GetProfile().GetEmail() != "profile@test.com" {
		t.Errorf("Expected email 'profile@test.com', got '%s'", response.Msg.GetProfile().GetEmail())
	}
}

func TestHandleAdminProfileUpdate_SessionInvalidation(t *testing.T) {
	db := setupTestDB(t)
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

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	response, err := adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).UpdateAdminProfile(context.Background(), profileUpdateRequest(req, &lpbsv1.UpdateAdminProfileRequest{CurrentPassword: "oldpassword123", NewPassword: "newpassword456"}))
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if response.Msg.GetProfile().GetEmail() != "update@test.com" {
		t.Errorf("updated profile email = %q", response.Msg.GetProfile().GetEmail())
	}

	// Verify other sessions were invalidated
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admin_sessions WHERE admin_email = 'update@test.com'`).Scan(&count); err != nil {
		t.Fatalf("Failed to count admin sessions: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session remaining (current), got %d", count)
	}

	// Verify current session still exists
	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admin_sessions WHERE id = 'current_session')`).Scan(&exists); err != nil {
		t.Fatalf("Failed to check current session: %v", err)
	}
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
