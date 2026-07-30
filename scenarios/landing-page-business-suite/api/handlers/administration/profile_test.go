package administration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"landing-page-business-suite-api/internal/administration"
)

func TestProfileReturnsAuthenticatedAdministrator(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("not-the-default-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{"email": "admin@example.test"}
	service := &fakeProfileService{profile: administration.AdminProfile{ID: 7, Email: "admin@example.test", PasswordHash: string(hash)}}
	w := httptest.NewRecorder()

	Profile(profileDependencies(service, manager)).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/profile", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("content type=%q", got)
	}
	var response ProfileResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Email != "admin@example.test" || response.IsDefaultEmail || response.IsDefaultPassword {
		t.Fatalf("response=%+v", response)
	}
}

func TestValidateProfilePasswordRejectsConfiguredDefault(t *testing.T) {
	currentHash, err := bcrypt.GenerateFromPassword([]byte("current-password-123"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateProfilePassword("default-password-123", string(currentHash), "default-password-123"); err == nil {
		t.Fatal("expected configured default password to be rejected")
	}
}

type fakeProfileService struct{ profile administration.AdminProfile }

func (f *fakeProfileService) Profile(context.Context, string) (administration.AdminProfile, error) {
	return f.profile, nil
}

func (*fakeProfileService) EmailInUse(context.Context, string, int64) (bool, error) {
	return false, nil
}
func (*fakeProfileService) UpdateProfile(context.Context, int64, string, string) error { return nil }
func (*fakeProfileService) RevokeOtherSessions(context.Context, string, string) (int64, error) {
	return 0, nil
}

func profileDependencies(service ProfileService, manager SessionManager) ProfileDependencies {
	return ProfileDependencies{
		Auth: service, Sessions: manager,
		DefaultEmail:    func() string { return "default@example.test" },
		DefaultPassword: func() string { return "" },
		ValidateEmail:   func(string) error { return nil },
		WriteError:      func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
		Log:             func(string, map[string]any) {}, LogError: func(string, map[string]any) {},
	}
}
