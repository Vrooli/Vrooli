package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, livemode BOOLEAN NOT NULL DEFAULT FALSE, actual_amount INTEGER, charged_amount INTEGER NOT NULL DEFAULT 0, overrun_amount INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter', browser_session_minutes INTEGER NOT NULL DEFAULT 0, browser_actions INTEGER NOT NULL DEFAULT 0, browser_bytes INTEGER NOT NULL DEFAULT 0, browser_run_id TEXT NOT NULL DEFAULT '', browser_session_id TEXT NOT NULL DEFAULT '', browser_grant_fingerprint TEXT NOT NULL DEFAULT '', browser_tier TEXT NOT NULL DEFAULT '', browser_evidence_fingerprint TEXT NOT NULL DEFAULT '')`,
		`CREATE UNIQUE INDEX idx_test_browser_evidence_reservation_unique ON business_account_credit_reservations(browser_evidence_fingerprint) WHERE browser_evidence_fingerprint <> ''`,
		`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT NOT NULL, gateway_root_operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, request_fingerprint TEXT NOT NULL, reserved_amount INTEGER NOT NULL DEFAULT 0, state TEXT NOT NULL, provider TEXT NOT NULL DEFAULT '', model TEXT NOT NULL DEFAULT '', credits INTEGER NOT NULL DEFAULT 0, cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, receipt_fingerprint TEXT NOT NULL DEFAULT '', created_at DATETIME, settled_at DATETIME, PRIMARY KEY(reservation_id,gateway_root_operation_id))`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, browser_session_minutes INTEGER NOT NULL DEFAULT 0, browser_actions INTEGER NOT NULL DEFAULT 0, browser_bytes INTEGER NOT NULL DEFAULT 0, browser_run_id TEXT NOT NULL DEFAULT '', browser_session_id TEXT NOT NULL DEFAULT '', browser_grant_fingerprint TEXT NOT NULL DEFAULT '', browser_evidence_fingerprint TEXT NOT NULL DEFAULT '', created_at DATETIME)`,
		`CREATE UNIQUE INDEX idx_test_browser_evidence_unique ON business_account_usage_events(browser_evidence_fingerprint) WHERE browser_evidence_fingerprint <> ''`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
	}
	now := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, account := range []string{"acct-alpha", "acct-beta", "acct-gamma", "acct-idempotent", "acct-period"} {
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
	reserveID, created, err := service.ReserveCreditsWithID(ctx, "acct-idempotent", "buyer@example.com", "pro", "ai_credits", 10, "00000000-0000-4000-8000-000000000101")
	if err != nil || !created {
		t.Fatalf("first idempotent reserve id=%q created=%t err=%v", reserveID, created, err)
	}
	replayedReserveID, replayCreated, err := service.ReserveCreditsWithID(ctx, "acct-idempotent", "buyer@example.com", "pro", "ai_credits", 10, "00000000-0000-4000-8000-000000000101")
	if err != nil || replayCreated || replayedReserveID != reserveID {
		t.Fatalf("reserve replay id=%q created=%t err=%v", replayedReserveID, replayCreated, err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,expires_at,status,created_at) VALUES('expiring-first','acct-alpha','paid','test','expiring-first',FALSE,'settled',15,15,?,'open',?)`, now.Add(30*24*time.Hour), now.Add(-24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	first, err := service.ReserveCredits(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 40)
	if err != nil {
		t.Fatalf("reserve first account credits: %v", err)
	}
	usageAdapter := NewUsageServiceWithOptions(UsageServiceOptions{DB: db, LimitsService: fixedAccountLimits{value: 100}, Dialect: "sqlite", LiveMode: boolRef(false)})
	featureReservation, created, err := usageAdapter.ReserveBusinessAccountInferenceCreditsWithID(ctx, "acct-beta", "buyer@example.com", "pro", "web-console", 10, "00000000-0000-4000-8000-000000000201")
	if err != nil || !created || featureReservation == "" {
		t.Fatalf("reserve feature-scoped inference hold id=%q created=%t err=%v", featureReservation, created, err)
	}
	var reservedApp string
	if err := db.QueryRow(`SELECT app_bundle_key FROM business_account_credit_reservations WHERE id=?`, featureReservation).Scan(&reservedApp); err != nil || reservedApp != "web-console" {
		t.Fatalf("inference reservation app=%q err=%v; want web-console", reservedApp, err)
	}
	if _, created, err := usageAdapter.ReserveBusinessAccountInferenceCreditsWithID(ctx, "acct-beta", "buyer@example.com", "pro", "web-console", 10, "00000000-0000-4000-8000-000000000201"); err != nil || created {
		t.Fatalf("feature-scoped operation replay created=%t err=%v; want idempotent replay", created, err)
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "attacker@example.com", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("settlement adapter must reject a different actor")
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-beta", "buyer@example.com", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("reservation must not settle against a different business account")
	}
	fingerprint := strings.Repeat("a", 64)
	if err := usageAdapter.BindBusinessAccountBrowserEvidence(ctx, "acct-alpha", "buyer@example.com", first, "run-17", fingerprint); err != nil {
		t.Fatalf("bind exact BAS evidence to reservation: %v", err)
	}
	if err := usageAdapter.BindBusinessAccountBrowserEvidence(ctx, "acct-alpha", "buyer@example.com", first, "run-17", fingerprint); err != nil {
		t.Fatalf("identical evidence binding retry: %v", err)
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "buyer@example.com", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err != nil {
		t.Fatalf("settle measured usage: %v", err)
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "buyer@example.com", first, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err != nil {
		t.Fatalf("retry identical settlement: %v", err)
	}
	replay, err := usageAdapter.ReserveBusinessAccountCredits(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", 40)
	if err != nil {
		t.Fatalf("reserve replay attempt: %v", err)
	}
	if err := usageAdapter.BindBusinessAccountBrowserEvidence(ctx, "acct-alpha", "buyer@example.com", replay, "run-17", fingerprint); err == nil {
		t.Fatal("replayed BAS evidence must be denied before provider dispatch")
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "buyer@example.com", replay, 27, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("same BAS evidence must not settle a different LPBS reservation")
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "buyer@example.com", replay, 41, "web-console", "openrouter/test", 1234, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-2", strings.Repeat("b", 64)); err == nil {
		t.Fatal("browser receipt settlement above the reserved budget must be denied")
	}
	if err := usageAdapter.FinalizeBusinessAccountReservationWithBrowserUsage(ctx, "acct-alpha", "buyer@example.com", first, 27, "web-console", "openrouter/test", 9999, 11, 7, "openrouter", 12, 19, 4096, "run-17", "bas-session-1", strings.Repeat("a", 64)); err == nil {
		t.Fatal("expected conflicting settlement metadata to be rejected")
	}
	var browserMinutes, browserActions, browserBytes int64
	var browserRunID, browserSessionID, evidenceFingerprint string
	if err := db.QueryRow(`SELECT browser_session_minutes,browser_actions,browser_bytes,browser_run_id,browser_session_id,browser_evidence_fingerprint FROM business_account_usage_events WHERE operation_id=?`, first).Scan(&browserMinutes, &browserActions, &browserBytes, &browserRunID, &browserSessionID, &evidenceFingerprint); err != nil {
		t.Fatal(err)
	}
	if browserMinutes != 12 || browserActions != 19 || browserBytes != 4096 || browserRunID != "run-17" || browserSessionID != "bas-session-1" || evidenceFingerprint != strings.Repeat("a", 64) {
		t.Fatalf("receipt browser usage=%d minutes/%d actions/%d bytes", browserMinutes, browserActions, browserBytes)
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
	retainedInferenceHold, err := service.ReserveCredits(ctx, "acct-beta", "buyer@example.com", "pro", "ai_credits", 2)
	if err != nil {
		t.Fatalf("reserve uncertain hold: %v", err)
	}
	if err := service.RetainUncertainReservation(ctx, "acct-beta", retainedInferenceHold); err != nil {
		t.Fatal(err)
	}
	var retainedExpiry time.Time
	if err := db.QueryRow(`SELECT expires_at FROM business_account_credit_reservations WHERE id=?`, retainedInferenceHold).Scan(&retainedExpiry); err != nil {
		t.Fatal(err)
	}
	if retainedExpiry.Year() != 9999 {
		t.Fatalf("uncertain reservation expiry=%s; want durable retained hold", retainedExpiry)
	}
	if err := service.ReleaseReservation(ctx, "acct-beta", retainedInferenceHold); err != nil {
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
	for id, want := range map[string]string{first: "finalized", expiringHold: "expired"} {
		var status string
		if err := db.QueryRow(`SELECT status FROM business_account_credit_reservations WHERE id=?`, id).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != want {
			t.Fatalf("reservation %s status=%s; want %s", id, status, want)
		}
	}

	sessionHold, created, err := service.ReserveScopedCreditsWithID(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", "web-console", 20, "browser-session-operation")
	if err != nil || !created {
		t.Fatalf("reserve browser session hold id=%q created=%v error=%v", sessionHold, created, err)
	}
	grantFingerprint := strings.Repeat("c", 64)
	if err := service.BindBrowserSessionIdentity(ctx, "acct-alpha", "buyer@example.com", sessionHold, "pro", "run-20", "bas-session-20", grantFingerprint); err != nil {
		t.Fatalf("bind browser session identity: %v", err)
	}
	if err := service.BindBrowserSessionIdentity(ctx, "acct-alpha", "buyer@example.com", sessionHold, "pro", "run-20", "bas-session-20", grantFingerprint); err != nil {
		t.Fatalf("identical session binding retry: %v", err)
	}
	if err := service.BindBrowserSessionIdentity(ctx, "acct-alpha", "buyer@example.com", sessionHold, "different-tier", "run-20", "bas-session-20", grantFingerprint); err == nil {
		t.Fatal("browser session retry cannot change the bound tier")
	}
	if err := service.BindBrowserSessionIdentity(ctx, "acct-alpha", "attacker@example.com", sessionHold, "pro", "run-20", "bas-session-20", grantFingerprint); err == nil {
		t.Fatal("browser session identity must remain account and actor bound")
	}
	requestFingerprint, receiptFingerprint := strings.Repeat("d", 64), strings.Repeat("e", 64)
	rootOperation := "gateway-root-session-20-operation-1"
	claimed, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "pro", "web-console", sessionHold, "run-20", "bas-session-20", grantFingerprint, rootOperation, requestFingerprint, 10)
	if err != nil || !claimed {
		t.Fatalf("claim browser Gateway operation claimed=%v error=%v", claimed, err)
	}
	if _, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "pro", "web-console", sessionHold, "run-20", "bas-session-20", grantFingerprint, rootOperation, requestFingerprint, 10); !errors.Is(err, ErrBrowserGatewayOperationReplay) {
		t.Fatalf("duplicate Gateway operation should refuse redispatch, got %v", err)
	}
	operationReceipt := BrowserGatewayReceipt{RootOperationID: rootOperation, RequestFingerprint: requestFingerprint, State: "settled", Provider: "hosted-provider", Model: "model-a", Credits: 6, CostMicros: 700, PromptTokens: 13, CompletionTokens: 5, ReceiptFingerprint: receiptFingerprint}
	if err := service.RecordBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", sessionHold, operationReceipt); err != nil {
		t.Fatalf("persist immutable Gateway child receipt: %v", err)
	}
	if err := service.RecordBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", sessionHold, operationReceipt); err != nil {
		t.Fatalf("retry identical Gateway child receipt: %v", err)
	}
	operationReceipt.Credits++
	if err := service.RecordBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", sessionHold, operationReceipt); err == nil {
		t.Fatal("conflicting Gateway receipt replay must fail")
	}
	secondRoot := "gateway-root-session-20-operation-2"
	secondRequest := strings.Repeat("f", 64)
	if _, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "pro", "other-app", sessionHold, "run-20", "bas-session-20", grantFingerprint, secondRoot, secondRequest, 10); err == nil {
		t.Fatal("later Gateway operation cannot move a session hold to another app")
	}
	if _, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "different-tier", "web-console", sessionHold, "run-20", "bas-session-20", grantFingerprint, secondRoot, secondRequest, 10); err == nil {
		t.Fatal("later Gateway operation cannot change the session tier")
	}
	if claimed, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "pro", "web-console", sessionHold, "run-20", "bas-session-20", grantFingerprint, secondRoot, secondRequest, 10); err != nil || !claimed {
		t.Fatalf("claim second session Gateway operation claimed=%v error=%v", claimed, err)
	}
	secondReceipt := BrowserGatewayReceipt{RootOperationID: secondRoot, RequestFingerprint: secondRequest, State: "settled", Provider: "hosted-provider", Model: "model-b", Credits: 4, CostMicros: 300, PromptTokens: 1, CompletionTokens: 2, ReceiptFingerprint: strings.Repeat("a", 64)}
	if err := service.RecordBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", sessionHold, secondReceipt); err != nil {
		t.Fatalf("persist second session Gateway child receipt: %v", err)
	}
	var reservedAmount int64
	if err := db.QueryRow(`SELECT reserved_amount FROM business_account_credit_reservations WHERE id=?`, sessionHold).Scan(&reservedAmount); err != nil || reservedAmount != 30 {
		t.Fatalf("session hold should extend once for the second operation, amount=%d error=%v", reservedAmount, err)
	}
	aggregate, err := service.BrowserGatewayReceiptAggregate(ctx, "acct-alpha", "buyer@example.com", sessionHold)
	if err != nil || aggregate.Operations != 2 || aggregate.Credits != 10 || aggregate.CostMicros != 1000 || aggregate.PromptTokens != 14 || aggregate.CompletionTokens != 7 {
		t.Fatalf("browser Gateway receipt aggregate=%+v error=%v", aggregate, err)
	}
	proofFingerprint := strings.Repeat("b", 64)
	if err := service.FinalizeBrowserSessionAggregate(ctx, "acct-alpha", sessionHold, "run-20", "bas-session-20", grantFingerprint, proofFingerprint, "", aggregate, 3, 5, 1024); err != nil {
		t.Fatalf("settle session aggregate after closed proof: %v", err)
	}
	if err := service.FinalizeBrowserSessionAggregate(ctx, "acct-alpha", sessionHold, "run-20", "bas-session-20", grantFingerprint, proofFingerprint, "", aggregate, 3, 5, 1024); err != nil {
		t.Fatalf("retry exact session settlement: %v", err)
	}
	if err := service.FinalizeBrowserSessionAggregate(ctx, "acct-alpha", sessionHold, "run-20", "bas-session-20", grantFingerprint, strings.Repeat("c", 64), "", aggregate, 3, 5, 1024); err == nil {
		t.Fatal("a different proof replay must not change closed-session settlement")
	}
	uncertainHold, created, err := service.ReserveScopedCreditsWithID(ctx, "acct-alpha", "buyer@example.com", "pro", "ai_credits", "web-console", 20, "browser-session-operation-uncertain")
	if err != nil || !created {
		t.Fatalf("reserve uncertain browser session hold: id=%q created=%v error=%v", uncertainHold, created, err)
	}
	uncertainGrant := strings.Repeat("d", 64)
	if err := service.BindBrowserSessionIdentity(ctx, "acct-alpha", "buyer@example.com", uncertainHold, "pro", "run-21", "bas-session-21", uncertainGrant); err != nil {
		t.Fatal(err)
	}
	uncertainRoot := "gateway-root-session-21-operation-1"
	if claimed, err := service.BeginBrowserGatewayReceipt(ctx, "acct-alpha", "buyer@example.com", "pro", "web-console", uncertainHold, "run-21", "bas-session-21", uncertainGrant, uncertainRoot, strings.Repeat("e", 64), 10); err != nil || !claimed {
		t.Fatalf("claim uncertain browser Gateway operation claimed=%v error=%v", claimed, err)
	}
	if err := service.MarkBrowserGatewayOperationUncertain(ctx, "acct-alpha", "buyer@example.com", uncertainHold, uncertainRoot); err != nil {
		t.Fatalf("retain uncertain browser Gateway operation: %v", err)
	}
	if _, err := service.BrowserGatewayReceiptAggregate(ctx, "acct-alpha", "buyer@example.com", uncertainHold); !errors.Is(err, ErrBrowserSessionOperationsPending) {
		t.Fatalf("uncertain child receipt must block session settlement, got %v", err)
	}
	periodHold, created, err := service.ReserveScopedCreditsWithID(ctx, "acct-period", "buyer@example.com", "pro", "ai_credits", "web-console", 20, "browser-session-period-boundary")
	if err != nil || !created {
		t.Fatalf("reserve period-boundary browser session hold: id=%q created=%v err=%v", periodHold, created, err)
	}
	periodGrant := strings.Repeat("9", 64)
	if err := service.BindBrowserSessionIdentity(ctx, "acct-period", "buyer@example.com", periodHold, "pro", "run-period", "bas-period", periodGrant); err != nil {
		t.Fatal(err)
	}
	if claimed, err := service.BeginBrowserGatewayReceipt(ctx, "acct-period", "buyer@example.com", "pro", "web-console", periodHold, "run-period", "bas-period", periodGrant, "gateway-period-op-1", strings.Repeat("8", 64), 1); err != nil || !claimed {
		t.Fatalf("claim first period operation: claimed=%v err=%v", claimed, err)
	}
	if _, err := db.Exec(`INSERT INTO business_account_usage_records(business_account_id,actor_identity,billing_period,limit_key,usage_amount,app_bundle_key) VALUES(?,?,?,?,?,?)`, "acct-period", "buyer@example.com", now.Format("2006-01"), "ai_credits", 75, "web-console"); err != nil {
		t.Fatal(err)
	}
	now = now.AddDate(0, 1, 0)
	if _, err := service.BeginBrowserGatewayReceipt(ctx, "acct-period", "buyer@example.com", "pro", "web-console", periodHold, "run-period", "bas-period", periodGrant, "gateway-period-op-2", strings.Repeat("7", 64), 10); err == nil {
		t.Fatal("session extension must apply the reservation billing period cap across month boundary")
	}
}

func TestGetBrowserSessionReceiptReturnsSettledJointMeterAndScopesByAccount(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,status TEXT NOT NULL,provider TEXT NOT NULL,model TEXT NOT NULL,cost_micros INTEGER NOT NULL,prompt_tokens INTEGER NOT NULL,completion_tokens INTEGER NOT NULL,charged_amount INTEGER NOT NULL,browser_session_minutes INTEGER NOT NULL,browser_actions INTEGER NOT NULL,browser_bytes INTEGER NOT NULL,settled_at DATETIME,browser_session_id TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT NOT NULL,gateway_root_operation_id TEXT NOT NULL,business_account_id TEXT NOT NULL,provider TEXT NOT NULL,model TEXT NOT NULL,credits INTEGER NOT NULL,cost_micros INTEGER NOT NULL,prompt_tokens INTEGER NOT NULL,completion_tokens INTEGER NOT NULL,state TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	settledAt := time.Date(2026, 11, 21, 10, 11, 12, 0, time.UTC)
	_, err = db.Exec(`INSERT INTO business_account_credit_reservations (id,business_account_id,status,provider,model,cost_micros,prompt_tokens,completion_tokens,charged_amount,browser_session_minutes,browser_actions,browser_bytes,settled_at,browser_session_id) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, "receipt-1", "acct-owner", string(reservationstate.ReservationFinalized), "ai-gateway-session", "multiple", 900, 12, 5, 7, 3, 8, 4096, settledAt, "opaque-session")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO business_account_browser_gateway_receipts (reservation_id,gateway_root_operation_id,business_account_id,provider,model,credits,cost_micros,prompt_tokens,completion_tokens,state) VALUES ('receipt-1','root-op-1','acct-owner','hosted-provider','hosted-model',7,900,12,5,'settled')`)
	if err != nil {
		t.Fatal(err)
	}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Dialect: "sqlite"})
	receipt, found, err := service.GetBrowserSessionReceipt(context.Background(), "acct-owner", "receipt-1")
	if err != nil || !found {
		t.Fatalf("receipt found=%t err=%v", found, err)
	}
	if receipt.Status != "settled" || receipt.Provider != "ai-gateway-session" || receipt.Model != "multiple" || receipt.PromptTokens != 12 || receipt.CompletionTokens != 5 || receipt.CreditsCharged != 7 || receipt.SessionMinutes != 3 || receipt.Actions != 8 || receipt.Bytes != 4096 || receipt.SettledAt == nil || !receipt.SettledAt.Equal(settledAt) {
		t.Fatalf("joint browser and model receipt = %+v", receipt)
	}
	if len(receipt.InferenceOperations) != 1 || receipt.InferenceOperations[0].RootOperationID != "root-op-1" || receipt.InferenceOperations[0].Provider != "hosted-provider" || receipt.InferenceOperations[0].Model != "hosted-model" || receipt.InferenceOperations[0].PromptTokens != 12 || receipt.InferenceOperations[0].CompletionTokens != 5 {
		t.Fatalf("per-operation provider/model receipt = %+v", receipt.InferenceOperations)
	}
	foreign, found, err := service.GetBrowserSessionReceipt(context.Background(), "acct-foreign", "receipt-1")
	if err != nil || found || foreign.ReservationID != "" || len(foreign.InferenceOperations) != 0 {
		t.Fatalf("foreign account lookup leaked receipt=%+v found=%t err=%v", foreign, found, err)
	}
	if _, err := db.Exec(`UPDATE business_account_browser_gateway_receipts SET cost_micros=901 WHERE reservation_id='receipt-1'`); err != nil {
		t.Fatal(err)
	}
	if _, found, err := service.GetBrowserSessionReceipt(context.Background(), "acct-owner", "receipt-1"); err == nil || found {
		t.Fatalf("corrupt settled child timeline must not be returned: found=%t err=%v", found, err)
	}
}

func TestResolveBrowserSessionSettlementBindingUsesExactRunSessionAndGrant(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	_, err = db.Exec(`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,status TEXT NOT NULL,browser_session_id TEXT NOT NULL,browser_run_id TEXT NOT NULL,browser_grant_fingerprint TEXT NOT NULL,browser_evidence_fingerprint TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO business_account_credit_reservations VALUES ('reserve-a','acct-a','buyer@example.test','pending','session-a','run-a','grant-a','')`)
	if err != nil {
		t.Fatal(err)
	}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Dialect: "sqlite"})
	reservation, account, actor, status, evidence, found, err := service.ResolveBrowserSessionSettlementBinding(context.Background(), "run-a", "session-a", "grant-a")
	if err != nil || !found || reservation != "reserve-a" || account != "acct-a" || actor != "buyer@example.test" || status != "pending" || evidence != "" {
		t.Fatalf("resolved binding = (%q,%q,%q,%q,%q,%t), err=%v", reservation, account, actor, status, evidence, found, err)
	}
	if _, err := db.Exec(`UPDATE business_account_credit_reservations SET status=? WHERE id='reserve-a'`, string(reservationstate.ReservationFinalized)); err != nil {
		t.Fatal(err)
	}
	_, _, _, status, _, found, err = service.ResolveBrowserSessionSettlementBinding(context.Background(), "run-a", "session-a", "grant-a")
	if err != nil || !found || status != "settled" {
		t.Fatalf("finalized LPBS reservation status = %q found=%t err=%v, want stable settled status", status, found, err)
	}
	for _, mismatch := range [][3]string{{"run-other", "session-a", "grant-a"}, {"run-a", "session-other", "grant-a"}, {"run-a", "session-a", "grant-other"}} {
		if _, _, _, _, _, found, err := service.ResolveBrowserSessionSettlementBinding(context.Background(), mismatch[0], mismatch[1], mismatch[2]); err != nil || found {
			t.Fatalf("mismatched binding %v found=%t err=%v", mismatch, found, err)
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
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, reserved_amount INTEGER NOT NULL, livemode BOOLEAN NOT NULL DEFAULT FALSE, actual_amount INTEGER, charged_amount INTEGER NOT NULL DEFAULT 0, overrun_amount INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, expires_at DATETIME NOT NULL, created_at DATETIME, settled_at DATETIME, app_bundle_key TEXT NOT NULL DEFAULT 'unattributed', model TEXT NOT NULL DEFAULT 'unknown', cost_micros INTEGER NOT NULL DEFAULT 0, prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0, provider TEXT NOT NULL DEFAULT 'openrouter', browser_session_minutes INTEGER NOT NULL DEFAULT 0, browser_actions INTEGER NOT NULL DEFAULT 0, browser_bytes INTEGER NOT NULL DEFAULT 0, browser_run_id TEXT NOT NULL DEFAULT '', browser_session_id TEXT NOT NULL DEFAULT '', browser_grant_fingerprint TEXT NOT NULL DEFAULT '', browser_evidence_fingerprint TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT, operation_id TEXT NOT NULL UNIQUE, business_account_id TEXT NOT NULL, actor_identity TEXT NOT NULL, app_bundle_key TEXT, model TEXT, credits INTEGER NOT NULL, cost_micros INTEGER, prompt_tokens INTEGER, completion_tokens INTEGER, provider TEXT, browser_session_minutes INTEGER NOT NULL DEFAULT 0, browser_actions INTEGER NOT NULL DEFAULT 0, browser_bytes INTEGER NOT NULL DEFAULT 0, browser_run_id TEXT NOT NULL DEFAULT '', browser_session_id TEXT NOT NULL DEFAULT '', browser_grant_fingerprint TEXT NOT NULL DEFAULT '', browser_evidence_fingerprint TEXT NOT NULL DEFAULT '', created_at DATETIME)`,
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
