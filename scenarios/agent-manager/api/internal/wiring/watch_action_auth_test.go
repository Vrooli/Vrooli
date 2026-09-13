package wiring

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/handlers"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/testutil"
	"agent-manager/internal/supervision"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type identityVerifierStub struct {
	result *orchestration.IdentityVerifyResult
}

func (s identityVerifierStub) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return s.result, nil
}

type ownerValidatorStub struct {
	identity coreidentity.Principal
	err      error
}

func (s ownerValidatorStub) Verify(context.Context, string) (coreidentity.Principal, error) {
	return s.identity, s.err
}

func TestWatchActionAuthorizerRejectsClaimsAndDerivesAuthenticatedIdentity(t *testing.T) {
	parentID := uuid.New()
	authorizer := watchActionAuthorizer{
		orchestrator: identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: parentID}}},
		owners:       ownerValidatorStub{identity: coreidentity.Principal{Subject: "operator-1", Kind: coreidentity.ActorHuman, Verified: true, Scopes: []string{"agent-manager:supervise"}}},
	}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "", &domainpb.RequestCohortWatchActionRequest{Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR}); !errors.Is(err, handlers.ErrWatchActionUnauthenticated) {
		t.Fatalf("missing token err=%v", err)
	}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "token", &domainpb.RequestCohortWatchActionRequest{Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_SYSTEM}); !errors.Is(err, handlers.ErrWatchActionForbidden) {
		t.Fatalf("external system claim err=%v", err)
	}
	operator := &domainpb.RequestCohortWatchActionRequest{Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR, RequestedBy: "forged"}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "owner-token", operator); err != nil || operator.GetRequestedBy() != "operator-1" {
		t.Fatalf("operator=%+v err=%v", operator, err)
	}
	parent := &domainpb.RequestCohortWatchActionRequest{Authority: domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT, RequestedBy: "forged"}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "run-token", parent); err != nil || parent.GetRequestedBy() != parentID.String() {
		t.Fatalf("parent=%+v err=%v", parent, err)
	}
	authorizer.owners = ownerValidatorStub{identity: coreidentity.Principal{Subject: "operator-1", Kind: coreidentity.ActorHuman, Verified: true}}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "owner-token", operator); !errors.Is(err, handlers.ErrWatchActionForbidden) {
		t.Fatalf("unscoped operator err=%v", err)
	}
}

func TestEffortAuthorizerUsesVerifiedClaimsForStableWakeDelegation(t *testing.T) {
	id := uuid.New()
	a := watchActionAuthorizer{orchestrator: identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: id, Subject: "signed-owner", Scopes: []string{"exact-delegation"}, ProfileKey: "not-authority"}}}}
	actor, err := a.AuthorizeEffortAction(context.Background(), "signed-run-token", domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT)
	if err != nil || actor.ID != id.String() || actor.OwnerSubject != "signed-owner" || len(actor.Scopes) != 1 || actor.Operator {
		t.Fatal(actor, err)
	}
	a.orchestrator = identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: false, Claims: &identity.Claims{RunID: id, Subject: "forged-owner"}}}
	if _, err = a.AuthorizeEffortAction(context.Background(), "forged", domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT); !errors.Is(err, handlers.ErrWatchActionUnauthenticated) {
		t.Fatal("unverified claims admitted", err)
	}
}

// Stub only the external token verifier, not the production effort actor.
// This reproduces the coarse operator/write gate followed by the stricter
// recurring delegation ceiling through the same handler selector as production.
func TestIssueDispatchUsesRealOwnerAuthorizerScopeCeiling(t *testing.T) {
	for _, tc := range []struct {
		name   string
		scopes []string
		allow  bool
	}{
		{"coarse-write-is-not-supervise", []string{"agent-manager:read", "agent-manager:write", "vrooli-bridge:read", "vrooli-bridge:write"}, false},
		{"explicit-supervise", []string{"agent-manager:supervise"}, true},
		{"existing-wildcard", []string{"*"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, cleanup := testutil.SetupTestDB(t)
			t.Cleanup(cleanup)
			repo := supervision.NewRepository(db)
			service := supervision.NewEffortService(repo, nil, nil, supervision.EffortDiscoveryConfig{Root: t.TempDir()})
			provisioned := false
			service.ConfigureDispatch([]byte("isolated-owner-authorizer-test-signer"), func(string) error {
				provisioned = true
				return nil
			}, func(context.Context, string) error { return nil })
			principal := coreidentity.Principal{Subject: "verified-owner", Kind: coreidentity.ActorHuman, Verified: true, Scopes: slices.Clone(tc.scopes), ExpiresAt: time.Now().Add(time.Hour)}
			authorizer := watchActionAuthorizer{owners: ownerValidatorStub{identity: principal}}
			actor, err := authorizer.AuthorizeEffortAction(t.Context(), "private-owner-fixture", domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR)
			if err != nil || actor.ID != principal.Subject || actor.OwnerSubject != principal.Subject || !actor.Operator || !slices.Equal(actor.Scopes, principal.Scopes) {
				t.Fatal("real authorizer did not preserve the verified owner ceiling", err)
			}
			actor.Scopes[0] = "must-not-mutate-principal"
			if !slices.Equal(principal.Scopes, tc.scopes) {
				t.Fatal("actor aliases verifier-owned scopes")
			}
			rpc := handlers.NewAgentManagerConnectHandler(nil, &supervision.Service{Efforts: service})
			rpc.SetWatchActionAuthorizer(authorizer)
			enroll := connect.NewRequest(&domainpb.EnrollEffortRequest{Enrollment: &domainpb.EffortEnrollment{EffortRef: "service:standing-fixture", SupervisorOwnerSubject: principal.Subject, SupervisorScope: supervision.SupervisorDispatchScope}, IdempotencyKey: "enroll"})
			enroll.Header().Set("Authorization", "Bearer private-owner-fixture")
			enrolled, err := rpc.EnrollEffort(t.Context(), enroll)
			if err != nil {
				t.Fatal(err)
			}
			issue := connect.NewRequest(&api.IssueSupervisorDispatchRequest{EffortRef: enrolled.Msg.EffortRef, ExpectedRevision: enrolled.Msg.Revision, TeamId: "supervisors", MemberId: "supervisor", ProfileKey: "qualified", MaximumRuns: 1, MinimumIntervalSeconds: 60, ExpiresAt: timestamppb.New(time.Now().Add(30 * time.Minute)), IdempotencyKey: "issue"})
			issue.Header().Set("Authorization", "Bearer private-owner-fixture")
			issued, err := rpc.IssueSupervisorDispatch(t.Context(), issue)
			if !tc.allow {
				if !errors.Is(err, supervision.ErrDispatchAuthority) || !strings.Contains(err.Error(), "issue-dispatch predicate=owner_scope_missing") || strings.Contains(err.Error(), "private-owner-fixture") || provisioned {
					t.Fatal("coarse write must be refused without credential disclosure or provisioning", err)
				}
				stored, _, readErr := repo.GetEffort(t.Context(), enrolled.Msg.EffortRef)
				if readErr != nil || stored.Revision != enrolled.Msg.Revision || stored.DispatchAuthorization != nil {
					t.Fatal("refusal mutated durable enrollment", readErr)
				}
				return
			}
			if err != nil || !provisioned || !slices.Equal(issued.Msg.GetDispatchAuthorization().GetScopes(), []string{supervision.SupervisorDispatchScope}) {
				t.Fatal("verified ceiling was not attenuated to exact supervisor authority", err)
			}
		})
	}
}
