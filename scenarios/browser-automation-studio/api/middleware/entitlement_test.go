package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/config"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

type identitySettings struct{ value string }

func (s identitySettings) GetSetting(context.Context, string) (string, error) { return s.value, nil }

func TestInjectEntitlementUsesStoredIdentityOnlyForOfflineLookup(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	service := entitlement.NewService(config.EntitlementConfig{}, logrus.New())
	middleware := NewEntitlementMiddleware(service, logrus.New(), identitySettings{value: " Alice@Example.com "})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai", nil)
	req.Header.Set("Authorization", "Bearer short-lived-token")
	req.Header.Set("X-User-Identity", "attacker@example.com")
	recorder := httptest.NewRecorder()
	called := false
	middleware.InjectEntitlement(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if got := monetization.AccessTokenFromContext(r.Context()); got != "short-lived-token" {
			t.Errorf("shared access token = %q", got)
		}
		if got := monetization.UserIdentityFromContext(r.Context()); got != "alice@example.com" {
			t.Errorf("identity hint = %q", got)
		}
		if monetization.SnapshotFromContext(r.Context()) != nil {
			t.Error("unavailable LPBS authority unexpectedly injected an entitlement")
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(recorder, req)
	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("middleware called=%v status=%d", called, recorder.Code)
	}
}

func TestInjectEntitlementResolvesIdentityFromLPBSForRemoteLease(t *testing.T) {
	t.Setenv("APP_DATA_DIR", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/auth/me":
			if r.Header.Get("Authorization") != "Bearer short-lived-token" {
				t.Errorf("authorization = %q", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"user":{"email":"Verified@Example.com"}}`))
		case "/api/v1/entitlements":
			if got := r.URL.Query().Get("user"); got != "verified@example.com" {
				t.Errorf("requested identity = %q", got)
			}
			http.Error(w, "fake authority unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	service := entitlement.NewService(config.EntitlementConfig{ServiceURL: server.URL}, logrus.New())
	middleware := NewEntitlementMiddleware(service, logrus.New(), identitySettings{value: "attacker@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ai", nil)
	req.Header.Set("Authorization", "Bearer short-lived-token")
	req.Header.Set("X-User-Identity", "attacker@example.com")
	recorder := httptest.NewRecorder()
	middleware.InjectEntitlement(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := monetization.UserIdentityFromContext(r.Context()); got != "verified@example.com" {
			t.Errorf("verified identity = %q", got)
		}
		if monetization.SnapshotFromContext(r.Context()) != nil {
			t.Error("unavailable entitlement authority unexpectedly injected a lease")
		}
	})).ServeHTTP(recorder, req)
}
