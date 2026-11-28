package journeytest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"connectrpc.com/connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	"landing-page-business-suite-api/internal/commerce"
)

func TestPY1SignedLifecycleEventsPersistPostgresAndLedgerReceipts(t *testing.T) {
	c := newCheckout(t)
	const runID = "py1-lifecycle-postgres-001"
	const signingSecret = "whsec_lifecycle_journey"
	webhook := commerce.NewStripeWebhookService(commerce.StripeWebhookOptions{
		DB: c.s.db, PlanService: c.plans,
		Config: func() commerce.StripeWebhookConfig {
			return commerce.StripeWebhookConfig{WebhookSecret: signingSecret, HasWebhook: true}
		},
		RefreshSubscription: func(string, string) (*shared.SubscriptionStatus, error) { return nil, nil },
		PersistSubscription: func(_ string, _ *commerce.StripeSubscription) (*shared.SubscriptionStatus, error) { return nil, nil },
	})

	paidInvoice := map[string]any{
		"id": "in_test_cycle_001", "subscription": "sub_test_renewal_001", "customer": "cus_test_renewal_001",
		"amount_paid": float64(2900), "currency": "usd", "billing_reason": "subscription_cycle",
		"subscription_details": map[string]any{"metadata": map[string]string{"lpbs_py1_test_run_id": runID, "lpbs_py1_test_outcome": "renewal"}},
	}
	failedInvoice := map[string]any{
		"id": "in_test_failed_001", "subscription": "sub_test_past_due_001", "customer": "cus_test_past_due_001",
		"subscription_details": map[string]any{"metadata": map[string]string{"lpbs_py1_test_run_id": runID, "lpbs_py1_test_outcome": "past_due"}},
	}
	pastDueSubscription := map[string]any{
		"id": "sub_test_past_due_001", "status": "past_due",
		"metadata": map[string]string{"lpbs_py1_test_run_id": runID, "lpbs_py1_test_outcome": "past_due"},
	}
	for _, event := range []struct {
		id, kind string
		object   map[string]any
	}{
		{"evt_test_cycle_001", "invoice.paid", paidInvoice},
		{"evt_test_failed_001", "invoice.payment_failed", failedInvoice},
		{"evt_test_past_due_001", "customer.subscription.updated", pastDueSubscription},
	} {
		if err := handleSignedPY1LifecycleWebhook(t, webhook, signingSecret, event.id, event.kind, event.object); err != nil {
			t.Fatalf("handle signed %s: %v", event.kind, err)
		}
	}

	renewal, err := webhook.LoadTestLifecycleRenewalReceipt(context.Background(), runID, "renewal")
	if err != nil || renewal == nil || renewal.EventID != "evt_test_cycle_001" || renewal.PaymentStatus != "paid" || renewal.BillingReason != "subscription_cycle" || renewal.MoneyLedgerStatus != "pending" {
		t.Fatalf("Postgres paid renewal receipt = %+v, err=%v", renewal, err)
	}
	failed, err := webhook.LoadTestLifecycleReceiptByOutcomeAndEventType(context.Background(), runID, "past_due", "invoice.payment_failed")
	if err != nil || failed == nil || failed.EventID != "evt_test_failed_001" || failed.SubscriptionStatus != "" || failed.PaymentStatus != "failed" || failed.MoneyLedgerStatus != "not_applicable" {
		t.Fatalf("Postgres failed-invoice receipt = %+v, err=%v", failed, err)
	}
	state, err := webhook.LoadTestLifecycleReceiptByOutcomeEventTypeAndStatus(context.Background(), runID, "past_due", "customer.subscription.updated", "past_due")
	if err != nil || state == nil || state.EventID != "evt_test_past_due_001" || state.SubscriptionID != failed.SubscriptionID {
		t.Fatalf("Postgres past-due status receipt = %+v, err=%v", state, err)
	}

	ledgerConfig := startFundingLedger(t)
	writer, err := commerce.NewMoneyLedgerPY1TestServiceWriter(ledgerConfig["url"], py1JourneyLedgerToken(ledgerConfig["landing_page_token"]), &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	target := commerce.PY1TestLedgerTarget{AdapterID: ledgerConfig["adapter"], BookID: ledgerConfig["book"], RevenueAccountID: ledgerConfig["service_order_revenue"]}
	posted, err := webhook.DispatchTestLifecycleReceipt(context.Background(), writer, target, runID, "renewal")
	if err != nil || posted == nil || posted.MoneyLedgerStatus != "posted" || posted.MoneyLedgerExternalID != "lpbs-py1-invoice:in_test_cycle_001" || posted.MoneyLedgerPostingID == "" || posted.MoneyLedgerReceiptID == "" {
		t.Fatalf("Money Ledger lifecycle posting = %+v, err=%v", posted, err)
	}
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, ledgerConfig["url"]).ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: ledgerConfig["book"], Limit: 20}))
	if err != nil || len(postings.Msg.GetPostings()) != 1 {
		t.Fatalf("Money Ledger postings = %d, err=%v; want one paid renewal", len(postings.Msg.GetPostings()), err)
	}
	actual := postings.Msg.GetPostings()[0]
	if actual.GetId() != posted.MoneyLedgerPostingID || actual.GetEvent().GetExternalId() != posted.MoneyLedgerExternalID || actual.GetEvent().GetAmountMinor() != 2900 || actual.GetEvent().GetCurrency() != "USD" || actual.GetEvent().GetCategory() != "test.py1.stripe_subscription_invoice" {
		t.Fatalf("accepted lifecycle posting mismatch: %+v", actual)
	}
	if err := webhook.DispatchPendingTestLifecycleReceipts(context.Background(), writer, target, runID); err != nil {
		t.Fatal(err)
	}
	if count, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, ledgerConfig["url"]).ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: ledgerConfig["book"], Limit: 20})); err != nil || len(count.Msg.GetPostings()) != 1 {
		t.Fatalf("lifecycle outbox retry duplicated the posting: count=%d err=%v", len(count.Msg.GetPostings()), err)
	}
	t.Logf("synthetic signed renewal=%s failed-invoice=%s status=%s Postgres lifecycle receipts=persisted Money Ledger posting=%s receipt=%s", renewal.EventID, failed.EventID, state.EventID, posted.MoneyLedgerPostingID, posted.MoneyLedgerReceiptID)
}

func handleSignedPY1LifecycleWebhook(t *testing.T, service *commerce.StripeWebhookService, secret, eventID, eventType string, object map[string]any) error {
	t.Helper()
	body, err := json.Marshal(map[string]any{"id": eventID, "type": eventType, "livemode": false, "data": map[string]any{"object": object}})
	if err != nil {
		return err
	}
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	return service.HandleWebhook(body, signature)
}
