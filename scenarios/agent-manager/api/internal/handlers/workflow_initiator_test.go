package handlers

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"github.com/vrooli/api-core/identity"
)

func TestWorkflowInitiatorUsesVerifiedIdentityEvidence(t *testing.T) {
	verifiedHuman := identity.Principal{Kind: identity.ActorHuman, Subject: "owner@example.test", Verified: true, Source: identity.SourceScenarioAuthenticator}
	for _, tc := range []struct {
		name  string
		ctx   context.Context
		token string
		want  domain.WorkflowInitiator
	}{
		{name: "missing proof is programmatic", ctx: context.Background(), want: domain.WorkflowInitiatorProgrammatic},
		{name: "verified human is human", ctx: identity.WithPrincipal(context.Background(), verifiedHuman), want: domain.WorkflowInitiatorHuman},
		{name: "offered agent proof remains agent", ctx: context.Background(), token: "invalid-or-unverified-agent-token", want: domain.WorkflowInitiatorAgent},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := workflowInitiator(tc.ctx, tc.token); got != tc.want {
				t.Fatalf("workflowInitiator() = %q, want %q", got, tc.want)
			}
		})
	}
}
