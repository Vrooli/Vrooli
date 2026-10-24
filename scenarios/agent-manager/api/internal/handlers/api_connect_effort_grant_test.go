package handlers

import (
	"context"
	"github.com/google/uuid"
	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/cli-core/cliutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func grantContext(operations, targets []string) context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "session-run", Owner: "owner", Verified: true,
		Scopes: []string{"agent-manager:read", "agent-manager:write"},
		Grant:  &identity.OperationGrant{ID: "grant-1", Approver: "owner", Operations: operations, Targets: targets, ExpiresAt: time.Now().Add(time.Hour)},
	})
}

func TestInteractiveGrantReconcilesOnlyItsNamedEffort(t *testing.T) {
	const ref = "effort:browser-automation-studio-rehabilitation"
	operator := pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR

	actor, delegated, err := effortGrantActor(grantContext([]string{"effort.reconcile"}, []string{EffortGrantTarget(ref)}), ref, operator)
	if err != nil || !delegated || !actor.Delegated || actor.ApprovedBy != "owner" || actor.ID != "session-run" || actor.Operator {
		t.Fatalf("approved grant for this effort was not delegated: %+v delegated=%v err=%v", actor, delegated, err)
	}

	for name, tc := range map[string]struct {
		operations, targets []string
		authority           pb.WatchAuthority
	}{
		"another effort":          {[]string{"effort.reconcile"}, []string{EffortGrantTarget("effort:other")}, operator},
		"run management policy":   {[]string{"agent.investigate", "agent.manage"}, []string{"agent-manager:*"}, operator},
		"family parent authority": {[]string{"effort.reconcile"}, []string{EffortGrantTarget(ref)}, pb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT},
	} {
		_, delegated, err := effortGrantActor(grantContext(tc.operations, tc.targets), ref, tc.authority)
		if !delegated || connect.CodeOf(err) != connect.CodePermissionDenied {
			t.Fatalf("%s: delegated=%v err=%v, want permission denied", name, delegated, err)
		}
	}

	noGrant := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "orchestrator-run", Verified: true})
	if _, delegated, err := effortGrantActor(noGrant, ref, operator); delegated || err != nil {
		t.Fatalf("a run without an interactive grant left the owner/orchestrator path: delegated=%v err=%v", delegated, err)
	}
}

func TestEffortMetadataGrantUsesSeparateOperationProof(t *testing.T) {
	now := time.Now()
	runID := uuid.New()
	baseline := managementClaimsFixture(t, runID, []string{"agent-manager:read"}, nil, now.Add(time.Hour))
	grant := managementGrantFixture(now,
		managementRuleFixture("effort-metadata", []string{"effort.reconcile"}, []string{"agent-manager:effort/*"}),
		managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:run/*"}),
	)
	operation := managementClaimsFixture(t, runID, []string{"agent-manager:read", "agent-manager:write"}, grant, now.Add(time.Hour))
	for _, tc := range []struct {
		name, ref string
		allowed   bool
	}{{"covered scoped wildcard", "effort:target", true}, {"wrong operation target", "other-domain", false}} {
		t.Run(tc.name, func(t *testing.T) {
			verifier := managementGrantVerifier{baseline: baseline, operation: operation}
			if !tc.allowed {
				verifier.operation = managementClaimsFixture(t, runID, []string{"agent-manager:write"}, managementGrantFixture(now, managementRuleFixture("agent-management", []string{"agent.manage"}, []string{"agent-manager:*"})), now.Add(time.Hour))
			}
			handler := provenance.Middleware(verifier)(authn.Middleware(authn.Config{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p, _ := identity.PrincipalFromContext(r.Context())
				if p.Grant != nil {
					t.Fatal("operation grant contaminated baseline principal")
				}
				actor, delegated, err := effortGrantActor(r.Context(), tc.ref, pb.WatchAuthority_WATCH_AUTHORITY_OPERATOR)
				if tc.allowed && (err != nil || !delegated || !actor.Delegated || actor.ApprovedBy != grant.Approver || actor.ID != runID.String()) {
					t.Fatalf("approved paired effort grant lost: %+v delegated=%v err=%v", actor, delegated, err)
				}
				if !tc.allowed && err == nil {
					t.Fatal("uncovered rule crossed operation and target")
				}
			})))
			req := httptest.NewRequest(http.MethodPost, "http://localhost/effort", nil)
			req.Header.Set(cliutil.HeaderAgentIdentityToken, managementBaselineToken)
			req.Header.Set(cliutil.HeaderAgentOperationToken, managementOperationToken)
			handler.ServeHTTP(httptest.NewRecorder(), req)
		})
	}
}
