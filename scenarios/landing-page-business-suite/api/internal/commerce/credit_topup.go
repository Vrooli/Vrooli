package commerce

import (
	"errors"
	"fmt"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// BundleProductReader is the narrow catalog dependency needed to turn a
// provider payment into wallet credits.
//
// seam: BundleProductReader keeps provider-event processing independent of the
// concrete plan catalog while preserving its configured credit conversion rule.
type BundleProductReader interface {
	GetBundleProduct() (*BundleProduct, error)
}

// CreditTopupService owns the provider-neutral portion of a credit top-up.
// Stripe webhook parsing stays at the integration edge; this service validates
// the normalized event and records the immutable ledger mutation.
type CreditTopupService struct {
	plans  BundleProductReader
	wallet CreditWallet
	logf   func(string, map[string]interface{})
}

func NewCreditTopupService(plans BundleProductReader, wallet CreditWallet, logf func(string, map[string]interface{})) *CreditTopupService {
	return &CreditTopupService{plans: plans, wallet: wallet, logf: logf}
}

// Apply records credits for a successfully completed provider checkout. The
// provider event ID is retained as the wallet's replay key.
func (s *CreditTopupService) Apply(customerEmail string, amountCents int64, plan *PlanOption, providerEventID string, metadata map[string]interface{}) error {
	if customerEmail == "" {
		return errors.New("customer email required for credit top-up")
	}
	if plan == nil {
		return errors.New("plan required for credit top-up")
	}
	if !strings.EqualFold(plan.PlanTier, "credits") && plan.Kind != shared.PlanKind_PLAN_KIND_CREDITS_TOPUP {
		return errors.New("plan is not a credit top-up")
	}
	if plan.IsVariableAmount || !IsApprovedCreditTopupAmount(plan.AmountCents) {
		return fmt.Errorf("credit top-up price %d cents is not approved", plan.AmountCents)
	}
	if amountCents == 0 {
		amountCents = plan.AmountCents
	}
	if amountCents != plan.AmountCents {
		return fmt.Errorf("credit top-up amount %d cents does not match catalog price %d cents", amountCents, plan.AmountCents)
	}
	if s.plans == nil {
		return errors.New("plan catalog unavailable for credit top-up")
	}
	bundle, err := s.plans.GetBundleProduct()
	if err != nil {
		return err
	}
	if bundle == nil {
		if s.logf != nil {
			s.logf("bundle_product_not_configured", map[string]interface{}{
				"customer_email":  customerEmail,
				"amount_cents":    amountCents,
				"stripe_event_id": providerEventID,
			})
		}
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
	metadata["session_type"] = "credits_topup"
	if s.wallet == nil {
		return errors.New("credit wallet unavailable for credit top-up")
	}
	if businessAccountID := metadataString(metadata, "business_account_id"); businessAccountID != "" {
		scoped, ok := s.wallet.(BusinessAccountCreditWallet)
		if !ok {
			return errors.New("account-scoped credit wallet unavailable")
		}
		return scoped.AddCreditsForBusinessAccount(businessAccountID, customerEmail, credits, "credit_topup", "stripe", providerEventID, metadata)
	}
	if sourceWallet, ok := s.wallet.(SourceCreditWallet); ok {
		return sourceWallet.AddCreditsFromSource(customerEmail, credits, "credit_topup", "stripe", providerEventID, metadata)
	}
	return s.wallet.AddCredits(customerEmail, credits, "credit_topup", providerEventID, metadata)
}
