package session

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	domain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
)

func TestDraftHandlersBindOwnershipToVerifiedPrincipal(t *testing.T) {
	var gotActor string
	handler := NewConnectHandler(domain.Service{
		SaveDraftFn: func(_ context.Context, target, actor, expectedRevision, baseRevision, stepID string, choices map[string]string) (domain.Draft, error) {
			gotActor = actor
			return domain.Draft{Target: target, Actor: actor, BaseRevision: baseRevision, Revision: expectedRevision, StepID: stepID, Choices: choices}, nil
		},
	})
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "verified-operator", Source: identity.SourceScenarioAuthenticator, Verified: true,
	})
	response, err := handler.SaveDraft(ctx, connect.NewRequest(&sessionv1.SaveDraftRequest{
		Target: "local", Actor: "attacker-supplied-actor", BaseRevision: "base", StepId: "scenarios",
	}))
	if err != nil {
		t.Fatalf("SaveDraft() error = %v", err)
	}
	want := string(identity.SourceScenarioAuthenticator) + ":verified-operator"
	if gotActor != want || response.Msg.GetDraft().GetActor() != want {
		t.Fatalf("draft actor = %q / %q, want verified principal %q", gotActor, response.Msg.GetDraft().GetActor(), want)
	}
}
