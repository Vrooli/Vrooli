package main

import (
	"net/http"
	"time"

	"github.com/vrooli/api-core/health"
	accounthttp "landing-page-business-suite-api/handlers/account"
	adminhttp "landing-page-business-suite-api/handlers/admin"
	billinghttp "landing-page-business-suite-api/handlers/billing"
	measureshandler "landing-page-business-suite-api/handlers/measures"
	pricinghandler "landing-page-business-suite-api/handlers/pricing"
	variantspacehttp "landing-page-business-suite-api/handlers/variant_space"
)

func (s *Server) setupRoutes() {
	s.router.Use(securityHeadersMiddleware)
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
	registerMeasuresRoutes(s)
	registerDeployReadinessRoute(s)
}

func registerMeasuresRoutes(s *Server) {
	if err := measureshandler.RegisterRoutes(s.router, s.primaryDB(), nil, s.requireAdminOrService); err != nil {
		panic("register measures routes: " + err.Error())
	}
}

func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("X-XSS-Protection", "0")
		header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func registerDeployReadinessRoute(s *Server) {
	s.router.HandleFunc(
		"/api/v1/deploy-readiness",
		s.requireAdminOrService(handleDeployReadiness(s.downloadHosting, s.downloadService, s.remoteProfileService, s.planService)),
	).Methods("POST")
}

func registerHealthRoutes(s *Server) {
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.primaryDB()), health.Critical).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func registerLandingRoutes(s *Server) {
	registerLandingConfigConnectRoutes(s.router, s.landingConfigService)
	pricinghandler.RegisterRoutes(s.router, s.planService.GetPricingOverview)
	variantspacehttp.RegisterRoutes(s.router, s.variantSpace.JSONBytes)

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
	accounthttp.RegisterRoutes(s.router, s.accountService, getUserEmail, s.requireUserAuth)
	s.router.HandleFunc("/api/v1/downloads", s.requireUserAuth(handleDownloads(s.downloadAuthorizer, s.downloadHosting, s.planService))).Methods("GET")
}

func registerBillingRoutes(s *Server) {
	// Generated Connect payment procedures preserve the public checkout,
	// authenticated portal, and admin cancellation boundaries.
	billinghttp.RegisterConnectRoutes(s.router, billingConnectDependencies(s.stripeService), s.requireUserAuth, s.requireAdmin)

	// Stripe webhook remains Stripe's signed HTTP callback, not a browser RPC.
	s.router.HandleFunc("/api/v1/webhooks/stripe", handleStripeWebhook(s.stripeService)).Methods("POST")
}

func registerAdminCoreRoutes(s *Server) {
	// Admin authentication endpoints (OT-P0-008)
	s.router.HandleFunc("/api/v1/admin/login", adminhttp.Login(s.adminSessionDependencies())).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/logout", s.requireAdmin(adminhttp.Logout(s.adminSessionDependencies()))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/session", adminhttp.Session(s.adminSessionDependencies())).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfile)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfileUpdate)).Methods("PUT")
	registerStripeSettingsConnectRoutes(s.router, s.paymentSettings, s.stripeService, s.paymentAnomaly, s.requireAdmin)
	s.router.HandleFunc("/api/v1/admin/stripe/verify-price", s.requireAdmin(handleAdminVerifyStripePrice(s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/reset-demo-data", s.requireAdmin(adminhttp.ResetDemoData(adminhttp.ResetDependencies{Reset: s.resetDemoData, Now: time.Now, LogError: logStructuredError}))).Methods("POST")
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
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions", s.requireAdmin(handleAdminListIncomingRemoteProfileSessions(s.routedDB))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions/{session_id}", s.requireAdmin(handleAdminRevokeIncomingRemoteProfileSession(s.routedDB))).Methods("DELETE")
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
	registerBundleAdminConnectRoutes(s.router, s.planService, s.stripeService, s.requireAdmin)
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices", s.requireAdmin(handleAdminCreateBundlePrice(s.planService, s.stripeService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdmin(handleAdminDeleteBundlePrice(s.planService))).Methods("DELETE")

	// Stripe import tools
	s.router.HandleFunc("/api/v1/admin/stripe/import-preview", s.requireAdmin(handleAdminStripeImportPreview(s.stripeService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/stripe/import", s.requireAdmin(handleAdminStripeImport(s.stripeService, s.planService))).Methods("POST")

	registerCouponAdminConnectRoutes(s.router, s.stripeService, s.planService, s.routedDB, s.requireAdmin)
}

func registerVariantRoutes(s *Server) {
	registerVariantConnectRoutes(s.router, s.configStore, s.requireAdmin)

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
	registerBrandingConnectRoutes(s.router, s.configStore, s.requireAdmin)
	registerSEOConnectRoutes(s.router, s.seoService, s.configStore, s.requireAdmin)

	// Asset upload endpoints (admin-only for file uploads)
	s.router.HandleFunc("/api/v1/admin/assets", s.requireAdmin(handleAssetsList(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/upload", s.requireAdmin(handleAssetUpload(s.assetsService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetGet(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetDelete(s.assetsService))).Methods("DELETE")

	// Serve uploaded files publicly through the request-aware asset root.
	s.router.HandleFunc("/api/v1/uploads/{path:.*}", handleServeUpload(s.assetsService)).Methods("GET", "HEAD")

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
	// Feedback endpoints.
	s.router.HandleFunc("/api/v1/feedback", handleFeedbackCreateWithConfigStore(s.feedbackService, s.configStore, s.emailService)).Methods("POST")
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
	s.router.HandleFunc("/api/v1/admin/users", s.requireAdmin(handleAdminListUsers(s.userManagementService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}", s.requireAdmin(handleAdminGetUser(s.userManagementService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions", s.requireAdmin(handleAdminGetUserSessions(s.userManagementService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/{sid}", s.requireAdmin(handleAdminRevokeUserSession(s.userManagementService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/revoke-all", s.requireAdmin(handleAdminRevokeAllUserSessions(s.userManagementService))).Methods("POST")
}
