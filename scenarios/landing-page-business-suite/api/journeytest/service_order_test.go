package journeytest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	sharedpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/shared"
	"google.golang.org/protobuf/types/known/timestamppb"
	"landing-page-business-suite-api/internal/commerce"
)

type sandboxProvider struct {
	mu                                             sync.Mutex
	authorizations, captures, voids                int
	refuseCapture                                  bool
	captureKeys                                    []string
	voidErrorOnce                                  bool
	voidKeys                                       []string
	treasuryLedger                                 commerce.ServiceOrderLedgerWriter
	treasuryAdapter, treasuryBook, treasuryAccount string
	treasuryReceipts                               map[string]commerce.TreasuryX402Receipt
}

func (p *sandboxProvider) Authorize(_ context.Context, req commerce.ServiceOrderAuthorizationRequest) (commerce.ServiceOrderAuthorizationReceipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.authorizations++
	return commerce.ServiceOrderAuthorizationReceipt{EventID: "auth:" + req.OrderID, AuthorizationID: "sandbox-auth:" + req.OrderID, Sandbox: true, Approved: true}, nil
}

func (p *sandboxProvider) Capture(ctx context.Context, req commerce.ServiceOrderCaptureRequest) (commerce.ServiceOrderCaptureReceipt, error) {
	p.mu.Lock()
	p.captures++
	p.captureKeys = append(p.captureKeys, req.IdempotencyKey)
	refused := p.refuseCapture
	ledger := p.treasuryLedger
	adapter, book, account := p.treasuryAdapter, p.treasuryBook, p.treasuryAccount
	p.mu.Unlock()
	receipt := commerce.ServiceOrderCaptureReceipt{EventID: "capture-event:" + req.OrderID, CaptureID: "sandbox-capture:" + req.OrderID, Sandbox: true, Captured: !refused}
	if req.Door != commerce.ServiceOrderDoorAgentRun || refused {
		return receipt, nil
	}
	if ledger == nil {
		return commerce.ServiceOrderCaptureReceipt{}, errors.New("test Treasury settlement ledger is unavailable")
	}
	txID := "sandbox-tx-" + req.OrderID
	externalID := "treasury-x402-inflow:" + txID
	now := time.Now().UTC()
	event := &sharedpb.MoneyEvent{ExternalId: externalID, AdapterId: adapter, BookId: book, AccountId: account, AmountMinor: req.AmountMinor, Currency: req.Currency, OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now), Basis: sharedpb.Basis_BASIS_DERIVED, Description: "TEST ONLY Treasury x402 seller capture simulation; no real funds moved", Category: "test.service_order.treasury_x402"}
	ledgerReceipt, err := ledger.IngestSandboxCapture(ctx, event)
	if err != nil {
		return commerce.ServiceOrderCaptureReceipt{}, err
	}
	p.mu.Lock()
	if p.treasuryReceipts == nil {
		p.treasuryReceipts = make(map[string]commerce.TreasuryX402Receipt)
	}
	p.treasuryReceipts[externalID] = commerce.TreasuryX402Receipt{ExternalID: externalID, PostingID: ledgerReceipt.PostingID, ReceiptID: ledgerReceipt.ReceiptID, AcceptedAt: now, AmountMinor: req.AmountMinor, Currency: req.Currency, Basis: "derived"}
	p.mu.Unlock()
	receipt.CaptureID = txID
	receipt.TreasuryExternalID = externalID
	return receipt, nil
}

func (p *sandboxProvider) Void(_ context.Context, req commerce.ServiceOrderVoidRequest) (commerce.ServiceOrderVoidReceipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.voids++
	p.voidKeys = append(p.voidKeys, req.IdempotencyKey)
	if p.voidErrorOnce {
		p.voidErrorOnce = false
		return commerce.ServiceOrderVoidReceipt{}, errors.New("fixture lost void response")
	}
	return commerce.ServiceOrderVoidReceipt{EventID: "void:" + req.OrderID, VoidID: "sandbox-void:" + req.OrderID, Sandbox: true, Voided: true}, nil
}

func (p *sandboxProvider) ReadAccepted(_ context.Context, externalID string) (commerce.TreasuryX402Receipt, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	receipt, ok := p.treasuryReceipts[externalID]
	if !ok {
		return commerce.TreasuryX402Receipt{}, commerce.ErrTreasuryX402ReceiptNotFound
	}
	return receipt, nil
}

type fixedVerifier bool

func (v fixedVerifier) Verify(context.Context, commerce.ServiceOrder) (commerce.ServiceOrderVerification, error) {
	return commerce.ServiceOrderVerification{ReceiptID: "verified-company-market-scan", Passed: bool(v)}, nil
}

type orderBoundVerifier struct{}

func (orderBoundVerifier) Verify(_ context.Context, order commerce.ServiceOrder) (commerce.ServiceOrderVerification, error) {
	return commerce.ServiceOrderVerification{ReceiptID: "verified:" + order.ID, Passed: true}, nil
}

type offerFixture struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	X402PriceID string `json:"x402_price_id"`
	Sample      struct {
		SHA256 string `json:"sha256"`
	} `json:"sample"`
	ProposedPrice struct {
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
	} `json:"proposed_price"`
}

func loadCompanyMarketScan(t *testing.T) offerFixture {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "scenarios", "offer-desk", "api", "internal", "serviceoffer", "testdata", "company-market-scan-offer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var offer offerFixture
	if err := json.Unmarshal(data, &offer); err != nil {
		t.Fatal(err)
	}
	if offer.ID != "company-market-scan" || offer.Version != 1 || offer.X402PriceID != "company-market-scan-v1" || offer.ProposedPrice.AmountMinor != 4900 || offer.ProposedPrice.Currency != "USD" {
		t.Fatalf("unexpected Offer Desk test fixture: %+v", offer)
	}
	return offer
}

func newServiceOrderSandbox(t *testing.T, verifier commerce.ServiceOrderVerifier, provider *sandboxProvider) (*commerce.ServiceOrderSandbox, map[string]string) {
	t.Helper()
	c := newSpine(t)
	config := startFundingLedger(t)
	writer, err := commerce.NewMoneyLedgerTestFixtureWriter(config["url"], config["fixture_id"])
	if err != nil {
		t.Fatal(err)
	}
	target := commerce.PY1TestLedgerTarget{AdapterID: config["adapter"], BookID: config["book"], RevenueAccountID: config["service_order_revenue"]}
	if err := writer.ValidatePY1Target(target); err != nil {
		t.Fatalf("fixture did not prove its configured PY1 target: %v", err)
	}
	derivedTarget, err := writer.PY1Target()
	if err != nil || derivedTarget != target {
		t.Fatalf("fixture-derived PY1 target = %+v, err=%v; want %+v", derivedTarget, err, target)
	}
	wrongTarget := target
	wrongTarget.RevenueAccountID = "unrelated-account"
	if err := writer.ValidatePY1Target(wrongTarget); err == nil {
		t.Fatal("fixture accepted a PY1 revenue account outside its scope")
	}
	provider.treasuryLedger = writer
	provider.treasuryAdapter = config["adapter"]
	provider.treasuryBook = config["book"]
	provider.treasuryAccount = config["service_order_revenue"]
	service, err := commerce.NewServiceOrderSandbox(commerce.ServiceOrderSandboxOptions{
		DB: c.db, Provider: provider, Verifier: verifier, Ledger: writer, TreasuryReceipts: provider,
		LedgerTarget: commerce.ServiceOrderLedgerTarget{FixtureID: config["fixture_id"], AdapterID: config["adapter"], BookID: config["book"], RevenueAccountID: config["service_order_revenue"]},
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, config
}

func TestServiceOrderSandboxRetriesRefusalsAndLedgerReplay(t *testing.T) {
	ctx := context.Background()
	provider := &sandboxProvider{refuseCapture: true}
	service, config := newServiceOrderSandbox(t, fixedVerifier(true), provider)
	if _, err := commerce.NewMoneyLedgerTestFixtureWriter("http://example.invalid", config["fixture_id"]); err == nil {
		t.Fatal("Money Ledger writer accepted a non-loopback target")
	}
	if _, err := commerce.NewMoneyLedgerTestFixtureWriter(config["url"], config["fixture_id"]+"-mismatch"); err == nil {
		t.Fatal("Money Ledger writer accepted a mismatched fixture identity")
	}
	offer := loadCompanyMarketScan(t)
	request := commerce.BeginServiceOrderRequest{Quote: commerce.ServiceOrderQuote{OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor, Currency: offer.ProposedPrice.Currency}, Door: "browser", IdempotencyKey: "journey-retry-key", JobInput: json.RawMessage(`{"sector":"battery storage"}`)}
	first, err := service.Begin(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	retry, err := service.Begin(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != retry.ID || first.JobID != retry.JobID || provider.authorizations != 1 {
		t.Fatalf("idempotent begin changed durable identity or reauthorized: first=%+v retry=%+v auth=%d", first, retry, provider.authorizations)
	}
	var savedInput map[string]string
	if err := json.Unmarshal(first.JobInput, &savedInput); err != nil || savedInput["sector"] != "battery storage" {
		t.Fatalf("job input was not durably snapshotted: got %s error=%v", first.JobInput, err)
	}
	changed := request
	changed.Quote.AmountMinor++
	if _, err := service.Begin(ctx, changed); err != commerce.ErrServiceOrderIdempotencyConflict {
		t.Fatalf("changed idempotency payload error=%v", err)
	}
	if _, err := service.VerifyAndCapture(ctx, first.ID); err == nil || !strings.Contains(err.Error(), "capture receipt") {
		t.Fatalf("capture refusal error=%v", err)
	}
	pending, err := service.GetOrder(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != commerce.ServiceOrderCapturePending || provider.captures != 1 {
		t.Fatalf("capture refusal did not remain retryable: state=%s captures=%d", pending.State, provider.captures)
	}
	if _, err := service.VerifyAndCapture(ctx, first.ID); err == nil || !strings.Contains(err.Error(), "capture receipt") {
		t.Fatalf("repeat provider refusal error=%v", err)
	}
	pending, err = service.GetOrder(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != commerce.ServiceOrderCapturePending || provider.captures != 2 {
		t.Fatalf("capture refusal replay changed order or escaped provider idempotency: state=%s captures=%d", pending.State, provider.captures)
	}
	if provider.captureKeys[0] == "" || provider.captureKeys[0] != provider.captureKeys[1] {
		t.Fatalf("capture retries did not reuse provider idempotency identity: %v", provider.captureKeys)
	}
}

func TestServiceOrderSandboxFourDoorContractUsesOneOfferAndFourReceipts(t *testing.T) {
	ctx := context.Background()
	provider := &sandboxProvider{}
	service, config := newServiceOrderSandbox(t, orderBoundVerifier{}, provider)
	offer := loadCompanyMarketScan(t)
	doors := []string{
		commerce.ServiceOrderDoorBrowser,
		commerce.ServiceOrderDoorBrowserAgent,
		commerce.ServiceOrderDoorMCP,
		commerce.ServiceOrderDoorAgentRun,
	}
	orders := make(map[string]commerce.ServiceOrder, len(doors))
	jobs := make(map[string]struct{}, len(doors))
	for _, door := range doors {
		key := "four-door-contract:" + door
		jobInput := json.RawMessage(`{"sector":"battery storage","market":"US"}`)
		paymentSignature := ""
		if door == commerce.ServiceOrderDoorAgentRun {
			paymentSignature = "test-payment-signature:" + key
		}
		request := commerce.BeginServiceOrderRequest{
			Quote: commerce.ServiceOrderQuote{OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor, Currency: offer.ProposedPrice.Currency, X402PriceID: offer.X402PriceID},
			Door:  door, IdempotencyKey: key, JobInput: jobInput, PaymentSignature: paymentSignature,
		}
		first, err := service.Begin(ctx, request)
		if err != nil {
			t.Fatalf("%s begin: %v", door, err)
		}
		replay, err := service.Begin(ctx, request)
		if err != nil {
			t.Fatalf("%s exact replay: %v", door, err)
		}
		if replay.ID != first.ID || replay.JobID != first.JobID || replay.State != first.State {
			t.Fatalf("%s exact replay changed the durable order/job: first=%+v replay=%+v", door, first, replay)
		}
		changed := request
		changed.JobInput = json.RawMessage(`{"sector":"battery storage","market":"CA"}`)
		if _, err := service.Begin(ctx, changed); !errors.Is(err, commerce.ErrServiceOrderIdempotencyConflict) {
			t.Fatalf("%s accepted changed job input under the same idempotency key: %v", door, err)
		}
		if door == commerce.ServiceOrderDoorAgentRun {
			changedSignature := request
			changedSignature.PaymentSignature += ":changed"
			if _, err := service.Begin(ctx, changedSignature); !errors.Is(err, commerce.ErrServiceOrderIdempotencyConflict) {
				t.Fatalf("%s accepted changed x402 payment payload under the same idempotency key: %v", door, err)
			}
		}
		if first.OfferID != offer.ID || first.OfferVersion != offer.Version || first.AmountMinor != offer.ProposedPrice.AmountMinor || first.Currency != offer.ProposedPrice.Currency {
			t.Fatalf("%s did not use the immutable Offer Desk snapshot: %+v", door, first)
		}
		verified, err := service.VerifyAndCapture(ctx, first.ID)
		if err != nil {
			t.Fatalf("%s verify/capture: %v", door, err)
		}
		if verified.State != commerce.ServiceOrderLedgerPending || verified.VerificationID == "" || verified.CaptureID == "" {
			t.Fatalf("%s did not verify before capturing: %+v", door, verified)
		}
		reconciled, err := service.Reconcile(ctx, first.ID)
		if err != nil {
			t.Fatalf("%s ledger reconcile: %v", door, err)
		}
		if reconciled.State != commerce.ServiceOrderReconciled || reconciled.LedgerPostingID == "" {
			t.Fatalf("%s has no durable ledger receipt: %+v", door, reconciled)
		}
		captureReplay, err := service.VerifyAndCapture(ctx, first.ID)
		if err != nil {
			t.Fatalf("%s capture replay: %v", door, err)
		}
		ledgerReplay, err := service.Reconcile(ctx, first.ID)
		if err != nil {
			t.Fatalf("%s ledger replay: %v", door, err)
		}
		if captureReplay.CaptureID != reconciled.CaptureID || ledgerReplay.LedgerPostingID != reconciled.LedgerPostingID {
			t.Fatalf("%s replay changed capture or accepted Money Ledger receipt: initial=%+v capture-replay=%+v ledger-replay=%+v", door, reconciled, captureReplay, ledgerReplay)
		}
		if _, exists := jobs[reconciled.JobID]; exists {
			t.Fatalf("door %s reused job id %s", door, reconciled.JobID)
		}
		jobs[reconciled.JobID] = struct{}{}
		orders[door] = *reconciled
	}
	if provider.authorizations != len(doors) || provider.captures != len(doors) {
		t.Fatalf("door count did not match provider receipts: authorizations=%d captures=%d", provider.authorizations, provider.captures)
	}
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != len(doors) {
		t.Fatalf("got %d seller-side postings for %d successful test doors", len(postings.Msg.Postings), len(doors))
	}
	postingIDs := make(map[string]struct{}, len(doors))
	receiptIDs := make(map[string]struct{}, len(doors))
	postingCategories := make(map[string]string, len(postings.Msg.Postings))
	for _, posting := range postings.Msg.Postings {
		postingCategories[posting.Id] = posting.Event.Category
	}
	for door, order := range orders {
		if _, exists := postingIDs[order.LedgerPostingID]; exists {
			t.Fatalf("door %s reused ledger posting id %s", door, order.LedgerPostingID)
		}
		postingIDs[order.LedgerPostingID] = struct{}{}
		if order.LedgerReceiptID == "" {
			t.Fatalf("door %s has no accepted Money Ledger receipt id", door)
		}
		if _, exists := receiptIDs[order.LedgerReceiptID]; exists {
			t.Fatalf("door %s reused Money Ledger receipt id %s", door, order.LedgerReceiptID)
		}
		receiptIDs[order.LedgerReceiptID] = struct{}{}
		wantCategory := "test.service_order.sandbox"
		if door == commerce.ServiceOrderDoorAgentRun {
			wantCategory = "test.service_order.treasury_x402"
		}
		if got := postingCategories[order.LedgerPostingID]; got != wantCategory {
			t.Fatalf("door %s seller posting category = %q, want %q", door, got, wantCategory)
		}
	}
}

func TestServiceOrderVerificationFailureVoidsWithoutCapture(t *testing.T) {
	provider := &sandboxProvider{voidErrorOnce: true}
	service, _ := newServiceOrderSandbox(t, fixedVerifier(false), provider)
	offer := loadCompanyMarketScan(t)
	order, err := service.Begin(context.Background(), commerce.BeginServiceOrderRequest{Quote: commerce.ServiceOrderQuote{OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor, Currency: offer.ProposedPrice.Currency}, Door: "browser", IdempotencyKey: "verification-failure", JobInput: json.RawMessage(`{"sector":"battery storage"}`)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.VerifyAndCapture(context.Background(), order.ID); err == nil || !strings.Contains(err.Error(), "void is pending") {
		t.Fatalf("expected retryable void outcome, got %v", err)
	}
	pending, err := service.GetOrder(context.Background(), order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State != commerce.ServiceOrderVerifyFailed {
		t.Fatalf("failed void did not remain pending: %+v", pending)
	}
	result, err := service.VerifyAndCapture(context.Background(), order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != commerce.ServiceOrderVoided || provider.voids != 2 || provider.captures != 0 {
		t.Fatalf("verification failure did not retry release authorization: order=%+v voids=%d captures=%d", result, provider.voids, provider.captures)
	}
	if provider.voidKeys[0] == "" || provider.voidKeys[0] != provider.voidKeys[1] {
		t.Fatalf("void retries did not reuse provider idempotency identity: %v", provider.voidKeys)
	}
}

func TestServiceOrderSandboxBrowserJourney(t *testing.T) {
	ctx := context.Background()
	provider := &sandboxProvider{}
	service, config := newServiceOrderSandbox(t, fixedVerifier(true), provider)
	offer := loadCompanyMarketScan(t)
	var orderMu sync.Mutex
	var orderAttempts int
	var orderKeys, orderIDs []string
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Company Market Scan · sandbox test</title><style>body{font:16px system-ui,sans-serif;max-width:42rem;margin:auto;padding:1rem}label,input{display:block;width:100%;box-sizing:border-box}input{min-height:44px;margin:.4rem 0}button{display:block;width:100%;min-height:44px;margin:.5rem 0}pre{white-space:pre-wrap;overflow-wrap:anywhere;max-width:100%}</style></head><body><h1>Company Market Scan</h1><p id="mode">Sandbox simulation only — no real money moved.</p><label>Sector <input id="sector" value="battery storage"></label><button id="buy">Create sandbox order</button><button id="verify" disabled>Verify and capture</button><button id="ledger" disabled>Reconcile test ledger</button><pre id="result">waiting</pre><script>
const out=document.querySelector('#result'), buy=document.querySelector('#buy'); let order, submitting=false; const orderKey=crypto.randomUUID();
buy.onclick=async()=>{if(submitting||order)return;submitting=true;buy.disabled=true;try{let r=await fetch('/api/order',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({sector:document.querySelector('#sector').value,key:orderKey})});if(!r.ok)throw new Error('response lost; retry same order');let j=await r.json();order=j;out.textContent=JSON.stringify(j);document.querySelector('#verify').disabled=false}catch(e){out.textContent=e.message}finally{submitting=false;buy.disabled=Boolean(order)}};
document.querySelector('#verify').onclick=async()=>{let r=await fetch('/api/verify',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({id:order.order_id})});order=await r.json();out.textContent=JSON.stringify(order);document.querySelector('#ledger').disabled=false};
document.querySelector('#ledger').onclick=async()=>{let r=await fetch('/api/reconcile',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({id:order.order_id})});order=await r.json();out.textContent=JSON.stringify(order)};
</script></body></html>`)
	})
	decodeID := func(r *http.Request) string {
		var body struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		return body.ID
	}
	mux.HandleFunc("/api/order", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Sector string `json:"sector"`
			Key    string `json:"key"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil || strings.TrimSpace(body.Sector) == "" {
			http.Error(w, "sector required", 400)
			return
		}
		jobInput, _ := json.Marshal(map[string]string{"sector": body.Sector})
		order, err := service.Begin(ctx, commerce.BeginServiceOrderRequest{Quote: commerce.ServiceOrderQuote{OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor, Currency: offer.ProposedPrice.Currency}, Door: "browser", IdempotencyKey: body.Key, JobInput: jobInput})
		if err != nil {
			writeJourneyOrder(w, order, err)
			return
		}
		orderMu.Lock()
		orderAttempts++
		orderKeys = append(orderKeys, body.Key)
		orderIDs = append(orderIDs, order.ID)
		attempt := orderAttempts
		orderMu.Unlock()
		if attempt == 1 {
			time.Sleep(300 * time.Millisecond)
			http.Error(w, "simulated lost sandbox order response", http.StatusServiceUnavailable)
			return
		}
		writeJourneyOrder(w, order, nil)
	})
	for _, route := range []struct {
		path   string
		action func(context.Context, string) (*commerce.ServiceOrder, error)
	}{
		{"/api/verify", service.VerifyAndCapture}, {"/api/reconcile", service.Reconcile},
	} {
		route := route
		mux.HandleFunc(route.path, func(w http.ResponseWriter, r *http.Request) {
			order, err := route.action(r.Context(), decodeID(r))
			writeJourneyOrder(w, order, err)
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()
	script, err := os.ReadFile(filepath.Join("service_order_browser.py"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "service_order_browser.py")
	if err := os.WriteFile(path, script, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", path, server.URL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("sandbox browser journey failed: %v\n%s", err, output)
	}
	orderMu.Lock()
	if orderAttempts != 2 || len(orderKeys) != 2 || orderKeys[0] == "" || orderKeys[0] != orderKeys[1] || orderIDs[0] != orderIDs[1] {
		orderMu.Unlock()
		t.Fatalf("browser retry did not recover the same order with its original key: attempts=%d keys=%v ids=%v", orderAttempts, orderKeys, orderIDs)
	}
	orderMu.Unlock()
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 1 || postings.Msg.Postings[0].Event.Category != "test.service_order.sandbox" {
		t.Fatalf("browser flow did not create exactly one typed Money Ledger posting: %+v", postings.Msg.Postings)
	}
	if postings.Msg.Postings[0].Event.Description != "TEST ONLY sandbox capture simulation; no real funds moved" {
		t.Fatalf("browser flow ledger event is not labeled test-only: %+v", postings.Msg.Postings[0].Event)
	}
	writer, err := commerce.NewMoneyLedgerTestFixtureWriter(config["url"], config["fixture_id"])
	if err != nil {
		t.Fatal(err)
	}
	replay, err := writer.IngestSandboxCapture(ctx, postings.Msg.Postings[0].Event)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Duplicate || replay.PostingID != postings.Msg.Postings[0].Id {
		t.Fatalf("Money Ledger replay did not preserve the original posting: %+v", replay)
	}
	changedEvent := *postings.Msg.Postings[0].Event
	changedEvent.AmountMinor++
	if _, err := writer.IngestSandboxCapture(ctx, &changedEvent); err == nil || !strings.Contains(err.Error(), "same adapter_id and external_id already exists with a different payload") {
		t.Fatalf("Money Ledger accepted a changed payload under the captured external ID: %v", err)
	}
}

func writeJourneyOrder(w http.ResponseWriter, order *commerce.ServiceOrder, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		http.Error(w, err.Error(), 409)
		return
	}
	_ = json.NewEncoder(w).Encode(order)
}
