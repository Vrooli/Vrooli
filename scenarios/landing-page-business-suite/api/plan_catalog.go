package main

import (
	"strings"

	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/envx"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

type (
	PlanStore              = commerce.PlanStore
	PlanStoreReader        = commerce.PlanStoreReader
	PlanStoreWriter        = commerce.PlanStoreWriter
	PlanStorer             = commerce.PlanStorer
	PlanStoreOptions       = commerce.PlanStoreOptions
	PlanService            = commerce.PlanService
	PlanServiceOptions     = commerce.PlanServiceOptions
	BundleProduct          = commerce.BundleProduct
	PlanOption             = commerce.PlanOption
	PricingOverview        = commerce.PricingOverview
	StripePriceFetcher     = commerce.StripePriceFetcher
	ImportPlanSelection    = commerce.ImportPlanSelection
	StripeImportMode       = commerce.StripeImportMode
	StripeImportResult     = commerce.StripeImportResult
	BundleCatalogEntry     = commerce.BundleCatalogEntry
	UpdateBundlePriceInput = commerce.UpdateBundlePriceInput
	CreateBundlePriceInput = commerce.CreateBundlePriceInput
)

const (
	StripeImportModeMerge   = commerce.StripeImportModeMerge
	StripeImportModeReplace = commerce.StripeImportModeReplace
)

var (
	errStripeImportNoSelections                 = commerce.ErrStripeImportNoSelections
	errStripeImportNoValidSelections            = commerce.ErrStripeImportNoValidSelections
	errStripeImportMissingFetcher               = commerce.ErrStripeImportMissingFetcher
	errStripeImportBundleMissing                = commerce.ErrStripeImportBundleMissing
	errStripeImportBundleProductMissing         = commerce.ErrStripeImportBundleProductMissing
	errStripeImportInvalidMode                  = commerce.ErrStripeImportInvalidMode
	errStripeImportProductSwitchRequiresReplace = commerce.ErrStripeImportProductSwitchRequiresReplace
)

func planEnv(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func NewPlanStore(plansPath string) *PlanStore {
	return commerce.NewPlanStoreWithOptions(PlanStoreOptions{
		PlansPath:  plansPath,
		BundleKey:  planEnv(envx.Get("BUNDLE_KEY"), "business_suite"),
		DisplayEnv: planEnv(envx.Get("BUNDLE_ENVIRONMENT"), "production"),
		Log:        logStructured,
	})
}

func NewPlanStoreWithOptions(opts PlanStoreOptions) *PlanStore {
	if opts.Log == nil {
		opts.Log = logStructured
	}
	return commerce.NewPlanStoreWithOptions(opts)
}

func NewPlanService(_ any) *PlanService {
	bundle := planEnv(envx.Get("BUNDLE_KEY"), "business_suite")
	env := planEnv(envx.Get("BUNDLE_ENVIRONMENT"), "production")
	plansPath := commerce.ResolvePlansPath()
	store := commerce.NewPlanStoreWithOptions(PlanStoreOptions{PlansPath: plansPath, BundleKey: bundle, DisplayEnv: env, Log: logStructured})
	if err := store.LoadAll(); err != nil {
		logStructuredError("plan_store_load_failed", map[string]interface{}{"error": err.Error(), "path": plansPath})
	}
	return commerce.NewPlanServiceWithOptions(PlanServiceOptions{PlanStore: store, DefaultBundle: bundle, DisplayEnv: env})
}

func NewPlanServiceWithPlanStore(store *PlanStore) *PlanService {
	return NewPlanServiceWithOptions(PlanServiceOptions{PlanStore: store})
}

func NewPlanServiceWithOptions(opts PlanServiceOptions) *PlanService {
	if opts.Log == nil {
		opts.Log = logStructured
	}
	return commerce.NewPlanServiceWithOptions(opts)
}

// convertProtoMetadataToMap keeps the administrative transport focused on
// request shaping while commerce owns the catalog serialization contract.
func convertProtoMetadataToMap(metadata map[string]*commonv1.JsonValue) map[string]interface{} {
	return commerce.ConvertProtoMetadataToMap(metadata)
}
