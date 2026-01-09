package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// TestScenarioOverrideQueries contains integration tests for the scenario override system.
// These tests require a PostgreSQL database connection to run.
//
// Run with: go test -v -run TestScenarioOverride ./...
//
// For unit testing without a database, these tests will be skipped.

// mockTestLogger implements Logger interface for testing
type mockTestLogger struct{}

func (m *mockTestLogger) Debug(msg string, fields ...any) {}
func (m *mockTestLogger) Info(msg string, fields ...any)  {}
func (m *mockTestLogger) Warn(msg string, fields ...any)  {}
func (m *mockTestLogger) Error(msg string, fields ...any) {}

// TestUpsertOverrideCreatesNew tests creating a new override
func TestUpsertOverrideCreatesNew(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	logger := &mockTestLogger{}
	store := NewScenarioOverrideStore(db, &Logger{})
	_ = logger // use test logger for assertions
	ctx := context.Background()

	// Setup: Create a resource and secret first
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)

	// Test creating a new override
	handling := "strip"
	reason := "Test scenario needs different handling"
	override, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
		OverrideReason:   &reason,
	})

	if err != nil {
		t.Fatalf("UpsertOverride failed: %v", err)
	}

	if override.ScenarioName != "test-scenario" {
		t.Errorf("expected ScenarioName 'test-scenario', got '%s'", override.ScenarioName)
	}
	if override.HandlingStrategy == nil || *override.HandlingStrategy != "strip" {
		t.Error("expected HandlingStrategy 'strip'")
	}
	if override.OverrideReason == nil || *override.OverrideReason != reason {
		t.Errorf("expected OverrideReason '%s'", reason)
	}

	// Cleanup
	cleanupTestData(t, db, resourceSecretID, override.ID)
}

// TestUpsertOverrideUpdatesExisting tests updating an existing override
func TestUpsertOverrideUpdatesExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Setup
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)

	// Create initial override
	handling := "strip"
	override1, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("first UpsertOverride failed: %v", err)
	}

	// Update the override
	newHandling := "generate"
	newReason := "Changed requirements"
	override2, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &newHandling,
		OverrideReason:   &newReason,
	})
	if err != nil {
		t.Fatalf("second UpsertOverride failed: %v", err)
	}

	// Should be same ID (update, not insert)
	if override2.ID != override1.ID {
		t.Errorf("expected same ID on update, got different IDs: %s vs %s", override1.ID, override2.ID)
	}

	// Values should be updated
	if override2.HandlingStrategy == nil || *override2.HandlingStrategy != "generate" {
		t.Error("expected HandlingStrategy 'generate' after update")
	}
	if override2.OverrideReason == nil || *override2.OverrideReason != newReason {
		t.Errorf("expected OverrideReason '%s' after update", newReason)
	}

	// Cleanup
	cleanupTestData(t, db, resourceSecretID, override2.ID)
}

// TestFetchOverridesByScenarioTier tests fetching overrides for a scenario+tier
func TestFetchOverridesByScenarioTier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Setup: Create multiple overrides
	resourceSecretID1, resourceName1, secretKey1 := setupTestResourceSecret(t, db)
	resourceSecretID2, resourceName2, secretKey2 := setupTestResourceSecretWithKey(t, db, "OTHER_SECRET")

	handling1 := "strip"
	handling2 := "prompt"
	prompt := "Enter value"

	o1, _ := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName1, secretKey1, SetOverrideRequest{
		HandlingStrategy: &handling1,
	})
	o2, _ := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName2, secretKey2, SetOverrideRequest{
		HandlingStrategy: &handling2,
		PromptLabel:      &prompt,
	})

	// Fetch overrides
	overrides, err := store.FetchOverridesByScenarioTier(ctx, "test-scenario", "tier-2-desktop")
	if err != nil {
		t.Fatalf("FetchOverridesByScenarioTier failed: %v", err)
	}

	if len(overrides) < 2 {
		t.Errorf("expected at least 2 overrides, got %d", len(overrides))
	}

	// Cleanup
	cleanupTestData(t, db, resourceSecretID1, o1.ID)
	cleanupTestData(t, db, resourceSecretID2, o2.ID)
}

// TestDeleteOverride tests deleting an override
func TestDeleteOverride(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Setup
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)

	handling := "strip"
	_, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("UpsertOverride failed: %v", err)
	}

	// Delete the override
	err = store.DeleteOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey)
	if err != nil {
		t.Fatalf("DeleteOverride failed: %v", err)
	}

	// Verify deletion
	override, err := store.FetchOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey)
	if err != nil {
		t.Fatalf("FetchOverride failed: %v", err)
	}
	if override != nil {
		t.Error("expected nil override after deletion")
	}

	// Cleanup remaining resource
	_, _ = db.Exec("DELETE FROM resource_secrets WHERE id = $1", resourceSecretID)
}

// TestCopyOverridesFromTier tests copying overrides between tiers
func TestCopyOverridesFromTier(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Setup: Create override in source tier
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)

	handling := "generate"
	o1, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("UpsertOverride failed: %v", err)
	}

	// Copy to target tier
	copied, err := store.CopyOverridesFromTier(ctx, "test-scenario", "tier-2-desktop", "tier-3-mobile", false)
	if err != nil {
		t.Fatalf("CopyOverridesFromTier failed: %v", err)
	}

	if copied < 1 {
		t.Errorf("expected at least 1 copied override, got %d", copied)
	}

	// Verify target tier has the override
	overrides, err := store.FetchOverridesByScenarioTier(ctx, "test-scenario", "tier-3-mobile")
	if err != nil {
		t.Fatalf("FetchOverridesByScenarioTier failed: %v", err)
	}
	if len(overrides) < 1 {
		t.Error("expected override to exist in target tier")
	}

	// Cleanup
	_, _ = db.Exec("DELETE FROM scenario_secret_strategy_overrides WHERE scenario_name = $1", "test-scenario")
	_, _ = db.Exec("DELETE FROM resource_secrets WHERE id = $1", resourceSecretID)
	_ = o1 // silence unused warning
}

// TestFetchAllOverrides tests fetching all overrides
func TestFetchAllOverrides(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// This test verifies the query runs without error
	overrides, err := store.FetchAllOverrides(ctx)
	if err != nil {
		t.Fatalf("FetchAllOverrides failed: %v", err)
	}

	// Result should be a valid slice (possibly empty)
	if overrides == nil {
		t.Error("expected non-nil result from FetchAllOverrides")
	}
}

// TestDeleteOverridesByID tests deleting overrides by ID
func TestDeleteOverridesByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Setup
	resourceSecretID, resourceName, secretKey := setupTestResourceSecret(t, db)

	handling := "strip"
	override, err := store.UpsertOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey, SetOverrideRequest{
		HandlingStrategy: &handling,
	})
	if err != nil {
		t.Fatalf("UpsertOverride failed: %v", err)
	}

	// Delete by ID
	deleted, err := store.DeleteOverridesByID(ctx, []string{override.ID})
	if err != nil {
		t.Fatalf("DeleteOverridesByID failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", deleted)
	}

	// Verify deletion
	fetchedOverride, err := store.FetchOverride(ctx, "test-scenario", "tier-2-desktop", resourceName, secretKey)
	if err != nil {
		t.Fatalf("FetchOverride failed: %v", err)
	}
	if fetchedOverride != nil {
		t.Error("expected nil override after deletion")
	}

	// Cleanup resource
	_, _ = db.Exec("DELETE FROM resource_secrets WHERE id = $1", resourceSecretID)
}

// TestDeleteOverridesByIDEmpty tests deleting with empty slice
func TestDeleteOverridesByIDEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db := getTestDB(t)
	if db == nil {
		t.Skip("no database connection available")
	}
	defer db.Close()

	store := NewScenarioOverrideStore(db, &Logger{})
	ctx := context.Background()

	// Should return 0, no error
	deleted, err := store.DeleteOverridesByID(ctx, []string{})
	if err != nil {
		t.Fatalf("DeleteOverridesByID with empty slice failed: %v", err)
	}
	if deleted != 0 {
		t.Errorf("expected 0 deleted for empty input, got %d", deleted)
	}
}

// Test helper functions

func getTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Try to connect to the test database
	connStr := "host=localhost port=5432 user=secrets_manager password=dev_password dbname=secrets_manager_test sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Logf("could not open database: %v", err)
		return nil
	}

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Logf("could not ping database: %v", err)
		db.Close()
		return nil
	}

	return db
}

func setupTestResourceSecret(t *testing.T, db *sql.DB) (id, resourceName, secretKey string) {
	return setupTestResourceSecretWithKey(t, db, "TEST_SECRET_KEY")
}

func setupTestResourceSecretWithKey(t *testing.T, db *sql.DB, secretKey string) (id, resourceName, key string) {
	t.Helper()

	resourceName = "test-resource"
	key = secretKey

	// Check if the secret already exists and delete it
	_, _ = db.Exec("DELETE FROM resource_secrets WHERE resource_name = $1 AND secret_key = $2", resourceName, key)

	err := db.QueryRow(`
		INSERT INTO resource_secrets (resource_name, secret_key, secret_type, classification, required)
		VALUES ($1, $2, 'credential', 'infrastructure', true)
		RETURNING id
	`, resourceName, key).Scan(&id)

	if err != nil {
		t.Fatalf("failed to setup test resource secret: %v", err)
	}

	return id, resourceName, key
}

func setupTestResourceSecretWithStrategy(t *testing.T, db *sql.DB, handling, promptLabel string) (id, resourceName, secretKey string) {
	t.Helper()

	resourceName = "test-resource"
	secretKey = "TEST_SECRET_WITH_STRATEGY"

	// Clean up any existing data
	_, _ = db.Exec("DELETE FROM resource_secrets WHERE resource_name = $1 AND secret_key = $2", resourceName, secretKey)

	tierStrategies := map[string]string{"tier-2-desktop": handling}
	strategiesJSON, _ := json.Marshal(tierStrategies)

	err := db.QueryRow(`
		INSERT INTO resource_secrets (resource_name, secret_key, secret_type, classification, required, tier_strategies)
		VALUES ($1, $2, 'credential', 'infrastructure', true, $3)
		RETURNING id
	`, resourceName, secretKey, strategiesJSON).Scan(&id)

	if err != nil {
		t.Fatalf("failed to setup test resource secret with strategy: %v", err)
	}

	return id, resourceName, secretKey
}

func cleanupTestData(t *testing.T, db *sql.DB, resourceSecretID, overrideID string) {
	t.Helper()

	if overrideID != "" {
		_, _ = db.Exec("DELETE FROM scenario_secret_strategy_overrides WHERE id = $1", overrideID)
	}
	if resourceSecretID != "" {
		_, _ = db.Exec("DELETE FROM resource_secrets WHERE id = $1", resourceSecretID)
	}
}
