package main

import (
	"os"
	"strings"
	"testing"
)

// This inventory is intentionally source-backed: route registration is the
// security boundary, and a new sensitive registration must opt into the
// step-up wrapper in the same change.
func TestSensitiveAdminRouteInventoryUsesStepUp(t *testing.T) {
	source, err := os.ReadFile("routes.go")
	if err != nil {
		t.Fatal(err)
	}
	composition, err := os.ReadFile("admin_mfa_composition.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source) + string(composition)
	required := []string{
		"RegisterPresentationAdminRoutes(s.router, s.configStore, s.requireAdminStepUp)",
		"RegisterStripeSettingsConnectRoutes(s.router, s.paymentSettings, s.stripeService, s.paymentAnomaly, s.requireAdminStepUp)",
		"RegisterAPIKeyConnectRoutes(s.router, s.apiKeyService, s.requireAdminStepUp)",
		"RegisterConnectRoutes(s.router, billingDeps, s.requireUserAuth, s.requireAdminStepUp)",
		"RegisterConnectRoutes(s.router, s.stripeService, s.planService, s.routedDB, s.requireAdminStepUp",
		"RegisterBrandingConnectRoutes(s.router, s.configStore, s.requireAdminStepUp)",
		"RegisterSEOConnectRoutes(s.router, s.seoService, s.requireAdminStepUp)",
		"RegisterAssetsConnectRoutes(s.router, s.assetsService, s.requireAdminStepUp)",
		"RegisterConnectRoutes(s.router, docsConnectDependencies(), s.requireAdminStepUp)",
		"RegisterConnectRoutes(s.router, s.feedbackService, feedbackEmailNotifier{configStore: s.configStore, emailService: s.emailService}, s.requireAdminStepUp)",
		"RegisterProfileConnectRoutes(s.router, profileDeps, s.requireAdminProfile)",
		"/api/v1/admin/mfa/disable", "s.requireAdminStepUp(adminhttp.DisableAdminMFA",
		"/api/v1/admin/mfa/recovery-codes", "s.requireAdminStepUp(adminhttp.RegenerateAdminRecoveryCodes",
		"wrapped := s.requireAdminStepUp(http.HandlerFunc(handler.ServeHTTP))",
	}
	for _, fragment := range required {
		if !strings.Contains(text, fragment) {
			t.Errorf("sensitive admin inventory fragment missing: %s", fragment)
		}
	}
}
