package journeytest

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
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

type serviceOrderJourneyToken string

func (t serviceOrderJourneyToken) Token(context.Context) (string, error) { return string(t), nil }

type x402JourneyAdmission struct {
	priceID, payer, network, resourceID, orderID, jobID, paymentDigest string
	settlement                                                         *commerce.TreasuryX402Settlement
	receipt                                                            *commerce.TreasuryX402Receipt
	cancelled                                                          bool
}

type x402JourneyTreasury struct {
	mu                                                          sync.Mutex
	writer                                                      commerce.ServiceOrderLedgerWriter
	adapter, book, account                                      string
	admissions                                                  map[string]*x402JourneyAdmission
	byDigest                                                    map[string]string
	receiptVisible                                              bool
	cancelErrorOnce                                             bool
	verifyCalls, settleCalls, cancelCalls, settlementExecutions int
}

func newX402JourneyTreasury(writer commerce.ServiceOrderLedgerWriter, config map[string]string) *x402JourneyTreasury {
	return &x402JourneyTreasury{writer: writer, adapter: config["adapter"], book: config["book"], account: config["service_order_revenue"], admissions: make(map[string]*x402JourneyAdmission), byDigest: make(map[string]string)}
}

func (h *x402JourneyTreasury) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method == http.MethodPost && r.Header.Get("Authorization") != "Bearer x402-order-token" {
		http.Error(w, "service token rejected", http.StatusUnauthorized)
		return
	}
	if r.Method == http.MethodGet && r.Header.Get("Authorization") != "Bearer x402-receipt-token" {
		http.Error(w, "receipt token rejected", http.StatusUnauthorized)
		return
	}
	switch {
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/x402/authorizations/verify":
		var input commerce.TreasuryX402VerifyRequest
		if json.NewDecoder(r.Body).Decode(&input) != nil || input.PriceID == "" || input.ResourceID == "" || input.OrderID == "" || input.JobID == "" || input.PaymentSignature == "" {
			http.Error(w, "invalid verify request", http.StatusBadRequest)
			return
		}
		digest := fmt.Sprintf("%x", sha256.Sum256([]byte(input.PaymentSignature)))
		h.mu.Lock()
		h.verifyCalls++
		admissionID := h.byDigest[digest]
		if admissionID != "" {
			old := h.admissions[admissionID]
			if old.priceID != input.PriceID || old.resourceID != input.ResourceID || old.orderID != input.OrderID || old.jobID != input.JobID {
				h.mu.Unlock()
				http.Error(w, "payment digest is bound to different order terms", http.StatusConflict)
				return
			}
		} else {
			admissionID = "admission:" + digest[:16]
			h.byDigest[digest] = admissionID
			h.admissions[admissionID] = &x402JourneyAdmission{priceID: input.PriceID, payer: "0xtest-buyer", network: "eip155:8453", resourceID: input.ResourceID, orderID: input.OrderID, jobID: input.JobID, paymentDigest: digest}
		}
		h.mu.Unlock()
		writeX402JourneyJSON(w, map[string]any{"admission_id": admissionID, "status": "authorized", "mode": "local-fork", "payer": "0xtest-buyer", "network": "eip155:8453", "price_id": input.PriceID, "resource_id": input.ResourceID, "order_id": input.OrderID, "job_id": input.JobID})
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/x402/authorizations/") && strings.HasSuffix(r.URL.Path, "/settle"):
		var input commerce.TreasuryX402SettleRequest
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			http.Error(w, "invalid settlement request", http.StatusBadRequest)
			return
		}
		admissionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/x402/authorizations/"), "/settle")
		h.mu.Lock()
		h.settleCalls++
		admission := h.admissions[admissionID]
		if admission == nil || admission.cancelled || admission.orderID != input.OrderID || admission.jobID != input.JobID || admission.resourceID != input.ResourceID {
			h.mu.Unlock()
			http.Error(w, "admission binding conflict", http.StatusConflict)
			return
		}
		if admission.settlement != nil {
			response := *admission.settlement
			h.mu.Unlock()
			writeX402JourneyJSON(w, response)
			return
		}
		h.settlementExecutions++
		h.mu.Unlock()
		txID := "tx:" + admissionID
		externalID := "treasury-x402-inflow:" + txID
		now := time.Now().UTC()
		event := &sharedpb.MoneyEvent{ExternalId: externalID, AdapterId: h.adapter, BookId: h.book, AccountId: h.account, AmountMinor: 4900, Currency: "USD", OccurredAt: timestamppb.New(now), FetchedAt: timestamppb.New(now), Basis: sharedpb.Basis_BASIS_DERIVED, Description: "TEST ONLY Treasury local-fork seller capture simulation; no real funds moved", Category: "test.service_order.treasury_x402"}
		ledgerReceipt, err := h.writer.IngestSandboxCapture(r.Context(), event)
		if err != nil {
			http.Error(w, "isolated test ledger refused settlement", http.StatusServiceUnavailable)
			return
		}
		settlement := commerce.TreasuryX402Settlement{AdmissionID: admissionID, PriceID: admission.priceID, Status: "settled", Payer: admission.payer, Mode: "local-fork", Transaction: txID, Network: admission.network, JobID: admission.jobID, ResourceID: admission.resourceID, OrderID: admission.orderID, PostingID: ledgerReceipt.PostingID, ReceiptID: ledgerReceipt.ReceiptID}
		receipt := commerce.TreasuryX402Receipt{ExternalID: externalID, PostingID: ledgerReceipt.PostingID, ReceiptID: ledgerReceipt.ReceiptID, AcceptedAt: now, AmountMinor: 4900, Currency: "USD", Basis: "derived"}
		h.mu.Lock()
		admission.settlement = &settlement
		admission.receipt = &receipt
		h.mu.Unlock()
		writeX402JourneyJSON(w, settlement)
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/v1/x402/authorizations/") && strings.HasSuffix(r.URL.Path, "/cancel"):
		var input commerce.TreasuryX402CancelRequest
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			http.Error(w, "invalid cancellation request", http.StatusBadRequest)
			return
		}
		admissionID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/x402/authorizations/"), "/cancel")
		h.mu.Lock()
		admission := h.admissions[admissionID]
		if admission == nil || admission.settlement != nil || admission.orderID != input.OrderID || admission.jobID != input.JobID || admission.resourceID != input.ResourceID {
			h.mu.Unlock()
			http.Error(w, "admission cannot be cancelled under this binding", http.StatusConflict)
			return
		}
		h.cancelCalls++
		if h.cancelErrorOnce {
			h.cancelErrorOnce = false
			h.mu.Unlock()
			http.Error(w, "protected authorization cleanup is temporarily unavailable", http.StatusServiceUnavailable)
			return
		}
		admission.cancelled = true
		response := commerce.TreasuryX402Cancellation{AdmissionID: admissionID, PriceID: admission.priceID, Status: "cancelled", Payer: admission.payer, Network: admission.network, Mode: "local-fork", JobID: admission.jobID, ResourceID: admission.resourceID, OrderID: admission.orderID}
		h.mu.Unlock()
		writeX402JourneyJSON(w, response)
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/x402/receipts":
		externalID := r.URL.Query().Get("external_id")
		h.mu.Lock()
		defer h.mu.Unlock()
		if !h.receiptVisible {
			http.NotFound(w, r)
			return
		}
		for _, admission := range h.admissions {
			if admission.receipt != nil && admission.receipt.ExternalID == externalID {
				writeX402JourneyJSON(w, admission.receipt)
				return
			}
		}
		http.NotFound(w, r)
	default:
		http.NotFound(w, r)
	}
}

func writeX402JourneyJSON(w http.ResponseWriter, value any) {
	_ = json.NewEncoder(w).Encode(value)
}

type serviceOrderJourneyVerifier struct {
	mu    sync.Mutex
	fail  map[string]bool
	calls int
}

func (v *serviceOrderJourneyVerifier) Verify(_ context.Context, order commerce.ServiceOrder) (commerce.ServiceOrderVerification, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.calls++
	return commerce.ServiceOrderVerification{ReceiptID: "verified:" + order.ID, Passed: !v.fail[order.ID]}, nil
}

func TestServiceOrderTreasuryX402LinksOneAcceptedSellerPostingAfterVerification(t *testing.T) {
	ctx := context.Background()
	spine := newSpine(t)
	config := startFundingLedger(t)
	writer, err := commerce.NewMoneyLedgerTestFixtureWriter(config["url"], config["fixture_id"])
	if err != nil {
		t.Fatal(err)
	}
	treasury := newX402JourneyTreasury(writer, config)
	server := httptest.NewServer(treasury)
	defer server.Close()
	orderClient, err := commerce.NewTreasuryX402HTTPClient(server.URL, serviceOrderJourneyToken("x402-order-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	receiptClient, err := commerce.NewTreasuryX402ReceiptReader(server.URL, serviceOrderJourneyToken("x402-receipt-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	provider, err := commerce.NewTreasuryX402ServiceOrderProvider(orderClient)
	if err != nil {
		t.Fatal(err)
	}
	verifier := &serviceOrderJourneyVerifier{fail: make(map[string]bool)}
	service, err := commerce.NewServiceOrderSandbox(commerce.ServiceOrderSandboxOptions{
		DB: spine.db, Provider: provider, Verifier: verifier, Ledger: writer, TreasuryReceipts: receiptClient,
		LedgerTarget: commerce.ServiceOrderLedgerTarget{FixtureID: config["fixture_id"], AdapterID: config["adapter"], BookID: config["book"], RevenueAccountID: config["service_order_revenue"]},
	})
	if err != nil {
		t.Fatal(err)
	}
	offer := loadCompanyMarketScan(t)
	request := commerce.BeginServiceOrderRequest{
		Quote: commerce.ServiceOrderQuote{OfferID: offer.ID, OfferVersion: offer.Version, AmountMinor: offer.ProposedPrice.AmountMinor, Currency: offer.ProposedPrice.Currency, X402PriceID: offer.X402PriceID},
		Door:  commerce.ServiceOrderDoorAgentRun, AgentManagerRunID: "1b2be756-60d8-4865-9c28-0e1c13a75485", IdempotencyKey: "treasury-x402-order-one", PaymentSignature: "test signed payment one",
		JobInput: json.RawMessage(`{"sector":"battery storage"}`),
	}
	order, err := service.Begin(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	var persistedRunID, persistedPriceID, persistedPayer, persistedNetwork string
	if err := spine.db.QueryRow(`SELECT agent_manager_run_id,x402_price_id,x402_payer,x402_network FROM service_orders WHERE id=$1`, order.ID).Scan(&persistedRunID, &persistedPriceID, &persistedPayer, &persistedNetwork); err != nil || persistedRunID != request.AgentManagerRunID || persistedPriceID != offer.X402PriceID || persistedPayer != "0xtest-buyer" || persistedNetwork != "eip155:8453" {
		t.Fatalf("durable run and x402 binding = (%q,%q,%q,%q), err=%v", persistedRunID, persistedPriceID, persistedPayer, persistedNetwork, err)
	}
	replay, err := service.Begin(ctx, request)
	if err != nil || replay.ID != order.ID || replay.JobID != order.JobID || treasury.verifyCalls != 1 {
		t.Fatalf("authorization replay changed order/job or repeated verify: order=%+v replay=%+v verify_calls=%d err=%v", order, replay, treasury.verifyCalls, err)
	}
	changedSignature := request
	changedSignature.PaymentSignature += "-changed"
	if _, err := service.Begin(ctx, changedSignature); !errors.Is(err, commerce.ErrServiceOrderIdempotencyConflict) {
		t.Fatalf("changed payment signature did not conflict: %v", err)
	}
	changedRun := request
	changedRun.AgentManagerRunID = "c49f8ff4-26e2-46cc-856d-3f799668598c"
	if _, err := service.Begin(ctx, changedRun); !errors.Is(err, commerce.ErrServiceOrderIdempotencyConflict) {
		t.Fatalf("changed Agent Manager run id did not conflict: %v", err)
	}
	verified, err := service.VerifyAndCapture(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if verified.State != commerce.ServiceOrderLedgerPending || verified.TreasuryExternalID == "" || treasury.settlementExecutions != 1 {
		t.Fatalf("verification did not precede exactly one Treasury settlement: order=%+v settlements=%d", verified, treasury.settlementExecutions)
	}
	settlementReplay, err := orderClient.SettleAuthorization(ctx, commerce.TreasuryX402SettleRequest{AdmissionID: order.AuthorizationID, PriceID: "company-market-scan-v1", Payer: "0xtest-buyer", Network: "eip155:8453", ResourceID: "company-market-scan:v1", OrderID: order.ID, JobID: order.JobID})
	if err != nil || settlementReplay.Transaction == "" || treasury.settlementExecutions != 1 {
		t.Fatalf("Treasury settlement replay charged twice: response=%+v executions=%d err=%v", settlementReplay, treasury.settlementExecutions, err)
	}
	if _, err := service.Reconcile(ctx, order.ID); !errors.Is(err, commerce.ErrTreasuryX402ReceiptNotFound) {
		t.Fatalf("missing accepted Treasury receipt error=%v", err)
	}
	pending, err := service.GetOrder(ctx, order.ID)
	if err != nil || pending.State != commerce.ServiceOrderLedgerPending || pending.LedgerPostingID != "" {
		t.Fatalf("missing accepted receipt was incorrectly linked: order=%+v err=%v", pending, err)
	}
	treasury.mu.Lock()
	treasury.receiptVisible = true
	treasury.mu.Unlock()
	reconciled, err := service.Reconcile(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.State != commerce.ServiceOrderReconciled || reconciled.LedgerPostingID == "" || reconciled.LedgerReceiptID == "" || reconciled.TreasuryExternalID == "" {
		t.Fatalf("accepted Treasury receipt was not linked to LPBS job: %+v", reconciled)
	}
	if _, err := service.Reconcile(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.VerifyAndCapture(ctx, order.ID); err != nil {
		t.Fatal(err)
	}
	if treasury.settlementExecutions != 1 {
		t.Fatalf("terminal LPBS replay called Treasury settlement %d times", treasury.settlementExecutions)
	}
	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 1 || postings.Msg.Postings[0].Id != reconciled.LedgerPostingID {
		t.Fatalf("Treasury seller-side fixture has duplicate or unlinked postings: postings=%+v", postings.Msg.Postings)
	}
	var outboxCount int
	if err := spine.db.QueryRowContext(ctx, `SELECT count(*) FROM service_order_ledger_outbox WHERE order_id=$1`, order.ID).Scan(&outboxCount); err != nil || outboxCount != 0 {
		t.Fatalf("LPBS wrote a duplicate x402 seller-side outbox row: count=%d err=%v", outboxCount, err)
	}

	failed, err := service.Begin(ctx, commerce.BeginServiceOrderRequest{
		Quote: request.Quote, Door: commerce.ServiceOrderDoorAgentRun, AgentManagerRunID: "a57e74dc-9de4-48f3-b5e2-968dc7a448b9", IdempotencyKey: "treasury-x402-order-verify-fails",
		PaymentSignature: "test signed payment that will not settle", JobInput: json.RawMessage(`{"sector":"solar storage"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	verifier.mu.Lock()
	verifier.fail[failed.ID] = true
	verifier.mu.Unlock()
	treasury.mu.Lock()
	treasury.cancelErrorOnce = true
	treasury.mu.Unlock()
	if _, err := service.VerifyAndCapture(ctx, failed.ID); err == nil {
		t.Fatal("failed delivery must remain pending when Treasury cannot clear its protected payload")
	}
	pendingCancel, err := service.GetOrder(ctx, failed.ID)
	if err != nil || pendingCancel.State != commerce.ServiceOrderVerifyFailed {
		t.Fatalf("failed cancellation was incorrectly recorded as complete: order=%+v err=%v", pendingCancel, err)
	}
	failedOrder, err := service.VerifyAndCapture(ctx, failed.ID)
	if err != nil || failedOrder.State != commerce.ServiceOrderVoided {
		t.Fatalf("failed delivery did not release the un-settled authorization: order=%+v err=%v", failedOrder, err)
	}
	if treasury.settlementExecutions != 1 || treasury.cancelCalls != 2 {
		t.Fatalf("failed verification did not cancel only its un-settled authorization: settlements=%d cancellations=%d", treasury.settlementExecutions, treasury.cancelCalls)
	}
	postings, err = ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 1 {
		t.Fatalf("failed delivery created seller-side posting: count=%d", len(postings.Msg.Postings))
	}
}
