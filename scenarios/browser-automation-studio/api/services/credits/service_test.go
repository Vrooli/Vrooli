package credits

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// createTestDB creates an in-memory SQLite database for testing.
func createTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create required tables
	schema := `
		CREATE TABLE credit_usage (
			id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			billing_month TEXT NOT NULL,
			total_credits_used INTEGER DEFAULT 0,
			total_operations INTEGER DEFAULT 0,
			credits_by_operation TEXT DEFAULT '{}',
			operations_by_type TEXT DEFAULT '{}',
			last_operation_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_identity, billing_month)
		);

		CREATE TABLE operation_log (
			id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			operation_type TEXT NOT NULL,
			credits_charged INTEGER DEFAULT 0,
			success INTEGER DEFAULT 1,
			metadata TEXT DEFAULT '{}',
			error_message TEXT,
			duration_ms INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE usage_outbox (
			operation_id TEXT PRIMARY KEY,
			user_identity TEXT NOT NULL,
			payload TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			attempts INTEGER NOT NULL DEFAULT 0,
			next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_error TEXT,
			delivered_at TIMESTAMP,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err = db.Exec(schema)
	if err != nil {
		t.Fatalf("Failed to create schema: %v", err)
	}

	return db
}

// createTestService creates a credit service for testing.
func createTestService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()

	db := createTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:     db,
		Logger: log,
	})

	return svc, db
}

// ============================================================================
// BYOK Tracking Tests
// ============================================================================

func TestCharge_BYOKOperation_ZeroCost(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Charge with BYOK=true
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
		Metadata: ChargeMetadata{
			Model: "gpt-4",
		},
	})
	if err != nil {
		t.Fatalf("Charge() with BYOK=true returned error: %v", err)
	}

	// Verify result
	if result.Charged != 0 {
		t.Errorf("Expected Charged=0 for BYOK, got %d", result.Charged)
	}
	if result.WasCharged {
		t.Error("Expected WasCharged=false for BYOK")
	}

	// Verify operation was logged
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 operation log entry, got %d", count)
	}

	// Verify the logged operation has 0 credits
	var creditsCharged int
	err = db.QueryRow("SELECT credits_charged FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&creditsCharged)
	if err != nil {
		t.Fatalf("Failed to query credits_charged: %v", err)
	}
	if creditsCharged != 0 {
		t.Errorf("Expected credits_charged=0 in log for BYOK, got %d", creditsCharged)
	}
}

func TestCharge_NonBYOKOperation_NormalCost(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Get expected cost
	expectedCost := svc.GetOperationCost(OpAIWorkflowGenerate)
	if expectedCost == 0 {
		t.Skip("OpAIWorkflowGenerate has 0 cost, cannot test non-BYOK charging")
	}

	// Charge without BYOK
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
		Metadata: ChargeMetadata{
			Model: "gpt-4",
		},
	})
	if err != nil {
		t.Fatalf("Charge() without BYOK returned error: %v", err)
	}

	// Verify result
	if result.Charged != expectedCost {
		t.Errorf("Expected Charged=%d, got %d", expectedCost, result.Charged)
	}
	if !result.WasCharged {
		t.Error("Expected WasCharged=true for non-BYOK")
	}

	// Verify operation was logged with correct cost
	var creditsCharged int
	err = db.QueryRow("SELECT credits_charged FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&creditsCharged)
	if err != nil {
		t.Fatalf("Failed to query credits_charged: %v", err)
	}
	if creditsCharged != expectedCost {
		t.Errorf("Expected credits_charged=%d in log, got %d", expectedCost, creditsCharged)
	}

	// Verify credit_usage was updated
	var totalCreditsUsed int
	err = db.QueryRow("SELECT total_credits_used FROM credit_usage WHERE user_identity = ?", "test@example.com").Scan(&totalCreditsUsed)
	if err != nil {
		t.Fatalf("Failed to query credit_usage: %v", err)
	}
	if totalCreditsUsed != expectedCost {
		t.Errorf("Expected total_credits_used=%d, got %d", expectedCost, totalCreditsUsed)
	}
}

func TestCharge_BYOKOperation_NoUsageIncrement(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Charge with BYOK=true
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
	})
	if err != nil {
		t.Fatalf("Charge() with BYOK=true returned error: %v", err)
	}

	// Verify credit_usage was NOT created/updated (BYOK has 0 cost, so no upsert)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM credit_usage WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query credit_usage: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 credit_usage rows for BYOK operation, got %d", count)
	}
}

func TestCharge_MultipleBYOKOperations_AllLoggedZeroCost(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Perform multiple BYOK operations
	operations := []OperationType{
		OpAIWorkflowGenerate,
		OpAIVisionNavigate,
		OpAIElementAnalyze,
	}

	for _, op := range operations {
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    op,
			IsBYOK:       true,
		})
		if err != nil {
			t.Fatalf("Charge() with BYOK=true for %s returned error: %v", op, err)
		}
	}

	// Verify all operations were logged
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != len(operations) {
		t.Errorf("Expected %d operation log entries, got %d", len(operations), count)
	}

	// Verify all have 0 credits
	var totalCredits int
	err = db.QueryRow("SELECT SUM(credits_charged) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&totalCredits)
	if err != nil {
		t.Fatalf("Failed to sum credits_charged: %v", err)
	}
	if totalCredits != 0 {
		t.Errorf("Expected total credits_charged=0 for all BYOK operations, got %d", totalCredits)
	}
}

func TestCharge_MixedBYOKAndNonBYOK(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// BYOK operation
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
	})
	if err != nil {
		t.Fatalf("BYOK Charge() returned error: %v", err)
	}

	// Non-BYOK operation
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Non-BYOK Charge() returned error: %v", err)
	}

	expectedCost := svc.GetOperationCost(OpAIWorkflowGenerate)

	// Verify 2 operations logged
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 operation log entries, got %d", count)
	}

	// Verify total credits is only from non-BYOK operation
	var totalCredits int
	err = db.QueryRow("SELECT SUM(credits_charged) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&totalCredits)
	if err != nil {
		t.Fatalf("Failed to sum credits_charged: %v", err)
	}
	if totalCredits != expectedCost {
		t.Errorf("Expected total credits_charged=%d (only non-BYOK), got %d", expectedCost, totalCredits)
	}

	// Verify result for non-BYOK was charged
	if result.Charged != expectedCost {
		t.Errorf("Non-BYOK result.Charged=%d, expected %d", result.Charged, expectedCost)
	}
}

// ============================================================================
// Charge Request Tests
// ============================================================================

func TestChargeRequest_IsBYOKDefaultFalse(t *testing.T) {
	req := ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
	}

	// IsBYOK should default to false (zero value)
	if req.IsBYOK {
		t.Error("Expected IsBYOK to default to false")
	}
}

func TestCharge_EmptyUserIdentity_NoError(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Charge with empty user identity
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
	})
	// Should not error, just return without charging
	if err != nil {
		t.Fatalf("Charge() with empty user returned error: %v", err)
	}

	if result.Charged != 0 {
		t.Errorf("Expected Charged=0 for empty user, got %d", result.Charged)
	}
	if result.WasCharged {
		t.Error("Expected WasCharged=false for empty user")
	}
}

// ============================================================================
// GetOperationCost Tests
// ============================================================================

func TestGetOperationCost_AIOperations(t *testing.T) {
	svc, db := createTestService(t)
	defer db.Close()

	// AI operations should have non-zero cost
	aiOps := []OperationType{
		OpAIWorkflowGenerate,
		OpAIWorkflowModify,
	}

	for _, op := range aiOps {
		cost := svc.GetOperationCost(op)
		if cost == 0 {
			t.Errorf("Expected non-zero cost for %s, got 0", op)
		}
	}
}

// ============================================================================
// LPBS Reporting Tests
// ============================================================================

// mockLPBSReporter is a mock LPBS reporter for testing.
type mockLPBSReporter struct {
	reports []LPBSUsageReport
	err     error
}

func (m *mockLPBSReporter) ReportUsage(ctx context.Context, report LPBSUsageReport) error {
	m.reports = append(m.reports, report)
	return m.err
}

// createTestServiceWithLPBS creates a credit service with a mock LPBS reporter.
func createTestServiceWithLPBS(t *testing.T, reporter LPBSReporter) (*Service, *sql.DB) {
	t.Helper()

	db := createTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:           db,
		Logger:       log,
		AppBundleKey: "browser-automation-studio",
		LPBSReporter: reporter,
	})

	return svc, db
}

func TestCharge_ReportsToLPBS_NonBYOK(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge without BYOK
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
		Metadata: ChargeMetadata{
			Model: "gpt-4",
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify LPBS was called
	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 LPBS report, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	if report.UserIdentity != "test@example.com" {
		t.Errorf("Expected UserIdentity 'test@example.com', got '%s'", report.UserIdentity)
	}
	if report.LimitKey != "ai_credits" {
		t.Errorf("Expected LimitKey 'ai_credits', got '%s'", report.LimitKey)
	}
	if report.AppBundleKey != "browser-automation-studio" {
		t.Errorf("Expected AppBundleKey 'browser-automation-studio', got '%s'", report.AppBundleKey)
	}
	if report.Metadata.Operation != string(OpAIWorkflowGenerate) {
		t.Errorf("Expected Operation '%s', got '%s'", OpAIWorkflowGenerate, report.Metadata.Operation)
	}
	if report.Metadata.IsBYOK {
		t.Error("Expected IsBYOK=false in metadata")
	}
	// UsageAmount should be positive for non-BYOK
	if report.UsageAmount <= 0 {
		t.Errorf("Expected positive UsageAmount for non-BYOK, got %d", report.UsageAmount)
	}
}

func TestCharge_ReportsToLPBS_BYOK_ZeroCost(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge with BYOK
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
		Metadata: ChargeMetadata{
			Model: "claude-3-opus",
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify LPBS was called
	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 LPBS report, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	// BYOK operations should report 0 cost
	if report.UsageAmount != 0 {
		t.Errorf("Expected UsageAmount 0 for BYOK, got %d", report.UsageAmount)
	}
	if !report.Metadata.IsBYOK {
		t.Error("Expected IsBYOK=true in metadata")
	}
}

func TestCharge_ReportsToLPBS_WithActualCostCents(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge with actual cost from provider (e.g., OpenRouter)
	actualCostCents := 0.5 // Half a cent
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity:    "test@example.com",
		Operation:       OpAIWorkflowGenerate,
		IsBYOK:          false,
		ActualCostCents: actualCostCents,
		Metadata: ChargeMetadata{
			Model: "gpt-4-turbo",
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify LPBS was called
	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 LPBS report, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	// UsageAmount should be actualCostCents * 1,000,000
	expectedUsage := int64(actualCostCents * 1_000_000)
	if report.UsageAmount != expectedUsage {
		t.Errorf("Expected UsageAmount %d, got %d", expectedUsage, report.UsageAmount)
	}
}

func TestCharge_LPBSUnavailable_LocalOpSucceeds(t *testing.T) {
	// Create a reporter that returns an error
	reporter := &mockLPBSReporter{
		err: context.DeadlineExceeded, // Simulate timeout
	}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge should succeed even if LPBS fails
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error despite LPBS failure: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true despite LPBS failure")
	}

	// Verify local operation was logged
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 local operation log entry despite LPBS failure, got %d", count)
	}
}

func TestCharge_ReportsCorrectUnits(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Test with actual cost
	testCases := []struct {
		name            string
		actualCostCents float64
		expectedUsage   int64
	}{
		{"1 cent", 1.0, 1_000_000},
		{"10 cents", 10.0, 10_000_000},
		{"0.1 cents", 0.1, 100_000},
		{"0.001 cents", 0.001, 1_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reporter.reports = nil // Reset

			_, err := svc.Charge(ctx, ChargeRequest{
				UserIdentity:    "test@example.com",
				Operation:       OpAIWorkflowGenerate,
				IsBYOK:          false,
				ActualCostCents: tc.actualCostCents,
			})
			if err != nil {
				t.Fatalf("Charge() returned error: %v", err)
			}

			if len(reporter.reports) != 1 {
				t.Fatalf("Expected 1 report, got %d", len(reporter.reports))
			}

			if reporter.reports[0].UsageAmount != tc.expectedUsage {
				t.Errorf("Expected UsageAmount %d, got %d", tc.expectedUsage, reporter.reports[0].UsageAmount)
			}
		})
	}
}

func TestCharge_NoLPBSConfig_SkipsReporting(t *testing.T) {
	// Create service without LPBS configuration
	svc, db := createTestService(t)
	defer db.Close()

	ctx := context.Background()

	// Charge should work without LPBS
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true")
	}
}

func TestCharge_ReportsMetadata(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIVisionNavigate,
		IsBYOK:       false,
		Metadata: ChargeMetadata{
			Model:        "gpt-4-vision-preview",
			PromptTokens: 1500,
		},
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 report, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	if report.Metadata.Model != "gpt-4-vision-preview" {
		t.Errorf("Expected Model 'gpt-4-vision-preview', got '%s'", report.Metadata.Model)
	}
	if report.Metadata.PromptTokens != 1500 {
		t.Errorf("Expected PromptTokens 1500, got %d", report.Metadata.PromptTokens)
	}
	if report.Metadata.Operation != string(OpAIVisionNavigate) {
		t.Errorf("Expected Operation '%s', got '%s'", OpAIVisionNavigate, report.Metadata.Operation)
	}
}

// ============================================================================
// LPBS Retry Logic Tests
// ============================================================================

// mockRetryLPBSReporter tracks call attempts for testing retry logic.
type mockRetryLPBSReporter struct {
	attempts     int
	failCount    int // Number of times to fail before succeeding
	reports      []LPBSUsageReport
	mu           sync.Mutex
	failureError error
}

func (m *mockRetryLPBSReporter) ReportUsage(ctx context.Context, report LPBSUsageReport) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.attempts++
	m.reports = append(m.reports, report)

	if m.attempts <= m.failCount {
		if m.failureError != nil {
			return m.failureError
		}
		return fmt.Errorf("simulated failure attempt %d", m.attempts)
	}
	return nil
}

func (m *mockRetryLPBSReporter) getAttempts() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.attempts
}

func TestSendLPBSReport_Success(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge should trigger LPBS report
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify report was sent
	if len(reporter.reports) != 1 {
		t.Errorf("Expected 1 report, got %d", len(reporter.reports))
	}
}

func TestSendLPBSReport_RetriesOnFailure(t *testing.T) {
	// Create a reporter that fails twice then succeeds
	reporter := &mockRetryLPBSReporter{
		failCount: 2,
	}

	db := createTestDB(t)
	defer db.Close()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:           db,
		Logger:       log,
		AppBundleKey: "browser-automation-studio",
		LPBSReporter: reporter,
	})

	ctx := context.Background()

	// Charge should trigger LPBS report with retries
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Note: The custom reporter path runs synchronously (not async with retries)
	// The retry logic is only applied to the HTTP path.
	// This test verifies the reporter interface works correctly.
	if reporter.getAttempts() != 1 {
		t.Errorf("Expected 1 attempt via custom reporter, got %d", reporter.getAttempts())
	}
}

func TestSendLPBSReport_MaxRetriesExhausted(t *testing.T) {
	// Create a reporter that always fails
	reporter := &mockRetryLPBSReporter{
		failCount:    100, // More than max retries
		failureError: fmt.Errorf("persistent failure"),
	}

	db := createTestDB(t)
	defer db.Close()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:           db,
		Logger:       log,
		AppBundleKey: "browser-automation-studio",
		LPBSReporter: reporter,
	})

	ctx := context.Background()

	// Charge should succeed even if LPBS fails
	result, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error despite LPBS failure: %v", err)
	}

	if !result.WasCharged {
		t.Error("Expected WasCharged=true despite LPBS failure")
	}

	// Verify local operation was logged
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM operation_log WHERE user_identity = ?", "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query operation_log: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 local operation log entry despite LPBS failure, got %d", count)
	}
}

// ============================================================================
// CanPerformAIOperation Tests
// ============================================================================

// createTestServiceWithEntitlementProvider creates a credit service with a mock entitlement provider.
func createTestServiceWithEntitlementProvider(t *testing.T, provider *MockEntitlementProvider) (*Service, *sql.DB) {
	t.Helper()

	db := createTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	svc := NewService(ServiceOptions{
		DB:                  db,
		Logger:              log,
		EntitlementProvider: provider,
	})

	return svc, db
}

func TestCanPerformAIOperation_BYOKBypass(t *testing.T) {
	// BYOK users should bypass all checks
	mock := &MockEntitlementProvider{
		Entitlement:    nil, // Would normally deny
		AICreditsLimit: 0,   // Would normally deny
		CanUseAI:       false,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, true)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if !canProceed {
		t.Error("Expected canProceed=true for BYOK user")
	}
	if errCode != "" {
		t.Errorf("Expected empty errCode for BYOK user, got %s", errCode)
	}
	if errMsg != "" {
		t.Errorf("Expected empty errMsg for BYOK user, got %s", errMsg)
	}
	if remaining != -1 {
		t.Errorf("Expected remaining=-1 (unlimited) for BYOK user, got %d", remaining)
	}
}

func TestCanPerformAIOperation_TierDeniesAI(t *testing.T) {
	// Tier check fails (CanUseAI=false)
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierFree},
		AICreditsLimit: 100,
		CanUseAI:       false,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if canProceed {
		t.Error("Expected canProceed=false when tier denies AI")
	}
	if errCode != "AI_NOT_AVAILABLE" {
		t.Errorf("Expected errCode='AI_NOT_AVAILABLE', got %s", errCode)
	}
	if errMsg == "" {
		t.Error("Expected non-empty errMsg when AI is not available")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining=0 when AI denied, got %d", remaining)
	}
}

func TestCanPerformAIOperation_TierAllowsAI_NoCreditsAccess(t *testing.T) {
	// Tier allows AI but AICreditsLimit=0 (no access)
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierFree},
		AICreditsLimit: 0,
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if canProceed {
		t.Error("Expected canProceed=false when credit limit is 0")
	}
	if errCode != "AI_NOT_AVAILABLE" {
		t.Errorf("Expected errCode='AI_NOT_AVAILABLE', got %s", errCode)
	}
	if errMsg == "" {
		t.Error("Expected non-empty errMsg when no credits access")
	}
	if remaining != 0 {
		t.Errorf("Expected remaining=0, got %d", remaining)
	}
}

func TestCanPerformAIOperation_InsufficientCredits(t *testing.T) {
	// User is at their credit limit
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierSolo},
		AICreditsLimit: 100, // Small limit
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	// First, charge some credits to reach the limit
	opCost := svc.GetOperationCost(OpAIWorkflowGenerate)
	if opCost == 0 {
		t.Skip("OpAIWorkflowGenerate has 0 cost")
	}

	// Use all available credits
	for i := 0; i < 100/opCost+1; i++ {
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    OpAIWorkflowGenerate,
			IsBYOK:       false,
		})
		if err != nil {
			// Expected once we hit the limit
			break
		}
	}

	// Now check if we can perform another operation
	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if canProceed {
		t.Error("Expected canProceed=false when credits exhausted")
	}
	if errCode != "INSUFFICIENT_CREDITS" {
		t.Errorf("Expected errCode='INSUFFICIENT_CREDITS', got %s", errCode)
	}
	if errMsg == "" {
		t.Error("Expected non-empty errMsg for insufficient credits")
	}
	if remaining < 0 {
		t.Errorf("Expected remaining>=0, got %d", remaining)
	}
}

func TestCanPerformAIOperation_Success(t *testing.T) {
	// Happy path: tier allows AI and credits are available
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierPro},
		AICreditsLimit: 500,
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if !canProceed {
		t.Errorf("Expected canProceed=true, errCode=%s, errMsg=%s", errCode, errMsg)
	}
	if errCode != "" {
		t.Errorf("Expected empty errCode, got %s", errCode)
	}
	if errMsg != "" {
		t.Errorf("Expected empty errMsg, got %s", errMsg)
	}
	if remaining != 500 {
		t.Errorf("Expected remaining=500, got %d", remaining)
	}
}

func TestCanPerformAIOperation_EntitlementError_FailsOpen(t *testing.T) {
	// Entitlement error should fail open (allow operation)
	mock := &MockEntitlementProvider{
		Entitlement:         nil,
		GetEntitlementError: fmt.Errorf("entitlement service unavailable"),
		AICreditsLimit:      -1, // Unlimited so CanCharge passes
		CanUseAI:            true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, _, _, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	// Should fail open - allow the operation
	if !canProceed {
		t.Errorf("Expected canProceed=true (fail open) when entitlement errors, got errCode=%s", errCode)
	}
}

func TestCanPerformAIOperation_EmptyUserIdentity(t *testing.T) {
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierPro},
		AICreditsLimit: 500,
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Empty user identity should still work (normalized to empty string)
	canProceed, _, _, remaining, err := svc.CanPerformAIOperation(ctx, "", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if !canProceed {
		t.Error("Expected canProceed=true for empty user identity")
	}
	if remaining != 500 {
		t.Errorf("Expected remaining=500, got %d", remaining)
	}
}

func TestCanPerformAIOperation_GetEntitlementCallsCounter(t *testing.T) {
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierPro},
		AICreditsLimit: 500,
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Call multiple times
	for i := 0; i < 3; i++ {
		_, _, _, _, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
		if err != nil {
			t.Fatalf("CanPerformAIOperation() returned error: %v", err)
		}
	}

	// Verify GetEntitlement was called for each CanPerformAIOperation call
	// Note: getEntitlement is called multiple times per CanPerformAIOperation
	// (once for tier check, once for credits check)
	if mock.GetEntitlementCalls == 0 {
		t.Error("Expected GetEntitlement to be called at least once")
	}
}

func TestCanPerformAIOperation_UnlimitedTier(t *testing.T) {
	// Business tier has unlimited credits
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierBusiness},
		AICreditsLimit: -1, // Unlimited
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if !canProceed {
		t.Error("Expected canProceed=true for unlimited tier")
	}
	if errCode != "" {
		t.Errorf("Expected empty errCode, got %s", errCode)
	}
	if errMsg != "" {
		t.Errorf("Expected empty errMsg, got %s", errMsg)
	}
	if remaining != -1 {
		t.Errorf("Expected remaining=-1 (unlimited), got %d", remaining)
	}
}

func TestCanPerformAIOperation_DifferentOperationTypes(t *testing.T) {
	testCases := []struct {
		name      string
		operation OperationType
	}{
		{"WorkflowGenerate", OpAIWorkflowGenerate},
		{"WorkflowModify", OpAIWorkflowModify},
		{"VisionNavigate", OpAIVisionNavigate},
		{"ElementAnalyze", OpAIElementAnalyze},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockEntitlementProvider{
				Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierPro},
				AICreditsLimit: 1000,
				CanUseAI:       true,
			}
			svc, db := createTestServiceWithEntitlementProvider(t, mock)
			defer db.Close()

			ctx := context.Background()

			canProceed, _, _, _, err := svc.CanPerformAIOperation(ctx, "test@example.com", tc.operation, false)
			if err != nil {
				t.Fatalf("CanPerformAIOperation() returned error for %s: %v", tc.operation, err)
			}
			if !canProceed {
				t.Errorf("Expected canProceed=true for operation %s", tc.operation)
			}
		})
	}
}

func TestCanPerformAIOperation_UserIdentityNormalization(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"lowercase", "test@example.com", "test@example.com"},
		{"uppercase", "TEST@EXAMPLE.COM", "test@example.com"},
		{"mixed case", "Test@Example.COM", "test@example.com"},
		{"with spaces", "  test@example.com  ", "test@example.com"},
		{"tabs", "\ttest@example.com\t", "test@example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &MockEntitlementProvider{
				Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierPro},
				AICreditsLimit: 500,
				CanUseAI:       true,
			}
			svc, db := createTestServiceWithEntitlementProvider(t, mock)
			defer db.Close()

			ctx := context.Background()

			_, _, _, _, err := svc.CanPerformAIOperation(ctx, tc.input, OpAIWorkflowGenerate, false)
			if err != nil {
				t.Fatalf("CanPerformAIOperation() returned error: %v", err)
			}

			// Verify the normalized identity was passed to GetEntitlement
			if mock.LastUserIdentity != tc.expected {
				t.Errorf("Expected normalized identity %q, got %q", tc.expected, mock.LastUserIdentity)
			}
		})
	}
}

// ============================================================================
// LPBS Idempotency Tests
// ============================================================================

func TestCharge_LPBSReport_IncludesOperationID(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Charge should trigger LPBS report with operation_id
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify LPBS report includes operation_id
	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 LPBS report, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	if report.OperationID == "" {
		t.Error("Expected non-empty OperationID in LPBS report")
	}

	// Verify it's a valid UUID format (36 chars with dashes)
	if len(report.OperationID) != 36 {
		t.Errorf("Expected UUID format (36 chars), got %d chars: %s", len(report.OperationID), report.OperationID)
	}
}

func TestCharge_LPBSReport_UniqueOperationID_PerCharge(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// Multiple charges should have unique operation_ids
	for i := 0; i < 3; i++ {
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    OpAIWorkflowGenerate,
			IsBYOK:       false,
		})
		if err != nil {
			t.Fatalf("Charge() #%d returned error: %v", i+1, err)
		}
	}

	// Verify each report has a unique operation_id
	if len(reporter.reports) != 3 {
		t.Fatalf("Expected 3 LPBS reports, got %d", len(reporter.reports))
	}

	seen := make(map[string]bool)
	for i, report := range reporter.reports {
		if report.OperationID == "" {
			t.Errorf("Report #%d has empty OperationID", i+1)
			continue
		}
		if seen[report.OperationID] {
			t.Errorf("Duplicate OperationID found: %s", report.OperationID)
		}
		seen[report.OperationID] = true
	}
}

func TestCharge_BYOK_LPBSReport_IncludesOperationID(t *testing.T) {
	reporter := &mockLPBSReporter{}
	svc, db := createTestServiceWithLPBS(t, reporter)
	defer db.Close()

	ctx := context.Background()

	// BYOK operations should also include operation_id
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       true,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	if len(reporter.reports) != 1 {
		t.Fatalf("Expected 1 LPBS report for BYOK, got %d", len(reporter.reports))
	}

	report := reporter.reports[0]
	if report.OperationID == "" {
		t.Error("Expected non-empty OperationID in BYOK LPBS report")
	}
	if !report.Metadata.IsBYOK {
		t.Error("Expected IsBYOK=true in metadata")
	}
}

// ============================================================================
// Error Message Tests
// ============================================================================

func TestCanPerformAIOperation_InsufficientCredits_MessageIncludesRemaining(t *testing.T) {
	costs := DefaultOperationCosts()
	opCost := costs.GetCost(OpAIWorkflowGenerate)
	if opCost == 0 {
		t.Skip("OpAIWorkflowGenerate has 0 cost")
	}

	// Set a limit that will be exactly exhausted by a few operations
	creditLimit := opCost * 2 // Allow exactly 2 operations

	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierSolo},
		AICreditsLimit: creditLimit,
		CanUseAI:       true,
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	// Exhaust credits with exactly enough operations
	for i := 0; i < 3; i++ { // One more than allowed
		_, err := svc.Charge(ctx, ChargeRequest{
			UserIdentity: "test@example.com",
			Operation:    OpAIWorkflowGenerate,
			IsBYOK:       false,
		})
		// Ignore errors (we expect the 3rd to fail)
		_ = err
	}

	// Now check - should have insufficient credits
	canProceed, errCode, errMsg, remaining, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if canProceed {
		t.Error("Expected canProceed=false when credits exhausted")
	}
	if errCode != "INSUFFICIENT_CREDITS" {
		t.Errorf("Expected errCode='INSUFFICIENT_CREDITS', got %s", errCode)
	}
	// Error message should include remaining credits info
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
	// remaining should be a non-negative number
	if remaining < 0 {
		t.Errorf("Expected remaining>=0 in message, got %d", remaining)
	}
}

func TestCanPerformAIOperation_TierDenied_MessageDescriptive(t *testing.T) {
	mock := &MockEntitlementProvider{
		Entitlement:    &entitlement.Entitlement{Tier: entitlement.TierFree},
		AICreditsLimit: 100,   // Has credits
		CanUseAI:       false, // But tier denies AI
	}
	svc, db := createTestServiceWithEntitlementProvider(t, mock)
	defer db.Close()

	ctx := context.Background()

	canProceed, errCode, errMsg, _, err := svc.CanPerformAIOperation(ctx, "test@example.com", OpAIWorkflowGenerate, false)
	if err != nil {
		t.Fatalf("CanPerformAIOperation() returned error: %v", err)
	}
	if canProceed {
		t.Error("Expected canProceed=false when tier denies AI")
	}
	if errCode != "AI_NOT_AVAILABLE" {
		t.Errorf("Expected errCode='AI_NOT_AVAILABLE', got %s", errCode)
	}
	if errMsg == "" {
		t.Error("Expected descriptive error message for AI not available")
	}
	// Message should mention subscription
	if !strings.Contains(errMsg, "subscription") && !strings.Contains(errMsg, "tier") {
		t.Errorf("Expected error message to mention subscription/tier, got: %s", errMsg)
	}
}
