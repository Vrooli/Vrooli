package authz

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
)

func TestRequireMutationDeniesAnonymousAndUnderprivilegedCallers(t *testing.T) {
	if err := RequireMutation(context.Background()); connect.CodeOf(err) != connect.CodeUnauthenticated {
		t.Fatalf("anonymous code = %v, error = %v", connect.CodeOf(err), err)
	}
	underprivileged := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-1", Verified: true,
		Source: identity.SourceScenarioAuthenticator, Scopes: []string{"vrooli-onboarding:read"},
	})
	if err := RequireMutation(underprivileged); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("underprivileged code = %v, error = %v", connect.CodeOf(err), err)
	}
}

func TestRequireMutationDeniesAgentsAndAcceptsScopedHuman(t *testing.T) {
	agent := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorAgent, Subject: "run-1", Verified: true,
		Source: identity.SourceAgentProvenance, Scopes: []string{MutationCapability},
	})
	if err := RequireMutation(agent); connect.CodeOf(err) != connect.CodePermissionDenied {
		t.Fatalf("agent code = %v, error = %v", connect.CodeOf(err), err)
	}
	human := identity.WithPrincipal(context.Background(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-1", Verified: true,
		Source: identity.SourceScenarioAuthenticator, Scopes: []string{MutationCapability},
	})
	if err := RequireMutation(human); err != nil {
		t.Fatalf("scoped human rejected: %v", err)
	}
}

func TestIsMutationProcedureCoversDeclaredWriteBoundary(t *testing.T) {
	for _, procedure := range []string{
		"/vrooli.vrooli_onboarding.v1.apply.ApplyService/ReviewApply",
		"/vrooli.vrooli_onboarding.v1.apply.ApplyService/StartApply",
		"/vrooli.vrooli_onboarding.v1.apply.ApplyService/CancelApply",
		"/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/PreviewCapability",
		"/vrooli.vrooli_onboarding.v1.capabilities.CapabilitiesService/ApplyCapability",
		"/vrooli.vrooli_onboarding.v1.credentials.CredentialsService/ProvisionCredential",
		"/vrooli.vrooli_onboarding.v1.host.HostService/PatchHostSafeguardConfig",
		"/vrooli.vrooli_onboarding.v1.host.HostService/SetNotificationRecipient",
		"/vrooli.vrooli_onboarding.v1.operatorinputs.OperatorInputsService/ResolveOperatorInputs",
		"/vrooli.vrooli_onboarding.v1.operatorstate.OperatorStateService/PatchOperatorState",
		"/vrooli.vrooli_onboarding.v1.readiness.ReadinessService/AcknowledgeDegradedReadiness",
		"/vrooli.vrooli_onboarding.v1.selection.SelectionService/AcceptRecommendation",
		"/vrooli.vrooli_onboarding.v1.selection.SelectionService/CreateHandoff",
		"/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep",
		"/vrooli.vrooli_onboarding.v1.session.SessionService/SaveDraft",
		"/vrooli.vrooli_onboarding.v1.session.SessionService/DiscardDraft",
	} {
		if !IsMutationProcedure(procedure) {
			t.Errorf("mutation procedure not protected: %s", procedure)
		}
	}
	if IsMutationProcedure("/vrooli.vrooli_onboarding.v1.apply.ApplyService/GetApplyPlan") {
		t.Fatal("read procedure was classified as a mutation")
	}
}
