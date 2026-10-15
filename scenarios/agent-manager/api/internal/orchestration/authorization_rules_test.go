//go:build linux

package orchestration

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil"
	"errors"
	"github.com/vrooli/api-core/authn"

	requestidentity "github.com/vrooli/api-core/identity"
)

func authorizationFixture(t *testing.T, now *time.Time) *Orchestrator {
	t.Helper()
	db, cleanup := testutil.SetupTestDB(t)
	t.Cleanup(cleanup)
	repos, events, _ := testutil.SetupTestReposWithDB(t, db)
	return New(nil, nil, repos.Runs, WithClock(func() time.Time { return *now }), WithEvents(events), WithIdentitySecret([]byte("authorization-rules-test")), WithAuthorizations(database.NewAuthorizationRepository(db)))
}

func ownerContext(now time.Time) context.Context {
	return requestidentity.WithPrincipal(context.Background(), requestidentity.Principal{Kind: requestidentity.ActorHuman, Subject: "owner", Source: requestidentity.SourceCloudflareAccess, Verified: true, Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: now.Add(6 * time.Hour)})
}

func TestPreparationConsentNamesMissingOwnerCapabilitiesAndRemainsPending(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	g, err := o.RequestAuthorization(context.Background(), AuthorizationRequest{PID: os.Getpid(), HarnessKind: "codex", HarnessSession: "missing-owner-scopes", DurationSeconds: 3600, Policy: "team-preparation", Targets: []string{"prompt-manager:team/draft"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.ApproveAuthorization(ownerContext(now), g.ID)
	if !errors.Is(err, authn.ErrCapability) || !strings.Contains(err.Error(), "prompt-manager:read") || !strings.Contains(err.Error(), "prompt-manager:write") {
		t.Fatalf("owner receives no actionable missing-capability diagnosis: %v", err)
	}
	retained, err := o.GetAuthorization(context.Background(), g.ID)
	if err != nil || retained.State != domain.AuthorizationPending || retained.ApprovedAt != nil || retained.Grant != nil {
		t.Fatalf("failed consent changed authority: %+v, %v", retained, err)
	}
	if token, found, err := o.ResolveAuthorizationCredential(context.Background(), os.Getpid()); err != nil || found || token != "" {
		t.Fatalf("failed consent minted identity: found=%v, err=%v", found, err)
	}
	p, _ := requestidentity.PrincipalFromContext(ownerContext(now))
	p.Scopes = append(p.Scopes, "prompt-manager:read", "prompt-manager:write")
	approved, err := o.ApproveAuthorization(requestidentity.WithPrincipal(context.Background(), p), g.ID)
	if err != nil || approved.State != domain.AuthorizationApproved || approved.Grant == nil || !approved.Grant.Allows("team.prepare", "prompt-manager:team/draft", now) || approved.Grant.Allows("team.prepare", "prompt-manager:team/other", now) || approved.Grant.Allows("team.manage", "prompt-manager:team/draft", now) {
		t.Fatalf("explicit owner approval lost its reviewed boundary: %v", err)
	}
}

func TestOneConsentCarriesSeveralPoliciesWithoutCrossingTargets(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	ctx := context.Background()
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "rules", DurationSeconds: 3600, Rules: []domain.AuthorizationRule{
		{Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}},
		{Policy: "effort-metadata", Targets: []string{"agent-manager:effort/bas"}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if g.Policy != "agent-management+effort-metadata" || len(g.PolicyRules()) != 2 {
		t.Fatalf("request lost its rules: %+v", g)
	}
	approved, err := o.ApproveAuthorization(ownerContext(now), g.ID)
	if err != nil {
		t.Fatal(err)
	}
	grant := approved.Grant
	if !grant.Allows("agent.manage", "agent-manager:run/r1", now) || !grant.Allows("effort.reconcile", "agent-manager:effort/bas", now) {
		t.Fatal("an approved rule was refused")
	}
	if grant.Allows("agent.manage", "agent-manager:effort/bas", now) || grant.Allows("effort.reconcile", "agent-manager:run/r1", now) {
		t.Fatal("an operation reached another policy's target")
	}
	token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid())
	if err != nil || !found || token == "" {
		t.Fatalf("approved multi-policy consent did not resolve: %v", err)
	}
}

func TestAuthorizationRequestValidationRefusesMixedAndDuplicateRules(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	for name, req := range map[string]AuthorizationRequest{
		"mixed forms":      {Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}, Rules: []domain.AuthorizationRule{{Policy: "effort-metadata", Targets: []string{"agent-manager:effort/bas"}}}},
		"duplicate policy": {Rules: []domain.AuthorizationRule{{Policy: "effort-metadata", Targets: []string{"agent-manager:effort/a"}}, {Policy: "effort-metadata", Targets: []string{"agent-manager:effort/b"}}}},
		"foreign target":   {Rules: []domain.AuthorizationRule{{Policy: "effort-metadata", Targets: []string{"swarm-manager:*"}}}},
	} {
		req.PID, req.HarnessKind, req.HarnessSession, req.DurationSeconds = os.Getpid(), "claude-code", "invalid", 3600
		if _, err := o.RequestAuthorization(context.Background(), req); err == nil {
			t.Fatalf("%s: request accepted", name)
		}
	}
}

func TestNewRequestSupersedesPendingAndApprovalExplainsWhy(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	ctx := context.Background()
	base := AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "supersede", DurationSeconds: 3600, Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}}
	first, err := o.RequestAuthorization(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	again, err := o.RequestAuthorization(ctx, base)
	if err != nil || again.ID != first.ID {
		t.Fatalf("identical request was not returned unchanged: %v", err)
	}
	next := base
	next.Policy, next.Targets = "effort-metadata", []string{"agent-manager:effort/bas"}
	second, err := o.RequestAuthorization(ctx, next)
	if err != nil {
		t.Fatalf("a newer request was refused while one was pending: %v", err)
	}
	old, _ := o.GetAuthorization(ctx, first.ID)
	if old.State != domain.AuthorizationSuperseded || old.SupersededBy == nil || *old.SupersededBy != second.ID {
		t.Fatalf("outdated request was not superseded: %+v", old)
	}
	if _, err := o.ApproveAuthorization(ownerContext(now), first.ID); err == nil || !strings.Contains(err.Error(), second.ID.String()) {
		t.Fatalf("approving a replaced request did not name its replacement: %v", err)
	}
	if token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err != nil || found || token != "" {
		t.Fatalf("superseded or pending requests granted authority: found=%v err=%v", found, err)
	}
}

func TestExpiredRequestReadsExpiredAndCarriesNoAuthority(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	ctx := context.Background()
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "expiry", DurationSeconds: 3600, Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}})
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(AuthorizationReviewWindow)
	if state := g.EffectiveState(now); state != domain.AuthorizationExpired {
		t.Fatalf("request past its deadline reads %q", state)
	}
	if _, err := o.ApproveAuthorization(ownerContext(now), g.ID); err == nil || !strings.Contains(err.Error(), "expired at") {
		t.Fatalf("expired approval error is not specific: %v", err)
	}
	if _, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid()); err != nil || found {
		t.Fatalf("expired request affected credential resolution: found=%v err=%v", found, err)
	}
	if _, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "expiry", DurationSeconds: 3600, Policy: "effort-metadata", Targets: []string{"agent-manager:effort/bas"}}); err != nil {
		t.Fatalf("an expired request blocked a new one: %v", err)
	}
}

func TestLapsedConsentReturnsTheSessionToItsOwnIdentity(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	ctx := context.Background()
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "lapse", DurationSeconds: 3600, Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := o.ApproveAuthorization(ownerContext(now), g.ID); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Hour)
	token, found, err := o.ResolveAuthorizationCredential(ctx, os.Getpid())
	if err != nil || found || token != "" {
		t.Fatalf("lapsed consent must fall back to the session's own identity, got found=%v err=%v", found, err)
	}
	stored, err := o.GetAuthorization(ctx, g.ID)
	if err != nil {
		t.Fatal(err)
	}
	if state := stored.EffectiveState(now); state != domain.AuthorizationExpired {
		t.Fatalf("lapsed consent reads %q, want expired", state)
	}
}

func TestApprovedLifetimeOutlastsTheApproversLogin(t *testing.T) {
	now := time.Now().UTC()
	o := authorizationFixture(t, &now)
	ctx := context.Background()
	week := int64(MaxAuthorizationLifetime / time.Second)
	if _, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "long", DurationSeconds: week + 1, Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}}); err == nil {
		t.Fatal("a lifetime beyond the maximum was accepted")
	}
	g, err := o.RequestAuthorization(ctx, AuthorizationRequest{PID: os.Getpid(), HarnessKind: "claude-code", HarnessSession: "long", DurationSeconds: 24 * 3600, Policy: "agent-management", Targets: []string{"agent-manager:run/r1"}})
	if err != nil {
		t.Fatal(err)
	}
	approved, err := o.ApproveAuthorization(ownerContext(now), g.ID) // login proof lasts 6h
	if err != nil {
		t.Fatal(err)
	}
	if want := now.Add(24 * time.Hour); !approved.Grant.ExpiresAt.Equal(want) {
		t.Fatalf("grant expires %s, want the approved 24h (%s)", approved.Grant.ExpiresAt, want)
	}
}
