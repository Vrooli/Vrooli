package main

import (
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/logx"
)

// handleCreditTopup translates a completed Stripe checkout into the
// commerce-owned credit ledger. Stripe remains responsible only for extracting
// provider event data; wallet mutation is owned by CreditWalletService.
func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *commerce.PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	return commerce.NewCreditTopupService(s.planService, s.creditWallet, logx.Error).
		Apply(customerEmail, amountCents, plan, stripeEventID, metadata)
}
