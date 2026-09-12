package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"golang.org/x/crypto/bcrypt"
	adminhttp "landing-page-business-suite-api/handlers/administration"
)

func attachAdminSession(t *testing.T, manager SessionManager, req *http.Request, email string) {
	t.Helper()
	session, err := manager.GetSession(req, "admin_session")
	if err != nil {
		t.Fatalf("get admin session: %v", err)
	}
	session.Values["email"] = email
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("failed to save admin session: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
}

func attachProfileConnectSession[T any](request *http.Request, connectRequest *connect.Request[T]) {
	connectRequest.Header().Set("Cookie", request.Header.Get("Cookie"))
}

func profileGetRequest(request *http.Request) *connect.Request[lpbsv1.GetAdminProfileRequest] {
	connectRequest := connect.NewRequest(&lpbsv1.GetAdminProfileRequest{})
	attachProfileConnectSession(request, connectRequest)
	return connectRequest
}

func profileUpdateRequest(request *http.Request, message *lpbsv1.UpdateAdminProfileRequest) *connect.Request[lpbsv1.UpdateAdminProfileRequest] {
	connectRequest := connect.NewRequest(message)
	attachProfileConnectSession(request, connectRequest)
	return connectRequest
}

func attachConnectSessionCookies(request *http.Request, headers http.Header) {
	for _, cookie := range headers.Values("Set-Cookie") {
		if value, _, found := strings.Cut(cookie, ";"); found {
			request.Header.Add("Cookie", value)
		}
	}
}

func configureAdminProfileTestCredentials(t *testing.T) {
	t.Helper()
	t.Setenv("ADMIN_DEFAULT_EMAIL", defaultAdminEmail)
	t.Setenv("ADMIN_DEFAULT_PASSWORD", "changeme123")
	t.Setenv("SESSION_SECRET", "admin-profile-test-session-secret")
}

// seedAdminProfileTestCredentials establishes the credential that each profile
// test presents. setupTestDB reuses a shared testcontainer, and seedDefaultData
// intentionally never overwrites an existing administrator; relying on the
// process's first test to choose that hash makes these tests order-dependent.
func seedAdminProfileTestCredentials(t *testing.T, db *sql.DB) {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("changeme123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate admin profile fixture hash: %v", err)
	}
	if _, err := db.Exec(seedDeleteDuplicateAdminSQL, defaultAdminEmail, seededAdminID); err != nil {
		t.Fatalf("delete duplicate admin profile fixtures: %v", err)
	}
	if _, err := db.Exec(`UPDATE admin_users SET email = $1, password_hash = $2 WHERE id = $3`, defaultAdminEmail, string(passwordHash), seededAdminID); err != nil {
		t.Fatalf("seed admin profile credentials: %v", err)
	}
}

func TestHandleAdminProfile_ReturnsCurrentAdmin(t *testing.T) {
	configureAdminProfileTestCredentials(t)
	db := setupTestDB(t)
	seedAdminProfileTestCredentials(t, db)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	response, err := adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).GetAdminProfile(context.Background(), profileGetRequest(req))
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	profile := response.Msg.GetProfile()

	if profile.Email != defaultAdminEmail {
		t.Fatalf("expected email %s, got %s", defaultAdminEmail, profile.Email)
	}
	if !profile.IsDefaultEmail {
		t.Fatalf("expected default email flag to be true")
	}
	if !profile.IsDefaultPassword {
		t.Fatalf("expected default password flag to be true")
	}
}

func TestHandleAdminProfileUpdate_ChangesEmailAndPassword(t *testing.T) {
	configureAdminProfileTestCredentials(t)
	db := setupTestDB(t)
	seedAdminProfileTestCredentials(t, db)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr}

	replacer := strings.NewReplacer("/", "_", ".", "_")
	suffix := replacer.Replace(strings.ToLower(t.Name()))
	newEmail := fmt.Sprintf("owner-%s@test.com", suffix)
	if _, err := db.Exec(`DELETE FROM admin_users WHERE email = $1`, newEmail); err != nil {
		t.Fatalf("failed to cleanup admin user: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	response, err := adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).UpdateAdminProfile(context.Background(), profileUpdateRequest(req, &lpbsv1.UpdateAdminProfileRequest{CurrentPassword: "changeme123", NewEmail: newEmail, NewPassword: "Sup3rSecurePass!"}))
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	profile := response.Msg.GetProfile()

	if profile.Email != newEmail {
		t.Fatalf("expected updated email, got %s", profile.Email)
	}
	if profile.IsDefaultEmail {
		t.Fatalf("expected default email flag to be false")
	}
	if profile.IsDefaultPassword {
		t.Fatalf("expected default password flag to be false")
	}

	var storedEmail, storedHash string
	if err := db.QueryRow(`SELECT email, password_hash FROM admin_users WHERE email = $1`, newEmail).Scan(&storedEmail, &storedHash); err != nil {
		t.Fatalf("failed to load updated admin: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("Sup3rSecurePass!")); err != nil {
		t.Fatalf("stored password does not match new password: %v", err)
	}

	// Ensure the session now references the updated email
	sessionProbeReq := httptest.NewRequest(http.MethodGet, "/", nil)
	attachConnectSessionCookies(sessionProbeReq, response.Header())
	session, err := sessionMgr.GetSession(sessionProbeReq, "admin_session")
	if err != nil {
		t.Fatalf("read updated admin session: %v", err)
	}
	if session.Values["email"] != newEmail {
		t.Fatalf("session email not updated, got %v", session.Values["email"])
	}

	if err := seedDefaultData(db); err != nil {
		t.Fatalf("reseed defaults: %v", err)
	}
	if err := db.QueryRow(`SELECT email, password_hash FROM admin_users WHERE id = $1`, seededAdminID).Scan(&storedEmail, &storedHash); err != nil {
		t.Fatalf("load admin after reseed: %v", err)
	}
	if storedEmail != newEmail {
		t.Fatalf("reseed reset administrator email to %q, want %q", storedEmail, newEmail)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("Sup3rSecurePass!")); err != nil {
		t.Fatalf("reseed reset administrator password: %v", err)
	}
}

func TestHandleAdminProfileUpdate_InvalidPassword(t *testing.T) {
	configureAdminProfileTestCredentials(t)
	db := setupTestDB(t)
	seedAdminProfileTestCredentials(t, db)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr}

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	_, err := adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).UpdateAdminProfile(context.Background(), profileUpdateRequest(req, &lpbsv1.UpdateAdminProfileRequest{CurrentPassword: "wrongpass", NewPassword: "Sup3rSecurePass!"}))
	if connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("expected unauthenticated for invalid credentials, got %v", connect.CodeOf(err))
	}
}

func TestHandleAdminProfileUpdate_EmailConflict(t *testing.T) {
	configureAdminProfileTestCredentials(t)
	db := setupTestDB(t)
	seedAdminProfileTestCredentials(t, db)
	sessionMgr := initSessionManager()
	server := &Server{db: db, sessionManager: sessionMgr}

	// Use a unique suffix based on test name and timestamp for better isolation
	replacer := strings.NewReplacer("/", "_", ".", "_")
	suffix := replacer.Replace(strings.ToLower(t.Name()))
	timestamp := time.Now().UnixNano()
	takenEmail := fmt.Sprintf("taken-%s-%d@test.com", suffix, timestamp)

	// Clean up any existing entry and create a conflicting user
	// First, reset the sequence to avoid pkey conflicts from seed data
	if _, err := db.Exec(`SELECT setval('admin_users_id_seq', COALESCE((SELECT MAX(id) FROM admin_users), 0) + 1, false)`); err != nil {
		t.Logf("failed to reset sequence (might not exist): %v", err)
	}
	if _, err := db.Exec(`DELETE FROM admin_users WHERE email = $1`, takenEmail); err != nil {
		t.Fatalf("failed to cleanup conflicting admin: %v", err)
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("test-only-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("generate password hash: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)`, takenEmail, string(passwordHash)); err != nil {
		t.Fatalf("failed to seed conflicting admin: %v", err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec(`DELETE FROM admin_users WHERE email = $1`, takenEmail); err != nil {
			// Log but don't fail on cleanup
			t.Logf("cleanup failed: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	attachAdminSession(t, sessionMgr, req, defaultAdminEmail)
	_, err = adminhttp.NewProfileConnectHandler(server.adminProfileDependencies()).UpdateAdminProfile(context.Background(), profileUpdateRequest(req, &lpbsv1.UpdateAdminProfileRequest{CurrentPassword: "changeme123", NewEmail: takenEmail}))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("expected already exists for email conflict, got %v", connect.CodeOf(err))
	}
}
