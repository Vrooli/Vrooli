package commerce

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

type mutableLifecycleLimits struct{ multiplier int64 }

func (l *mutableLifecycleLimits) GetLimit(_ context.Context, tier, key string, app *string) (*TierLimit, error) {
	return &TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: 100, CostMultiplier: l.multiplier, AppBundleKey: app}, nil
}

type racingLifecycleLimits struct {
	mu    sync.Mutex
	calls int
	ready chan struct{}
}

func (l *racingLifecycleLimits) GetLimit(_ context.Context, tier, key string, app *string) (*TierLimit, error) {
	l.mu.Lock()
	l.calls++
	call := l.calls
	if call == 2 {
		close(l.ready)
	}
	l.mu.Unlock()
	<-l.ready
	multiplier := int64(call * 10000)
	return &TierLimit{TierID: tier, LimitKey: key, LimitType: "cost_based", LimitValue: 100, CostMultiplier: multiplier, AppBundleKey: app}, nil
}
func (l *racingLifecycleLimits) GetTierLimits(_ context.Context, tier string) ([]TierLimit, error) {
	app := "audio-tools"
	return []TierLimit{{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 100, CostMultiplier: 10000}, {TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 100, CostMultiplier: 10000, AppBundleKey: &app}}, nil
}
func (l *mutableLifecycleLimits) GetTierLimits(_ context.Context, tier string) ([]TierLimit, error) {
	app := "audio-tools"
	return []TierLimit{{TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 100, CostMultiplier: l.multiplier}, {TierID: tier, LimitKey: "ai_credits", LimitType: "cost_based", LimitValue: 100, CostMultiplier: l.multiplier, AppBundleKey: &app}}, nil
}

func TestCostMicrosToCreditsRoundingAndOverflow(t *testing.T) {
	for _, tc := range []struct{ cost, multiplier, want int64 }{{1, 10000, 1}, {10000, 10000, 10000}, {10001, 10000, 10001}, {1, 1, 1}} {
		got, err := costMicrosToCredits(tc.cost, tc.multiplier)
		if err != nil || got != tc.want {
			t.Fatalf("convert(%d,%d)=%d err=%v want %d", tc.cost, tc.multiplier, got, err, tc.want)
		}
	}
	if _, err := costMicrosToCredits(0, 1); err == nil {
		t.Fatal("zero cost accepted")
	}
	if _, err := costMicrosToCredits(math.MaxInt64, 2); err == nil {
		t.Fatal("overflowing conversion accepted")
	}
}

func TestAccountCreditLifecycleFencesDispatchSettlementReleaseAndRecovery(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_credit_wallets (business_account_id TEXT PRIMARY KEY,customer_email TEXT,balance_credits INTEGER DEFAULT 0,bonus_credits INTEGER DEFAULT 0,updated_at DATETIME)`,
		`CREATE TABLE credit_funding_lots (id TEXT PRIMARY KEY,business_account_id TEXT,source_kind TEXT,provider TEXT,provider_ref TEXT,livemode BOOLEAN,settlement_state TEXT,spendable_after DATETIME,reversal_window_ends DATETIME,granted INTEGER,remaining INTEGER,expires_at DATETIME,status TEXT,created_at DATETIME,UNIQUE(business_account_id,provider,provider_ref))`,
		`CREATE TABLE credit_lot_allocations (id TEXT PRIMARY KEY,lot_id TEXT,operation_id TEXT,credits INTEGER,cost_micros INTEGER,created_at DATETIME,UNIQUE(lot_id,operation_id))`,
		`CREATE TABLE business_account_usage_records (business_account_id TEXT,actor_identity TEXT,billing_period TEXT,limit_key TEXT,usage_amount INTEGER DEFAULT 0,app_bundle_key TEXT,last_operation_at DATETIME,created_at DATETIME,updated_at DATETIME,UNIQUE(business_account_id,billing_period,limit_key,app_bundle_key))`,
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT,actor_identity TEXT,billing_period TEXT,limit_key TEXT,reserved_amount INTEGER,livemode BOOLEAN,actual_amount INTEGER,charged_amount INTEGER DEFAULT 0,overrun_amount INTEGER DEFAULT 0,status TEXT,expires_at DATETIME,created_at DATETIME,settled_at DATETIME,app_bundle_key TEXT NOT NULL DEFAULT 'unattributed',model TEXT NOT NULL DEFAULT 'unknown',cost_micros INTEGER DEFAULT 0,prompt_tokens INTEGER DEFAULT 0,completion_tokens INTEGER DEFAULT 0,provider TEXT NOT NULL DEFAULT 'openrouter',browser_session_minutes INTEGER DEFAULT 0,browser_actions INTEGER DEFAULT 0,browser_bytes INTEGER DEFAULT 0,browser_run_id TEXT DEFAULT '',browser_session_id TEXT DEFAULT '',browser_grant_fingerprint TEXT DEFAULT '',browser_tier TEXT DEFAULT '',browser_evidence_fingerprint TEXT DEFAULT '')`,
		`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT,gateway_root_operation_id TEXT,business_account_id TEXT,actor_identity TEXT,request_fingerprint TEXT,reserved_amount INTEGER,state TEXT,provider TEXT,model TEXT,credits INTEGER,cost_micros INTEGER,prompt_tokens INTEGER,completion_tokens INTEGER,receipt_fingerprint TEXT,created_at DATETIME,settled_at DATETIME,PRIMARY KEY(reservation_id,gateway_root_operation_id))`,
		`CREATE TABLE business_account_usage_events (id INTEGER PRIMARY KEY AUTOINCREMENT,operation_id TEXT UNIQUE,business_account_id TEXT,actor_identity TEXT,app_bundle_key TEXT,model TEXT,credits INTEGER,cost_micros INTEGER,prompt_tokens INTEGER,completion_tokens INTEGER,provider TEXT,browser_session_minutes INTEGER DEFAULT 0,browser_actions INTEGER DEFAULT 0,browser_bytes INTEGER DEFAULT 0,browser_run_id TEXT DEFAULT '',browser_session_id TEXT DEFAULT '',browser_grant_fingerprint TEXT DEFAULT '',browser_evidence_fingerprint TEXT DEFAULT '',created_at DATETIME)`,
		`CREATE TABLE business_account_credit_lifecycle (reservation_id TEXT PRIMARY KEY,business_account_id TEXT,actor_identity TEXT,app_bundle_key TEXT,phase TEXT,lease_expires_at DATETIME,settlement_fingerprint TEXT DEFAULT '',provider_receipt_id TEXT DEFAULT '',contract_version INTEGER DEFAULT 0,usd_budget_micros INTEGER DEFAULT 0,cost_multiplier INTEGER DEFAULT 0,updated_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create lifecycle schema: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO business_account_credit_wallets(business_account_id,customer_email,balance_credits) VALUES('acct','owner@example.test',100)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES('lot','acct','test','fixture','lot',FALSE,'settled',100,100,'open',CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	usage := NewUsageServiceWithOptions(UsageServiceOptions{DB: db, LimitsService: fixedAccountLimits{value: 100}, Dialect: "sqlite", LiveMode: boolRef(false)})
	lifecycle := NewAccountCreditLifecycleService(usage)
	ctx := context.Background()
	const first = "00000000-0000-4000-8000-000000000101"
	state, created, err := lifecycle.Reserve(ctx, "acct", "owner@example.test", "pro", "audio-tools", first, 20)
	if err != nil || !created || state.Phase != "reserved" {
		t.Fatalf("reserve=%+v created=%t err=%v", state, created, err)
	}
	state, created, err = lifecycle.Reserve(ctx, "acct", "owner@example.test", "pro", "audio-tools", first, 20)
	if err != nil || created || state.Phase != "reserved" {
		t.Fatalf("identical reserve replay=%+v created=%t err=%v", state, created, err)
	}
	if _, err := lifecycle.Recover(ctx, "attacker@example.test", "audio-tools", first); err == nil {
		t.Fatal("different actor recovered reservation")
	}
	if started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "web-console", first); err != nil || started {
		t.Fatalf("different app dispatch=%t err=%v", started, err)
	}
	started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "audio-tools", first)
	if err != nil || !started {
		t.Fatalf("begin dispatch=%t err=%v", started, err)
	}
	if started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "audio-tools", first); err != nil || started {
		t.Fatalf("dispatch replay=%t err=%v; must not dispatch twice", started, err)
	}
	if err := lifecycle.Release(ctx, "owner@example.test", "audio-tools", first); !errors.Is(err, ErrCreditLifecycleState) {
		t.Fatalf("post-dispatch release error=%v", err)
	}
	var expiry time.Time
	if err := db.QueryRow(`SELECT expires_at FROM business_account_credit_reservations WHERE id=?`, first).Scan(&expiry); err != nil {
		t.Fatal(err)
	}
	if expiry.Year() != 9999 {
		t.Fatalf("dispatched wallet lease expires at %s; expected conservative owner hold", expiry)
	}
	settlement := AccountCreditSettlement{ActualAmount: 7, CostMicros: 1234, ActiveSeconds: int64Pointer(3), PausedSeconds: int64Pointer(2), PromptTokens: 11, CompletionTokens: 6, Model: "gpt-live-model", Provider: "openai-realtime", ProviderReceiptID: "resp_provider_123"}
	if err := lifecycle.Settle(ctx, "owner@example.test", "audio-tools", first, settlement); err != nil {
		t.Fatalf("measured settlement: %v", err)
	}
	if err := lifecycle.Settle(ctx, "owner@example.test", "audio-tools", first, settlement); err != nil {
		t.Fatalf("identical settlement replay: %v", err)
	}
	settlement.ActualAmount++
	if err := lifecycle.Settle(ctx, "owner@example.test", "audio-tools", first, settlement); err == nil {
		t.Fatal("conflicting settlement replay accepted")
	}
	recovered, err := lifecycle.Recover(ctx, "owner@example.test", "audio-tools", first)
	if err != nil || recovered.Phase != "settled" || recovered.WalletStatus != "finalized" || recovered.Actual == nil || *recovered.Actual != 7 || recovered.ProviderReceiptID != "resp_provider_123" {
		t.Fatalf("recovered=%+v err=%v", recovered, err)
	}

	const second = "00000000-0000-4000-8000-000000000102"
	if _, created, err := lifecycle.Reserve(ctx, "acct", "owner@example.test", "pro", "audio-tools", second, 10); err != nil || !created {
		t.Fatalf("second reserve created=%t err=%v", created, err)
	}
	if started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "audio-tools", second); err != nil || !started {
		t.Fatalf("second dispatch=%t err=%v", started, err)
	}
	if err := lifecycle.RetainUncertain(ctx, "owner@example.test", "audio-tools", second); err != nil {
		t.Fatalf("retain uncertainty: %v", err)
	}
	uncertain, err := lifecycle.Recover(ctx, "owner@example.test", "audio-tools", second)
	if err != nil || uncertain.Phase != "uncertain" || uncertain.WalletStatus != "pending" {
		t.Fatalf("uncertain recovery=%+v err=%v", uncertain, err)
	}
	settlement = AccountCreditSettlement{ActualAmount: 4, CostMicros: 1000, ActiveSeconds: int64Pointer(2), PausedSeconds: int64Pointer(1), PromptTokens: 3, CompletionTokens: 2, Model: "gpt-live-model", Provider: "openai-realtime", ProviderReceiptID: "resp_provider_456"}
	if err := lifecycle.Settle(ctx, "owner@example.test", "audio-tools", second, settlement); err != nil {
		t.Fatalf("settle recovered receipt: %v", err)
	}

	const third = "00000000-0000-4000-8000-000000000103"
	if _, created, err := lifecycle.Reserve(ctx, "acct", "owner@example.test", "pro", "audio-tools", third, 10); err != nil || !created {
		t.Fatalf("third reserve created=%t err=%v", created, err)
	}
	if err := lifecycle.Release(ctx, "owner@example.test", "audio-tools", third); err != nil {
		t.Fatalf("pre-dispatch release: %v", err)
	}
	if started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "audio-tools", third); err != nil || started {
		t.Fatalf("dispatch after release=%t err=%v", started, err)
	}

	const fourth = "00000000-0000-4000-8000-000000000104"
	if _, created, err := lifecycle.Reserve(ctx, "acct", "owner@example.test", "pro", "audio-tools", fourth, 5); err != nil || !created {
		t.Fatalf("fourth reserve created=%t err=%v", created, err)
	}
	if _, err := db.Exec(`UPDATE business_account_credit_lifecycle SET lease_expires_at=? WHERE reservation_id=?`, time.Now().Add(-time.Second), fourth); err != nil {
		t.Fatal(err)
	}
	expired, err := lifecycle.Recover(ctx, "owner@example.test", "audio-tools", fourth)
	if err != nil || expired.Phase != "released" || expired.WalletStatus != "released" {
		t.Fatalf("expired pre-dispatch recovery=%+v err=%v", expired, err)
	}

	limits := &mutableLifecycleLimits{multiplier: 10000}
	usage.limitsSvc = limits
	const fifth = "00000000-0000-4000-8000-000000000105"
	state, created, err = lifecycle.ReserveUSDBudgetV1(ctx, "acct", "owner@example.test", "pro", "audio-tools", fifth, 1)
	if err != nil || !created || state.Reserved != 1 || state.CostMultiplier == nil || *state.CostMultiplier != 10000 {
		t.Fatalf("USD reservation=%+v created=%t err=%v", state, created, err)
	}
	limits.multiplier = 20000
	state, created, err = lifecycle.ReserveUSDBudgetV1(ctx, "acct", "owner@example.test", "pro", "audio-tools", fifth, 1)
	if err != nil || created || state.Reserved != 1 || state.CostMultiplier == nil || *state.CostMultiplier != 10000 {
		t.Fatalf("USD replay after policy change=%+v created=%t err=%v", state, created, err)
	}
	if started, err := lifecycle.BeginDispatch(ctx, "owner@example.test", "audio-tools", fifth); err != nil || !started {
		t.Fatalf("USD dispatch=%t err=%v", started, err)
	}
	settlement = AccountCreditSettlement{CostMicros: 1, ActiveSeconds: int64Pointer(1), PausedSeconds: int64Pointer(0), Model: "small-model", Provider: "local", ProviderReceiptID: "receipt-usd-v1"}
	if err := lifecycle.SettleMeasuredCostV1(ctx, "owner@example.test", "audio-tools", fifth, settlement); err != nil {
		t.Fatalf("USD measured settlement: %v", err)
	}
	var actual int64
	if err := db.QueryRow(`SELECT actual_amount FROM business_account_credit_reservations WHERE id=?`, fifth).Scan(&actual); err != nil || actual != 1 {
		t.Fatalf("settled credits=%d err=%v want persisted-policy value 1", actual, err)
	}

	usage.limitsSvc = &racingLifecycleLimits{ready: make(chan struct{})}
	const sixth = "00000000-0000-4000-8000-000000000106"
	type reserveResult struct {
		state   AccountCreditLifecycle
		created bool
		err     error
	}
	results := make(chan reserveResult, 2)
	for range 2 {
		go func() {
			got, wasCreated, reserveErr := lifecycle.ReserveUSDBudgetV1(ctx, "acct", "owner@example.test", "pro", "audio-tools", sixth, 1)
			results <- reserveResult{state: got, created: wasCreated, err: reserveErr}
		}()
	}
	left, right := <-results, <-results
	if left.err != nil || right.err != nil || left.state.CostMultiplier == nil || right.state.CostMultiplier == nil || *left.state.CostMultiplier != *right.state.CostMultiplier || left.state.Reserved != right.state.Reserved || (left.created == right.created) {
		t.Fatalf("concurrent USD replay left=%+v right=%+v", left, right)
	}
}

func TestAccountCreditLifecycleRequiresMeasuredSettlement(t *testing.T) {
	for _, input := range []AccountCreditSettlement{
		{Model: "model", Provider: "openai"},
		{ActualAmount: -1, ActiveSeconds: int64Pointer(0), PausedSeconds: int64Pointer(0), Model: "model", Provider: "openai", ProviderReceiptID: "receipt"},
		{CostMicros: -1, ActiveSeconds: int64Pointer(0), PausedSeconds: int64Pointer(0), Model: "model", Provider: "openai", ProviderReceiptID: "receipt"},
		{ActualAmount: 1, ActiveSeconds: int64Pointer(0), PausedSeconds: int64Pointer(0), Model: "", Provider: "openai", ProviderReceiptID: "receipt"},
		{ActualAmount: 1, ActiveSeconds: int64Pointer(0), PausedSeconds: int64Pointer(0), Model: "model", Provider: "", ProviderReceiptID: "receipt"},
		{ActualAmount: 1, ActiveSeconds: int64Pointer(-1), PausedSeconds: int64Pointer(0), Model: "model", Provider: "openai", ProviderReceiptID: "receipt"},
		{ActualAmount: 1, ActiveSeconds: int64Pointer(0), Model: "model", Provider: "openai", ProviderReceiptID: "receipt"},
	} {
		if _, err := settlementFingerprint(input); err == nil {
			t.Fatalf("accepted invalid measured settlement %+v", input)
		}
	}
}

func int64Pointer(value int64) *int64 { return &value }
