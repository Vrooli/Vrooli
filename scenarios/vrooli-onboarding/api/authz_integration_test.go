package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vrooli/api-core/authn"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply/applyv1connect"
	capabilitiesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/capabilities/capabilitiesv1connect"
	credentialconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/credentials/credentialsv1connect"
	hostconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/host/hostv1connect"
	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"
	operatorstateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	readinessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/readiness/readinessv1connect"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	sessionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/session/sessionv1connect"
)

func TestRemoteMutationCannotUseLoopbackPersonalLocalShortcut(t *testing.T) {
	server := NewServer()
	request := newJSONRequest(http.MethodPost, "/vrooli.vrooli_onboarding.v1.session.SessionService/AdvanceSessionStep", `{"target":"local","stepId":"scenarios"}`)
	request.RemoteAddr = "198.51.100.20:44000"
	response := recordRequest(server.Handler(), request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous remote mutation = %d: %s", response.Code, response.Body.String())
	}
}

func TestPersonalLocalAuthenticationRequiresRuntimeSessionAndPreservesAgentProvenance(t *testing.T) {
	tokenPath := t.TempDir() + "/runtime-token"
	if err := os.WriteFile(tokenPath, []byte("runtime-session-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_AUTH_MODE", "personal_local")
	t.Setenv("VROOLI_AUTH_LOCAL_TOKEN_FILE", tokenPath)
	config := onboardingAuthenticationConfig()

	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/api", nil)
	request.RemoteAddr = "127.0.0.1:44000"
	if _, err := config.Authenticate(context.Background(), request); err == nil {
		t.Fatal("loopback request without runtime session unexpectedly authenticated")
	}
	request.Header.Set("Authorization", "Bearer wrong-token")
	if _, err := config.Authenticate(context.Background(), request); err == nil {
		t.Fatal("wrong runtime session unexpectedly authenticated")
	}
	request.Header.Set("Authorization", "LocalSession runtime-session-token")
	principal, err := config.Authenticate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if principal.Kind != identity.ActorHuman || principal.Subject == "" || principal.Subject == "osuser:1001" {
		t.Fatalf("personal-local principal = %#v", principal)
	}

	agentContext := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1",
	})
	if _, err := config.Authenticate(agentContext, request); err == nil {
		t.Fatal("verified agent provenance was promoted to a human local session")
	}
}

func TestEveryDeclaredMutationRouteRejectsAnonymousRemoteRequests(t *testing.T) {
	server := NewServer()
	procedures := []string{
		applyconnect.ApplyServiceStartApplyProcedure,
		applyconnect.ApplyServiceReviewApplyProcedure,
		applyconnect.ApplyServiceCancelApplyProcedure,
		capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure,
		capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure,
		credentialconnect.CredentialsServiceProvisionCredentialProcedure,
		hostconnect.HostServicePatchHostSafeguardConfigProcedure,
		hostconnect.HostServiceSetNotificationRecipientProcedure,
		operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure,
		operatorstateconnect.OperatorStateServicePatchOperatorStateProcedure,
		readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure,
		selectionconnect.SelectionServiceAcceptRecommendationProcedure,
		selectionconnect.SelectionServiceCreateHandoffProcedure,
		sessionconnect.SessionServiceAdvanceSessionStepProcedure,
		sessionconnect.SessionServiceSaveDraftProcedure,
		sessionconnect.SessionServiceDiscardDraftProcedure,
	}
	for _, procedure := range procedures {
		t.Run(procedure, func(t *testing.T) {
			request := newJSONRequest(http.MethodPost, procedure, `{}`)
			request.RemoteAddr = "198.51.100.20:44000"
			request.Header.Set("Origin", "https://attacker.example")
			request.Header.Set("X-Forwarded-For", "127.0.0.1")
			request.Header.Set("X-Real-IP", "127.0.0.1")
			request.Header.Set("X-Forwarded-Host", "localhost")
			request.Header.Set("Authorization", "Bearer forged-local-token")
			response := recordRequest(server.Handler(), request)
			if response.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous remote mutation = %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

// TestEveryDeclaredMutationRouteRejectsProviderAuthorityFailures keeps the
// production mutation boundary fail-closed when authentication has identified
// a token as invalid (for example, a wrong audience) or expired. The provider
// owns claim verification; this matrix proves no onboarding handler treats a
// failed provider result as a usable principal or reaches a side effect.
func TestEveryDeclaredMutationRouteRejectsProviderAuthorityFailures(t *testing.T) {
	procedures := []string{
		applyconnect.ApplyServiceStartApplyProcedure,
		applyconnect.ApplyServiceReviewApplyProcedure,
		applyconnect.ApplyServiceCancelApplyProcedure,
		capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure,
		capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure,
		credentialconnect.CredentialsServiceProvisionCredentialProcedure,
		hostconnect.HostServicePatchHostSafeguardConfigProcedure,
		hostconnect.HostServiceSetNotificationRecipientProcedure,
		operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure,
		operatorstateconnect.OperatorStateServicePatchOperatorStateProcedure,
		readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure,
		selectionconnect.SelectionServiceAcceptRecommendationProcedure,
		selectionconnect.SelectionServiceCreateHandoffProcedure,
		sessionconnect.SessionServiceAdvanceSessionStepProcedure,
		sessionconnect.SessionServiceSaveDraftProcedure,
		sessionconnect.SessionServiceDiscardDraftProcedure,
	}
	for _, tc := range []struct {
		name  string
		class identity.FailureClass
	}{
		{name: "wrong audience", class: identity.FailureInvalid},
		{name: "expired authority", class: identity.FailureExpired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer()
			server.auth = authn.Config{Providers: []authn.Provider{authorityFailureProvider{class: tc.class}}}
			for _, procedure := range procedures {
				t.Run(procedure, func(t *testing.T) {
					request := newJSONRequest(http.MethodPost, procedure, `{}`)
					request.RemoteAddr = "198.51.100.20:44000"
					request.Header.Set("Origin", "https://attacker.example")
					request.Header.Set("Authorization", "Bearer structurally-valid-but-rejected")
					response := recordRequest(server.Handler(), request)
					if response.Code != http.StatusUnauthorized {
						t.Fatalf("provider failure reached mutation boundary with status %d: %s", response.Code, response.Body.String())
					}
				})
			}
		})
	}
}

func TestEveryDeclaredMutationRouteAcceptsAuthenticatedLocalAdmission(t *testing.T) {
	procedures := []string{
		applyconnect.ApplyServiceStartApplyProcedure,
		applyconnect.ApplyServiceReviewApplyProcedure,
		applyconnect.ApplyServiceCancelApplyProcedure,
		capabilitiesconnect.CapabilitiesServicePreviewCapabilityProcedure,
		capabilitiesconnect.CapabilitiesServiceApplyCapabilityProcedure,
		credentialconnect.CredentialsServiceProvisionCredentialProcedure,
		hostconnect.HostServicePatchHostSafeguardConfigProcedure,
		hostconnect.HostServiceSetNotificationRecipientProcedure,
		operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure,
		operatorstateconnect.OperatorStateServicePatchOperatorStateProcedure,
		readinessconnect.ReadinessServiceAcknowledgeDegradedReadinessProcedure,
		selectionconnect.SelectionServiceAcceptRecommendationProcedure,
		selectionconnect.SelectionServiceCreateHandoffProcedure,
		sessionconnect.SessionServiceAdvanceSessionStepProcedure,
		sessionconnect.SessionServiceSaveDraftProcedure,
		sessionconnect.SessionServiceDiscardDraftProcedure,
	}
	for _, procedure := range procedures {
		t.Run(procedure, func(t *testing.T) {
			request := newJSONRequest(http.MethodPost, procedure, `{}`)
			request.RemoteAddr = "127.0.0.1:44000"
			request.Header.Set("Authorization", "LocalSession test-personal-local-session")
			response := recordRequest(NewServer().Handler(), request)
			if response.Code == http.StatusUnauthorized || response.Code == http.StatusForbidden {
				t.Fatalf("authenticated local mutation was rejected at admission with status %d: %s", response.Code, response.Body.String())
			}
		})
	}
}

type authorityFailureProvider struct {
	class identity.FailureClass
}

func (p authorityFailureProvider) Source() identity.AuthSource {
	return identity.SourceScenarioAuthenticator
}

func (p authorityFailureProvider) VerifyRequest(context.Context, *http.Request) (identity.Principal, error) {
	return identity.Principal{}, identity.NewFailure(p.class, p.Source())
}
