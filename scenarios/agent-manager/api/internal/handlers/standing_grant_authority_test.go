package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
)

// memStandingGrants is an in-memory StandingGrantRepository for middleware tests.
// It needs no operator and no database.
type memStandingGrants struct {
	bySubject map[string][]*domain.StandingGrant
}

func (m *memStandingGrants) Put(_ context.Context, g *domain.StandingGrant) error {
	if m.bySubject == nil {
		m.bySubject = map[string][]*domain.StandingGrant{}
	}
	m.bySubject[g.Subject] = append(m.bySubject[g.Subject], g)
	return nil
}

func (m *memStandingGrants) ActiveForSubject(_ context.Context, subject string, now time.Time) ([]*domain.StandingGrant, error) {
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

type fakeUsage struct{ concurrent int }

func (f fakeUsage) StandingGrantUsage(context.Context, string) (authn.StandingGrantUsage, error) {
	return authn.StandingGrantUsage{ConcurrentRuns: f.concurrent}, nil
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

// serviceChain builds the StandingGrant→AgentOperation middleware chain used in
// production and drives a POST /api/v1/runs as a verified service principal.
func serviceChain(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader) int {
	return serviceChainPath(t, repo, usage, "/api/v1/runs")
}

// serviceChainPath drives the production middleware chain for a verified service
// principal against an arbitrary path, returning the HTTP status.
func serviceChainPath(t *testing.T, repo domain.StandingGrantRepository, usage StandingGrantUsageReader, path string) int {
	t.Helper()
	authority := NewStandingGrantAuthority(repo, usage, nil)
	reached := false
	final := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true; w.WriteHeader(http.StatusAccepted) })
	chain := StandingGrantMiddleware(authority)(AgentOperationMiddleware(final))

	p := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	r := httptest.NewRequest(http.MethodPost, path, nil)
	r = r.WithContext(identity.WithPrincipal(r.Context(), p))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)
	if w.Code == http.StatusAccepted && !reached {
		t.Fatal("202 without reaching the handler")
	}
	return w.Code
}

func TestServiceWithStandingGrantReachesCreateRun(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	if code := serviceChain(t, repo, fakeUsage{concurrent: 2}); code != http.StatusAccepted {
		t.Fatalf("service with valid grant not admitted: %d", code)
	}
}

func TestServiceWithoutStandingGrantRefused(t *testing.T) {
	if code := serviceChain(t, &memStandingGrants{}, fakeUsage{}); code != http.StatusForbidden {
		t.Fatalf("service with no grant should be 403, got %d", code)
	}
}

// The heartbeat creates its TASK before its run; the standing grant must admit
// that leg too (one authorized unit), or the task 401s before the run. This is
// the live defect the run-only wiring missed.
func TestServiceWithStandingGrantReachesCreateTask(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	if code := serviceChainPath(t, repo, fakeUsage{concurrent: 2}, "/api/v1/tasks"); code != http.StatusAccepted {
		t.Fatalf("service with valid grant not admitted for task.create: %d", code)
	}
}

func TestServiceWithoutStandingGrantRefusedOnCreateTask(t *testing.T) {
	if code := serviceChainPath(t, &memStandingGrants{}, fakeUsage{}, "/api/v1/tasks"); code != http.StatusForbidden {
		t.Fatalf("service with no grant should be 403 on task.create, got %d", code)
	}
}

func TestServiceOverConcurrencyRefused(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 2}))
	if code := serviceChain(t, repo, fakeUsage{concurrent: 2}); code != http.StatusForbidden {
		t.Fatalf("service at concurrency limit should be 403, got %d", code)
	}
}

func TestServiceWithExpiredGrantRefused(t *testing.T) {
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(-time.Second), authn.StandingGrantLimits{}))
	if code := serviceChain(t, repo, fakeUsage{}); code != http.StatusForbidden {
		t.Fatalf("service with expired grant should be 403, got %d", code)
	}
}

func TestServiceWithRevokedGrantRefused(t *testing.T) {
	repo := &memStandingGrants{}
	g := standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{})
	_ = repo.Put(context.Background(), g)
	if _, err := repo.Revoke(context.Background(), g.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if code := serviceChain(t, repo, fakeUsage{}); code != http.StatusForbidden {
		t.Fatalf("service with revoked grant should be 403, got %d", code)
	}
}

func TestServiceGrantDoesNotReachGlobalManagement(t *testing.T) {
	// A standing grant admits run.create only; it must not open other service
	// writes (e.g. a global mutation RPC).
	repo := &memStandingGrants{}
	_ = repo.Put(context.Background(), standingGrantRecord("prompt-manager", "operator", time.Now().Add(time.Hour), authn.StandingGrantLimits{MaxConcurrentRuns: 5}))
	authority := NewStandingGrantAuthority(repo, fakeUsage{}, nil)
	chain := StandingGrantMiddleware(authority)(AgentOperationMiddleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("service grant reached a non-create operation")
	})))
	p := identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}}
	r := httptest.NewRequest(http.MethodPost, "/api/v1/runs/stop-all", nil).WithContext(identity.WithPrincipal(context.Background(), p))
	w := httptest.NewRecorder()
	chain.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("service grant escaped to global management: %d", w.Code)
	}
}
