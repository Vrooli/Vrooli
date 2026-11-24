package main

import (
	"context"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

// handleCreditTopup translates a completed Stripe checkout into the
// commerce-owned credit ledger. Stripe remains responsible only for extracting
// provider event data; wallet mutation is owned by CreditWalletService.
func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *commerce.PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	return s.handleCreditTopupContext(context.Background(), customerEmail, amountCents, plan, stripeEventID, metadata)
}

func (s *StripeService) handleCreditTopupContext(ctx context.Context, customerEmail string, amountCents int64, plan *commerce.PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	return commerce.NewCreditTopupService(s.planService, s.creditWallet, logx.Error).
		ApplyContext(ctx, customerEmail, amountCents, plan, stripeEventID, metadata)
}
