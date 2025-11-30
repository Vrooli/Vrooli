package main

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"

	"landing-manager/handlers"
	"landing-manager/services"
)

// setupTestDB creates a test database connection
// This is the canonical setup function used across all test files
func setupTestDB(t *testing.T) *sql.DB {
	// Use the same database as main tests
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		// Fallback to default test database URL if env vars not set
		// This allows tests to run in CI/CD or local environments without full lifecycle setup
		dbURL = os.Getenv("TEST_DATABASE_URL")
		if dbURL == "" {
			dbURL = "postgresql://vrooli:lUq9qvemypKpuEeXCV6Vnxak1@localhost:5433/landing-manager?sslmode=disable"
		}
		t.Logf("Using fallback database URL (lifecycle env vars not detected)")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping test database: %v", err)
	}

	return db
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
	registry := services.NewTemplateRegistry()
	generator := services.NewScenarioGenerator(registry)
	personaService := services.NewPersonaService(registry.GetTemplatesDir())
	previewService := services.NewPreviewService()
	analyticsService := services.NewAnalyticsService()

	// Create handler with all dependencies
	handler := handlers.NewHandler(db, registry, generator, personaService, previewService, analyticsService)

	server := &Server{
		config:  config,
		db:      db,
		handler: handler,
	}

	cleanup := func() {
		// Clean up test data after test completes
		db.Exec("DELETE FROM admin_sessions WHERE admin_user_id IN (SELECT id FROM admin_users WHERE email LIKE '%@test.com')")
		db.Exec("DELETE FROM admin_users WHERE email LIKE '%@test.com'")
		db.Close()
	}

	return server, cleanup
}
