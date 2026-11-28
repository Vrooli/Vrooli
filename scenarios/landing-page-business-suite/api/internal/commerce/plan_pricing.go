package commerce

import (
	"context"
	"fmt"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

type createdPlanPricing struct {
	amountCents    int64
	currency       string
	displayWeight  int32
	displayEnabled bool
	monthlyCredits int64
}

func (s *PlanService) fetchStripePriceForCreation(ctx context.Context, priceID string, fetcher StripePriceFetcher) *StripePriceImport {
	if fetcher == nil {
		return nil
	}

	details, err := fetcher(ctx, priceID)
	if err != nil {
		s.logEvent("stripe_price_not_verified", map[string]interface{}{
			"price_id": priceID,
			"error":    err.Error(),
		})
		return nil
	}
	return details
}

func resolveCreatedPlanPricing(
	input CreateBundlePriceInput,
	priceID string,
	billingInterval shared.BillingInterval,
	stripeDetails *StripePriceImport,
	bundle *BundleProduct,
) (createdPlanPricing, error) {
	if err := validateStripePriceForCreation(priceID, billingInterval, stripeDetails, bundle); err != nil {
		return createdPlanPricing{}, err
	}

	amountCents, err := resolveCreatedPlanAmount(input.AmountCents, stripeDetails)
	if err != nil {
		return createdPlanPricing{}, err
	}
	currency, err := resolveCreatedPlanCurrency(input.Currency, stripeDetails)
	if err != nil {
		return createdPlanPricing{}, err
	}

	displayWeight := int32(10)
	if input.DisplayWeight != nil {
		displayWeight = *input.DisplayWeight
	}
	if displayWeight < 0 {
		return createdPlanPricing{}, fmt.Errorf("display_weight must be >= 0")
	}

	displayEnabled := true
	if input.DisplayEnabled != nil {
		displayEnabled = *input.DisplayEnabled
	} else if stripeDetails != nil {
		displayEnabled = stripeDetails.Active
	}

	monthlyCredits := int64(0)
	if input.MonthlyIncludedCredits != nil {
		monthlyCredits = *input.MonthlyIncludedCredits
	}
	if monthlyCredits < 0 {
		return createdPlanPricing{}, fmt.Errorf("monthly_included_credits must be >= 0")
	}

	return createdPlanPricing{
		amountCents:    amountCents,
		currency:       currency,
		displayWeight:  displayWeight,
		displayEnabled: displayEnabled,
		monthlyCredits: monthlyCredits,
	}, nil
}

func validateStripePriceForCreation(priceID string, billingInterval shared.BillingInterval, stripeDetails *StripePriceImport, bundle *BundleProduct) error {
	if stripeDetails == nil {
		return nil
	}
	if strings.TrimSpace(stripeDetails.PriceID) != priceID {
		return fmt.Errorf("stripe price verification mismatch for %s", priceID)
	}
	if err := EnsureStripePriceMatchesBundle(bundle, stripeDetails); err != nil {
		return err
	}
	stripeInterval := MapBillingInterval(stripeDetails.Interval)
	if err := ValidateBillingInterval(stripeInterval); err != nil {
		return fmt.Errorf("stripe price has unsupported billing interval")
	}
	if billingInterval != stripeInterval {
		return fmt.Errorf("billing_interval does not match Stripe price")
	}
	return nil
}

func resolveCreatedPlanAmount(inputAmount *int64, stripeDetails *StripePriceImport) (int64, error) {
	if stripeDetails != nil {
		if inputAmount != nil && *inputAmount != stripeDetails.AmountCents {
			return 0, fmt.Errorf("amount_cents does not match Stripe price")
		}
		if stripeDetails.AmountCents < 0 {
			return 0, fmt.Errorf("amount_cents must be >= 0")
		}
		return stripeDetails.AmountCents, nil
	}
	if inputAmount == nil {
		return 0, fmt.Errorf("amount_cents is required when Stripe price cannot be verified")
	}
	if *inputAmount < 0 {
		return 0, fmt.Errorf("amount_cents must be >= 0")
	}
	return *inputAmount, nil
}

func resolveCreatedPlanCurrency(inputCurrency *string, stripeDetails *StripePriceImport) (string, error) {
	currency := "usd"
	if stripeDetails != nil {
		currency = stripeDetails.Currency
		if inputCurrency != nil && strings.TrimSpace(*inputCurrency) != "" && !strings.EqualFold(currency, *inputCurrency) {
			return "", fmt.Errorf("currency does not match Stripe price")
		}
	} else if inputCurrency != nil && strings.TrimSpace(*inputCurrency) != "" {
		currency = *inputCurrency
	}
	return NormalizeCurrency(currency)
}
