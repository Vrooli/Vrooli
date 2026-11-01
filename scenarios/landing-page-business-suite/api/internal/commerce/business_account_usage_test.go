package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type fixedAccountLimits struct{ value int64 }

func (l fixedAccountLimits) GetLimit(_ context.Context, tier, key string, _ *string) (*TierLimit, error) {
	return &TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: l.value}, nil
}

func (l fixedAccountLimits) GetTierLimits(_ context.Context, tier string) ([]TierLimit, error) {
	return []TierLimit{{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: l.value, CostMultiplier: 1000000}}, nil
}

func TestBusinessAccountUsageReservationIsAccountScopedAndIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL DEFAULT 0, app_bundle_key TEXT NOT NULL, last_operation_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, actual_amount INTEGER, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}

	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	ids := []string{"reservation-one", "reservation-two", "reservation-three", "reservation-four"}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{
		DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "sqlite",
		Now: func() time.Time { return now }, NewID: func() string { id := ids[0]; ids = ids[1:]; return id },
	})
	ctx := context.Background()
	first, err := service.ReserveCredits(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 40)
	if err != nil {
		t.Fatalf("reserve first account credits: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-beta", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter"); err == nil {
		t.Fatal("reservation must not settle against a different business account")
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-alpha", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter"); err != nil {
		t.Fatalf("settle measured usage: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-alpha", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter"); err != nil {
		t.Fatalf("retry identical settlement: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-alpha", first, 27, "web-console", "openrouter/test", 9999, 11, 7, "openrouter"); err == nil {
		t.Fatal("expected conflicting settlement metadata to be rejected")
	}

	var accountID, actor string
	var amount, eventCount int64
	if err := db.QueryRow(`SELECT business_account_id,actor_identity,usage_amount FROM business_account_usage_records WHERE billing_period=?`, now.Format("2006-01")).Scan(&accountID, &actor, &amount); err != nil {
		t.Fatalf("read account usage: %v", err)
	}
	if accountID != "acct-alpha" || actor != "buyer@example.com" || amount != 27 {
		t.Fatalf("account usage attribution = account %q actor %q amount %d", accountID, actor, amount)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id=?`, first).Scan(&eventCount); err != nil {
		t.Fatalf("count idempotent usage events: %v", err)
	}
	if eventCount != 1 {
		t.Fatalf("usage events after idempotent retry = %d, want 1", eventCount)
	}

	second, err := service.ReserveCredits(ctx, "acct-beta", "buyer@example.com", "pro", "ai_credits", 90)
	if err != nil {
		t.Fatalf("other account must have its own limit: %v", err)
	}
	summary, err := service.GetUsageSummary(ctx, "acct-beta", "pro")
	if err != nil {
		t.Fatalf("read account usage summary: %v", err)
	}
	if summary.BusinessAccountID != "acct-beta" || summary.Usage["ai_credits"] != 0 || summary.Reserved["ai_credits"] != 90 || summary.Remaining["ai_credits"] != 10 {
		t.Fatalf("account summary does not reflect its pending hold: %#v", summary)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-beta", second, 90, "web-console", "openrouter/test", 4567, 20, 30, "openrouter"); err != nil {
		t.Fatalf("settle other account usage: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(DISTINCT business_account_id) FROM business_account_usage_records`).Scan(&eventCount); err != nil {
		t.Fatalf("count account usage audiences: %v", err)
	}
	if eventCount != 2 {
		t.Fatalf("account usage audience count = %d, want 2", eventCount)
	}

	if _, err := service.ReserveCredits(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 74); err == nil {
		t.Fatal("expected account-specific limit to reject usage beyond 100")
	}

	pending, err := service.ReserveCredits(ctx, "acct-gamma", "buyer@example.com", "pro", "ai_credits", 60)
	if err != nil {
		t.Fatalf("reserve pending account hold: %v", err)
	}
	if _, err := service.ReserveCredits(ctx, "acct-gamma", "other@example.com", "pro", "ai_credits", 41); err == nil {
		t.Fatal("pending reservations must count toward the account limit")
	}
	if err := service.ReleaseReservation(ctx, "acct-gamma", pending); err != nil {
		t.Fatalf("release account reservation: %v", err)
	}
	if _, err := service.ReserveCredits(ctx, "acct-gamma", "buyer@example.com", "pro", "ai_credits", 100); err != nil {
		t.Fatalf("released hold must no longer consume capacity: %v", err)
	}
}

func TestBusinessAccountUsageConcurrentReservationsStayWithinLimit(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL DEFAULT 0, app_bundle_key TEXT NOT NULL, last_operation_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, actual_amount INTEGER, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}

	insufficient := errors.New("insufficient credits")
	var idMu sync.Mutex
	nextID := 0
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{
		DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "sqlite",
		Insufficient: insufficient,
		NewID: func() string {
			idMu.Lock()
			defer idMu.Unlock()
			nextID++
			return fmt.Sprintf("parallel-reservation-%d", nextID)
		},
	})

	const callers = 4
	start := make(chan struct{})
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	accepted := 0
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, reserveErr := service.ReserveCredits(context.Background(), "acct-concurrent", "buyer@example.com", "pro", "ai_credits", 30)
			resultMu.Lock()
			defer resultMu.Unlock()
			if reserveErr == nil {
				accepted++
			} else if !errors.Is(reserveErr, insufficient) {
				t.Errorf("ReserveCredits() returned unexpected error: %v", reserveErr)
			}
		}()
	}
	close(start)
	wg.Wait()

	if accepted != 3 {
		t.Fatalf("accepted concurrent reservations = %d, want 3 within a 100-credit account limit", accepted)
	}
	var pending int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND status='pending'`, "acct-concurrent").Scan(&pending); err != nil {
		t.Fatalf("read concurrent account holds: %v", err)
	}
	if pending != 90 {
		t.Fatalf("concurrent pending reservations = %d credits, want 90", pending)
	}
}
