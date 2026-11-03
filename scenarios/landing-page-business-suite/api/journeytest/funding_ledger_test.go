package journeytest

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	"landing-page-business-suite-api/internal/commerce"
)

func startFundingLedger(t *testing.T) map[string]string {
	t.Helper()
	dir, err := filepath.Abs("../../../money-ledger/api")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "test", ".", "-run", "^TestJourneyFundingLedgerServer$", "-count=1", "-v", "-timeout", "5m")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "VROOLI_JOURNEY_LEDGER_DB="+filepath.Join(t.TempDir(), "ledger.db"))
	input, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		input.Close()
		if err := cmd.Wait(); err != nil {
			t.Errorf("Money Ledger: %v %s", err, stderr.String())
		}
	})
	var startup strings.Builder
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		startup.WriteString(scanner.Text() + "\n")
		if strings.HasPrefix(scanner.Text(), "LEDGER_FIXTURE=") {
			var config map[string]string
			if err := json.Unmarshal([]byte(strings.TrimPrefix(scanner.Text(), "LEDGER_FIXTURE=")), &config); err != nil {
				t.Fatal(err)
			}
			go io.Copy(io.Discard, output)
			t.Cleanup(func() {
				resp, err := http.Post(config["url"]+"/__journey/stop", "text/plain", nil)
				if err == nil {
					resp.Body.Close()
				}
			})
			return config
		}
	}
	t.Fatalf("Money Ledger did not start: %s %s", startup.String(), stderr.String())
	return nil
}
func TestJ16_LedgerRollupMatchesLots(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: c.s.db, Limits: commerce.NewLimitsService(c.s.db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	hold, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 200)
	if err != nil {
		t.Fatal(err)
	}
	if err := wallet.FinalizeReservationWithCost(context.Background(), "account", hold, 200, "web-console", "fixture", 20000, 4, 3, "fixture"); err != nil {
		t.Fatal(err)
	}
	object := map[string]any{"id": "du_ledger", "payment_intent": "pi_fixture", "status": "warning_needs_response", "amount": c.plan.AmountCents, "balance_transactions": []any{}}
	if err := c.reversal(t, "ledger-dispute", "charge.dispute.created", object); err != nil {
		t.Fatal(err)
	}
	var inquiryCost, inquiryRevenue int64
	if err := c.s.db.QueryRow(`SELECT consumed_cost_micros,reversed_revenue_minor FROM funding_losses`).Scan(&inquiryCost, &inquiryRevenue); err != nil {
		t.Fatal(err)
	}
	if inquiryCost != 0 || inquiryRevenue != 0 {
		t.Fatal("inquiry posted a chargeback before funds were withdrawn")
	}
	object["status"] = "needs_response"
	object["balance_transactions"] = []any{"txn_debit"}
	if err := c.reversal(t, "formal-chargeback", "charge.dispute.updated", object); err != nil {
		t.Fatal(err)
	}
	// An initially unexpanded balance transaction is unknown, not a zero fee.
	object["balance_transactions"] = []any{map[string]any{"id": "txn_debit", "amount": -c.plan.AmountCents, "fee": 1500, "currency": "usd"}}
	if err := c.reversal(t, "ledger-expanded-fee", "charge.dispute.funds_withdrawn", object); err != nil {
		t.Fatal(err)
	}
	// Delayed inquiry snapshots for this same dispute cannot undo a formal
	// chargeback or release its source funding.
	object["balance_transactions"] = []any{}
	for _, stale := range []string{"warning_needs_response", "warning_closed"} {
		object["status"] = stale
		if err := c.reversal(t, "stale-"+stale, "charge.dispute.updated", object); err != nil {
			t.Fatal(err)
		}
	}
	var formalState, lotState string
	var formalCost, formalPrincipal int64
	if err := c.s.db.QueryRow(`SELECT d.status,l.status,f.consumed_cost_micros,f.disputed_revenue_minor FROM funding_disputes d JOIN credit_funding_lots l ON l.id=d.lot_id JOIN funding_losses f ON f.lot_id=l.id`).Scan(&formalState, &lotState, &formalCost, &formalPrincipal); err != nil {
		t.Fatal(err)
	}
	if formalState != "needs_response" || lotState != "frozen" || formalCost != 20000 || formalPrincipal != c.plan.AmountCents {
		t.Fatalf("stale inquiry changed formal money: %s %s %d %d", formalState, lotState, formalCost, formalPrincipal)
	}
	object["status"] = "needs_response"
	config := startFundingLedger(t)
	if _, err := c.s.db.Exec(`INSERT INTO funding_ledger_sinks VALUES(FALSE,$1,$2,$3,$4,$5,$6)`, config["url"], config["adapter"], config["book"], config["loss"], config["refund"], config["fee"]); err != nil {
		t.Fatal(err)
	}
	// Production main starts this existing worker and Cleanup joins it. Prove
	// automatic delivery rather than invoking the dispatch helper directly.
	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: c.s.db, Dialect: "postgres", LimitsService: commerce.NewLimitsService(c.s.db, "postgres", nil), LiveMode: &live})
	stop := usage.StartReservationCleanup(10 * time.Millisecond)
	t.Cleanup(stop)
	waitFundingLedgerDelivery(t, c.s)
	// Simulate lost delivery acknowledgement. Replay must keep one posting.
	if _, err := c.s.db.Exec(`UPDATE funding_ledger_outbox SET delivered_at=NULL`); err != nil {
		t.Fatal(err)
	}
	waitFundingLedgerDelivery(t, c.s)
	client := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"])
	postings, err := client.ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 3 {
		t.Fatalf("replay ledger count=%d", len(postings.Msg.Postings))
	}
	sums := map[string]int64{}
	for _, posting := range postings.Msg.Postings {
		sums[posting.Event.Category] += posting.Event.AmountMinor
	}
	var cost, refund, fee int64
	if err := c.s.db.QueryRow(`SELECT consumed_cost_micros,reversed_revenue_minor,fee_minor FROM funding_losses`).Scan(&cost, &refund, &fee); err != nil {
		t.Fatal(err)
	}
	for category, want := range map[string]int64{"funding.credit_topup.loss": cost / 10000, "funding.credit_topup.dispute": refund, "funding.credit_topup.fee": fee} {
		if sums[category] != want {
			t.Errorf("%s ledger=%d lot=%d", category, sums[category], want)
		}
	}
	// An independent dispute remains active while the first is lost and
	// genuinely won later. Corrections cannot thaw the independent funding risk.
	second := map[string]any{"id": "du_independent", "payment_intent": "pi_fixture", "status": "needs_response", "amount": c.plan.AmountCents / 2}
	if err := c.reversal(t, "independent-created", "charge.dispute.created", second); err != nil {
		t.Fatal(err)
	}
	object["status"] = "lost"
	if err := c.reversal(t, "ledger-lost", "charge.dispute.closed", object); err != nil {
		t.Fatal(err)
	}
	// Stripe can reinstate only part of the balance while status remains lost.
	// Immutable transaction IDs make a later older snapshot unable to erase it.
	object["balance_transactions"] = []any{map[string]any{"id": "txn_debit", "amount": -c.plan.AmountCents, "fee": 1500, "currency": "usd"}, map[string]any{"id": "txn_partial_return", "amount": c.plan.AmountCents / 2, "fee": 0, "currency": "usd"}}
	if err := c.reversal(t, "partial-reinstatement", "charge.dispute.funds_reinstated", object); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT status FROM funding_disputes WHERE dispute_id='du_ledger'`).Scan(&formalState); err != nil {
		t.Fatal(err)
	}
	if formalState != "lost" {
		t.Fatal("partial return fabricated a won dispute")
	}
	if err := c.s.db.QueryRow(`SELECT disputed_revenue_minor FROM funding_losses`).Scan(&formalPrincipal); err != nil {
		t.Fatal(err)
	}
	if formalPrincipal != c.plan.AmountCents {
		t.Fatalf("partial return lost: aggregate principal=%d", formalPrincipal)
	}
	object["balance_transactions"] = []any{map[string]any{"id": "txn_debit", "amount": -c.plan.AmountCents, "fee": 1500, "currency": "usd"}}
	if err := c.reversal(t, "older-balance-snapshot", "charge.dispute.updated", object); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT disputed_revenue_minor FROM funding_losses`).Scan(&formalPrincipal); err != nil {
		t.Fatal(err)
	}
	if formalPrincipal != c.plan.AmountCents {
		t.Fatal("stale balance snapshot erased reinstated funds")
	}
	object["status"] = "won"
	object["balance_transactions"] = []any{map[string]any{"id": "txn_debit", "amount": -c.plan.AmountCents, "fee": 1500, "currency": "usd"}, map[string]any{"id": "txn_partial_return", "amount": c.plan.AmountCents / 2, "fee": 0, "currency": "usd"}, map[string]any{"id": "txn_final_return", "amount": c.plan.AmountCents / 2, "fee": -1500, "currency": "usd"}}
	if err := c.reversal(t, "ledger-late-win", "charge.dispute.funds_reinstated", object); err != nil {
		t.Fatal(err)
	}
	var firstState string
	if err := c.s.db.QueryRow(`SELECT status FROM funding_disputes WHERE dispute_id='du_ledger'`).Scan(&firstState); err != nil {
		t.Fatal(err)
	}
	if firstState != "won" {
		t.Fatalf("late win ignored: %s", firstState)
	}
	if err := c.s.db.QueryRow(`SELECT status FROM credit_funding_lots`).Scan(&lotState); err != nil {
		t.Fatal(err)
	}
	if lotState != "frozen" {
		t.Fatal("late win thawed an unrelated active dispute")
	}
	if err := c.s.db.QueryRow(`SELECT consumed_cost_micros,disputed_revenue_minor,fee_minor FROM funding_losses`).Scan(&formalCost, &formalPrincipal, &fee); err != nil {
		t.Fatal(err)
	}
	if formalCost != 20000 || formalPrincipal != c.plan.AmountCents/2 || fee != 0 {
		t.Fatalf("late-win rollup cost=%d principal=%d fee=%d", formalCost, formalPrincipal, fee)
	}
	waitFundingLedgerDelivery(t, c.s)
	second["status"] = "won"
	if err := c.reversal(t, "independent-won", "charge.dispute.closed", second); err != nil {
		t.Fatal(err)
	}
	waitFundingLedgerDelivery(t, c.s)
	postings, err = client.ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	total := int64(0)
	for _, posting := range postings.Msg.Postings {
		total += posting.Event.AmountMinor
	}
	if total != 0 || len(postings.Msg.Postings) != 9 {
		t.Fatalf("won rollup=%d postings=%d", total, len(postings.Msg.Postings))
	}
	if err := c.s.db.QueryRow(`SELECT status FROM credit_funding_lots`).Scan(&lotState); err != nil {
		t.Fatal(err)
	}
	if lotState != "open" {
		t.Fatal("settled independent disputes did not restore funding")
	}
}

func waitFundingLedgerDelivery(t *testing.T, s *spine) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var pending int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM funding_ledger_outbox WHERE delivered_at IS NULL`).Scan(&pending); err != nil {
			t.Fatal(err)
		}
		if pending == 0 {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("automatic funding Ledger delivery did not finish: %v", ctx.Err())
		case <-ticker.C:
		}
	}
}

func assertAutomaticFundingLossOnce(t *testing.T, c *checkoutFixture, costMicros int64) {
	t.Helper()
	config := startFundingLedger(t)
	if _, err := c.s.db.Exec(`INSERT INTO funding_ledger_sinks VALUES(FALSE,$1,$2,$3,$4,$5,$6)`, config["url"], config["adapter"], config["book"], config["loss"], config["refund"], config["fee"]); err != nil {
		t.Fatal(err)
	}
	live := false
	usage := commerce.NewUsageServiceWithOptions(commerce.UsageServiceOptions{DB: c.s.db, Dialect: "postgres", LimitsService: commerce.NewLimitsService(c.s.db, "postgres", nil), LiveMode: &live})
	stop := usage.StartReservationCleanup(10 * time.Millisecond)
	t.Cleanup(stop)
	waitFundingLedgerDelivery(t, c.s)
	if _, err := c.s.db.Exec(`UPDATE funding_ledger_outbox SET delivered_at=NULL WHERE category='loss'`); err != nil {
		t.Fatal(err)
	}
	waitFundingLedgerDelivery(t, c.s)
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(context.Background(), connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	total := int64(0)
	for _, posting := range postings.Msg.Postings {
		if posting.Event.Category == "funding.credit_topup.loss" {
			count++
			total += posting.Event.AmountMinor
		}
	}
	if count != 1 || total != costMicros/10000 {
		t.Fatalf("automatic loss journal duplicated or lost: count=%d amount=%d", count, total)
	}
}
