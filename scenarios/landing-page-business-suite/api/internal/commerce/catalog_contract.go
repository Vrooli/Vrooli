package commerce

import (
	"context"
	"errors"
	"fmt"
	"strings"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
)

var (
	ErrStripeImportNoSelections                 = errors.New("no selections provided")
	ErrStripeImportNoValidSelections            = errors.New("no valid selections provided")
	ErrStripeImportMissingFetcher               = errors.New("stripe price fetcher required")
	ErrStripeImportBundleMissing                = errors.New("bundle not configured")
	ErrStripeImportBundleProductMissing         = errors.New("bundle_product_id is required")
	ErrStripeImportInvalidMode                  = errors.New("import mode must be merge or replace")
	ErrStripeImportProductSwitchRequiresReplace = errors.New("bundle product change requires replace mode")
)

// StripePriceFetcher resolves Stripe price details for catalog mutations.
type StripePriceFetcher func(ctx context.Context, priceID string) (*StripePriceImport, error)

// ImportPlanSelection is one requested Stripe catalog import action.
type ImportPlanSelection struct {
	PriceID string `json:"price_id"`
	Action  string `json:"action"`
}

// NormalizeStripeImportSelections validates and canonicalizes import actions.
func NormalizeStripeImportSelections(selections []ImportPlanSelection) ([]ImportPlanSelection, []string, error) {
	if len(selections) == 0 {
		return nil, nil, ErrStripeImportNoSelections
	}
	normalized := make([]ImportPlanSelection, 0, len(selections))
	seen := make(map[string]struct{}, len(selections))
	var validationErrors []string
	for _, selection := range selections {
		priceID := strings.TrimSpace(selection.PriceID)
		if priceID == "" {
			validationErrors = append(validationErrors, "empty price ID in selection")
			continue
		}
		if _, err := NormalizeStripePriceID(priceID); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("invalid price id %s: %s", priceID, err))
			continue
		}
		action := strings.ToLower(strings.TrimSpace(selection.Action))
		if action != "import" && action != "overwrite" && action != "skip" {
			validationErrors = append(validationErrors, "unknown action: "+selection.Action)
			continue
		}
		if _, exists := seen[priceID]; exists {
			validationErrors = append(validationErrors, "duplicate selection for price "+priceID)
			continue
		}
		seen[priceID] = struct{}{}
		normalized = append(normalized, ImportPlanSelection{PriceID: priceID, Action: action})
	}
	if len(normalized) == 0 {
		return nil, validationErrors, ErrStripeImportNoValidSelections
	}
	return normalized, validationErrors, nil
}

// StripeImportMode controls whether an import merges with or replaces a catalog.
type StripeImportMode string

const (
	StripeImportModeMerge   StripeImportMode = "merge"
	StripeImportModeReplace StripeImportMode = "replace"
)

// StripeImportResult reports the outcome of a Stripe catalog import.
type StripeImportResult struct {
	Imported    int      `json:"imported"`
	Overwritten int      `json:"overwritten"`
	Skipped     int      `json:"skipped"`
	Errors      []string `json:"errors,omitempty"`
}

// BundleCatalogEntry groups a bundle with every configured price.
type BundleCatalogEntry struct {
	Bundle *shared.Bundle       `json:"bundle"`
	Prices []*shared.PlanOption `json:"prices"`
}

// UpdateBundlePriceInput contains display metadata editable by operators.
type UpdateBundlePriceInput struct {
	StripePriceID  *string
	PlanName       *string
	DisplayWeight  *int
	DisplayEnabled *bool
	Subtitle       *string
	Badge          *string
	CtaLabel       *string
	Highlight      *bool
	Features       *[]string
}

// CreateBundlePriceInput contains a complete operator-provided price definition.
type CreateBundlePriceInput struct {
	StripePriceID          string
	PlanName               string
	PlanTier               string
	BillingInterval        string
	AmountCents            *int64
	Currency               *string
	DisplayWeight          *int32
	DisplayEnabled         *bool
	MonthlyIncludedCredits *int64
	Subtitle               *string
	Badge                  *string
	CtaLabel               *string
	Highlight              *bool
	Features               []string
}
