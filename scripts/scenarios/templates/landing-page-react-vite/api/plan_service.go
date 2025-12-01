package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// PlanService exposes helper utilities for pricing/plan metadata.
type PlanService struct {
	db            *sql.DB
	defaultBundle string
	displayEnv    string
}

// BundleProduct captures Stripe product metadata.
type BundleProduct struct {
	ID                       int64                  `json:"id"`
	BundleKey                string                 `json:"bundle_key"`
	Name                     string                 `json:"name"`
	StripeProductID          string                 `json:"stripe_product_id"`
	CreditsPerUSD            int64                  `json:"credits_per_usd"`
	DisplayCreditsMultiplier float64                `json:"display_credits_multiplier"`
	DisplayCreditsLabel      string                 `json:"display_credits_label"`
	Environment              string                 `json:"environment"`
	Metadata                 map[string]interface{} `json:"metadata,omitempty"`
}

// PlanOption conveys a specific price entry.
type PlanOption struct {
	PlanName               string                 `json:"plan_name"`
	PlanTier               string                 `json:"plan_tier"`
	BillingInterval        string                 `json:"billing_interval"`
	AmountCents            int64                  `json:"amount_cents"`
	Currency               string                 `json:"currency"`
	IntroEnabled           bool                   `json:"intro_enabled"`
	IntroType              string                 `json:"intro_type,omitempty"`
	IntroAmountCents       *int64                 `json:"intro_amount_cents,omitempty"`
	IntroPeriods           int                    `json:"intro_periods,omitempty"`
	IntroPriceLookupKey    string                 `json:"intro_price_lookup_key,omitempty"`
	StripePriceID          string                 `json:"stripe_price_id"`
	MonthlyIncludedCredits int64                  `json:"monthly_included_credits"`
	OneTimeBonusCredits    int64                  `json:"one_time_bonus_credits"`
	PlanRank               int                    `json:"plan_rank"`
	BonusType              string                 `json:"bonus_type,omitempty"`
	Kind                   string                 `json:"kind,omitempty"`
	IsVariableAmount       bool                   `json:"is_variable_amount,omitempty"`
	BundleKey              string                 `json:"bundle_key,omitempty"`
	DisplayWeight          int                    `json:"display_weight"`
	Metadata               map[string]interface{} `json:"metadata,omitempty"`
}

// PricingOverview groups monthly/yearly options plus bundle metadata.
type PricingOverview struct {
	Bundle  BundleProduct `json:"bundle"`
	Monthly []PlanOption  `json:"monthly"`
	Yearly  []PlanOption  `json:"yearly"`
	Updated time.Time     `json:"updated_at"`
}

func NewPlanService(db *sql.DB) *PlanService {
	bundle := stringsTrimOrDefault(os.Getenv("BUNDLE_KEY"), "business_suite")
	env := stringsTrimOrDefault(os.Getenv("BUNDLE_ENVIRONMENT"), "production")
	return &PlanService{db: db, defaultBundle: bundle, displayEnv: env}
}

func stringsTrimOrDefault(value string, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

// BundleKey returns the configured bundle key used for plan lookups.
func (s *PlanService) BundleKey() string {
	return s.defaultBundle
}

// GetPricingOverview loads the product and price rows for the default bundle.
func (s *PlanService) GetPricingOverview() (*PricingOverview, error) {
	product, err := s.loadBundleProduct(s.defaultBundle)
	if err != nil {
		return nil, err
	}

	prices, err := s.loadBundlePrices(product.ID)
	if err != nil {
		return nil, err
	}

	var monthly, yearly []PlanOption
	for _, price := range prices {
		if price.BillingInterval == "month" {
			monthly = append(monthly, price)
		} else {
			yearly = append(yearly, price)
		}
	}

	sort.SliceStable(monthly, func(i, j int) bool {
		if monthly[i].DisplayWeight == monthly[j].DisplayWeight {
			return monthly[i].PlanRank < monthly[j].PlanRank
		}
		return monthly[i].DisplayWeight > monthly[j].DisplayWeight
	})
	sort.SliceStable(yearly, func(i, j int) bool {
		if yearly[i].DisplayWeight == yearly[j].DisplayWeight {
			return yearly[i].PlanRank < yearly[j].PlanRank
		}
		return yearly[i].DisplayWeight > yearly[j].DisplayWeight
	})

	return &PricingOverview{
		Bundle:  *product,
		Monthly: monthly,
		Yearly:  yearly,
		Updated: time.Now().UTC(),
	}, nil
}

// GetPlanByPriceID fetches a plan option for a Stripe price identifier.
func (s *PlanService) GetPlanByPriceID(priceID string) (*PlanOption, error) {
	if priceID == "" {
		return nil, fmt.Errorf("price id is required")
	}

	query := `
		SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type,
		       bp.intro_amount_cents, bp.intro_periods, bp.intro_price_lookup_key,
		       bp.monthly_included_credits, bp.one_time_bonus_credits, bp.plan_rank,
		       bp.bonus_type, bp.kind, bp.is_variable_amount, b.bundle_key,
		       bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.stripe_price_id = $1
	`

	row := s.db.QueryRow(query, priceID)
	var option PlanOption
	var metadataBytes []byte
	var introAmount sql.NullInt64
	err := row.Scan(
		&option.StripePriceID,
		&option.PlanName,
		&option.PlanTier,
		&option.BillingInterval,
		&option.AmountCents,
		&option.Currency,
		&option.IntroEnabled,
		&option.IntroType,
		&introAmount,
		&option.IntroPeriods,
		&option.IntroPriceLookupKey,
		&option.MonthlyIncludedCredits,
		&option.OneTimeBonusCredits,
		&option.PlanRank,
		&option.BonusType,
		&option.Kind,
		&option.IsVariableAmount,
		&option.BundleKey,
		&metadataBytes,
		&option.DisplayWeight,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("price %s not found", priceID)
	}
	if err != nil {
		return nil, err
	}

	if introAmount.Valid {
		val := introAmount.Int64
		option.IntroAmountCents = &val
	}

	if len(metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			option.Metadata = meta
		}
	}

	return &option, nil
}

func (s *PlanService) loadBundleProduct(bundleKey string) (*BundleProduct, error) {
	query := `
		SELECT id, bundle_key, bundle_name, stripe_product_id, credits_per_usd,
		       display_credits_multiplier, display_credits_label, environment, metadata
		FROM bundle_products
		WHERE bundle_key = $1 AND environment = $2
		LIMIT 1
	`
	row := s.db.QueryRow(query, bundleKey, s.displayEnv)

	var product BundleProduct
	var metadataBytes []byte
	if err := row.Scan(
		&product.ID,
		&product.BundleKey,
		&product.Name,
		&product.StripeProductID,
		&product.CreditsPerUSD,
		&product.DisplayCreditsMultiplier,
		&product.DisplayCreditsLabel,
		&product.Environment,
		&metadataBytes,
	); err != nil {
		return nil, fmt.Errorf("bundle %s not found: %w", bundleKey, err)
	}

	if len(metadataBytes) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(metadataBytes, &meta); err == nil {
			product.Metadata = meta
		}
	}

	return &product, nil
}

func (s *PlanService) loadBundlePrices(productID int64) ([]PlanOption, error) {
	query := `
		SELECT bp.stripe_price_id, bp.plan_name, bp.plan_tier, bp.billing_interval,
		       bp.amount_cents, bp.currency, bp.intro_enabled, bp.intro_type, bp.intro_amount_cents,
		       bp.intro_periods, bp.intro_price_lookup_key, bp.monthly_included_credits,
		       bp.one_time_bonus_credits, bp.plan_rank, bp.bonus_type, bp.kind, bp.is_variable_amount, b.bundle_key,
		       bp.metadata, bp.display_weight
		FROM bundle_prices bp
		JOIN bundle_products b ON bp.product_id = b.id
		WHERE bp.product_id = $1
		ORDER BY bp.display_weight DESC, bp.plan_rank ASC
	`

	rows, err := s.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []PlanOption
	for rows.Next() {
		var option PlanOption
		var metadataBytes []byte
		var introAmount sql.NullInt64
		if err := rows.Scan(
			&option.StripePriceID,
			&option.PlanName,
			&option.PlanTier,
			&option.BillingInterval,
			&option.AmountCents,
			&option.Currency,
			&option.IntroEnabled,
			&option.IntroType,
			&introAmount,
			&option.IntroPeriods,
			&option.IntroPriceLookupKey,
			&option.MonthlyIncludedCredits,
			&option.OneTimeBonusCredits,
			&option.PlanRank,
			&option.BonusType,
			&option.Kind,
			&option.IsVariableAmount,
			&option.BundleKey,
			&metadataBytes,
			&option.DisplayWeight,
		); err != nil {
			return nil, err
		}

		if introAmount.Valid {
			val := introAmount.Int64
			option.IntroAmountCents = &val
		}

		if len(metadataBytes) > 0 {
			var meta map[string]interface{}
			if err := json.Unmarshal(metadataBytes, &meta); err == nil {
				option.Metadata = meta
			}
		}

		options = append(options, option)
	}

	return options, nil
}

// GetBundleProduct returns the configured bundle product metadata.
func (s *PlanService) GetBundleProduct() (*BundleProduct, error) {
	return s.loadBundleProduct(s.defaultBundle)
}
