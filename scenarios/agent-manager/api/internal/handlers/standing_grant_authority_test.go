package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// memStandingGrants is an in-memory StandingGrantRepository for middleware tests.
// It needs no operator and no database.
type memStandingGrants struct {
	bySubject map[string][]*domain.StandingGrant
	err       error
}

func (m *memStandingGrants) Put(_ context.Context, g *domain.StandingGrant) error {
	if m.bySubject == nil {
		m.bySubject = map[string][]*domain.StandingGrant{}
	}
	m.bySubject[g.Subject] = append(m.bySubject[g.Subject], g)
	return nil
}

func (m *memStandingGrants) ActiveForSubject(_ context.Context, subject string, now time.Time) ([]*domain.StandingGrant, error) {
	if m.err != nil {
		return nil, m.err
	}
	var out []*domain.StandingGrant
	for _, g := range m.bySubject[subject] {
		if g.Active(now) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (m *memStandingGrants) Revoke(_ context.Context, id string, now time.Time) (bool, error) {
	for _, grants := range m.bySubject {
		for _, g := range grants {
			if g.ID == id && g.State == domain.StandingGrantApproved {
				g.State = domain.StandingGrantRevoked
				g.RevokedAt = &now
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *memStandingGrants) List(_ context.Context, _ int) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, grants := range m.bySubject {
		out = append(out, grants...)
	}
	return out, nil
}

func (m *memStandingGrants) ListForApprover(_ context.Context, approver string, _ int) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, grants := range m.bySubject {
		for _, grant := range grants {
			if grant != nil && grant.ApprovedBy == approver && grant.Grant != nil && grant.Grant.Approver == approver {
				out = append(out, grant)
			}
		}
	}
	return out, nil
}

type fakeUsage struct {
	concurrent int
	err        error
}

func (f fakeUsage) StandingGrantUsage(context.Context, string) (authn.StandingGrantUsage, error) {
	if f.err != nil {
		return authn.StandingGrantUsage{}, f.err
	}
	return authn.StandingGrantUsage{ConcurrentRuns: f.concurrent}, nil
}

// fakeRaiser records needs-approval raises so a test can assert the "ask, don't
// block" path fired. flood simulates the per-subject cap.
type fakeRaiser struct {
	raised   []orchestration.AuthorizationApprovalRequest
	flood    bool
	err      error
	raisedOK bool
}

func (f *fakeRaiser) RaiseAuthorizationApproval(_ context.Context, req orchestration.AuthorizationApprovalRequest) (bool, error) {
	if f.flood {
		return false, orchestration.ErrAuthorizationFlood
	}
	if f.err != nil {
		return false, f.err
	}
	f.raised = append(f.raised, req)
	if !f.raisedOK {
		return true, nil
	}
	return false, nil
}

func standingGrantRecord(subject, owner string, exp time.Time, limits authn.StandingGrantLimits) *domain.StandingGrant {
	return &domain.StandingGrant{
		ID:      "sg-" + subject,
		Subject: subject,
		Grant: &identity.OperationGrant{
			ID: "sg-" + subject, Approver: owner, Policy: "run-creation",
			Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: exp,
		},
		Limits:     limits,
		State:      domain.StandingGrantApproved,
		ApprovedBy: owner,
		ApprovedAt: time.Now().Add(-time.Minute),
		ExpiresAt:  exp,
	}
}

// serviceChain drives the production middleware chain for a verified service
// principal against POST /api/v1/runs with a raiser wired (the production shape).
func serviceChain(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader) int {
	return serviceChainFull(t, repo, usage, &fakeRaiser{}, "/api/v1/runs")
}

// serviceChainPath is serviceChain against an arbitrary path.
func serviceChainPath(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader, path string) int {
	return serviceChainFull(t, repo, usage, &fakeRaiser{}, path)
}

// admitAdmittedStatus is the status the final (post-middleware) handler writes
// when the chain ADMITS the request. It is deliberately NOT 202, so an admit is
// distinguishable from the middleware's 202 "approval pending" short-circuit.
const admitAdmittedStatus = http.StatusOK

// serviceChainFull drives the chain with an explicit raiser so a test can assert
// the needs-approval (202) path and inspect what was raised.

func serviceChainFull(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader, raiser ApprovalRaiser, path string) int {
	t.Helper()
	return serviceChainResponse(t, repo, usage, raiser, path).Code
}

func serviceChainResponse(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader, raiser ApprovalRaiser, path string) *httptest.ResponseRecorder {
	t.Helper()
	authority := NewStandingGrantAuthority(repo, usage, nil)
	final := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(admitAdmittedStatus) })
	chain := StandingGrantMiddleware(authority, raiser)(AgentOperationMiddleware(final))

	p := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	r := httptest.NewRequest(http.MethodPost, path, nil)
	r = r.WithContext(identity.WithPrincipal(r.Context(), p))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)
	return w
}

func assertAdmissionReason(t *testing.T, response *httptest.ResponseRecorder, status int, reason string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("want status %d, got %d: %s", status, response.Code, response.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode refusal response: %v", err)
	}
	if body["admission_reason"] != reason {
		t.Fatalf("want admission_reason %q, got %#v", reason, body)
	}
	if body["error"] != "service authority does not include this operation" {
		t.Fatalf("refusal message changed or exposed details: %#v", body)
	}
}

func TestServiceWithStandingGrantReachesCreateRun(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	if code := serviceChain(t, repo, fakeUsage{concurrent: 2}); code != admitAdmittedStatus {
		t.Fatalf("service with valid grant not admitted: %d", code)
	}
}

// BOUNDARY: a verified service with no covering grant is "ask, don't block" —
// 202 approval-pending with a raise fired, NOT a hard refuse.
func TestServiceWithoutStandingGrantAsksForApproval(t *testing.T) {
	raiser := &fakeRaiser{}
	code := serviceChainFull(t, &memStandingGrants{}, fakeUsage{}, raiser, "/api/v1/runs")
	if code != http.StatusAccepted {
		t.Fatalf("service with no grant should get 202 approval-pending, got %d", code)
	}
	if len(raiser.raised) != 1 || raiser.raised[0].Subject != "prompt-manager" || raiser.raised[0].Operation != "run.create" {
		t.Fatalf("expected one raised authorization ask for the subject, got %+v", raiser.raised)
	}
}

// BOUNDARY: with NO raiser configured the needs-approval outcome degrades to a
// hard refuse (no regression / no accidental allow).
func TestServiceWithoutStandingGrantAndNoRaiserRefused(t *testing.T) {
	response := serviceChainResponse(t, &memStandingGrants{}, fakeUsage{}, nil, "/api/v1/runs")
	assertAdmissionReason(t, response, http.StatusForbidden, "approval_raiser_unavailable")
}

// ANTI-FLOOD: over the per-subject outstanding cap → refuse WITHOUT a 202, with
// 429 (too many requests), and no further raise attempted beyond the cap.
func TestServiceOverPendingCapRefusedWithoutRaising(t *testing.T) {
	response := serviceChainResponse(t, &memStandingGrants{}, fakeUsage{}, &fakeRaiser{flood: true}, "/api/v1/runs")
	assertAdmissionReason(t, response, http.StatusTooManyRequests, "approval_raise_limited")
}

// The heartbeat creates its TASK before its run; the standing grant must admit
// that leg too (one authorized unit).
func TestServiceWithStandingGrantReachesCreateTask(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	if code := serviceChainPath(t, repo, fakeUsage{concurrent: 2}, "/api/v1/tasks"); code != admitAdmittedStatus {
		t.Fatalf("service with valid grant not admitted for task.create: %d", code)
	}
}

func TestServiceWithoutStandingGrantAsksOnCreateTask(t *testing.T) {
	if code := serviceChainPath(t, &memStandingGrants{}, fakeUsage{}, "/api/v1/tasks"); code != http.StatusAccepted {
		t.Fatalf("service with no grant should get 202 approval-pending on task.create, got %d", code)
	}
}

// BOUNDARY: over-limit is a HARD refuse (403), NOT an ask — approval cannot fix
// a bound the caller has already hit.
func TestServiceOverConcurrencyHardRefused(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 2}))
	raiser := &fakeRaiser{}
	code := serviceChainFull(t, repo, fakeUsage{concurrent: 2}, raiser, "/api/v1/runs")
	if code != http.StatusForbidden {
		t.Fatalf("service at concurrency limit should be hard-refused 403, got %d", code)
	}
	if len(raiser.raised) != 0 {
		t.Fatalf("over-limit must NOT raise an approval ask, got %+v", raiser.raised)
	}
}

func TestServiceAdmissionRefusalReasonsAreRedactedAndCausal(t *testing.T) {
	secret := errors.New("private grant store: owner=operator@example.test grant=sg-secret")
	cases := []struct {
		name   string
		repo   *memStandingGrants
		usage  StandingGrantUsageReader
		raiser ApprovalRaiser
		want   string
	}{
		{name: "grant lookup failure", repo: &memStandingGrants{err: secret}, usage: fakeUsage{}, raiser: &fakeRaiser{}, want: "authority_state_unavailable"},
		{name: "usage lookup failure", repo: &memStandingGrants{bySubject: map[string][]*domain.StandingGrant{"prompt-manager": {standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{})}}}, usage: fakeUsage{err: secret}, raiser: &fakeRaiser{}, want: "authority_state_unavailable"},
		{name: "missing raiser", repo: &memStandingGrants{}, usage: fakeUsage{}, raiser: nil, want: "approval_raiser_unavailable"},
		{name: "raise failure", repo: &memStandingGrants{}, usage: fakeUsage{}, raiser: &fakeRaiser{err: secret}, want: "approval_raise_failed"},
		{name: "raise not accepted", repo: &memStandingGrants{}, usage: fakeUsage{}, raiser: &fakeRaiser{raisedOK: true}, want: "approval_raise_failed"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := serviceChainResponse(t, tc.repo, tc.usage, tc.raiser, "/api/v1/runs")
			assertAdmissionReason(t, response, http.StatusForbidden, tc.want)
			if strings.Contains(response.Body.String(), "private grant store") || strings.Contains(response.Body.String(), "operator@example.test") || strings.Contains(response.Body.String(), "sg-secret") {
				t.Fatalf("private error details leaked: %s", response.Body.String())
			}
		})
	}
}

func TestServiceGrantLimitReasonRemainsHardRefusal(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 1}))
	raiser := &fakeRaiser{}
	response := serviceChainResponse(t, repo, fakeUsage{concurrent: 1}, raiser, "/api/v1/tasks")
	assertAdmissionReason(t, response, http.StatusForbidden, "grant_limit_reached")
	if len(raiser.raised) != 0 {
		t.Fatalf("grant limit refusal must not raise approval, got %+v", raiser.raised)
	}
}

func TestServiceUnclassifiedAuthorityFailureStaysGenericRefusal(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true})
	ctx = authn.WithRequestOperationAuthority(ctx, failingOperationAuthority{err: errors.New("private unexpected details")})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unknown authority failure reached handler") })).ServeHTTP(w, r)
	assertAdmissionReason(t, w, http.StatusForbidden, "operation_not_admitted")
	if strings.Contains(w.Body.String(), "private unexpected details") {
		t.Fatalf("unknown authority error leaked: %s", w.Body.String())
	}
}

func TestServiceStandingGrantStateUnavailableTakesPrecedenceOverWrappedAdmissionCause(t *testing.T) {
	for _, cause := range []error{authn.ErrNeedsApproval, authn.ErrStandingGrantLimit} {
		t.Run(cause.Error(), func(t *testing.T) {
			raiser := &fakeRaiser{}
			wrapped := fmt.Errorf("%w: %w", errStandingGrantStateUnavailable, cause)
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{
				Kind: identity.ActorService, Subject: "prompt-manager", Verified: true,
				Scopes: []string{"agent-manager:write"},
			})
			ctx = authn.WithRequestOperationAuthority(ctx, failingOperationAuthority{err: wrapped})
			ctx = context.WithValue(ctx, approvalRaiserKey{}, raiser)
			r := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil).WithContext(ctx)
			w := httptest.NewRecorder()
			AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("unavailable standing-grant state reached handler")
			})).ServeHTTP(w, r)

			assertAdmissionReason(t, w, http.StatusForbidden, "authority_state_unavailable")
			if len(raiser.raised) != 0 {
				t.Fatalf("unavailable authority must not raise an approval: %+v", raiser.raised)
			}
			if strings.Contains(w.Body.String(), cause.Error()) {
				t.Fatalf("typed cause leaked to response: %s", w.Body.String())
			}
		})
	}
}

func TestServiceMissingOperationAuthorityIsRedactedRefusal(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	r := httptest.NewRequest(http.MethodPost, "/api/v1/runs", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("missing authority reached handler") })).ServeHTTP(w, r)
	assertAdmissionReason(t, w, http.StatusForbidden, "authority_unavailable")
}

type failingOperationAuthority struct{ err error }

func (a failingOperationAuthority) AuthorizeOperation(context.Context, identity.Principal, string, string, string) error {
	return a.err
}

// An expired or revoked grant = a known caller with no ACTIVE grant → ask, not
// a silent block. (ActiveForSubject filters both out, so the authority sees no
// covering grant and returns needs-approval.)
func TestServiceWithExpiredGrantAsksForApproval(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(-time.Second), authn.StandingGrantLimits{}))
	if code := serviceChain(t, repo, fakeUsage{}); code != http.StatusAccepted {
		t.Fatalf("service with expired grant should ask (202), got %d", code)
	}
}

func TestServiceWithRevokedGrantAsksForApproval(t *testing.T) {
	repo := &memStandingGrants{}
	g := standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{})
	_ = repo.Put(context.Background(), g)
	if _, err := repo.Revoke(context.Background(), g.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if code := serviceChain(t, repo, fakeUsage{}); code != http.StatusAccepted {
		t.Fatalf("service with revoked grant should ask (202), got %d", code)
	}
}

func TestServiceGrantDoesNotReachGlobalManagement(t *testing.T) {
	// A standing grant admits run.create only; it must not open other service
	// writes (e.g. a global mutation RPC) — and global mgmt is a hard refuse, not
	// an approval ask (the ask path is scoped to run.create/task only).
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	authority := NewStandingGrantAuthority(repo, fakeUsage{}, nil)
	raiser := &fakeRaiser{}
	chain := StandingGrantMiddleware(authority, raiser)(AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("service grant reached a non-create operation")
	})))
	p := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", nil).WithContext(identity.WithPrincipal(context.Background(), p))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("service grant escaped to global management: %d", w.Code)
	}
	if len(raiser.raised) != 0 {
		t.Fatalf("a non-create service op must NOT raise an approval ask, got %+v", raiser.raised)
	}
}
