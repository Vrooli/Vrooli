package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a test database connection
// This is the canonical setup function used across all test files
func setupTestDB(t *testing.T) *sql.DB {
	// Tests must never fall back to DATABASE_URL. That variable is owned by the
	// scenario lifecycle and may point at a developer or deployed database; using
	// it makes the suite both unsafe and dependent on whatever schema happens to
	// exist there. An explicitly named test database is allowed for CI. When it
	// is absent or unavailable, the suite creates an isolated testcontainer.
	if dbURL, configured := configuredTestDatabaseURL(); configured {
		if db, err := openAndPrepareTestDatabase(dbURL); err == nil {
			return db
		} else {
			t.Logf("Failed to connect using configured test database; using isolated container: %v", err)
		}
	}

	db, err := openAndPrepareTestDatabase(startTestContainerDB(t))
	if err != nil {
		t.Fatalf("Failed to initialize isolated test database: %v", err)
	}
	return db
}

func configuredTestDatabaseURL() (string, bool) {
	dbURL := strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	return dbURL, dbURL != ""
}

func openAndPrepareTestDatabase(dbURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := applyRuntimeSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := seedDefaultData(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	// NOTE: syncVariantSnapshots has been removed.
	// Variant configuration is now loaded from JSON files via ConfigStore.
	// For tests that need ConfigStore, use setupTestConfigStore().

	return db, nil
}

var (
	testRequestContext   = context.Background()
	testContainerOnce    sync.Once
	testContainerURL     string
	testContainerCleanup func()
	testContainerInitErr error
)

func startTestContainerDB(t *testing.T) string {
	t.Helper()

	testContainerOnce.Do(func() {
		if strings.EqualFold(os.Getenv("TESTCONTAINERS_DISABLED"), "true") {
			testContainerInitErr = fmt.Errorf("testcontainers explicitly disabled")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		user := "testuser"
		pass := "testpass"
		dbName := "landing_manager_test"

		//nolint:staticcheck // legacy testcontainers helper still required for current runtime
		container, err := postgres.RunContainer(ctx,
			tc.WithImage("postgres:15-alpine"),
			tc.WithWaitStrategy(wait.ForListeningPort("5432/tcp").WithStartupTimeout(90*time.Second)),
			postgres.WithDatabase(dbName),
			postgres.WithUsername(user),
			postgres.WithPassword(pass),
		)
		if err != nil {
			testContainerInitErr = fmt.Errorf("start postgres container: %w", err)
			return
		}

		connStr, err := container.ConnectionString(ctx, "sslmode=disable")
		if err != nil {
			testContainerInitErr = fmt.Errorf("connection string: %w", err)
			return
		}

		testContainerURL = connStr
		testContainerCleanup = func() {
			_ = container.Terminate(context.Background())
		}
	})

	if testContainerInitErr != nil {
		t.Fatalf("Failed to initialize test container: %v", testContainerInitErr)
	}
	if testContainerURL == "" {
		t.Fatalf("Test container URL was not set")
	}
	return testContainerURL
}

// setupTestConfigStore creates a ConfigStore for testing using the project's JSON files
func setupTestConfigStore(t *testing.T) *ConfigStore {
	t.Helper()

	// Find the variants directory - look up from the api directory
	variantsDir := ""
	brandingPath := ""

	candidates := []string{
		filepath.Join("..", "config", "variants"),
		filepath.Join(".", "config", "variants"),
		filepath.Join("..", "..", "config", "variants"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			variantsDir = candidate
			break
		}
	}

	brandingCandidates := []string{
		filepath.Join("..", "config", "branding.json"),
		filepath.Join(".", "config", "branding.json"),
		filepath.Join("..", "..", "config", "branding.json"),
	}
	for _, candidate := range brandingCandidates {
		if _, err := os.Stat(candidate); err == nil {
			brandingPath = candidate
			break
		}
	}

	if variantsDir == "" {
		t.Skip("variants directory not found - skipping ConfigStore test")
	}
	if brandingPath == "" {
		t.Skip("branding.json not found - skipping ConfigStore test")
	}

	cs := NewConfigStore(variantsDir, brandingPath, nil) // nil uses defaultVariantSpace
	if err := cs.LoadAll(); err != nil {
		t.Fatalf("Failed to load ConfigStore: %v", err)
	}

	return cs
}

// setupTestServer creates a complete test server instance with all services initialized
//
//nolint:unused // helper retained for future handler tests
func setupTestServer(t *testing.T) (*Server, func()) {
	db := setupTestDB(t)

	// Clean up any existing test data BEFORE creating the server
	// This prevents duplicate key violations from previous test runs
	if _, err := db.Exec("DELETE FROM admin_sessions WHERE admin_user_id IN (SELECT id FROM admin_users WHERE email LIKE '%@test.com')"); err != nil {
		t.Fatalf("failed to cleanup admin sessions: %v", err)
	}
	if _, err := db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'"); err != nil {
		t.Fatalf("failed to cleanup admin users: %v", err)
	}

	// Create a test config
	config := &Config{
		Port:        "0", // Use random port for testing
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	// Initialize ConfigStore from JSON files
	configStore := setupTestConfigStore(t)

	// Initialize all services
	metricsService := NewMetricsService(db)
	planService := NewPlanService(db)
	paymentSettings := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	downloadService := NewDownloadService(db)
	seoService := NewSEOServiceWithConfigStore(configStore)
	feedbackService := NewFeedbackService(db)
	emailService := NewEmailService()

	server := &Server{
		config:               config,
		db:                   db,
		variantSpace:         defaultVariantSpace,
		configStore:          configStore,
		metricsService:       metricsService,
		stripeService:        stripeService,
		planService:          planService,
		downloadService:      downloadService,
		paymentSettings:      paymentSettings,
		landingConfigService: NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService, stripeService),
		seoService:           seoService,
		feedbackService:      feedbackService,
		emailService:         emailService,
	}

	cleanup := func() {
		// Clean up test data after test completes
		if _, err := db.Exec("DELETE FROM admin_sessions WHERE admin_user_id IN (SELECT id FROM admin_users WHERE email LIKE '%@test.com')"); err != nil {
			t.Fatalf("failed to cleanup admin sessions: %v", err)
		}
		if _, err := db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'"); err != nil {
			t.Fatalf("failed to cleanup admin users: %v", err)
		}
		db.Close()
	}

	return server, cleanup
}

func resetStripeTestData(t *testing.T, db *sql.DB) {
	t.Helper()
	tables := []string{
		"subscription_schedules",
		"subscriptions",
		"checkout_sessions",
		"credit_transactions",
		"credit_wallets",
		"payment_settings",
		"bundle_prices",
		"bundle_products",
	}
	for _, table := range tables {
		if _, err := db.Exec("DELETE FROM " + table); err != nil {
			t.Fatalf("failed to clean %s: %v", table, err)
		}
	}

	if err := seedDefaultData(db); err != nil {
		t.Fatalf("failed to reseed defaults: %v", err)
	}
}

// DEPRECATED: createTestVariant creates a test variant in the database.
// The 'variants' table has been removed - variants are now stored in JSON files.
// This function is retained for legacy test compatibility only.
// For new tests, use setupTestConfigStore() instead.
//
//nolint:unused // retained for legacy tests that depend on database-backed variants
func createTestVariant(t *testing.T, db *sql.DB) int64 {
	t.Skip("variants table removed - use ConfigStore for variant tests")
	return 0
}

// writeSnapshot writes a variant snapshot to a JSON file for testing
func writeSnapshot(t *testing.T, dir string, snapshot VariantSnapshotInput) {
	t.Helper()

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}

	path := filepath.Join(dir, snapshot.Variant.Slug+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write snapshot: %v", err)
	}
}

// boolPtr returns a pointer to the given bool value (test helper)
func boolPtr(value bool) *bool {
	return &value
}

// defaultAxesSelection returns default axes selection for testing
func defaultAxesSelection() map[string]string {
	return map[string]string{
		"persona":         "ops_leader",
		"jtbd":            "launch_bundle",
		"conversionStyle": "demo_led",
	}
}

// setupMinimalAuthServer creates a lightweight server for middleware testing.
// Only initializes the userAuthService, which is all that's needed for auth middleware tests.
func setupMinimalAuthServer(t *testing.T, authService *UserAuthService) *Server {
	t.Helper()
	return &Server{
		userAuthService: authService,
	}
}

// MinIO testcontainer support for S3-compatible storage testing
//
//nolint:unused // used by integration-tagged tests
var (
	minioOnce      sync.Once
	minioContainer tc.Container
	minioEndpoint  string
	minioInitErr   error
)

// setupMinIOContainer starts a MinIO container for S3 integration tests.
// Returns endpoint URL, access key, and secret key.
// The container is shared across all tests in the test run.
//
//nolint:unused // used by integration-tagged tests
func setupMinIOContainer(t *testing.T) (endpoint, accessKey, secretKey string) {
	t.Helper()

	minioOnce.Do(func() {
		if strings.EqualFold(os.Getenv("TESTCONTAINERS_DISABLED"), "true") {
			minioInitErr = fmt.Errorf("testcontainers explicitly disabled")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		req := tc.ContainerRequest{
			Image:        "minio/minio:latest",
			ExposedPorts: []string{"9000/tcp"},
			Env: map[string]string{
				"MINIO_ROOT_USER":     "minioadmin",
				"MINIO_ROOT_PASSWORD": "minioadmin",
			},
			Cmd:        []string{"server", "/data"},
			WaitingFor: wait.ForHTTP("/minio/health/live").WithPort("9000"),
		}

		container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			minioInitErr = fmt.Errorf("start minio container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			minioInitErr = fmt.Errorf("get container host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "9000")
		if err != nil {
			minioInitErr = fmt.Errorf("get mapped port: %w", err)
			return
		}

		minioContainer = container
		minioEndpoint = fmt.Sprintf("http://%s:%s", host, port.Port())
	})

	if minioInitErr != nil {
		t.Skipf("MinIO container not available: %v", minioInitErr)
	}

	return minioEndpoint, "minioadmin", "minioadmin"
}
