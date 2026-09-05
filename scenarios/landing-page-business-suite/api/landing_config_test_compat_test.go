package main

import (
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/landing"
)

type (
	LandingConfigResponse = landing.LandingConfigResponse
	LandingVariantSummary = landing.LandingVariantSummary
	LandingSection        = landing.LandingSection
	LandingBranding       = landing.LandingBranding
	LandingConfigPayload  = landing.LandingConfigPayload
)

func NewLandingConfigServiceWithConfigStore(config *experimentation.ConfigStore, plans *commerce.PlanService, downloads *delivery.CatalogService, stripe *StripeService) *landing.LandingConfigService {
	return landing.NewLandingConfigServiceWithConfigStore(config, plans, downloads, landingIntroOfferLookup(stripe))
}

func parseFallbackLandingConfig(data []byte) (*landing.LandingConfigPayload, error) {
	return landing.ParseFallbackLandingConfig(data)
}
