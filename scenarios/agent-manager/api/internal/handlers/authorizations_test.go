package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

type authorizationFake struct{ approvals int }

func (*authorizationFake) RequestAuthorization(context.Context, orchestration.AuthorizationRequest) (*domain.AuthorizationGrant, error) {
	return &domain.AuthorizationGrant{ID: uuid.New(), State: "pending"}, nil
}

func (*authorizationFake) GetAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error) {
	return &domain.AuthorizationGrant{State: "pending", Policy: "effort-metadata", Targets: []string{"agent-manager:effort/bas"}, DurationSeconds: 3600, RequestExpiresAt: time.Now().Add(time.Hour)}, nil
}

func (*authorizationFake) ListAuthorizations(context.Context, int) ([]*domain.AuthorizationGrant, error) {
	return nil, nil
}

func (f *authorizationFake) ApproveAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error) {
	f.approvals++
	return &domain.AuthorizationGrant{State: "approved"}, nil
}

func (*authorizationFake) RevokeAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error) {
	return nil, nil
}

func TestAuthorizationConsentRequiresIndependentHumanAndConfiguredOrigin(t *testing.T) {
	f := &authorizationFake{}
	router := mux.NewRouter()
	RegisterAuthorizations(router, f)
	path := "/api/v1/authorizations/" + uuid.NewString() + "/approve"
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, ExpiresAt: time.Now().Add(time.Hour)}
	invoke := func(human bool, origin, agent string) int {
		req := httptest.NewRequest(http.MethodPost, "http://backend.local"+path, strings.NewReader(""))
		if human {
			req = req.WithContext(identity.WithPrincipal(req.Context(), p))
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		if agent != "" {
			req.Header.Set("X-Agent-Identity-Token", agent)
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	if invoke(false, "http://backend.local", "") != http.StatusUnauthorized {
		t.Fatal("anonymous consent accepted")
	}
	if invoke(true, "http://backend.local", "agent") != http.StatusForbidden {
		t.Fatal("agent-bearing request approved its own authority")
	}
	if invoke(true, "https://attacker.example.test", "") != http.StatusForbidden {
		t.Fatal("cross-origin consent accepted")
	}
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	if invoke(true, "https://agent-manager.example.test", "") != http.StatusOK || f.approvals != 1 {
		t.Fatal("authenticated proxied browser consent failed")
	}
}

func TestAgentOperationBoundaryDoesNotAdmitGlobalOrAnonymousMutations(t *testing.T) {
	for _, kind := range []identity.ActorKind{identity.ActorUnknown, identity.ActorAgent} {
		for _, path := range []string{"/api/v1/profiles", "/api/v1/runs/stop-all", "/agent_manager.v1.AgentManagerService/UnknownMutation"} {
			req := httptest.NewRequest(http.MethodPost, path, nil)
			if kind == identity.ActorAgent {
				req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: kind, Subject: "run", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: &identity.OperationGrant{ID: "consent"}}))
			}
			rec := httptest.NewRecorder()
			AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden && rec.Code != http.StatusUnauthorized {
				t.Fatalf("kind=%s path=%s accepted global mutation", kind, path)
			}
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/agent_manager.v1.AgentManagerService/ListRuns", nil)
	rec := httptest.NewRecorder()
	AgentOperationMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(rec, req)
	if rec.Code != 204 {
		t.Fatal("existing read projection was treated as a mutation")
	}
}

func TestBrowserConsentShowsReadablePagesNotJSON(t *testing.T) {
	router := mux.NewRouter()
	RegisterAuthorizations(router, &authorizationFake{})
	path := "/api/v1/authorizations/" + uuid.NewString() + "/approve"
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, ExpiresAt: time.Now().Add(time.Hour)}
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	serve := func(method, accept string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "http://backend.local"+path, strings.NewReader(""))
		req = req.WithContext(identity.WithPrincipal(req.Context(), p))
		req.Header.Set("Origin", "https://agent-manager.example.test")
		req.Header.Set("Accept", accept)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	page := serve(http.MethodGet, "text/html")
	if !strings.Contains(page.Body.String(), "Confirm approve") {
		t.Fatalf("pending consent page offers no confirmation: %s", page.Body.String())
	}
	done := serve(http.MethodPost, "text/html,application/xhtml+xml")
	if done.Code != http.StatusOK || !strings.HasPrefix(done.Header().Get("Content-Type"), "text/html") || !strings.Contains(done.Body.String(), "approved") {
		t.Fatalf("browser approval did not render a confirmation page: %d %s", done.Code, done.Body.String())
	}
	api := serve(http.MethodPost, "application/json")
	if !strings.HasPrefix(api.Header().Get("Content-Type"), "application/json") {
		t.Fatal("a programmatic approval lost its JSON receipt")
	}
}
