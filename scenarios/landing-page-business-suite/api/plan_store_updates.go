package main

import (
	"fmt"
	"math"
	"strings"

	"google.golang.org/protobuf/proto"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

// UpdatePlan applies partial updates to a plan.
func (ps *PlanStore) UpdatePlan(priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	return ps.UpdatePlanWithStripeDetails(priceID, input, nil)
}

// UpdatePlanWithStripeDetails applies partial updates to a plan and optionally syncs Stripe fields.
func (ps *PlanStore) UpdatePlanWithStripeDetails(priceID string, input UpdateBundlePriceInput, stripeDetails *StripePriceImport) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	derivedTier := ""
	if stripeDetails != nil {
		if tier, ok := derivePlanTierFromStripe(stripeDetails); ok {
			derivedTier = tier
		}
	}

	updatedPlan, err := ps.updatePlanWithStripeDetailsLocked(priceID, input, stripeDetails, derivedTier)
	if err != nil {
		return nil, err
	}

	if err := ps.savePlansLocked(); err != nil {
		return nil, err
	}

	return proto.Clone(updatedPlan).(*PlanOption), nil
}

func (ps *PlanStore) updatePlanWithStripeDetailsLocked(priceID string, input UpdateBundlePriceInput, stripeDetails *StripePriceImport, derivedTier string) (*PlanOption, error) {
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	if stripeDetails != nil {
		if err := ps.ensureStripePriceMatchesBundleLocked(stripeDetails); err != nil {
			return nil, err
		}
	}

	targetIdx, err := ps.planIndexByPriceIDLocked(priceID)
	if err != nil {
		return nil, err
	}

	currentPlan := ps.plans[targetIdx]
	updatedPlan := proto.Clone(currentPlan).(*PlanOption)

	if err := ps.applyRequestedPlanFieldsLocked(updatedPlan, currentPlan, input, stripeDetails); err != nil {
		return nil, err
	}
	if err := applyStripePriceDetails(updatedPlan, input, stripeDetails); err != nil {
		return nil, err
	}
	if err := applyDerivedPlanTier(updatedPlan, derivedTier); err != nil {
		return nil, err
	}
	applyPlanMetadata(updatedPlan, input)

	updatedPlan.BundleKey = ps.bundleKey
	if err := validateUpdatedPlan(updatedPlan); err != nil {
		return nil, err
	}

	if updatedPlan.Metadata != nil && len(updatedPlan.Metadata) == 0 {
		updatedPlan.Metadata = nil
	}

	ps.plans[targetIdx] = updatedPlan
	return updatedPlan, nil
}

func (ps *PlanStore) planIndexByPriceIDLocked(priceID string) (int, error) {
	for index, plan := range ps.plans {
		if plan.StripePriceId == priceID {
			return index, nil
		}
	}
	return -1, fmt.Errorf("price %s not found", priceID)
}

func (ps *PlanStore) applyRequestedPlanFieldsLocked(updated, current *PlanOption, input UpdateBundlePriceInput, stripeDetails *StripePriceImport) error {
	if input.StripePriceID != nil {
		priceID, err := normalizeStripePriceID(*input.StripePriceID)
		if err != nil {
			return err
		}
		if priceID != current.StripePriceId {
			if stripeDetails == nil {
				return fmt.Errorf("stripe price changes require a verified Stripe price")
			}
			if strings.TrimSpace(stripeDetails.PriceID) != priceID {
				return fmt.Errorf("stripe price verification mismatch for %s", priceID)
			}
			if _, err := ps.planIndexByPriceIDLocked(priceID); err == nil {
				return fmt.Errorf("plan with price ID %s already exists", priceID)
			}
			updated.StripePriceId = priceID
		}
	}
	if input.PlanName != nil {
		name := strings.TrimSpace(*input.PlanName)
		if name == "" {
			return fmt.Errorf("plan_name is required")
		}
		updated.PlanName = name
	}
	if input.DisplayWeight != nil {
		if *input.DisplayWeight < 0 {
			return fmt.Errorf("display_weight must be >= 0")
		}
		if *input.DisplayWeight > math.MaxInt32 {
			return fmt.Errorf("display_weight must be <= %d", math.MaxInt32)
		}
		updated.DisplayWeight = int32(*input.DisplayWeight)
	}
	if input.DisplayEnabled != nil {
		updated.DisplayEnabled = *input.DisplayEnabled
	}
	return nil
}

func applyStripePriceDetails(plan *PlanOption, input UpdateBundlePriceInput, details *StripePriceImport) error {
	if details == nil {
		return nil
	}
	interval := mapBillingInterval(details.Interval)
	if err := validateBillingInterval(interval); err != nil {
		return fmt.Errorf("invalid stripe billing interval: %w", err)
	}
	currency, err := normalizeCurrency(details.Currency)
	if err != nil {
		return fmt.Errorf("invalid stripe currency: %w", err)
	}
	if details.AmountCents < 0 {
		return fmt.Errorf("stripe amount_cents must be >= 0")
	}
	plan.AmountCents = details.AmountCents
	plan.Currency = currency
	plan.BillingInterval = interval
	if input.DisplayEnabled == nil && !details.Active {
		plan.DisplayEnabled = false
	}
	return nil
}

func applyDerivedPlanTier(plan *PlanOption, derivedTier string) error {
	if strings.TrimSpace(derivedTier) == "" {
		return nil
	}
	normalizedTier, err := normalizePlanTier(derivedTier)
	if err != nil {
		return err
	}
	if normalizedTier != plan.PlanTier {
		plan.PlanTier = normalizedTier
		plan.PlanRank = planRankForTier(normalizedTier)
		plan.Kind = planKindForTier(normalizedTier)
	}
	return nil
}

func applyPlanMetadata(plan *PlanOption, input UpdateBundlePriceInput) {
	if plan.Metadata == nil {
		plan.Metadata = map[string]*commonv1.JsonValue{}
	}
	updateMetadataString := func(key string, value *string) {
		if value == nil {
			return
		}
		trimmed := strings.TrimSpace(*value)
		if trimmed == "" {
			delete(plan.Metadata, key)
			return
		}
		plan.Metadata[key] = newStringJsonValue(trimmed)
	}
	updateMetadataString("subtitle", input.Subtitle)
	updateMetadataString("badge", input.Badge)
	updateMetadataString("cta_label", input.CtaLabel)
	if input.Highlight != nil {
		if *input.Highlight {
			plan.Metadata["highlight"] = newBoolJsonValue(true)
		} else {
			delete(plan.Metadata, "highlight")
		}
	}
	if input.Features != nil {
		features := make([]*commonv1.JsonValue, 0, len(*input.Features))
		for _, feature := range *input.Features {
			if trimmed := strings.TrimSpace(feature); trimmed != "" {
				features = append(features, newStringJsonValue(trimmed))
			}
		}
		if len(features) == 0 {
			delete(plan.Metadata, "features")
		} else {
			plan.Metadata["features"] = newListJsonValue(features)
		}
	}
}

func validateUpdatedPlan(plan *PlanOption) error {
	if plan.Kind == shared.PlanKind_PLAN_KIND_UNSPECIFIED {
		plan.Kind = planKindForTier(plan.PlanTier)
	}
	normalizedTier, err := normalizePlanTier(plan.PlanTier)
	if err != nil {
		return err
	}
	plan.PlanTier = normalizedTier
	if _, err := normalizePlanName(plan.PlanName); err != nil {
		return err
	}
	if err := validateBillingInterval(plan.BillingInterval); err != nil {
		return err
	}
	normalizedCurrency, err := normalizeCurrency(plan.Currency)
	if err != nil {
		return err
	}
	plan.Currency = normalizedCurrency
	if plan.AmountCents < 0 {
		return fmt.Errorf("amount_cents must be >= 0")
	}
	if plan.MonthlyIncludedCredits < 0 {
		return fmt.Errorf("monthly_included_credits must be >= 0")
	}
	if plan.OneTimeBonusCredits < 0 {
		return fmt.Errorf("one_time_bonus_credits must be >= 0")
	}
	if plan.PlanRank < 0 {
		return fmt.Errorf("plan_rank must be >= 0")
	}
	return validatePlanTierConstraints(plan)
}

func (ps *PlanStore) ensureStripePriceMatchesBundleLocked(stripeDetails *StripePriceImport) error {
	if stripeDetails == nil {
		return nil
	}
	if ps.bundle == nil {
		return fmt.Errorf("bundle not configured")
	}

	bundleProductID := strings.TrimSpace(ps.bundle.StripeProductId)
	if bundleProductID == "" {
		return fmt.Errorf("bundle stripe_product_id is required")
	}

	priceProductID := strings.TrimSpace(stripeDetails.ProductID)
	if priceProductID == "" {
		return fmt.Errorf("stripe price %s missing product_id", strings.TrimSpace(stripeDetails.PriceID))
	}

	if priceProductID != bundleProductID {
		return fmt.Errorf("stripe price %s belongs to product %s (expected %s)", stripeDetails.PriceID, priceProductID, bundleProductID)
	}

	return nil
}
