package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginCreatesASevenDaySecureSession(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	service := &fakeAuth{hash: string(hash)}
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{}
	deps := testDependencies(service, manager)
	Login(deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(`{"email":"admin@example.test","password":"correct-password"}`)))
	if service.createdID != "session-1" || manager.session.Values["email"] != "admin@example.test" {
		t.Fatalf("createdID=%q values=%#v", service.createdID, manager.session.Values)
	}
	if manager.session.Options.MaxAge != 604800 || !manager.session.Options.Secure || manager.session.Options.SameSite != http.SameSiteLaxMode {
		t.Fatalf("options=%+v", manager.session.Options)
	}
}

func TestSessionReturnsUnauthenticatedResponseWhenCookieHasNoEmail(t *testing.T) {
	manager := &fakeSessions{session: sessions.NewSession(nil, sessionName)}
	manager.session.Values = map[any]any{}
	w := httptest.NewRecorder()
	Session(testDependencies(&fakeAuth{}, manager)).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
	var response SessionResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Authenticated {
		t.Fatalf("response=%+v", response)
	}
}

type fakeAuth struct{ hash, createdID string }

func (f *fakeAuth) PasswordHash(context.Context, string) (string, error) { return f.hash, nil }
func (f *fakeAuth) UpdateLastLogin(context.Context, string) error        { return nil }
func (f *fakeAuth) CreateSession(_ context.Context, id, _ string, _ time.Time, _, _ string) error {
	f.createdID = id
	return nil
}
func (f *fakeAuth) DeleteSession(context.Context, string) error { return nil }
func (f *fakeAuth) SessionExpiry(context.Context, string, string) (time.Time, error) {
	return time.Now().Add(time.Hour), nil
}
func (f *fakeAuth) TouchSession(context.Context, string) error { return nil }

type fakeSessions struct{ session *sessions.Session }

func (f *fakeSessions) GetSession(*http.Request, string) (*sessions.Session, error) {
	return f.session, nil
}

func (f *fakeSessions) SaveSession(*http.Request, http.ResponseWriter, *sessions.Session) error {
	return nil
}

func testDependencies(auth AuthService, sessions SessionManager) Dependencies {
	return Dependencies{Auth: auth, Sessions: sessions, GenerateID: func() (string, error) { return "session-1", nil }, Now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }, ClientIP: func(*http.Request) string { return "127.0.0.1" }, SecureCookies: func() bool { return true }, WriteError: func(w http.ResponseWriter, status int, message, kind string) { w.WriteHeader(status) }, Log: func(string, map[string]any) {}, LogError: func(string, map[string]any) {}}
}
