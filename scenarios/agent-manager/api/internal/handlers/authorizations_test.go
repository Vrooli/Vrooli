package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
)

type authorizationFake struct {
	approvals        int
	policy           string
	bigPicture       string
	technicalDetails string
	request          *orchestration.AuthorizationRequest
}

func (f *authorizationFake) RequestAuthorization(_ context.Context, request orchestration.AuthorizationRequest) (*domain.AuthorizationGrant, error) {
	f.request = &request
	return &domain.AuthorizationGrant{ID: uuid.New(), State: "pending"}, nil
}

func (f *authorizationFake) GetAuthorization(context.Context, uuid.UUID) (*domain.AuthorizationGrant, error) {
	policy := f.policy
	if policy == "" {
		policy = "effort-metadata"
	}
	return &domain.AuthorizationGrant{State: "pending", Policy: policy, Targets: []string{"agent-manager:effort/bas"}, DurationSeconds: 3600, RequestExpiresAt: time.Now().Add(time.Hour), RequesterContext: &domain.AuthorizationRequesterContext{BigPicture: f.bigPicture, TechnicalDetails: f.technicalDetails}}, nil
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
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
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

func TestAgentOperationBoundaryDelegatesExactWorkflowStartToTriggerPolicy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		grant *identity.OperationGrant
	}{
		{name: "verified identity without a coarse grant"},
		{name: "unrelated interactive grant", grant: &identity.OperationGrant{ID: "unrelated", Rules: []identity.OperationRule{{Operations: []string{"effort.reconcile"}, Targets: []string{"agent-manager:effort/bas"}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			passed := 0
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				passed++
				w.WriteHeader(http.StatusNoContent)
			})
			principal := identity.Principal{Kind: identity.ActorAgent, Subject: "verified-run", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: tc.grant}

			invoke := func(method, path string) int {
				req := httptest.NewRequest(method, path, nil)
				req = req.WithContext(identity.WithPrincipal(req.Context(), principal))
				rec := httptest.NewRecorder()
				AgentOperationMiddleware(next).ServeHTTP(rec, req)
				return rec.Code
			}

			if status := invoke(http.MethodPost, "/api/v1/workflow-executions"); status != http.StatusNoContent || passed != 1 {
				t.Fatalf("exact workflow start did not reach its trigger-policy owner: status=%d passed=%d", status, passed)
			}
			for _, path := range []string{
				"/api/v1/workflow-executions/123/signals",
				"/api/v1/workflow-executions-extra",
			} {
				if status := invoke(http.MethodPost, path); status != http.StatusForbidden || passed != 1 {
					t.Fatalf("adjacent workflow mutation %s was admitted: status=%d passed=%d", path, status, passed)
				}
			}
		})
	}
}

func TestBrowserConsentShowsReadablePagesNotJSON(t *testing.T) {
	router := mux.NewRouter()
	RegisterAuthorizations(router, &authorizationFake{})
	path := "/api/v1/authorizations/" + uuid.NewString() + "/approve"
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
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

func TestBrowserConsentListsMissingCapabilitiesBeforeConfirmation(t *testing.T) {
	f := &authorizationFake{policy: "team-preparation"}
	router := mux.NewRouter()
	RegisterAuthorizations(router, f)
	path := "/api/v1/authorizations/" + uuid.NewString() + "/approve"
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(identity.WithPrincipal(req.Context(), p))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Approval access needs setup") || !strings.Contains(body, "prompt-manager:read") || !strings.Contains(body, "prompt-manager:write") || strings.Contains(body, "Confirm approve") || strings.Contains(body, "can no longer") || f.approvals != 0 {
		t.Fatalf("preflight is misleading or authoritative: %d %s", rec.Code, body)
	}
	p.Scopes = append(p.Scopes, "prompt-manager:read", "prompt-manager:write")
	req = req.WithContext(identity.WithPrincipal(req.Context(), p))
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "Confirm approve") || f.approvals != 0 {
		t.Fatal("eligible pending request lacks independent confirmation")
	}
}

func TestBrowserConsentRendersRequesterContextAsUntrustedEscapedText(t *testing.T) {
	f := &authorizationFake{bigPicture: "Reconcile <backlog> & keep the run scoped", technicalDetails: "First line\n<script>alert(1)</script> & second line"}
	router := mux.NewRouter()
	RegisterAuthorizations(router, f)
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	p := identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Source: identity.SourceCloudflareAccess, Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour)}
	path := "/api/v1/authorizations/" + uuid.NewString() + "/approve"
	req := httptest.NewRequest(http.MethodGet, "http://backend.local"+path, nil)
	req = req.WithContext(identity.WithPrincipal(req.Context(), p))
	req.Header.Set("Origin", "https://agent-manager.example.test")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	body := rec.Body.String()
	bigPictureAt := strings.Index(body, `<details open><summary>Big picture</summary>`)
	technicalDetailsAt := strings.Index(body, `<details open><summary>Technical details</summary>`)
	if rec.Code != http.StatusOK || bigPictureAt < 0 || technicalDetailsAt < 0 || bigPictureAt >= technicalDetailsAt {
		t.Fatalf("consent page lacks two ordered native disclosure controls: status=%d body=%s", rec.Code, body)
	}
	bigPictureEnd := strings.Index(body[bigPictureAt:], `</details>`)
	technicalDetailsEnd := strings.Index(body[technicalDetailsAt:], `</details>`)
	if bigPictureEnd < 0 || technicalDetailsEnd < 0 || strings.Contains(body, `<details open name=`) || !strings.Contains(body[bigPictureAt:bigPictureAt+bigPictureEnd], "&lt;backlog&gt; &amp;") || !strings.Contains(body[technicalDetailsAt:technicalDetailsAt+technicalDetailsEnd], "&lt;script&gt;alert(1)&lt;/script&gt;") || !strings.Contains(body, "Requester-provided explanation") || !strings.Contains(body, "Requester-provided context") || strings.Contains(body, "<script>alert(1)</script>") {
		t.Fatalf("consent page failed to safely display independent requester disclosures: status=%d body=%s", rec.Code, body)
	}

	f.bigPicture, f.technicalDetails = "", ""
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if !strings.Contains(rec.Body.String(), "No additional requester explanation was supplied") {
		t.Fatal("legacy request without explanation lacks honest fallback")
	}
}

func TestAuthorizationRequestAcceptsExistingRequesterContextCodePointLimits(t *testing.T) {
	f := &authorizationFake{}
	router := mux.NewRouter()
	RegisterAuthorizations(router, f)
	body, err := json.Marshal(orchestration.AuthorizationRequest{
		PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "context-body",
		DurationSeconds: 3600, Policy: "swarm-bookkeeping", Targets: []string{"swarm-manager:backlog/item"},
		BigPicture: strings.Repeat("界", 4000), TechnicalDetails: strings.Repeat("界", 12000),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api/v1/authorizations/requests", strings.NewReader(string(body)))
	req.RemoteAddr = "127.0.0.1:43123"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated || f.request == nil || f.request.BigPicture != strings.Repeat("界", 4000) || f.request.TechnicalDetails != strings.Repeat("界", 12000) {
		t.Fatalf("valid bounded Unicode context was rejected or changed: status=%d request=%+v body=%s", rec.Code, f.request, rec.Body.String())
	}
}
