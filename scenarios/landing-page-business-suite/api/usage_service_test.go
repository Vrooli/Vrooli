package main

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"landing-page-business-suite-api/internal/commerce"
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

		CREATE TABLE IF NOT EXISTS usage_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			operation_id TEXT UNIQUE,
			user_identity TEXT NOT NULL,
			app_bundle_key TEXT NOT NULL DEFAULT 'unattributed',
			model TEXT NOT NULL DEFAULT 'unknown',
			credits INTEGER NOT NULL,
			cost_micros INTEGER NOT NULL DEFAULT 0,
			prompt_tokens INTEGER NOT NULL DEFAULT 0,
			completion_tokens INTEGER NOT NULL DEFAULT 0,
			provider TEXT NOT NULL DEFAULT 'local',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

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
func createTestUsageService(t *testing.T) (*commerce.UsageService, *commerce.LimitsService, *sql.DB) {
	t.Helper()

	db := createTestUsageDB(t)
	seedTestUsageTierLimits(t, db)
	limitsSvc := NewLimitsService(db, "sqlite")

	// Construct the domain service directly so unit tests never inherit a
	// process-level production secret from the composition root.
	usageSvc := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB: db, LimitsService: limitsSvc, Dialect: "sqlite",
	})

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
		{"free", "count_based", "compute_minutes", 0},
		{"solo", "count_based", "compute_minutes", 500000000}, // count quota
		{"pro", "count_based", "compute_minutes", 2000000000}, // count quota
		{"business", "count_based", "compute_minutes", -1},    // unlimited
	}

	for _, l := range limits {
		_, err := db.Exec(`
			INSERT INTO subscription_tier_limits (tier_id, limit_type, limit_key, limit_value)
			SELECT ?, ?, ?, ? WHERE NOT EXISTS (SELECT 1 FROM subscription_tier_limits WHERE tier_id=? AND limit_key=? AND app_bundle_key IS NULL)
		`, l.tierID, l.limitType, l.limitKey, l.limitValue, l.tierID, l.limitKey)
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
	GetLimitFn      func(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*commerce.TierLimit, error)
	GetTierLimitsFn func(ctx context.Context, tierID string) ([]commerce.TierLimit, error)
	GetLimitCalls   []struct {
		TierID       string
		LimitKey     string
		AppBundleKey *string
	}
}

func (m *MockLimitsService) GetLimit(ctx context.Context, tierID, limitKey string, appBundleKey *string) (*commerce.TierLimit, error) {
	m.GetLimitCalls = append(m.GetLimitCalls, struct {
		TierID       string
		LimitKey     string
		AppBundleKey *string
	}{tierID, limitKey, appBundleKey})
	if m.GetLimitFn != nil {
		return m.GetLimitFn(ctx, tierID, limitKey, appBundleKey)
	}
	// Default: unlimited
	return &commerce.TierLimit{LimitValue: -1, LimitType: "cost_based", CostMultiplier: 1000000}, nil
}

func (m *MockLimitsService) GetTierLimits(ctx context.Context, tierID string) ([]commerce.TierLimit, error) {
	if m.GetTierLimitsFn != nil {
		return m.GetTierLimitsFn(ctx, tierID)
	}
	return []commerce.TierLimit{}, nil
}

// Compile-time check
var _ commerce.LimitsServicer = (*MockLimitsService)(nil)

// createTestUsageServiceWithMock creates a usage service with a mock limits service.
func createTestUsageServiceWithMock(t *testing.T, mock *MockLimitsService) (*commerce.UsageService, *sql.DB) {
	t.Helper()

	db := createTestUsageDB(t)

	usageSvc := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
		DB: db, LimitsService: mock, Dialect: "sqlite",
	})

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

	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", currentPeriod, "compute_minutes").Scan(&usageAmount)
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
	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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
	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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

	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "",
		LimitKey:     "compute_minutes",
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

	req := commerce.UsageReportRequest{Tier: "solo",
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

	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", currentPeriod, "compute_minutes", 250000, "browser-automation-studio")
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get usage
	usage, err := svc.GetUsage(ctx, "user@example.com", "compute_minutes", nil)
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

	usage, err := svc.GetUsage(ctx, "nonexistent@example.com", "compute_minutes", nil)
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
		"user@example.com", currentPeriod, "compute_minutes", 100000, "app1",
		"user@example.com", currentPeriod, "compute_minutes", 50000, "app2",
	)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get total usage (nil appBundleKey should sum all)
	usage, err := svc.GetUsage(ctx, "user@example.com", "compute_minutes", nil)
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
	`, "user@example.com", currentPeriod, "compute_minutes", 100000000) // 100 million internal units
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Get the authenticated principal's count quota summary.
	summary, err := svc.GetUsageSummary(ctx, "user@example.com", "solo")
	if err != nil {
		t.Fatalf("GetUsageSummary() returned error: %v", err)
	}

	// Check usage
	if summary.Usage["compute_minutes"] != 100000000 {
		t.Errorf("Expected usage 100000000, got %d", summary.Usage["compute_minutes"])
	}

	// Check remaining (500,000,000 - 100,000,000 = 400,000,000)
	if summary.Remaining["compute_minutes"] != 400000000 {
		t.Errorf("Expected remaining 400000000, got %d", summary.Remaining["compute_minutes"])
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
	if summary.Remaining["compute_minutes"] != -1 {
		t.Errorf("Expected remaining -1 (unlimited), got %d", summary.Remaining["compute_minutes"])
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
	`, "user@example.com", currentPeriod, "compute_minutes", 100000000) // 100 million (below $5 = 500 million)
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Check if user can proceed with 10 million more units
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "solo", "compute_minutes", 10000000)
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
	`, "user@example.com", currentPeriod, "compute_minutes", 500000000) // At limit
	if err != nil {
		t.Fatalf("Failed to seed usage: %v", err)
	}

	// Check if user can proceed with any more units
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "solo", "compute_minutes", 1)
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
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "business", "compute_minutes", 1000000000)
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
	canProceed, remaining, err := svc.CheckLimit(ctx, "user@example.com", "free", "compute_minutes", 1)
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

func TestUsageService_CheckLimit_NoTierFailsClosed(t *testing.T) {
	svc, _, db := createTestUsageService(t)
	defer db.Close()
	allowed, _, err := svc.CheckLimit(context.Background(), "user@example.com", "", "compute_minutes", 1)
	if err == nil || allowed {
		t.Fatal("missing server tier must not authorize a quota operation")
	}
}

func TestUsageService_HealthCheck_ReportsServiceAuthState(t *testing.T) {
	cases := []struct {
		name       string
		configured func() bool
		want       bool
		wantMode   string
	}{
		{"unconfigured seam", nil, false, "disabled"},
		{"configured", func() bool { return true }, true, "enabled"},
		{"explicitly disabled", func() bool { return false }, false, "disabled"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := createTestUsageDB(t)
			defer db.Close()

			svc := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{
				DB:                    db,
				LimitsService:         NewLimitsService(db, "sqlite"),
				Dialect:               "sqlite",
				ServiceAuthConfigured: tc.configured,
			})

			status, err := svc.HealthCheck(context.Background())
			if err != nil {
				t.Fatalf("HealthCheck() returned error: %v", err)
			}
			if status.ServiceAuthConfigured != tc.want {
				t.Fatalf("service_auth_configured = %v, want %v", status.ServiceAuthConfigured, tc.want)
			}
			if status.ServiceAuthMode != tc.wantMode {
				t.Fatalf("service_auth_mode = %q, want %q", status.ServiceAuthMode, tc.wantMode)
			}
		})
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
		"user1@example.com", currentPeriod, "compute_minutes", 100000, "app1",
		"user2@example.com", currentPeriod, "compute_minutes", 200000, "app2",
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
	`, "user@example.com", currentPeriod, "compute_minutes", 100000)
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

	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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

	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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
	req1 := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user1@example.com",
		LimitKey:     "compute_minutes",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID1,
	}
	err := svc.RecordUsage(ctx, req1)
	if err != nil {
		t.Fatalf("RecordUsage() for user1 returned error: %v", err)
	}

	// Second user with different operation_id
	req2 := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user2@example.com",
		LimitKey:     "compute_minutes",
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
	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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
	req := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
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
	req1 := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
		Amount:       100000,
		AppBundleKey: "browser-automation-studio",
		OperationID:  &operationID1,
	}
	err := svc.RecordUsage(ctx, req1)
	if err != nil {
		t.Fatalf("First RecordUsage() returned error: %v", err)
	}

	req2 := commerce.UsageReportRequest{Tier: "solo",
		UserIdentity: "user@example.com",
		LimitKey:     "compute_minutes",
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
	`, "user@example.com", "compute_minutes").Scan(&usageAmount)
	if err != nil {
		t.Fatalf("Failed to query usage record: %v", err)
	}

	// Should be 150000 (100000 + 50000)
	if usageAmount != 150000 {
		t.Errorf("Expected usage_amount 150000, got %d", usageAmount)
	}
}

// The retired personal credit reservation and adjustment API is deliberately
// absent. Account settlement/replay/overrun and funding order are proved by
// internal/commerce's BusinessAccountUsage and Postgres funding-lot tests.
// This boundary retains the independent count-quota contract.
func TestUsageService_CountQuotaLifecycle(t *testing.T) {
	mock := &MockLimitsService{GetLimitFn: func(context.Context, string, string, *string) (*commerce.TierLimit, error) {
		return &commerce.TierLimit{LimitType: "count_based", LimitValue: 100}, nil
	}}
	svc, db := createTestUsageServiceWithMock(t, mock)
	defer db.Close()
	ctx := context.Background()
	for _, test := range []struct {
		actor, key string
		amount     int64
		window     time.Duration
	}{{"", "compute_minutes", 1, 0}, {"user@example.com", "", 1, 0}, {"user@example.com", "compute_minutes", 0, 0}, {"user@example.com", "compute_minutes", -1, 0}, {"user@example.com", "compute_minutes", 1, -time.Second}, {"user@example.com", "compute_minutes", 1, 25 * time.Hour}} {
		if _, err := svc.ReserveQuota(ctx, test.actor, "solo", test.key, test.amount, test.window); err == nil {
			t.Errorf("invalid quota input accepted: %+v", test)
		}
	}
	id, err := svc.ReserveQuota(ctx, "user@example.com", "solo", "compute_minutes", 60, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReserveQuota(ctx, "user@example.com", "solo", "compute_minutes", 41, 0); err == nil {
		t.Fatal("pending capacity must count against quota")
	}
	if err := svc.FinalizeQuota(ctx, "other@example.com", "solo", id, 30); err == nil {
		t.Fatal("cross-subject finalization accepted")
	}
	if err := svc.ReleaseQuota(ctx, "other@example.com", "solo", id); err == nil {
		t.Fatal("cross-subject release accepted")
	}
	if err := svc.FinalizeQuota(ctx, "user@example.com", "solo", id, -1); err == nil {
		t.Fatal("negative settlement accepted")
	}
	// A failed count write must leave its reservation pending in the same transaction.
	if _, err := db.Exec(`CREATE TRIGGER fail_quota BEFORE INSERT ON usage_records BEGIN SELECT RAISE(ABORT,'forced count write failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinalizeQuota(ctx, "user@example.com", "solo", id, 30); err == nil {
		t.Fatal("expected forced persistence error")
	}
	var state string
	if err := db.QueryRow(`SELECT status FROM credit_reservations WHERE id=?`, id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "pending" {
		t.Fatalf("failed finalization changed state to %s", state)
	}
	if _, err := db.Exec(`DROP TRIGGER fail_quota`); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinalizeQuota(ctx, "user@example.com", "solo", id, 30); err != nil {
		t.Fatal(err)
	}
	if err := svc.FinalizeQuota(ctx, "user@example.com", "solo", id, 31); err == nil {
		t.Fatal("terminal count reservation changed")
	}
	second, err := svc.ReserveQuota(ctx, "user@example.com", "solo", "compute_minutes", 40, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ReleaseQuota(ctx, "user@example.com", "solo", second); err != nil {
		t.Fatal(err)
	}
	if err := svc.ReleaseQuota(ctx, "user@example.com", "solo", second); err != nil {
		t.Fatal("release retry should preserve terminal state", err)
	}
	if err := svc.FinalizeQuota(ctx, "user@example.com", "solo", second, 1); err == nil {
		t.Fatal("released quota finalized")
	}
	var amount int64
	if err := db.QueryRow(`SELECT SUM(usage_amount) FROM usage_records WHERE user_identity='user@example.com'`).Scan(&amount); err != nil {
		t.Fatal(err)
	}
	if amount != 30 {
		t.Fatalf("count usage=%d, expected30", amount)
	}
	var events int
	if err := db.QueryRow(`SELECT COUNT(*) FROM usage_events`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Fatal("non-credit quota must not emit credit usage events")
	}
}

func TestUsageService_CountQuotaRejectsCreditKind(t *testing.T) {
	svc, db := createTestUsageServiceWithMock(t, &MockLimitsService{})
	defer db.Close()
	if _, err := svc.ReserveQuota(context.Background(), "user@example.com", "solo", "ai_credits", 1, 0); err == nil {
		t.Fatal("quota owner accepted a cost-based meter")
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_reservations`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("cost-based rejection wrote quota state")
	}
}
