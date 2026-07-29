package main

import (
	"context"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	accounthttp "landing-page-business-suite-api/handlers/account"
	"landing-page-business-suite-api/internal/commerce"
)

// accountTransportReader is the composition adapter between generated Connect
// transport and commerce. Handlers receive only transport DTOs, which avoids
// a direct sibling-domain import from handlers/account into internal/commerce.
type accountTransportSource interface {
	GetSubscriptionContext(context.Context, string) (*shared.SubscriptionStatus, error)
	GetCreditsContext(context.Context, string) (*commerce.CreditsEnvelope, error)
	GetEntitlementsContext(context.Context, string) (*commerce.EntitlementPayload, error)
}

type accountTransportReader struct{ service accountTransportSource }

func (a accountTransportReader) GetSubscriptionContext(ctx context.Context, user string) (*shared.SubscriptionStatus, error) {
	return a.service.GetSubscriptionContext(ctx, user)
}

func (a accountTransportReader) GetCreditsContext(ctx context.Context, user string) (*accounthttp.Credits, error) {
	credits, err := a.service.GetCreditsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	return &accounthttp.Credits{Balance: credits.Balance, DisplayCreditsLabel: credits.DisplayCreditsLabel, DisplayCreditsMultiplier: credits.DisplayCreditsMultiplier}, nil
}

func (a accountTransportReader) GetEntitlementsContext(ctx context.Context, user string) (*accounthttp.Entitlements, error) {
	entitlements, err := a.service.GetEntitlementsContext(ctx, user)
	if err != nil {
		return nil, err
	}
	return &accounthttp.Entitlements{Status: entitlements.Status, PlanTier: entitlements.PlanTier, PriceID: entitlements.PriceID, Features: entitlements.Features, BillingCycleStart: entitlements.BillingCycleStart, Credits: entitlements.Credits, Subscription: entitlements.Subscription}, nil
}

var _ accounthttp.Reader = accountTransportReader{}
var _ accountTransportSource = (*AccountService)(nil)
