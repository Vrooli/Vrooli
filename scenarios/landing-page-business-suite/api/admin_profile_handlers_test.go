package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func attachAdminSession(t *testing.T, req *http.Request, email string) {
	t.Helper()
	session, _ := sessionStore.Get(req, "admin_session")
	session.Values["email"] = email
	rr := httptest.NewRecorder()
	if err := session.Save(req, rr); err != nil {
		t.Fatalf("failed to save admin session: %v", err)
	}
	for _, cookie := range rr.Result().Cookies() {
		req.AddCookie(cookie)
	}
}

func TestHandleAdminProfile_ReturnsCurrentAdmin(t *testing.T) {
	db := setupTestDB(t)
	initSessionStore()
	server := &Server{db: db}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil)
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(server.handleAdminProfile)(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	var profile AdminProfileResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &profile); err != nil {
		t.Fatalf("invalid profile json: %v", err)
	}

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
	db := setupTestDB(t)
	initSessionStore()
	server := &Server{db: db}

	payload := `{"current_password":"changeme123","new_email":"owner@test.com","new_password":"Sup3rSecurePass!"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/profile", bytes.NewBufferString(payload))
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(server.handleAdminProfileUpdate)(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	var profile AdminProfileResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &profile); err != nil {
		t.Fatalf("invalid profile json: %v", err)
	}

	if profile.Email != "owner@test.com" {
		t.Fatalf("expected updated email, got %s", profile.Email)
	}
	if profile.IsDefaultEmail {
		t.Fatalf("expected default email flag to be false")
	}
	if profile.IsDefaultPassword {
		t.Fatalf("expected default password flag to be false")
	}

	var storedEmail, storedHash string
	if err := db.QueryRow(`SELECT email, password_hash FROM admin_users WHERE email = $1`, "owner@test.com").Scan(&storedEmail, &storedHash); err != nil {
		t.Fatalf("failed to load updated admin: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte("Sup3rSecurePass!")); err != nil {
		t.Fatalf("stored password does not match new password: %v", err)
	}

	// Ensure the session now references the updated email
	sessionProbeReq := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Result().Cookies() {
		sessionProbeReq.AddCookie(cookie)
	}
	session, _ := sessionStore.Get(sessionProbeReq, "admin_session")
	if session.Values["email"] != "owner@test.com" {
		t.Fatalf("session email not updated, got %v", session.Values["email"])
	}
}

func TestHandleAdminProfileUpdate_InvalidPassword(t *testing.T) {
	db := setupTestDB(t)
	initSessionStore()
	server := &Server{db: db}

	payload := `{"current_password":"wrongpass","new_password":"Sup3rSecurePass!"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/profile", bytes.NewBufferString(payload))
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(server.handleAdminProfileUpdate)(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid credentials, got %d", resp.Code)
	}
}

func TestHandleAdminProfileUpdate_EmailConflict(t *testing.T) {
	db := setupTestDB(t)
	initSessionStore()
	server := &Server{db: db}

	if _, err := db.Exec(`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)`, "taken@test.com", defaultAdminPasswordHash); err != nil {
		t.Fatalf("failed to seed conflicting admin: %v", err)
	}

	payload := `{"current_password":"changeme123","new_email":"taken@test.com"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/profile", bytes.NewBufferString(payload))
	attachAdminSession(t, req, defaultAdminEmail)
	resp := httptest.NewRecorder()

	server.requireAdmin(server.handleAdminProfileUpdate)(resp, req)

	if resp.Code != http.StatusConflict {
		t.Fatalf("expected 409 for email conflict, got %d", resp.Code)
	}
}
