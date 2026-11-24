package commerce

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// Account IDs are resolved by LPBS membership/identity owners at the API edge.
// This test verifies that the existing billing owner keeps every browser
// reservation operation scoped to that account once it reaches the ledger.
func TestBrowserReservationAndReceiptsCannotCrossBusinessAccounts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	for _, statement := range []string{
		`CREATE TABLE business_account_credit_reservations (id TEXT PRIMARY KEY,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,billing_period TEXT NOT NULL,limit_key TEXT NOT NULL,reserved_amount INTEGER NOT NULL,status TEXT NOT NULL,actual_amount INTEGER,charged_amount INTEGER NOT NULL DEFAULT 0,overrun_amount INTEGER NOT NULL DEFAULT 0,cost_micros INTEGER NOT NULL DEFAULT 0,prompt_tokens INTEGER NOT NULL DEFAULT 0,completion_tokens INTEGER NOT NULL DEFAULT 0,app_bundle_key TEXT NOT NULL DEFAULT '',model TEXT NOT NULL DEFAULT '',provider TEXT NOT NULL DEFAULT '',livemode BOOLEAN NOT NULL DEFAULT FALSE,browser_session_minutes INTEGER NOT NULL DEFAULT 0,browser_actions INTEGER NOT NULL DEFAULT 0,browser_bytes INTEGER NOT NULL DEFAULT 0,browser_run_id TEXT NOT NULL DEFAULT '',browser_session_id TEXT NOT NULL DEFAULT '',browser_grant_fingerprint TEXT NOT NULL DEFAULT '',browser_evidence_fingerprint TEXT NOT NULL DEFAULT '',browser_tier TEXT NOT NULL DEFAULT '')`,
		`CREATE TABLE business_account_browser_gateway_receipts (reservation_id TEXT NOT NULL,gateway_root_operation_id TEXT NOT NULL UNIQUE,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,request_fingerprint TEXT NOT NULL,reserved_amount INTEGER NOT NULL DEFAULT 0,state TEXT NOT NULL,provider TEXT NOT NULL DEFAULT '',model TEXT NOT NULL DEFAULT '',credits INTEGER NOT NULL DEFAULT 0,cost_micros INTEGER NOT NULL DEFAULT 0,prompt_tokens INTEGER NOT NULL DEFAULT 0,completion_tokens INTEGER NOT NULL DEFAULT 0,receipt_fingerprint TEXT NOT NULL DEFAULT '',created_at DATETIME,settled_at DATETIME,PRIMARY KEY(reservation_id,gateway_root_operation_id))`,
		`CREATE TABLE business_account_usage_events (operation_id TEXT NOT NULL UNIQUE,business_account_id TEXT NOT NULL,actor_identity TEXT NOT NULL,app_bundle_key TEXT NOT NULL,model TEXT NOT NULL,credits INTEGER NOT NULL,cost_micros INTEGER NOT NULL,prompt_tokens INTEGER NOT NULL,completion_tokens INTEGER NOT NULL,provider TEXT NOT NULL,browser_session_minutes INTEGER NOT NULL DEFAULT 0,browser_actions INTEGER NOT NULL DEFAULT 0,browser_bytes INTEGER NOT NULL DEFAULT 0,browser_run_id TEXT NOT NULL DEFAULT '',browser_session_id TEXT NOT NULL DEFAULT '',browser_grant_fingerprint TEXT NOT NULL DEFAULT '',browser_evidence_fingerprint TEXT NOT NULL DEFAULT '',feature_id TEXT NOT NULL DEFAULT '',created_at DATETIME)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create fixture schema: %v", err)
		}
	}
	const reservationID = "00000000-0000-4000-8000-000000000117"
	const rootID = "gateway-root-tenant-a-000000000001"
	const runID = "00000000-0000-4000-8000-000000000118"
	const sessionID = "bas-session-tenant-a"
	grantFingerprint, requestFingerprint, receiptFingerprint := strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64)
	if _, err := db.Exec(`INSERT INTO business_account_credit_reservations(id,business_account_id,actor_identity,billing_period,limit_key,reserved_amount,status,app_bundle_key,model,provider,browser_run_id,browser_session_id,browser_grant_fingerprint,browser_tier) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, reservationID, "tenant-account-a", "member@example.test", "2026-11", "ai_credits", 100, "pending", "web-console", "multiple", "ai-gateway-session", runID, sessionID, grantFingerprint, "pro"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO business_account_browser_gateway_receipts(reservation_id,gateway_root_operation_id,business_account_id,actor_identity,request_fingerprint,state,provider,model,credits,cost_micros,prompt_tokens,completion_tokens,receipt_fingerprint) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, reservationID, rootID, "tenant-account-a", "member@example.test", requestFingerprint, "settled", "provider", "model", 7, 900, 12, 5, receiptFingerprint); err != nil {
		t.Fatal(err)
	}
	service := NewBusinessAccountUsageService(BusinessAccountUsageOptions{DB: db, Dialect: "sqlite", Now: func() time.Time { return time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC) }})
	ctx := context.Background()

	if err := service.BindBrowserSessionIdentity(ctx, "tenant-account-b", "member@example.test", reservationID, "pro", runID, sessionID, grantFingerprint); err == nil {
		t.Fatal("tenant B bound tenant A's reservation")
	}
	if _, err := service.BeginBrowserGatewayReceipt(ctx, "tenant-account-b", "member@example.test", "pro", "web-console", reservationID, runID, sessionID, grantFingerprint, "gateway-root-tenant-b-00000000002", requestFingerprint, 5); err == nil {
		t.Fatal("tenant B claimed a child operation on tenant A's reservation")
	}
	conflictingReceipt := BrowserGatewayReceipt{RootOperationID: rootID, RequestFingerprint: requestFingerprint, State: "settled", Provider: "attacker", Model: "attacker", Credits: 99, ReceiptFingerprint: strings.Repeat("d", 64)}
	if err := service.RecordBrowserGatewayReceipt(ctx, "tenant-account-b", "member@example.test", reservationID, conflictingReceipt); err == nil {
		t.Fatal("tenant B replaced tenant A's immutable Gateway receipt")
	}
	if err := service.MarkBrowserGatewayOperationUncertain(ctx, "tenant-account-b", "member@example.test", reservationID, rootID); err == nil {
		t.Fatal("tenant B changed tenant A's Gateway child state")
	}
	if err := service.FinalizeBrowserSessionAggregate(ctx, "tenant-account-b", reservationID, runID, sessionID, grantFingerprint, strings.Repeat("e", 64), "web-console", BrowserGatewayAggregate{Operations: 1, Credits: 7, CostMicros: 900, PromptTokens: 12, CompletionTokens: 5}, 2, 3, 1024); err == nil {
		t.Fatal("tenant B settled tenant A's browser session")
	}
	for _, binding := range []struct{ run, session, grant string }{
		{run: "stale-run", session: sessionID, grant: grantFingerprint},
		{run: runID, session: "replayed-lease-session", grant: grantFingerprint},
		{run: runID, session: sessionID, grant: strings.Repeat("f", 64)},
	} {
		err := service.FinalizeBrowserSessionAggregate(ctx, "tenant-account-a", reservationID, binding.run, binding.session, binding.grant, strings.Repeat("e", 64), "web-console", BrowserGatewayAggregate{Operations: 1, Credits: 7, CostMicros: 900, PromptTokens: 12, CompletionTokens: 5}, 2, 3, 1024)
		if err == nil {
			t.Fatalf("settlement accepted stale run or replayed session/grant binding: %+v", binding)
		}
	}

	foreign, err := service.BrowserGatewayReceiptAggregate(ctx, "tenant-account-b", "member@example.test", reservationID)
	if err != nil {
		t.Fatalf("tenant B aggregate lookup: %v", err)
	}
	if foreign != (BrowserGatewayAggregate{}) {
		t.Fatalf("tenant B observed tenant A's receipt totals: %+v", foreign)
	}
	var account, state, savedReceipt string
	if err := db.QueryRow(`SELECT business_account_id,status,browser_evidence_fingerprint FROM business_account_credit_reservations WHERE id=?`, reservationID).Scan(&account, &state, &savedReceipt); err != nil {
		t.Fatal(err)
	}
	if account != "tenant-account-a" || state != "pending" || savedReceipt != "" {
		t.Fatalf("foreign attempts changed account reservation: account=%s state=%s evidence=%q", account, state, savedReceipt)
	}
	if err := db.QueryRow(`SELECT state,receipt_fingerprint FROM business_account_browser_gateway_receipts WHERE reservation_id=? AND gateway_root_operation_id=?`, reservationID, rootID).Scan(&state, &savedReceipt); err != nil {
		t.Fatal(err)
	}
	if state != "settled" || savedReceipt != receiptFingerprint {
		t.Fatalf("foreign attempts changed immutable child: state=%s receipt=%s", state, savedReceipt)
	}

	ownerAggregate, err := service.BrowserGatewayReceiptAggregate(ctx, "tenant-account-a", "member@example.test", reservationID)
	if err != nil || ownerAggregate != (BrowserGatewayAggregate{Operations: 1, Credits: 7, CostMicros: 900, PromptTokens: 12, CompletionTokens: 5}) {
		t.Fatalf("owner aggregate=%+v err=%v", ownerAggregate, err)
	}
	if err := service.RecordBrowserGatewayReceipt(ctx, "tenant-account-a", "member@example.test", reservationID, BrowserGatewayReceipt{RootOperationID: rootID, RequestFingerprint: requestFingerprint, State: "settled", Provider: "provider", Model: "model", Credits: 7, CostMicros: 900, PromptTokens: 12, CompletionTokens: 5, ReceiptFingerprint: receiptFingerprint}); err != nil {
		t.Fatalf("identical owner retry changed behavior: %v", err)
	}
}
