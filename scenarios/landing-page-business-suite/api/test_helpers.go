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
	dbURL := ""
	if resolved, err := resolveDatabaseURL(); err == nil {
		dbURL = resolved
	}
	if dbURL == "" {
		dbURL = strings.TrimSpace(os.Getenv("TEST_DATABASE_URL"))
	}
	if dbURL == "" {
		dbURL = startTestContainerDB(t)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		// Retry with container if an external URL is misconfigured
		if dbURL != "" {
			t.Logf("Failed to connect using %s, retrying with container: %v", dbURL, err)
			dbURL = startTestContainerDB(t)
			db, err = sql.Open("postgres", dbURL)
		}
		if err != nil {
			t.Fatalf("Failed to connect to test database: %v", err)
		}
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	if err := ensureSchema(db); err != nil {
		t.Fatalf("Failed to ensure schema: %v", err)
	}

	if err := seedDefaultData(db); err != nil {
		t.Fatalf("Failed to seed default data: %v", err)
	}

	// NOTE: syncVariantSnapshots has been removed.
	// Variant configuration is now loaded from JSON files via ConfigStore.
	// For tests that need ConfigStore, use setupTestConfigStore().

	return db
}

var (
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
		filepath.Join("..", ".vrooli", "variants"),
		filepath.Join(".", ".vrooli", "variants"),
		filepath.Join("..", "..", ".vrooli", "variants"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			variantsDir = candidate
			break
		}
	}

	brandingCandidates := []string{
		filepath.Join("..", ".vrooli", "branding.json"),
		filepath.Join(".", ".vrooli", "branding.json"),
		filepath.Join("..", "..", ".vrooli", "branding.json"),
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
		landingConfigService: NewLandingConfigServiceWithConfigStore(configStore, planService, downloadService),
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

// testVariantSpace creates a variant space for testing
func testVariantSpace() *VariantSpace {
	return &VariantSpace{
		Name:          "test-space",
		SchemaVersion: 1,
		Axes: map[string]*AxisDefinition{
			"persona": {
				Variants: []AxisVariant{{ID: "ops_leader", Label: "Ops Leader"}},
			},
			"jtbd": {
				Variants: []AxisVariant{{ID: "launch_bundle", Label: "Launch bundle"}},
			},
			"conversionStyle": {
				Variants: []AxisVariant{{ID: "demo_led", Label: "Demo-led"}},
			},
		},
	}
}
