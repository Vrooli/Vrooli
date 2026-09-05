package wiring

import (
	"context"
	"errors"
	"testing"

	"agent-manager/internal/handlers"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/owneridentity"
	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type identityVerifierStub struct {
	result *orchestration.IdentityVerifyResult
}

func (s identityVerifierStub) VerifyIdentityToken(context.Context, string) (*orchestration.IdentityVerifyResult, error) {
	return s.result, nil
}

type ownerValidatorStub struct {
	identity owneridentity.Identity
	err      error
}

func (s ownerValidatorStub) Validate(context.Context, string) (owneridentity.Identity, error) {
	return s.identity, s.err
}

func TestWatchActionAuthorizerRejectsClaimsAndDerivesAuthenticatedIdentity(t *testing.T) {
	parentID := uuid.New()
	authorizer := watchActionAuthorizer{
		orchestrator: identityVerifierStub{result: &orchestration.IdentityVerifyResult{Valid: true, Claims: &identity.Claims{RunID: parentID}}},
		owners:       ownerValidatorStub{identity: owneridentity.Identity{Subject: "operator-1", Scopes: []string{"agent-manager:supervise"}}},
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
	authorizer.owners = ownerValidatorStub{identity: owneridentity.Identity{Subject: "operator-1"}}
	if err := authorizer.AuthorizeWatchAction(context.Background(), "owner-token", operator); !errors.Is(err, handlers.ErrWatchActionForbidden) {
		t.Fatalf("unscoped operator err=%v", err)
	}
}
