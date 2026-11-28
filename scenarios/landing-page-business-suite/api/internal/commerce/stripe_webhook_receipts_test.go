package commerce

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	_ "modernc.org/sqlite"
)

func TestStripeWebhookReceiptPersistsTestOutcomeAndRejectsConflictingReplay(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeWebhookReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	ctx := context.Background()
	link := &StripeTestOutcomeLink{RunID: "py1-test-run-1", Outcome: "real_webhook_delivery"}
	body := []byte(`{"id":"evt_123","type":"customer.updated"}`)
	claim, process, err := service.beginWebhookReceipt(ctx, "evt_123", "customer.updated", "cus_123", false, body, link)
	if err != nil || !process {
		t.Fatalf("begin receipt = (%t,%v), want process", process, err)
	}
	if err := service.finishWebhookReceipt(ctx, "evt_123", claim, nil); err != nil {
		t.Fatal(err)
	}
	receipt, err := service.loadTestOutcomeWebhookReceipt(ctx, link.RunID, link.Outcome)
	if err != nil || receipt == nil {
		t.Fatalf("load outcome receipt = (%+v,%v), want retained receipt", receipt, err)
	}
	if receipt.EventID != "evt_123" || receipt.ObjectID != "cus_123" || receipt.Status != "processed" || receipt.Livemode {
		t.Fatalf("receipt = %+v, want test-mode processed event/object IDs", receipt)
	}
	_, process, err = service.beginWebhookReceipt(ctx, "evt_123", "customer.updated", "cus_123", false, body, link)
	if err != nil || process {
		t.Fatalf("duplicate processed receipt = (%t,%v), want acknowledged duplicate", process, err)
	}
	if _, _, err := service.beginWebhookReceipt(ctx, "evt_123", "customer.updated", "cus_123", false, []byte(`{"id":"evt_123","type":"customer.updated","data":{}}`), link); err == nil {
		t.Fatal("same event ID with changed payload was accepted")
	}
}

func TestStripeWebhookReceiptRetriesFailedHandlerWithoutReplacingIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeWebhookReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	ctx := context.Background()
	link := &StripeTestOutcomeLink{RunID: "py1-test-run-2", Outcome: "checkout_success"}
	body := []byte(`{"id":"evt_retry","type":"checkout.session.completed"}`)
	claim, process, err := service.beginWebhookReceipt(ctx, "evt_retry", "checkout.session.completed", "cs_123", false, body, link)
	if err != nil || !process {
		t.Fatalf("first begin receipt = (%t,%v), want process", process, err)
	}
	if err := service.finishWebhookReceipt(ctx, "evt_retry", claim, errors.New("sanitized handler failure")); err != nil {
		t.Fatal(err)
	}
	claim, process, err = service.beginWebhookReceipt(ctx, "evt_retry", "checkout.session.completed", "cs_123", false, body, link)
	if err != nil || !process {
		t.Fatalf("retry begin receipt = (%t,%v), want retry", process, err)
	}
	if err := service.finishWebhookReceipt(ctx, "evt_retry", claim, nil); err != nil {
		t.Fatal(err)
	}
	var status string
	var attempts int
	var failure sql.NullString
	if err := db.QueryRow(`SELECT status,attempts,failure_code FROM stripe_webhook_events WHERE stripe_event_id='evt_retry'`).Scan(&status, &attempts, &failure); err != nil {
		t.Fatal(err)
	}
	if status != "processed" || attempts != 2 || failure.Valid {
		t.Fatalf("status=%q attempts=%d failure=%+v, want processed/2/no failure", status, attempts, failure)
	}
}

func TestStripeTestOutcomeMetadataFailsClosed(t *testing.T) {
	valid := map[string]interface{}{"lpbs_py1_test_run_id": "py1-run", "lpbs_py1_test_outcome": "donation"}
	link, err := stripeTestOutcomeLink(valid, false)
	if err != nil || link == nil || link.RunID != "py1-run" || link.Outcome != "donation" {
		t.Fatalf("valid link = (%+v,%v)", link, err)
	}
	if liveLink, err := stripeTestOutcomeLink(valid, true); err != nil || liveLink != nil {
		t.Fatalf("live event created PY1 test receipt link: (%+v,%v)", liveLink, err)
	}
	if _, err := stripeTestOutcomeLink(map[string]interface{}{"lpbs_py1_test_run_id": "py1-run", "lpbs_py1_test_outcome": "unknown"}, false); err == nil {
		t.Fatal("unknown outcome name was accepted")
	}
}

func TestHandleWebhookPersistsMetadataLinkedOutcomeAfterSuccessfulHandler(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, statement := range []string{
		stripeWebhookReceiptsSchema,
		`CREATE TABLE subscriptions (customer_email TEXT, customer_id TEXT, updated_at TIMESTAMP)`,
		`CREATE TABLE users (email TEXT, updated_at TIMESTAMP, stripe_customer_id TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	const signingSecret = "whsec_test_outcome"
	service := NewStripeWebhookService(StripeWebhookOptions{DB: db, Config: func() StripeWebhookConfig {
		return StripeWebhookConfig{WebhookSecret: signingSecret, HasWebhook: true}
	}})
	body := []byte(`{"id":"evt_durable","type":"customer.updated","livemode":false,"data":{"object":{"id":"cus_durable","email":"py1@example.com","metadata":{"lpbs_py1_test_run_id":"py1-run-3","lpbs_py1_test_outcome":"real_webhook_delivery"}}}}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	signature := fmt.Sprintf("t=%s,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
	if err := service.HandleWebhook(body, signature); err != nil {
		t.Fatalf("HandleWebhook(): %v", err)
	}
	receipt, err := service.LoadTestOutcomeWebhookReceipt(context.Background(), "py1-run-3", "real_webhook_delivery")
	if err != nil || receipt == nil || receipt.Status != "processed" || receipt.EventID != "evt_durable" || receipt.ObjectID != "cus_durable" {
		t.Fatalf("durable webhook receipt = (%+v,%v)", receipt, err)
	}
	if err := service.HandleWebhook(body, signature); err != nil {
		t.Fatalf("duplicate HandleWebhook(): %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM stripe_webhook_events WHERE stripe_event_id='evt_durable'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("event receipt rows = %d, err=%v; want one idempotent row", count, err)
	}
}

func TestStripeTestEconomicReceiptRetainsPaidProviderIdsAndRejectsConflicts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, schema := range []string{stripeWebhookReceiptsSchema, stripeTestEconomicReceiptsSchema} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	service := &StripeWebhookService{db: db}
	link := &StripeTestOutcomeLink{RunID: "py1-economic-run-1", Outcome: "checkout_success"}
	object := map[string]interface{}{
		"id": "cs_test_123", "payment_status": "paid", "payment_intent": "pi_test_123",
		"subscription": "sub_test_123", "amount_total": float64(2900), "currency": "usd",
	}
	if err := service.recordStripeTestEconomicReceipt(context.Background(), "evt_test_123", object, link); err != nil {
		t.Fatalf("recordStripeTestEconomicReceipt(): %v", err)
	}
	receipt, err := service.LoadTestEconomicReceipt(context.Background(), link.RunID, link.Outcome)
	if err != nil || receipt == nil || receipt.EventID != "evt_test_123" || receipt.CheckoutSessionID != "cs_test_123" || receipt.PaymentIntentID != "pi_test_123" || receipt.SubscriptionID != "sub_test_123" || receipt.AmountMinor != 2900 || receipt.Currency != "usd" || receipt.MoneyLedgerStatus != "pending" || receipt.MoneyLedgerExternalID != "lpbs-py1-checkout:cs_test_123" {
		t.Fatalf("LoadTestEconomicReceipt() = %+v, %v", receipt, err)
	}
	if err := service.recordStripeTestEconomicReceipt(context.Background(), "evt_test_123", object, link); err != nil {
		t.Fatalf("idempotent economic receipt replay: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM py1_test_economic_receipts`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("economic receipt rows = %d, err=%v; want one", count, err)
	}
	object["amount_total"] = float64(9900)
	if err := service.recordStripeTestEconomicReceipt(context.Background(), "evt_test_456", object, link); err == nil {
		t.Fatal("conflicting second payment for one Checkout Session was accepted")
	}
}

type py1LedgerWriterStub struct {
	event *sharedpb.MoneyEvent
	count int
	err   error
}

func (w *py1LedgerWriterStub) IngestSandboxCapture(_ context.Context, event *sharedpb.MoneyEvent) (ServiceOrderLedgerReceipt, error) {
	w.count++
	w.event = event
	if w.err != nil {
		return ServiceOrderLedgerReceipt{}, w.err
	}
	return ServiceOrderLedgerReceipt{PostingID: "posting_py1", ReceiptID: "receipt_py1", ExternalID: event.GetExternalId()}, nil
}

func TestDispatchPY1TestEconomicReceiptRequiresAcceptedMoneyLedgerIdentity(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeTestEconomicReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	link := &StripeTestOutcomeLink{RunID: "py1-ledger-run-1", Outcome: "checkout_success"}
	object := map[string]interface{}{"id": "cs_test_ledger", "payment_status": "paid", "payment_intent": "pi_test_ledger", "amount_total": float64(2900), "currency": "usd"}
	if err := service.recordStripeTestEconomicReceipt(context.Background(), "evt_test_ledger", object, link); err != nil {
		t.Fatal(err)
	}
	writer := &py1LedgerWriterStub{}
	target := PY1TestLedgerTarget{AdapterID: "lpbs-test", BookID: "test-book", RevenueAccountID: "test-revenue"}
	receipt, err := DispatchPY1TestEconomicReceipt(context.Background(), db, writer, target, link.RunID, link.Outcome)
	if err != nil || receipt == nil || receipt.MoneyLedgerStatus != "posted" || receipt.MoneyLedgerPostingID != "posting_py1" || receipt.MoneyLedgerReceiptID != "receipt_py1" {
		t.Fatalf("DispatchPY1TestEconomicReceipt() = %+v, %v", receipt, err)
	}
	if writer.count != 1 || writer.event.GetExternalId() != "lpbs-py1-checkout:cs_test_ledger" || writer.event.GetAmountMinor() != 2900 || writer.event.GetCurrency() != "USD" || writer.event.GetCategory() != "test.py1.stripe_checkout" || writer.event.GetBookId() != "test-book" || writer.event.GetAccountId() != "test-revenue" {
		t.Fatalf("Money Ledger event = %+v; calls=%d", writer.event, writer.count)
	}
	if _, err := DispatchPY1TestEconomicReceipt(context.Background(), db, writer, target, link.RunID, link.Outcome); err != nil || writer.count != 1 {
		t.Fatalf("posted receipt was not idempotent: calls=%d err=%v", writer.count, err)
	}
}

func TestStripeWebhookReceiptAttemptClaimsFenceFinish(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeWebhookReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	body := []byte(`{"id":"evt_stale","type":"future.event","data":{"object":{"id":"obj_stale"}}}`)
	firstClaim, claimed, err := service.beginWebhookReceipt(context.Background(), "evt_stale", "future.event", "obj_stale", false, body, nil)
	if err != nil || !claimed {
		t.Fatalf("initial receipt claim = %t, %v", claimed, err)
	}
	if _, err := db.Exec(`UPDATE stripe_webhook_events SET status='retryable_failed',last_seen_at='2000-01-01 00:00:00' WHERE stripe_event_id='evt_stale'`); err != nil {
		t.Fatal(err)
	}
	secondClaim, claimed, err := service.beginWebhookReceipt(context.Background(), "evt_stale", "future.event", "obj_stale", false, body, nil)
	if err != nil || !claimed {
		t.Fatalf("expired processing receipt claim = %t, %v", claimed, err)
	}
	var attempts int
	if err := db.QueryRow(`SELECT attempts FROM stripe_webhook_events WHERE stripe_event_id='evt_stale'`).Scan(&attempts); err != nil || attempts != 2 {
		t.Fatalf("recovered receipt attempts = %d, err=%v", attempts, err)
	}
	if err := service.finishWebhookReceipt(context.Background(), "evt_stale", firstClaim, nil); !errors.Is(err, ErrStripeWebhookReceiptClaimLost) {
		t.Fatalf("stale receipt owner finish = %v; want claim-lost", err)
	}
	if err := service.finishWebhookReceipt(context.Background(), "evt_stale", secondClaim, nil); err != nil {
		t.Fatalf("recovered receipt owner finish = %v", err)
	}
}

func TestOperationsSchemaIncludesStripeWebhookInbox(t *testing.T) {
	schema := strings.ToLower(OperationsSchema())
	if !strings.Contains(schema, "create table if not exists stripe_webhook_events") || !strings.Contains(schema, "payload_sha256") || !strings.Contains(schema, "create table if not exists py1_test_economic_receipts") || !strings.Contains(schema, "money_ledger_external_id") {
		t.Fatal("operations schema omitted the Stripe inbox or PY1 economic/ledger outbox receipt")
	}
}
