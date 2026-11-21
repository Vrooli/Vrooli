package identity

import (
	"context"
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
	return &LPBSConsumerProvider{
		issuer: issuer,
		cache:  consumeridentity.NewScenarioCache(issuer, "landing-page-business-suite"),
	}
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
