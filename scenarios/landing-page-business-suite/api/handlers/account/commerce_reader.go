package account

import (
	"context"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// CommerceSource is the narrow commerce seam consumed by AccountService
// transport. Keeping this adapter beside the handler prevents the API root
// from becoming a second account domain.
type CommerceSource interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*commerce.CreditsEnvelope, error)
	GetEntitlementsContext(context.Context, string) (*commerce.EntitlementPayload, error)
}

type commerceReader struct{ service CommerceSource }

// NewCommerceReader projects commerce DTOs into AccountService transport DTOs.
func NewCommerceReader(service CommerceSource) Reader {
	return commerceReader{service: service}
}

func (r commerceReader) GetSubscriptionContext(ctx context.Context, user string) (*shared.SubscriptionStatus, error) {
	return r.service.GetSubscriptionContext(ctx, user)
}

func (r commerceReader) GetCreditsContext(ctx context.Context, user string) (*Credits, error) {
	credits, err := r.service.GetCreditsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	return &Credits{Balance: credits.Balance, DisplayCreditsLabel: credits.DisplayCreditsLabel, DisplayCreditsMultiplier: credits.DisplayCreditsMultiplier}, nil
}

func (r commerceReader) GetEntitlementsContext(ctx context.Context, user string) (*Entitlements, error) {
	entitlements, err := r.service.GetEntitlementsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	return &Entitlements{Status: entitlements.Status, PlanTier: entitlements.PlanTier, PriceID: entitlements.PriceID, Features: entitlements.Features, BillingCycleStart: entitlements.BillingCycleStart, Credits: entitlements.Credits, Subscription: entitlements.Subscription}, nil
}

var (
	_ Reader         = commerceReader{}
	_ CommerceSource = (*commerce.Service)(nil)
)
