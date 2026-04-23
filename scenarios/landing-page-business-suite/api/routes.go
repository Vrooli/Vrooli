package main

import (
	"net/http"

	"github.com/vrooli/api-core/health"
)

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	registerHealthRoutes(s)
	registerLandingRoutes(s)
	registerAuthRoutes(s)
	registerAccountRoutes(s)
	registerBillingRoutes(s)
	registerAdminCoreRoutes(s)
	registerRemoteProfileRoutes(s)
	registerCommerceAdminRoutes(s)
	registerVariantRoutes(s)
	registerContentRoutes(s)
	registerMetricsRoutes(s)
	registerFeedbackRoutes(s)
	registerWaitlistRoutes(s)
	registerCreditsRoutes(s)
	registerAIRoutes(s)
	registerDocsRoutes(s)
	registerAdminUserRoutes(s)
	registerUpdateRoutes(s)
	registerDeployReadinessRoute(s)
}

func registerDeployReadinessRoute(s *Server) {
	s.router.HandleFunc(
		"/api/v1/deploy-readiness",
		s.requireAdminOrService(handleDeployReadiness(s.downloadHosting, s.downloadService, s.remoteProfileService, s.planService)),
	).Methods("POST")
}

func registerHealthRoutes(s *Server) {
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.db), health.Critical).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func registerLandingRoutes(s *Server) {
	// Landing config + plans
	s.router.HandleFunc("/api/v1/landing-config", handleLandingConfig(s.landingConfigService)).Methods("GET")
	s.router.HandleFunc("/api/v1/plans", handlePlans(s.planService)).Methods("GET")
	s.router.HandleFunc("/api/v1/variant-space", handleVariantSpaceRoute(s.variantSpace)).Methods("GET")

	// Customization command for landing updates
	s.router.HandleFunc("/api/v1/customize", s.handleCustomize).Methods("POST")
}

func registerAuthRoutes(s *Server) {
	// User Authentication endpoints (magic link + JWT)
	// Public auth endpoints (no auth required)
	s.router.HandleFunc("/api/v1/auth/magic-link", handleMagicLinkRequest(s.userAuthService, s.magicLinkLimiter)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/verify", handleMagicLinkVerify(s.userAuthService)).Methods("GET")
	s.router.HandleFunc("/api/v1/auth/refresh", handleTokenRefresh(s.userAuthService)).Methods("POST")
	// Protected auth endpoints (require user auth)
	s.router.HandleFunc("/api/v1/auth/logout", s.requireUserAuth(handleUserLogout(s.userAuthService))).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/me", s.requireUserAuth(handleAuthMe(s.userAuthService))).Methods("GET")
}

func registerAccountRoutes(s *Server) {
	// Account endpoints (all require user auth)
	s.router.HandleFunc("/api/v1/me/subscription", s.requireUserAuth(handleMeSubscription(s.accountService))).Methods("GET")
	s.router.HandleFunc("/api/v1/me/credits", s.requireUserAuth(handleMeCredits(s.accountService))).Methods("GET")
	s.router.HandleFunc("/api/v1/entitlements", s.requireUserAuth(handleEntitlements(s.accountService))).Methods("GET")
	s.router.HandleFunc("/api/v1/downloads", s.requireUserAuth(handleDownloads(s.downloadAuthorizer, s.downloadHosting, s.planService))).Methods("GET")
}

func registerBillingRoutes(s *Server) {
	// Billing APIs (checkout sessions are public, portal requires auth)
	s.router.HandleFunc("/api/v1/billing/create-checkout-session", handleBillingCreateCheckoutSession(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/billing/create-credits-checkout-session", handleBillingCreateCreditsSession(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/billing/portal-url", s.requireUserAuth(handleBillingPortalURL(s.stripeService))).Methods("GET")

	// Stripe Payment endpoints (OT-P0-025 through OT-P0-030)
	s.router.HandleFunc("/api/v1/checkout/create", handleCheckoutCreate(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/webhooks/stripe", handleStripeWebhook(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/subscription/verify", handleSubscriptionVerify(s.stripeService)).Methods("GET")
	s.router.HandleFunc("/api/v1/subscription/cancel", s.requireAdmin(handleSubscriptionCancel(s.stripeService))).Methods("POST")
}

func registerAdminCoreRoutes(s *Server) {
	// Admin authentication endpoints (OT-P0-008)
	s.router.HandleFunc("/api/v1/admin/login", s.handleAdminLogin).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/logout", s.requireAdmin(s.handleAdminLogout)).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/session", s.handleAdminSession).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfile)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfileUpdate)).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/settings/stripe", s.requireAdmin(handleGetStripeSettings(s.paymentSettings, s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/settings/stripe", s.requireAdmin(handleUpdateStripeSettings(s.paymentSettings, s.stripeService, s.paymentAnomaly))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/settings/stripe/reveal", s.requireAdmin(handleRevealStripeSecret(s.stripeService, s.paymentSettings))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/stripe/verify-price", s.requireAdmin(handleAdminVerifyStripePrice(s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/reset-demo-data", s.requireAdmin(s.handleAdminResetDemoData)).Methods("POST")
}

func registerRemoteProfileRoutes(s *Server) {
	// List and test/proxy use requireAdminOrService so inter-scenario clients (e.g. s2d) can call them with a service bearer token.
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdminOrService(handleAdminListRemoteProfiles(s.remoteProfileService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdmin(handleAdminCreateRemoteProfile(s.remoteProfileService, s.sessionAdminEmail))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdmin(handleAdminUpdateRemoteProfile(s.remoteProfileService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdmin(handleAdminDeleteRemoteProfile(s.remoteProfileService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/login", s.requireAdmin(handleAdminRemoteProfileLogin(s.remoteProfileService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/logout", s.requireAdmin(handleAdminRemoteProfileLogout(s.remoteProfileService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/test", s.requireAdminOrService(handleAdminRemoteProfileTest(s.remoteProfileService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/session-links", s.requireAdmin(handleAdminRemoteProfileSessionLinks(s.remoteProfileService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/remote-revoke", s.requireAdmin(handleAdminRemoteProfileRemoteRevoke(s.remoteProfileService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/proxy", s.requireAdminOrService(handleAdminRemoteProfileProxy(s.remoteProfileService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions", s.requireAdmin(handleAdminListIncomingRemoteProfileSessions(s.db))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions/{session_id}", s.requireAdmin(handleAdminRevokeIncomingRemoteProfileSession(s.db))).Methods("DELETE")
}

func registerCommerceAdminRoutes(s *Server) {
	// Download hosting + assets
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(handleAdminListDownloadApps(s.downloadService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(handleAdminCreateDownloadApp(s.downloadService, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(handleAdminSaveDownloadApp(s.downloadService, s.planService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(handleAdminDeleteDownloadApp(s.downloadService, s.planService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdmin(handleAdminGetDownloadStorage(s.downloadHosting, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdmin(handleAdminUpdateDownloadStorage(s.downloadHosting, s.planService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-storage/test", s.requireAdmin(handleAdminTestDownloadStorage(s.downloadHosting, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts", s.requireAdmin(handleAdminListDownloadArtifacts(s.downloadHosting, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/by-app", s.requireAdmin(handleAdminListDownloadArtifactsByApp(s.downloadHosting, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/presign-upload", s.requireAdminOrService(handleAdminPresignUploadDownloadArtifact(s.downloadHosting, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/commit", s.requireAdminOrService(handleAdminCommitDownloadArtifact(s.downloadHosting, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/{artifact_id}/presign-get", s.requireAdmin(handleAdminPresignGetDownloadArtifact(s.downloadHosting, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-assets/apply", s.requireAdminOrService(handleAdminApplyDownloadArtifact(s.downloadService, s.downloadHosting, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-assets/set-current", s.requireAdmin(handleAdminSetArtifactAsCurrent(s.downloadService, s.downloadHosting, s.planService))).Methods("POST")

	// Bundles + pricing
	s.router.HandleFunc("/api/v1/admin/bundles", s.requireAdmin(handleAdminBundleCatalog(s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdmin(handleAdminUpdateBundlePrice(s.planService, s.stripeService))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices", s.requireAdmin(handleAdminCreateBundlePrice(s.planService, s.stripeService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdmin(handleAdminDeleteBundlePrice(s.planService))).Methods("DELETE")

	// Stripe import tools
	s.router.HandleFunc("/api/v1/admin/stripe/import-preview", s.requireAdmin(handleAdminStripeImportPreview(s.stripeService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/stripe/import", s.requireAdmin(handleAdminStripeImport(s.stripeService, s.planService))).Methods("POST")

	// Coupon management endpoints
	s.router.HandleFunc("/api/v1/admin/coupons", s.requireAdmin(handleAdminListCoupons(s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/coupons", s.requireAdmin(handleAdminCreateCoupon(s.stripeService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/coupons/usage", s.requireAdmin(handleAdminCouponUsage(s.stripeService, s.db))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/coupons/{coupon_id}", s.requireAdmin(handleAdminGetCoupon(s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/coupons/{coupon_id}", s.requireAdmin(handleAdminUpdateCoupon(s.stripeService))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/admin/coupons/{coupon_id}", s.requireAdmin(handleAdminDeleteCoupon(s.stripeService))).Methods("DELETE")
	// Coupon-plan mapping endpoints
	s.router.HandleFunc("/api/v1/admin/coupon-mappings", s.requireAdmin(handleAdminGetCouponMappings(s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/plans/{price_id}/coupon", s.requireAdmin(handleAdminSetCouponForPlan(s.planService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/plans/{price_id}/coupon", s.requireAdmin(handleAdminRemoveCouponFromPlan(s.planService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/stripe/coupons-preview", s.requireAdmin(handleAdminStripeCouponsPreview(s.stripeService))).Methods("GET")
}

func registerVariantRoutes(s *Server) {
	// A/B Testing variant endpoints (OT-P0-014 through OT-P0-018)
	// Public endpoints (no auth required for landing page display)
	s.router.HandleFunc("/api/v1/variants/select", handleVariantSelect(s.configStore)).Methods("GET")
	s.router.HandleFunc("/api/v1/public/variants/{slug}", handlePublicVariantBySlug(s.configStore)).Methods("GET")
	s.router.HandleFunc("/api/v1/public/variants/{variant_slug}/sections", handleGetPublicSectionsFromConfigStore(s.configStore)).Methods("GET")

	// Admin endpoints (require auth)
	s.router.HandleFunc("/api/v1/variants", s.requireAdmin(handleVariantsList(s.configStore))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantBySlug(s.configStore))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantUpdate(s.configStore))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantDelete(s.configStore))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/variants/sync", s.requireAdmin(handleVariantSnapshotSync(s.configStore))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/export", s.requireAdmin(handleVariantExport(s.configStore))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/import", s.requireAdmin(handleVariantImport(s.configStore))).Methods("PUT")

	// Content Customization endpoints - sections are part of variant snapshots
	s.router.HandleFunc("/api/v1/variants/{variant_slug}/sections", s.requireAdmin(handleGetSectionsFromConfigStore(s.configStore))).Methods("GET")
}

func registerContentRoutes(s *Server) {
	// Branding endpoints (admin-only for site-wide branding)
	s.router.HandleFunc("/api/v1/admin/branding", s.requireAdmin(handleGetBranding(s.configStore))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/branding", s.requireAdmin(handleUpdateBranding(s.configStore))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/branding/clear-field", s.requireAdmin(handleClearBrandingField(s.configStore))).Methods("POST")
	s.router.HandleFunc("/api/v1/branding", handleGetPublicBranding(s.configStore)).Methods("GET")

	// Asset upload endpoints (admin-only for file uploads)
	s.router.HandleFunc("/api/v1/admin/assets", s.requireAdmin(handleAssetsList(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/upload", s.requireAdmin(handleAssetUpload(s.assetsService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetGet(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetDelete(s.assetsService))).Methods("DELETE")

	// Serve uploaded files publicly
	s.router.PathPrefix("/api/v1/uploads/").Handler(http.StripPrefix("/api/v1/uploads/", http.FileServer(http.Dir(s.assetsService.GetUploadDir()))))

	// SEO endpoints
	s.router.HandleFunc("/api/v1/seo/{slug}", handleGetVariantSEO(s.seoService)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/seo", s.requireAdmin(handleUpdateVariantSEOConfigStore(s.configStore))).Methods("PUT")

	// Sitemap and robots.txt
	s.router.HandleFunc("/sitemap.xml", handleSitemapXML(s.seoService)).Methods("GET")
	s.router.HandleFunc("/robots.txt", handleRobotsTXT(s.seoService)).Methods("GET")
}

func registerMetricsRoutes(s *Server) {
	// Metrics & Analytics endpoints (OT-P0-019 through OT-P0-024)
	s.router.HandleFunc("/api/v1/metrics/track", handleMetricsTrack(s.metricsService)).Methods("POST")
	s.router.HandleFunc("/api/v1/metrics/summary", s.requireAdmin(handleMetricsSummary(s.metricsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/metrics/variants", s.requireAdmin(handleMetricsVariantStats(s.metricsService))).Methods("GET")
}

func registerFeedbackRoutes(s *Server) {
	// Feedback endpoints
	s.router.HandleFunc("/api/feedback", handleFeedbackCreateWithConfigStore(s.feedbackService, s.configStore, s.emailService)).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/feedback", s.requireAdmin(handleFeedbackList(s.feedbackService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/feedback/bulk-delete", s.requireAdmin(handleFeedbackDeleteBulk(s.feedbackService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/feedback/{id}", s.requireAdmin(handleFeedbackGet(s.feedbackService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/feedback/{id}", s.requireAdmin(handleFeedbackDelete(s.feedbackService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/feedback/{id}/status", s.requireAdmin(handleFeedbackUpdateStatus(s.feedbackService))).Methods("PATCH")
}

func registerWaitlistRoutes(s *Server) {
	// Waitlist endpoints (for coming soon mode)
	s.router.HandleFunc("/api/v1/waitlist", handleWaitlistCreate(s.waitlistService)).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/waitlist", s.requireAdmin(handleWaitlistList(s.waitlistService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/waitlist/{id}", s.requireAdmin(handleWaitlistDelete(s.waitlistService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/waitlist/export", s.requireAdmin(handleWaitlistExport(s.waitlistService))).Methods("GET")
}

func registerCreditsRoutes(s *Server) {
	// Credit System: API Keys Management (Admin)
	s.router.HandleFunc("/api/v1/admin/api-keys", s.requireAdmin(handleListAPIKeys(s.apiKeyService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/api-keys", s.requireAdmin(handleCreateAPIKey(s.apiKeyService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/api-keys", s.requireAdmin(handleDeleteAPIKey(s.apiKeyService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/api-keys/test", s.requireAdmin(handleTestAPIKey(s.apiKeyService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/api-keys/toggle", s.requireAdmin(handleToggleAPIKey(s.apiKeyService))).Methods("POST")

	// Credit System: Tier Limits (Admin)
	s.router.HandleFunc("/api/v1/admin/tiers/limits", s.requireAdmin(handleGetTierLimits(s.limitsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdmin(handleGetTierLimits(s.limitsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdmin(handleUpdateTierLimits(s.limitsService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdmin(handleCreateTierLimit(s.limitsService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdmin(handleDeleteTierLimit(s.limitsService))).Methods("DELETE")

	// Credit System: App Limits (Admin)
	s.router.HandleFunc("/api/v1/admin/apps/{app}/limits", s.requireAdmin(handleGetAppLimits(s.limitsService))).Methods("GET")

	// Credit System: Usage (Service-to-Service + User Auth + Admin)
	s.router.HandleFunc("/api/v1/usage/report", s.usageService.requireServiceAuth(handleReportUsage(s.usageService))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/summary", s.requireUserAuth(handleGetUsageSummary(s.usageService, s.accountService))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/check", s.requireUserAuth(handleCheckLimit(s.usageService))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/health", handleUsageHealth(s.usageService)).Methods("GET") // Unauthenticated for monitoring
	s.router.HandleFunc("/api/v1/admin/usage", s.requireAdmin(handleAdminUsageSummary(s.usageService))).Methods("GET")
}

func registerAIRoutes(s *Server) {
	// AI Gateway endpoints
	// Public endpoint for listing available models
	s.router.HandleFunc("/api/v1/ai/models", handleAIModels(s.aiGatewayDeps)).Methods("GET")
	// Health check (public for monitoring)
	s.router.HandleFunc("/api/v1/ai/health", handleAIHealth(s.aiGatewayDeps)).Methods("GET")
	// User auth required for AI operations
	s.router.HandleFunc("/api/v1/ai/chat", s.requireUserAuth(handleAIChat(s.aiGatewayDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/stream", s.requireUserAuth(handleAIStream(s.aiGatewayDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/usage", s.requireUserAuth(handleAIUsage(s.aiGatewayDeps))).Methods("GET")
}

func registerDocsRoutes(s *Server) {
	// Documentation endpoints (admin-only for viewing docs)
	s.router.HandleFunc("/api/v1/admin/docs/tree", s.requireAdmin(handleDocsTree())).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/docs/content", s.requireAdmin(handleDocsContent())).Methods("GET")
}

func registerAdminUserRoutes(s *Server) {
	// User Management endpoints (Admin)
	s.router.HandleFunc("/api/v1/admin/users", s.requireAdmin(handleAdminListUsers(s.db))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}", s.requireAdmin(handleAdminGetUser(s.db))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions", s.requireAdmin(handleAdminGetUserSessions(s.db))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/{sid}", s.requireAdmin(handleAdminRevokeUserSession(s.db))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/revoke-all", s.requireAdmin(handleAdminRevokeAllUserSessions(s.db))).Methods("POST")
}

func handleVariantSpaceRoute(space *VariantSpace) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(space.JSONBytes()); err != nil {
			logStructuredError("variant_space_write_failed", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
}
