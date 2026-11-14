package landing

import (
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/commerce"
)

// sanitizePresentationPricing projects commerce facts, never its arbitrary
// metadata or hidden plans, into the public presentation. Marketing descriptions
// belong to the versioned page. The owner remains unchanged for admin/checkout.
func sanitizePresentationPricing(pricing *commerce.PricingOverview) *commerce.PricingOverview {
	if pricing == nil {
		return nil
	}
	result := &commerce.PricingOverview{
		Monthly:      publicPresentationPlans(pricing.Monthly),
		Yearly:       publicPresentationPlans(pricing.Yearly),
		CreditTopups: publicPresentationPlans(pricing.CreditTopups),
	}
	if bundle := pricing.Bundle; bundle != nil {
		result.Bundle = &commerce.BundleProduct{
			BundleKey: bundle.BundleKey, Name: bundle.Name, StripeProductId: bundle.StripeProductId,
			CreditsPerUsd: bundle.CreditsPerUsd, DisplayCreditsMultiplier: bundle.DisplayCreditsMultiplier,
			DisplayCreditsLabel: bundle.DisplayCreditsLabel, Environment: bundle.Environment,
		}
	}
	if pricing.UpdatedAt != nil {
		result.UpdatedAt = &timestamppb.Timestamp{Seconds: pricing.UpdatedAt.Seconds, Nanos: pricing.UpdatedAt.Nanos}
	}
	return result
}

func publicPresentationPlans(plans []*commerce.PlanOption) []*commerce.PlanOption {
	result := make([]*commerce.PlanOption, 0, len(plans))
	for _, plan := range plans {
		if plan == nil || !plan.DisplayEnabled {
			continue
		}
		copyPlan := &commerce.PlanOption{
			PlanName: plan.PlanName, PlanTier: plan.PlanTier, BillingInterval: plan.BillingInterval,
			AmountCents: plan.AmountCents, Currency: plan.Currency, IntroEnabled: plan.IntroEnabled,
			IntroType: plan.IntroType, IntroPeriods: plan.IntroPeriods, IntroPriceLookupKey: plan.IntroPriceLookupKey,
			StripePriceId: plan.StripePriceId, MonthlyIncludedCredits: plan.MonthlyIncludedCredits,
			OneTimeBonusCredits: plan.OneTimeBonusCredits, PlanRank: plan.PlanRank, BonusType: plan.BonusType,
			Kind: plan.Kind, IsVariableAmount: plan.IsVariableAmount, DisplayEnabled: true,
			BundleKey: plan.BundleKey, DisplayWeight: plan.DisplayWeight,
		}
		if plan.IntroAmountCents != nil {
			amount := *plan.IntroAmountCents
			copyPlan.IntroAmountCents = &amount
		}
		result = append(result, copyPlan)
	}
	return result
}
