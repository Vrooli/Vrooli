package duplex

import (
	"context"
	"testing"

	"audio-tools/internal/providercatalog"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/identity"
)

func eligibleTestProvider(providerID, modelID string) bool {
	return providerID == "openai-gpt-live" && modelID == "gpt-live-1.1"
}

func validAdmissionRequest() Request {
	return Request{
		ProviderID: "openai-gpt-live", Model: "gpt-live-1.1",
		RemoteAllowed: true, Consent: true, ConsentReceipt: "opaque", BudgetMicros: 3000,
	}
}

type consentVerifierFixture struct{}

func (consentVerifierFixture) Verify(_ context.Context, subject, receipt, source string) error {
	if subject == "human" && receipt == "opaque" && source == declaredOpenAIConsentSource() {
		return nil
	}
	return ErrConsentInvalid
}

func declaredOpenAIConsentSource() string {
	requirement, ok := providercatalog.ConsentRequirementFor("openai-gpt-live")
	if !ok {
		panic("openai-gpt-live consent requirement missing from provider catalog")
	}
	return string(requirement.Source)
}

func validHumanContext() context.Context {
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	return ctx
}

func TestAdmissionPolicyRequiresConsentEligibilityAndBoundedSpend(t *testing.T) {
	policy := NewAdmissionPolicy(NewBudget(5000), 4000, eligibleTestProvider).WithConsentVerifier(consentVerifierFixture{})
	tests := []struct {
		name string
		edit func(*Request)
		want error
	}{
		{name: "remote processing denied", edit: func(r *Request) { r.RemoteAllowed = false }, want: ErrRemoteProcessingDenied},
		{name: "consent missing", edit: func(r *Request) { r.Consent = false }, want: ErrConsentRequired},
		{name: "provider not eligible", edit: func(r *Request) { r.ProviderID = "unknown" }, want: ErrProviderUnavailable},
		{name: "model not eligible", edit: func(r *Request) { r.Model = "unknown" }, want: ErrProviderUnavailable},
		{name: "zero session budget", edit: func(r *Request) { r.BudgetMicros = 0 }, want: ErrSessionBudgetRequired},
		{name: "above configured maximum", edit: func(r *Request) { r.BudgetMicros = 4001 }, want: ErrSessionBudgetTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := validAdmissionRequest()
			tt.edit(&req)
			reservation, err := policy.Reserve(validHumanContext(), req)
			require.ErrorIs(t, err, tt.want)
			require.Nil(t, reservation)
			require.Equal(t, uint64(0), policy.budget.Snapshot().ReservedMicros)
		})
	}
}

func TestAdmissionPolicyReservesAtomicallyAndAuthorizeDoesNotLeak(t *testing.T) {
	budget := NewBudget(5000)
	policy := NewAdmissionPolicy(budget, 4000, eligibleTestProvider).WithConsentVerifier(consentVerifierFixture{})
	req := validAdmissionRequest()
	req.ConsentReceipt = "opaque"
	require.NoError(t, policy.Authorize(validHumanContext(), req))
	require.Equal(t, uint64(0), budget.Snapshot().ReservedMicros)

	first, err := policy.Reserve(validHumanContext(), req)
	require.NoError(t, err)
	require.Equal(t, uint64(3000), budget.Snapshot().ReservedMicros)
	_, err = policy.Reserve(validHumanContext(), req)
	require.ErrorIs(t, err, ErrBudgetExceeded)
	first.Release()
	require.Equal(t, uint64(0), budget.Snapshot().ReservedMicros)
}

func TestAdmissionPolicyRejectsCanceledContextAndMissingRules(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	policy := NewAdmissionPolicy(NewBudget(1000), 1000, eligibleTestProvider)
	_, err := policy.Reserve(ctx, validAdmissionRequest())
	require.ErrorIs(t, err, context.Canceled)
	_, err = (*AdmissionPolicy)(nil).Reserve(context.Background(), validAdmissionRequest())
	require.ErrorIs(t, err, ErrNilAdmission)
	_, err = NewAdmissionPolicy(NewBudget(1000), 1000, nil).Reserve(context.Background(), validAdmissionRequest())
	require.ErrorIs(t, err, ErrNilAdmission)
}

func TestAccountCreditPolicyDoesNotDependOnProcessLocalBudget(t *testing.T) {
	policy := NewAdmissionPolicy(nil, 4000, eligibleTestProvider).WithConsentVerifier(consentVerifierFixture{})
	req := validAdmissionRequest()
	req.ConsentReceipt = "opaque"
	require.NoError(t, policy.Authorize(validHumanContext(), req), "durable-wallet policy authorization must not depend on a process-local budget")
	_, err := policy.Reserve(validHumanContext(), req)
	require.ErrorIs(t, err, ErrNilAdmission, "local-only reservation without a local budget must fail closed")
}

func TestCatalogEligibilityFailsClosedWhileGPTLiveIsUnqualified(t *testing.T) {
	if CatalogEligibility("openai-gpt-live", "gpt-live-1.1") {
		t.Fatal("planned/unqualified provider and model must not be admissible")
	}
	if CatalogEligibility("openai-gpt-live", "unknown-model") {
		t.Fatal("undeclared model must not be admissible")
	}
}

type captureConsentVerifier struct {
	calls                    int
	subject, receipt, source string
	err                      error
}

func (v *captureConsentVerifier) Verify(_ context.Context, subject, receipt, source string) error {
	v.calls++
	v.subject = subject
	v.receipt = receipt
	v.source = source
	return v.err
}

func TestAdmissionRequiresVerifiedOwnerAndGatewayReceiptForGPTLive(t *testing.T) {
	verifier := &captureConsentVerifier{}
	policy := NewAdmissionPolicy(NewBudget(5000), 4000, eligibleTestProvider).WithConsentVerifier(verifier)
	req := validAdmissionRequest()
	require.ErrorIs(t, policy.Authorize(context.Background(), req), ErrConsentInvalid)
	require.Equal(t, 0, verifier.calls)
	ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true})
	require.NoError(t, policy.Authorize(ctx, req))
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, "human", verifier.subject)
	require.Equal(t, "opaque", verifier.receipt)
	require.Equal(t, declaredOpenAIConsentSource(), verifier.source)
	verifier.err = ErrConsentInvalid
	require.ErrorIs(t, policy.Authorize(ctx, req), ErrConsentInvalid)
}

func TestAdmissionUsesDeclaredConsentContractForAnyDuplexProvider(t *testing.T) {
	const providerID, modelID, source = "fixture-remote-voice", "fixture-live-model", "fixture-source-bound-consent"
	verifier := &captureConsentVerifier{}
	policy := NewAdmissionPolicy(NewBudget(5000), 4000, func(provider, model string) bool {
		return provider == providerID && model == modelID
	}).WithConsentVerifier(verifier)
	policy.consentRequirement = func(id string) (providercatalog.ConsentRequirement, bool) {
		if id != providerID {
			return providercatalog.ConsentRequirement{}, false
		}
		return providercatalog.ConsentRequirement{Source: providercatalog.ConsentSource(source), RevalidateActiveSessions: true}, true
	}
	req := Request{ProviderID: providerID, Model: modelID, RemoteAllowed: true, Consent: true, ConsentReceipt: "opaque", BudgetMicros: 1000}
	ctx := validHumanContext()
	require.NoError(t, policy.Authorize(ctx, req))
	require.Equal(t, 1, verifier.calls)
	require.Equal(t, source, verifier.source)
	require.True(t, policy.RequiresActiveConsentRevalidation(req))

	for name, principal := range map[string]identity.Principal{
		"unverified":    {Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceConsumerAccessToken},
		"service actor": {Kind: identity.ActorService, Subject: "human", Source: identity.SourceConsumerAccessToken, Verified: true},
		"wrong source":  {Kind: identity.ActorHuman, Subject: "human", Source: identity.SourceScenarioAuthenticator, Verified: true},
	} {
		t.Run(name, func(t *testing.T) {
			calls := verifier.calls
			badCtx := identity.WithPrincipal(context.Background(), principal)
			require.ErrorIs(t, policy.Authorize(badCtx, req), ErrConsentInvalid)
			require.Equal(t, calls, verifier.calls, "invalid principal must not reach consent authority")
		})
	}
	missingReceipt := req
	missingReceipt.ConsentReceipt = ""
	require.ErrorIs(t, policy.Authorize(ctx, missingReceipt), ErrConsentInvalid)
}

func TestAdmissionDoesNotRequireProviderConsentWithoutCatalogRequirement(t *testing.T) {
	policy := NewAdmissionPolicy(NewBudget(5000), 4000, func(provider, model string) bool {
		return provider == "fixture-unbound-provider" && model == "fixture-model"
	})
	policy.consentRequirement = func(string) (providercatalog.ConsentRequirement, bool) {
		return providercatalog.ConsentRequirement{}, false
	}
	req := Request{ProviderID: "fixture-unbound-provider", Model: "fixture-model", RemoteAllowed: true, Consent: true, BudgetMicros: 1000}
	require.NoError(t, policy.Authorize(context.Background(), req))
	require.False(t, policy.RequiresActiveConsentRevalidation(req))
}
