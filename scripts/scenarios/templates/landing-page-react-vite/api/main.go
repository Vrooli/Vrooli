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

	srv := &Server{
		config:               cfg,
		db:                   db,
		router:               mux.NewRouter(),
		variantSpace:         variantSpace,
		variantService:       variantService,
		metricsService:       NewMetricsService(db),
		stripeService:        NewStripeService(db),
		contentService:       contentService,
		planService:          planService,
		downloadService:      downloadService,
		downloadAuthorizer:   downloadAuthorizer,
		accountService:       accountService,
		landingConfigService: NewLandingConfigService(variantService, contentService, planService, downloadService),
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
	s.router.HandleFunc("/api/v1/billing/portal-url", handleBillingPortalURL()).Methods("GET")

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
		"service": "{{SCENARIO_ID}}-api",
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
		"service":   "{{SCENARIO_DISPLAY_NAME}} API",
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

// seedDefaultData ensures the public landing page has content without requiring admin setup
func seedDefaultData(db *sql.DB) error {
	if err := ensureSchema(db); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	// Seed default admin user for local use
	const defaultAdminEmail = "admin@localhost"
	const defaultAdminHash = "$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG"

	if _, err := db.Exec(
		`INSERT INTO admin_users (email, password_hash) VALUES ($1, $2)
		 ON CONFLICT (email) DO UPDATE SET password_hash = EXCLUDED.password_hash`,
		defaultAdminEmail,
		defaultAdminHash,
	); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
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
		hero := `{"headline": "Build Landing Pages in Minutes", "subheadline": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in", "cta_text": "Get Started Free", "cta_url": "/signup", "background_color": "#0f172a"}`
		features := `{"title": "Everything You Need", "items": [{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "Zap"}, {"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "BarChart"}, {"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "CreditCard"}]}`
		pricing := `{"title": "Simple Pricing", "plans": [{"name": "Starter", "price": "$29", "features": ["5 landing pages", "Basic analytics", "Email support"], "cta_text": "Start Free Trial"}, {"name": "Pro", "price": "$99", "features": ["Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"], "cta_text": "Get Started", "highlighted": true}]}`
		cta := `{"headline": "Ready to Launch Your Landing Page?", "subheadline": "Join thousands of marketers using Landing Manager", "cta_text": "Start Building Now", "cta_url": "/signup"}`

		if _, err := db.Exec(`
			INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
			($1, 'hero', $2::jsonb, 1, TRUE),
			($1, 'features', $3::jsonb, 2, TRUE),
			($1, 'pricing', $4::jsonb, 3, TRUE),
			($1, 'cta', $5::jsonb, 4, TRUE)
		`, controlID, hero, features, pricing, cta); err != nil {
			return fmt.Errorf("failed to seed default sections: %w", err)
		}
	}

	if err := ensureVariantAxesDefaults(db, defaultVariantSpace); err != nil {
		return fmt.Errorf("failed to seed variant axes defaults: %w", err)
	}

	return nil
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
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			archived_at TIMESTAMP
		);`,
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
			section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video')),
			content JSONB NOT NULL,
			"order" INTEGER DEFAULT 0,
			enabled BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_variant ON content_sections(variant_id);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_type ON content_sections(section_type);`,
		`CREATE INDEX IF NOT EXISTS idx_content_sections_order ON content_sections("order");`,
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
			stripe_price_id VARCHAR(255) UNIQUE NOT NULL,
			plan_name VARCHAR(100) NOT NULL,
			plan_tier VARCHAR(50) NOT NULL CHECK (plan_tier IN ('solo','pro','studio','business','credits','donation')),
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
			metadata JSONB DEFAULT '{}'::jsonb,
			display_weight INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_bundle_prices_tier ON bundle_prices(plan_tier);`,
		`CREATE INDEX IF NOT EXISTS idx_bundle_prices_interval ON bundle_prices(billing_interval);`,
		`CREATE TABLE IF NOT EXISTS download_assets (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			platform VARCHAR(50) NOT NULL CHECK (platform IN ('windows','mac','linux')),
			artifact_url TEXT NOT NULL,
			release_version VARCHAR(50) NOT NULL,
			release_notes TEXT,
			checksum VARCHAR(255),
			requires_entitlement BOOLEAN DEFAULT TRUE,
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_platform ON download_assets(bundle_key, platform);`,
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
