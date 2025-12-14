package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config               *Config
	db                   *sql.DB
	router               *mux.Router
	variantSpace         *VariantSpace
	variantService       *VariantService
	metricsService       *MetricsService
	stripeService        *StripeService
	contentService       *ContentService
	planService          *PlanService
	downloadService      *DownloadService
	downloadAuthorizer   *DownloadAuthorizer
	accountService       *AccountService
	landingConfigService *LandingConfigService
	paymentSettings      *PaymentSettingsService
	brandingService      *BrandingService
	assetsService        *AssetsService
	seoService           *SEOService
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := seedDefaultData(db); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}

	variantSpace := defaultVariantSpace
	variantService := NewVariantService(db, variantSpace)
	contentService := NewContentService(db)
	planService := NewPlanService(db)
	downloadService := NewDownloadService(db)
	accountService := NewAccountService(db, planService)
	downloadAuthorizer := NewDownloadAuthorizer(downloadService, accountService, planService.BundleKey())
	paymentSettings := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	brandingService := NewBrandingService(db)
	assetsService := NewAssetsService(db)
	seoService := NewSEOService(brandingService, variantService)

	if err := syncVariantSnapshots(variantService, contentService); err != nil {
		return nil, fmt.Errorf("failed to sync variant snapshots: %w", err)
	}

	srv := &Server{
		config:               cfg,
		db:                   db,
		router:               mux.NewRouter(),
		variantSpace:         variantSpace,
		variantService:       variantService,
		metricsService:       NewMetricsService(db),
		stripeService:        stripeService,
		contentService:       contentService,
		planService:          planService,
		downloadService:      downloadService,
		downloadAuthorizer:   downloadAuthorizer,
		accountService:       accountService,
		landingConfigService: NewLandingConfigService(variantService, contentService, planService, downloadService, brandingService),
		paymentSettings:      paymentSettings,
		brandingService:      brandingService,
		assetsService:        assetsService,
		seoService:           seoService,
	}

	// Initialize session store for authentication
	initSessionStore()

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")

	// Landing config + plans
	s.router.HandleFunc("/api/v1/landing-config", handleLandingConfig(s.landingConfigService)).Methods("GET")
	s.router.HandleFunc("/api/v1/plans", handlePlans(s.planService)).Methods("GET")
	s.router.HandleFunc("/api/v1/variant-space", handleVariantSpaceRoute(s.variantSpace)).Methods("GET")

	// Billing APIs
	s.router.HandleFunc("/api/v1/billing/create-checkout-session", handleBillingCreateCheckoutSession(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/billing/create-credits-checkout-session", handleBillingCreateCreditsSession(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/billing/portal-url", handleBillingPortalURL(s.stripeService)).Methods("GET")

	// Account endpoints
	s.router.HandleFunc("/api/v1/me/subscription", handleMeSubscription(s.accountService)).Methods("GET")
	s.router.HandleFunc("/api/v1/me/credits", handleMeCredits(s.accountService)).Methods("GET")
	s.router.HandleFunc("/api/v1/entitlements", handleEntitlements(s.accountService)).Methods("GET")
	s.router.HandleFunc("/api/v1/downloads", handleDownloads(s.downloadAuthorizer)).Methods("GET")

	s.router.HandleFunc("/api/v1/customize", s.handleCustomize).Methods("POST")

	// Admin authentication endpoints (OT-P0-008)
	s.router.HandleFunc("/api/v1/admin/login", s.handleAdminLogin).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/logout", s.requireAdmin(s.handleAdminLogout)).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/session", s.handleAdminSession).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfile)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/profile", s.requireAdmin(s.handleAdminProfileUpdate)).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/settings/stripe", s.requireAdmin(handleGetStripeSettings(s.paymentSettings, s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/settings/stripe", s.requireAdmin(handleUpdateStripeSettings(s.paymentSettings, s.stripeService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/stripe/verify-price", s.requireAdmin(handleAdminVerifyStripePrice(s.stripeService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/reset-demo-data", s.requireAdmin(s.handleAdminResetDemoData)).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(handleAdminListDownloadApps(s.downloadService, s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/download-apps", s.requireAdmin(handleAdminCreateDownloadApp(s.downloadService, s.planService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(handleAdminSaveDownloadApp(s.downloadService, s.planService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/download-apps/{app_key}", s.requireAdmin(handleAdminDeleteDownloadApp(s.downloadService, s.planService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/bundles", s.requireAdmin(handleAdminBundleCatalog(s.planService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/bundles/{bundle_key}/prices/{price_id}", s.requireAdmin(handleAdminUpdateBundlePrice(s.planService))).Methods("PATCH")

	// A/B Testing variant endpoints (OT-P0-014 through OT-P0-018)
	// Public endpoints (no auth required for landing page display)
	s.router.HandleFunc("/api/v1/variants/select", handleVariantSelect(s.variantService)).Methods("GET")
	s.router.HandleFunc("/api/v1/public/variants/{slug}", handlePublicVariantBySlug(s.variantService)).Methods("GET")
	// Admin endpoints (require auth)
	s.router.HandleFunc("/api/v1/variants", s.requireAdmin(handleVariantsList(s.variantService))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants", s.requireAdmin(handleVariantCreateWithSections(s.variantService, s.contentService))).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantBySlug(s.variantService))).Methods("GET")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantUpdate(s.variantService))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/variants/{slug}/archive", s.requireAdmin(handleVariantArchive(s.variantService))).Methods("POST")
	s.router.HandleFunc("/api/v1/variants/{slug}", s.requireAdmin(handleVariantDelete(s.variantService))).Methods("DELETE")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/export", s.requireAdmin(handleVariantExport(s.variantService, s.contentService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/import", s.requireAdmin(handleVariantImport(s.variantService, s.contentService))).Methods("PUT")

	// Metrics & Analytics endpoints (OT-P0-019 through OT-P0-024)
	s.router.HandleFunc("/api/v1/metrics/track", handleMetricsTrack(s.metricsService)).Methods("POST")
	s.router.HandleFunc("/api/v1/metrics/summary", s.requireAdmin(handleMetricsSummary(s.metricsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/metrics/variants", s.requireAdmin(handleMetricsVariantStats(s.metricsService))).Methods("GET")

	// Stripe Payment endpoints (OT-P0-025 through OT-P0-030)
	s.router.HandleFunc("/api/v1/checkout/create", handleCheckoutCreate(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/webhooks/stripe", handleStripeWebhook(s.stripeService)).Methods("POST")
	s.router.HandleFunc("/api/v1/subscription/verify", handleSubscriptionVerify(s.stripeService)).Methods("GET")
	s.router.HandleFunc("/api/v1/subscription/cancel", s.requireAdmin(handleSubscriptionCancel(s.stripeService))).Methods("POST")

	// Public content endpoint for landing page display (no auth required)
	s.router.HandleFunc("/api/v1/public/variants/{variant_id}/sections", handleGetPublicSections(s.contentService)).Methods("GET")

	// Content Customization endpoints (OT-P0-012, OT-P0-013: CUSTOM-SPLIT, CUSTOM-LIVE)
	s.router.HandleFunc("/api/v1/variants/{variant_id}/sections", s.requireAdmin(handleGetSections(s.contentService))).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", s.requireAdmin(handleGetSection(s.contentService))).Methods("GET")
	s.router.HandleFunc("/api/v1/sections/{id}", s.requireAdmin(handleUpdateSection(s.contentService))).Methods("PATCH")
	s.router.HandleFunc("/api/v1/sections", s.requireAdmin(handleCreateSection(s.contentService))).Methods("POST")
	s.router.HandleFunc("/api/v1/sections/{id}", s.requireAdmin(handleDeleteSection(s.contentService))).Methods("DELETE")

	// Branding endpoints (admin-only for site-wide branding)
	s.router.HandleFunc("/api/v1/admin/branding", s.requireAdmin(handleGetBranding(s.brandingService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/branding", s.requireAdmin(handleUpdateBranding(s.brandingService))).Methods("PUT")
	s.router.HandleFunc("/api/v1/admin/branding/clear-field", s.requireAdmin(handleClearBrandingField(s.brandingService))).Methods("POST")
	s.router.HandleFunc("/api/v1/branding", handleGetPublicBranding(s.brandingService)).Methods("GET")

	// Asset upload endpoints (admin-only for file uploads)
	s.router.HandleFunc("/api/v1/admin/assets", s.requireAdmin(handleAssetsList(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/upload", s.requireAdmin(handleAssetUpload(s.assetsService))).Methods("POST")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetGet(s.assetsService))).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/assets/{id}", s.requireAdmin(handleAssetDelete(s.assetsService))).Methods("DELETE")

	// Serve uploaded files publicly
	s.router.PathPrefix("/api/v1/uploads/").Handler(http.StripPrefix("/api/v1/uploads/", http.FileServer(http.Dir(s.assetsService.GetUploadDir()))))

	// SEO endpoints
	s.router.HandleFunc("/api/v1/seo/{slug}", handleGetVariantSEO(s.seoService)).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/variants/{slug}/seo", s.requireAdmin(handleUpdateVariantSEO(s.variantService))).Methods("PUT")

	// Sitemap and robots.txt
	s.router.HandleFunc("/sitemap.xml", handleSitemapXML(s.seoService)).Methods("GET")
	s.router.HandleFunc("/robots.txt", handleRobotsTXT(s.seoService)).Methods("GET")

	// Documentation endpoints (admin-only for viewing docs)
	s.router.HandleFunc("/api/v1/admin/docs/tree", s.requireAdmin(handleDocsTree())).Methods("GET")
	s.router.HandleFunc("/api/v1/admin/docs/content", s.requireAdmin(handleDocsContent())).Methods("GET")
}

func handleVariantSpaceRoute(space *VariantSpace) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(space.JSONBytes())
	}
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "landing-page-business-suite-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := "healthy"
	dbStatus := "connected"

	if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "landing-page-business-suite API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleAdminResetDemoData(w http.ResponseWriter, r *http.Request) {
	if !adminResetEnabled() {
		http.Error(w, "demo reset disabled", http.StatusForbidden)
		return
	}

	if err := s.resetDemoData(r.Context()); err != nil {
		logStructuredError("admin_reset_failed", map[string]interface{}{"error": err.Error()})
		http.Error(w, "failed to reset demo data", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"reset":     true,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) resetDemoData(ctx context.Context) error {
	tables := []string{
		"content_sections",
		"variant_axes",
		"variants",
		"download_assets",
		"download_apps",
		"bundle_prices",
		"bundle_products",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range tables {
		stmt := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return seedDefaultData(s.db)
}

func adminResetEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_ADMIN_RESET")), "true")
}

// seedDefaultData ensures the public landing page has content without requiring admin setup
func seedDefaultData(db *sql.DB) error {
	if err := ensureSchema(db); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	// Seed default admin user for local use
	if _, err := db.Exec(
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)
		 ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		defaultAdminEmail,
		defaultAdminPasswordHash,
	); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	if _, err := db.Exec(`
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, dashboard_url, updated_at)
		VALUES (1, NULL, NULL, NULL, NULL, NOW())
		ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed payment settings: %w", err)
	}

	// Seed default site branding (singleton)
	if _, err := db.Exec(`
		INSERT INTO site_branding (id, site_name, robots_txt)
		VALUES (1, 'My Landing', E'User-agent: *\nAllow: /')
		ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed site branding: %w", err)
	}

	// Ensure control and variant-a exist and are active
	controlID, err := upsertVariant(db, "control", "Control (Original)", "Original landing page design", 50)
	if err != nil {
		return err
	}
	if _, err := upsertVariant(db, "variant-a", "Variant A", "Experimental variant A", 50); err != nil {
		return err
	}

	if err := upsertVariantAxesBySlug(db, "control", map[string]string{
		"persona":         "ops_leader",
		"jtbd":            "launch_bundle",
		"conversionStyle": "demo_led",
	}); err != nil {
		return err
	}
	if err := upsertVariantAxesBySlug(db, "variant-a", map[string]string{
		"persona":         "automation_freelancer",
		"jtbd":            "scale_services",
		"conversionStyle": "self_serve",
	}); err != nil {
		return err
	}

	// Seed sections for control if none exist
	var sectionCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM content_sections WHERE variant_id = $1`, controlID).Scan(&sectionCount); err != nil {
		return fmt.Errorf("failed to count sections: %w", err)
	}

	if sectionCount == 0 {
		if err := seedSectionsFromFallback(db, controlID); err != nil {
			return err
		}
	}

	if err := ensureVariantAxesDefaults(db, defaultVariantSpace); err != nil {
		return fmt.Errorf("failed to seed variant axes defaults: %w", err)
	}

	if err := seedBundlePricingDefaults(db, fallbackLanding.Pricing); err != nil {
		return err
	}

	if err := seedDownloadDefaults(db, fallbackLanding.Downloads); err != nil {
		return err
	}

	return nil
}

func defaultSectionSeeds() []LandingSection {
	if len(fallbackLanding.Sections) > 0 {
		return fallbackLanding.Sections
	}

	return []LandingSection{
		{
			SectionType: "hero",
			Order:       1,
			Enabled:     true,
			Content: map[string]interface{}{
				"headline":    "Build Landing Pages in Minutes",
				"subheadline": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in",
				"cta_text":    "Get Started Free",
				"cta_url":     "/signup",
			},
		},
		{
			SectionType: "features",
			Order:       2,
			Enabled:     true,
			Content: map[string]interface{}{
				"title": "Everything You Need",
				"items": []map[string]interface{}{
					{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "Zap"},
					{"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "BarChart"},
					{"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "CreditCard"},
				},
			},
		},
		{
			SectionType: "pricing",
			Order:       3,
			Enabled:     true,
			Content: map[string]interface{}{
				"title": "Simple Pricing",
				"plans": []map[string]interface{}{
					{"name": "Starter", "price": "$29", "features": []string{"5 landing pages", "Basic analytics", "Email support"}, "cta_text": "Start Free Trial"},
					{"name": "Pro", "price": "$99", "features": []string{"Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"}, "cta_text": "Get Started", "highlighted": true},
				},
			},
		},
		{
			SectionType: "cta",
			Order:       4,
			Enabled:     true,
			Content: map[string]interface{}{
				"headline":    "Ready to Launch Your Landing Page?",
				"subheadline": "Join thousands of marketers using Landing Manager",
				"cta_text":    "Start Building Now",
				"cta_url":     "/signup",
			},
		},
	}
}

func seedSectionsFromFallback(db *sql.DB, variantID int) error {
	sections := defaultSectionSeeds()
	for idx, section := range sections {
		sectionType := strings.TrimSpace(section.SectionType)
		if sectionType == "" {
			continue
		}

		order := section.Order
		if order <= 0 {
			order = idx + 1
		}

		content := section.Content
		if content == nil {
			content = map[string]interface{}{}
		}
		payload, err := json.Marshal(content)
		if err != nil {
			return fmt.Errorf("marshal section %s: %w", sectionType, err)
		}

		enabled := section.Enabled
		if _, err := db.Exec(`
			INSERT INTO content_sections (variant_id, section_type, content, "order", enabled)
			VALUES ($1, $2, $3::jsonb, $4, $5)
		`, variantID, sectionType, payload, order, enabled); err != nil {
			return fmt.Errorf("failed to seed section %s: %w", sectionType, err)
		}
	}
	return nil
}

func seedBundlePricingDefaults(db *sql.DB, pricing PricingOverview) error {
	if pricing.Bundle.BundleKey == "" {
		return nil
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM bundle_products`).Scan(&count); err != nil {
		return fmt.Errorf("count bundle products: %w", err)
	}
	if count > 0 {
		return nil
	}

	bundle := pricing.Bundle
	bundleMetadata, err := json.Marshal(jsonValueToMap(bundle.Metadata))
	if err != nil {
		return fmt.Errorf("marshal bundle metadata: %w", err)
	}

	var productID int64
	insertProduct := `
		INSERT INTO bundle_products (bundle_key, bundle_name, stripe_product_id, credits_per_usd,
			display_credits_multiplier, display_credits_label, environment, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		ON CONFLICT (bundle_key)
		DO UPDATE SET bundle_name = EXCLUDED.bundle_name,
			stripe_product_id = EXCLUDED.stripe_product_id,
			credits_per_usd = EXCLUDED.credits_per_usd,
			display_credits_multiplier = EXCLUDED.display_credits_multiplier,
			display_credits_label = EXCLUDED.display_credits_label,
			environment = EXCLUDED.environment,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id
	`
	if err := db.QueryRow(insertProduct,
		bundle.BundleKey,
		bundle.Name,
		bundle.StripeProductId,
		bundle.CreditsPerUsd,
		bundle.DisplayCreditsMultiplier,
		bundle.DisplayCreditsLabel,
		bundle.Environment,
		bundleMetadata,
	).Scan(&productID); err != nil {
		return fmt.Errorf("seed bundle product: %w", err)
	}

	plans := append([]*PlanOption{}, pricing.Monthly...)
	plans = append(plans, pricing.Yearly...)
	for _, option := range plans {
		priceID := strings.TrimSpace(option.StripePriceId)
		if priceID == "" {
			continue
		}

		planTier := strings.TrimSpace(option.PlanTier)
		allowedPlanTiers := map[string]bool{
			"solo":     true,
			"pro":      true,
			"studio":   true,
			"business": true,
			"credits":  true,
			"donation": true,
		}
		if !allowedPlanTiers[strings.ToLower(planTier)] {
			logStructured("plan_tier_skipped_for_seed", map[string]interface{}{
				"plan_name": option.PlanName,
				"plan_tier": option.PlanTier,
			})
			continue
		}

		planMetadataJSON, err := json.Marshal(jsonValueToMap(option.Metadata))
		if err != nil {
			return fmt.Errorf("marshal plan metadata %s: %w", option.PlanName, err)
		}

		displayEnabled := option.DisplayEnabled
		if !displayEnabled {
			displayEnabled = true
		}

		insertPrice := `
			INSERT INTO bundle_prices (
				product_id, stripe_price_id, plan_name, plan_tier, billing_interval, amount_cents, currency,
				intro_enabled, intro_type, intro_amount_cents, intro_periods, intro_price_lookup_key,
				monthly_included_credits, one_time_bonus_credits, plan_rank, bonus_type, kind,
				is_variable_amount, display_enabled, metadata, display_weight
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,
				$8,$9,$10,$11,$12,
				$13,$14,$15,$16,$17,
				$18,$19,$20::jsonb,$21
			)
			ON CONFLICT (stripe_price_id)
			DO UPDATE SET plan_name = EXCLUDED.plan_name,
				plan_tier = EXCLUDED.plan_tier,
				billing_interval = EXCLUDED.billing_interval,
				amount_cents = EXCLUDED.amount_cents,
				currency = EXCLUDED.currency,
				intro_enabled = EXCLUDED.intro_enabled,
				intro_type = EXCLUDED.intro_type,
				intro_amount_cents = EXCLUDED.intro_amount_cents,
				intro_periods = EXCLUDED.intro_periods,
				intro_price_lookup_key = EXCLUDED.intro_price_lookup_key,
				monthly_included_credits = EXCLUDED.monthly_included_credits,
				one_time_bonus_credits = EXCLUDED.one_time_bonus_credits,
				plan_rank = EXCLUDED.plan_rank,
				bonus_type = EXCLUDED.bonus_type,
				kind = EXCLUDED.kind,
				is_variable_amount = EXCLUDED.is_variable_amount,
				display_enabled = EXCLUDED.display_enabled,
				metadata = EXCLUDED.metadata,
				display_weight = EXCLUDED.display_weight,
				updated_at = NOW()
		`

		var introAmount interface{}
		if option.IntroAmountCents != nil {
			introAmount = *option.IntroAmountCents
		}

		intervalLabel := billingIntervalLabel(option.BillingInterval)
		if intervalLabel == "unspecified" {
			intervalLabel = "month"
		}

		if _, err := db.Exec(insertPrice,
			productID,
			priceID,
			option.PlanName,
			planTier,
			intervalLabel,
			option.AmountCents,
			option.Currency,
			option.IntroEnabled,
			option.IntroType,
			introAmount,
			option.IntroPeriods,
			option.IntroPriceLookupKey,
			option.MonthlyIncludedCredits,
			option.OneTimeBonusCredits,
			option.PlanRank,
			option.BonusType,
			planKindString(option.Kind),
			option.IsVariableAmount,
			displayEnabled,
			planMetadataJSON,
			option.DisplayWeight,
		); err != nil {
			return fmt.Errorf("seed bundle price %s: %w", option.PlanName, err)
		}
	}

	return nil
}

func seedDownloadDefaults(db *sql.DB, downloads []DownloadApp) error {
	if len(downloads) == 0 {
		return nil
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM download_apps`).Scan(&count); err != nil {
		return fmt.Errorf("count download apps: %w", err)
	}
	if count > 0 {
		return nil
	}

	for idx, app := range downloads {
		bundleKey := strings.TrimSpace(app.BundleKey)
		appKey := strings.TrimSpace(app.AppKey)
		if bundleKey == "" {
			bundleKey = "business_suite"
		}
		if appKey == "" {
			appKey = fmt.Sprintf("bundle_app_%d", idx+1)
		}

		installSteps, err := json.Marshal(app.InstallSteps)
		if err != nil {
			return fmt.Errorf("marshal install steps for %s: %w", appKey, err)
		}
		storefronts, err := json.Marshal(app.Storefronts)
		if err != nil {
			return fmt.Errorf("marshal storefronts for %s: %w", appKey, err)
		}
		metadata, err := json.Marshal(app.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata for %s: %w", appKey, err)
		}

		displayOrder := app.DisplayOrder
		if displayOrder == 0 {
			displayOrder = idx + 1
		}

		if _, err := db.Exec(`
			INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, install_overview, install_steps, storefronts, metadata, display_order)
			VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10)
		`,
			bundleKey,
			appKey,
			app.Name,
			app.Tagline,
			app.Description,
			app.InstallOverview,
			installSteps,
			storefronts,
			metadata,
			displayOrder,
		); err != nil {
			return fmt.Errorf("seed download app %s: %w", appKey, err)
		}

		for _, asset := range app.Platforms {
			platform := strings.TrimSpace(asset.Platform)
			if platform == "" {
				continue
			}
			assetMeta, err := json.Marshal(asset.Metadata)
			if err != nil {
				return fmt.Errorf("marshal asset metadata %s:%s: %w", appKey, platform, err)
			}

			assetBundle := asset.BundleKey
			if strings.TrimSpace(assetBundle) == "" {
				assetBundle = bundleKey
			}
			assetAppKey := asset.AppKey
			if strings.TrimSpace(assetAppKey) == "" {
				assetAppKey = appKey
			}

			if _, err := db.Exec(`
				INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb)
				ON CONFLICT (bundle_key, app_key, platform)
				DO UPDATE SET artifact_url = EXCLUDED.artifact_url,
					release_version = EXCLUDED.release_version,
					release_notes = EXCLUDED.release_notes,
					checksum = EXCLUDED.checksum,
					requires_entitlement = EXCLUDED.requires_entitlement,
					metadata = EXCLUDED.metadata,
					updated_at = NOW()
			`,
				assetBundle,
				assetAppKey,
				platform,
				asset.ArtifactURL,
				asset.ReleaseVersion,
				asset.ReleaseNotes,
				asset.Checksum,
				asset.RequiresEntitlement,
				assetMeta,
			); err != nil {
				return fmt.Errorf("seed download asset %s:%s: %w", appKey, platform, err)
			}
		}
	}

	return nil
}

func valueOrDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func upsertVariant(db *sql.DB, slug, name, description string, weight int) (int, error) {
	var id int
	err := db.QueryRow(`
		INSERT INTO variants (slug, name, description, weight, status)
		VALUES ($1, $2, $3, $4, 'active')
		ON CONFLICT (slug)
		DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, weight = EXCLUDED.weight, status = 'active', updated_at = NOW()
		RETURNING id
	`, slug, name, description, weight).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("failed to upsert variant %s: %w", slug, err)
	}
	return id, nil
}

func ensureVariantAxesDefaults(db *sql.DB, space *VariantSpace) error {
	if space == nil {
		space = defaultVariantSpace
	}

	for axisID, axis := range space.Axes {
		if axis == nil || len(axis.Variants) == 0 {
			continue
		}
		defaultValue := axis.Variants[0].ID
		if strings.TrimSpace(defaultValue) == "" {
			continue
		}
		if _, err := db.Exec(`
			INSERT INTO variant_axes (variant_id, axis_id, variant_value, created_at, updated_at)
			SELECT id, $1, $2, NOW(), NOW()
			FROM variants
			WHERE NOT EXISTS (
				SELECT 1 FROM variant_axes WHERE variant_axes.variant_id = variants.id AND axis_id = $3
			)
		`, axisID, defaultValue, axisID); err != nil {
			return fmt.Errorf("failed to seed default axis %s: %w", axisID, err)
		}
	}

	return nil
}

func upsertVariantAxesBySlug(db *sql.DB, slug string, axes map[string]string) error {
	if len(axes) == 0 {
		return nil
	}
	var variantID int
	if err := db.QueryRow(`SELECT id FROM variants WHERE slug = $1`, slug).Scan(&variantID); err != nil {
		return fmt.Errorf("variant %s not found for axes assignment: %w", slug, err)
	}

	for axisID, value := range axes {
		if _, err := db.Exec(`
			INSERT INTO variant_axes (variant_id, axis_id, variant_value, created_at, updated_at)
			VALUES ($1, $2, $3, NOW(), NOW())
			ON CONFLICT (variant_id, axis_id)
			DO UPDATE SET variant_value = EXCLUDED.variant_value, updated_at = NOW()
		`, variantID, axisID, value); err != nil {
			return fmt.Errorf("failed to upsert axis %s for %s: %w", axisID, slug, err)
		}
	}

	return nil
}

// syncVariantSnapshots loads variant snapshot JSON files and ensures the database matches them.
// This keeps landing variants in sync with the checked-in source of truth when the service starts.
func syncVariantSnapshots(vs *VariantService, cs *ContentService) error {
	if vs == nil || cs == nil {
		return fmt.Errorf("variant or content service missing")
	}

	dir := strings.TrimSpace(os.Getenv("VARIANT_SNAPSHOT_DIR"))
	if dir == "" {
		candidates := []string{
			filepath.Join("..", ".vrooli", "variants"),
			filepath.Join(".", ".vrooli", "variants"),
		}
		for _, candidate := range candidates {
			if info, err := os.Stat(candidate); err == nil && info.IsDir() {
				dir = candidate
				break
			}
		}
	}

	if dir == "" {
		logStructured("variant_snapshot_sync_skipped", map[string]interface{}{
			"reason": "no snapshot directory found",
		})
		return nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logStructured("variant_snapshot_sync_skipped", map[string]interface{}{
				"reason": "snapshot directory missing",
				"path":   dir,
			})
			return nil
		}
		return fmt.Errorf("read snapshot directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read snapshot %s: %w", path, err)
		}

		var snapshot VariantSnapshotInput
		if err := json.Unmarshal(raw, &snapshot); err != nil {
			return fmt.Errorf("parse snapshot %s: %w", path, err)
		}

		slug := strings.TrimSpace(snapshot.Variant.Slug)
		if slug == "" {
			logStructuredError("variant_snapshot_missing_slug", map[string]interface{}{
				"file": path,
			})
			continue
		}

		if snapshot.Variant.HeaderConfig == nil {
			cfg := defaultLandingHeaderConfig(valueOrDefault(snapshot.Variant.Name, slug))
			snapshot.Variant.HeaderConfig = &cfg
		}

		if len(snapshot.Variant.Axes) == 0 {
			return fmt.Errorf("snapshot %s missing axes for variant %s", path, slug)
		}

		if _, err := vs.GetVariantBySlug(slug); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "not found") {
				name := valueOrDefault(snapshot.Variant.Name, slug)
				desc := snapshot.Variant.Description
				if strings.TrimSpace(desc) == "" {
					desc = "Imported from snapshot"
				}
				weight := snapshot.Variant.Weight
				if weight < 0 || weight > 100 {
					weight = 50
				}
				if _, err := vs.CreateVariant(slug, name, desc, weight, snapshot.Variant.Axes); err != nil {
					return fmt.Errorf("create variant %s: %w", slug, err)
				}
			} else {
				return fmt.Errorf("lookup variant %s: %w", slug, err)
			}
		}

		if _, err := vs.ImportVariantSnapshot(slug, snapshot, cs); err != nil {
			return fmt.Errorf("import variant %s: %w", slug, err)
		}

		logStructured("variant_snapshot_synced", map[string]interface{}{
			"slug": slug,
			"file": path,
		})
	}

	return nil
}

// ensureSchema creates required tables if they do not exist (runtime guard when psql is unavailable)
func ensureSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admin_users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			last_login TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);`,
		`CREATE TABLE IF NOT EXISTS variants (
			id SERIAL PRIMARY KEY,
			slug VARCHAR(100) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			weight INTEGER DEFAULT 50 CHECK (weight >= 0 AND weight <= 100),
			status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
			header_config JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			archived_at TIMESTAMP
		);`,
		`ALTER TABLE variants ADD COLUMN IF NOT EXISTS header_config JSONB DEFAULT '{}'::jsonb;`,
		`CREATE INDEX IF NOT EXISTS idx_variants_slug ON variants(slug);`,
		`CREATE INDEX IF NOT EXISTS idx_variants_status ON variants(status);`,
		`CREATE TABLE IF NOT EXISTS variant_axes (
			variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
			axis_id VARCHAR(100) NOT NULL,
			variant_value VARCHAR(100) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			PRIMARY KEY (variant_id, axis_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_variant_axes_axis ON variant_axes(axis_id);`,
		`CREATE TABLE IF NOT EXISTS metrics_events (
			id SERIAL PRIMARY KEY,
			variant_id INTEGER REFERENCES variants(id),
			event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download')),
			event_data JSONB,
			session_id VARCHAR(255),
			visitor_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_events_type ON metrics_events(event_type);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_events_created ON metrics_events(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_events_session ON metrics_events(session_id);`,
		`ALTER TABLE metrics_events DROP CONSTRAINT IF EXISTS metrics_events_event_type_check;`,
		`ALTER TABLE metrics_events ADD CONSTRAINT metrics_events_event_type_check CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download'));`,
		`CREATE TABLE IF NOT EXISTS checkout_sessions (
			id SERIAL PRIMARY KEY,
			session_id VARCHAR(255) UNIQUE NOT NULL,
			customer_email VARCHAR(255),
			price_id VARCHAR(255),
			subscription_id VARCHAR(255),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_checkout_sessions_session_id ON checkout_sessions(session_id);`,
		`CREATE INDEX IF NOT EXISTS idx_checkout_sessions_status ON checkout_sessions(status);`,
		`ALTER TABLE checkout_sessions ADD COLUMN IF NOT EXISTS session_type VARCHAR(50) DEFAULT 'subscription';`,
		`ALTER TABLE checkout_sessions ADD COLUMN IF NOT EXISTS amount_cents INTEGER;`,
		`ALTER TABLE checkout_sessions ADD COLUMN IF NOT EXISTS schedule_id VARCHAR(255);`,
		`ALTER TABLE checkout_sessions ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;`,
		`ALTER TABLE checkout_sessions ADD COLUMN IF NOT EXISTS customer_id VARCHAR(255);`,
		`CREATE INDEX IF NOT EXISTS idx_checkout_sessions_type ON checkout_sessions(session_type);`,
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id SERIAL PRIMARY KEY,
			subscription_id VARCHAR(255) UNIQUE NOT NULL,
			customer_id VARCHAR(255),
			customer_email VARCHAR(255),
			status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'trialing', 'past_due', 'canceled', 'unpaid')),
			canceled_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_subscription_id ON subscriptions(subscription_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_email ON subscriptions(customer_email);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id ON subscriptions(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS plan_tier VARCHAR(50);`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS price_id VARCHAR(255);`,
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS bundle_key VARCHAR(100);`,
		`CREATE TABLE IF NOT EXISTS subscription_schedules (
			id SERIAL PRIMARY KEY,
			schedule_id VARCHAR(255) UNIQUE NOT NULL,
			subscription_id VARCHAR(255),
			price_id VARCHAR(255) NOT NULL,
			billing_interval VARCHAR(20) NOT NULL CHECK (billing_interval IN ('month','year','one_time')),
			intro_enabled BOOLEAN DEFAULT FALSE,
			intro_amount_cents INTEGER,
			intro_periods INTEGER DEFAULT 0,
			normal_amount_cents INTEGER,
			next_billing_at TIMESTAMP,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_schedules_schedule_id ON subscription_schedules(schedule_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_schedules_subscription_id ON subscription_schedules(subscription_id);`,
		`CREATE TABLE IF NOT EXISTS content_sections (
			id SERIAL PRIMARY KEY,
			variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
			section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video', 'downloads')),
			content JSONB NOT NULL,
			"order" INTEGER DEFAULT 0,
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_variant ON content_sections(variant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_type ON content_sections(section_type);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_order ON content_sections("order");`,
		`ALTER TABLE content_sections DROP CONSTRAINT IF EXISTS content_sections_section_type_check;`,
		`ALTER TABLE content_sections ADD CONSTRAINT content_sections_section_type_check CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video', 'downloads'));`,
		`CREATE TABLE IF NOT EXISTS bundle_products (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) UNIQUE NOT NULL,
			bundle_name VARCHAR(255) NOT NULL,
			stripe_product_id VARCHAR(255) UNIQUE NOT NULL,
			credits_per_usd BIGINT NOT NULL,
			display_credits_multiplier NUMERIC(12,6) DEFAULT 1.0,
			display_credits_label VARCHAR(50) DEFAULT 'credits',
			environment VARCHAR(50) DEFAULT 'production',
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_bundle_products_env ON bundle_products(environment);`,
		`CREATE TABLE IF NOT EXISTS bundle_prices (
			id SERIAL PRIMARY KEY,
			product_id INTEGER REFERENCES bundle_products(id) ON DELETE CASCADE,
			stripe_price_id VARCHAR(255) UNIQUE,
			plan_name VARCHAR(100) NOT NULL,
			plan_tier VARCHAR(50) NOT NULL CHECK (plan_tier IN ('free','solo','pro','studio','business','credits','donation')),
			billing_interval VARCHAR(20) NOT NULL CHECK (billing_interval IN ('month','year','one_time')),
			amount_cents INTEGER NOT NULL,
			currency VARCHAR(10) DEFAULT 'usd',
			intro_enabled BOOLEAN DEFAULT FALSE,
			intro_type VARCHAR(50),
			intro_amount_cents INTEGER,
			intro_periods INTEGER DEFAULT 0,
			intro_price_lookup_key VARCHAR(255),
			monthly_included_credits INTEGER DEFAULT 0,
			one_time_bonus_credits INTEGER DEFAULT 0,
			plan_rank INTEGER DEFAULT 0,
			bonus_type VARCHAR(50),
			kind VARCHAR(50) DEFAULT 'subscription',
			is_variable_amount BOOLEAN DEFAULT FALSE,
			display_enabled BOOLEAN DEFAULT TRUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			display_weight INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`ALTER TABLE bundle_prices ALTER COLUMN stripe_price_id DROP NOT NULL;`,
		`ALTER TABLE bundle_prices DROP CONSTRAINT IF EXISTS bundle_prices_plan_tier_check;`,
		`ALTER TABLE bundle_prices ADD CONSTRAINT bundle_prices_plan_tier_check CHECK (plan_tier IN ('free','solo','pro','studio','business','credits','donation'));`,
		`ALTER TABLE bundle_prices ADD COLUMN IF NOT EXISTS display_enabled BOOLEAN DEFAULT TRUE;`,
		`CREATE INDEX IF NOT EXISTS idx_bundle_prices_tier ON bundle_prices(plan_tier);`,
		`CREATE INDEX IF NOT EXISTS idx_bundle_prices_interval ON bundle_prices(billing_interval);`,
		`CREATE TABLE IF NOT EXISTS download_apps (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			app_key VARCHAR(100) NOT NULL,
			name VARCHAR(255) NOT NULL,
			tagline TEXT,
			description TEXT,
			install_overview TEXT,
			install_steps JSONB DEFAULT '[]'::jsonb,
			storefronts JSONB DEFAULT '[]'::jsonb,
			metadata JSONB DEFAULT '{}'::jsonb,
			display_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE (bundle_key, app_key)
		);`,
		`CREATE TABLE IF NOT EXISTS download_assets (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			app_key VARCHAR(100) NOT NULL,
			platform VARCHAR(50) NOT NULL CHECK (platform IN ('windows','mac','linux')),
			artifact_url TEXT NOT NULL,
			release_version VARCHAR(50) NOT NULL,
			release_notes TEXT,
			checksum VARCHAR(255),
			requires_entitlement BOOLEAN DEFAULT FALSE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key)
				REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_download_apps_bundle ON download_apps(bundle_key);`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS app_key VARCHAR(100);`,
		`UPDATE download_assets SET app_key = bundle_key WHERE app_key IS NULL OR app_key = '';`,
		`ALTER TABLE download_assets ALTER COLUMN app_key SET NOT NULL;`,
		`ALTER TABLE download_assets DROP CONSTRAINT IF EXISTS fk_download_app;`,
		`ALTER TABLE download_assets ADD CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key)
			REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE;`,
		`DROP INDEX IF EXISTS idx_download_assets_bundle_platform;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_app_platform ON download_assets(bundle_key, app_key, platform);`,
		`INSERT INTO download_apps (bundle_key, app_key, name, display_order)
			SELECT DISTINCT bundle_key, app_key, CONCAT(bundle_key, ' downloads'), 0
			FROM download_assets
			ON CONFLICT (bundle_key, app_key) DO NOTHING;`,
		`CREATE TABLE IF NOT EXISTS credit_wallets (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) UNIQUE NOT NULL,
			balance_credits BIGINT DEFAULT 0,
			bonus_credits BIGINT DEFAULT 0,
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS credit_transactions (
			id SERIAL PRIMARY KEY,
			customer_email VARCHAR(255) NOT NULL,
			amount_credits BIGINT NOT NULL,
			transaction_type VARCHAR(50) NOT NULL,
			source VARCHAR(100),
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS payment_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			publishable_key TEXT,
			secret_key TEXT,
			webhook_secret TEXT,
			dashboard_url TEXT,
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS site_branding (
			id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
			site_name TEXT NOT NULL DEFAULT 'My Landing',
			tagline TEXT,
			logo_url TEXT,
			logo_icon_url TEXT,
			favicon_url TEXT,
			apple_touch_icon_url TEXT,
			default_title TEXT,
			default_description TEXT,
			default_og_image_url TEXT,
			theme_primary_color TEXT,
			theme_background_color TEXT,
			canonical_base_url TEXT,
			google_site_verification TEXT,
			robots_txt TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS site_branding_singleton ON site_branding ((1));`,
		`CREATE TABLE IF NOT EXISTS assets (
			id SERIAL PRIMARY KEY,
			filename TEXT NOT NULL,
			original_filename TEXT NOT NULL,
			mime_type TEXT NOT NULL,
			size_bytes BIGINT NOT NULL,
			storage_path TEXT NOT NULL,
			thumbnail_path TEXT,
			alt_text TEXT,
			category TEXT DEFAULT 'general' CHECK (category IN ('logo', 'favicon', 'og_image', 'general')),
			uploaded_by TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_assets_category ON assets(category);`,
		`CREATE INDEX IF NOT EXISTS idx_assets_created ON assets(created_at);`,
		`ALTER TABLE variants ADD COLUMN IF NOT EXISTS seo_config JSONB DEFAULT '{}'::jsonb;`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) handleCustomize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScenarioID string   `json:"scenario_id"`
		Brief      string   `json:"brief"`
		Assets     []string `json:"assets"`
		Preview    bool     `json:"preview"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Stub implementation
	response := map[string]interface{}{
		"job_id":   fmt.Sprintf("job-%d", time.Now().Unix()),
		"status":   "queued",
		"agent_id": "agent-claude-code-1",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(response)
}

// loggingMiddleware prints structured request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fields := map[string]interface{}{
			"method":   r.Method,
			"path":     r.RequestURI,
			"duration": time.Since(start).String(),
		}
		logStructured("request_completed", fields)
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	logStructured(msg, fields)
}

func logStructured(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Printf(`{"level":"info","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"info","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

func logStructuredError(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Printf(`{"level":"error","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	log.Printf(`{"level":"error","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start landing-manager

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
