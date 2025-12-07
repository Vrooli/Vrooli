package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/lib/pq"
	tc "github.com/testcontainers/testcontainers-go"
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

		req := tc.ContainerRequest{
			Image:        "postgres:15-alpine",
			Env:          map[string]string{"POSTGRES_USER": user, "POSTGRES_PASSWORD": pass, "POSTGRES_DB": dbName},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
		}

		container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			testContainerInitErr = fmt.Errorf("start postgres container: %w", err)
			return
		}

		host, err := container.Host(ctx)
		if err != nil {
			testContainerInitErr = fmt.Errorf("container host: %w", err)
			return
		}

		port, err := container.MappedPort(ctx, "5432/tcp")
		if err != nil {
			testContainerInitErr = fmt.Errorf("container port: %w", err)
			return
		}

		testContainerURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port.Port(), dbName)
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

// setupTestServer creates a complete test server instance with all services initialized
func setupTestServer(t *testing.T) (*Server, func()) {
	db := setupTestDB(t)

	// Clean up any existing test data BEFORE creating the server
	// This prevents duplicate key violations from previous test runs
	db.Exec("DELETE FROM admin_sessions WHERE admin_user_id IN (SELECT id FROM admin_users WHERE email LIKE '%@test.com')")
	db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'")

	// Create a test config
	config := &Config{
		Port:        "0", // Use random port for testing
		DatabaseURL: os.Getenv("DATABASE_URL"),
	}

	// Initialize all services
	variantService := NewVariantService(db, defaultVariantSpace)
	metricsService := NewMetricsService(db)
	planService := NewPlanService(db)
	paymentSettings := NewPaymentSettingsService(db)
	stripeService := NewStripeServiceWithSettings(db, planService, paymentSettings)
	contentService := NewContentService(db)

	server := &Server{
		config:          config,
		db:              db,
		variantSpace:    defaultVariantSpace,
		variantService:  variantService,
		metricsService:  metricsService,
		stripeService:   stripeService,
		contentService:  contentService,
		planService:     planService,
		paymentSettings: paymentSettings,
	}

	cleanup := func() {
		// Clean up test data after test completes
		db.Exec("DELETE FROM admin_sessions WHERE admin_user_id IN (SELECT id FROM admin_users WHERE email LIKE '%@test.com')")
		db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'")
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

// createTestVariant is a helper to create a test variant for content tests
func createTestVariant(t *testing.T, db *sql.DB) int64 {
	// Clean up any existing test variant first
	db.Exec("DELETE FROM variants WHERE slug = 'test-variant'")

	query := `
		INSERT INTO variants (slug, name, description, weight, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	var id int64
	err := db.QueryRow(
		query,
		"test-variant",
		"Test Variant",
		"Test description",
		50,
		"active",
		time.Now(),
		time.Now(),
	).Scan(&id)

	if err != nil {
		t.Fatalf("Failed to create test variant: %v", err)
	}

	return id
}
