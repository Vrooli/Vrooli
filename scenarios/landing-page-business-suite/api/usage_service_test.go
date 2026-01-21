package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// createTestUsageDB creates an in-memory SQLite database for testing.
func createTestUsageDB(t *testing.T) *sql.DB {
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

		CREATE TABLE usage_records (
			id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			user_identity TEXT NOT NULL,
			billing_period TEXT NOT NULL,
			limit_key TEXT NOT NULL,
			usage_amount INTEGER NOT NULL DEFAULT 0,
			app_bundle_key TEXT,
			last_operation_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// createTestUsageService creates a usage service for testing.
func createTestUsageService(t *testing.T) (*UsageService, *LimitsService, *sql.DB) {
	t.Helper()

	db := createTestUsageDB(t)
	limitsSvc := NewLimitsService(db)

	// Create usage service without service token for basic tests (with SQLite dialect)
	usageSvc := &UsageService{
		db:           db,
		limitsSvc:    limitsSvc,
		serviceToken: "",
		dialect:      "sqlite",
	}

	return usageSvc, limitsSvc, db
}

// createTestUsageServiceWithToken creates a usage service with a service token.
func createTestUsageServiceWithToken(t *testing.T, token string) (*UsageService, *LimitsService, *sql.DB) {
	t.Helper()

	db := createTestUsageDB(t)
	limitsSvc := NewLimitsService(db)

	usageSvc := &UsageService{
		db:           db,
		limitsSvc:    limitsSvc,
		serviceToken: token,
		dialect:      "sqlite",
	}

	return usageSvc, limitsSvc, db
}

// seedTestUsageTierLimits seeds tier limits for usage tests.
func seedTestUsageTierLimits(t *testing.T, db *sql.DB) {
	t.Helper()

	limits := []struct {
		tierID     string
		limitType  string
		limitKey   string
		limitValue int64
	}{
		{"free", "cost_based", "ai_credits", 0},
		{"solo", "cost_based", "ai_credits", 500000000},   // $5
		{"pro", "cost_based", "ai_credits", 2000000000},   // $20
		{"business", "cost_based", "ai_credits", -1},      // unlimited
	}

	for _, l := range limits {
		_, err := db.Exec(`
			INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value)
			VALUES (?, ?, ?, ?)
		`, l.tierID, l.limitType, l.limitKey, l.limitValue)
		if err != nil {
			t.Fatalf("Failed to seed tier limit: %v", err)
		}
	}
}

// getCurrentBillingPeriodTest returns the current billing period for tests.
func getCurrentBillingPeriodTest() string {
	return time.Now().Format("2006-01")
}

// ============================================================================
// RecordUsage Tests
// ============================================================================

func TestUsageService_RecordUsage_InsertsNewRecord(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
	}

	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("RecordUsage() returned error: %v", err)
	}

	// Verify the record was created
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND billing_period = ? AND limit_key = ?
	`, "user@example.com", currentPeriod, "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 100000 {
		t.Errorf("Expected usage_amount 100000, got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_IncrementsExisting(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// First usage
	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
	}
	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	// Second usage
	req.Amount = 50000
	err = svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("Second RecordUsage() returned error: %v", err)
	}

	// Verify the record was incremented
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	// Should be 100000 + 50000 = 150000
	if usageAmount != 150000 {
		t.Errorf("Expected usage_amount 150000, got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_BYOK_ZeroAmount(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// BYOK request with positive amount (should be recorded as 0)
	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000, // This should be ignored because IsBYOK is true
		AppBundleKey: "browser-automation-studio",
		IsBYOK:       true,
	}

	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("RecordUsage() with BYOK returned error: %v", err)
	}

	// Verify the record was created with 0 amount
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 0 {
		t.Errorf("Expected usage_amount 0 for BYOK, got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_EmptyUserIdentity_ReturnsError(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	req := UsageReportRequest{
		UserIdentity: "",
		LimitKey:     "ai_credits",
		Amount:       100000,
	}

	err := svc.RecordUsage(ctx, req)
	if err == nil {
		t.Error("Expected error for empty user_identity, got nil")
	}
}

func TestUsageService_RecordUsage_EmptyLimitKey_ReturnsError(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "",
		Amount:       100000,
	}

	err := svc.RecordUsage(ctx, req)
	if err == nil {
		t.Error("Expected error for empty limit_key, got nil")
	}
}

func TestUsageService_RecordUsage_ZeroAmount_NonBYOK_ReturnsError(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       0, // Should error because IsBYOK is false
		IsBYOK:       false,
	}

	err := svc.RecordUsage(ctx, req)
	if err == nil {
		t.Error("Expected error for zero amount without BYOK, got nil")
	}
}

// ============================================================================
// GetUsage Tests
// ============================================================================

func TestUsageService_GetUsage_ReturnsCorrectPeriod(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed usage record
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key)
		VALUES (?, ?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 250000, "browser-automation-studio")
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get usage
	usage, err := svc.GetUsage(ctx, "user@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	if usage != 250000 {
		t.Errorf("Expected usage 250000, got %d", usage)
	}
}

func TestUsageService_GetUsage_NoRecords_ReturnsZero(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	usage, err := svc.GetUsage(ctx, "nonexistent@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	if usage != 0 {
		t.Errorf("Expected usage 0, got %d", usage)
	}
}

func TestUsageService_GetUsage_SumsAcrossApps(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed usage from multiple apps
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`,
		"user@example.com", currentPeriod, "ai_credits", 100000, "app1",
		"user@example.com", currentPeriod, "ai_credits", 50000, "app2",
	)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get total usage (nil appBundleKey should sum all)
	usage, err := svc.GetUsage(ctx, "user@example.com", "ai_credits", nil)
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	if usage != 150000 {
		t.Errorf("Expected total usage 150000, got %d", usage)
	}
}

// ============================================================================
// GetUsageSummary Tests
// ============================================================================

func TestUsageService_GetUsageSummary_CalculatesRemaining(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed some usage
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 100000000) // 100 million internal units
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get summary for 'solo' tier ($5 = 500,000,000 units)
	summary, err := svc.GetUsageSummary(ctx, "user@example.com", "solo")
	if err != nil {
		t.Fatalf("GetUsageSummary() returned error: %v", err)
	}

	// Check usage
	if summary.Usage["ai_credits"] != 100000000 {
		t.Errorf("Expected usage 100000000, got %d", summary.Usage["ai_credits"])
	}

	// Check remaining (500,000,000 - 100,000,000 = 400,000,000)
	if summary.Remaining["ai_credits"] != 400000000 {
		t.Errorf("Expected remaining 400000000, got %d", summary.Remaining["ai_credits"])
	}
}

func TestUsageService_GetUsageSummary_HandlesUnlimited(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()

	// Get summary for 'business' tier (unlimited)
	summary, err := svc.GetUsageSummary(ctx, "user@example.com", "business")
	if err != nil {
		t.Fatalf("GetUsageSummary() returned error: %v", err)
	}

	// Remaining should be -1 for unlimited
	if summary.Remaining["ai_credits"] != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", summary.Remaining["ai_credits"])
	}
}

// ============================================================================
// CheckLimit Tests
// ============================================================================

func TestUsageService_CheckLimit_AllowsBelowLimit(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed some usage (below limit)
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 100000000) // 100 million (below $5 = 500 million)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Check if user can proceed with 10 million more units
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "solo", "ai_credits", 10000000)
	if err != nil {
		t.Fatalf("CheckLimit() returned error: %v", err)
	}

	if !canProceed {
		t.Error("Expected canProceed=true for usage below limit")
	}
	if remaining != 400000000 {
		t.Errorf("Expected remaining 400000000, got %d", remaining)
	}
}

func TestUsageService_CheckLimit_DeniesOverLimit(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed usage at the limit
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 500000000) // At limit
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Check if user can proceed with any more units
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "solo", "ai_credits", 1)
	if err != nil {
		t.Fatalf("CheckLimit() returned error: %v", err)
	}

	if canProceed {
		t.Error("Expected canProceed=false for usage at limit")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", remaining)
	}
}

func TestUsageService_CheckLimit_UnlimitedTier_AlwaysAllows(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()

	// Check unlimited tier
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "business", "ai_credits", 1000000000)
	if err != nil {
		t.Fatalf("CheckLimit() returned error: %v", err)
	}

	if !canProceed {
		t.Error("Expected canProceed=true for unlimited tier")
	}
	if remaining != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", remaining)
	}
}

func TestUsageService_CheckLimit_FreeTier_DeniesAccess(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	seedTestUsageTierLimits(t, db)

	ctx := context.Background()

	// Free tier has limit 0 (no access)
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "free", "ai_credits", 1)
	if err != nil {
		t.Fatalf("CheckLimit() returned error: %v", err)
	}

	if canProceed {
		t.Error("Expected canProceed=false for free tier (0 limit)")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining 0, got %d", remaining)
	}
}

func TestUsageService_CheckLimit_NoTier_AllowsAll(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Empty tier should allow everything
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "", "ai_credits", 1000000000)
	if err != nil {
		t.Fatalf("CheckLimit() returned error: %v", err)
	}

	if !canProceed {
		t.Error("Expected canProceed=true when no tier specified")
	}
	if remaining != -1 {
		t.Errorf("Expected remaining -1 when no tier, got %d", remaining)
	}
}

// ============================================================================
// ValidateServiceToken Tests
// ============================================================================

func TestUsageService_ValidateServiceToken_ValidToken(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "secret-token-123")
	defer db.Close()

	valid := svc.ValidateServiceToken("secret-token-123")
	if !valid {
		t.Error("Expected valid=true for correct token")
	}
}

func TestUsageService_ValidateServiceToken_InvalidToken(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "secret-token-123")
	defer db.Close()

	valid := svc.ValidateServiceToken("wrong-token")
	if valid {
		t.Error("Expected valid=false for incorrect token")
	}
}

func TestUsageService_ValidateServiceToken_EmptyConfigured_RejectsAll(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "")
	defer db.Close()

	// When no token is configured, should reject all tokens
	valid := svc.ValidateServiceToken("any-token")
	if valid {
		t.Error("Expected valid=false when no service token configured")
	}

	valid = svc.ValidateServiceToken("")
	if valid {
		t.Error("Expected valid=false for empty token when none configured")
	}
}

// ============================================================================
// GetAllUsageForPeriod Tests
// ============================================================================

func TestUsageService_GetAllUsageForPeriod_ReturnsRecords(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed multiple usage records
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount, app_bundle_key)
		VALUES (?, ?, ?, ?, ?), (?, ?, ?, ?, ?)
	`,
		"user1@example.com", currentPeriod, "ai_credits", 100000, "app1",
		"user2@example.com", currentPeriod, "ai_credits", 200000, "app2",
	)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	records, err := svc.GetAllUsageForPeriod(ctx, currentPeriod)
	if err != nil {
		t.Fatalf("GetAllUsageForPeriod() returned error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("Expected 2 records, got %d", len(records))
	}
}

func TestUsageService_GetAllUsageForPeriod_EmptyPeriod_UsesCurrentMonth(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	currentPeriod := getCurrentBillingPeriodTest()

	// Seed a record for current period
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", currentPeriod, "ai_credits", 100000)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Call with empty period
	records, err := svc.GetAllUsageForPeriod(ctx, "")
	if err != nil {
		t.Fatalf("GetAllUsageForPeriod() returned error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("Expected 1 record for current period, got %d", len(records))
	}
}
