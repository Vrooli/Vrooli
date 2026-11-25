package main

import (
	"context"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

func (s *StripeService) handleCreditTopupContext(ctx context.Context, customerEmail string, amountCents int64, plan *commerce.PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	return commerce.NewCreditTopupService(s.planService, s.creditWallet, logx.Error).
		ApplyContext(ctx, customerEmail, amountCents, plan, stripeEventID, metadata)
}
