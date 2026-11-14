package main

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/identity"
)

func TestVerifiedApplyActorRequiresVerifiedSubjectAndPreservesProviderSource(t *testing.T) {
	if _, _, err := verifiedApplyActor(context.Background()); err == nil {
		t.Fatal("anonymous apply actor was accepted")
	}
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-1", Source: identity.SourceScenarioAuthenticator,
	})
	if _, _, err := verifiedApplyActor(ctx); err == nil {
		t.Fatal("unverified apply actor was accepted")
	}
	ctx = identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-1", Source: identity.SourceScenarioAuthenticator, Verified: true,
	})
	source, subject, err := verifiedApplyActor(ctx)
	if err != nil || source != string(identity.SourceScenarioAuthenticator) || subject != "operator-1" {
		t.Fatalf("actor = %q/%q, err = %v", source, subject, err)
	}
}
