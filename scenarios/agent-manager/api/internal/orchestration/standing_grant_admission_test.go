package orchestration

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/vrooli/api-core/authn"
	coreidentity "github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/owneridentity"
)

type fakeStandingGrants struct{ grants []*domain.StandingGrant }

func (f *fakeStandingGrants) Put(_ context.Context, grant *domain.StandingGrant) error {
	if grant != nil {
		f.grants = append(f.grants, grant)
	}
	return nil
}
func (f *fakeStandingGrants) ActiveForSubject(_ context.Context, subject string, now time.Time) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, g := range f.grants {
		if g.Subject == subject && g.Active(now) {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeStandingGrants) Revoke(context.Context, string, time.Time) (bool, error) {
	return false, nil
}

func (f *fakeStandingGrants) List(context.Context, int) ([]*domain.StandingGrant, error) {
	return append([]*domain.StandingGrant(nil), f.grants...), nil
}

func (f *fakeStandingGrants) ListForApprover(_ context.Context, approver string, _ int) ([]*domain.StandingGrant, error) {
	var out []*domain.StandingGrant
	for _, grant := range f.grants {
		if grant != nil && grant.ApprovedBy == approver && grant.Grant != nil && grant.Grant.Approver == approver {
			out = append(out, grant)
		}
	}
	return out, nil
}

// A service-standing caller re-validates idempotently: with the grant unchanged
// between admit and revalidation it PASSES (a service bearer carries no owner
// expiry, so the human/run ceiling comparison must not apply to it).
func TestRevalidateServiceStandingPassesWhenGrantUnchanged(t *testing.T) {
	now := time.Now()
	store := &fakeStandingGrants{grants: []*domain.StandingGrant{serviceStandingGrant("prompt-manager", "operator", now.Add(time.Hour))}}
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: store}
	ctx := coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	req := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		t.Fatalf("admit refused: %v", err)
	}
	if err := o.revalidateCreateRunCaller(ctx, req); err != nil {
		t.Fatalf("revalidation refused an unchanged service-standing caller: %v", err)
	}
}

// If the grant is revoked (or expires) between admit and revalidation, the
// service-standing caller is refused — the anti-escalation guarantee holds.
func TestRevalidateServiceStandingRefusedWhenGrantRevokedMidAdmission(t *testing.T) {
	now := time.Now()
	grant := serviceStandingGrant("prompt-manager", "operator", now.Add(time.Hour))
	store := &fakeStandingGrants{grants: []*domain.StandingGrant{grant}}
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: store}
	ctx := coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	req := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		t.Fatalf("admit refused: %v", err)
	}
	// Grant is revoked after admit, before the first effect.
	revokedAt := now
	grant.State = domain.StandingGrantRevoked
	grant.RevokedAt = &revokedAt
	if err := o.revalidateCreateRunCaller(ctx, req); err == nil {
		t.Fatal("revalidation admitted a service-standing caller whose grant was revoked mid-admission")
	}
}

// Likewise if the grant expires between admit and revalidation.
func TestRevalidateServiceStandingRefusedWhenGrantExpiresMidAdmission(t *testing.T) {
	now := time.Now()
	grant := serviceStandingGrant("prompt-manager", "operator", now.Add(time.Hour))
	store := &fakeStandingGrants{grants: []*domain.StandingGrant{grant}}
	clock := now
	o := &Orchestrator{clock: func() time.Time { return clock }, standingGrants: store}
	ctx := coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	req := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		t.Fatalf("admit refused: %v", err)
	}
	// Time advances past the grant's expiry before the first effect.
	clock = now.Add(2 * time.Hour)
	if err := o.revalidateCreateRunCaller(ctx, req); err == nil {
		t.Fatal("revalidation admitted a service-standing caller whose grant expired mid-admission")
	}
}

// A standing grant may live up to MaxStandingGrantLifetime (30 days) — longer
// than the 7-day interactive cap — because too short a TTL silently reintroduces
// the unattended-heartbeat breakage this effort fixes. Beyond 30 days is refused.
func TestCreateStandingGrantLifetimeCap(t *testing.T) {
	now := time.Now()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: &fakeStandingGrants{}}
	req := StandingGrantRequest{Subject: "prompt-manager", Approver: "operator", Limits: authn.StandingGrantLimits{MaxConcurrentRuns: 3}}

	// Exactly 30 days is accepted.
	req.TTL = MaxStandingGrantLifetime
	g, err := o.CreateStandingGrant(context.Background(), req)
	if err != nil || g == nil {
		t.Fatalf("30-day standing grant should be accepted: %v", err)
	}
	if !g.ExpiresAt.Equal(now.Add(MaxStandingGrantLifetime)) {
		t.Fatalf("expiry not set to the full 30-day lifetime: %v", g.ExpiresAt)
	}

	// Just over 30 days is refused.
	req.TTL = MaxStandingGrantLifetime + time.Minute
	if _, err := o.CreateStandingGrant(context.Background(), req); err == nil {
		t.Fatal("standing grant longer than 30 days should be refused")
	}

	// A standing grant may exceed the interactive 7-day cap (the two are
	// deliberately separate); 14 days is fine.
	req.TTL = 14 * 24 * time.Hour
	if _, err := o.CreateStandingGrant(context.Background(), req); err != nil {
		t.Fatalf("14-day standing grant (beyond interactive 7d cap) should be accepted: %v", err)
	}
}

func serviceStandingGrant(subject, owner string, exp time.Time) *domain.StandingGrant {
	return &domain.StandingGrant{
		ID:      "sg",
		Subject: subject,
		Grant: &coreidentity.OperationGrant{
			ID: "sg", Approver: owner, Policy: "run-creation",
			Operations: []string{"run.create"}, Targets: []string{"agent-manager:*"}, ExpiresAt: exp,
		},
		Limits:     authn.StandingGrantLimits{MaxConcurrentRuns: 5},
		State:      domain.StandingGrantApproved,
		ApprovedBy: owner,
		ApprovedAt: time.Now().Add(-time.Minute),
		ExpiresAt:  exp,
	}
}

// A verified service principal admitted on a standing grant attributes its run
// to the grant's approver (the operator), not the service and not the anonymous
// path. The middleware already authorized the operation; this asserts the owner
// attribution the admission layer adds.
func TestServiceStandingGrantAttributesOwnerToApprover(t *testing.T) {
	now := time.Now()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: &fakeStandingGrants{grants: []*domain.StandingGrant{serviceStandingGrant("prompt-manager", "operator", now.Add(time.Hour))}}}
	ctx := coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "prompt-manager", Verified: true, Scopes: []string{"agent-manager:write"}})
	req := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &req); err != nil {
		t.Fatalf("service standing-grant caller refused: %v", err)
	}
	if req.caller == nil || req.caller.Kind != "service-standing" || req.caller.Subject != "operator" {
		t.Fatalf("caller not attributed to a service-standing operator owner: %+v", req.caller)
	}
	if req.OwnerSubject != "operator" {
		t.Fatalf("run owner not attributed to the grant approver: %q", req.OwnerSubject)
	}
}

// A verified service principal with no standing grant does NOT get the
// service-standing attribution. With AUTH-01 off it keeps the pre-AUTH-01
// anonymous path (no caller record); with it on it is refused.
// A verified service principal with no covering standing grant is REFUSED and
// never gets a service-standing attribution — regardless of the AUTH-01 switch.
// A service bearer is identity, not run-creation authority: without a grant it
// must not fall back to the pre-AUTH-01 anonymous path. (The middleware already
// refuses this; reaching admission means a grant was revoked mid-request.)
func TestServiceWithoutStandingGrantIsRefused(t *testing.T) {
	now := time.Now()
	o := &Orchestrator{clock: func() time.Time { return now }, standingGrants: &fakeStandingGrants{}}
	ctx := coreidentity.WithPrincipal(context.Background(), coreidentity.Principal{Kind: coreidentity.ActorService, Subject: "prompt-manager", Verified: true})

	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "")
	req := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &req); err == nil {
		t.Fatal("verified service without a standing grant should be refused, not anonymous")
	}
	if req.caller != nil {
		t.Fatalf("service without grant acquired a caller record: %+v", req.caller)
	}

	t.Setenv(owneridentity.CreateRunCallerEnforceEnv, "true")
	enforced := CreateRunRequest{}
	if err := o.authenticateCreateRunCaller(ctx, &enforced); err == nil {
		t.Fatal("switch-on service without grant should be refused")
	}
}
