package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	reservationstate "landing-page-business-suite-api/internal/commerce/reservation/flow/generated"
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
		`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY, customer_email TEXT NOT NULL, balance_credits INTEGER NOT NULL DEFAULT 0, bonus_credits INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, source_kind TEXT NOT NULL, provider TEXT NOT NULL, provider_ref TEXT, livemode BOOLEAN, settlement_state TEXT NOT NULL, spendable_after DATETIME, reversal_window_ends DATETIME, granted INTEGER NOT NULL, remaining INTEGER NOT NULL, expires_at DATETIME, status TEXT NOT NULL, created_at DATETIME, UNIQUE(business_account_id,provider,provider_ref))`,
		`CREATE TABLE credit_lot_allocations (id TEXT PRIMARY KEY, lot_id TEXT NOT NULL, operation_id TEXT NOT NULL, credits INTEGER NOT NULL, cost_micros INTEGER NOT NULL, created_at DATETIME, UNIQUE(lot_id,operation_id))`,
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL DEFAULT 0, app_bundle_key TEXT NOT NULL, last_operation_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, livemode BOOLEAN NOT NULL DEFAULT FALSE, actual_amount INTEGER, charged_amount INTEGER NOT NULL DEFAULT 0, overrun_amount INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, account := range []string{"acct-alpha", "acct-beta", "acct-gamma"} {
		if _, err := db.Exec(`INSERT INTO business_account_credit_wallets(business_account_id,customer_email,balance_credits) VALUES(?,?,1000)`, account, "buyer@example.com"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES(?,?, 'test','fixture',?,FALSE,'settled',1000,1000,'open',?)`, "test-grant-"+account, account, "test-grant-"+account, now); err != nil {
			t.Fatal(err)
		}
	}

	ids := []string{"reservation-one", "reservation-two", "reservation-three", "reservation-four"}
	generatedID := 0
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{
		DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "sqlite",
		Now: func() time.Time { return now }, LiveMode: boolRef(false), NewID: func() string {
			if len(ids) == 0 {
				generatedID++
				return fmt.Sprintf("generated-%d", generatedID)
			}
			id := ids[0]
			ids = ids[1:]
			return id
		},
	})
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,expires_at,status,created_at) VALUES('expiring-first','acct-alpha','paid','test','expiring-first',FALSE,'settled',15,15,?,'open',?)`, now.Add(30*24*time.Hour), now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
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
	var allocated, earliestRemaining int64
	if err := db.QueryRow(`SELECT SUM(credits) FROM credit_lot_allocations WHERE operation_id=?`, first).Scan(&allocated); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE id='expiring-first'`).Scan(&earliestRemaining); err != nil {
		t.Fatal(err)
	}
	if allocated != 27 || earliestRemaining != 0 {
		t.Fatalf("lot allocation total=%d expiring remainder=%d; want 27 and 0", allocated, earliestRemaining)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES('test-only','acct-test','paid','stripe','test-only',FALSE,'settled',80,80,'open'),('held-live','acct-test','paid','stripe','held-live',TRUE,'pending',80,80,'open')`); err != nil {
		t.Fatal(err)
	}
	liveService := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "sqlite", Now: func() time.Time { return now }, LiveMode: boolRef(true)})
	if _, err := liveService.ReserveCredits(ctx, "acct-test", "buyer@example.com", "pro", "ai_credits", 1); err == nil {
		t.Fatal("test-mode and unsettled lots must not fund the live-mode route")
	}
	assertReservationOverrunCapped(t, ctx, db, service, now)

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
	if eventCount != 3 {
		t.Fatalf("account usage audience count = %d, want 3 including the overrun account", eventCount)
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
	expiringHold, err := service.ReserveCredits(ctx, "acct-gamma", "buyer@example.com", "pro", "ai_credits", 100)
	if err != nil {
		t.Fatalf("released hold must no longer consume capacity: %v", err)
	}
	if err := service.RecordUsage(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 2, "usage-replay-one", "web-console", "model", 77, 1, 1, "provider"); err != nil {
		t.Fatalf("record direct usage: %v", err)
	}
	if err := service.RecordUsage(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 2, "usage-replay-one", "web-console", "model", 77, 1, 1, "provider"); err != nil {
		t.Fatalf("replay direct usage: %v", err)
	}
	if err := service.RecordUsage(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 3, "usage-replay-one", "web-console", "model", 77, 1, 1, "provider"); err == nil {
		t.Fatal("conflicting direct usage replay must be rejected")
	}
	var replayAllocations, replayEvents, finalUsage int64
	if err := db.QueryRow(`SELECT COUNT(*) FROM credit_lot_allocations WHERE operation_id='usage-replay-one'`).Scan(&replayAllocations); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_usage_events WHERE operation_id='usage-replay-one'`).Scan(&replayEvents); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT usage_amount FROM business_account_usage_records WHERE business_account_id='acct-alpha' AND limit_key='ai_credits'`).Scan(&finalUsage); err != nil {
		t.Fatal(err)
	}
	if replayAllocations != 1 || replayEvents != 1 || finalUsage != 29 {
		t.Fatalf("replay state allocations=%d events=%d usage=%d; want 1,1,29", replayAllocations, replayEvents, finalUsage)
	}
	// A terminal reservation never reopens, and an expired hold cannot settle.
	if err := service.ReleaseReservation(ctx, "acct-alpha", first); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE business_account_credit_reservations SET expires_at=? WHERE id=?`, now.Add(-time.Second), expiringHold); err != nil {
		t.Fatal(err)
	}
	if count, err := service.CleanupExpiredReservations(ctx); err != nil || count != 1 {
		t.Fatalf("expire pending holds count=%d error=%v; want 1", count, err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-gamma", expiringHold, 1, "web-console", "model", 1, 1, 1, "provider"); err == nil {
		t.Fatal("an expired reservation must not settle")
	}
	if err := service.RetainUncertainReservation(ctx, "acct-gamma", expiringHold); err != nil {
		t.Fatal(err)
	}
	if err := service.ReleaseReservation(ctx, "acct-gamma", expiringHold); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{first: "finalized", expiringHold: "expired"} {
		var status string
		if err := db.QueryRow(`SELECT status FROM business_account_credit_reservations WHERE id=?`, id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != want {
			t.Fatalf("reservation %s status=%s; want %s", id, status, want)
		}
	}
}

func assertReservationOverrunCapped(t *testing.T, ctx context.Context, db *sql.DB, service *BusinessAccountUsageService, now time.Time) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets(business_account_id,customer_email,balance_credits) VALUES('acct-overrun','buyer@example.com',10)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES('overrun-lot','acct-overrun','test','fixture','overrun-lot',FALSE,'settled',10,10,'open',?)`, now); err != nil {
		t.Fatal(err)
	}
	overrunID, err := service.ReserveCredits(ctx, "acct-overrun", "buyer@example.com", "pro", "ai_credits", 10)
	if err != nil {
		t.Fatalf("reserve overrun test credits: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-overrun", overrunID, 13, "web-console", "openrouter/test", 9876, 3, 4, "openrouter"); err != nil {
		t.Fatalf("settle overrun reservation: %v", err)
	}
	var overrunStatus string
	var observed, charged, overrun, remaining, eventCredits, eventCost int64
	if err := db.QueryRow(`SELECT status,actual_amount,charged_amount,overrun_amount FROM business_account_credit_reservations WHERE id=?`, overrunID).Scan(&overrunStatus, &observed, &charged, &overrun); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE id='overrun-lot'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT credits,cost_micros FROM business_account_usage_events WHERE operation_id=?`, overrunID).Scan(&eventCredits, &eventCost); err != nil {
		t.Fatal(err)
	}
	if overrunStatus != string(reservationstate.ReservationOverrun) || observed != 13 || charged != 10 || overrun != 3 || remaining != 0 || eventCredits != 10 || eventCost != 9876 {
		t.Fatalf("overrun settlement status=%s observed=%d charged=%d overrun=%d remaining=%d event credits=%d provider cost=%d", overrunStatus, observed, charged, overrun, remaining, eventCredits, eventCost)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-overrun", overrunID, 13, "web-console", "openrouter/test", 9876, 3, 4, "openrouter"); err != nil {
		t.Fatalf("replay overrun settlement: %v", err)
	}
	if err := service.FinalizeReservationWithCost(ctx, "acct-overrun", overrunID, 12, "web-console", "openrouter/test", 9876, 3, 4, "openrouter"); err == nil {
		t.Fatal("conflicting overrun replay must fail")
	}
	if err := service.ReleaseReservation(ctx, "acct-overrun", overrunID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT status FROM business_account_credit_reservations WHERE id=?`, overrunID).Scan(&overrunStatus); err != nil || overrunStatus != string(reservationstate.ReservationOverrun) {
		t.Fatalf("release must leave terminal overrun state unchanged: status=%s err=%v", overrunStatus, err)
	}
	var allocated, allocationCount int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(credits),0),COUNT(*) FROM credit_lot_allocations WHERE operation_id=?`, overrunID).Scan(&allocated, &allocationCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE id='overrun-lot'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if allocated != charged || allocationCount != 1 || remaining != 0 {
		t.Fatalf("overrun replay allocations=%d count=%d lot remaining=%d; want charged=%d once and zero remaining", allocated, allocationCount, remaining, charged)
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
		`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY, customer_email TEXT NOT NULL, balance_credits INTEGER NOT NULL DEFAULT 0, bonus_credits INTEGER NOT NULL DEFAULT 0, updated_at DATETIME)`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, source_kind TEXT NOT NULL, provider TEXT NOT NULL, provider_ref TEXT, livemode BOOLEAN, settlement_state TEXT NOT NULL, spendable_after DATETIME, reversal_window_ends DATETIME, granted INTEGER NOT NULL, remaining INTEGER NOT NULL, expires_at DATETIME, status TEXT NOT NULL, created_at DATETIME, UNIQUE(business_account_id,provider,provider_ref))`,
		`CREATE TABLE credit_lot_allocations (id TEXT PRIMARY KEY, lot_id TEXT NOT NULL, operation_id TEXT NOT NULL, credits INTEGER NOT NULL, cost_micros INTEGER NOT NULL, created_at DATETIME, UNIQUE(lot_id,operation_id))`,
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL DEFAULT 0, app_bundle_key TEXT NOT NULL, last_operation_at DATETIME, created_at DATETIME, updated_at DATETIME, UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, livemode BOOLEAN NOT NULL DEFAULT FALSE, actual_amount INTEGER, charged_amount INTEGER NOT NULL DEFAULT 0, overrun_amount INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	insufficient := errors.New("insufficient credits")
	var idMu sync.Mutex
	nextID := 0
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets(business_account_id,customer_email,balance_credits) VALUES(?,?,1000)`, "acct-concurrent", "buyer@example.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status) VALUES('parallel-test-lot','acct-concurrent','test','fixture','parallel-test-lot',FALSE,'settled',1000,1000,'open')`); err != nil {
		t.Fatal(err)
	}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{
		DB: db, Limits: fixedAccountLimits{value: 100}, Dialect: "sqlite",
		LiveMode:     boolRef(false),
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

func TestUsageServicePropagatesConfiguredFundingMode(t *testing.T) {
	liveMode := false
	service := NewUsageServiceWithOptions(UsageServiceOptions{LiveMode: &liveMode})
	if got := service.businessAccountUsageService().liveMode; got {
		t.Fatal("configured test-mode route became live-mode")
	}
}
