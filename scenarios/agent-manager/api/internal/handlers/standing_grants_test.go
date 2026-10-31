package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	created    *orchestration.StandingGrantRequest
	revoked    string
	grants     []*domain.StandingGrant
	listedFor  string
	listCalls  int
	listLimit  int
	ownerLimit int
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

func (f *fakeGrantAdmin) ListStandingGrants(_ context.Context, limit int) ([]*domain.StandingGrant, error) {
	f.listCalls++
	f.listLimit = limit
	return grantPage(f.grants, limit), nil
}

func (f *fakeGrantAdmin) ListStandingGrantsForApprover(_ context.Context, approver string, limit int) ([]*domain.StandingGrant, error) {
	f.listedFor = approver
	f.ownerLimit = limit
	var grants []*domain.StandingGrant
	for _, grant := range f.grants {
		if grant != nil && grant.ApprovedBy == approver && grant.Grant != nil && grant.Grant.Approver == approver {
			grants = append(grants, grant)
		}
	}
	return grantPage(grants, limit), nil
}

func grantPage(grants []*domain.StandingGrant, limit int) []*domain.StandingGrant {
	if limit > 0 && len(grants) > limit {
		return grants[:limit]
	}
	return grants
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

func TestStandingGrantListScopesVerifiedAgentToOwner(t *testing.T) {
	admin := &fakeGrantAdmin{grants: []*domain.StandingGrant{
		{ID: "other-owner", Subject: "prompt-manager", ApprovedBy: "other-owner", Grant: &requestidentity.OperationGrant{Approver: "other-owner", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
		{ID: "owner-with-mismatched-grant-approver", Subject: "prompt-manager", ApprovedBy: "owner-1", Grant: &requestidentity.OperationGrant{Approver: "owner-2", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
		{ID: "owner-grant", Subject: "prompt-manager", ApprovedBy: "owner-1", Grant: &requestidentity.OperationGrant{Approver: "owner-1", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}}},
	}}
	h := grantAdminRouter(admin)
	// An attached run's verified owner identity is sufficient for this scoped
	// diagnostic read; attachment intentionally carries no coarse scopes.
	agent := identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Verified: true}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/standing-grants", nil)
	r = r.WithContext(identity.WithPrincipal(r.Context(), agent))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("verified agent read refused: %d (%s)", w.Code, w.Body.String())
	}
	if admin.listedFor != "owner-1" || admin.ownerLimit != standingGrantReadFetchLimit {
		t.Fatalf("agent listing was not owner-filtered and bounded: owner=%q limit=%d", admin.listedFor, admin.ownerLimit)
	}
	var response struct {
		Grants []map[string]any `json:"grants"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Grants) != 1 || response.Grants[0]["id"] != "owner-grant" {
		t.Fatalf("agent saw grants outside its verified owner or mismatched attribution: %+v", response.Grants)
	}
}

func TestStandingGrantListRefusesUnownedOrUnprivilegedNonHuman(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    *identity.Principal
	}{
		{name: "agent without owner", p: &identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Verified: true, Scopes: []string{"agent-manager:read"}}},
		{name: "human without read", p: &identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}}},
		{name: "unverified agent", p: &identity.Principal{Kind: identity.ActorAgent, Subject: "run-1", Owner: "owner-1", Scopes: []string{"agent-manager:read"}}},
		{name: "service", p: &identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:read"}}},
		{name: "anonymous"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admin := &fakeGrantAdmin{}
			h := grantAdminRouter(admin)
			r := httptest.NewRequest(http.MethodGet, "/api/v1/standing-grants", nil)
			if tc.p != nil {
				r = r.WithContext(identity.WithPrincipal(r.Context(), *tc.p))
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code == http.StatusOK || admin.listedFor != "" || admin.listCalls != 0 {
				t.Fatalf("%s reached list (%d, owner filter %q, global calls %d)", tc.name, w.Code, admin.listedFor, admin.listCalls)
			}
		})
	}
}

func TestStandingGrantListRefusesForwardedAgentHumanContext(t *testing.T) {
	admin := &fakeGrantAdmin{}
	h := grantAdminRouter(admin)
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:read"}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/standing-grants", nil)
	ctx := identity.WithPrincipal(r.Context(), human)
	ctx = provenance.NewContext(ctx, provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "worker"})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r.WithContext(ctx))
	if w.Code == http.StatusOK || admin.listCalls != 0 || admin.listedFor != "" {
		t.Fatalf("human with forwarded agent provenance reached global list: %d", w.Code)
	}
}

func TestStandingGrantListPreservesHumanGlobalAccess(t *testing.T) {
	admin := &fakeGrantAdmin{grants: []*domain.StandingGrant{
		{ID: "owner-1", ApprovedBy: "owner-1", Grant: &requestidentity.OperationGrant{Approver: "owner-1"}},
		{ID: "owner-2", ApprovedBy: "owner-2", Grant: &requestidentity.OperationGrant{Approver: "owner-2"}},
	}}
	h := grantAdminRouter(admin)
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:read"}}
	r := httptest.NewRequest(http.MethodGet, "/api/v1/standing-grants", nil)
	r = r.WithContext(identity.WithPrincipal(r.Context(), human))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("verified human read refused: %d (%s)", w.Code, w.Body.String())
	}
	var response struct {
		Grants []map[string]any `json:"grants"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Grants) != 2 || admin.listedFor != "" || admin.listLimit != standingGrantReadFetchLimit {
		t.Fatalf("human global listing changed: grants=%+v ownerFilter=%q limit=%d", response.Grants, admin.listedFor, admin.listLimit)
	}
}

func TestStandingGrantListReportsMoreThanResponseLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		kind    identity.ActorKind
		subject string
		owner   string
	}{
		{name: "owned agent", kind: identity.ActorAgent, subject: "run-1", owner: "owner-1"},
		{name: "global human", kind: identity.ActorHuman, subject: "operator"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			admin := &fakeGrantAdmin{}
			for i := 0; i < standingGrantReadLimit+1; i++ {
				admin.grants = append(admin.grants, &domain.StandingGrant{
					ID: fmt.Sprintf("grant-%02d", i), Subject: "prompt-manager", ApprovedBy: "owner-1",
					Grant: &requestidentity.OperationGrant{Approver: "owner-1", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}},
				})
			}
			h := grantAdminRouter(admin)
			principal := identity.Principal{Kind: tc.kind, Subject: tc.subject, Owner: tc.owner, Verified: true, Scopes: []string{"agent-manager:read"}}
			r := httptest.NewRequest(http.MethodGet, "/api/v1/standing-grants", nil)
			r = r.WithContext(identity.WithPrincipal(r.Context(), principal))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != http.StatusOK {
				t.Fatalf("list refused: %d (%s)", w.Code, w.Body.String())
			}
			var response struct {
				Grants  []map[string]any `json:"grants"`
				HasMore bool             `json:"has_more"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if len(response.Grants) != standingGrantReadLimit || !response.HasMore {
				t.Fatalf("truncated listing is not reported accurately: grants=%d has_more=%t", len(response.Grants), response.HasMore)
			}
		})
	}
}
