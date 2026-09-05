package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	accounthttp "landing-page-business-suite-api/handlers/account"
	adminhttp "landing-page-business-suite-api/handlers/administration"
	assethttp "landing-page-business-suite-api/handlers/assets"
	bundlehttp "landing-page-business-suite-api/handlers/bundles"
	billinghttp "landing-page-business-suite-api/handlers/commerce"
	landinghttp "landing-page-business-suite-api/handlers/config"
	contenthttp "landing-page-business-suite-api/handlers/content"
	couponhttp "landing-page-business-suite-api/handlers/coupons"
	downloadhttp "landing-page-business-suite-api/handlers/delivery"
	deploymenthttp "landing-page-business-suite-api/handlers/deployment"
	docshandler "landing-page-business-suite-api/handlers/docs"
	varianthttp "landing-page-business-suite-api/handlers/experimentation"
	feedbackhttp "landing-page-business-suite-api/handlers/feedback"
	intelligencehandler "landing-page-business-suite-api/handlers/intelligence"
	measureshandler "landing-page-business-suite-api/handlers/measures"
	metricshttp "landing-page-business-suite-api/handlers/metrics"
	pricinghandler "landing-page-business-suite-api/handlers/pricing"
	seohttp "landing-page-business-suite-api/handlers/seo"
	variantspacehttp "landing-page-business-suite-api/handlers/variant_space"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/monetization"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	entitlementclient "github.com/vrooli/vrooli/packages/entitlementclient-go"
)

func (s *Server) setupRoutes() {
	s.router.Use(securityHeadersMiddleware)
	s.router.Use(loggingMiddleware)

	registerHealthRoutes(s)
	registerMonetizationJourneyRoute(s)
	registerLandingRoutes(s)
	registerBackdropRoutes(s)
	registerAuthRoutes(s)
	registerFixtureRoutes(s)
	registerAccountRoutes(s)
	registerReceiptRoutes(s)
	registerBillingRoutes(s)
	registerAdminCoreRoutes(s)
	registerRemoteProfileRoutes(s)
	registerCommerceAdminRoutes(s)
	registerVariantRoutes(s)
	registerContentRoutes(s)
	registerMetricsRoutes(s)
	monetization.RegisterRoutes(s.router, s.primaryDB(), s.planService.BundleKey)
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

// registerBackdropRoutes keeps the dynamic Backdrop Studio address on the
// server side. The UI calls this same-origin endpoint; api-core discovery
// resolves the current lifecycle-managed port for every request.
func registerBackdropRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/backdrops/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimSpace(mux.Vars(r)["id"])
		if id == "" {
			http.Error(w, "backdrop id is required", http.StatusBadRequest)
			return
		}
		resolve := s.backdropResolver
		if resolve == nil {
			resolve = func(ctx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(ctx, "backdrop-studio")
			}
		}
		baseURL, err := resolve(r.Context())
		if err != nil || strings.TrimSpace(baseURL) == "" {
			http.Error(w, "backdrop studio is unavailable", http.StatusServiceUnavailable)
			return
		}
		body, err := json.Marshal(map[string]string{"id": id})
		if err != nil {
			http.Error(w, "failed to encode backdrop request", http.StatusInternalServerError)
			return
		}
		request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(baseURL, "/")+"/vrooli.backdrop_studio.v1.release.ReleaseService/GetReference", bytes.NewReader(body))
		if err != nil {
			http.Error(w, "failed to create backdrop request", http.StatusInternalServerError)
			return
		}
		request.Header.Set("Content-Type", "application/json")
		client := s.backdropHTTPClient
		if client == nil {
			client = http.DefaultClient
		}
		response, err := client.Do(request)
		if err != nil {
			http.Error(w, "backdrop studio request failed", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(response.StatusCode)
		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
			payload, readErr := io.ReadAll(response.Body)
			if readErr == nil {
				var metadata map[string]any
				if json.Unmarshal(payload, &metadata) == nil {
					if uri, ok := metadata["uri"].(string); ok && uri != "" {
						if assetURL, ok := metadata["url"].(string); !ok || assetURL == "" || !strings.HasPrefix(assetURL, "http://") && !strings.HasPrefix(assetURL, "https://") {
							metadata["url"] = uri
							if !strings.HasPrefix(uri, "http://") && !strings.HasPrefix(uri, "https://") {
								metadata["url"] = strings.TrimRight(baseURL, "/") + "/" + strings.TrimLeft(uri, "/")
							}
						}
					}
					if encoded, encodeErr := json.Marshal(metadata); encodeErr == nil {
						_, _ = w.Write(encoded)
						return
					}
				}
				_, _ = w.Write(payload)
				return
			}
		}
		_, _ = io.Copy(w, response.Body)
	}).Methods(http.MethodGet)
}

func registerReceiptRoutes(s *Server) {
	s.router.HandleFunc("/api/v1/subscriptions/receipts", s.requireUserAuth(billinghttp.RegisterReceipt(billinghttp.ReceiptDependencies{
		Validators: s.receiptValidators,
		Register:   s.accountService.RegisterReceipt,
		UserIdentity: func(ctx context.Context) string {
			return getUserEmail(ctx)
		},
		WriteError: writeJSONError,
	}))).Methods(http.MethodPost)
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
	deps := deploymenthttp.Dependencies{
		Storage: s.downloadHosting, Catalog: s.downloadService, RemoteProfiles: s.remoteProfileService,
		BundleKey: s.planService.BundleKey,
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			writeJSONError(w, status, message, kind)
		},
	}
	deploymenthttp.RegisterConnectRoutes(s.router, deps, s.requireAdminOrService)
	// Preserve the documented JSON endpoint until its existing callers migrate.
	s.router.HandleFunc(
		"/api/v1/deploy-readiness",
		s.requireAdminOrService(deploymenthttp.Readiness(deps)),
	).Methods("POST")
}

func registerHealthRoutes(s *Server) {
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.primaryDB()), health.Critical).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")
}

func registerLandingRoutes(s *Server) {
	landinghttp.RegisterLandingConfigConnectRoutes(s.router, s.landingConfigService)
	pricinghandler.RegisterRoutes(s.router, s.planService.GetPricingOverview)
	variantspacehttp.RegisterRoutes(s.router, s.variantSpace.JSONBytes)

	// Customization command for landing updates
	s.router.HandleFunc("/api/v1/customize", contenthttp.Customize(time.Now)).Methods("POST")
}

func registerAuthRoutes(s *Server) {
	// Consumer access-token verification is intentionally public: bundled
	// scenarios need the public keys, while the private signing key stays in
	// this authority and is never distributed to a relying party.
	s.router.HandleFunc("/.well-known/jwks.json", func(w http.ResponseWriter, _ *http.Request) {
		body, err := s.userAuthService.PublicKeySet()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "consumer key set unavailable", ApiErrorTypeServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=300")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}).Methods("GET")
	// User Authentication endpoints (magic link + JWT)
	// Public auth endpoints (no auth required)
	deps := userAuthHandlerDependencies(s.userAuthService, s.magicLinkLimiter)
	s.router.HandleFunc("/api/v1/auth/magic-link", adminhttp.RequestMagicLink(deps)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/verify", adminhttp.VerifyMagicLink(deps)).Methods("GET")
	s.router.HandleFunc("/api/v1/auth/authorize", adminhttp.AuthorizeWithPKCE(deps, s.authorizationCodes)).Methods("GET")
	s.router.HandleFunc("/api/v1/auth/token", adminhttp.ExchangeAuthorizationCode(deps, s.authorizationCodes)).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/refresh", adminhttp.RefreshTokens(deps)).Methods("POST")
	// Protected auth endpoints (require user auth)
	s.router.HandleFunc("/api/v1/auth/logout", s.requireUserAuth(adminhttp.LogoutUser(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/auth/me", s.requireUserAuth(adminhttp.Me(deps))).Methods("GET")
}

func registerAccountRoutes(s *Server) {
	accounthttp.RegisterRoutes(s.router, accounthttp.NewCommerceReader(s.accountService), getUserEmail, s.requireUserAuth)
	registerEntitlementRoute(s)
	downloadhttp.RegisterConnectAuthorizationRoute(s.router, s.planService.BundleKey, s.downloadService, downloadConnectAuthorizationDependencies(s.downloadAuthorizer, s.downloadHosting, s.planService), s.requireUserAuth)
	s.router.HandleFunc("/api/v1/downloads", s.requireUserAuth(downloadhttp.Authorize(downloadAuthorizationDependencies(s.downloadAuthorizer, s.downloadHosting, s.planService)))).Methods("GET")
}

func registerEntitlementRoute(s *Server) {
	s.router.HandleFunc("/api/v1/entitlements", s.requireUserAuth(func(w http.ResponseWriter, r *http.Request) {
		identity := getUserEmail(r.Context())
		if requested := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("user"))); requested != "" && requested != strings.ToLower(identity) {
			writeJSONError(w, http.StatusForbidden, "entitlement identity does not match token", ApiErrorTypeForbidden)
			return
		}
		payload, err := s.accountService.GetEntitlementsContext(r.Context(), identity)
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "entitlement service unavailable", ApiErrorTypeServerError)
			return
		}
		if payload.SharedDecision().Warning {
			w.Header().Set("X-Entitlement-Warning", "past_due")
		}
		lease, err := s.userAuthService.SignEntitlementLease(entitlementclient.Payload{
			UserIdentity:      identity,
			Status:            payload.Status,
			PlanTier:          payload.PlanTier,
			PlanRank:          payload.PlanRank,
			PriceID:           payload.PriceID,
			Features:          payload.Features,
			Limits:            payload.Limits,
			NotAfter:          payload.NotAfter,
			BillingCycleStart: payload.BillingCycleStart,
			Credits:           payload.Credits,
			Subscription:      payload.Subscription,
		})
		if err != nil {
			writeJSONError(w, http.StatusServiceUnavailable, "entitlement signing unavailable", ApiErrorTypeServerError)
			return
		}
		payload.Lease = lease
		writeJSON(w, payload)
	})).Methods(http.MethodGet)
}

func registerBillingRoutes(s *Server) {
	// Generated Connect payment procedures preserve the public checkout,
	// authenticated portal, and admin cancellation boundaries.
	billinghttp.RegisterConnectRoutes(s.router, billingConnectDependencies(s.stripeService), s.requireUserAuth, s.requireAdmin)

	// Stripe webhook remains Stripe's signed HTTP callback, not a browser RPC.
	s.router.HandleFunc("/api/v1/webhooks/stripe", billinghttp.Webhook(billingWebhookDependencies(s.stripeService))).Methods("POST")
}

func registerAdminCoreRoutes(s *Server) {
	// Admin authentication/reset are generated Connect services. Session cookies
	// remain response headers and never enter protobuf payloads.
	adminhttp.RegisterSessionConnectRoutes(s.router, s.adminSessionDependencies(), adminhttp.ResetDependencies{Reset: s.resetDemoData, Now: time.Now, LogError: logx.Error}, s.requireAdmin)
	profileDeps := s.adminProfileDependencies()
	adminhttp.RegisterProfileConnectRoutes(s.router, profileDeps, s.requireAdmin)
	billinghttp.RegisterStripeSettingsConnectRoutes(s.router, s.paymentSettings, s.stripeService, s.paymentAnomaly, s.requireAdmin)
	s.router.HandleFunc("/api/v1/admin/stripe/verify-price", s.requireAdmin(bundlehttp.VerifyStripePrice(bundleStripeHandlerDependencies(s.stripeService)))).Methods("GET")
	adminhttp.RegisterAPIKeyConnectRoutes(s.router, s.apiKeyService, s.requireAdmin)
}

func registerRemoteProfileRoutes(s *Server) {
	// List and test/proxy use requireAdminOrService so inter-scenario clients (e.g. s2d) can call them with a service bearer token.
	deps := remoteProfileHandlerDependencies(s.remoteProfileService, s.sessionAdminEmail)
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdminOrService(adminhttp.ListRemoteProfiles(deps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profiles", s.requireAdmin(adminhttp.CreateRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdmin(adminhttp.UpdateRemoteProfile(deps))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}", s.requireAdmin(adminhttp.DeleteRemoteProfile(deps))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/login", s.requireAdmin(adminhttp.LoginRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/logout", s.requireAdmin(adminhttp.LogoutRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/test", s.requireAdminOrService(adminhttp.TestRemoteProfile(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/session-links", s.requireAdmin(adminhttp.RemoteProfileSessionLinks(deps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/remote-revoke", s.requireAdmin(adminhttp.RevokeRemoteProfileSessions(deps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/remote-profiles/{id}/proxy", s.requireAdminOrService(adminhttp.ProxyRemoteProfile(deps))).Methods("POST")
	sessionDeps := remoteProfileSessionDependencies(s.routedDB)
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions", s.requireAdmin(adminhttp.ListIncomingRemoteProfileSessions(sessionDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/remote-profile-sessions/{session_id}", s.requireAdmin(adminhttp.RevokeIncomingRemoteProfileSession(sessionDeps))).Methods("DELETE")
}

func registerCommerceAdminRoutes(s *Server) {
	// Download hosting + assets
	downloadhttp.RegisterConnectAppRoutes(s.router, s.planService.BundleKey, s.downloadService, s.requireAdmin)
	downloadAppDependencies := deliveryAppDependencies(s.planService)
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(downloadhttp.ListApps(downloadAppDependencies, s.downloadService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(downloadhttp.CreateApp(downloadAppDependencies, s.downloadService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(downloadhttp.SaveApp(downloadAppDependencies, s.downloadService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(downloadhttp.DeleteApp(downloadAppDependencies, s.downloadService))).Methods("DELETE")
	downloadAdminDependencies := downloadAdminDependencies(s.downloadHosting, s.planService)
	downloadAdminAssetDependencies := downloadAdminAssetDependencies(s.downloadService, s.downloadHosting, s.planService)
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdmin(downloadhttp.GetStorage(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-storage", s.requireAdmin(downloadhttp.UpdateStorage(downloadAdminDependencies))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-storage/test", s.requireAdmin(downloadhttp.TestStorage(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts", s.requireAdmin(downloadhttp.ListArtifacts(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/by-app", s.requireAdmin(downloadhttp.ListArtifactsByApp(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/presign-upload", s.requireAdminOrService(downloadhttp.PresignUpload(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/commit", s.requireAdminOrService(downloadhttp.CommitArtifact(downloadAdminDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-artifacts/{artifact_id}/presign-get", s.requireAdmin(downloadhttp.PresignGet(downloadAdminDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-assets/apply", s.requireAdminOrService(downloadhttp.ApplyArtifact(downloadAdminAssetDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-assets/set-current", s.requireAdmin(downloadhttp.SetArtifactCurrent(downloadAdminAssetDependencies))).Methods("POST")

	// Bundles + pricing
	registerBundleAdminConnectRoutes(s.router, s.planService, s.stripeService, s.requireAdmin)
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices", s.requireAdmin(bundlehttp.CreatePrice(bundleCreateHandlerDependencies(s.planService, s.stripeService)))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdmin(bundlehttp.DeletePrice(bundleDeleteHandlerDependencies(s.planService)))).Methods("DELETE")

	// Stripe import tools
	s.router.HandleFunc("/api/v1/admin/stripe/import-preview", s.requireAdmin(bundlehttp.PreviewStripeImport(bundleImportHandlerDependencies(s.stripeService, s.planService)))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/stripe/import", s.requireAdmin(bundlehttp.ImportStripePrices(bundleStripeImportDependencies(s.stripeService, s.planService)))).Methods("POST")

	couponhttp.RegisterConnectRoutes(s.router, s.stripeService, s.planService, s.routedDB, s.requireAdmin, couponProviderError, logx.Info)
}

func registerVariantRoutes(s *Server) {
	varianthttp.RegisterConnectRoutes(s.router, s.configStore, s.requireAdmin)
	writeDependencies := varianthttp.WriteDependencies{Store: s.configStore, WriteJSON: writeJSON, WriteError: writeJSONError, Log: logx.Info, LogError: logx.Error}

	// A/B Testing variant endpoints (OT-P0-014 through OT-P0-018)
	// Public endpoints (no auth required for landing page display)
	s.router.HandleFunc("/api/v1/variants/select", varianthttp.Select(variantReadDependencies(s.configStore, "/api/v1/variants/"))).Methods("GET")
	s.router.HandleFunc("/api/v1/public/variants/{slug}", varianthttp.PublicGet(variantReadDependencies(s.configStore, "/api/v1/public/variants/"))).Methods("GET")
	s.router.HandleFunc("/api/v1/public/variants/{variant_slug}/sections", contenthttp.Public(contentHTTPDependencies(s.configStore))).Methods("GET")

	// Admin endpoints (require auth)
	s.router.HandleFunc("/api/v1/variants", s.requireAdmin(varianthttp.List(variantReadDependencies(s.configStore, "")))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(varianthttp.AdminGet(variantReadDependencies(s.configStore, "/api/v1/variants/")))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(varianthttp.Update(writeDependencies))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(varianthttp.Delete(writeDependencies))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/variants/sync", s.requireAdmin(varianthttp.Sync(writeDependencies))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/export", s.requireAdmin(varianthttp.Export(writeDependencies))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/import", s.requireAdmin(varianthttp.Import(writeDependencies))).Methods("PUT")

	// Content Customization endpoints - sections are part of variant snapshots
	s.router.HandleFunc("/api/v1/variants/{variant_slug}/sections", s.requireAdmin(contenthttp.Admin(contentHTTPDependencies(s.configStore)))).Methods("GET")
}

func registerContentRoutes(s *Server) {
	varianthttp.RegisterBrandingConnectRoutes(s.router, s.configStore, s.requireAdmin)
	contenthttp.RegisterSEOConnectRoutes(s.router, s.seoService, s.requireAdmin)
	contenthttp.RegisterAssetsConnectRoutes(s.router, s.assetsService, s.requireAdmin)

	// Asset upload endpoints (admin-only for file uploads)
	s.router.HandleFunc("/api/v1/admin/assets/upload", s.requireAdmin(assethttp.Upload(assetsHTTPDependencies(s.assetsService)))).Methods("POST")

	// Serve uploaded files publicly through the request-aware asset root.
	s.router.HandleFunc("/api/v1/uploads/{path:.*}", assethttp.Serve(assetsHTTPDependencies(s.assetsService))).Methods("GET", "HEAD")

	// Sitemap and robots.txt
	s.router.HandleFunc("/sitemap.xml", seohttp.Sitemap(seoHTTPDependencies(s.seoService))).Methods("GET")
	s.router.HandleFunc("/robots.txt", seohttp.Robots(seoHTTPDependencies(s.seoService))).Methods("GET")
}

func registerMetricsRoutes(s *Server) {
	metricshttp.RegisterConnectRoutes(s.router, metricsConnectDependencies(s.metricsService), s.requireAdminOrService)
}

func registerFeedbackRoutes(s *Server) {
	feedbackhttp.RegisterConnectRoutes(s.router, s.feedbackService, feedbackEmailNotifier{configStore: s.configStore, emailService: s.emailService}, s.requireAdmin)
}

func registerWaitlistRoutes(s *Server) {
	metricshttp.RegisterWaitlistConnectRoutes(s.router, metricshttp.WaitlistConnectDependencies{
		Service: s.waitlistService, ValidateEmail: ValidateEmail,
	}, s.requireAdmin)
}

func registerCreditsRoutes(s *Server) {
	// Credit System: Tier Limits (Admin)
	limitsDeps := billingLimitsDependencies()
	s.router.HandleFunc("/api/v1/admin/tiers/limits", s.requireAdmin(billinghttp.GetTierLimits(s.limitsService, limitsDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdmin(billinghttp.GetTierLimits(s.limitsService, limitsDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/tiers/{tier}/limits", s.requireAdmin(billinghttp.UpdateTierLimits(s.limitsService, limitsDeps))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdmin(billinghttp.CreateTierLimit(s.limitsService, limitsDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/limits", s.requireAdmin(billinghttp.DeleteTierLimit(s.limitsService, limitsDeps))).Methods("DELETE")

	// Credit System: App Limits (Admin)
	s.router.HandleFunc("/api/v1/admin/apps/{app}/limits", s.requireAdmin(billinghttp.GetAppLimits(s.limitsService, limitsDeps))).Methods("GET")

	// Credit System: Usage (User Auth + Admin). The write path derives identity
	// from the verified access token; no shared service credential is accepted.
	usageDeps := usageHTTPDependencies()
	s.router.HandleFunc("/api/v1/usage/report", s.requireUserAuth(billinghttp.ReportUsage(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations", s.requireUserAuth(billinghttp.ReserveCredits(s.usageService, s.accountService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations/{reservationID}/finalize", s.requireUserAuth(billinghttp.FinalizeReservation(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/reservations/{reservationID}/release", s.requireUserAuth(billinghttp.ReleaseReservation(s.usageService, usageDeps))).Methods("POST")
	s.router.HandleFunc("/api/v1/usage/summary", s.requireUserAuth(billinghttp.GetUsageSummary(s.usageService, s.accountService, usageDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/check", s.requireUserAuth(billinghttp.CheckLimit(s.usageService, usageDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/usage/health", billinghttp.UsageHealth(s.usageService, usageDeps)).Methods("GET") // Unauthenticated for monitoring
	s.router.HandleFunc("/api/v1/admin/usage", s.requireAdmin(billinghttp.AdminUsageSummary(s.usageService, usageDeps))).Methods("GET")
}

func usageHTTPDependencies() billinghttp.UsageDependencies {
	return billinghttp.UsageDependencies{
		UserEmail:  getUserEmail,
		WriteError: writeJSONError,
		LogError:   logx.Error,
	}
}

func registerAIRoutes(s *Server) {
	intelligencehandler.RegisterConnectRoutes(s.router, s.meteredInferenceDeps, s.requireUserAuth)

	// AI MeteredInferenceProvider endpoints
	// Public endpoint for listing available models
	s.router.HandleFunc("/api/v1/ai/models", s.meteredInferenceHandler.Models()).Methods("GET")
	// Health check (public for monitoring)
	s.router.HandleFunc("/api/v1/ai/health", s.meteredInferenceHandler.Health()).Methods("GET")
	// User auth required for AI operations
	s.router.HandleFunc("/api/v1/ai/chat", s.requireUserAuth(s.meteredInferenceHandler.Chat())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/inference", s.requireUserAuth(s.meteredInferenceHandler.Inference())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/stream", s.requireUserAuth(s.meteredInferenceHandler.Stream())).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/usage", s.requireUserAuth(s.meteredInferenceHandler.Usage())).Methods("GET")
}

func registerDocsRoutes(s *Server) {
	// Documentation is a generated Connect service; the UI uses its typed client.
	docshandler.RegisterConnectRoutes(s.router, docsConnectDependencies(), s.requireAdmin)
}

func registerAdminUserRoutes(s *Server) {
	// User Management endpoints (Admin)
	userDeps := userManagementDependencies(s.userManagementService)
	s.router.HandleFunc("/api/v1/admin/users", s.requireAdmin(adminhttp.ListUsers(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}", s.requireAdmin(adminhttp.GetUser(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions", s.requireAdmin(adminhttp.ListUserSessions(userDeps))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/{sid}", s.requireAdmin(adminhttp.RevokeUserSession(userDeps))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/users/{id}/sessions/revoke-all", s.requireAdmin(adminhttp.RevokeAllUserSessions(userDeps))).Methods("POST")
}
