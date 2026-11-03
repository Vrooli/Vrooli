package journeytest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"landing-page-business-suite-api/internal/commerce"
	"slices"
	"testing"
	"time"
)

func (c *checkoutFixture) reversal(t *testing.T, id, kind string, object map[string]any) error {
	t.Helper()
	body, err := json.Marshal(map[string]any{"id": id, "type": kind, "livemode": false, "data": map[string]any{"object": object}})
	if err != nil {
		t.Fatal(err)
	}
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte("whsec_journey"))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	return c.webhook.HandleWebhook(body, "t="+timestamp+",v1="+hex.EncodeToString(mac.Sum(nil)))
}
func TestJ16_DisputeFreezesUnspentAndRecordsLoss(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	resp := c.s.request(t, c.s.key.Token, "before-dispute", "hello")
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("spend %d", resp.StatusCode)
	}
	object := map[string]any{"id": "du_fixture", "charge": "ch_fixture", "status": "needs_response", "amount": c.plan.AmountCents}
	for i := 0; i < 2; i++ {
		if err := c.reversal(t, "dispute", "charge.dispute.created", object); err != nil {
			t.Fatal(err)
		}
	}
	var status string
	if err := c.s.db.QueryRow(`SELECT status FROM credit_funding_lots`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "frozen" {
		t.Fatalf("disputed funding remained %s", status)
	}
	var loss int64
	if err := c.s.db.QueryRow(`SELECT COALESCE(SUM(consumed_cost_micros),0) FROM funding_losses`).Scan(&loss); err != nil {
		t.Fatal(err)
	}
	if loss != 2 {
		t.Fatalf("lost provider cost=%d want 2", loss)
	}
	resp = c.s.request(t, c.s.key.Token, "after-dispute", "hello")
	resp.Body.Close()
	if resp.StatusCode != 402 {
		t.Fatalf("disputed lot still spends: %d", resp.StatusCode)
	}
	// Multiple dispute IDs on one payment are independent provider facts.
	object["id"] = "du_second"
	if err := c.reversal(t, "second-dispute", "charge.dispute.created", object); err != nil {
		t.Fatal(err)
	}
	object["id"] = "du_fixture"
	object["status"] = "won"
	if err := c.reversal(t, "first-won", "charge.dispute.closed", object); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT status FROM credit_funding_lots`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "frozen" {
		t.Fatal("first win thawed a second active dispute")
	}
	object["id"] = "du_second"
	if err := c.reversal(t, "second-won", "charge.dispute.closed", object); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT status FROM credit_funding_lots`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "open" {
		t.Fatal("all disputes won did not restore funding")
	}

}

func TestJ16_RefundStopsLotBeingSpendable(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	object := map[string]any{"id": "ch_fixture", "payment_intent": "pi_fixture", "amount_refunded": c.plan.AmountCents}
	for i := 0; i < 2; i++ {
		if err := c.reversal(t, "refund", "charge.refunded", object); err != nil {
			t.Fatal(err)
		}
	}
	resp := c.s.request(t, c.s.key.Token, "after-refund", "hello")
	resp.Body.Close()
	if resp.StatusCode != 402 {
		t.Fatalf("refunded lot still spends: %d", resp.StatusCode)
	}
	var remaining int64
	if err := c.s.db.QueryRow(`SELECT remaining FROM credit_funding_lots`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("refunded lot remaining %d", remaining)
	}
	// A stale partial refund cannot restore any balance.
	object["amount_refunded"] = c.plan.AmountCents / 2
	if err := c.reversal(t, "refund-older", "charge.refunded", object); err != nil {
		t.Fatal(err)
	}
}
func TestJ16_FreezeRacedAgainstFinalizeNeverDoubleCounts(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: c.s.db, Limits: commerce.NewLimitsService(c.s.db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	hold, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 200)
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		<-start
		done <- wallet.FinalizeReservationWithCost(context.Background(), "account", hold, 200, "web-console", "model", 2, 4, 3, "fixture")
	}()
	object := map[string]any{"id": "du_race", "payment_intent": "pi_fixture", "status": "needs_response"}
	go func() { <-start; done <- c.reversal(t, "race", "charge.dispute.created", object) }()
	close(start)
	for i := 0; i < 2; i++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}
	if err := wallet.FinalizeReservationWithCost(context.Background(), "account", hold, 200, "web-console", "model", 2, 4, 3, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := c.reversal(t, "race", "charge.dispute.created", object); err != nil {
		t.Fatal(err)
	}
	var loss, allocations, remaining, granted int64
	if err := c.s.db.QueryRow(`SELECT SUM(consumed_cost_micros) FROM funding_losses`).Scan(&loss); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT SUM(credits) FROM credit_lot_allocations`).Scan(&allocations); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT remaining,granted FROM credit_funding_lots`).Scan(&remaining, &granted); err != nil {
		t.Fatal(err)
	}
	if loss != 2 || allocations != 200 || remaining+allocations != granted {
		t.Fatalf("race loss=%d allocations=%d remaining=%d grant=%d", loss, allocations, remaining, granted)
	}
	for _, prior := range []string{"won_dispute", "partial_refund"} {
		t.Run(prior, func(t *testing.T) { assertSettlementAcrossRestrictionCycles(t, prior) })
	}

}
func TestJ16_CapsAndHoldsByAccountAge(t *testing.T) {
	c := newCheckout(t)
	if _, err := c.s.db.Exec(`INSERT INTO funding_source_policies VALUES('fixture-only','stripe','credit_topup',86400,3600,100,500)`); err != nil {
		t.Fatal(err)
	}
	c.event(t, "paid", "checkout.session.completed", "paid")
	resp := c.s.request(t, c.s.key.Token, "young-held", "hello")
	resp.Body.Close()
	if resp.StatusCode != 402 {
		t.Fatalf("young held credit spent: %d", resp.StatusCode)
	}
	if _, err := c.s.db.Exec(`UPDATE credit_funding_lots SET spendable_after=NOW()-interval '1 second'`); err != nil {
		t.Fatal(err)
	}
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: c.s.db, Limits: commerce.NewLimitsService(c.s.db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	if _, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 200); err == nil {
		t.Fatal("young account cap ignored")
	}
	if _, err := c.s.db.Exec(`UPDATE business_accounts SET created_at=NOW()-interval '2 days'`); err != nil {
		t.Fatal(err)
	}
	hold, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 200)
	if err != nil {
		t.Fatal(err)
	}
	if hold == "" {
		t.Fatal("mature account hold missing")
	}
	if _, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 400); err == nil {
		t.Fatal("concurrent hold evaded mature age cap")
	}
}

type blockedStripe struct{ called bool }

func (s *blockedStripe) Request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	s.called = true
	return nil, errors.New("blocked purchase reached Stripe")
}
func TestJ16_BlockAfterLostDispute(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	object := map[string]any{"id": "du_lost", "payment_intent": "pi_fixture", "status": "lost"}
	if err := c.reversal(t, "lost", "charge.dispute.closed", object); err != nil {
		t.Fatal(err)
	}
	stripe := &blockedStripe{}
	checkout := commerce.NewCheckoutService(commerce.CheckoutServiceOptions{Store: c.s.db, Plans: c.plans, Requester: stripe})
	if _, err := checkout.CreateForBusinessAccount(context.Background(), c.plan.StripePriceId, "/success", "/cancel", "actor@example.invalid", "account"); err == nil {
		t.Fatal("lost dispute did not block purchase")
	}
	if stripe.called {
		t.Fatal("blocked checkout contacted Stripe")
	}
	// A delayed created event cannot reopen a lost dispute or unblock purchases.
	object["status"] = "needs_response"
	if err := c.reversal(t, "late-created", "charge.dispute.created", object); err != nil {
		t.Fatal(err)
	}
	var status string
	if err := c.s.db.QueryRow(`SELECT status FROM funding_disputes`).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "lost" {
		t.Fatalf("late event changed closure: %s", status)
	}
}

func assertSettlementAcrossRestrictionCycles(t *testing.T, prior string) {
	t.Helper()
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	if prior == "won_dispute" {
		first := map[string]any{"id": "du_old_cycle", "payment_intent": "pi_fixture", "status": "needs_response", "amount": c.plan.AmountCents}
		if err := c.reversal(t, "old-dispute", "charge.dispute.created", first); err != nil {
			t.Fatal(err)
		}
		first["status"] = "won"
		if err := c.reversal(t, "old-won", "charge.dispute.closed", first); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := c.reversal(t, "old-partial-refund", "charge.refunded", map[string]any{"id": "ch_fixture", "payment_intent": "pi_fixture", "amount_refunded": c.plan.AmountCents / 2}); err != nil {
			t.Fatal(err)
		}
	}
	live := false
	wallet := commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: c.s.db, Limits: commerce.NewLimitsService(c.s.db, "postgres", nil), Dialect: "postgres", LiveMode: &live})
	hold, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 200)
	if err != nil {
		t.Fatal(err)
	}
	var admittedAfterPrior bool
	if err := c.s.db.QueryRow(`SELECT p.restricted_at<r.created_at FROM funding_payment_states p JOIN business_account_credit_reservations r ON r.business_account_id='account' WHERE r.id=$1`, hold).Scan(&admittedAfterPrior); err != nil {
		t.Fatal(err)
	}
	if !admittedAfterPrior {
		t.Fatal("fixture did not admit a fresh hold after the prior restriction")
	}
	second := map[string]any{"id": "du_new_cycle", "payment_intent": "pi_fixture", "status": "needs_response", "amount": c.plan.AmountCents}
	if err := c.reversal(t, "new-cycle-dispute", "charge.dispute.created", second); err != nil {
		t.Fatal(err)
	}
	if err := wallet.FinalizeReservationWithCost(context.Background(), "account", hold, 200, "web-console", "model", 20000, 4, 3, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := wallet.FinalizeReservationWithCost(context.Background(), "account", hold, 200, "web-console", "model", 20000, 4, 3, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := c.reversal(t, "new-cycle-dispute", "charge.dispute.created", second); err != nil {
		t.Fatal(err)
	}
	var status string
	var remaining, charged, allocated, cost int64
	var events, allocations int
	if err := c.s.db.QueryRow(`SELECT status,remaining FROM credit_funding_lots`).Scan(&status, &remaining); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(credits),0) FROM business_account_usage_events WHERE operation_id=$1`, hold).Scan(&events, &charged); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT COUNT(*),COALESCE(SUM(credits),0) FROM credit_lot_allocations WHERE operation_id=$1`, hold).Scan(&allocations, &allocated); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT consumed_cost_micros FROM funding_losses`).Scan(&cost); err != nil {
		t.Fatal(err)
	}
	if status != "frozen" || remaining < 0 || events != 1 || allocations != 1 || charged != 200 || allocated != 200 || cost != 20000 {
		t.Fatalf("new restriction settlement: status=%s remaining=%d events=%d allocations=%d charged=%d allocated=%d cost=%d", status, remaining, events, allocations, charged, allocated, cost)
	}
	if _, err := wallet.ReserveCredits(context.Background(), "account", "actor@example.invalid", "free", "ai_credits", 1); err == nil {
		t.Fatal("new spending bypassed the second freeze")
	}
	assertAutomaticFundingLossOnce(t, c, 20000)
}

func TestJ17_FundingDisputeSpecRealTraces(t *testing.T) {
	spec := loadPaymentContract(t)
	pairs := map[[2]string]string{}
	for _, row := range spec.Transitions {
		pairs[[2]string{row.FromState, row.Event}] = row.To
	}
	if len(spec.Traces) == 0 {
		t.Fatal("missing dispute traces")
	}
	for _, trace := range spec.Traces {
		t.Run(trace.ID, func(t *testing.T) {
			c := newCheckout(t)
			c.event(t, "paid", "checkout.session.completed", "paid")
			object := map[string]any{"id": "du_spec", "payment_intent": "pi_fixture", "amount": c.plan.AmountCents}
			state := ""
			check := func(want string) {
				t.Helper()
				var actual, lot string
				if err := c.s.db.QueryRow(`SELECT d.status,l.status FROM funding_disputes d JOIN credit_funding_lots l ON l.id=d.lot_id WHERE d.dispute_id='du_spec'`).Scan(&actual, &lot); err != nil {
					t.Fatal(err)
				}
				wantLot := "frozen"
				if want == "won" || want == "warning_closed" || want == "prevented" {
					wantLot = "open"
				}
				if actual != want || lot != wantLot {
					t.Fatalf("dispute=%s lot=%s want %s/%s", actual, lot, want, wantLot)
				}
			}
			if trace.Initial != "" {
				object["status"] = trace.Initial
				if err := c.reversal(t, "initial", "charge.dispute.created", object); err != nil {
					t.Fatal(err)
				}
				state = trace.Initial
				check(state)
			}
			if len(trace.Incoming) != len(trace.Expected) {
				t.Fatal("invalid dispute trace")
			}
			for i, event := range trace.Incoming {
				want, ok := pairs[[2]string{state, event}]
				if !ok || want != trace.Expected[i] {
					t.Fatal("dispute trace drifted from matrix")
				}
				object["status"] = event
				if err := c.reversal(t, fmt.Sprintf("step_%d", i), "charge.dispute.updated", object); err != nil {
					t.Fatal(err)
				}
				check(want)
				// Exact retry of each signed provider event must preserve the observation.
				if err := c.reversal(t, fmt.Sprintf("step_%d", i), "charge.dispute.updated", object); err != nil {
					t.Fatal(err)
				}
				check(want)
				t.Logf("SM4 real dispute trace %s: %q --%s--> %s", trace.ID, state, event, want)
				state = want
			}
		})
	}
}

func TestJ17_FundingRefundSpecMatrixAndTrace(t *testing.T) {
	spec := loadPaymentContract(t).Refund
	if !slices.Equal(spec.States, []string{"unrefunded", "partial", "full"}) || !slices.Equal(spec.Events, []string{"zero", "half", "full", "negative", "overpaid"}) || len(spec.Traces) == 0 {
		t.Fatal("missing refund contract")
	}
	pairs := map[[2]string]paymentSpecRow{}
	for _, row := range spec.Transitions {
		key := [2]string{row.FromState, row.Event}
		if _, duplicate := pairs[key]; duplicate || !slices.Contains(spec.States, row.FromState) || !slices.Contains(spec.States, row.To) || !slices.Contains(spec.Events, row.Event) {
			t.Fatalf("duplicate/unknown refund pair %+v", row)
		}
		pairs[key] = row
	}
	c := newCheckout(t)
	paid := paymentSpecEvent{ID: "paid", Kind: "checkout.session.completed", Payment: "paid"}
	amount := func(event string) int64 {
		switch event {
		case "zero":
			return 0
		case "half":
			return c.plan.AmountCents / 2
		case "full":
			return c.plan.AmountCents
		case "negative":
			return -1
		case "overpaid":
			return c.plan.AmountCents + 1
		default:
			t.Fatalf("unknown refund event %q", event)
			return 0
		}
	}
	apply := func(t *testing.T, session, id, event string) error {
		t.Helper()
		return c.reversal(t, id, "charge.refunded", map[string]any{"id": "ch_" + session, "payment_intent": "pi_" + session, "amount_refunded": amount(event)})
	}
	check := func(t *testing.T, session, state string) {
		t.Helper()
		var refunded, remaining, granted, ledger, rollup int64
		var lot string
		if err := c.s.db.QueryRow(`SELECT l.status,l.remaining,l.granted,COALESCE(p.refunded_minor,0),COALESCE((SELECT SUM(o.amount_minor) FROM funding_ledger_outbox o WHERE o.lot_id=l.id AND o.category='refund'),0),COALESCE((SELECT SUM(f.refunded_revenue_minor) FROM funding_losses f WHERE f.lot_id=l.id),0) FROM credit_funding_lots l LEFT JOIN funding_payment_states p ON p.lot_id=l.id WHERE l.provider_ref=$1`, "pi_"+session).Scan(&lot, &remaining, &granted, &refunded, &ledger, &rollup); err != nil {
			t.Fatal(err)
		}
		want := int64(0)
		wantLot := "open"
		switch state {
		case "unrefunded":
		case "partial":
			want = c.plan.AmountCents / 2
		case "full":
			want = c.plan.AmountCents
			wantLot = "reversed"
		default:
			t.Fatalf("unknown refund state %q", state)
		}
		wantRemaining := granted - (granted*want+c.plan.AmountCents-1)/c.plan.AmountCents
		if refunded != want || ledger != want || rollup != want || remaining != wantRemaining || lot != wantLot {
			t.Fatalf("refund %s: total=%d ledger=%d rollup=%d remaining=%d status=%s want total=%d remaining=%d status=%s", session, refunded, ledger, rollup, remaining, lot, want, wantRemaining, wantLot)
		}
	}
	create := func(t *testing.T, session string) {
		t.Helper()
		c.createSpecCheckout(t, session)
		if err := c.paymentSpecEvent(t, session, "paid_"+session, paid); err != nil {
			t.Fatal(err)
		}
		check(t, session, "unrefunded")
	}
	index := 0
	for _, state := range spec.States {
		for _, event := range spec.Events {
			row, ok := pairs[[2]string{state, event}]
			if !ok {
				t.Fatalf("missing refund pair %s/%s", state, event)
			}
			index++
			t.Run(state+"/"+event, func(t *testing.T) {
				session := fmt.Sprintf("sm4_refund_%d", index)
				create(t, session)
				if state != "unrefunded" {
					seed := "half"
					if state == "full" {
						seed = "full"
					}
					if err := apply(t, session, "seed_"+session, seed); err != nil {
						t.Fatal(err)
					}
				}
				check(t, session, state)
				err := apply(t, session, "apply_"+session, event)
				if (err != nil) != row.Reject {
					t.Fatalf("refund error %v want rejection %t", err, row.Reject)
				}
				check(t, session, row.To)
				t.Logf("SM4 real refund: %s --%s--> %s reject=%t", state, event, row.To, row.Reject)
			})
		}
	}
	for _, trace := range spec.Traces {
		t.Run(trace.ID, func(t *testing.T) {
			session := "sm4_" + trace.ID
			create(t, session)
			state := trace.Initial
			if state != "unrefunded" || len(trace.Events) != len(trace.Expected) {
				t.Fatal("invalid refund trace")
			}
			for i, event := range trace.Events {
				row, ok := pairs[[2]string{state, event}]
				if !ok || row.To != trace.Expected[i] {
					t.Fatal("refund trace drifted from matrix")
				}
				err := apply(t, session, session+"_"+event, event)
				if (err != nil) != row.Reject {
					t.Fatalf("refund trace error %v want rejection %t", err, row.Reject)
				}
				check(t, session, row.To)
				t.Logf("SM4 real refund trace %s: %s --%s--> %s", trace.ID, state, event, row.To)
				state = row.To
			}
		})
	}
}

func TestJ17_FundingFraudWarningSpecMatrixAndTrace(t *testing.T) {
	spec := loadPaymentContract(t).FraudWarning
	if !slices.Equal(spec.States, []string{"clear", "warned"}) || !slices.Equal(spec.Events, []string{"radar.early_fraud_warning.created", "radar.early_fraud_warning.updated"}) {
		t.Fatal("warning vocabulary drift")
	}
	pairs := map[[2]string]paymentSpecRow{}
	for _, row := range spec.Transitions {
		key := [2]string{row.FromState, row.Event}
		if _, duplicate := pairs[key]; duplicate || !slices.Contains(spec.States, row.FromState) || !slices.Contains(spec.States, row.To) || !slices.Contains(spec.Events, row.Event) {
			t.Fatalf("duplicate/unknown warning pair %+v", row)
		}
		pairs[key] = row
	}
	c := newCheckout(t)
	apply := func(t *testing.T, session, id, event string) {
		t.Helper()
		if err := c.reversal(t, id, event, map[string]any{"id": "issfr_" + session, "payment_intent": "pi_" + session}); err != nil {
			t.Fatal(err)
		}
	}
	check := func(t *testing.T, session, want string) {
		t.Helper()
		var warning, conserved bool
		var status string
		if err := c.s.db.QueryRow(`SELECT COALESCE(p.fraud_warning,FALSE),l.status,l.remaining=l.granted FROM credit_funding_lots l LEFT JOIN funding_payment_states p ON p.lot_id=l.id WHERE l.provider_ref=$1`, "pi_"+session).Scan(&warning, &status, &conserved); err != nil {
			t.Fatal(err)
		}
		wantStatus := "open"
		if want == "warned" {
			wantStatus = "frozen"
		}
		if warning != (want == "warned") || status != wantStatus || !conserved {
			t.Fatalf("warning=%t lot=%s conserved=%t want %s", warning, status, conserved, want)
		}
	}
	create := func(t *testing.T, session string) {
		t.Helper()
		c.createSpecCheckout(t, session)
		if err := c.paymentSpecEvent(t, session, "paid_"+session, paymentSpecEvent{Kind: "checkout.session.completed", Payment: "paid"}); err != nil {
			t.Fatal(err)
		}
		check(t, session, "clear")
	}
	index := 0
	for _, state := range spec.States {
		for _, event := range spec.Events {
			row, ok := pairs[[2]string{state, event}]
			if !ok {
				t.Fatalf("missing warning pair %s/%s", state, event)
			}
			index++
			t.Run(state+"/"+event, func(t *testing.T) {
				session := fmt.Sprintf("sm4_warning_%d", index)
				create(t, session)
				if state == "warned" {
					apply(t, session, "seed_"+session, spec.Events[0])
				}
				check(t, session, state)
				apply(t, session, "apply_"+session, event)
				check(t, session, row.To)
				t.Logf("SM4 real warning: %s --%s--> %s", state, event, row.To)
			})
		}
	}
	for _, trace := range spec.Traces {
		t.Run(trace.ID, func(t *testing.T) {
			session := "sm4_" + trace.ID
			create(t, session)
			state := trace.Initial
			if state != "clear" || len(trace.Events) == 0 || len(trace.Events) != len(trace.Expected) {
				t.Fatal("invalid warning trace")
			}
			for i, event := range trace.Events {
				row, ok := pairs[[2]string{state, event}]
				if !ok || row.To != trace.Expected[i] {
					t.Fatal("warning trace drifted from matrix")
				}
				apply(t, session, session+event, event)
				check(t, session, row.To)
				t.Logf("SM4 real warning trace %s: %s --%s--> %s", trace.ID, state, event, row.To)
				state = row.To
			}
		})
	}
}
