package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/envx"
	"landing-page-business-suite-api/internal/experimentation"

	_ "github.com/lib/pq"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestDB creates a test database connection
// This is the canonical setup function used across all test files
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Tests must never fall back to DATABASE_URL. That variable is owned by the
	// scenario lifecycle and may point at a developer or deployed database; using
	// it makes the suite both unsafe and dependent on whatever schema happens to
	// exist there. An explicitly named test database is allowed for CI. When it
	// is absent or unavailable, the suite creates an isolated testcontainer.
	if dbURL, configured := configuredTestDatabaseURL(); configured {
		if db, err := openAndPrepareTestDatabase(dbURL); err == nil {
			return registerTestDatabaseCleanup(t, db, nil)
		} else {
			t.Logf("Failed to connect using configured test database; using isolated container: %v", err)
		}
	}

	db, cleanup, err := openIsolatedTestDatabase(t, startTestContainerDB(t))
	if err != nil {
		t.Fatalf("Failed to initialize isolated test database: %v", err)
	}
	return registerTestDatabaseCleanup(t, db, cleanup)
}

// registerTestDatabaseCleanup makes database lifecycle ownership explicit at
// the fixture boundary. Existing callers that close the database themselves
// remain safe during the incremental migration because sql.DB.Close is
// idempotent for this purpose.
func registerTestDatabaseCleanup(t *testing.T, db *sql.DB, cleanup func()) *sql.DB {
	t.Helper()
	t.Cleanup(func() {
		_ = db.Close()
		if cleanup != nil {
			cleanup()
		}
	})
	return db
}

// decodeJSONResponse decodes a handler response and fails the current test with
// the response body when it is not valid JSON. Keeping this at the test fixture
// boundary makes handler assertions concise without hiding their response types
// or behavior checks.
func decodeJSONResponse(t testing.TB, body []byte, destination any) {
	t.Helper()
	if err := json.Unmarshal(body, destination); err != nil {
		t.Fatalf("decode JSON response: %v; body: %s", err, body)
	}
}

// assertJSONResponse records an assertion failure when a handler body cannot
// be decoded, allowing tests that intentionally collect multiple assertions to
// retain their non-fatal behavior.
func assertJSONResponse(t testing.TB, body []byte, destination any) {
	t.Helper()
	if err := json.Unmarshal(body, destination); err != nil {
		t.Errorf("decode JSON response: %v; body: %s", err, body)
	}
}

func configuredTestDatabaseURL() (string, bool) {
	dbURL := strings.TrimSpace(envx.Get("TEST_DATABASE_URL"))
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

	if err := resetTestSchema(db); err != nil {
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

// resetTestSchema gives every test that uses setupTestDB an empty, isolated
// schema. Several integration tests intentionally recreate individual tables
// to model Stripe edge cases; without this reset those abbreviated tables leak
// into unrelated tests and diverge from the declarative runtime schema.
func resetTestSchema(db *sql.DB) error {
	if _, err := db.Exec(`DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		return fmt.Errorf("reset test schema: %w", err)
	}
	return nil
}

func TestSetupTestDBRestoresDeclarativeSubscriptionSchema(t *testing.T) {
	db := setupTestDB(t)
	if _, err := db.Exec(`DROP TABLE subscriptions; CREATE TABLE subscriptions (subscription_id TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create intentionally incomplete subscription table: %v", err)
	}

	restored := setupTestDB(t)
	for _, column := range []string{"plan_tier", "canceled_at"} {
		var exists bool
		if err := restored.QueryRow(`
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.columns
				WHERE table_schema = 'public' AND table_name = 'subscriptions' AND column_name = $1
			)
		`, column).Scan(&exists); err != nil {
			t.Fatalf("query subscription column %q: %v", column, err)
		}
		if !exists {
			t.Errorf("subscriptions.%s was not restored from the declarative schema", column)
		}
	}
}

var (
	testRequestContext   = context.Background()
	testContainerOnce    sync.Once
	testContainerURL     string
	testContainerCleanup func()
	testContainerInitErr error
	testTemplateOnce     sync.Once
	testTemplateInitErr  error
	testDatabaseSequence atomic.Uint64
)

const testTemplateDatabaseName = "lpbs_test_template"

// openIsolatedTestDatabase creates a database cloned from one declarative,
// seeded template. Tests can freely drop or recreate tables without leaking
// DDL to the next test, while avoiding a full schema rebuild for every call to
// setupTestDB.
func openIsolatedTestDatabase(t *testing.T, containerURL string) (*sql.DB, func(), error) {
	t.Helper()
	if err := initializeTestTemplate(containerURL); err != nil {
		return nil, nil, err
	}

	cloneName := fmt.Sprintf("lpbs_test_%d", testDatabaseSequence.Add(1))
	adminURL, err := databaseURLWithName(containerURL, "postgres")
	if err != nil {
		return nil, nil, err
	}
	admin, err := sql.Open("postgres", adminURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open template admin database: %w", err)
	}
	defer admin.Close()
	if _, err := admin.Exec(`CREATE DATABASE ` + quoteDatabaseIdentifier(cloneName) + ` TEMPLATE ` + quoteDatabaseIdentifier(testTemplateDatabaseName)); err != nil {
		return nil, nil, fmt.Errorf("clone test database: %w", err)
	}

	cloneURL, err := databaseURLWithName(containerURL, cloneName)
	if err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("postgres", cloneURL)
	if err != nil {
		return nil, nil, fmt.Errorf("open cloned test database: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping cloned test database: %w", err)
	}

	return db, func() {
		cleanupAdmin, err := sql.Open("postgres", adminURL)
		if err != nil {
			return
		}
		defer cleanupAdmin.Close()
		_, _ = cleanupAdmin.Exec(`DROP DATABASE IF EXISTS ` + quoteDatabaseIdentifier(cloneName) + ` WITH (FORCE)`)
	}, nil
}

func initializeTestTemplate(containerURL string) error {
	testTemplateOnce.Do(func() {
		base, err := openAndPrepareTestDatabase(containerURL)
		if err != nil {
			testTemplateInitErr = fmt.Errorf("prepare test database template: %w", err)
			return
		}
		if err := base.Close(); err != nil {
			testTemplateInitErr = fmt.Errorf("close prepared template source: %w", err)
			return
		}

		adminURL, err := databaseURLWithName(containerURL, "postgres")
		if err != nil {
			testTemplateInitErr = err
			return
		}
		admin, err := sql.Open("postgres", adminURL)
		if err != nil {
			testTemplateInitErr = fmt.Errorf("open template admin database: %w", err)
			return
		}
		defer admin.Close()
		if _, err := admin.Exec(`DROP DATABASE IF EXISTS ` + quoteDatabaseIdentifier(testTemplateDatabaseName) + ` WITH (FORCE)`); err != nil {
			testTemplateInitErr = fmt.Errorf("drop stale test database template: %w", err)
			return
		}
		baseName, err := databaseNameFromURL(containerURL)
		if err != nil {
			testTemplateInitErr = err
			return
		}
		if _, err := admin.Exec(`CREATE DATABASE ` + quoteDatabaseIdentifier(testTemplateDatabaseName) + ` TEMPLATE ` + quoteDatabaseIdentifier(baseName)); err != nil {
			testTemplateInitErr = fmt.Errorf("create test database template: %w", err)
			return
		}
	})
	return testTemplateInitErr
}

func databaseURLWithName(rawURL, name string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse test database URL: %w", err)
	}
	parsed.Path = "/" + name
	return parsed.String(), nil
}

func databaseNameFromURL(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse test database URL: %w", err)
	}
	name := strings.TrimPrefix(parsed.EscapedPath(), "/")
	if name == "" || strings.ContainsAny(name, "\"'") {
		return "", fmt.Errorf("invalid test database name")
	}
	return name, nil
}

func quoteDatabaseIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func startTestContainerDB(t *testing.T) string {
	t.Helper()

	testContainerOnce.Do(func() {
		if strings.EqualFold(envx.Get("TESTCONTAINERS_DISABLED"), "true") {
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
func setupTestConfigStore(t *testing.T) *experimentation.ConfigStore {
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
		t.Fatal("variants directory not found; ConfigStore tests require the tracked scenario config")
	}
	if brandingPath == "" {
		t.Fatal("branding.json not found; ConfigStore tests require the tracked scenario config")
	}

	cs := experimentation.NewConfigStore(variantsDir, brandingPath, nil)
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
		DatabaseURL: envx.Get("DATABASE_URL"),
	}

	// Initialize ConfigStore from JSON files
	configStore := setupTestConfigStore(t)

	// Initialize all services
	metricsService := NewMetricsService(db)
	planService := NewPlanService(db)
	paymentSettings := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	downloadService := NewDownloadService(db)
	seoService := NewSEOService(configStore)
	feedbackService := NewFeedbackService(db)
	emailService := NewEmailService()

	server := &Server{
		config:               config,
		db:                   db,
		variantSpace:         experimentation.DefaultVariantSpace(),
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

// writeSnapshot writes a variant snapshot to a JSON file for testing
func writeSnapshot(t *testing.T, dir string, snapshot experimentation.VariantSnapshotInput) {
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
func newUserAuthServiceForTest(db *sql.DB, emailService *EmailService) *administration.UserAuthService {
	return newUserAuthServiceForTestWithOptions(db, emailService, 15*time.Minute, 7*24*time.Hour, 15*time.Minute)
}

func newUserAuthServiceForTestWithOptions(db *sql.DB, emailService *EmailService, accessTTL, refreshTTL, magicLinkTTL time.Duration) *administration.UserAuthService {
	return administration.NewUserAuthService(administration.UserAuthServiceOptions{
		Store:        db,
		EmailService: emailService,
		JWTSecret:    "test-secret-key",
		JWTIssuer:    "test",
		BaseURL:      "http://localhost:3000/auth/verify",
		AppName:      "Test App",
		AccessTTL:    accessTTL,
		RefreshTTL:   refreshTTL,
		MagicLinkTTL: magicLinkTTL,
	})
}

func setupMinimalAuthServer(t *testing.T, authService *administration.UserAuthService) *Server {
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
		if strings.EqualFold(envx.Get("TESTCONTAINERS_DISABLED"), "true") {
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
