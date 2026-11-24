package journeytest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	"landing-page-business-suite-api/internal/commerce"
)

type py1JourneyLedgerToken string

func (t py1JourneyLedgerToken) Token(context.Context) (string, error) { return string(t), nil }

func TestPY1TestCheckoutSignedWebhookPersistsPostgresReceiptAndLedgerPosting(t *testing.T) {
	c := newCheckout(t)
	const runID = "py1-journey-test-001"
	const outcome = "credit_topup"
	const sessionID = "cs_test_py1_journey_001"
	if _, err := c.s.db.Exec(`INSERT INTO checkout_sessions(session_id,customer_email,price_id,status,session_type,amount_cents,business_account_id) VALUES($1,'actor@example.invalid',$2,'pending','credits_topup',$3,'account')`, sessionID, c.plan.StripePriceId, c.plan.AmountCents); err != nil {
		t.Fatal(err)
	}
	event := map[string]any{
		"id": "evt_py1_journey_001", "type": "checkout.session.completed", "livemode": false,
		"data": map[string]any{"object": map[string]any{
			"id": sessionID, "customer_email": "actor@example.invalid", "customer": "cus_test_py1_journey",
			"payment_intent": "pi_test_py1_journey", "payment_status": "paid",
			"amount_total": float64(c.plan.AmountCents), "currency": "usd",
			"metadata": map[string]string{"lpbs_py1_test_run_id": runID, "lpbs_py1_test_outcome": outcome},
		}},
	}
	body, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	const webhookSecret = "whsec_journey"
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	_, _ = mac.Write([]byte(timestamp + "."))
	_, _ = mac.Write(body)
	signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if err := c.webhook.HandleWebhook(body, signature); err != nil {
		t.Fatalf("signed Stripe test webhook: %v", err)
	}
	economic, err := c.webhook.LoadTestEconomicReceipt(context.Background(), runID, outcome)
	if err != nil || economic == nil || economic.EventID != "evt_py1_journey_001" || economic.CheckoutSessionID != sessionID || economic.PaymentIntentID != "pi_test_py1_journey" || economic.AmountMinor != c.plan.AmountCents || economic.Currency != "usd" || economic.MoneyLedgerStatus != "pending" {
		t.Fatalf("durable Postgres economic receipt = %+v, err=%v", economic, err)
	}
	var checkoutStatus string
	if err := c.s.db.QueryRow(`SELECT status FROM checkout_sessions WHERE session_id=$1`, sessionID).Scan(&checkoutStatus); err != nil || checkoutStatus != "complete" {
		t.Fatalf("checkout state = %q, err=%v", checkoutStatus, err)
	}

	ledgerConfig := startFundingLedger(t)
	writer, err := commerce.NewMoneyLedgerPY1TestServiceWriter(ledgerConfig["url"], py1JourneyLedgerToken(ledgerConfig["landing_page_token"]), &http.Client{Timeout: 10 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	target := commerce.PY1TestLedgerTarget{AdapterID: ledgerConfig["adapter"], BookID: ledgerConfig["book"], RevenueAccountID: ledgerConfig["service_order_revenue"]}
	posted, err := c.webhook.DispatchTestEconomicReceipt(context.Background(), writer, target, runID, outcome)
	if err != nil || posted == nil || posted.MoneyLedgerStatus != "posted" || posted.MoneyLedgerPostingID == "" || posted.MoneyLedgerReceiptID == "" {
		t.Fatalf("Money Ledger receipt = %+v, err=%v", posted, err)
	}
	if duplicate, err := c.webhook.DispatchTestEconomicReceipt(context.Background(), writer, target, runID, outcome); err != nil || duplicate.MoneyLedgerPostingID != posted.MoneyLedgerPostingID || duplicate.MoneyLedgerReceiptID != posted.MoneyLedgerReceiptID {
		t.Fatalf("idempotent Money Ledger retry = %+v, err=%v", duplicate, err)
	}
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, ledgerConfig["url"]).ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: ledgerConfig["book"], Limit: 20}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.GetPostings()) != 1 {
		t.Fatalf("Money Ledger postings = %d; want exactly one", len(postings.Msg.GetPostings()))
	}
	actual := postings.Msg.GetPostings()[0]
	if actual.GetId() != posted.MoneyLedgerPostingID || actual.GetEvent().GetExternalId() != "lpbs-py1-checkout:"+sessionID || actual.GetEvent().GetAmountMinor() != c.plan.AmountCents || actual.GetEvent().GetCurrency() != "USD" || actual.GetEvent().GetCategory() != "test.py1.stripe_checkout" || actual.GetEvent().GetAccountId() != target.RevenueAccountID {
		t.Fatalf("accepted Money Ledger posting does not match paid Checkout: %+v", actual)
	}
	var count int
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM py1_test_economic_receipts WHERE stripe_event_id=$1 AND money_ledger_status='posted' AND money_ledger_posting_id=$2 AND money_ledger_receipt_id=$3`, economic.EventID, posted.MoneyLedgerPostingID, posted.MoneyLedgerReceiptID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("Postgres accepted-ledger link count = %d, err=%v", count, err)
	}
	t.Logf("synthetic signed test webhook=%s checkout=%s payment_intent=%s amount=%d %s Postgres receipt=persisted Money Ledger external_id=%s posting=%s receipt=%s category=%s", economic.EventID, economic.CheckoutSessionID, economic.PaymentIntentID, economic.AmountMinor, strings.ToUpper(economic.Currency), economic.MoneyLedgerExternalID, posted.MoneyLedgerPostingID, posted.MoneyLedgerReceiptID, actual.GetEvent().GetCategory())
}
