package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// createTestUsageDB creates an in-memory SQLite database for testing.
func createTestUsageDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use shared-cache mode for in-memory SQLite to enable concurrent access
	dsn := fmt.Sprintf("file:usage_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// SQLite in-memory databases require single connection to preserve state
	db.SetMaxOpenConns(1)

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
			operation_id TEXT,
			last_operation_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
		);

		CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_operation_id ON usage_records(operation_id) WHERE operation_id IS NOT NULL;

		CREATE TABLE IF NOT EXISTS credit_reservations (
			id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			billing_period TEXT NOT NULL,
			limit_key TEXT NOT NULL,
			reserved_amount INTEGER NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'finalized', 'released', 'expired')),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			finalized_at TIMESTAMP,
			expires_at TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_credit_reservations_user ON credit_reservations(user_identity, status);
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
	limitsSvc := NewLimitsService(db, "sqlite")

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
	limitsSvc := NewLimitsService(db, "sqlite")

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
		{"solo", "cost_based", "ai_credits", 500000000}, // $5
		{"pro", "cost_based", "ai_credits", 2000000000}, // $20
		{"business", "cost_based", "ai_credits", -1},    // unlimited
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

// MockLimitsService implements LimitsServicer for testing.
type MockLimitsService struct {
	GetLimitFn      func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error)
	GetTierLimitsFn func(ctx context.Context, tierID string) ([]TierLimit, error)
	GetLimitCalls   []struct {
		TierID       string
		LimitKey     string
		AppBundleKey *string
	}
}

func (m *MockLimitsService) GetLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
	m.GetLimitCalls = append(m.GetLimitCalls, struct {
		TierID       string
		LimitKey     string
		AppBundleKey *string
	}{tierID, limitKey, appBundleKey})
	if m.GetLimitFn != nil {
		return m.GetLimitFn(ctx, tierID, limitKey, appBundleKey)
	}
	// Default: unlimited
	return &TierLimit{LimitValue: -1, LimitType: "cost_based", CostMultiplier: 1000000}, nil
}

func (m *MockLimitsService) GetTierLimits(ctx context.Context, tierID string) ([]TierLimit, error) {
	if m.GetTierLimitsFn != nil {
		return m.GetTierLimitsFn(ctx, tierID)
	}
	return []TierLimit{}, nil
}

// Compile-time check
var _ LimitsServicer = (*MockLimitsService)(nil)

// createTestUsageServiceWithMock creates a usage service with a mock limits service.
func createTestUsageServiceWithMock(t *testing.T, mock *MockLimitsService) (*UsageService, *sql.DB) {
	t.Helper()

	db := createTestUsageDB(t)

	usageSvc := &UsageService{
		db:           db,
		limitsSvc:    mock,
		serviceToken: "",
		dialect:      "sqlite",
	}

	return usageSvc, db
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

// TestUsageService_ValidateServiceToken_ConstantTime tests that token validation
// works correctly with constant-time comparison (functional test, not timing test).
// This verifies that subtle.ConstantTimeCompare is used correctly.
func TestUsageService_ValidateServiceToken_ConstantTime(t *testing.T) {
	// Test with various token lengths to ensure constant-time compare works
	testCases := []struct {
		name       string
		configured string
		provided   string
		expectOK   bool
	}{
		{"exact match", "secret-token-123", "secret-token-123", true},
		{"same length different content", "secret-token-123", "secret-token-456", false},
		{"shorter provided", "secret-token-123", "short", false},
		{"longer provided", "secret-token-123", "secret-token-123-extra-long", false},
		{"prefix match only", "secret-token-123", "secret-token", false},
		{"suffix match only", "secret-token-123", "token-123", false},
		{"case sensitive", "Secret-Token-123", "secret-token-123", false},
		{"empty provided", "secret-token-123", "", false},
		{"single char match", "x", "x", true},
		{"single char mismatch", "x", "y", false},
		{"unicode tokens", "tökën-123", "tökën-123", true},
		{"unicode mismatch", "tökën-123", "token-123", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, db := createTestUsageServiceWithToken(t, tc.configured)
			defer db.Close()

			valid := svc.ValidateServiceToken(tc.provided)
			if valid != tc.expectOK {
				t.Errorf("ValidateServiceToken(%q) = %v, expected %v (configured: %q)",
					tc.provided, valid, tc.expectOK, tc.configured)
			}
		})
	}
}

func TestUsageService_HealthCheck_ServiceAuthConfigured_True(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "service-secret")
	defer db.Close()

	status, err := svc.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() returned error: %v", err)
	}

	if !status.ServiceAuthConfigured {
		t.Fatal("expected service_auth_configured=true")
	}
	if status.ServiceAuthMode != "token" {
		t.Fatalf("expected service_auth_mode=token, got %q", status.ServiceAuthMode)
	}
}

func TestUsageService_HealthCheck_ServiceAuthConfigured_False(t *testing.T) {
	svc, _, db := createTestUsageServiceWithToken(t, "")
	defer db.Close()

	status, err := svc.HealthCheck(context.Background())
	if err != nil {
		t.Fatalf("HealthCheck() returned error: %v", err)
	}

	if status.ServiceAuthConfigured {
		t.Fatal("expected service_auth_configured=false")
	}
	if status.ServiceAuthMode != "disabled" {
		t.Fatalf("expected service_auth_mode=disabled, got %q", status.ServiceAuthMode)
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

// ============================================================================
// Idempotency Tests
// ============================================================================

func TestUsageService_RecordUsage_WithOperationID_FirstTime_RecordsUsage(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	operationID := "test-operation-id-12345"

	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID,
	}

	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("RecordUsage() returned error: %v", err)
	}

	// Verify the record was created with correct amount
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 100000 {
		t.Errorf("Expected usage_amount 100000, got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_WithOperationID_Duplicate_NoIncrement(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	operationID := "test-idempotent-operation-12345"

	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID,
	}

	// First call - should record usage
	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	// Second call with SAME operation_id - should NOT increment
	err = svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("Second RecordUsage() returned error: %v", err)
	}

	// Third call with SAME operation_id - should NOT increment
	err = svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("Third RecordUsage() returned error: %v", err)
	}

	// Verify the usage was NOT incremented (should still be 100000, not 300000)
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 100000 {
		t.Errorf("Expected usage_amount 100000 (no duplicate increment), got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_WithOperationID_DifferentUser_BothRecorded(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	operationID1 := "operation-user1-12345"
	operationID2 := "operation-user2-67890"

	// First user
	req1 := UsageReportRequest{
		UserIdentity: "user1@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID1,
	}
	err := svc.RecordUsage(ctx, req1)
	if err != nil {
		t.Fatalf("RecordUsage() for user1 returned error: %v", err)
	}

	// Second user with different operation_id
	req2 := UsageReportRequest{
		UserIdentity: "user2@example.com",
		LimitKey:     "ai_credits",
		Amount:       50000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID2,
	}
	err = svc.RecordUsage(ctx, req2)
	if err != nil {
		t.Fatalf("RecordUsage() for user2 returned error: %v", err)
	}

	// Verify both users have correct usage
	var usage1, usage2 int64
	err = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ?`, "user1@example.com").Scan(&usage1)
	if err != nil {
		t.Fatalf("Failed to query user1 usage: %v", err)
	}
	err = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ?`, "user2@example.com").Scan(&usage2)
	if err != nil {
		t.Fatalf("Failed to query user2 usage: %v", err)
	}

	if usage1 != 100000 {
		t.Errorf("Expected user1 usage 100000, got %d", usage1)
	}
	if usage2 != 50000 {
		t.Errorf("Expected user2 usage 50000, got %d", usage2)
	}
}

func TestUsageService_RecordUsage_NoOperationID_BackwardCompatible(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Request without operation_id (backward compatibility)
	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		// No OperationID set
	}

	// First call
	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	// Second call without operation_id - should increment (old behavior)
	err = svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("Second RecordUsage() returned error: %v", err)
	}

	// Verify both were recorded (no idempotency without operation_id)
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 200000 {
		t.Errorf("Expected usage_amount 200000 (both calls recorded), got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_EmptyOperationID_TreatedAsNil(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	emptyID := ""

	// Request with empty operation_id (should behave like nil)
	req := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &emptyID,
	}

	// First call
	err := svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	// Second call - should increment since empty operation_id is treated as nil
	err = svc.RecordUsage(ctx, req)
	if err != nil {
		t.Fatalf("Second RecordUsage() returned error: %v", err)
	}

	// Verify both were recorded
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	if usageAmount != 200000 {
		t.Errorf("Expected usage_amount 200000 (both calls recorded), got %d", usageAmount)
	}
}

func TestUsageService_RecordUsage_DifferentOperationIDs_BothRecorded(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	operationID1 := "operation-1-12345"
	operationID2 := "operation-2-67890"

	// Same user, different operation_ids
	req1 := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID1,
	}
	err := svc.RecordUsage(ctx, req1)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	req2 := UsageReportRequest{
		UserIdentity: "user@example.com",
		LimitKey:     "ai_credits",
		Amount:       50000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID2,
	}
	err = svc.RecordUsage(ctx, req2)
	if err != nil {
		t.Fatalf("Second RecordUsage() returned error: %v", err)
	}

	// Verify both amounts were added
	var usageAmount int64
	err = db.QueryRow(`
		SELECT usage_amount FROM usage_records
		WHERE user_identity = ? AND limit_key = ?
	`, "user@example.com", "ai_credits").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	// Should be 150000 (100000 + 50000)
	if usageAmount != 150000 {
		t.Errorf("Expected usage_amount 150000, got %d", usageAmount)
	}
}

// ============================================================================
// ReserveAndCharge Tests
// ============================================================================

func TestUsageService_ReserveAndCharge_Success_WithSufficientCredits(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: 500000000, LimitType: "cost_based", CostMultiplier: 1000000}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	err := svc.ReserveAndCharge(ctx, "user@example.com", "solo", "ai_credits", 100000000, UsageReportRequest{
		AppBundleKey: "test-app",
	})
	if err != nil {
		t.Fatalf("ReserveAndCharge() returned error: %v", err)
	}

	// Verify usage was recorded
	var usageAmount int64
	err = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ?`, "user@example.com").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage: %v", err)
	}
	if usageAmount != 100000000 {
		t.Errorf("Expected usage 100000000, got %d", usageAmount)
	}
}

func TestUsageService_ReserveAndCharge_InsufficientCredits_ReturnsError(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: 100000000, LimitType: "cost_based", CostMultiplier: 1000000}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Try to charge more than limit
	err := svc.ReserveAndCharge(ctx, "user@example.com", "solo", "ai_credits", 200000000, UsageReportRequest{})
	if err == nil {
		t.Error("Expected ErrInsufficientCredits, got nil")
	}
}

func TestUsageService_ReserveAndCharge_EmptyUserIdentity_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.ReserveAndCharge(ctx, "", "solo", "ai_credits", 100000, UsageReportRequest{})
	if err == nil {
		t.Error("Expected error for empty user_identity, got nil")
	}
}

func TestUsageService_ReserveAndCharge_EmptyLimitKey_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.ReserveAndCharge(ctx, "user@example.com", "solo", "", 100000, UsageReportRequest{})
	if err == nil {
		t.Error("Expected error for empty limit_key, got nil")
	}
}

func TestUsageService_ReserveAndCharge_ZeroAmount_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.ReserveAndCharge(ctx, "user@example.com", "solo", "ai_credits", 0, UsageReportRequest{})
	if err == nil {
		t.Error("Expected error for zero amount, got nil")
	}
}

func TestUsageService_ReserveAndCharge_NegativeAmount_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.ReserveAndCharge(ctx, "user@example.com", "solo", "ai_credits", -100000, UsageReportRequest{})
	if err == nil {
		t.Error("Expected error for negative amount, got nil")
	}
}

func TestUsageService_ReserveAndCharge_UnlimitedTier_AllowsAnyAmount(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based", CostMultiplier: 1000000}, nil // unlimited
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Should allow very large amount for unlimited tier
	err := svc.ReserveAndCharge(ctx, "user@example.com", "business", "ai_credits", 9999999999, UsageReportRequest{})
	if err != nil {
		t.Fatalf("ReserveAndCharge() returned error for unlimited tier: %v", err)
	}
}

// ============================================================================
// ReserveCredits Tests
// ============================================================================

func TestUsageService_ReserveCredits_Success_ReturnsReservationID(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: 500000000, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	reservationID, err := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000000)
	if err != nil {
		t.Fatalf("ReserveCredits() returned error: %v", err)
	}
	if reservationID == "" {
		t.Error("Expected non-empty reservation ID")
	}

	// Verify reservation was created
	var status string
	err = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, reservationID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query reservation: %v", err)
	}
	if status != "pending" {
		t.Errorf("Expected status 'pending', got %q", status)
	}
}

func TestUsageService_ReserveCredits_PendingReservationsCountedAgainstLimit(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: 100000000, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// First reservation takes most of the limit
	_, err := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 80000000)
	if err != nil {
		t.Fatalf("First ReserveCredits() returned error: %v", err)
	}

	// Second reservation should fail (would exceed limit)
	_, err = svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 30000000)
	if err == nil {
		t.Error("Expected error for second reservation exceeding limit, got nil")
	}
}

func TestUsageService_ReserveCredits_InsufficientCredits_ReturnsError(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: 50000000, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000000)
	if err == nil {
		t.Error("Expected ErrInsufficientCredits, got nil")
	}
}

func TestUsageService_ReserveCredits_EmptyUserIdentity_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	_, err := svc.ReserveCredits(ctx, "", "solo", "ai_credits", 100000)
	if err == nil {
		t.Error("Expected error for empty user_identity, got nil")
	}
}

func TestUsageService_ReserveCredits_ReservationExpiresAtSetCorrectly(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil // unlimited
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	before := time.Now()
	reservationID, err := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)
	if err != nil {
		t.Fatalf("ReserveCredits() returned error: %v", err)
	}
	after := time.Now()

	// Verify expires_at is set to ~10 minutes from now
	var expiresAt time.Time
	err = db.QueryRow(`SELECT expires_at FROM credit_reservations WHERE id = ?`, reservationID).Scan(&expiresAt)
	if err != nil {
		t.Fatalf("Failed to query reservation: %v", err)
	}

	expectedMin := before.Add(9 * time.Minute)
	expectedMax := after.Add(11 * time.Minute)
	if expiresAt.Before(expectedMin) || expiresAt.After(expectedMax) {
		t.Errorf("Expected expires_at around 10 minutes from now, got %v", expiresAt)
	}
}

// ============================================================================
// FinalizeReservation Tests
// ============================================================================

func TestUsageService_FinalizeReservation_Success_RecordsUsage(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Create a reservation
	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)

	// Finalize with actual amount
	err := svc.FinalizeReservation(ctx, reservationID, 80000)
	if err != nil {
		t.Fatalf("FinalizeReservation() returned error: %v", err)
	}

	// Verify reservation status
	var status string
	err = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, reservationID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query reservation: %v", err)
	}
	if status != "finalized" {
		t.Errorf("Expected status 'finalized', got %q", status)
	}

	// Verify usage was recorded
	var usageAmount int64
	err = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ?`, "user@example.com").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage: %v", err)
	}
	if usageAmount != 80000 {
		t.Errorf("Expected usage 80000, got %d", usageAmount)
	}
}

func TestUsageService_FinalizeReservation_EmptyReservationID_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.FinalizeReservation(ctx, "", 100000)
	if err == nil {
		t.Error("Expected error for empty reservation_id, got nil")
	}
}

func TestUsageService_FinalizeReservation_NotFound_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.FinalizeReservation(ctx, "non-existent-id", 100000)
	if err == nil {
		t.Error("Expected error for non-existent reservation, got nil")
	}
}

func TestUsageService_FinalizeReservation_AlreadyFinalized_ReturnsError(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Create and finalize a reservation
	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)
	_ = svc.FinalizeReservation(ctx, reservationID, 80000)

	// Try to finalize again
	err := svc.FinalizeReservation(ctx, reservationID, 50000)
	if err == nil {
		t.Error("Expected error for already finalized reservation, got nil")
	}
}

func TestUsageService_FinalizeReservation_AlreadyReleased_ReturnsError(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Create and release a reservation
	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)
	_ = svc.ReleaseReservation(ctx, reservationID)

	// Try to finalize
	err := svc.FinalizeReservation(ctx, reservationID, 50000)
	if err == nil {
		t.Error("Expected error for already released reservation, got nil")
	}
}

func TestUsageService_FinalizeReservation_NegativeAmount_ReturnsError(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)

	err := svc.FinalizeReservation(ctx, reservationID, -50000)
	if err == nil {
		t.Error("Expected error for negative amount, got nil")
	}
}

// ============================================================================
// ReleaseReservation Tests
// ============================================================================

func TestUsageService_ReleaseReservation_Success_NoUsageRecorded(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Create a reservation
	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)

	// Release it
	err := svc.ReleaseReservation(ctx, reservationID)
	if err != nil {
		t.Fatalf("ReleaseReservation() returned error: %v", err)
	}

	// Verify reservation status
	var status string
	err = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, reservationID).Scan(&status)
	if err != nil {
		t.Fatalf("Failed to query reservation: %v", err)
	}
	if status != "released" {
		t.Errorf("Expected status 'released', got %q", status)
	}

	// Verify no usage was recorded
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM usage_records WHERE user_identity = ?`, "user@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query usage: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected no usage records, got %d", count)
	}
}

func TestUsageService_ReleaseReservation_EmptyID_ReturnsError(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	err := svc.ReleaseReservation(ctx, "")
	if err == nil {
		t.Error("Expected error for empty reservation_id, got nil")
	}
}

func TestUsageService_ReleaseReservation_AlreadyFinalized_LogsNoop(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Create and finalize a reservation
	reservationID, _ := svc.ReserveCredits(ctx, "user@example.com", "solo", "ai_credits", 100000)
	_ = svc.FinalizeReservation(ctx, reservationID, 80000)

	// Release should succeed but be a no-op
	err := svc.ReleaseReservation(ctx, reservationID)
	if err != nil {
		t.Fatalf("ReleaseReservation() returned error: %v", err)
	}

	// Status should still be finalized
	var status string
	_ = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, reservationID).Scan(&status)
	if status != "finalized" {
		t.Errorf("Expected status to remain 'finalized', got %q", status)
	}
}

func TestUsageService_ReleaseReservation_NotFound_LogsNoop(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()

	ctx := context.Background()

	// Should not return error for non-existent reservation
	err := svc.ReleaseReservation(ctx, "non-existent-id")
	if err != nil {
		t.Fatalf("ReleaseReservation() returned error for non-existent ID: %v", err)
	}
}

// ============================================================================
// CleanupExpiredReservations Tests
// ============================================================================

func TestUsageService_CleanupExpiredReservations_ExpiresOldPending(t *testing.T) {
	mock := &MockLimitsService{
		GetLimitFn: func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*TierLimit, error) {
			return &TierLimit{LimitValue: -1, LimitType: "cost_based"}, nil
		},
	}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Insert an expired pending reservation directly
	_, err := db.Exec(`
		INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'pending', datetime('now', '-1 hour'))
	`, "expired-res-1", "user@example.com", getCurrentBillingPeriodTest(), "ai_credits", 100000)
	if err != nil {
		t.Fatalf("Failed to insert expired reservation: %v", err)
	}

	// Run cleanup
	count, err := svc.CleanupExpiredReservations(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredReservations() returned error: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 expired reservation, got %d", count)
	}

	// Verify status changed
	var status string
	_ = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, "expired-res-1").Scan(&status)
	if status != "expired" {
		t.Errorf("Expected status 'expired', got %q", status)
	}
}

func TestUsageService_CleanupExpiredReservations_ReturnsCorrectCount(t *testing.T) {
	mock := &MockLimitsService{}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()
	period := getCurrentBillingPeriodTest()

	// Insert multiple expired reservations
	for i := 1; i <= 3; i++ {
		_, _ = db.Exec(`
			INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at)
			VALUES (?, ?, ?, ?, ?, 'pending', datetime('now', '-1 hour'))
		`, "expired-res-"+string(rune('0'+i)), "user@example.com", period, "ai_credits", 100000)
	}

	count, err := svc.CleanupExpiredReservations(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredReservations() returned error: %v", err)
	}
	if count != 3 {
		t.Errorf("Expected 3 expired reservations, got %d", count)
	}
}

func TestUsageService_CleanupExpiredReservations_NoExpired_ReturnsZero(t *testing.T) {
	mock := &MockLimitsService{}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Insert a non-expired reservation with a future expiry using SQLite datetime format
	_, err := db.Exec(`
		INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'pending', datetime('now', '+1 hour'))
	`, "non-expired-res", "user@example.com", getCurrentBillingPeriodTest(), "ai_credits", 100000)
	if err != nil {
		t.Fatalf("Failed to insert non-expired reservation: %v", err)
	}

	// Run cleanup
	count, err := svc.CleanupExpiredReservations(ctx)
	if err != nil {
		t.Fatalf("CleanupExpiredReservations() returned error: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 expired reservations, got %d", count)
	}
}

// ============================================================================
// AdjustUsage Tests
// ============================================================================

func TestUsageService_AdjustUsage_PositiveAdjustment_IncreasesUsage(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	period := getCurrentBillingPeriodTest()

	// Seed initial usage
	_, err := db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", period, "ai_credits", 100000)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Apply positive adjustment
	err = svc.AdjustUsage(ctx, "user@example.com", "ai_credits", 50000, "test adjustment")
	if err != nil {
		t.Fatalf("AdjustUsage() returned error: %v", err)
	}

	// Verify usage increased
	var usageAmount int64
	_ = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ? AND limit_key = ?`,
		"user@example.com", "ai_credits").Scan(&usageAmount)
	if usageAmount != 150000 {
		t.Errorf("Expected usage 150000, got %d", usageAmount)
	}
}

func TestUsageService_AdjustUsage_NegativeAdjustment_DecreasesUsage(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	period := getCurrentBillingPeriodTest()

	// Seed initial usage
	_, _ = db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", period, "ai_credits", 100000)

	// Apply negative adjustment (refund)
	err := svc.AdjustUsage(ctx, "user@example.com", "ai_credits", -30000, "refund")
	if err != nil {
		t.Fatalf("AdjustUsage() returned error: %v", err)
	}

	// Verify usage decreased
	var usageAmount int64
	_ = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ? AND limit_key = ?`,
		"user@example.com", "ai_credits").Scan(&usageAmount)
	if usageAmount != 70000 {
		t.Errorf("Expected usage 70000, got %d", usageAmount)
	}
}

func TestUsageService_AdjustUsage_FloorAtZero_PreventsNegative(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()
	period := getCurrentBillingPeriodTest()

	// Seed initial usage
	_, _ = db.Exec(`
		INSERT INTO usage_records (user_identity, billing_period, limit_key, usage_amount)
		VALUES (?, ?, ?, ?)
	`, "user@example.com", period, "ai_credits", 50000)

	// Apply a large negative adjustment that would go negative
	err := svc.AdjustUsage(ctx, "user@example.com", "ai_credits", -100000, "large refund")
	if err != nil {
		t.Fatalf("AdjustUsage() returned error: %v", err)
	}

	// Verify usage floored at 0
	var usageAmount int64
	_ = db.QueryRow(`SELECT usage_amount FROM usage_records WHERE user_identity = ? AND limit_key = ?`,
		"user@example.com", "ai_credits").Scan(&usageAmount)
	if usageAmount != 0 {
		t.Errorf("Expected usage 0 (floored), got %d", usageAmount)
	}
}

func TestUsageService_AdjustUsage_EmptyUserIdentity_ReturnsError(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	err := svc.AdjustUsage(ctx, "", "ai_credits", 50000, "test")
	if err == nil {
		t.Error("Expected error for empty user_identity, got nil")
	}
}

func TestUsageService_AdjustUsage_ZeroAdjustment_IsNoOp(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()

	ctx := context.Background()

	// Zero adjustment should return nil immediately
	err := svc.AdjustUsage(ctx, "user@example.com", "ai_credits", 0, "no-op")
	if err != nil {
		t.Fatalf("AdjustUsage(0) returned error: %v", err)
	}
}

// ============================================================================
// StartReservationCleanup Tests
// ============================================================================

func TestUsageService_StartReservationCleanup_GoroutineStartsAndRuns(t *testing.T) {
	mock := &MockLimitsService{}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	// Insert an expired reservation
	_, _ = db.Exec(`
		INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'pending', datetime('now', '-1 hour'))
	`, "expired-cleanup-test", "user@example.com", getCurrentBillingPeriodTest(), "ai_credits", 100000)

	// Start cleanup with short interval
	cancel := svc.StartReservationCleanup(100 * time.Millisecond)
	defer cancel()

	// Wait for cleanup to run
	time.Sleep(250 * time.Millisecond)

	// Verify the expired reservation was cleaned up
	var status string
	_ = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, "expired-cleanup-test").Scan(&status)
	if status != "expired" {
		t.Errorf("Expected status 'expired' after cleanup, got %q", status)
	}
}

func TestUsageService_StartReservationCleanup_CancelStopsGoroutine(t *testing.T) {
	mock := &MockLimitsService{}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()

	// Start cleanup
	cancel := svc.StartReservationCleanup(50 * time.Millisecond)

	// Cancel immediately
	cancel()

	// Insert an expired reservation after cancel
	_, _ = db.Exec(`
		INSERT INTO credit_reservations (id, user_identity, billing_period, limit_key, reserved_amount, status, expires_at)
		VALUES (?, ?, ?, ?, ?, 'pending', datetime('now', '-1 hour'))
	`, "expired-after-cancel", "user@example.com", getCurrentBillingPeriodTest(), "ai_credits", 100000)

	// Wait a bit
	time.Sleep(150 * time.Millisecond)

	// Verify the reservation was NOT cleaned up (goroutine stopped)
	var status string
	_ = db.QueryRow(`SELECT status FROM credit_reservations WHERE id = ?`, "expired-after-cancel").Scan(&status)
	if status != "pending" {
		t.Errorf("Expected status 'pending' (cleanup stopped), got %q", status)
	}
}
