package main

import (
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/landing"
)

func newLandingConfigService(configStore *experimentation.ConfigStore, planService *commerce.PlanService, downloadService *delivery.CatalogService) *landing.LandingConfigService {
	return landing.NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService)
}
