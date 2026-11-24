package commerce

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type mb17BrowserCreditLimits struct{ value int64 }

func (l mb17BrowserCreditLimits) GetLimit(_ context.Context, tier, key string, _ *string) (*TierLimit, error) {
	return &TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: l.value}, nil
}

func (l mb17BrowserCreditLimits) GetTierLimits(_ context.Context, tier string) ([]TierLimit, error) {
	return []TierLimit{{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: l.value, CostMultiplier: 1_000_000}}, nil
}

func TestBrowserAccountCreditCapSerializesReservationsAndExpiresAbandonedHolds(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, schema := range []string{
		`CREATE TABLE business_account_usage_records (business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,billing_period TEXT NOT NULL,limit_key TEXT NOT NULL,usage_amount INTEGER NOT NULL DEFAULT 0,app_bundle_key TEXT NOT NULL,UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,remaining INTEGER NOT NULL,livemode BOOLEAN NOT NULL,spendable_after DATETIME,reversal_window_ends DATETIME,expires_at DATETIME,status TEXT NOT NULL,settlement_state TEXT NOT NULL)`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,billing_period TEXT NOT NULL,limit_key TEXT NOT NULL,reserved_amount INTEGER NOT NULL,livemode BOOLEAN NOT NULL,actual_amount INTEGER,charged_amount INTEGER NOT NULL DEFAULT 0,overrun_amount INTEGER NOT NULL DEFAULT 0,status TEXT NOT NULL,expires_at DATETIME NOT NULL,created_at DATETIME,settled_at DATETIME,app_bundle_key TEXT NOT NULL DEFAULT 'unattributed',model TEXT NOT NULL DEFAULT 'unknown',cost_micros INTEGER NOT NULL DEFAULT 0,prompt_tokens INTEGER NOT NULL DEFAULT 0,completion_tokens INTEGER NOT NULL DEFAULT 0,provider TEXT NOT NULL DEFAULT 'openrouter',browser_session_minutes INTEGER NOT NULL DEFAULT 0,browser_actions INTEGER NOT NULL DEFAULT 0,browser_bytes INTEGER NOT NULL DEFAULT 0,browser_run_id TEXT NOT NULL DEFAULT '',browser_session_id TEXT NOT NULL DEFAULT '',browser_grant_fingerprint TEXT NOT NULL DEFAULT '',browser_tier TEXT NOT NULL DEFAULT '',browser_evidence_fingerprint TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT NOT NULL,gateway_root_operation_id TEXT NOT NULL UNIQUE,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,request_fingerprint TEXT NOT NULL,reserved_amount INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL,PRIMARY KEY(reservation_id,gateway_root_operation_id))`,
	} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatalf("create fixture schema: %v", err)
		}
	}

	now := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	for _, account := range []string{"mb17-account-a", "mb17-account-b"} {
		if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,remaining,livemode,status,settlement_state) VALUES(?,?,100,0,'open','settled')`, "lot-"+account, account); err != nil {
			t.Fatal(err)
		}
	}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{
		DB: db, Limits: mb17BrowserCreditLimits{value: 100}, Dialect: "sqlite",
		Now: func() time.Time { return now }, LiveMode: boolRef(false),
	})
	ctx := context.Background()

	// Two concurrent browser-session holds race against one account-wide
	// subscription cap. The existing LPBS account transaction serializes them.
	type result struct {
		id  string
		err error
	}
	results := make(chan result, 2)
	start := make(chan struct{})
	var callers sync.WaitGroup
	for i := 0; i < 2; i++ {
		callers.Add(1)
		go func(i int) {
			defer callers.Done()
			<-start
			id, _, err := service.ReserveScopedCreditsWithID(ctx, "mb17-account-a", "actor@example.test", "pro", "ai_credits", "browser-automation-studio", 60, fmt.Sprintf("hold-a-%d", i))
			results <- result{id: id, err: err}
		}(i)
	}
	close(start)
	callers.Wait()
	close(results)
	var accepted int
	var acceptedID string
	for outcome := range results {
		if outcome.err == nil && outcome.id != "" {
			accepted++
			acceptedID = outcome.id
		} else {
			t.Logf("reservation attempt id=%q err=%v", outcome.id, outcome.err)
		}
	}
	if accepted != 1 {
		t.Fatalf("accepted same-account reservations=%d; want exactly 1 under 100-credit limit", accepted)
	}
	var reservedA int64
	if err := db.QueryRow(`SELECT COALESCE(SUM(reserved_amount),0) FROM business_account_credit_reservations WHERE business_account_id=? AND status='pending'`, "mb17-account-a").Scan(&reservedA); err != nil {
		t.Fatal(err)
	}
	if reservedA != 60 {
		t.Fatalf("account A pending hold=%d; want 60", reservedA)
	}
	if id, created, err := service.ReserveScopedCreditsWithID(ctx, "mb17-account-a", "actor@example.test", "pro", "ai_credits", "browser-automation-studio", 60, acceptedID); err != nil || created || id != acceptedID {
		t.Fatalf("same-account exact reservation replay id=%q created=%t err=%v", id, created, err)
	}
	if _, _, err := service.ReserveScopedCreditsWithID(ctx, "mb17-account-b", "actor@example.test", "pro", "ai_credits", "browser-automation-studio", 60, acceptedID); err == nil {
		t.Fatal("reservation operation replay must not transfer an account A hold to account B")
	}
	grantFingerprint := strings.Repeat("a", 64)
	requestFingerprint := strings.Repeat("b", 64)
	if err := service.BindBrowserSessionIdentity(ctx, "mb17-account-a", "actor@example.test", acceptedID, "pro", "run-a", "session-a", grantFingerprint); err != nil {
		t.Fatalf("bind browser session to account hold: %v", err)
	}
	if claimed, err := service.BeginBrowserGatewayReceipt(ctx, "mb17-account-a", "actor@example.test", "pro", "browser-automation-studio", acceptedID, "run-a", "session-a", grantFingerprint, "gateway-op-1", requestFingerprint, 60); err != nil || !claimed {
		t.Fatalf("first browser inference claim claimed=%t err=%v", claimed, err)
	}
	if claimed, err := service.BeginBrowserGatewayReceipt(ctx, "mb17-account-a", "actor@example.test", "pro", "browser-automation-studio", acceptedID, "run-a", "session-a", grantFingerprint, "gateway-op-2", strings.Repeat("c", 64), 50); err == nil || claimed {
		t.Fatalf("inference above the account cap must be refused before dispatch claimed=%t err=%v", claimed, err)
	}
	var childCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM business_account_browser_gateway_receipts WHERE reservation_id=?`, acceptedID).Scan(&childCount); err != nil || childCount != 1 {
		t.Fatalf("Gateway child receipts after denied cap claim=%d err=%v; want only first claim", childCount, err)
	}

	// A different account has an independent subscription cap.
	if _, _, err := service.ReserveScopedCreditsWithID(ctx, "mb17-account-b", "actor@example.test", "pro", "ai_credits", "browser-automation-studio", 60, "hold-b-1"); err != nil {
		t.Fatalf("account B should have an independent 100-credit cap: %v", err)
	}

	// Existing LPBS hold retention is ten minutes. Expiry applies to billing
	// reservations only; this is not a browser artifact/history retention claim.
	now = now.Add(11 * time.Minute)
	expired, err := service.CleanupExpiredReservations(ctx)
	if err != nil || expired != 2 {
		t.Fatalf("expired reservations=%d err=%v; want 2", expired, err)
	}
	if _, _, err := service.ReserveScopedCreditsWithID(ctx, "mb17-account-a", "actor@example.test", "pro", "ai_credits", "browser-automation-studio", 60, "hold-a-after-expiry"); err != nil {
		t.Fatalf("expired billing holds must release account cap: %v", err)
	}
}
