package main

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// createTestLimitsDB creates an in-memory SQLite database for testing.
func createTestLimitsDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create required tables
	schema := `
		CREATE TABLE subscription_tier_limits (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			tier_id TEXT NOT NULL,
			limit_type TEXT NOT NULL,
			limit_key TEXT NOT NULL,
			limit_value INTEGER NOT NULL,
			cost_multiplier INTEGER DEFAULT 1000000,
			app_bundle_key TEXT,
			reset_period TEXT DEFAULT 'monthly',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(tier_id, limit_type, limit_key, app_bundle_key)
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// createTestLimitsService creates a limits service for testing.
func createTestLimitsService(t *testing.T) (*LimitsService, *sql.DB) {
	t.Helper()

	db := createTestLimitsDB(t)
	svc := NewLimitsService(db, "sqlite") // Use SQLite dialect for tests

	return svc, db
}

// seedTestTierLimits seeds test data into the database.
func seedTestTierLimits(t *testing.T, db *sql.DB) {
	t.Helper()

	limits := []struct {
		tierID         string
		limitType      string
		limitKey       string
		limitValue     int64
		costMultiplier int64
		appBundleKey   *string
	}{
		{"free", "cost_based", "ai_credits", 0, 1000000, nil},
		{"solo", "cost_based", "ai_credits", 500000000, 1000000, nil},     // $5
		{"pro", "cost_based", "ai_credits", 2000000000, 1000000, nil},     // $20
		{"studio", "cost_based", "ai_credits", 10000000000, 1000000, nil}, // $100
		{"business", "cost_based", "ai_credits", -1, 1000000, nil},        // unlimited
	}

	for _, l := range limits {
		_, err := db.Exec(`
			INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
			VALUES (?, ?, ?, ?, ?, ?)
		`, l.tierID, l.limitType, l.limitKey, l.limitValue, l.costMultiplier, l.appBundleKey)
		if err != nil {
			t.Fatalf("Failed to seed tier limit: %v", err)
		}
	}
}

// ============================================================================
// GetTierLimits Tests
// ============================================================================

func TestLimitsService_GetTierLimits_ReturnsCorrectTier(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Get limits for 'solo' tier
	limits, err := svc.GetTierLimits(ctx, "solo")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 1 {
		t.Errorf("Expected 1 limit for solo tier, got %d", len(limits))
	}

	if limits[0].TierID != "solo" {
		t.Errorf("Expected tier_id 'solo', got '%s'", limits[0].TierID)
	}
	if limits[0].LimitValue != 500000000 {
		t.Errorf("Expected limit_value 500000000, got %d", limits[0].LimitValue)
	}
}

func TestLimitsService_GetTierLimits_CalculatesDisplayDollars(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Get limits for 'pro' tier ($20)
	limits, err := svc.GetTierLimits(ctx, "pro")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 1 {
		t.Errorf("Expected 1 limit, got %d", len(limits))
		return
	}

	// Check display dollars calculation
	// 2,000,000,000 / 1,000,000 / 100 = $20
	if limits[0].DisplayDollars == nil {
		t.Error("Expected DisplayDollars to be calculated")
		return
	}
	if *limits[0].DisplayDollars != 20.0 {
		t.Errorf("Expected DisplayDollars = 20.0, got %f", *limits[0].DisplayDollars)
	}
}

func TestLimitsService_GetTierLimits_NonExistentTier_ReturnsEmpty(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	limits, err := svc.GetTierLimits(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 0 {
		t.Errorf("Expected empty slice for nonexistent tier, got %d items", len(limits))
	}
}

// ============================================================================
// GetAllTierLimits Tests
// ============================================================================

func TestLimitsService_GetAllTierLimits_GroupsByTier(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	allLimits, err := svc.GetAllTierLimits(ctx)
	if err != nil {
		t.Fatalf("GetAllTierLimits() returned error: %v", err)
	}

	// Should have 5 tiers
	if len(allLimits) != 5 {
		t.Errorf("Expected 5 tier groups, got %d", len(allLimits))
	}

	// Check each tier has exactly 1 limit
	for tierID, limits := range allLimits {
		if len(limits) != 1 {
			t.Errorf("Expected 1 limit for tier %s, got %d", tierID, len(limits))
		}
	}
}

// ============================================================================
// GetLimit Tests
// ============================================================================

func TestLimitsService_GetLimit_ReturnsSpecificLimit(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	limit, err := svc.GetLimit(ctx, "studio", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetLimit() returned error: %v", err)
	}

	if limit == nil {
		t.Fatal("Expected limit, got nil")
	}

	if limit.TierID != "studio" {
		t.Errorf("Expected tier_id 'studio', got '%s'", limit.TierID)
	}
	if limit.LimitValue != 10000000000 {
		t.Errorf("Expected limit_value 10000000000, got %d", limit.LimitValue)
	}
}

func TestLimitsService_GetLimit_NonExistent_ReturnsNil(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	limit, err := svc.GetLimit(ctx, "nonexistent", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetLimit() returned error: %v", err)
	}

	if limit != nil {
		t.Error("Expected nil for nonexistent limit")
	}
}

// ============================================================================
// UpdateLimit Tests
// ============================================================================

func TestLimitsService_UpdateLimit_WithDisplayDollars(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Update solo tier from $5 to $10
	newDollars := 10.0
	update := TierLimitUpdate{DisplayDollars: &newDollars}

	limit, err := svc.UpdateLimit(ctx, "solo", "ai_credits", nil, update)
	if err != nil {
		t.Fatalf("UpdateLimit() returned error: %v", err)
	}

	// $10 = 10 * 100 * 1,000,000 = 1,000,000,000 internal units
	expectedValue := int64(1000000000)
	if limit.LimitValue != expectedValue {
		t.Errorf("Expected limit_value %d, got %d", expectedValue, limit.LimitValue)
	}
}

func TestLimitsService_UpdateLimit_SetUnlimited(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Set solo tier to unlimited
	isUnlimited := true
	update := TierLimitUpdate{IsUnlimited: &isUnlimited}

	limit, err := svc.UpdateLimit(ctx, "solo", "ai_credits", nil, update)
	if err != nil {
		t.Fatalf("UpdateLimit() returned error: %v", err)
	}

	if limit.LimitValue != -1 {
		t.Errorf("Expected limit_value -1 (unlimited), got %d", limit.LimitValue)
	}
}

func TestLimitsService_UpdateLimit_WithLimitValue(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Set specific internal unit value
	newValue := int64(123456789)
	update := TierLimitUpdate{LimitValue: &newValue}

	limit, err := svc.UpdateLimit(ctx, "solo", "ai_credits", nil, update)
	if err != nil {
		t.Fatalf("UpdateLimit() returned error: %v", err)
	}

	if limit.LimitValue != newValue {
		t.Errorf("Expected limit_value %d, got %d", newValue, limit.LimitValue)
	}
}

func TestLimitsService_UpdateLimit_NonExistent_ReturnsError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newValue := int64(100)
	update := TierLimitUpdate{LimitValue: &newValue}

	_, err := svc.UpdateLimit(ctx, "nonexistent", "ai_credits", nil, update)
	if err == nil {
		t.Error("Expected error for nonexistent limit, got nil")
	}
}

// ============================================================================
// CreateLimit Tests
// ============================================================================

func TestLimitsService_CreateLimit_InsertsNew(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newLimit := TierLimit{
		TierID:         "enterprise",
		LimitType:      "cost_based",
		LimitKey:       "ai_credits",
		LimitValue:     50000000000, // $500
		CostMultiplier: 1000000,
		ResetPeriod:    "monthly",
	}

	limit, err := svc.CreateLimit(ctx, newLimit)
	if err != nil {
		t.Fatalf("CreateLimit() returned error: %v", err)
	}

	if limit.TierID != "enterprise" {
		t.Errorf("Expected tier_id 'enterprise', got '%s'", limit.TierID)
	}
	if limit.LimitValue != 50000000000 {
		t.Errorf("Expected limit_value 50000000000, got %d", limit.LimitValue)
	}

	// Verify in database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM subscription_tier_limits WHERE tier_id = ?", "enterprise").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 row in database, got %d", count)
	}
}

func TestLimitsService_CreateLimit_InvalidLimitType_ReturnsError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newLimit := TierLimit{
		TierID:    "test",
		LimitType: "invalid_type",
		LimitKey:  "ai_credits",
	}

	_, err := svc.CreateLimit(ctx, newLimit)
	if err == nil {
		t.Error("Expected error for invalid limit type, got nil")
	}
}

func TestLimitsService_CreateLimit_DefaultsMultiplier(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newLimit := TierLimit{
		TierID:     "test",
		LimitType:  "cost_based",
		LimitKey:   "ai_credits",
		LimitValue: 100,
		// CostMultiplier not set
	}

	limit, err := svc.CreateLimit(ctx, newLimit)
	if err != nil {
		t.Fatalf("CreateLimit() returned error: %v", err)
	}

	// Should default to 1,000,000
	if limit.CostMultiplier != 1000000 {
		t.Errorf("Expected default cost_multiplier 1000000, got %d", limit.CostMultiplier)
	}
}

// ============================================================================
// DeleteLimit Tests
// ============================================================================

func TestLimitsService_DeleteLimit_RemovesRecord(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Verify limit exists
	var countBefore int
	err := db.QueryRow("SELECT COUNT(*) FROM subscription_tier_limits WHERE tier_id = ?", "solo").Scan(&countBefore)
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("Expected 1 limit before delete, got %d", countBefore)
	}

	// Delete the limit
	err = svc.DeleteLimit(ctx, "solo", "ai_credits", nil)
	if err != nil {
		t.Fatalf("DeleteLimit() returned error: %v", err)
	}

	// Verify it's gone
	var countAfter int
	err = db.QueryRow("SELECT COUNT(*) FROM subscription_tier_limits WHERE tier_id = ?", "solo").Scan(&countAfter)
	if err != nil {
		t.Fatalf("Failed to count: %v", err)
	}
	if countAfter != 0 {
		t.Errorf("Expected 0 limits after delete, got %d", countAfter)
	}
}

func TestLimitsService_DeleteLimit_NonExistent_ReturnsError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	err := svc.DeleteLimit(ctx, "nonexistent", "ai_credits", nil)
	if err == nil {
		t.Error("Expected error for deleting nonexistent limit, got nil")
	}
}

// ============================================================================
// GetAppLimits Tests
// ============================================================================

func TestLimitsService_GetAppLimits_FiltersCorrectly(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	// Seed app-specific limits
	appKey := "browser-automation-studio"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'app_specific', 'workflow_exports', 10, 1, ?),
		       ('pro', 'app_specific', 'workflow_exports', 50, 1, ?)
	`, appKey, appKey)
	if err != nil {
		t.Fatalf("Failed to seed app limits: %v", err)
	}

	// Also seed a limit for a different app
	_, err = db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'app_specific', 'exports', 5, 1, 'other-app')
	`)
	if err != nil {
		t.Fatalf("Failed to seed other app limits: %v", err)
	}

	// Get limits for our app
	limits, err := svc.GetAppLimits(ctx, appKey)
	if err != nil {
		t.Fatalf("GetAppLimits() returned error: %v", err)
	}

	// Should have 2 tiers with limits
	totalLimits := 0
	for _, tierLimits := range limits {
		totalLimits += len(tierLimits)
	}

	if totalLimits != 2 {
		t.Errorf("Expected 2 total limits for app, got %d", totalLimits)
	}
}

// ============================================================================
// Conversion Utility Tests
// ============================================================================

func TestDollarsToInternalUnits(t *testing.T) {
	testCases := []struct {
		dollars  float64
		expected int64
	}{
		{5.0, 500000000},
		{20.0, 2000000000},
		{100.0, 10000000000},
		{0.01, 1000000}, // 1 cent
		{0.0, 0},
	}

	for _, tc := range testCases {
		result := DollarsToInternalUnits(tc.dollars)
		if result != tc.expected {
			t.Errorf("DollarsToInternalUnits(%f) = %d, expected %d", tc.dollars, result, tc.expected)
		}
	}
}

func TestInternalUnitsToDollars(t *testing.T) {
	testCases := []struct {
		units    int64
		expected float64
	}{
		{500000000, 5.0},
		{2000000000, 20.0},
		{10000000000, 100.0},
		{1000000, 0.01}, // 1 cent
		{0, 0.0},
	}

	for _, tc := range testCases {
		result := InternalUnitsToDollars(tc.units)
		if result != tc.expected {
			t.Errorf("InternalUnitsToDollars(%d) = %f, expected %f", tc.units, result, tc.expected)
		}
	}
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func TestLimitsService_GetLimit_WithAppBundleKey_FiltersCorrectly(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	// Seed general and app-specific limits
	appKey := "test-app"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'cost_based', 'ai_credits', 500000000, 1000000, NULL),
		       ('solo', 'app_specific', 'ai_credits', 100000000, 1000000, ?)
	`, appKey)
	if err != nil {
		t.Fatalf("Failed to seed limits: %v", err)
	}

	// Get app-specific limit
	limit, err := svc.GetLimit(ctx, "solo", "ai_credits", &appKey)
	if err != nil {
		t.Fatalf("GetLimit() returned error: %v", err)
	}

	if limit == nil {
		t.Fatal("Expected limit, got nil")
	}
	if limit.LimitValue != 100000000 {
		t.Errorf("Expected app-specific limit 100000000, got %d", limit.LimitValue)
	}
}

func TestLimitsService_UpdateLimit_EmptyUpdate_ReturnsValidationError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Empty update (no fields set)
	update := TierLimitUpdate{}

	_, err := svc.UpdateLimit(ctx, "solo", "ai_credits", nil, update)
	if err == nil {
		t.Error("Expected validation error for empty update, got nil")
	}
}

func TestLimitsService_CreateLimit_EmptyTierID_ReturnsValidationError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newLimit := TierLimit{
		TierID:    "", // Empty
		LimitType: "cost_based",
		LimitKey:  "ai_credits",
	}

	_, err := svc.CreateLimit(ctx, newLimit)
	if err == nil {
		t.Error("Expected validation error for empty tier_id, got nil")
	}
}

func TestLimitsService_CreateLimit_EmptyLimitKey_ReturnsValidationError(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	newLimit := TierLimit{
		TierID:    "test",
		LimitType: "cost_based",
		LimitKey:  "", // Empty
	}

	_, err := svc.CreateLimit(ctx, newLimit)
	if err == nil {
		t.Error("Expected validation error for empty limit_key, got nil")
	}
}

func TestLimitsService_CreateLimit_WithAllFields(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	// Create limit with all fields specified
	appKey := "test-bundle"
	newLimit := TierLimit{
		TierID:         "enterprise",
		LimitType:      "cost_based",
		LimitKey:       "api_calls",
		LimitValue:     999999999,
		CostMultiplier: 500000, // Custom multiplier
		ResetPeriod:    "weekly",
		AppBundleKey:   &appKey,
	}

	limit, err := svc.CreateLimit(ctx, newLimit)
	if err != nil {
		t.Fatalf("CreateLimit() returned error: %v", err)
	}

	if limit.TierID != "enterprise" {
		t.Errorf("Expected tier_id 'enterprise', got '%s'", limit.TierID)
	}
	if limit.LimitKey != "api_calls" {
		t.Errorf("Expected limit_key 'api_calls', got '%s'", limit.LimitKey)
	}
}

func TestLimitsService_DeleteLimit_WithAppBundleKey_FiltersCorrectly(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	// Seed both general and app-specific limits
	appKey := "test-app"
	_, err := db.Exec(`
		INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value, cost_multiplier, app_bundle_key)
		VALUES ('solo', 'cost_based', 'ai_credits', 500000000, 1000000, NULL),
		       ('solo', 'app_specific', 'ai_credits', 100000000, 1000000, ?)
	`, appKey)
	if err != nil {
		t.Fatalf("Failed to seed limits: %v", err)
	}

	// Delete only the app-specific limit
	err = svc.DeleteLimit(ctx, "solo", "ai_credits", &appKey)
	if err != nil {
		t.Fatalf("DeleteLimit() returned error: %v", err)
	}

	// General limit should still exist
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM subscription_tier_limits WHERE tier_id = 'solo' AND app_bundle_key IS NULL`).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected general limit to still exist, got count %d", count)
	}
}

func TestLimitsService_GetAppLimits_EmptyResult_ReturnsEmptyMap(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	// Query for non-existent app
	limits, err := svc.GetAppLimits(ctx, "nonexistent-app")
	if err != nil {
		t.Fatalf("GetAppLimits() returned error: %v", err)
	}

	if limits == nil {
		t.Error("Expected non-nil map, got nil")
	}
	if len(limits) != 0 {
		t.Errorf("Expected empty map, got %d entries", len(limits))
	}
}

func TestLimitsService_DisplayDollars_NegativeLimit_ReturnsNil(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Get unlimited tier (limit_value = -1)
	limits, err := svc.GetTierLimits(ctx, "business")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 1 {
		t.Fatalf("Expected 1 limit, got %d", len(limits))
	}

	// DisplayDollars should be nil for unlimited (-1)
	if limits[0].DisplayDollars != nil {
		t.Errorf("Expected DisplayDollars to be nil for unlimited tier, got %f", *limits[0].DisplayDollars)
	}
}

func TestLimitsService_DisplayDollars_ZeroLimit_ReturnsNil(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Get free tier (limit_value = 0)
	limits, err := svc.GetTierLimits(ctx, "free")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 1 {
		t.Fatalf("Expected 1 limit, got %d", len(limits))
	}

	// DisplayDollars should be nil for zero limit
	if limits[0].DisplayDollars != nil {
		t.Errorf("Expected DisplayDollars to be nil for free tier (0 limit), got %f", *limits[0].DisplayDollars)
	}
}

func TestLimitsService_GetTierLimits_UnlimitedTier_ReturnsCorrectValues(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	limits, err := svc.GetTierLimits(ctx, "business")
	if err != nil {
		t.Fatalf("GetTierLimits() returned error: %v", err)
	}

	if len(limits) != 1 {
		t.Fatalf("Expected 1 limit, got %d", len(limits))
	}

	if limits[0].LimitValue != -1 {
		t.Errorf("Expected limit_value -1 for unlimited, got %d", limits[0].LimitValue)
	}
}

func TestLimitsService_CreateLimit_AppSpecificType(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	ctx := context.Background()

	appKey := "my-app"
	newLimit := TierLimit{
		TierID:       "test",
		LimitType:    "app_specific",
		LimitKey:     "custom_feature",
		LimitValue:   100,
		AppBundleKey: &appKey,
	}

	limit, err := svc.CreateLimit(ctx, newLimit)
	if err != nil {
		t.Fatalf("CreateLimit() returned error: %v", err)
	}

	if limit.LimitType != "app_specific" {
		t.Errorf("Expected limit_type 'app_specific', got '%s'", limit.LimitType)
	}
}

func TestLimitsService_GetAllTierLimits_IncludesAllFields(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	allLimits, err := svc.GetAllTierLimits(ctx)
	if err != nil {
		t.Fatalf("GetAllTierLimits() returned error: %v", err)
	}

	// Check that pro tier has all expected fields
	proLimits, ok := allLimits["pro"]
	if !ok {
		t.Fatal("Expected 'pro' tier in results")
	}

	if len(proLimits) != 1 {
		t.Fatalf("Expected 1 limit for pro tier, got %d", len(proLimits))
	}

	limit := proLimits[0]
	if limit.TierID != "pro" {
		t.Errorf("Expected tier_id 'pro', got '%s'", limit.TierID)
	}
	if limit.LimitType != "cost_based" {
		t.Errorf("Expected limit_type 'cost_based', got '%s'", limit.LimitType)
	}
	if limit.CostMultiplier != 1000000 {
		t.Errorf("Expected cost_multiplier 1000000, got %d", limit.CostMultiplier)
	}
}

func TestLimitsService_UpdateLimit_PriorityIsUnlimitedOverDollars(t *testing.T) {
	svc, db := createTestLimitsService(t)
	defer db.Close()

	seedTestTierLimits(t, db)

	ctx := context.Background()

	// Set both is_unlimited and display_dollars - unlimited should take priority
	isUnlimited := true
	dollars := 50.0
	update := TierLimitUpdate{
		IsUnlimited:    &isUnlimited,
		DisplayDollars: &dollars,
	}

	limit, err := svc.UpdateLimit(ctx, "solo", "ai_credits", nil, update)
	if err != nil {
		t.Fatalf("UpdateLimit() returned error: %v", err)
	}

	// IsUnlimited should take priority
	if limit.LimitValue != -1 {
		t.Errorf("Expected limit_value -1 (unlimited takes priority), got %d", limit.LimitValue)
	}
}
