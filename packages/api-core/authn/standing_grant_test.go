package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vrooli/api-core/identity"
)

func TestRunCreationPolicyIsBoundedToRunCreateOnAgentManager(t *testing.T) {
	p, ok := InteractivePolicy("run-creation")
	if !ok {
		t.Fatal("run-creation policy is not reviewed")
	}
	if len(p.Operations) != 1 || p.Operations[0] != "run.create" {
		t.Fatalf("run-creation must carry only run.create: %+v", p.Operations)
	}
	if len(p.Domains) != 1 || p.Domains[0] != "agent-manager" {
		t.Fatalf("run-creation must be bounded to agent-manager: %+v", p.Domains)
	}
	for _, scope := range p.Scopes {
		if scope != "agent-manager:read" && scope != "agent-manager:write" {
			t.Fatalf("run-creation leaked a scope beyond agent-manager: %q", scope)
		}
	}
}

func runCreationGrant(owner string, exp time.Time) *identity.OperationGrant {
	return &identity.OperationGrant{ID: "standing", Approver: owner, Policy: "run-creation", Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: exp}
}

func TestAuthorizeStandingGrantAdmitsCoveredRunCreate(t *testing.T) {
	now := time.Now()
	grant := runCreationGrant("operator", now.Add(time.Hour))
	if err := AuthorizeStandingGrant(grant, StandingGrantLimits{MaxConcurrentRuns: 5, MaxRunBudgetUSD: 100}, StandingGrantUsage{ConcurrentRuns: 2, CommittedBudgetUSD: 40, RequestBudgetUSD: 10}, "run.create", "agent-manager:run/new", now); err != nil {
		t.Fatalf("covered within-limit request refused: %v", err)
	}
}

func TestAuthorizeStandingGrantRefusalPaths(t *testing.T) {
	now := time.Now()
	cases := []struct {
		name    string
		grant   *identity.OperationGrant
		limits  StandingGrantLimits
		usage   StandingGrantUsage
		op, tgt string
		wantErr error
	}{
		{"missing grant", nil, StandingGrantLimits{}, StandingGrantUsage{}, "run.create", "agent-manager:run/new", ErrOperation},
		{"wrong operation", runCreationGrant("operator", now.Add(time.Hour)), StandingGrantLimits{}, StandingGrantUsage{}, "agent.manage", "agent-manager:run/new", ErrOperation},
		{"wrong domain target", runCreationGrant("operator", now.Add(time.Hour)), StandingGrantLimits{}, StandingGrantUsage{}, "run.create", "git-control-tower:run/new", ErrOperation},
		{"expired grant", runCreationGrant("operator", now.Add(-time.Second)), StandingGrantLimits{}, StandingGrantUsage{}, "run.create", "agent-manager:run/new", ErrOperation},
		{"at concurrency limit", runCreationGrant("operator", now.Add(time.Hour)), StandingGrantLimits{MaxConcurrentRuns: 2}, StandingGrantUsage{ConcurrentRuns: 2}, "run.create", "agent-manager:run/new", ErrStandingGrantLimit},
		{"over budget", runCreationGrant("operator", now.Add(time.Hour)), StandingGrantLimits{MaxRunBudgetUSD: 50}, StandingGrantUsage{CommittedBudgetUSD: 45, RequestBudgetUSD: 10}, "run.create", "agent-manager:run/new", ErrStandingGrantLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := AuthorizeStandingGrant(tc.grant, tc.limits, tc.usage, tc.op, tc.tgt, now)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("want %v, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestAuthorizeStandingGrantUnboundedLimitsAllowAnyUsage(t *testing.T) {
	now := time.Now()
	grant := runCreationGrant("operator", now.Add(time.Hour))
	// Zero limits mean "no bound declared"; a large usage must still admit.
	if err := AuthorizeStandingGrant(grant, StandingGrantLimits{}, StandingGrantUsage{ConcurrentRuns: 9999, CommittedBudgetUSD: 1e9, RequestBudgetUSD: 1e9}, "run.create", "agent-manager:run/new", now); err != nil {
		t.Fatalf("unbounded grant refused a large usage: %v", err)
	}
}

func TestEvaluateStandingGrantMatchesAdmissionAndReportsReachedDimensions(t *testing.T) {
	now := time.Now()
	grant := runCreationGrant("operator", now.Add(time.Hour))
	limits := StandingGrantLimits{MaxConcurrentRuns: 2, MaxRunBudgetUSD: 50}
	usage := StandingGrantUsage{ConcurrentRuns: 2, CommittedBudgetUSD: 45, RequestBudgetUSD: 10}

	evaluation := EvaluateStandingGrant(grant, limits, usage, "run.create", "agent-manager:run/new", now)
	if !evaluation.Covered || evaluation.Admitted || len(evaluation.Exceeded) != 2 {
		t.Fatalf("unexpected evaluation: %+v", evaluation)
	}
	if evaluation.Exceeded[0].Dimension != StandingGrantConcurrentRunsLimit || evaluation.Exceeded[0].CurrentRuns != 2 || evaluation.Exceeded[0].BoundRuns != 2 {
		t.Fatalf("concurrency detail was not retained: %+v", evaluation.Exceeded[0])
	}
	if evaluation.Exceeded[1].Dimension != StandingGrantBudgetLimit || evaluation.Exceeded[1].CurrentBudgetUSD != 55 || evaluation.Exceeded[1].BoundBudgetUSD != 50 {
		t.Fatalf("budget detail was not retained: %+v", evaluation.Exceeded[1])
	}
	if err := AuthorizeStandingGrant(grant, limits, usage, "run.create", "agent-manager:run/new", now); err != ErrStandingGrantLimit {
		t.Fatalf("normal admission diverged from evaluator: %v", err)
	}

	uncovered := EvaluateStandingGrant(grant, limits, usage, "agent.manage", "agent-manager:run/new", now)
	if uncovered.Covered || uncovered.Admitted || len(uncovered.Exceeded) != 0 {
		t.Fatalf("non-covering grant reported a limit: %+v", uncovered)
	}
}

// standingAuthority simulates the agent-manager ingress that, in Phase 2b, looks
// up the operator-approved grant for the verified subject and evaluates it. In
// these tests the grant is injected directly — no store or operator needed.
type standingAuthority struct {
	grant  *identity.OperationGrant
	limits StandingGrantLimits
	usage  StandingGrantUsage
}

func (a standingAuthority) AuthorizeOperation(_ context.Context, p identity.Principal, _, operation, target string) error {
	if p.Kind != identity.ActorService {
		return ErrOperation
	}
	return AuthorizeStandingGrant(a.grant, a.limits, a.usage, operation, target, time.Now())
}

func serviceCtx(auth RequestOperationAuthority) context.Context {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	return WithRequestOperationAuthority(ctx, auth)
}

func TestRequireOperationAdmitsServiceWithStandingGrant(t *testing.T) {
	auth := standingAuthority{grant: runCreationGrant("operator", time.Now().Add(time.Hour)), limits: StandingGrantLimits{MaxConcurrentRuns: 3}, usage: StandingGrantUsage{ConcurrentRuns: 1}}
	p, err := RequireOperation(serviceCtx(auth), "agent-manager:write", "run.create", "agent-manager:run/new")
	if err != nil {
		t.Fatalf("service with valid standing grant refused: %v", err)
	}
	if p.Kind != identity.ActorService || p.Subject != "prompt-manager" {
		t.Fatalf("unexpected admitted principal: %+v", p)
	}
}

func TestRequireOperationRefusesServiceWithoutGrant(t *testing.T) {
	auth := standingAuthority{grant: nil}
	if _, err := RequireOperation(serviceCtx(auth), "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("service with no standing grant was admitted")
	}
}

func TestRequireOperationRefusesServiceOverLimit(t *testing.T) {
	auth := standingAuthority{grant: runCreationGrant("operator", time.Now().Add(time.Hour)), limits: StandingGrantLimits{MaxConcurrentRuns: 1}, usage: StandingGrantUsage{ConcurrentRuns: 1}}
	_, err := RequireOperation(serviceCtx(auth), "agent-manager:write", "run.create", "agent-manager:run/new")
	if !errors.Is(err, ErrStandingGrantLimit) {
		t.Fatalf("over-limit service not refused with limit error: %v", err)
	}
}

func TestRequireOperationRefusesServiceWithExpiredGrant(t *testing.T) {
	auth := standingAuthority{grant: runCreationGrant("operator", time.Now().Add(-time.Second))}
	if _, err := RequireOperation(serviceCtx(auth), "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("service with expired standing grant was admitted")
	}
}

func TestRequireOperationStillRefusesUnknownActorThroughSeam(t *testing.T) {
	// The ingress seam must not admit a human/unknown actor as a service-style
	// standing-grant caller; those paths keep their own authorization.
	auth := standingAuthority{grant: runCreationGrant("operator", time.Now().Add(time.Hour))}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "operator", Verified: true, Scopes: []string{"agent-manager:write"}})
	ctx = WithRequestOperationAuthority(ctx, auth)
	if _, err := RequireOperation(ctx, "agent-manager:write", "run.create", "agent-manager:run/new"); err == nil {
		t.Fatal("human actor admitted through the standing-grant seam")
	}
}
