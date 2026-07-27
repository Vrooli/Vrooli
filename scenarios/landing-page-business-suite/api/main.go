// DOC: docs/concepts/ARCHITECTURE.md - System design and component overview
// DOC: docs/QUICKSTART.md - Getting started guide
// DOC: PRD.md - Product requirements and operational targets
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	corestorage "github.com/vrooli/api-core/storage"
	"landing-page-business-suite-api/internal/admin"
	"landing-page-business-suite-api/internal/analytics"
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/download"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/financial"
	"landing-page-business-suite-api/internal/logx"
	"landing-page-business-suite-api/internal/operations"
	runtimeschema "landing-page-business-suite-api/internal/schema"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config               *Config
	db                   StartupStore
	routedDB             *database.RoutedDB
	fileRoots            *filerouting.RoutedRoots
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
	paymentAnomaly       *PaymentAnomalyService
	assetsService        *AssetsService
	seoService           *SEOService
	feedbackService      *FeedbackService
	adminAuthService     *AdminAuthService
	emailService         *EmailService
	waitlistService      *WaitlistService
	// Credit system services
	apiKeyService *APIKeyService
	limitsService *LimitsService
	usageService  *UsageService
	// Remote profile service (admin-managed remote connections)
	remoteProfileService *RemoteProfileService
	// User authentication services
	userAuthService       *UserAuthService
	userManagementService *UserManagementService
	magicLinkLimiter      *RateLimiter
	// AI Gateway service
	aiGatewayService *AIGatewayService
	aiGatewayDeps    *AIGatewayDeps
	// Session management for admin auth
	sessionManager SessionManager
}

type StartupStore interface {
	Close() error
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryRow(string, ...any) *sql.Row
	QueryRowContext(context.Context, string, ...any) *sql.Row
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// runtimeSchema composes system and product schemas at the composition root.
// Keeping this import graph here prevents the shared schema substrate from
// depending on product domains.
func runtimeSchema() string {
	return strings.Join([]string{
		runtimeschema.System(),
		admin.Schema(),
		analytics.Schema(),
		financial.Schema(),
		download.Schema(),
		content.Schema(),
		operations.Schema(),
	}, "\n")
}

func (s *Server) primaryDB() *sql.DB {
	if db, ok := s.db.(*sql.DB); ok {
		return db
	}
	return nil
}

// devRoutingMux adapts Gorilla's fluent Handle signature to the narrow
// development-routing registration contract. The adapter deliberately ignores
// the returned route because devrouting owns only a fixed Connect endpoint.
type devRoutingMux struct{ router *mux.Router }

func (m devRoutingMux) Handle(pattern string, handler http.Handler) {
	m.router.Handle(pattern, handler)
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	if err := validateProductionCredentials(); err != nil {
		return nil, err
	}

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	routedDB, err := database.Open(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	db := routedDB.Primary()
	routedDB.SetTestPoolInitializer(func(ctx context.Context, testDB *sql.DB) error {
		return database.EnsureSchemas(ctx, testDB, database.SchemaProviderFunc(runtimeSchema))
	})

	if err := seedDefaultData(db); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}

	// Initialize config store from tracked scenario config files.
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
	accountService := NewAccountService(routedDB, planService)
	downloadAuthorizer := NewDownloadAuthorizer(downloadService, accountService, planService.BundleKey())
	paymentSettings := NewPaymentSettingsService(routedDB)
	paymentAnomaly := NewPaymentAnomalyService(context.Background(), routedDB, context.Background())
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	stripeService.SetPaymentAnomaly(paymentAnomaly)
	assetsService := NewAssetsService(db)
	fileRoots := filerouting.New(runtimeStoragePaths(variantsDir, assetsService.GetUploadDir()))
	assetsService.SetFileRoots(fileRoots)
	seoService := NewSEOServiceWithConfigStore(configStore)
	feedbackService := NewFeedbackService(routedDB)
	emailService := NewEmailService()
	// Waitlist is the first request-context-aware domain migrated to RoutedDB.
	// Test-mode requests reach the lease-owned pool while all other services
	// continue their explicit, staged migration from the primary pool.
	waitlistService := NewWaitlistService(routedDB)

	// Initialize credit system services
	apiKeyService, err := NewAPIKeyService(routedDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API key service: %w", err)
	}
	remoteProfileService, err := NewRemoteProfileService(routedDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote profile service: %w", err)
	}
	limitsService := NewLimitsService(routedDB, "postgres")
	usageService := NewUsageService(routedDB, limitsService, "postgres")

	// Initialize user authentication services
	userAuthService := NewUserAuthService(routedDB, emailService)
	userManagementService := NewUserManagementService(routedDB)
	// Rate limiter: 5 requests per 15 minutes per email for magic link
	magicLinkLimiter := NewRateLimiter(5, 15*time.Minute)

	// Initialize AI gateway service
	aiGatewayService := NewAIGatewayService(AIGatewayServiceOptions{
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
		routedDB:             routedDB,
		fileRoots:            fileRoots,
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
		paymentAnomaly:       paymentAnomaly,
		assetsService:        assetsService,
		seoService:           seoService,
		feedbackService:      feedbackService,
		adminAuthService:     NewAdminAuthService(routedDB),
		emailService:         emailService,
		waitlistService:      waitlistService,
		// Credit system services
		apiKeyService: apiKeyService,
		limitsService: limitsService,
		usageService:  usageService,
		// Remote profile service
		remoteProfileService: remoteProfileService,
		// User authentication services
		userAuthService:       userAuthService,
		userManagementService: userManagementService,
		magicLinkLimiter:      magicLinkLimiter,
		// AI Gateway service
		aiGatewayService: aiGatewayService,
		aiGatewayDeps:    aiGatewayDeps,
		// Session management
		sessionManager: initSessionManager(),
	}

	srv.setupRoutes()
	devrouting.RegisterWithFileRoots(devRoutingMux{router: srv.router}, routedDB, fileRoots)
	return srv, nil
}

// resolveVariantsDir finds the variants directory
func resolveVariantsDir() string {
	dir := strings.TrimSpace(envx.Get("VARIANT_SNAPSHOT_DIR"))
	if dir != "" {
		return dir
	}
	candidates := []string{
		filepath.Join("..", "config", "variants"),
		filepath.Join(".", "config", "variants"),
		filepath.Join("..", "..", "config", "variants"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
	}
	return filepath.Join("..", "config", "variants")
}

// resolveBrandingPath finds the branding.json file
func resolveBrandingPath() string {
	candidates := []string{
		filepath.Join("..", "config", "branding.json"),
		filepath.Join(".", "config", "branding.json"),
		filepath.Join("..", "..", "config", "branding.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return filepath.Join("..", "config", "branding.json")
}

// runtimeStoragePaths classifies scenario-owned mutable roots for lease-based
// file isolation. Assets are data; tracked landing configuration is config;
// the remaining roots support runtime artifacts without sharing a test run's
// writes with the developer's working tree.
func runtimeStoragePaths(variantsDir, uploadDir string) corestorage.Paths {
	abs := func(path string) string {
		resolved, err := filepath.Abs(path)
		if err != nil {
			return path
		}
		return resolved
	}
	stateRoot := filepath.Join("..", ".vrooli")
	return corestorage.Paths{
		ConfigDir: abs(filepath.Dir(variantsDir)),
		DataDir:   abs(uploadDir),
		CacheDir:  abs(filepath.Join(stateRoot, "cache")),
		LogsDir:   abs(filepath.Join(stateRoot, "logs")),
		StateDir:  abs(filepath.Join(stateRoot, "state")),
	}
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return handlers.RecoveryHandler()(apihttp.TestModeMiddleware(s.router))
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	if s.routedDB != nil {
		return s.routedDB.Close()
	}
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

// applyRuntimeSchema applies the ordered, domain-owned declarative schemas.
// There are deliberately no migration or data-move statements at application
// startup; existing deployments are reconciled by an explicit operator step.
func applyRuntimeSchema(db StartupStore) error {
	concrete, ok := db.(*sql.DB)
	if !ok {
		return fmt.Errorf("schema initialization requires a concrete database connection")
	}
	return database.EnsureSchemas(context.Background(), concrete, database.SchemaProviderFunc(runtimeSchema))
}

// seedDefaultData sets up baseline records that are not variant-specific.
func seedDefaultData(db StartupStore) error {
	if err := applyRuntimeSchema(db); err != nil {
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
		seedDeleteDuplicateAdminSQL,
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
		seedAdminSQL,
		seededAdminID,
		adminEmail,
		adminPasswordHash,
	); err != nil {
		return fmt.Errorf("failed to seed admin user: %w", err)
	}
	// The bootstrap account uses a fixed id so it can be updated in place. Keep
	// the SERIAL sequence in sync or the first subsequently-created admin would
	// collide with that reserved id.
	if _, err := db.Exec(seedAdminSequenceSQL); err != nil {
		return fmt.Errorf("synchronize admin user sequence: %w", err)
	}

	// Log if custom credentials were configured
	if adminEmail != defaultAdminEmail {
		logStructured("admin_user_seeded_custom", map[string]interface{}{
			"level": "info",
			"email": adminEmail,
		})
	}

	if _, err := db.Exec(seedPaymentSettingsSQL); err != nil {
		return fmt.Errorf("failed to seed payment settings: %w", err)
	}

	// NOTE: Site branding is now stored in tracked config JSON.
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

func seedDownloadDefaults(db StartupStore, downloads []DownloadApp) error {
	if len(downloads) == 0 {
		return nil
	}
	var count int
	if err := db.QueryRow(seedDownloadAppCountSQL).Scan(&count); err != nil {
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

		if _, err := db.Exec(seedDownloadAppSQL,
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

			if _, err := db.Exec(seedDownloadAssetSQL,
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
func seedTierLimitsDefaults(db StartupStore) error {
	var count int
	if err := db.QueryRow(seedTierLimitCountSQL).Scan(&count); err != nil {
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

	for _, tl := range tierLimits {
		if _, err := db.Exec(seedTierLimitSQL, tl.tierID, tl.limitType, tl.limitKey, tl.limitValue, tl.appBundleKey); err != nil {
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

// applyRuntimeSchema applies the declarative domain schemas used by both boot
// and tests. It intentionally contains no migration or data-move logic.
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
		logx.Printf(`{"level":"info","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	logx.Printf(`{"level":"info","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

func logStructuredError(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		logx.Printf(`{"level":"error","message":"%s","timestamp":"%s"}`, msg, time.Now().UTC().Format(time.RFC3339))
		return
	}
	fieldsJSON, _ := json.Marshal(fields)
	logx.Printf(`{"level":"error","message":"%s","fields":%s,"timestamp":"%s"}`, msg, fieldsJSON, time.Now().UTC().Format(time.RFC3339))
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(envx.Get("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(envx.Get("POSTGRES_HOST"))
	port := strings.TrimSpace(envx.Get("POSTGRES_PORT"))
	user := strings.TrimSpace(envx.Get("POSTGRES_USER"))
	password := strings.TrimSpace(envx.Get("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(envx.Get("POSTGRES_DB"))
	// This scenario uses its own database (as defined in service.json).
	// Override the default POSTGRES_DB if it's set to the global 'vrooli' database.
	if name == "" || name == "vrooli" {
		name = "landing-page-business-suite"
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
		logx.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		logx.Fatalf("server error: %v", err)
	}
}
