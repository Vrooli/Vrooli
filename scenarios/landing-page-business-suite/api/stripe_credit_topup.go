package main

import (
	"errors"

	"landing-page-business-suite-api/internal/commerce"
)

// handleCreditTopup translates a completed Stripe checkout into the
// commerce-owned credit ledger. Stripe remains responsible only for extracting
// provider event data; wallet mutation is owned by CreditWalletService.
func (s *StripeService) handleCreditTopup(customerEmail string, amountCents int64, plan *commerce.PlanOption, stripeEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}
	if amountCents == 0 {
		amountCents = plan.AmountCents
	}

	bundle, err := s.planService.GetBundleProduct()
	if err != nil {
		return err
	}
	if bundle == nil {
		logStructuredError("bundle_product_not_configured", map[string]interface{}{
			"customer_email":  customerEmail,
			"amount_cents":    amountCents,
			"stripe_event_id": stripeEventID,
		})
		return errors.New("bundle product not configured - cannot process credit topup")
	}
	if amountCents == 0 {
		return errors.New("amount is zero - cannot process credit topup")
	}

	credits := (bundle.CreditsPerUsd * amountCents) / 100
	if credits <= 0 {
		return nil
	}
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["price_id"] = plan.StripePriceId
	metadata["session_type"] = sessionTypeCreditsTopup
	return s.creditWallet.AddCredits(customerEmail, credits, "credit_topup", stripeEventID, metadata)
}
