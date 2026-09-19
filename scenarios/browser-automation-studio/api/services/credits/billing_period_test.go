package credits

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/services/entitlement"
)

// ============================================================================
// Billing Period Boundary Tests
//
// These tests verify credit usage tracking across billing period boundaries,
// custom billing cycles, and edge cases like year boundaries.
// ============================================================================

// MockEntitlementProviderWithBillingCycle extends MockEntitlementProvider to support custom billing cycles.
type MockEntitlementProviderWithBillingCycle struct {
	Entitlement         *entitlement.Entitlement
	GetEntitlementError error
	AICreditsLimit      int
	CanUseAI            bool
	GetEntitlementCalls int
	LastUserIdentity    string
}

func (m *MockEntitlementProviderWithBillingCycle) GetEntitlement(ctx context.Context, userIdentity string) (*entitlement.Entitlement, error) {
	m.GetEntitlementCalls++
	m.LastUserIdentity = userIdentity
	return m.Entitlement, m.GetEntitlementError
}

func (m *MockEntitlementProviderWithBillingCycle) LimitForEntitlement(_ *entitlement.Entitlement) (int, bool) {
	return m.AICreditsLimit, true
}

func (m *MockEntitlementProviderWithBillingCycle) CanUseAIWithEntitlement(ent *entitlement.Entitlement) bool {
	return m.CanUseAI
}

var _ EntitlementProvider = (*MockEntitlementProviderWithBillingCycle)(nil)

// createTestServiceWithBillingCycle creates a service with a custom billing cycle.
func createTestServiceWithBillingCycle(t *testing.T, billingCycleStart int) (*Service, *sql.DB, *MockEntitlementProviderWithBillingCycle) {
	t.Helper()

	db := createTestDB(t)
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	provider := &MockEntitlementProviderWithBillingCycle{
		Entitlement: &entitlement.Entitlement{
			Tier:              entitlement.TierPro,
			Status:            entitlement.StatusActive,
			BillingCycleStart: billingCycleStart,
		},
		AICreditsLimit: 1000,
		CanUseAI:       true,
	}

	svc := NewService(ServiceOptions{
		DB:                  db,
		Logger:              log,
		EntitlementProvider: provider,
	})

	return svc, db, provider
}

func TestBillingPeriod_CustomCycle_Day15(t *testing.T) {
	// Test mid-month billing cycle (day 15)
	svc, db, _ := createTestServiceWithBillingCycle(t, 15)
	defer db.Close()

	ctx := context.Background()

	// Charge operation
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify billing month format includes day
	var billingMonth string
	err = db.QueryRow(`
		SELECT billing_month FROM credit_usage WHERE user_identity = ?
	`, "test@example.com").Scan(&billingMonth)
	if err != nil {
		t.Fatalf("Failed to query billing_month: %v", err)
	}

	// The billing month should be in YYYY-MM-DD format when custom cycle is set
	// Format depends on current date and billing cycle start
	if len(billingMonth) != 10 { // YYYY-MM-DD
		t.Errorf("Expected billing_month in YYYY-MM-DD format, got %s (len=%d)", billingMonth, len(billingMonth))
	}
}

func TestBillingPeriod_CustomCycle_Day28(t *testing.T) {
	// Test near month-end billing cycle (day 28)
	svc, db, _ := createTestServiceWithBillingCycle(t, 28)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	// Verify credit was charged
	var totalUsed int
	err = db.QueryRow(`
		SELECT total_credits_used FROM credit_usage WHERE user_identity = ?
	`, "test@example.com").Scan(&totalUsed)
	if err != nil {
		t.Fatalf("Failed to query total_credits_used: %v", err)
	}

	if totalUsed == 0 {
		t.Error("Expected credits to be charged")
	}
}

func TestBillingPeriod_CalendarMonth_Fallback(t *testing.T) {
	// BillingCycleStart=0 should fall back to calendar month
	svc, db, _ := createTestServiceWithBillingCycle(t, 0)
	defer db.Close()

	ctx := context.Background()

	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("Charge() returned error: %v", err)
	}

	var billingMonth string
	err = db.QueryRow(`
		SELECT billing_month FROM credit_usage WHERE user_identity = ?
	`, "test@example.com").Scan(&billingMonth)
	if err != nil {
		t.Fatalf("Failed to query billing_month: %v", err)
	}

	// Calendar month fallback uses YYYY-MM format
	if len(billingMonth) != 7 { // YYYY-MM
		t.Errorf("Expected billing_month in YYYY-MM format for calendar month fallback, got %s", billingMonth)
	}
}

func TestBillingPeriod_GetBillingMonth_CorrectFormat(t *testing.T) {
	// Test the Entitlement.GetBillingMonth method directly
	testCases := []struct {
		name              string
		billingCycleStart int
		date              time.Time
		expectedFormat    string // Either "YYYY-MM-DD" or "YYYY-MM"
	}{
		{
			name:              "custom day 15 before cycle",
			billingCycleStart: 15,
			date:              time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedFormat:    "2006-01-02", // Custom cycle uses YYYY-MM-DD
		},
		{
			name:              "custom day 15 after cycle",
			billingCycleStart: 15,
			date:              time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC),
			expectedFormat:    "2006-01-02",
		},
		{
			name:              "calendar month fallback",
			billingCycleStart: 0,
			date:              time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedFormat:    "2006-01-02", // GetBillingMonth always returns start date
		},
		{
			name:              "day 1 same as calendar",
			billingCycleStart: 1,
			date:              time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC),
			expectedFormat:    "2006-01-02",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ent := &entitlement.Entitlement{
				BillingCycleStart: tc.billingCycleStart,
			}

			billingMonth := ent.GetBillingMonth(tc.date)

			// Verify the format
			_, err := time.Parse(tc.expectedFormat, billingMonth)
			if err != nil {
				t.Errorf("GetBillingMonth returned invalid format: %s (expected %s format)", billingMonth, tc.expectedFormat)
			}
		})
	}
}

func TestBillingPeriod_YearBoundary_DecemberToJanuary(t *testing.T) {
	// Test year boundary with custom billing cycle
	ent := &entitlement.Entitlement{
		BillingCycleStart: 15, // Mid-month
	}

	// Test dates around year boundary
	testCases := []struct {
		name          string
		date          time.Time
		expectedYear  int
		expectedMonth time.Month
	}{
		{
			name:          "Jan 5 (before cycle day)",
			date:          time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC),
			expectedYear:  2025, // Period started in December
			expectedMonth: time.December,
		},
		{
			name:          "Jan 20 (after cycle day)",
			date:          time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC),
			expectedYear:  2026, // Period started in January
			expectedMonth: time.January,
		},
		{
			name:          "Dec 10 (before cycle day)",
			date:          time.Date(2025, 12, 10, 12, 0, 0, 0, time.UTC),
			expectedYear:  2025,
			expectedMonth: time.November,
		},
		{
			name:          "Dec 20 (after cycle day)",
			date:          time.Date(2025, 12, 20, 12, 0, 0, 0, time.UTC),
			expectedYear:  2025,
			expectedMonth: time.December,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, _ := ent.GetBillingPeriod(tc.date)

			if start.Year() != tc.expectedYear {
				t.Errorf("Expected year %d, got %d", tc.expectedYear, start.Year())
			}
			if start.Month() != tc.expectedMonth {
				t.Errorf("Expected month %v, got %v", tc.expectedMonth, start.Month())
			}
		})
	}
}

func TestBillingPeriod_UsageReset_AtPeriodBoundary(t *testing.T) {
	// Verify that usage is tracked separately per billing period
	svc, db, provider := createTestServiceWithBillingCycle(t, 15)
	defer db.Close()

	ctx := context.Background()

	// First charge in current period
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "test@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("First charge returned error: %v", err)
	}

	// Get the billing month that was used
	var firstBillingMonth string
	err = db.QueryRow(`
		SELECT billing_month FROM credit_usage WHERE user_identity = ?
	`, "test@example.com").Scan(&firstBillingMonth)
	if err != nil {
		t.Fatalf("Failed to query first billing_month: %v", err)
	}

	// Simulate a different billing period by manually inserting a record with a different month
	differentMonth := "2025-06-15" // A different period
	_, err = db.Exec(`
		INSERT INTO credit_usage (id, user_identity, billing_month, total_credits_used, total_operations)
		VALUES ('different-period-id', ?, ?, 999, 10)
	`, "test@example.com", differentMonth)
	if err != nil {
		t.Fatalf("Failed to insert different period usage: %v", err)
	}

	// Get current usage - should only count current period
	usage, err := svc.GetUsage(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	// The usage should be from the current billing period only
	if usage.BillingMonth == differentMonth {
		t.Error("GetUsage returned usage from different billing period")
	}

	// The total should not include the manually inserted 999 credits
	if usage.TotalCreditsUsed >= 999 {
		t.Errorf("Usage appears to include credits from different billing period: %d", usage.TotalCreditsUsed)
	}

	// Verify we have 2 distinct billing periods in the database
	var count int
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT billing_month) FROM credit_usage WHERE user_identity = ?
	`, "test@example.com").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count billing periods: %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 distinct billing periods, got %d", count)
	}

	_ = provider // Silence unused warning
}

func TestBillingPeriod_ChargeOnDay28_CheckOnDay1(t *testing.T) {
	// Simulate charging on day 28 and checking on day 1 of next month
	// This tests period rollover with charges

	// For day 28 billing cycle:
	// - If we're on Jan 30, we're in the Jan 28 - Feb 27 period
	// - If we're on Feb 1, we're still in the Jan 28 - Feb 27 period
	ent := &entitlement.Entitlement{
		BillingCycleStart: 28,
	}

	// Jan 30 and Feb 1 should be in the same billing period
	jan30 := time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC)
	feb1 := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

	jan30Month := ent.GetBillingMonth(jan30)
	feb1Month := ent.GetBillingMonth(feb1)

	if jan30Month != feb1Month {
		t.Errorf("Jan 30 and Feb 1 should be in same billing period for day-28 cycle, got %s and %s", jan30Month, feb1Month)
	}

	// Verify they both have the Jan 28 start date
	expectedStart := "2026-01-28"
	if jan30Month != expectedStart {
		t.Errorf("Expected billing month start %s, got %s", expectedStart, jan30Month)
	}
}

func TestBillingPeriod_Day1BillingCycle(t *testing.T) {
	// Day 1 billing cycle should behave like calendar months
	ent := &entitlement.Entitlement{
		BillingCycleStart: 1,
	}

	testDate := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	start, end := ent.GetBillingPeriod(testDate)

	// Should be Jan 1 to Jan 31
	expectedStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}

func TestBillingPeriod_InvalidCycleDay_FallsBackToCalendarMonth(t *testing.T) {
	// Invalid cycle days (< 1 or > 28) should fall back to calendar month
	testCases := []struct {
		name              string
		billingCycleStart int
	}{
		{"negative day", -1},
		{"day 0", 0},
		{"day 29", 29},
		{"day 31", 31},
		{"day 100", 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ent := &entitlement.Entitlement{
				BillingCycleStart: tc.billingCycleStart,
			}

			testDate := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
			start, end := ent.GetBillingPeriod(testDate)

			// Should fall back to calendar month (March 1 - March 31)
			expectedStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
			expectedEnd := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

			if !start.Equal(expectedStart) {
				t.Errorf("Expected calendar month start %v, got %v", expectedStart, start)
			}
			if !end.Equal(expectedEnd) {
				t.Errorf("Expected calendar month end %v, got %v", expectedEnd, end)
			}
		})
	}
}

func TestBillingPeriod_UsageSummary_IncludesPeriodDates(t *testing.T) {
	// Verify GetUsage returns correct period start/end dates
	svc, db, _ := createTestServiceWithBillingCycle(t, 15)
	defer db.Close()

	ctx := context.Background()

	usage, err := svc.GetUsage(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("GetUsage() returned error: %v", err)
	}

	// Verify period dates are set
	if usage.PeriodStart.IsZero() {
		t.Error("Expected PeriodStart to be set")
	}
	if usage.PeriodEnd.IsZero() {
		t.Error("Expected PeriodEnd to be set")
	}
	if usage.ResetDate.IsZero() {
		t.Error("Expected ResetDate to be set")
	}

	// ResetDate should be after PeriodEnd
	if !usage.ResetDate.After(usage.PeriodEnd) {
		t.Errorf("Expected ResetDate (%v) to be after PeriodEnd (%v)", usage.ResetDate, usage.PeriodEnd)
	}

	// PeriodStart should be before PeriodEnd
	if !usage.PeriodStart.Before(usage.PeriodEnd) {
		t.Errorf("Expected PeriodStart (%v) to be before PeriodEnd (%v)", usage.PeriodStart, usage.PeriodEnd)
	}
}

func TestBillingPeriod_MultipleUsersIndependentPeriods(t *testing.T) {
	// Different users can have different billing cycles
	svc, db, provider := createTestServiceWithBillingCycle(t, 15)
	defer db.Close()

	ctx := context.Background()

	// First user with day-15 cycle
	_, err := svc.Charge(ctx, ChargeRequest{
		UserIdentity: "user1@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("User1 charge returned error: %v", err)
	}

	// Change the billing cycle for "user2"
	provider.Entitlement = &entitlement.Entitlement{
		Tier:              entitlement.TierPro,
		Status:            entitlement.StatusActive,
		BillingCycleStart: 1, // Different cycle
	}

	_, err = svc.Charge(ctx, ChargeRequest{
		UserIdentity: "user2@example.com",
		Operation:    OpAIWorkflowGenerate,
		IsBYOK:       false,
	})
	if err != nil {
		t.Fatalf("User2 charge returned error: %v", err)
	}

	// Get both users' billing months
	var user1Month, user2Month string
	err = db.QueryRow(`SELECT billing_month FROM credit_usage WHERE user_identity = ?`, "user1@example.com").Scan(&user1Month)
	if err != nil {
		t.Fatalf("Failed to get user1 billing month: %v", err)
	}
	err = db.QueryRow(`SELECT billing_month FROM credit_usage WHERE user_identity = ?`, "user2@example.com").Scan(&user2Month)
	if err != nil {
		t.Fatalf("Failed to get user2 billing month: %v", err)
	}

	// Billing months should be independent (but may be the same if both cycle days
	// result in the same period start for the current date)
	// Just verify both were recorded
	if user1Month == "" || user2Month == "" {
		t.Error("Both users should have billing months recorded")
	}
}

// ============================================================================
// Edge Case Tests for Invalid/Out-of-Range Billing Days
// ============================================================================

func TestGetBillingMonth_Day29_FallsBackToCalendarMonth(t *testing.T) {
	// Day 29 is invalid (> 28), should fall back to calendar month behavior
	ent := &entitlement.Entitlement{
		BillingCycleStart: 29,
	}

	// Test in February (which never has day 29 in non-leap years)
	testDate := time.Date(2025, 2, 15, 12, 0, 0, 0, time.UTC) // 2025 is not a leap year
	start, end := ent.GetBillingPeriod(testDate)

	// Should fall back to calendar month: Feb 1 - Feb 28
	expectedStart := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected calendar month start %v for day 29, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected calendar month end %v for day 29, got %v", expectedEnd, end)
	}
}

func TestGetBillingMonth_Day30_FallsBackToCalendarMonth(t *testing.T) {
	// Day 30 is invalid (> 28), should fall back to calendar month behavior
	ent := &entitlement.Entitlement{
		BillingCycleStart: 30,
	}

	// Test in any month
	testDate := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	start, end := ent.GetBillingPeriod(testDate)

	// Should fall back to calendar month: June 1 - June 30
	expectedStart := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected calendar month start %v for day 30, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected calendar month end %v for day 30, got %v", expectedEnd, end)
	}
}

func TestGetBillingMonth_FebruaryLeapYear_HandledCorrectly(t *testing.T) {
	// Test February in a leap year (2024 is a leap year)
	ent := &entitlement.Entitlement{
		BillingCycleStart: 15,
	}

	// Feb 20, 2024 (leap year) should be in the Feb 15 - Mar 14 period
	leapDate := time.Date(2024, 2, 20, 12, 0, 0, 0, time.UTC)
	start, end := ent.GetBillingPeriod(leapDate)

	expectedStart := time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !start.Equal(expectedStart) {
		t.Errorf("Leap year Feb: Expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Leap year Feb: Expected end %v, got %v", expectedEnd, end)
	}
}

func TestGetBillingMonth_February_Day28Cycle(t *testing.T) {
	// Day 28 cycle in February - should work in both leap and non-leap years
	ent := &entitlement.Entitlement{
		BillingCycleStart: 28,
	}

	testCases := []struct {
		name          string
		date          time.Time
		expectedStart time.Time
		expectedEnd   time.Time
	}{
		{
			name:          "leap year Feb 28",
			date:          time.Date(2024, 2, 28, 12, 0, 0, 0, time.UTC),
			expectedStart: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			name:          "leap year Feb 29 (day after cycle start)",
			date:          time.Date(2024, 2, 29, 12, 0, 0, 0, time.UTC),
			expectedStart: time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2024, 3, 28, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			name:          "non-leap year Feb 28",
			date:          time.Date(2025, 2, 28, 12, 0, 0, 0, time.UTC),
			expectedStart: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2025, 3, 28, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
		{
			name:          "non-leap year Mar 1 (day after Feb 28)",
			date:          time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC),
			expectedStart: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC),
			expectedEnd:   time.Date(2025, 3, 28, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := ent.GetBillingPeriod(tc.date)

			if !start.Equal(tc.expectedStart) {
				t.Errorf("Expected start %v, got %v", tc.expectedStart, start)
			}
			if !end.Equal(tc.expectedEnd) {
				t.Errorf("Expected end %v, got %v", tc.expectedEnd, end)
			}
		})
	}
}

func TestGetBillingMonth_FebruaryNonLeapYear_Day15Cycle(t *testing.T) {
	// Ensure mid-month cycle works correctly in February (non-leap year)
	ent := &entitlement.Entitlement{
		BillingCycleStart: 15,
	}

	// Feb 10, 2025 (before cycle day, non-leap year)
	testDate := time.Date(2025, 2, 10, 12, 0, 0, 0, time.UTC)
	start, end := ent.GetBillingPeriod(testDate)

	// Should be in Jan 15 - Feb 14 period
	expectedStart := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	expectedEnd := time.Date(2025, 2, 15, 0, 0, 0, 0, time.UTC).Add(-time.Nanosecond)

	if !start.Equal(expectedStart) {
		t.Errorf("Expected start %v, got %v", expectedStart, start)
	}
	if !end.Equal(expectedEnd) {
		t.Errorf("Expected end %v, got %v", expectedEnd, end)
	}
}
