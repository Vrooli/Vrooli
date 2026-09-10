package operatorinputs

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	operatorinputsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	internaloperatorinputs "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorinputs"
)

func TestResolveMutationRequiresVerifiedPrincipal(t *testing.T) {
	err := authz.RequireMutation(context.Background())
	if connect.CodeOf(err) != connect.CodeUnauthenticated || !strings.Contains(err.Error(), "verified onboarding operator") {
		t.Fatalf("authorization error = %v", err)
	}
}

func TestResolveMutationAcceptsVerifiedHumanCapability(t *testing.T) {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "osuser:1001", Verified: true,
		Source: identity.SourcePersonalLocal, Scopes: []string{authz.MutationCapability},
	})
	if err := authz.RequireMutation(ctx); err != nil {
		t.Fatalf("verified local authorization = %v", err)
	}
}

func TestResolveOperatorInputsMapsStaleRevisionToAborted(t *testing.T) {
	handler := NewConnectHandler(internaloperatorinputs.Service{
		CurrentRevision: func(context.Context) (string, error) { return "new", nil },
	})
	_, err := handler.ResolveOperatorInputs(context.Background(), connect.NewRequest(&operatorinputsv1.ResolveOperatorInputsRequest{
		Target: "local", ExpectedRevision: "old", Answers: []*operatorinputsv1.Answer{{RequestId: "input", Value: "value"}},
	}))
	if connect.CodeOf(err) != connect.CodeAborted {
		t.Fatalf("Connect code = %v, want aborted; error = %v", connect.CodeOf(err), err)
	}
	if !strings.Contains(err.Error(), "reload the target questions") {
		t.Fatalf("error = %v, want actionable conflict explanation", err)
	}
}
