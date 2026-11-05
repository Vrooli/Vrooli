package wiring

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/handlers"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
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

func TestEffortAuthorizerUsesVerifiedRunClaims(t *testing.T) {
	id := uuid.New()
	a := watchActionAuthorizer{orchestrator: identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: id, Subject: "signed-owner", Scopes: []string{"exact-delegation"}, ProfileKey: "not-authority"}}}}
	actor, err := a.AuthorizeEffortAction(context.Background(), "signed-run-token", domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT)
	if err != nil || actor.ID != id.String() || len(actor.Scopes) != 1 || actor.Operator {
		t.Fatal(actor, err)
	}
	a.orchestrator = identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: false, Claims: &identity.Claims{RunID: id, Subject: "forged-owner"}}}
	if _, err = a.AuthorizeEffortAction(context.Background(), "forged", domainpb.WatchAuthority_WATCH_AUTHORITY_FAMILY_PARENT); !errors.Is(err, handlers.ErrWatchActionUnauthenticated) {
		t.Fatal("unverified claims admitted", err)
	}
}

// Stub only the external token verifier, not the production effort actor.
func TestEffortActorPreservesVerifiedOwnerScopesWithoutAliasing(t *testing.T) {
	scopes := []string{"agent-manager:read", "agent-manager:write"}
	principal := coreidentity.Principal{Subject: "verified-owner", Kind: coreidentity.ActorHuman, Verified: true, Scopes: slices.Clone(scopes), ExpiresAt: time.Now().Add(time.Hour)}
	authorizer := watchActionAuthorizer{owners: ownerValidatorStub{identity: principal}}
	actor, err := authorizer.AuthorizeEffortAction(t.Context(), "private-owner-fixture", domainpb.WatchAuthority_WATCH_AUTHORITY_OPERATOR)
	if err != nil || actor.ID != principal.Subject || !actor.Operator || !slices.Equal(actor.Scopes, principal.Scopes) {
		t.Fatal("real authorizer did not preserve the verified owner ceiling", err)
	}
	actor.Scopes[0] = "must-not-mutate-principal"
	if !slices.Equal(principal.Scopes, scopes) {
		t.Fatal("actor aliases verifier-owned scopes")
	}
}
