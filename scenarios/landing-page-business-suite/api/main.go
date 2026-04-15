// DOC: docs/concepts/ARCHITECTURE.md - System design and component overview
// DOC: docs/QUICKSTART.md - Getting started guide
// DOC: PRD.md - Product requirements and operational targets
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
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
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
	configStore          *ConfigStore
	metricsService       *MetricsService
	stripeService        *StripeService
	planService          *PlanService
	downloadService      *DownloadService
	downloadHosting      *DownloadHostingService
	downloadAuthorizer   *DownloadAuthorizer
	accountService       *AccountService
	landingConfigService *LandingConfigService
	paymentSettings      *PaymentSettingsService
	assetsService        *AssetsService
	seoService           *SEOService
	feedbackService      *FeedbackService
	emailService         *EmailService
	waitlistService      *WaitlistService
	// Credit system services
	apiKeyService *APIKeyService
	limitsService *LimitsService
	usageService  *UsageService
	// Remote profile service (admin-managed remote connections)
	remoteProfileService *RemoteProfileService
	// User authentication services
	userAuthService  *UserAuthService
	magicLinkLimiter *RateLimiter
	// AI Gateway service
	aiGatewayService *AIGatewayService
	aiGatewayDeps    *AIGatewayDeps
	// Session management for admin auth
	sessionManager SessionManager
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := seedDefaultData(db); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}

	// Initialize config store from JSON files (source of truth for variants and branding)
	variantsDir := resolveVariantsDir()
	brandingPath := resolveBrandingPath()
	configStore := NewConfigStore(variantsDir, brandingPath, defaultVariantSpace)
	if err := configStore.LoadAll(); err != nil {
		return nil, fmt.Errorf("failed to load config from JSON files: %w", err)
	}

	variantSpace := defaultVariantSpace
	planService := NewPlanService(db)
	downloadService := NewDownloadService(db)
	downloadHosting := NewDownloadHostingService(db, S3DownloadStorageProvider{})
	accountService := NewAccountService(db, planService)
	downloadAuthorizer := NewDownloadAuthorizer(downloadService, accountService, planService.BundleKey())
	paymentSettings := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	assetsService := NewAssetsService(db)
	seoService := NewSEOServiceWithConfigStore(configStore)
	feedbackService := NewFeedbackService(db)
	emailService := NewEmailService()
	waitlistService := NewWaitlistService(db)

	// Initialize credit system services
	apiKeyService, err := NewAPIKeyService(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API key service: %w", err)
	}
	remoteProfileService, err := NewRemoteProfileService(db)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote profile service: %w", err)
	}
	limitsService := NewLimitsService(db, "postgres")
	usageService := NewUsageService(db, limitsService, "postgres")

	// Initialize user authentication services
	userAuthService := NewUserAuthService(db, emailService)
	// Rate limiter: 5 requests per 15 minutes per email for magic link
	magicLinkLimiter := NewRateLimiter(5, 15*time.Minute)

	// Initialize AI gateway service
	aiGatewayService := NewAIGatewayService(AIGatewayServiceOptions{
		DB:             db,
		APIKeyService:  apiKeyService,
		UsageService:   usageService,
		AccountService: accountService,
		LimitsService:  limitsService,
		Logger:         logStructured,
	})

	// Create AI gateway dependencies with rate limiters
	aiGatewayDeps := DefaultAIGatewayDeps(aiGatewayService, usageService, accountService)

	srv := &Server{
		config:               &Config{},
		db:                   db,
		router:               mux.NewRouter(),
		variantSpace:         variantSpace,
		configStore:          configStore,
		metricsService:       NewMetricsService(db),
		stripeService:        stripeService,
		planService:          planService,
		downloadService:      downloadService,
		downloadHosting:      downloadHosting,
		downloadAuthorizer:   downloadAuthorizer,
		accountService:       accountService,
		landingConfigService: NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService, stripeService),
		paymentSettings:      paymentSettings,
		assetsService:        assetsService,
		seoService:           seoService,
		feedbackService:      feedbackService,
		emailService:         emailService,
		waitlistService:      waitlistService,
		// Credit system services
		apiKeyService: apiKeyService,
		limitsService: limitsService,
		usageService:  usageService,
		// Remote profile service
		remoteProfileService: remoteProfileService,
		// User authentication services
		userAuthService:  userAuthService,
		magicLinkLimiter: magicLinkLimiter,
		// AI Gateway service
		aiGatewayService: aiGatewayService,
		aiGatewayDeps:    aiGatewayDeps,
		// Session management
		sessionManager: initSessionManager(),
	}

	srv.setupRoutes()
	return srv, nil
}

// resolveVariantsDir finds the variants directory
func resolveVariantsDir() string {
	dir := strings.TrimSpace(os.Getenv("VARIANT_SNAPSHOT_DIR"))
	if dir != "" {
		return dir
	}
	candidates := []string{
		filepath.Join("..", ".vrooli", "variants"),
		filepath.Join(".", ".vrooli", "variants"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("..", ".vrooli", "variants")
}

// resolveBrandingPath finds the branding.json file
func resolveBrandingPath() string {
	candidates := []string{
		filepath.Join("..", ".vrooli", "branding.json"),
		filepath.Join(".", ".vrooli", "branding.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", ".vrooli", "branding.json")
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Server) handleAdminResetDemoData(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) resetDemoData(ctx context.Context) error {
	// Only reset database tables for runtime data (not config, which is in JSON files)
	// NOTE: bundle_prices and bundle_products removed - pricing now stored in .vrooli/plans.json
	tables := []string{
		"download_assets",
		"download_apps",
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range tables {
		stmt := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if err := seedDefaultData(s.db); err != nil {
		return err
	}

	// Reload config from JSON files
	if err := s.configStore.LoadAll(); err != nil {
		return fmt.Errorf("reload config: %w", err)
	}

	return nil
}

// seedDefaultData sets up baseline records that are not variant-specific.
func seedDefaultData(db *sql.DB) error {
	if err := ensureSchema(db); err != nil {
		return fmt.Errorf("failed to apply schema: %w", err)
	}

	// Seed admin user, using secrets from env vars or ~/.vrooli/secrets.json (scenario-to-cloud Secrets Tab)
	// Log secrets resolution source for debugging
	if secretsFile := findSecretsFile(); secretsFile != "" {
		logStructured("secrets_file_found", map[string]interface{}{
			"level": "info",
			"path":  secretsFile,
		})
	}

	adminEmail, adminPasswordHash, err := getAdminDefaults()
	if err != nil {
		return fmt.Errorf("failed to get admin defaults: %w", err)
	}

	if _, err := db.Exec(
		`DELETE FROM admin_users WHERE LOWER(email) = LOWER($1) AND id <> $2`,
		adminEmail,
		seededAdminID,
	); err != nil {
		return fmt.Errorf("failed to cleanup admin duplicates: %w", err)
	}

	// Upsert the seeded admin at reserved ID. This ensures credential changes work correctly:
	// - Default → Custom: updates email and password at id=1
	// - Custom → Different Custom: updates email and password at id=1
	// - Custom → Default: updates email and password at id=1
	// No orphan accounts are created regardless of email changes.
	if _, err := db.Exec(
		`INSERT INTO admin_users (id, email, password_hash) VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email, password_hash = EXCLUDED.password_hash`,
		seededAdminID,
		adminEmail,
		adminPasswordHash,
	); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}

	// Log if custom credentials were configured
	if adminEmail != defaultAdminEmail {
		logStructured("admin_user_seeded_custom", map[string]interface{}{
			"level": "info",
			"email": adminEmail,
		})
	}

	if _, err := db.Exec(`
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, dashboard_url, updated_at)
		VALUES (1, NULL, NULL, NULL, NULL, NOW())
		ON CONFLICT (id) DO NOTHING
	`); err != nil {
		return fmt.Errorf("failed to seed payment settings: %w", err)
	}

	// NOTE: Site branding is now stored in JSON file (.vrooli/branding.json)
	// and loaded into memory at startup via ConfigStore.

	// NOTE: Bundle pricing is now stored in JSON file (.vrooli/plans.json)
	// and loaded into memory at startup via PlanStore. Database seeding removed.

	if err := seedDownloadDefaults(db, fallbackLanding.Downloads); err != nil {
		return err
	}

	if err := seedTierLimitsDefaults(db); err != nil {
		return err
	}

	return nil
}

// NOTE: seedBundlePricingDefaults function removed - pricing now stored in .vrooli/plans.json

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
				INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata, variant_key)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,'default')
				ON CONFLICT (bundle_key, app_key, platform, variant_key)
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

// seedTierLimitsDefaults seeds default subscription tier limits for the cost-based credit system.
// These values define AI credit limits per subscription tier.
// Cost multiplier is 1,000,000 (so $5 = 500,000,000 internal units)
func seedTierLimitsDefaults(db *sql.DB) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM subscription_tier_limits`).Scan(&count); err != nil {
		return fmt.Errorf("count tier limits: %w", err)
	}
	if count > 0 {
		return nil // Already seeded
	}

	// Default tier limits for cost-based AI credits
	// -1 = unlimited, values are in internal units (cents x 1,000,000)
	tierLimits := []struct {
		tierID       string
		limitType    string
		limitKey     string
		limitValue   int64 // Internal units
		appBundleKey *string
	}{
		// Cost-based AI credits per tier (shared across all apps)
		// free: 0 (no AI access)
		{"free", "cost_based", "ai_credits", 0, nil},
		// solo: $5/month worth of AI = 500,000,000 internal units
		{"solo", "cost_based", "ai_credits", 500000000, nil},
		// pro: $20/month worth of AI = 2,000,000,000 internal units
		{"pro", "cost_based", "ai_credits", 2000000000, nil},
		// studio: $100/month worth of AI = 10,000,000,000 internal units
		{"studio", "cost_based", "ai_credits", 10000000000, nil},
		// business: unlimited
		{"business", "cost_based", "ai_credits", -1, nil},
	}

	insertQuery := `
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, app_bundle_key)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tier_id, limit_type, limit_key, app_bundle_key) DO NOTHING
	`

	for _, tl := range tierLimits {
		if _, err := db.Exec(insertQuery, tl.tierID, tl.limitType, tl.limitKey, tl.limitValue, tl.appBundleKey); err != nil {
			return fmt.Errorf("seed tier limit %s/%s: %w", tl.tierID, tl.limitKey, err)
		}
	}

	logStructured("tier_limits_seeded", map[string]interface{}{
		"level": "info",
		"count": len(tierLimits),
	})

	return nil
}

func valueOrDefault(value, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

// NOTE: syncVariantSnapshots function has been removed.
// Variant configuration is now loaded directly from JSON files via ConfigStore.LoadAll()
// which is called in NewServer(). No database sync is needed.

// ensureSchema creates required tables if they do not exist (runtime guard when psql is unavailable)
// NOTE: Variant, section, and branding configuration is now stored in JSON files
// (.vrooli/variants/*.json and .vrooli/branding.json) and loaded into memory at startup.
// This schema only contains tables for runtime/dynamic data.
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
		`CREATE TABLE IF NOT EXISTS remote_profiles (
			id SERIAL PRIMARY KEY,
			tag VARCHAR(100) UNIQUE NOT NULL,
			label TEXT,
			api_base TEXT NOT NULL,
			connector_id VARCHAR(64) UNIQUE,
			remote_session_id TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'unknown' CHECK (status IN ('unknown','active','expired','error')),
			encrypted_session TEXT,
			session_expires_at TIMESTAMP,
			remote_session_last_synced_at TIMESTAMP,
			last_login_at TIMESTAMP,
			last_used_at TIMESTAMP,
			created_by INTEGER REFERENCES admin_users(id) ON DELETE SET NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`ALTER TABLE remote_profiles ADD COLUMN IF NOT EXISTS connector_id VARCHAR(64);`,
		`ALTER TABLE remote_profiles ADD COLUMN IF NOT EXISTS remote_session_id TEXT;`,
		`ALTER TABLE remote_profiles ADD COLUMN IF NOT EXISTS remote_session_last_synced_at TIMESTAMP;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_remote_profiles_connector_id ON remote_profiles(connector_id);`,
		`CREATE INDEX IF NOT EXISTS idx_remote_profiles_tag ON remote_profiles(tag);`,
		`CREATE INDEX IF NOT EXISTS idx_remote_profiles_status ON remote_profiles(status);`,
		// Admin sessions table for server-side session tracking
		`CREATE TABLE IF NOT EXISTS admin_sessions (
			id TEXT PRIMARY KEY,
			admin_email TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			last_activity TIMESTAMP DEFAULT NOW(),
			expires_at TIMESTAMP NOT NULL,
			ip_address TEXT,
			user_agent TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_sessions_email ON admin_sessions(admin_email);`,
		`CREATE INDEX IF NOT EXISTS idx_admin_sessions_expires ON admin_sessions(expires_at);`,
		// NOTE: variants, variant_axes, and content_sections tables have been removed.
		// Variant configuration is now stored in JSON files (.vrooli/variants/*.json).
		`CREATE TABLE IF NOT EXISTS metrics_events (
			id SERIAL PRIMARY KEY,
			variant_slug VARCHAR(100),
			event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download')),
			event_data JSONB,
			session_id VARCHAR(255),
			visitor_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`ALTER TABLE metrics_events ADD COLUMN IF NOT EXISTS variant_slug VARCHAR(100);`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_slug);`,
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
		`ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS billing_cycle_start INTEGER DEFAULT 0;`,
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
		// Ensure subscription_schedules has all required columns (for schema migrations)
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS price_id VARCHAR(255);`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS billing_interval VARCHAR(20);`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS intro_enabled BOOLEAN DEFAULT FALSE;`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS intro_amount_cents INTEGER;`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS intro_periods INTEGER DEFAULT 0;`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS normal_amount_cents INTEGER;`,
		`ALTER TABLE subscription_schedules ADD COLUMN IF NOT EXISTS next_billing_at TIMESTAMP;`,
		// NOTE: content_sections table has been removed.
		// Sections are now stored in JSON files (.vrooli/variants/*.json) as part of variant snapshots.
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
			icon_url TEXT,
			screenshot_url TEXT,
			install_overview TEXT,
			install_steps JSONB DEFAULT '[]'::jsonb,
			storefronts JSONB DEFAULT '[]'::jsonb,
			metadata JSONB DEFAULT '{}'::jsonb,
			display_order INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE (bundle_key, app_key)
		);`,
		`ALTER TABLE download_apps ADD COLUMN IF NOT EXISTS icon_url TEXT;`,
		`ALTER TABLE download_apps ADD COLUMN IF NOT EXISTS screenshot_url TEXT;`,
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
		`ALTER TABLE download_assets ALTER COLUMN artifact_url DROP NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS download_artifacts (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) NOT NULL,
			provider VARCHAR(50) NOT NULL DEFAULT 's3',
			bucket TEXT NOT NULL,
			object_key TEXT NOT NULL,
			etag TEXT,
			size_bytes BIGINT,
			sha256 TEXT,
			content_type TEXT,
			original_filename TEXT,
			platform VARCHAR(50),
			release_version VARCHAR(50),
			metadata JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE (bundle_key, bucket, object_key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_download_artifacts_bundle ON download_artifacts(bundle_key);`,
		`CREATE INDEX IF NOT EXISTS idx_download_artifacts_platform ON download_artifacts(platform);`,
		`CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_version ON download_artifacts(release_version);`,
		`ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS app_key VARCHAR(100);`,
		`CREATE INDEX IF NOT EXISTS idx_download_artifacts_app_key ON download_artifacts(app_key);`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS artifact_source VARCHAR(20) NOT NULL DEFAULT 'direct';`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS artifact_id INTEGER NULL;`,
		`ALTER TABLE download_assets DROP CONSTRAINT IF EXISTS fk_download_assets_artifact;`,
		`ALTER TABLE download_assets ADD CONSTRAINT fk_download_assets_artifact FOREIGN KEY (artifact_id)
			REFERENCES download_artifacts(id) ON DELETE SET NULL;`,
		`CREATE INDEX IF NOT EXISTS idx_download_assets_artifact_id ON download_assets(artifact_id);`,
		`CREATE TABLE IF NOT EXISTS download_storage_settings (
			id SERIAL PRIMARY KEY,
			bundle_key VARCHAR(100) UNIQUE NOT NULL,
			provider VARCHAR(50) NOT NULL DEFAULT 's3',
			bucket TEXT,
			region TEXT,
			endpoint TEXT,
			force_path_style BOOLEAN DEFAULT FALSE,
			default_prefix TEXT,
			signed_url_ttl_seconds INTEGER DEFAULT 900,
			public_base_url TEXT,
			access_key_id TEXT,
			secret_access_key TEXT,
			session_token TEXT,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_download_storage_settings_bundle ON download_storage_settings(bundle_key);`,
		`CREATE INDEX IF NOT EXISTS idx_download_apps_bundle ON download_apps(bundle_key);`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS app_key VARCHAR(100);`,
		`UPDATE download_assets SET app_key = bundle_key WHERE app_key IS NULL OR app_key = '';`,
		`ALTER TABLE download_assets ALTER COLUMN app_key SET NOT NULL;`,
		`ALTER TABLE download_assets DROP CONSTRAINT IF EXISTS fk_download_app;`,
		`ALTER TABLE download_assets ADD CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key)
			REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE;`,
		`DROP INDEX IF EXISTS idx_download_assets_bundle_platform;`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS variant_key VARCHAR(50) NOT NULL DEFAULT 'default';`,
		`ALTER TABLE download_assets ADD COLUMN IF NOT EXISTS display_order INTEGER DEFAULT 0;`,
		`DROP INDEX IF EXISTS idx_download_assets_bundle_app_platform;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_app_platform_variant ON download_assets(bundle_key, app_key, platform, variant_key);`,
		`INSERT INTO download_apps (bundle_key, app_key, name, display_order)
			SELECT DISTINCT bundle_key, app_key, CONCAT(bundle_key, ' downloads'), 0
			FROM download_assets
			ON CONFLICT (bundle_key, app_key) DO NOTHING;`,
		`ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS sha512 TEXT;`,
		`ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS git_commit_hash TEXT;`,
		`ALTER TABLE download_apps ADD COLUMN IF NOT EXISTS update_api_key TEXT;`,
		`ALTER TABLE download_artifacts ADD COLUMN IF NOT EXISTS release_id TEXT;`,
		`CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_id ON download_artifacts(release_id);`,
		`ALTER TABLE download_apps ADD COLUMN IF NOT EXISTS update_policy JSONB NOT NULL DEFAULT '{"check_interval_hours": 4, "update_mode": "optional", "allow_downgrade": false}'::jsonb;`,
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
			stripe_event_id VARCHAR(255) UNIQUE,
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
		// NOTE: site_branding table has been removed.
		// Branding is now stored in JSON file (.vrooli/branding.json).
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
		// NOTE: ALTER TABLE statements for variants and site_branding have been removed
		// as those tables no longer exist (config is now in JSON files).
		`CREATE TABLE IF NOT EXISTS feedback_requests (
			id SERIAL PRIMARY KEY,
			type VARCHAR(50) NOT NULL CHECK (type IN ('refund', 'bug', 'feature', 'general')),
			email VARCHAR(255) NOT NULL,
			subject VARCHAR(500) NOT NULL,
			message TEXT NOT NULL,
			order_id VARCHAR(255),
			status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'resolved', 'rejected')),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_requests_type ON feedback_requests(type);`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_requests_status ON feedback_requests(status);`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_requests_email ON feedback_requests(email);`,
		`CREATE INDEX IF NOT EXISTS idx_feedback_requests_created ON feedback_requests(created_at);`,
		`CREATE TABLE IF NOT EXISTS waitlist_emails (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			source VARCHAR(50) DEFAULT 'coming_soon',
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_waitlist_emails_email ON waitlist_emails(email);`,
		// Cost-based credit system tables
		`CREATE TABLE IF NOT EXISTS subscription_tier_limits (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			tier_id VARCHAR(50) NOT NULL,
			limit_type VARCHAR(20) NOT NULL,
			limit_key VARCHAR(100) NOT NULL,
			limit_value BIGINT NOT NULL,
			cost_multiplier BIGINT DEFAULT 1000000,
			app_bundle_key VARCHAR(100),
			reset_period VARCHAR(20) DEFAULT 'monthly',
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(tier_id, limit_type, limit_key, app_bundle_key)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_tier ON subscription_tier_limits(tier_id);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_type ON subscription_tier_limits(limit_type);`,
		`CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_app ON subscription_tier_limits(app_bundle_key);`,
		`CREATE TABLE IF NOT EXISTS usage_records (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_identity VARCHAR(255) NOT NULL,
			billing_period VARCHAR(20) NOT NULL,
			limit_key VARCHAR(100) NOT NULL,
			usage_amount BIGINT NOT NULL DEFAULT 0,
			app_bundle_key VARCHAR(100),
			operation_id UUID,
			last_operation_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
		);`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS user_identity VARCHAR(255);`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS billing_period VARCHAR(20);`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS limit_key VARCHAR(100);`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS usage_amount BIGINT DEFAULT 0;`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS app_bundle_key VARCHAR(100);`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS last_operation_at TIMESTAMP;`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT NOW();`,
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT NOW();`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_identity_period_key_app ON usage_records(user_identity, billing_period, limit_key, app_bundle_key);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_user_period ON usage_records(user_identity, billing_period);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_limit_key ON usage_records(limit_key);`,
		`CREATE INDEX IF NOT EXISTS idx_usage_records_app ON usage_records(app_bundle_key);`,
		// Migration: Add operation_id column for idempotency (safe to run multiple times)
		`ALTER TABLE usage_records ADD COLUMN IF NOT EXISTS operation_id UUID;`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_operation_id ON usage_records(operation_id) WHERE operation_id IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			provider VARCHAR(50) NOT NULL UNIQUE,
			encrypted_key TEXT NOT NULL,
			key_hint VARCHAR(20),
			is_active BOOLEAN DEFAULT true,
			last_verified_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider);`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);`,
		// User authentication tables
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			email_verified BOOLEAN DEFAULT FALSE,
			stripe_customer_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW(),
			updated_at TIMESTAMP DEFAULT NOW(),
			last_login_at TIMESTAMP
		);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE INDEX IF NOT EXISTS idx_users_stripe_customer ON users(stripe_customer_id);`,
		`CREATE TABLE IF NOT EXISTS auth_tokens (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			token_hash VARCHAR(255) NOT NULL,
			token_type VARCHAR(50) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			used_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT NOW(),
			ip_address INET,
			user_agent TEXT
		);`,
		`CREATE INDEX IF NOT EXISTS idx_auth_tokens_hash ON auth_tokens(token_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_expires ON auth_tokens(user_id, expires_at);`,
		`CREATE TABLE IF NOT EXISTS user_sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID REFERENCES users(id) ON DELETE CASCADE,
			refresh_token_hash VARCHAR(255) NOT NULL,
			expires_at TIMESTAMP NOT NULL,
			created_at TIMESTAMP DEFAULT NOW(),
			last_used_at TIMESTAMP DEFAULT NOW(),
			ip_address INET,
			user_agent TEXT,
			device_info JSONB DEFAULT '{}',
			revoked BOOLEAN DEFAULT FALSE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_hash ON user_sessions(refresh_token_hash);`,
		`CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id, revoked, expires_at);`,
		// Migration: Add stripe_event_id column for webhook idempotency
		`ALTER TABLE credit_transactions ADD COLUMN IF NOT EXISTS stripe_event_id VARCHAR(255);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;`,
		// Credit reservations table for TOCTOU prevention in streaming requests
		`CREATE TABLE IF NOT EXISTS credit_reservations (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_identity VARCHAR(255) NOT NULL,
			billing_period VARCHAR(20) NOT NULL,
			limit_key VARCHAR(100) NOT NULL,
			reserved_amount BIGINT NOT NULL,
			status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'finalized', 'released', 'expired')),
			created_at TIMESTAMP DEFAULT NOW(),
			finalized_at TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);`,
		`ALTER TABLE credit_reservations ADD COLUMN IF NOT EXISTS user_identity VARCHAR(255);`,
		`CREATE INDEX IF NOT EXISTS idx_credit_reservations_user ON credit_reservations(user_identity, status);`,
		`CREATE INDEX IF NOT EXISTS idx_credit_reservations_expires ON credit_reservations(expires_at) WHERE status = 'pending';`,
		// Intro pricing flag on users table (coupon-based intro pricing)
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS has_used_intro BOOLEAN DEFAULT FALSE;`,
		`CREATE INDEX IF NOT EXISTS idx_users_has_used_intro ON users(has_used_intro);`,
		// Intro coupon usage audit table
		`CREATE TABLE IF NOT EXISTS intro_coupon_usage (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) NOT NULL,
			stripe_customer_id VARCHAR(255),
			coupon_id VARCHAR(255) NOT NULL,
			plan_tier VARCHAR(50),
			subscription_id VARCHAR(255),
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_intro_coupon_usage_email ON intro_coupon_usage(email);`,
		// Intro anomaly log for fraud detection and admin review
		`CREATE TABLE IF NOT EXISTS intro_anomaly_log (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) NOT NULL,
			customer_id VARCHAR(255),
			coupon_id VARCHAR(255),
			anomaly_type VARCHAR(50) NOT NULL,
			details JSONB DEFAULT '{}'::jsonb,
			created_at TIMESTAMP DEFAULT NOW()
		);`,
		`CREATE INDEX IF NOT EXISTS idx_intro_anomaly_log_email ON intro_anomaly_log(email);`,
		`CREATE INDEX IF NOT EXISTS idx_intro_anomaly_log_type ON intro_anomaly_log(anomaly_type);`,
		`CREATE INDEX IF NOT EXISTS idx_intro_anomaly_log_created ON intro_anomaly_log(created_at);`,
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
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

//nolint:unused // reserved for future structured logging hooks
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

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))
	// This scenario uses the landing-manager database (as defined in service.json).
	// Override the default POSTGRES_DB if it's set to the global 'vrooli' database.
	if name == "" || name == "vrooli" {
		name = "landing-manager"
	}

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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "landing-page-business-suite",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
