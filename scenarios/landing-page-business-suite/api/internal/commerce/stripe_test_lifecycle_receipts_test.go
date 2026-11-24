package commerce

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	_ "modernc.org/sqlite"
)

func TestPY1LifecyclePaidInvoiceLinksPostgresReceiptContractAndLedgerCapture(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeTestLifecycleReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	link := &StripeTestOutcomeLink{RunID: "py1-renewal-run", Outcome: "renewal"}
	invoice := map[string]interface{}{
		"id": "in_test_renewal", "subscription": "sub_test_renewal", "amount_paid": float64(2900), "currency": "usd", "billing_reason": "subscription_cycle",
		"payment_intent": "pi_test_renewal",
	}
	if err := service.recordStripeTestLifecycleReceipt(context.Background(), "evt_test_renewal", "invoice.paid", invoice, link); err != nil {
		t.Fatal(err)
	}
	writer := &py1LedgerWriterStub{}
	target := PY1TestLedgerTarget{AdapterID: "lpbs-test", BookID: "test-book", RevenueAccountID: "test-revenue"}
	if err := DispatchPendingPY1TestLifecycleReceipts(context.Background(), db, writer, target, link.RunID); err != nil {
		t.Fatalf("dispatch pending lifecycle outbox: %v", err)
	}
	receipt, err := (&StripeWebhookService{db: db}).LoadTestLifecycleReceiptByEvent(context.Background(), "evt_test_renewal")
	if err != nil || receipt == nil || receipt.EventID != "evt_test_renewal" || receipt.SubscriptionID != "sub_test_renewal" || receipt.SubscriptionStatus != "" || receipt.PaymentStatus != "paid" || receipt.MoneyLedgerStatus != "posted" {
		t.Fatalf("renewal receipt = %+v, %v", receipt, err)
	}
	if receipt.MoneyLedgerExternalID != "lpbs-py1-invoice:in_test_renewal" || receipt.MoneyLedgerPostingID != "posting_py1" || receipt.MoneyLedgerReceiptID != "receipt_py1" {
		t.Fatalf("renewal ledger linkage = %+v", receipt)
	}
	if writer.count != 1 || writer.event.GetExternalId() != receipt.MoneyLedgerExternalID || writer.event.GetAmountMinor() != 2900 || writer.event.GetCurrency() != "USD" || writer.event.GetCategory() != "test.py1.stripe_subscription_invoice" || writer.event.GetBookId() != target.BookID || writer.event.GetAccountId() != target.RevenueAccountID {
		t.Fatalf("ledger capture = %+v calls=%d", writer.event, writer.count)
	}
	if err := service.recordStripeTestLifecycleReceipt(context.Background(), "evt_test_renewal", "invoice.paid", invoice, link); err != nil {
		t.Fatal(err)
	}
	if err := DispatchPendingPY1TestLifecycleReceipts(context.Background(), db, writer, target, link.RunID); err != nil || writer.count != 1 {
		t.Fatalf("paid invoice replay was not idempotent: calls=%d err=%v", writer.count, err)
	}
}

func TestPY1LifecyclePastDueAndCancelRetainStateWithoutInventingLedgerRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(stripeTestLifecycleReceiptsSchema); err != nil {
		t.Fatal(err)
	}
	service := &StripeWebhookService{db: db}
	for _, test := range []struct {
		eventID, eventType, outcome string
		object                      map[string]interface{}
		wantStatus, wantPayment     string
	}{
		{"evt_test_failed_renewal", "invoice.payment_failed", "past_due", map[string]interface{}{"id": "in_test_failed", "subscription": "sub_test_failed"}, "", "failed"},
		{"evt_test_past_due_status", "customer.subscription.updated", "past_due", map[string]interface{}{"id": "sub_test_failed", "status": "past_due"}, "past_due", "not_applicable"},
		{"evt_test_cancelled", "customer.subscription.deleted", "cancel", map[string]interface{}{"id": "sub_test_cancelled"}, "canceled", "not_applicable"},
	} {
		link := &StripeTestOutcomeLink{RunID: "py1-lifecycle-run", Outcome: test.outcome}
		if err := service.recordStripeTestLifecycleReceipt(context.Background(), test.eventID, test.eventType, test.object, link); err != nil {
			t.Fatalf("record %s: %v", test.eventType, err)
		}
		receipt, err := service.LoadTestLifecycleReceipt(context.Background(), link.RunID, link.Outcome)
		if err != nil || receipt == nil || receipt.SubscriptionStatus != test.wantStatus || receipt.PaymentStatus != test.wantPayment || receipt.MoneyLedgerStatus != "not_applicable" || receipt.MoneyLedgerExternalID != "" || receipt.AmountMinor != 0 {
			t.Fatalf("%s receipt = %+v, %v", test.eventType, receipt, err)
		}
	}
	failed, err := service.LoadTestLifecycleReceiptByOutcomeAndEventType(context.Background(), "py1-lifecycle-run", "past_due", "invoice.payment_failed")
	status, statusErr := service.LoadTestLifecycleReceiptByOutcomeEventTypeAndStatus(context.Background(), "py1-lifecycle-run", "past_due", "customer.subscription.updated", "past_due")
	if err != nil || statusErr != nil || failed == nil || status == nil || failed.SubscriptionID != status.SubscriptionID || failed.PaymentStatus != "failed" || status.SubscriptionStatus != "past_due" {
		t.Fatalf("past_due causal pair failed=%+v status=%+v errs=(%v,%v)", failed, status, err, statusErr)
	}
}

func TestOperationsSchemaIncludesLifecycleReceiptAndRenewalOutbox(t *testing.T) {
	schema := OperationsSchema()
	for _, required := range []string{"CREATE TABLE IF NOT EXISTS py1_test_lifecycle_receipts", "money_ledger_external_id", "money_ledger_posting_id", "money_ledger_receipt_id"} {
		if !strings.Contains(strings.ToLower(schema), strings.ToLower(required)) {
			t.Fatalf("operations schema omitted lifecycle outbox element %q", required)
		}
	}
}

func TestPY1LifecycleLinkReadsCurrentAndLegacyInvoiceSubscriptionMetadata(t *testing.T) {
	metadata := map[string]interface{}{"lpbs_py1_test_run_id": "py1-renewal-run", "lpbs_py1_test_outcome": "renewal"}
	for _, object := range []map[string]interface{}{
		{"metadata": map[string]interface{}{}, "subscription_details": map[string]interface{}{"metadata": metadata}},
		{"metadata": map[string]interface{}{}, "parent": map[string]interface{}{"subscription_details": map[string]interface{}{"metadata": metadata}}},
		{"metadata": map[string]interface{}{"invoice_key": "ignored"}, "parent": map[string]interface{}{"subscription_details": map[string]interface{}{"metadata": metadata}}},
	} {
		link, err := stripeTestOutcomeLink(stripeObjectOutcomeMetadata("invoice.paid", object), false)
		if err != nil || link == nil || link.RunID != "py1-renewal-run" || link.Outcome != "renewal" {
			t.Fatalf("invoice link = %+v, %v for object %#v", link, err, object)
		}
	}
}

func TestSignedSyntheticCancelWebhookCreatesInboxAndLifecycleReceipts(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, schema := range []string{stripeWebhookReceiptsSchema, stripeTestLifecycleReceiptsSchema} {
		if _, err := db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	const secret = "whsec_synthetic_py1"
	service := NewStripeWebhookService(StripeWebhookOptions{
		DB:     db,
		Config: func() StripeWebhookConfig { return StripeWebhookConfig{WebhookSecret: secret, HasWebhook: true} },
		PersistSubscription: func(_ string, subscription *StripeSubscription) (*shared.SubscriptionStatus, error) {
			if subscription.ID != "sub_synthetic_cancel" || subscription.Status != "canceled" {
				return nil, fmt.Errorf("unexpected synthetic cancel subscription: %+v", subscription)
			}
			return nil, nil
		},
	})
	body := []byte(`{"id":"evt_synthetic_cancel","type":"customer.subscription.deleted","livemode":false,"data":{"object":{"id":"sub_synthetic_cancel","status":"canceled","metadata":{"lpbs_py1_test_run_id":"py1-synthetic-run","lpbs_py1_test_outcome":"cancel"}}}}`)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	signature := fmt.Sprintf("t=%s,v1=%s", timestamp, hex.EncodeToString(mac.Sum(nil)))
	if err := service.HandleWebhook(body, signature); err != nil {
		t.Fatalf("HandleWebhook synthetic cancel: %v", err)
	}
	webhook, err := service.LoadTestOutcomeWebhookReceipt(context.Background(), "py1-synthetic-run", "cancel")
	if err != nil || webhook == nil || webhook.EventID != "evt_synthetic_cancel" || webhook.EventType != "customer.subscription.deleted" || webhook.Status != "processed" || webhook.Livemode {
		t.Fatalf("webhook receipt = %+v, %v", webhook, err)
	}
	lifecycle, err := service.LoadTestLifecycleReceiptByEvent(context.Background(), "evt_synthetic_cancel")
	if err != nil || lifecycle == nil || lifecycle.RunID != "py1-synthetic-run" || lifecycle.SubscriptionStatus != "canceled" || lifecycle.MoneyLedgerStatus != "not_applicable" {
		t.Fatalf("lifecycle receipt = %+v, %v", lifecycle, err)
	}
}
