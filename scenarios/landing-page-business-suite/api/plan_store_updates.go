package main

import (
	"fmt"
	"math"
	"strings"

	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/commerce"
)

// UpdatePlan applies partial updates to a plan.
func (ps *PlanStore) UpdatePlan(priceID string, input UpdateBundlePriceInput) (*PlanOption, error) {
	return ps.UpdatePlanWithStripeDetails(priceID, input, nil)
}

// UpdatePlanWithStripeDetails applies partial updates to a plan and optionally syncs Stripe fields.
func (ps *PlanStore) UpdatePlanWithStripeDetails(priceID string, input UpdateBundlePriceInput, stripeDetails *commerce.StripePriceImport) (*PlanOption, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	derivedTier := ""
	if stripeDetails != nil {
		if tier, ok := commerce.DerivePlanTierFromStripe(stripeDetails); ok {
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

func (ps *PlanStore) updatePlanWithStripeDetailsLocked(priceID string, input UpdateBundlePriceInput, stripeDetails *commerce.StripePriceImport, derivedTier string) (*PlanOption, error) {
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
	if err := commerce.ApplyStripePriceDetails(updatedPlan, input, stripeDetails); err != nil {
		return nil, err
	}
	if err := commerce.ApplyDerivedPlanTier(updatedPlan, derivedTier); err != nil {
		return nil, err
	}
	commerce.ApplyPlanMetadata(updatedPlan, input)

	updatedPlan.BundleKey = ps.bundleKey
	if err := commerce.ValidateUpdatedPlan(updatedPlan); err != nil {
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

func (ps *PlanStore) applyRequestedPlanFieldsLocked(updated, current *PlanOption, input UpdateBundlePriceInput, stripeDetails *commerce.StripePriceImport) error {
	if input.StripePriceID != nil {
		priceID, err := commerce.NormalizeStripePriceID(*input.StripePriceID)
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

func (ps *PlanStore) ensureStripePriceMatchesBundleLocked(stripeDetails *commerce.StripePriceImport) error {
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
