package bundles

import (
	"errors"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"landing-page-business-suite-api/internal/commerce"
)

// CatalogResponse is the stable legacy-HTTP representation of the editable
// bundle catalog. It stays in the transport package so API composition only
// wires dependencies and never owns response serialization policy.
type CatalogResponse struct {
	Bundles []CatalogEntryResponse `json:"bundles"`
}

type CatalogEntryResponse struct {
	Bundle BundleProductResponse `json:"bundle"`
	Prices []PlanOptionResponse  `json:"prices"`
}

type BundleProductResponse struct {
	BundleKey                string                 `json:"bundle_key"`
	Name                     string                 `json:"name"`
	StripeProductID          string                 `json:"stripe_product_id"`
	CreditsPerUSD            int64                  `json:"credits_per_usd"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	Environment              string                 `json:"environment,omitempty"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
}

type PlanOptionResponse struct {
	PlanName               string                 `json:"plan_name"`
	PlanTier               string                 `json:"plan_tier"`
	BillingInterval        string                 `json:"billing_interval"`
	AmountCents            int64                  `json:"amount_cents"`
	Currency               string                 `json:"currency"`
	IntroEnabled           bool                   `json:"intro_enabled"`
	IntroType              *string                `json:"intro_type,omitempty"`
	IntroAmountCents       *int64                 `json:"intro_amount_cents,omitempty"`
	IntroPeriods           *int32                 `json:"intro_periods,omitempty"`
	IntroPriceLookupKey    *string                `json:"intro_price_lookup_key,omitempty"`
	StripePriceID          string                 `json:"stripe_price_id"`
	MonthlyIncludedCredits int64                  `json:"monthly_included_credits"`
	OneTimeBonusCredits    int64                  `json:"one_time_bonus_credits"`
	PlanRank               *int32                 `json:"plan_rank,omitempty"`
	BonusType              *string                `json:"bonus_type,omitempty"`
	Kind                   *string                `json:"kind,omitempty"`
	IsVariableAmount       bool                   `json:"is_variable_amount"`
	DisplayEnabled         bool                   `json:"display_enabled"`
	BundleKey              string                 `json:"bundle_key,omitempty"`
	DisplayWeight          int32                  `json:"display_weight"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

func BuildCatalogResponse(entries []commerce.BundleCatalogEntry) (CatalogResponse, error) {
	bundles := make([]CatalogEntryResponse, 0, len(entries))
	for _, entry := range entries {
		if entry.Bundle == nil {
			return CatalogResponse{}, errors.New("bundle catalog entry missing bundle")
		}
		prices := make([]PlanOptionResponse, 0, len(entry.Prices))
		for _, price := range entry.Prices {
			if price != nil {
				prices = append(prices, PlanOptionResponseFromProto(price))
			}
		}
		bundles = append(bundles, CatalogEntryResponse{Bundle: BundleProductResponseFromProto(entry.Bundle), Prices: prices})
	}
	return CatalogResponse{Bundles: bundles}, nil
}

func BundleProductResponseFromProto(bundle *commerce.BundleProduct) BundleProductResponse {
	if bundle == nil {
		return BundleProductResponse{}
	}
	response := BundleProductResponse{BundleKey: bundle.BundleKey, Name: bundle.Name, StripeProductID: bundle.StripeProductId, CreditsPerUSD: bundle.CreditsPerUsd, DisplayCreditsMultiplier: bundle.DisplayCreditsMultiplier, DisplayCreditsLabel: bundle.DisplayCreditsLabel, Environment: bundle.Environment}
	if len(bundle.Metadata) > 0 {
		response.Metadata = commerce.ConvertProtoMetadataToMap(bundle.Metadata)
	}
	return response
}

func PlanOptionResponseFromProto(plan *commerce.PlanOption) PlanOptionResponse {
	if plan == nil {
		return PlanOptionResponse{}
	}
	interval := commerce.BillingIntervalLabel(plan.BillingInterval)
	if interval == "unspecified" {
		interval = "one_time"
	}
	var introType, bonusType, kind *string
	if value := commerce.IntroPricingTypeString(plan.IntroType); value != "" {
		introType = &value
	}
	if value := strings.TrimSpace(plan.BonusType); value != "" {
		bonusType = &value
	}
	if plan.Kind != shared.PlanKind_PLAN_KIND_UNSPECIFIED {
		value := commerce.PlanKindString(plan.Kind)
		kind = &value
	}
	var introPeriods, planRank *int32
	if plan.IntroPeriods > 0 {
		value := plan.IntroPeriods
		introPeriods = &value
	}
	if plan.PlanRank > 0 {
		value := plan.PlanRank
		planRank = &value
	}
	response := PlanOptionResponse{PlanName: plan.PlanName, PlanTier: plan.PlanTier, BillingInterval: interval, AmountCents: plan.AmountCents, Currency: plan.Currency, IntroEnabled: plan.IntroEnabled, IntroType: introType, IntroAmountCents: plan.IntroAmountCents, IntroPeriods: introPeriods, IntroPriceLookupKey: optionalString(plan.IntroPriceLookupKey), StripePriceID: plan.StripePriceId, MonthlyIncludedCredits: plan.MonthlyIncludedCredits, OneTimeBonusCredits: plan.OneTimeBonusCredits, PlanRank: planRank, BonusType: bonusType, Kind: kind, IsVariableAmount: plan.IsVariableAmount, DisplayEnabled: plan.DisplayEnabled, BundleKey: plan.BundleKey, DisplayWeight: plan.DisplayWeight}
	if len(plan.Metadata) > 0 {
		response.Metadata = commerce.ConvertProtoMetadataToMap(plan.Metadata)
	}
	return response
}

func optionalString(value string) *string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return &trimmed
	}
	return nil
}
