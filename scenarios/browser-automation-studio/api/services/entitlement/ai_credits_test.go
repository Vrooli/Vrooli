package entitlement

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// ============================================================================
// Mock Database for Testing
// ============================================================================

// mockDB implements a minimal SQL database interface for testing.
type mockDB struct {
	usage    map[string]*mockUsageRecord
	requests []AIRequestLog
}

type mockUsageRecord struct {
	creditsUsed   int
	requestsCount int
}

func newMockDB() *mockDB {
	return &mockDB{
		usage:    make(map[string]*mockUsageRecord),
		requests: []AIRequestLog{},
	}
}

// ============================================================================
// AICreditsUsage Tests
// ============================================================================

func TestAICreditsUsage_Calculations(t *testing.T) {
	tests := []struct {
		name             string
		creditsUsed      int
		creditsLimit     int
		wantRemaining    int
		wantIsUnlimited  bool
	}{
		{
			name:            "unlimited credits",
			creditsUsed:     50,
			creditsLimit:    -1,
			wantRemaining:   -1,
			wantIsUnlimited: true,
		},
		{
			name:            "limited with credits remaining",
			creditsUsed:     30,
			creditsLimit:    100,
			wantRemaining:   70,
			wantIsUnlimited: false,
		},
		{
			name:            "limited with no credits remaining",
			creditsUsed:     100,
			creditsLimit:    100,
			wantRemaining:   0,
			wantIsUnlimited: false,
		},
		{
			name:            "overused (negative becomes 0)",
			creditsUsed:     150,
			creditsLimit:    100,
			wantRemaining:   0,
			wantIsUnlimited: false,
		},
		{
			name:            "no access tier",
			creditsUsed:     0,
			creditsLimit:    0,
			wantRemaining:   0,
			wantIsUnlimited: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &AICreditsUsage{
				CreditsUsed:  tt.creditsUsed,
				CreditsLimit: tt.creditsLimit,
			}

			// Calculate remaining
			remaining := tt.creditsLimit - tt.creditsUsed
			if tt.creditsLimit < 0 {
				remaining = -1
			} else if remaining < 0 {
				remaining = 0
			}
			usage.CreditsRemaining = remaining

			if usage.CreditsRemaining != tt.wantRemaining {
				t.Errorf("CreditsRemaining = %d, want %d", usage.CreditsRemaining, tt.wantRemaining)
			}

			isUnlimited := tt.creditsLimit < 0
			if isUnlimited != tt.wantIsUnlimited {
				t.Errorf("isUnlimited = %v, want %v", isUnlimited, tt.wantIsUnlimited)
			}
		})
	}
}

func TestAICreditsTracker_CanUseCredits_Unlimited(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create tracker without DB - unlimited doesn't need DB
	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	ctx := context.Background()
	canUse, remaining, err := tracker.CanUseCredits(ctx, "test@example.com", -1)

	if err != nil {
		t.Fatalf("unexpected error for unlimited: %v", err)
	}
	if !canUse {
		t.Error("expected canUse = true for unlimited")
	}
	if remaining != -1 {
		t.Errorf("expected remaining = -1, got %d", remaining)
	}
}

func TestAICreditsTracker_CanUseCredits_NoAccess(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create tracker without DB - no access doesn't need DB
	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	ctx := context.Background()
	canUse, remaining, err := tracker.CanUseCredits(ctx, "test@example.com", 0)

	if err != nil {
		t.Fatalf("unexpected error for no access: %v", err)
	}
	if canUse {
		t.Error("expected canUse = false for no access tier")
	}
	if remaining != 0 {
		t.Errorf("expected remaining = 0, got %d", remaining)
	}
}

func TestAICreditsTracker_CanUseCredits_WithCache(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create tracker with cached usage
	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	// Pre-populate cache with 30 credits used
	currentMonth := time.Now().Format("2006-01")
	tracker.setCached("test@example.com", currentMonth, 30, 5)

	ctx := context.Background()
	canUse, remaining, err := tracker.CanUseCredits(ctx, "test@example.com", 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canUse {
		t.Error("expected canUse = true when credits available")
	}
	if remaining != 70 {
		t.Errorf("expected remaining = 70, got %d", remaining)
	}
}

func TestAICreditsTracker_CanUseCredits_ExhaustedWithCache(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	// Create tracker with cached usage at limit
	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	// Pre-populate cache with 100 credits used (at limit)
	currentMonth := time.Now().Format("2006-01")
	tracker.setCached("test@example.com", currentMonth, 100, 20)

	ctx := context.Background()
	canUse, remaining, err := tracker.CanUseCredits(ctx, "test@example.com", 100)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canUse {
		t.Error("expected canUse = false when credits exhausted")
	}
	if remaining != 0 {
		t.Errorf("expected remaining = 0, got %d", remaining)
	}
}

func TestAICreditsTracker_Cache(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	userIdentity := "test@example.com"
	currentMonth := time.Now().Format("2006-01")

	// Test cache miss
	cached := tracker.getCached(userIdentity, currentMonth)
	if cached != nil {
		t.Error("expected cache miss for new user")
	}

	// Set cache
	tracker.setCached(userIdentity, currentMonth, 50, 10)

	// Test cache hit
	cached = tracker.getCached(userIdentity, currentMonth)
	if cached == nil {
		t.Fatal("expected cache hit after setting")
	}
	if cached.creditsUsed != 50 {
		t.Errorf("expected creditsUsed = 50, got %d", cached.creditsUsed)
	}
	if cached.requestsCount != 10 {
		t.Errorf("expected requestsCount = 10, got %d", cached.requestsCount)
	}

	// Test cache invalidation
	tracker.invalidateCache(userIdentity)
	cached = tracker.getCached(userIdentity, currentMonth)
	if cached != nil {
		t.Error("expected cache miss after invalidation")
	}
}

func TestAICreditsTracker_CacheExpiry(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	userIdentity := "test@example.com"
	currentMonth := time.Now().Format("2006-01")

	// Set cache with old timestamp
	tracker.cacheMu.Lock()
	tracker.cache[userIdentity] = &aiCreditsCache{
		creditsUsed:   50,
		requestsCount: 10,
		month:         currentMonth,
		updatedAt:     time.Now().Add(-2 * time.Minute), // 2 minutes ago (expired)
	}
	tracker.cacheMu.Unlock()

	// Should return nil for expired cache
	cached := tracker.getCached(userIdentity, currentMonth)
	if cached != nil {
		t.Error("expected cache miss for expired entry")
	}
}

func TestAICreditsTracker_DifferentMonth(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	userIdentity := "test@example.com"
	currentMonth := time.Now().Format("2006-01")
	lastMonth := time.Now().AddDate(0, -1, 0).Format("2006-01")

	// Set cache for last month
	tracker.setCached(userIdentity, lastMonth, 50, 10)

	// Should not hit cache for current month
	cached := tracker.getCached(userIdentity, currentMonth)
	if cached != nil {
		t.Error("expected cache miss for different month")
	}
}

// ============================================================================
// AIRequestLog Tests
// ============================================================================

func TestAIRequestLog_Fields(t *testing.T) {
	log := &AIRequestLog{
		UserIdentity:     "test@example.com",
		RequestType:      "workflow_generate",
		Model:            "gpt-4",
		PromptTokens:     100,
		CompletionTokens: 50,
		CreditsCharged:   1,
		Success:          true,
		ErrorMessage:     "",
		DurationMs:       1500,
	}

	if log.UserIdentity != "test@example.com" {
		t.Errorf("expected UserIdentity 'test@example.com', got %q", log.UserIdentity)
	}
	if log.RequestType != "workflow_generate" {
		t.Errorf("expected RequestType 'workflow_generate', got %q", log.RequestType)
	}
	if !log.Success {
		t.Error("expected Success to be true")
	}
}

// ============================================================================
// Helper Function Tests
// ============================================================================

func TestFirstDayOfMonth(t *testing.T) {
	now := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	first := firstDayOfMonth(now)

	expected := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	if !first.Equal(expected) {
		t.Errorf("firstDayOfMonth() = %v, want %v", first, expected)
	}
}

func TestLastDayOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		wantDay  int
	}{
		{
			name:    "January (31 days)",
			input:   time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			wantDay: 31,
		},
		{
			name:    "February leap year (29 days)",
			input:   time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC),
			wantDay: 29,
		},
		{
			name:    "February non-leap year (28 days)",
			input:   time.Date(2023, 2, 15, 0, 0, 0, 0, time.UTC),
			wantDay: 28,
		},
		{
			name:    "April (30 days)",
			input:   time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC),
			wantDay: 30,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			last := lastDayOfMonth(tt.input)
			if last.Day() != tt.wantDay {
				t.Errorf("lastDayOfMonth() day = %d, want %d", last.Day(), tt.wantDay)
			}
		})
	}
}

// ============================================================================
// Integration Tests (require real DB - skipped without connection)
// ============================================================================

func TestAICreditsTracker_Integration(t *testing.T) {
	// This test requires a real database connection
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Placeholder for integration tests
	// These would test:
	// - GetAICreditsUsage with real DB
	// - ChargeCredits with real DB
	// - LogRequest with real DB
	// - Concurrent access patterns
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkAICreditsTracker_Cache(b *testing.B) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	tracker := &AICreditsTracker{
		log:   log,
		cache: make(map[string]*aiCreditsCache),
	}

	currentMonth := time.Now().Format("2006-01")
	tracker.setCached("test@example.com", currentMonth, 50, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.getCached("test@example.com", currentMonth)
	}
}
