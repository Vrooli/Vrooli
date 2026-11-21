package session

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	sessionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session"
	domain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/session"
	"google.golang.org/protobuf/types/known/structpb"
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

func TestProfileSessionHandlerUsesVerifiedActorAndTypedAnswers(t *testing.T) {
	var got domain.ProfileSession
	handler := NewConnectHandler(domain.Service{
		SaveProfileSessionFn: func(_ context.Context, value domain.ProfileSession, _ string) (*domain.ProfileSession, error) {
			got = value
			return &value, nil
		},
	})
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "verified-operator", Source: identity.SourceScenarioAuthenticator, Verified: true,
	})
	answers, err := structpb.NewStruct(map[string]any{"purposes": []any{"develop-apps"}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := handler.SaveProfileSession(ctx, connect.NewRequest(&sessionv1.SaveProfileSessionRequest{
		Target:  "local",
		Actor:   "attacker-supplied-actor",
		Session: &sessionv1.ProfileSession{Mode: "guided", ProfileId: "develop-and-publish", BaseRevision: "r1", Answers: answers},
	}))
	if err != nil {
		t.Fatalf("SaveProfileSession() error = %v", err)
	}
	want := string(identity.SourceScenarioAuthenticator) + ":verified-operator"
	if got.Actor != want || response.Msg.GetSession().GetActor() != want {
		t.Fatalf("profile actor = %q / %q, want verified principal %q", got.Actor, response.Msg.GetSession().GetActor(), want)
	}
	if string(got.Answers["purposes"]) != `["develop-apps"]` {
		t.Fatalf("typed answer = %s", got.Answers["purposes"])
	}
}
