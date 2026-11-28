package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
	"web-console/internal/sessionstore"
)

type notificationScopeStore struct {
	sessionstore.Store
	owners map[string]string
}

func (s notificationScopeStore) Get(_ context.Context, id string) (sessionstore.Metadata, error) {
	owner, ok := s.owners[id]
	if !ok {
		return sessionstore.Metadata{}, errors.New("not found")
	}
	return sessionstore.Metadata{ID: id, Owner: owner}, nil
}

func notificationOwnerRequest(req *http.Request, owner string) *http.Request {
	p := identity.Principal{Kind: identity.ActorHuman, Subject: owner, Verified: true, Source: identity.SourcePersonalLocal, ExpiresAt: time.Now().Add(time.Hour)}
	return req.WithContext(identity.WithPrincipal(req.Context(), p))
}

func TestNotificationsProxyRejectsExpiredOwnerForScopedWrite(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { called = true }))
	defer upstream.Close()
	store := notificationScopeStore{owners: map[string]string{"mine": "operator"}}
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil }, store)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", strings.NewReader(`{"category":"session.input_needed","scope_key":"mine","enabled":false}`))
	expired := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Source: identity.SourcePersonalLocal, ExpiresAt: time.Now().Add(-time.Minute)}
	req = req.WithContext(identity.WithPrincipal(req.Context(), expired))
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d called=%t body=%q", rec.Code, called, rec.Body.String())
	}
}

func TestNotificationsProxyScopedWritesRequireOwnedSession(t *testing.T) {
	for _, tc := range []struct {
		name, scope, wantOwner string
		status                 int
	}{
		{"owned", "mine", "operator", http.StatusOK},
		{"cross-owner", "theirs", "someone-else", http.StatusForbidden},
		{"nonexistent", "missing", "", http.StatusForbidden},
		{"whitespace-scope", "   ", "", http.StatusForbidden},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(http.StatusOK) }))
			defer upstream.Close()
			store := notificationScopeStore{owners: map[string]string{"mine": "operator", "theirs": "someone-else"}}
			proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil }, store)
			req := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences", strings.NewReader(`{"source_scenario":"web-console","category":"session.input_needed","scope_key":"`+tc.scope+`","owner":"operator","enabled":false}`))
			req = notificationOwnerRequest(req, "operator")
			rec := httptest.NewRecorder()
			proxy.ServeHTTP(rec, req)
			if rec.Code != tc.status || called != (tc.status == http.StatusOK) {
				t.Fatalf("status=%d called=%t body=%q", rec.Code, called, rec.Body.String())
			}
		})
	}
}

func TestNotificationsProxyFiltersScopedReadsBySessionOwner(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"preferences":[{"source_scenario":"web-console","scope_key":"","category":"global"},{"source_scenario":"web-console","scope_key":"mine","category":"mine"},{"source_scenario":"web-console","scope_key":"theirs","category":"theirs"},{"source_scenario":"web-console","scope_key":"missing","category":"missing"}]}`)
	}))
	defer upstream.Close()
	store := notificationScopeStore{owners: map[string]string{"mine": "operator", "theirs": "someone-else"}}
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil }, store)
	req := notificationOwnerRequest(httptest.NewRequest(http.MethodGet, "/api/v1/notifications/preferences", nil), "operator")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	for _, want := range []string{`"category":"global"`, `"category":"mine"`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("missing %s in %s", want, rec.Body.String())
		}
	}
	for _, forbidden := range []string{`"category":"theirs"`, `"category":"missing"`} {
		if strings.Contains(rec.Body.String(), forbidden) {
			t.Fatalf("leaked %s in %s", forbidden, rec.Body.String())
		}
	}
}

func TestNotificationsProxyForwardsCatalogGET(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/notifications/catalog" {
			t.Fatalf("method=%q path=%q", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"sources":[],"preferences":[]}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/catalog", nil)
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "preferences") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

// A caller cannot widen Web Console's source-scoped settings view by changing
// the query parameter sent to the facade.
func TestNotificationsProxyPinsSettingsReadSourceFilter(t *testing.T) {
	var gotSources []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSources = r.URL.Query()["source_scenario"]
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"sources":[],"preferences":[]}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	for _, path := range []string{
		"/api/v1/notifications/catalog?source_scenario=nutrition-planner&source_scenario=other",
		"/api/v1/notifications/preferences?source_scenario=nutrition-planner",
	} {
		record := httptest.NewRecorder()
		proxy.ServeHTTP(record, httptest.NewRequest(http.MethodGet, path, nil))
		if record.Code != http.StatusOK || len(gotSources) != 1 || gotSources[0] != "web-console" {
			t.Fatalf("path=%q status=%d forwarded source_scenario=%v", path, record.Code, gotSources)
		}
	}
}

func TestNotificationsProxyForwardsPUTBodyAndAuthorization(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/v1/notifications/preferences" {
			t.Fatalf("method=%q path=%q", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer owner-token" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"source_scenario":"web-console"`) {
			t.Fatalf("body=%q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences",
		strings.NewReader(`{"source_scenario":"web-console","category":"sessions","enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer owner-token")
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusOK || !strings.Contains(record.Body.String(), "ok") {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
}

func TestNotificationsProxyRejectsCrossSourcePreferenceWrite(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/notifications/preferences",
		strings.NewReader(`{"source_scenario":"nutrition-planner","category":"meals","enabled":true}`))
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusForbidden || called {
		t.Fatalf("status=%d upstream_called=%t body=%q", record.Code, called, record.Body.String())
	}
}

func TestNotificationsProxyForwardsCloudflareAccessAssertion(t *testing.T) {
	var sawAccess, sawAccessWhenAbsent string
	present := true
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if present {
			sawAccess = r.Header.Get("Cf-Access-Jwt-Assertion")
		} else {
			sawAccessWhenAbsent = r.Header.Get("Cf-Access-Jwt-Assertion")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	proxy := newNotificationsProxy(func(context.Context) (string, error) { return upstream.URL, nil })

	// With the assertion present, it is forwarded so the hub's human-proof path
	// attributes the answer to the actual signed-in human.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/answer", strings.NewReader(`{}`))
	req.Header.Set("Cf-Access-Jwt-Assertion", "eyJhbGciOiJSUzI1NiJ9.payload.sig")
	proxy.ServeHTTP(httptest.NewRecorder(), req)
	if sawAccess != "eyJhbGciOiJSUzI1NiJ9.payload.sig" {
		t.Fatalf("Cf-Access-Jwt-Assertion was not forwarded: %q", sawAccess)
	}

	// Without it (bearer/cookie path), the header is simply omitted.
	present = false
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/answer", strings.NewReader(`{}`))
	proxy.ServeHTTP(httptest.NewRecorder(), req2)
	if sawAccessWhenAbsent != "" {
		t.Fatalf("Cf-Access-Jwt-Assertion must be omitted when absent, got %q", sawAccessWhenAbsent)
	}
}

func TestNotificationsProxyUpstreamDownReturns503(t *testing.T) {
	proxy := newNotificationsProxy(func(context.Context) (string, error) {
		return "", errors.New("notification-hub not discovered")
	})
	record := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/catalog", nil)
	proxy.ServeHTTP(record, request)
	if record.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%q", record.Code, record.Body.String())
	}
	if !strings.Contains(record.Body.String(), "unavailable") {
		t.Fatalf("expected JSON error, got %q", record.Body.String())
	}
}
