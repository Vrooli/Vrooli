package administration

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
	admin "landing-page-business-suite-api/internal/administration"
)

type fakeAdminCredentialStore struct {
	profile      admin.AdminProfile
	emailInUse   bool
	updateCalls  int
	updatedEmail string
	updatedHash  string
}

func (f *fakeAdminCredentialStore) Profile(_ context.Context, email string) (admin.AdminProfile, error) {
	if !strings.EqualFold(strings.TrimSpace(email), f.profile.Email) {
		return admin.AdminProfile{}, sql.ErrNoRows
	}
	return f.profile, nil
}

func (f *fakeAdminCredentialStore) EmailInUse(context.Context, string, int64) (bool, error) {
	return f.emailInUse, nil
}

func (f *fakeAdminCredentialStore) UpdateProfile(_ context.Context, _ int64, email, hash string) error {
	f.updateCalls++
	f.updatedEmail = email
	f.updatedHash = hash
	return nil
}

type credentialResetCapture struct {
	authorityKey   string
	authorityValue string
	authorityCalls int
	revokedEmail   string
	revokeCalls    int
	events         []string
}

func newCredentialResetDeps(t *testing.T, store *fakeAdminCredentialStore, isService bool, capture *credentialResetCapture) AdminCredentialResetDependencies {
	t.Helper()
	return AdminCredentialResetDependencies{
		Auth: store,
		IsService: func(*http.Request) bool {
			return isService
		},
		DefaultPassword: func() string { return "" },
		ValidateEmail:   func(string) error { return nil },
		PutAuthority: func(key, value string) error {
			capture.authorityCalls++
			capture.authorityKey = key
			capture.authorityValue = value
			return nil
		},
		RevokeOtherSessions: func(_ context.Context, email, _ string) (int64, error) {
			capture.revokeCalls++
			capture.revokedEmail = email
			return 1, nil
		},
		SecurityEvents: fakeSecurityEvents{recorded: &capture.events},
		Notifier:       fakeSecurityNotifier{},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": message, "type": kind})
		},
		Log:      func(string, map[string]any) {},
		LogError: func(string, map[string]any) {},
	}
}

type fakeSecurityEvents struct{ recorded *[]string }

func (f fakeSecurityEvents) Record(_ context.Context, event, _ string, _ string, _ string, _ string, _ map[string]any) error {
	if f.recorded != nil {
		*f.recorded = append(*f.recorded, event)
	}
	return nil
}

type fakeSecurityNotifier struct{}

func (fakeSecurityNotifier) Notify(context.Context, string, string, string) error { return nil }

func credentialResetRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/admin-credentials/reset", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func newCredentialResetStore(t *testing.T, password string) *fakeAdminCredentialStore {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	return &fakeAdminCredentialStore{profile: admin.AdminProfile{ID: 1, Email: "admin@localhost", PasswordHash: string(hash)}}
}

func TestResetAdminCredentialRejectsBrowserSession(t *testing.T) {
	store := newCredentialResetStore(t, "OldSecret-12345")
	capture := &credentialResetCapture{}
	deps := newCredentialResetDeps(t, store, false, capture)
	recorder := httptest.NewRecorder()
	ResetAdminCredential(deps)(recorder, credentialResetRequest(t, `{"email":"admin@localhost","new_password":"NewSecret-12345"}`))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if store.updateCalls != 0 || capture.authorityCalls != 0 {
		t.Fatal("a browser session must not reset the administrator credential")
	}
}

func TestResetAdminCredentialUpdatesPasswordAndAuthority(t *testing.T) {
	store := newCredentialResetStore(t, "OldSecret-12345")
	capture := &credentialResetCapture{}
	deps := newCredentialResetDeps(t, store, true, capture)
	recorder := httptest.NewRecorder()
	ResetAdminCredential(deps)(recorder, credentialResetRequest(t, `{"email":"admin@localhost","new_password":"NewSecret-12345"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if store.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", store.updateCalls)
	}
	if bcrypt.CompareHashAndPassword([]byte(store.updatedHash), []byte("NewSecret-12345")) != nil {
		t.Fatal("stored hash does not match the new password")
	}
	if capture.authorityCalls != 1 || capture.authorityKey != "ADMIN_DEFAULT_PASSWORD" || capture.authorityValue != "NewSecret-12345" {
		t.Fatalf("authority write = %d %q %q", capture.authorityCalls, capture.authorityKey, capture.authorityValue)
	}
	if capture.revokeCalls != 1 || capture.revokedEmail != "admin@localhost" {
		t.Fatalf("session revoke = %d %q", capture.revokeCalls, capture.revokedEmail)
	}
	if len(capture.events) != 1 || capture.events[0] != "admin_credential_reset" {
		t.Fatalf("security events = %v", capture.events)
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response["password_changed"] != true || response["authority_updated"] != true || response["email"] != "admin@localhost" {
		t.Fatalf("response = %v", response)
	}
	if strings.Contains(recorder.Body.String(), "NewSecret-12345") {
		t.Fatal("response must never echo the new password")
	}
}

func TestResetAdminCredentialRejectsWeakPassword(t *testing.T) {
	store := newCredentialResetStore(t, "OldSecret-12345")
	capture := &credentialResetCapture{}
	deps := newCredentialResetDeps(t, store, true, capture)
	recorder := httptest.NewRecorder()
	ResetAdminCredential(deps)(recorder, credentialResetRequest(t, `{"email":"admin@localhost","new_password":"short"}`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	if store.updateCalls != 0 || capture.authorityCalls != 0 {
		t.Fatal("a rejected password must not mutate state")
	}
}

func TestResetAdminCredentialCanChangeEmail(t *testing.T) {
	store := newCredentialResetStore(t, "OldSecret-12345")
	capture := &credentialResetCapture{}
	deps := newCredentialResetDeps(t, store, true, capture)
	recorder := httptest.NewRecorder()
	ResetAdminCredential(deps)(recorder, credentialResetRequest(t, `{"email":"admin@localhost","new_email":"ops@example.com"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s, want 200", recorder.Code, recorder.Body.String())
	}
	if store.updatedEmail != "ops@example.com" {
		t.Fatalf("updated email = %q", store.updatedEmail)
	}
	if capture.authorityCalls != 0 {
		t.Fatal("an email-only change must not rewrite the password authority")
	}
}

func TestResetAdminCredentialUnknownEmail(t *testing.T) {
	store := newCredentialResetStore(t, "OldSecret-12345")
	capture := &credentialResetCapture{}
	deps := newCredentialResetDeps(t, store, true, capture)
	recorder := httptest.NewRecorder()
	ResetAdminCredential(deps)(recorder, credentialResetRequest(t, `{"email":"nobody@example.com","new_password":"NewSecret-12345"}`))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if store.updateCalls != 0 {
		t.Fatal("unknown account must not be created or updated")
	}
}
