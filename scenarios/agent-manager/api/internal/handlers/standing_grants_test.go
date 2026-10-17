package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	requestidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
)

// fakeGrantAdmin records whether a write reached the service. The security
// property under test is that only a verified independent human gets here.
type fakeGrantAdmin struct {
	created *orchestration.StandingGrantRequest
	revoked string
}

func (f *fakeGrantAdmin) CreateStandingGrant(_ context.Context, req orchestration.StandingGrantRequest) (*domain.StandingGrant, error) {
	f.created = &req
	return &domain.StandingGrant{
		ID: "sg-1", Subject: req.Subject, Limits: req.Limits,
		Grant:     &requestidentity.OperationGrant{ID: "sg-1", Approver: req.Approver, Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(req.TTL)},
		ExpiresAt: time.Now().Add(req.TTL),
	}, nil
}

func (f *fakeGrantAdmin) RevokeStandingGrant(_ context.Context, id string) (bool, error) {
	f.revoked = id
	return true, nil
}

func (f *fakeGrantAdmin) ListStandingGrants(context.Context, int) ([]*domain.StandingGrant, error) {
	return nil, nil
}

// grantAdminRouter builds the production middleware chain in front of the
// standing-grant admin routes, with no standing-grant authority installed (the
// admin endpoint is human-only and must not depend on one).
func grantAdminRouter(admin StandingGrantAdmin) http.Handler {
	r := mux.NewRouter()
	r.Use(AgentOperationMiddleware)
	RegisterStandingGrants(r, admin)
	return r
}

func postGrantCreate(h http.Handler, p *identity.Principal) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodPost, "/api/v1/standing-grants", bytes.NewReader([]byte(`{"subject":"prompt-manager","max_concurrent_runs":3,"ttl_seconds":3600}`)))
	if p != nil {
		r = r.WithContext(identity.WithPrincipal(r.Context(), *p))
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestStandingGrantCreateAllowsVerifiedHumanOperator(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}
	w := postGrantCreate(h, &human)
	if w.Code != http.StatusCreated {
		t.Fatalf("verified human operator refused: %d (%s)", w.Code, w.Body.String())
	}
	if admin.created == nil || admin.created.Subject != "prompt-manager" {
		t.Fatalf("create did not reach the service with the subject: %+v", admin.created)
	}
	// The approver is the verified human — never from the body.
	if admin.created.Approver != "operator" {
		t.Fatalf("approver was not the verified human: %q", admin.created.Approver)
	}
}

func TestStandingGrantCreateRefusesAgent(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	// An agent, even one carrying a grant, must never create a standing grant.
	agent := identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "operator", Verified: true, Scopes: []string{"agent-manager:write"}, Grant: &identity.OperationGrant{ID: "g", Approver: "operator", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: time.Now().Add(time.Hour)}}
	w := postGrantCreate(h, &agent)
	if w.Code == http.StatusCreated || admin.created != nil {
		t.Fatalf("agent created a standing grant (code %d, created %+v)", w.Code, admin.created)
	}
}

// A human principal that ALSO carries agent provenance (a forwarded agent
// credential) must not be able to write a standing grant: RequireIndependentHuman
// rejects the request so the approver can never be a forwarded agent borrowing a
// human's presence. This is the subtle escalation path the loop must close.
func TestStandingGrantCreateRefusesHumanWithAgentProvenance(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/standing-grants", bytes.NewReader([]byte(`{"subject":"prompt-manager","max_concurrent_runs":3,"ttl_seconds":3600}`)))
	ctx := identity.WithPrincipal(r.Context(), human)
	ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "worker"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r.WithContext(ctx))
	if w.Code == http.StatusCreated || admin.created != nil {
		t.Fatalf("human-with-agent-provenance created a standing grant (code %d, created %+v)", w.Code, admin.created)
	}
}

func TestStandingGrantCreateRefusesService(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	// A service principal must never create its own standing grant.
	service := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	w := postGrantCreate(h, &service)
	if w.Code == http.StatusCreated || admin.created != nil {
		t.Fatalf("service created a standing grant (code %d, created %+v)", w.Code, admin.created)
	}
}

func TestStandingGrantCreateRefusesAnonymous(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	w := postGrantCreate(h, nil)
	if w.Code == http.StatusCreated || admin.created != nil {
		t.Fatalf("anonymous created a standing grant (code %d)", w.Code)
	}
}

func TestStandingGrantRevokeIsHumanOnly(t *testing.T) {
	for _, tc := range []struct {
		name  string
		p     *identity.Principal
		human bool
	}{
		{"human", &identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}, true},
		{"agent", &identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Scopes: []string{"agent-manager:write"}}, false},
		{"service", &identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admin := &fakeGrantAdmin{}
			h := grantAdminRouter(admin)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/standing-grants/sg-1/revoke", nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), *tc.p))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if tc.human {
				if w.Code != http.StatusOK || admin.revoked != "sg-1" {
					t.Fatalf("human revoke failed: code %d revoked %q", w.Code, admin.revoked)
				}
			} else if w.Code == http.StatusOK || admin.revoked != "" {
				t.Fatalf("%s reached revoke (code %d revoked %q)", tc.name, w.Code, admin.revoked)
			}
		})
	}
}
