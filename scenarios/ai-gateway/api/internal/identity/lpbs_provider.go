package identity

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/identity"
	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// LPBSConsumerProvider verifies the signed LPBS browser access token used by
// consumer-funded scenarios. It trusts only LPBS's published JWKS and issuer.
type LPBSConsumerProvider struct {
	issuer string
	cache  *consumeridentity.Cache
}

// RunCaller is the verified LPBS account identity for one typed Gateway Run.
// It is derived only from an LPBS signature bound to the request's role and
// root operation id; request JSON never supplies these fields.
type RunCaller struct {
	BusinessAccountID string
	ActorID           string
	AppKey            string
	OperationID       string
}

type runCallerContextKey struct{}

const RunCallerAssertionHeader = "X-Vrooli-LPBS-Run-Assertion"

var ErrRunCallerVerifierUnavailable = errors.New("LPBS Run caller verifier is unavailable")

func WithRunCaller(ctx context.Context, caller RunCaller) context.Context {
	return context.WithValue(ctx, runCallerContextKey{}, caller)
}

func RunCallerFromContext(ctx context.Context) (RunCaller, bool) {
	caller, ok := ctx.Value(runCallerContextKey{}).(RunCaller)
	return caller, ok
}

// VerifyRunCallerAssertion binds a verified credit assertion to a single
// Run role and root operation. Only LPBS can mint the assertion; the role and
// operation are checked against the signed claims before any provider work.
func (p *LPBSConsumerProvider) VerifyRunCallerAssertion(req *http.Request, raw, role, operation string) (RunCaller, error) {
	if p == nil || p.cache == nil || req == nil || strings.TrimSpace(raw) == "" {
		return RunCaller{}, identity.NewFailure(identity.FailureMissing, identity.SourceConsumerAccessToken)
	}
	claims, err := p.cache.Verify(req, raw)
	if err != nil {
		return RunCaller{}, err
	}
	assertion := claims.Credit
	if assertion == nil {
		return RunCaller{}, fmt.Errorf("LPBS Run assertion is missing its credit scope")
	}
	expected := consumeridentity.CreditAssertionExpectation{
		BusinessAccountID: assertion.BusinessAccountID,
		ActorID:           assertion.ActorID,
		AppKey:            assertion.AppKey,
		FeatureID:         "ai-gateway.run/" + strings.TrimSpace(role),
		MaxCredits:        assertion.MaxCredits,
		Audience:          consumeridentity.CreditGatewayAudience,
		OperationID:       strings.TrimSpace(operation),
		Source:            consumeridentity.CreditAssertionSource,
	}
	if err := consumeridentity.ValidateCreditAssertionClaims(claims, expected); err != nil {
		return RunCaller{}, err
	}
	return RunCaller{BusinessAccountID: assertion.BusinessAccountID, ActorID: assertion.ActorID, AppKey: assertion.AppKey, OperationID: assertion.OperationID}, nil
}

func (p *LPBSConsumerProvider) VerifyMeteredInferenceAssertion(req *http.Request, raw, operation string, owner monetization.CreditFeatureOwner) error {
	if p == nil || p.cache == nil {
		return consumeridentity.ErrKeySetUnavailable
	}
	claims, err := p.cache.Verify(req, raw)
	if err != nil {
		return err
	}
	return consumeridentity.ValidateCreditAssertionOwnerClaims(claims, consumeridentity.CreditAssertionOwnerBinding{
		AppKey: owner.AppKey, FeatureID: owner.Feature.ID, OperationID: operation,
		Source: consumeridentity.CreditAssertionSourceMeteredInference,
	})
}

func (p *LPBSConsumerProvider) VerifyCreditAssertion(req *http.Request, raw string, expected consumeridentity.CreditAssertionExpectation) error {
	if p == nil || p.cache == nil || req == nil || strings.TrimSpace(raw) == "" {
		return identity.NewFailure(identity.FailureMissing, identity.SourceConsumerAccessToken)
	}
	claims, err := p.cache.Verify(req, raw)
	if err != nil {
		return err
	}
	return consumeridentity.ValidateCreditAssertionClaims(claims, expected)
}

func NewLPBSConsumerProvider(issuer string) *LPBSConsumerProvider {
	issuer = strings.TrimSpace(issuer)
	if issuer == "" {
		issuer = "https://vrooli.com"
	}
	return NewLPBSConsumerProviderWithCache(issuer, consumeridentity.NewScenarioCache(issuer, "landing-page-business-suite"))
}

// NewLPBSConsumerProviderWithCache allows a deployment or local integration
// fixture to inject the issuer's published key cache while keeping verification
// policy centralized in this provider.
func NewLPBSConsumerProviderWithCache(issuer string, cache *consumeridentity.Cache) *LPBSConsumerProvider {
	return &LPBSConsumerProvider{issuer: strings.TrimSpace(issuer), cache: cache}
}

func (p *LPBSConsumerProvider) Source() identity.AuthSource {
	return identity.SourceConsumerAccessToken
}

func (p *LPBSConsumerProvider) VerifyRequest(ctx context.Context, req *http.Request) (identity.Principal, error) {
	if p == nil {
		return (*consumeridentity.RequestProvider)(nil).VerifyRequest(ctx, req)
	}
	return consumeridentity.NewRequestProvider(p.cache).VerifyRequest(ctx, req)
}
