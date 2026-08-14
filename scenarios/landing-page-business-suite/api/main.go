// DOC: docs/concepts/ARCHITECTURE.md - System design and component overview
// DOC: docs/QUICKSTART.md - Getting started guide
// DOC: PRD.md - Product requirements and operational targets
package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
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
	"github.com/vrooli/api-core/consumeridentity"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	corestorage "github.com/vrooli/api-core/storage"
	aihandler "landing-page-business-suite-api/handlers/intelligence"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/analytics"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/content"
	"landing-page-business-suite-api/internal/delivery"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/experimentation"
	"landing-page-business-suite-api/internal/intelligence"
	"landing-page-business-suite-api/internal/landing"
	"landing-page-business-suite-api/internal/logx"
	domainmetrics "landing-page-business-suite-api/internal/metrics"
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
	variantSpace         *experimentation.VariantSpace
	configStore          *experimentation.ConfigStore
	metricsService       *domainmetrics.Service
	stripeService        *StripeService
	planService          *commerce.PlanService
	downloadService      *delivery.CatalogService
	downloadHosting      *delivery.Service
	downloadAuthorizer   *delivery.DownloadAuthorizer
	accountService       *commerce.Service
	landingConfigService *landing.LandingConfigService
	paymentSettings      *commerce.PaymentSettingsService
	paymentAnomaly       *commerce.PaymentAnomalyService
	assetsService        *content.AssetsService
	seoService           *content.SEOService
	feedbackService      *domainmetrics.FeedbackService
	adminAuthService     *administration.AdminAuthService
	emailService         *EmailService
	waitlistService      *domainmetrics.WaitlistService
	// Credit system services
	apiKeyService *administration.APIKeyService
	limitsService *commerce.LimitsService
	usageService  *commerce.UsageService
	// Remote profile service (admin-managed remote connections)
	remoteProfileService *administration.RemoteProfileService
	// User authentication services
	userAuthService       *administration.UserAuthService
	userManagementService *administration.UserManagementService
	magicLinkLimiter      *RateLimiter
	// AI MeteredInferenceProvider service
	meteredInferenceService *intelligence.MeteredInferenceService
	meteredInferenceHandler *aihandler.Handler
	meteredInferenceDeps    aihandler.Dependencies
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
		administration.Schema(),
		analytics.Schema(),
		commerce.FinancialSchema(),
		delivery.Schema(),
		content.Schema(),
		commerce.OperationsSchema(),
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

// Mount satisfies devrouting's optional subtree-mount contract. Connect
// services expose several RPC paths below one service prefix; registering that
// prefix as a Gorilla path prefix ensures every generated procedure reaches
// the handler rather than falling through to the application router as a 404.
func (m devRoutingMux) Mount(pattern string, handler http.Handler) {
	m.router.PathPrefix(pattern).Handler(handler)
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	logStructured("server_initialization_started", nil)
	if err := validateProductionCredentials(); err != nil {
		return nil, err
	}
	logStructured("server_credentials_validated", nil)

	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	routedDB, err := database.Open(context.Background(), database.Config{
		Driver: "postgres",
		Logger: logx.Printf,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	logStructured("server_database_connected", nil)
	db := routedDB.Primary()
	routedDB.SetTestPoolInitializer(func(ctx context.Context, testDB *sql.DB) error {
		if err := database.EnsureSchemas(ctx, testDB, database.SchemaProviderFunc(runtimeSchema)); err != nil {
			return err
		}
		return seedDefaultData(testDB)
	})

	if err := seedDefaultData(db); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}
	logStructured("server_database_seeded", nil)

	// Initialize config store from tracked scenario config files.
	variantsDir := resolveVariantsDir()
	brandingPath := resolveBrandingPath()
	variantSpace := experimentation.DefaultVariantSpace()
	configStore := experimentation.NewConfigStore(variantsDir, brandingPath, variantSpace)
	if err := configStore.LoadAll(); err != nil {
		return nil, fmt.Errorf("failed to load config from JSON files: %w", err)
	}
	logStructured("server_configuration_loaded", nil)

	planService := NewPlanService(db)
	downloadService := delivery.NewCatalogService(delivery.NewRoutedCatalogStore(routedDB))
	downloadHosting := delivery.NewService(db, delivery.S3StorageProvider{})
	limitsService := commerce.NewLimitsService(routedDB, "postgres", logStructured)
	accountService := newAccountService(routedDB, planService, limitsService)
	downloadAuthorizer := delivery.NewDownloadAuthorizer(downloadService, accountService, planService.BundleKey())
	paymentSettings := commerce.NewPaymentSettingsService(routedDB)
	paymentAnomaly := commerce.NewPaymentAnomalyService(context.Background(), routedDB, context.Background(), commerce.PaymentAnomalyRuntime{
		ScenarioName:   "landing-page-business-suite",
		NormalizeEmail: NormalizeEmail,
		Log:            logStructured,
		LogError:       logStructuredError,
	})
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	stripeService.SetPaymentAnomaly(paymentAnomaly)
	assetsService := NewAssetsService(db)
	fileRoots := filerouting.New(runtimeStoragePaths(variantsDir, assetsService.GetUploadDir()))
	assetsService.SetFileRoots(fileRoots)
	seoService := NewSEOService(configStore)
	feedbackService := domainmetrics.NewFeedbackService(routedDB)
	emailService := NewEmailService()
	// Waitlist is the first request-context-aware domain migrated to RoutedDB.
	// Test-mode requests reach the lease-owned pool while all other services
	// continue their explicit, staged migration from the primary pool.
	waitlistService := domainmetrics.NewWaitlistService(routedDB)

	// Initialize credit system services
	apiKeyService, err := NewAPIKeyService(routedDB)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API key service: %w", err)
	}
	logStructured("server_api_key_service_initialized", nil)
	remoteProfileService, err := administration.NewRemoteProfileServiceWithRuntime(
		routedDB,
		nil,
		resolveSecret,
		isProductionEnvironment,
		logStructured,
		logStructuredError,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize remote profile service: %w", err)
	}
	logStructured("server_remote_profile_service_initialized", nil)
	usageService := newRuntimeUsageService(routedDB, limitsService, "postgres")

	// Initialize user authentication services
	consumerSigningKeyPEM, keyErr := resolveConsumerSigningKey()
	if keyErr != nil {
		return nil, fmt.Errorf("resolve consumer signing key: %w", keyErr)
	}
	if isProductionEnvironment() && strings.TrimSpace(consumerSigningKeyPEM) == "" {
		return nil, fmt.Errorf("CONSUMER_AUTH_PRIVATE_KEY is required in production")
	}
	consumerKeyID := strings.TrimSpace(resolveConfig("CONSUMER_AUTH_KEY_ID"))
	if isProductionEnvironment() && consumerKeyID == "" {
		return nil, fmt.Errorf("CONSUMER_AUTH_KEY_ID is required in production")
	}
	previousConsumerKeys, keyErr := resolvePreviousConsumerKeys()
	if keyErr != nil {
		return nil, fmt.Errorf("resolve previous consumer signing keys: %w", keyErr)
	}
	userAuthService := administration.NewUserAuthService(administration.UserAuthServiceOptions{
		Store:                 routedDB,
		EmailService:          emailService,
		JWTIssuer:             resolveSecret("JWT_ISSUER"),
		ConsumerSigningKeyPEM: consumerSigningKeyPEM,
		ConsumerSigningKeyID:  consumerKeyID,
		ConsumerPreviousKeys:  previousConsumerKeys,
		ConsumerClockSkew:     30 * time.Second,
		BaseURL:               resolveConfig("AUTH_MAGIC_LINK_BASE_URL"),
		AppName:               resolveConfig("EMAIL_FROM_NAME"),
		Log:                   logStructured,
		LogError:              logStructuredError,
	})
	if userAuthService == nil {
		return nil, fmt.Errorf("failed to initialize consumer signing key")
	}
	if _, err := userAuthService.PublicKeySet(); err != nil {
		return nil, fmt.Errorf("failed to publish consumer key set: %w", err)
	}
	userManagementService := administration.NewUserManagementService(routedDB)
	// Rate limiter: 5 requests per 15 minutes per email for magic link
	magicLinkLimiter := NewRateLimiter(5, 15*time.Minute)

	// Initialize metered inference provider service
	meteredInferenceService := intelligence.NewMeteredInferenceService(intelligence.MeteredInferenceServiceOptions{
		APIKeyService:  apiKeyService,
		UsageService:   newCommerceUsageServicer(usageService),
		AccountService: accountService,
		Logger:         logStructured,
		ClientFactory: func(apiKey string, logger func(string, map[string]interface{})) intelligence.OpenRouterClient {
			return intelligence.NewOpenRouterClient(intelligence.OpenRouterClientOptions{
				APIKey:  apiKey,
				BaseURL: envx.Get("OPENROUTER_BASE_URL"),
				Referer: envx.Get("OPENROUTER_REFERER"),
				Title:   envx.Get("OPENROUTER_TITLE"),
				Logger:  logger,
			})
		},
	})

	// Create metered inference provider dependencies with rate limiters
	meteredInferenceDeps := newMeteredInferenceDependencies(meteredInferenceService, usageService, accountService)
	meteredInferenceHandler := aihandler.New(meteredInferenceDeps)

	srv := &Server{
		config:               &Config{},
		db:                   db,
		routedDB:             routedDB,
		fileRoots:            fileRoots,
		router:               mux.NewRouter(),
		variantSpace:         variantSpace,
		configStore:          configStore,
		metricsService:       domainmetrics.NewService(db),
		stripeService:        stripeService,
		planService:          planService,
		downloadService:      downloadService,
		downloadHosting:      downloadHosting,
		downloadAuthorizer:   downloadAuthorizer,
		accountService:       accountService,
		landingConfigService: newLandingConfigService(configStore, planService, downloadService, stripeService),
		paymentSettings:      paymentSettings,
		paymentAnomaly:       paymentAnomaly,
		assetsService:        assetsService,
		seoService:           seoService,
		feedbackService:      feedbackService,
		adminAuthService:     administration.NewAdminAuthService(routedDB),
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
		// AI MeteredInferenceProvider service
		meteredInferenceService: meteredInferenceService,
		meteredInferenceHandler: meteredInferenceHandler,
		meteredInferenceDeps:    meteredInferenceDeps,
		// Session management
		sessionManager: initSessionManager(),
	}

	srv.setupRoutes()
	if err := registerScenarioDevRouting(srv.router, routedDB, fileRoots); err != nil {
		_ = routedDB.Close()
		return nil, fmt.Errorf("register development routing: %w", err)
	}
	logStructured("server_initialization_completed", nil)
	return srv, nil
}

// resolvePreviousConsumerKeys loads the public overlap set used during key
// rotation. It is deliberately configuration, not a credential: only public
// keys are accepted and no private material is ever read here.
func resolvePreviousConsumerKeys() ([]consumeridentity.PublicKey, error) {
	raw := strings.TrimSpace(resolveConfig("CONSUMER_AUTH_PREVIOUS_JWKS"))
	if raw == "" {
		return nil, nil
	}
	set, err := consumeridentity.ParseJWKS([]byte(raw))
	if err != nil {
		return nil, err
	}
	keys := make([]consumeridentity.PublicKey, 0, len(set.Keys))
	for id, key := range set.Keys {
		keys = append(keys, consumeridentity.PublicKey{ID: id, Key: key})
	}
	return keys, nil
}

// resolveConsumerSigningKey keeps local development tokens valid across API
// restarts without ever making a generated private key part of the repository.
// Production deployments must inject CONSUMER_AUTH_PRIVATE_KEY through their
// secret manager; the persisted local path is intentionally not used there.
func resolveConsumerSigningKey() (string, error) {
	if key := strings.TrimSpace(resolveSecret("CONSUMER_AUTH_PRIVATE_KEY")); key != "" {
		return key, nil
	}
	if isProductionEnvironment() {
		return "", nil
	}
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return "", err
	}
	scenarioID, err := corestorage.ScenarioNamespace("landing-page-business-suite")
	if err != nil {
		return "", err
	}
	path, err := resolver.Path(corestorage.Options{ScenarioID: scenarioID}, corestorage.ClassConfig, "consumer-auth-private.pem")
	if err != nil {
		return "", err
	}
	if data, readErr := os.ReadFile(path); readErr == nil {
		return string(data), nil
	} else if !os.IsNotExist(readErr) {
		return "", readErr
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return "", err
	}
	data := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return string(data), nil
}

// newRuntimeUsageService supplies process configuration at the composition
// boundary. Commerce owns usage rules; this function owns only environment and
// logging policy needed to start the process.
func newRuntimeUsageService(db commerce.UsageStore, limitsSvc commerce.LimitsServicer, dialect string) *commerce.UsageService {
	return commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB:                  db,
		LimitsService:       limitsSvc,
		Dialect:             dialect,
		Log:                 logStructured,
		InsufficientCredits: intelligence.ErrInsufficientCredits,
	})
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

// NOTE: syncVariantSnapshots function has been removed.
// Variant configuration is now loaded directly from JSON files via ConfigStore.LoadAll()
// which is called in NewServer(). No database sync is needed.

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
		Handler: apihttp.TestModeMiddleware(srv.Router()),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		logx.Fatalf("server error: %v", err)
	}
}
