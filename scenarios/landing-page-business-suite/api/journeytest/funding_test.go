package journeytest

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/shared"
	"io"
	"landing-page-business-suite-api/internal/aidoor"
	"landing-page-business-suite-api/internal/analytics"
	"landing-page-business-suite-api/internal/commerce"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"
)

type checkoutFixture struct {
	s       *spine
	webhook *commerce.StripeWebhookService
	plan    *commerce.PlanOption
	plans   *commerce.PlanService
}

func newCheckout(t *testing.T) *checkoutFixture {
	t.Helper()
	s := newSpine(t)
	for _, schema := range []string{commerce.FinancialSchema(), analytics.Schema(), analytics.IndexesSchema()} {
		if _, err := s.db.Exec(schema); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.db.Exec(`DELETE FROM credit_funding_lots`); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile("../../.vrooli/plans.json")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "plans.json")
	if err := os.WriteFile(path, source, 0600); err != nil {
		t.Fatal(err)
	}
	store := commerce.NewPlanStoreWithOptions(commerce.PlanStoreOptions{PlansPath: path, BundleKey: "business_suite", DisplayEnv: "test"})
	if err := store.LoadAll(); err != nil {
		t.Fatal(err)
	}
	plans := commerce.NewPlanServiceWithOptions(commerce.PlanServiceOptions{PlanStore: store, DefaultBundle: "business_suite", DisplayEnv: "test"})
	var price *commerce.PlanOption
	for _, plan := range store.GetPlans() {
		if plan.Kind == shared.PlanKind_PLAN_KIND_CREDITS_TOPUP {
			price = plan
			break
		}
	}
	if price == nil {
		t.Fatal("catalog has no credit topup")
	}
	if _, err := s.db.Exec(`INSERT INTO checkout_sessions(session_id,customer_email,price_id,status,session_type,amount_cents,business_account_id) VALUES('checkout','actor@example.invalid',$1,'pending','credits_topup',$2,'account')`, price.StripePriceId, price.AmountCents); err != nil {
		t.Fatal(err)
	}
	topup := commerce.NewCreditTopupService(plans, commerce.NewCreditWalletService(s.db), nil)
	webhook := commerce.NewStripeWebhookService(commerce.StripeWebhookOptions{DB: s.db, Provider: journeyStripeCharge{}, PlanService: plans, Config: func() commerce.StripeWebhookConfig {
		return commerce.StripeWebhookConfig{WebhookSecret: "whsec_journey", HasWebhook: true}
	}, LoadCheckoutSession: commerce.NewStripeRepository(s.db).LoadCheckoutSession, ExtractAmount: func(obj map[string]interface{}, session *commerce.CheckoutSessionRecord) int64 {
		if amount, ok := obj["amount_total"].(float64); ok {
			return int64(amount)
		}
		return session.AmountCents.Int64
	}, HandleCreditTopup: topup.Apply})
	return &checkoutFixture{s: s, webhook: webhook, plan: price, plans: plans}
}
func (c *checkoutFixture) event(t *testing.T, id, kind, payment string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"id": id, "type": kind, "livemode": false, "data": map[string]any{"object": map[string]any{"id": "checkout", "customer_email": "actor@example.invalid", "payment_intent": "pi_fixture", "payment_status": payment, "amount_total": c.plan.AmountCents}}})
	if err != nil {
		t.Fatal(err)
	}
	timestamp := fmt.Sprint(time.Now().Unix())
	mac := hmac.New(sha256.New, []byte("whsec_journey"))
	mac.Write([]byte(timestamp + "."))
	mac.Write(body)
	signature := "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	if err := c.webhook.HandleWebhook(body, signature); err != nil {
		t.Fatal(err)
	}
}
func (c *checkoutFixture) lots(t *testing.T) int {
	t.Helper()
	var count int
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
func TestJ1_PaidCheckoutCreatesSpendableLot(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	if c.lots(t) != 1 {
		t.Fatal("paid checkout has no lot")
	}
	resp := c.s.request(t, c.s.key.Token, "spend", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("paid lot cannot spend: %d", resp.StatusCode)
	}
	assertCharged(t, c.s, 200)
}
func TestJ1_UnpaidSessionGrantsNothing(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "unpaid", "checkout.session.completed", "unpaid")
	if c.lots(t) != 0 {
		t.Fatal("unpaid checkout granted credit")
	}
	resp := c.s.request(t, c.s.key.Token, "spend", "hello")
	if resp.StatusCode != 402 {
		t.Fatalf("unpaid wallet status %d", resp.StatusCode)
	}
}
func TestJ1_DelayedPaymentGrantsOnSuccessOnly(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "pending", "checkout.session.completed", "unpaid")
	if c.lots(t) != 0 {
		t.Fatal("delayed payment credited early")
	}
	c.event(t, "success", "checkout.session.async_payment_succeeded", "paid")
	if c.lots(t) != 1 {
		t.Fatal("settled delayed payment lacks lot")
	}
}
func TestJ1_WebhookReplayIsSafe(t *testing.T) {
	c := newCheckout(t)
	c.event(t, "paid", "checkout.session.completed", "paid")
	c.event(t, "paid", "checkout.session.completed", "paid")
	c.event(t, "paid-again", "checkout.session.async_payment_succeeded", "paid")
	if c.lots(t) != 1 {
		t.Fatal("replay duplicated lots")
	}
	var transactions int
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_transactions`).Scan(&transactions); err != nil {
		t.Fatal(err)
	}
	if transactions != 1 {
		t.Fatalf("replay transactions %d", transactions)
	}
}

func TestJ1_SpendAcrossTwoLotsInOrder(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET remaining=100,granted=100,expires_at=NOW()+interval '1 day'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO credit_funding_lots(id,business_account_id,source_kind,provider,provider_ref,livemode,settlement_state,granted,remaining,status,created_at) VALUES(gen_random_uuid(),'account','paid','stripe','second',FALSE,'settled',1000,1000,'open',NOW())`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "two-lots", "hello")
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	var first, second int64
	if err := s.db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='fixture-payment'`).Scan(&first); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT remaining FROM credit_funding_lots WHERE provider_ref='second'`).Scan(&second); err != nil {
		t.Fatal(err)
	}
	if first != 0 || second != 900 {
		t.Fatalf("lot order balances=%d,%d", first, second)
	}
}
func TestJ1_ReceiptNamesFundingLots(t *testing.T) {
	s := newSpine(t)
	resp := s.request(t, s.key.Token, "lots-receipt", "hello")
	io.Copy(io.Discard, resp.Body)
	var provider, ref string
	var allocated, charged int64
	err := s.db.QueryRow(`SELECT l.provider,l.provider_ref,a.credits,e.credits FROM credit_lot_allocations a JOIN credit_funding_lots l ON l.id=a.lot_id JOIN business_account_usage_events e ON e.operation_id=a.operation_id WHERE e.operation_id=$1`, resp.Header.Get("X-Operation-ID")).Scan(&provider, &ref, &allocated, &charged)
	if err != nil {
		t.Fatal(err)
	}
	if provider != "stripe" || ref != "fixture-payment" || allocated != 200 || charged != 200 {
		t.Fatalf("funding receipt %s %s %d %d", provider, ref, allocated, charged)
	}
}
func TestJ1_PersonalWalletPathRetired(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(commerce.FinancialSchema()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`DELETE FROM credit_funding_lots;INSERT INTO credit_wallets(customer_email,balance_credits) VALUES('actor@example.invalid',1000000)`); err != nil {
		t.Fatal(err)
	}
	resp := s.request(t, s.key.Token, "legacy-wallet", "hello")
	if resp.StatusCode != 402 {
		t.Fatalf("legacy email balance spent: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}

func TestJ1_HeldOrTestLotCannotFundRealRoute(t *testing.T) {
	s := newSpine(t)
	live := true
	server := httptest.NewServer(&aidoor.Door{Keys: aidoor.KeyStore{DB: s.db}, Wallet: commerce.NewBusinessAccountUsageService(commerce.BusinessAccountUsageOptions{DB: s.db, Limits: commerce.NewLimitsService(s.db, "postgres", nil), Dialect: "postgres", LiveMode: &live}), GatewayURL: s.gateway})
	defer server.Close()
	original := s.server
	s.server = server
	defer func() { s.server = original }()
	resp := s.request(t, s.key.Token, "test-funds", "hello")
	if resp.StatusCode != 402 {
		t.Fatalf("test lot funded live call: %d", resp.StatusCode)
	}
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET livemode=TRUE,settlement_state='pending'`); err != nil {
		t.Fatal(err)
	}
	resp = s.request(t, s.key.Token, "held-funds", "hello")
	if resp.StatusCode != 402 {
		t.Fatalf("held lot funded live call: %d", resp.StatusCode)
	}
	assertCharged(t, s, 0)
}
func TestJ1_ParallelStreamsNeverGoNegative(t *testing.T) {
	s := newSpine(t)
	if _, err := s.db.Exec(`UPDATE credit_funding_lots SET remaining=1000,granted=1000`); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for i := 0; i < 8; i++ {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			resp := s.request(t, s.key.Token, fmt.Sprintf("parallel-%d", i), "hello")
			body, _ := io.ReadAll(resp.Body)
			if resp.StatusCode != 200 && resp.StatusCode != 402 {
				t.Errorf("parallel status %d body %s", resp.StatusCode, body)
			}
		}(i)
	}
	group.Wait()
	var remaining, charged, pending int64
	if err := s.db.QueryRow(`SELECT SUM(remaining) FROM credit_funding_lots`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COALESCE(SUM(credits),0) FROM business_account_usage_events`).Scan(&charged); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_reservations WHERE status='pending'`).Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if remaining < 0 || charged <= 0 || charged+remaining != 1000 || pending != 0 {
		t.Fatalf("parallel remaining=%d charged=%d pending=%d", remaining, charged, pending)
	}
}

type journeyStripeCharge struct{}

func (journeyStripeCharge) Request(ctx context.Context, method, path string, body io.Reader, contentType string) ([]byte, error) {
	if method != "GET" || path != "/charges/ch_fixture" {
		return nil, fmt.Errorf("unexpected fake Stripe request %s %s", method, path)
	}
	return []byte(`{"id":"ch_fixture","payment_intent":"pi_fixture","livemode":false}`), nil
}

// SM4 contracts stay with commerce. These adapters only drive real services and
// observe durable state; they do not implement an alternative payment machine.
type paymentSpecEvent struct {
	ID, Kind, Payment string
	InvalidSignature  bool
}
type paymentSpecRow struct {
	FromState, Event, To string
	Reject               bool
}
type paymentSpecTrace struct {
	ID, Initial, Prelude string
	Events, Expected     []string
}
type paymentContract struct {
	States, Events []string
	Transitions    []paymentSpecRow
	Traces         []struct {
		ID, Initial        string
		Incoming, Expected []string
	}
	Checkout struct {
		Creation                     struct{ Owner, From, To, Identity string }
		States, Terminal, Invariants []string
		Events                       []paymentSpecEvent
		Transitions                  []paymentSpecRow
		Traces                       []paymentSpecTrace
	}
	FraudWarning struct {
		States, Events []string
		Transitions    []paymentSpecRow
		Traces         []paymentSpecTrace
	}
	Refund struct {
		States, Events []string
		Transitions    []paymentSpecRow
		Traces         []paymentSpecTrace
	}
}

func loadPaymentContract(t *testing.T) paymentContract {
	t.Helper()
	data, err := os.ReadFile("../internal/commerce/funding_reversal.spec.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec paymentContract
	if err = json.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	return spec
}

// Only the Stripe edge is fake. Reject unexpected network intent instead of
// passing an unknown path to a live SDK or credential-bearing client.
type paymentSpecStripe struct {
	session string
	amount  int64
}

func (p paymentSpecStripe) Request(_ context.Context, method, path string, body io.Reader, _ string) ([]byte, error) {
	if method != "POST" || path != "/v1/checkout/sessions" {
		return nil, fmt.Errorf("unexpected Stripe request %s %s", method, path)
	}
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(string(data))
	if err != nil || values.Get("metadata[business_account_id]") != "account" || values.Get("mode") != "payment" {
		return nil, fmt.Errorf("checkout lost account/payment binding: %s", data)
	}
	return json.Marshal(map[string]any{"id": p.session, "status": "open", "url": "https://stripe.example.invalid/checkout/" + p.session, "customer_email": "actor@example.invalid", "amount_total": p.amount})
}
func (c *checkoutFixture) createSpecCheckout(t *testing.T, session string) {
	t.Helper()
	service := commerce.NewCheckoutService(commerce.CheckoutServiceOptions{Store: c.s.db, Plans: c.plans, Requester: paymentSpecStripe{session: session, amount: c.plan.AmountCents}})
	result, err := service.CreateForBusinessAccount(context.Background(), c.plan.StripePriceId, "/success", "/cancel", "actor@example.invalid", "account")
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionId != session {
		t.Fatalf("checkout returned %q want %q", result.SessionId, session)
	}
	c.assertSpecCheckout(t, session, "open")
}
func (c *checkoutFixture) paymentSpecEvent(t *testing.T, session, eventID string, event paymentSpecEvent) error {
	t.Helper()
	object := map[string]any{"id": session, "payment_intent": "pi_" + session, "customer_email": "actor@example.invalid", "payment_status": event.Payment, "amount_total": c.plan.AmountCents}
	if event.InvalidSignature {
		body, err := json.Marshal(map[string]any{"id": eventID, "type": event.Kind, "livemode": false, "data": map[string]any{"object": object}})
		if err != nil {
			t.Fatal(err)
		}
		return c.webhook.HandleWebhook(body, "t=0,v1=invalid")
	}
	return c.reversal(t, eventID, event.Kind, object)
}
func (c *checkoutFixture) assertSpecCheckout(t *testing.T, session, want string) {
	t.Helper()
	var state, account string
	err := c.s.db.QueryRow(`SELECT status,business_account_id FROM checkout_sessions WHERE session_id=$1`, session).Scan(&state, &account)
	if errors.Is(err, sql.ErrNoRows) {
		state = "absent"
	} else if err != nil {
		t.Fatal(err)
	} else if account != "account" {
		t.Fatal("checkout account binding changed")
	}
	if state != want {
		t.Fatalf("checkout %s status=%q want %q", session, state, want)
	}
	var lots, grants, conversions int
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM credit_funding_lots WHERE provider_ref=$1`, "pi_"+session).Scan(&lots); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM business_account_credit_transactions WHERE metadata->>'session_id'=$1`, session).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if err := c.s.db.QueryRow(`SELECT COUNT(*) FROM metrics_events WHERE event_id=$1`, "conversion:"+session).Scan(&conversions); err != nil {
		t.Fatal(err)
	}
	wantCount := 0
	if want == "complete" {
		wantCount = 1
	}
	if lots != wantCount || grants != wantCount || conversions != wantCount {
		t.Fatalf("checkout %s status=%s lots=%d grants=%d conversions=%d want %d", session, state, lots, grants, conversions, wantCount)
	}
	if lots == 1 {
		var bound bool
		if err := c.s.db.QueryRow(`SELECT business_account_id='account' AND NOT livemode AND settlement_state='settled' AND paid_amount_minor=$2 AND granted>0 AND remaining>=0 AND remaining<=granted FROM credit_funding_lots WHERE provider_ref=$1`, "pi_"+session, c.plan.AmountCents).Scan(&bound); err != nil {
			t.Fatal(err)
		}
		if !bound {
			t.Fatal("settled lot lost payment/mode/account/value invariant")
		}
	}
}

func TestJ17_PaymentCheckoutSpecMatrixAndTraces(t *testing.T) {
	spec := loadPaymentContract(t).Checkout
	if spec.Creation.Owner != "CheckoutService.CreateForBusinessAccount" || spec.Creation.From != "absent" || spec.Creation.To != "open" || spec.Creation.Identity == "" {
		t.Fatal("checkout factory contract drift")
	}
	if !slices.Equal(spec.States, []string{"absent", "open", "pending", "complete"}) || len(spec.Events) == 0 || len(spec.Traces) == 0 || len(spec.Invariants) == 0 {
		t.Fatal("incomplete checkout contract")
	}
	if !slices.Equal(spec.Terminal, []string{"complete"}) {
		t.Fatal("checkout terminal drift")
	}
	expectedEvents := []string{"completed_paid", "completed_unpaid", "completed_no_payment_required", "async_paid", "async_unpaid", "async_failed", "expired", "unknown", "bad_signature"}
	var declaredEvents []string
	for _, event := range spec.Events {
		declaredEvents = append(declaredEvents, event.ID)
	}
	if !slices.Equal(declaredEvents, expectedEvents) {
		t.Fatal("checkout event vocabulary drift")
	}
	events := map[string]paymentSpecEvent{}
	for _, event := range spec.Events {
		if _, exists := events[event.ID]; exists || event.ID == "" {
			t.Fatalf("duplicate/unnamed event %+v", event)
		}
		events[event.ID] = event
	}
	rows := map[[2]string]paymentSpecRow{}
	for _, row := range spec.Transitions {
		key := [2]string{row.FromState, row.Event}
		_, knownEvent := events[row.Event]
		if _, exists := rows[key]; exists || !knownEvent || !slices.Contains(spec.States, row.FromState) || !slices.Contains(spec.States, row.To) {
			t.Fatalf("duplicate/unknown checkout row %+v", row)
		}
		if slices.Contains(spec.Terminal, row.FromState) && row.To != row.FromState {
			t.Fatalf("terminal escape %+v", row)
		}
		rows[key] = row
	}
	c := newCheckout(t)
	index := 0
	for _, state := range spec.States {
		for _, event := range spec.Events {
			row, ok := rows[[2]string{state, event.ID}]
			if !ok {
				t.Fatalf("missing checkout pair %s/%s", state, event.ID)
			}
			index++
			t.Run(state+"/"+event.ID, func(t *testing.T) {
				session := fmt.Sprintf("sm4_matrix_%d", index)
				if state != "absent" {
					c.createSpecCheckout(t, session)
				}
				switch state {
				case "pending": // A legacy persisted session remains a supported precondition.
					if _, err := c.s.db.Exec(`UPDATE checkout_sessions SET status='pending' WHERE session_id=$1`, session); err != nil {
						t.Fatal(err)
					}
				case "complete":
					if err := c.paymentSpecEvent(t, session, "seed_"+session, events["completed_paid"]); err != nil {
						t.Fatal(err)
					}
				}
				c.assertSpecCheckout(t, session, state)
				err := c.paymentSpecEvent(t, session, "matrix_"+session, event)
				if (err != nil) != row.Reject {
					t.Fatalf("%s + %s error=%v want rejection=%t", state, event.ID, err, row.Reject)
				}
				c.assertSpecCheckout(t, session, row.To)
				t.Logf("SM4 real checkout: %s --%s--> %s reject=%t", state, event.ID, row.To, row.Reject)
			})
		}
	}
	for _, trace := range spec.Traces {
		t.Run(trace.ID, func(t *testing.T) {
			if trace.Prelude != "create" || trace.Initial != "open" || len(trace.Events) == 0 || len(trace.Events) != len(trace.Expected) {
				t.Fatal("invalid checkout trace")
			}
			session := "sm4_trace_" + trace.ID
			state := trace.Initial
			c.createSpecCheckout(t, session)
			c.assertSpecCheckout(t, session, state)
			for i, id := range trace.Events {
				{
					row, ok := rows[[2]string{state, id}]
					if !ok || row.Reject || row.To != trace.Expected[i] {
						t.Fatal("trace drifted from checkout matrix")
					}
					// Repeated events use the same provider identity, proving actual replay.
					if err := c.paymentSpecEvent(t, session, session+"_"+id, events[id]); err != nil {
						t.Fatal(err)
					}
				}
				c.assertSpecCheckout(t, session, trace.Expected[i])
				t.Logf("SM4 real checkout trace %s: %s --%s--> %s", trace.ID, state, id, trace.Expected[i])
				state = trace.Expected[i]
			}
		})
	}
	t.Logf("SM4 real checkout: %d complete matrix pairs, %d traces", index, len(spec.Traces))
}
