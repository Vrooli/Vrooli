package journeytest

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	ledgerpb "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger"
	ledgerconnect "github.com/vrooli/vrooli/packages/proto/gen/go/money-ledger/v1/ledger/ledger_v1connect"
	"landing-page-business-suite-api/handlers/serviceorders"
	"landing-page-business-suite-api/internal/commerce"
)

func TestServiceOrderHTTPFourDoorJourneyLinksOneOfferToFourAcceptedLedgerReceipts(t *testing.T) {
	ctx := context.Background()
	spine := newSpine(t)
	config := startFundingLedger(t)
	ledgerAuth := &serviceOrderLedgerAuthTransport{base: http.DefaultTransport}
	lpbsLedgerWriter, err := commerce.NewMoneyLedgerTestServiceWriter(config["url"], serviceOrderJourneyToken("lpbs-run-only-money-ledger-write"), &http.Client{Transport: ledgerAuth})
	if err != nil {
		t.Fatal(err)
	}
	treasuryLedgerWriter, err := commerce.NewMoneyLedgerTestFixtureWriter(config["url"], config["fixture_id"])
	if err != nil {
		t.Fatal(err)
	}

	// This isolated Offer Desk HTTP projection is seeded from the canonical
	// service-offer fixture. LPBS resolves every quote and order from this same
	// source; callers cannot supply or override price terms.
	offer := loadCompanyMarketScan(t)
	offerDesk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"offer": offer})
	}))
	defer offerDesk.Close()
	offers, err := commerce.NewOfferDeskServiceOrderSource(offerDesk.URL, offerDesk.Client())
	if err != nil {
		t.Fatal(err)
	}

	treasury := newX402JourneyTreasury(treasuryLedgerWriter, config)
	treasuryServer := httptest.NewServer(treasury)
	defer treasuryServer.Close()
	treasuryOrderClient, err := commerce.NewTreasuryX402HTTPClient(treasuryServer.URL, serviceOrderJourneyToken("x402-order-token"), treasuryServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	treasuryReceiptClient, err := commerce.NewTreasuryX402ReceiptReader(treasuryServer.URL, serviceOrderJourneyToken("x402-receipt-token"), treasuryServer.Client())
	if err != nil {
		t.Fatal(err)
	}
	treasuryProvider, err := commerce.NewTreasuryX402ServiceOrderProvider(treasuryOrderClient)
	if err != nil {
		t.Fatal(err)
	}
	provider, err := commerce.NewServiceOrderProviderRouter(commerce.NewLocalTestServiceOrderProvider(), treasuryProvider)
	if err != nil {
		t.Fatal(err)
	}
	service, err := commerce.NewServiceOrderSandbox(commerce.ServiceOrderSandboxOptions{
		DB:               spine.db,
		Provider:         provider,
		Verifier:         &serviceOrderJourneyVerifier{fail: make(map[string]bool)},
		Ledger:           lpbsLedgerWriter,
		TreasuryReceipts: treasuryReceiptClient,
		LedgerTarget: commerce.ServiceOrderLedgerTarget{
			FixtureID: config["fixture_id"], AdapterID: config["adapter"], BookID: config["book"], RevenueAccountID: config["service_order_revenue"],
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := serviceorders.NewHandler(serviceorders.Dependencies{Runtime: service, Offers: offers, TestMode: true})
	if err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	handler.RegisterRoutes(router)
	api := httptest.NewServer(router)
	defer api.Close()

	quoteResponse, quoteBody := serviceOrderHTTPCall(t, api.Client(), http.MethodGet,
		api.URL+"/api/v1/service-orders/offer?offer_id="+offer.ID+"&offer_version=1", "", http.StatusOK)
	if got := quoteResponse.Header.Get("Cache-Control"); got != "no-store" {
		t.Fatalf("canonical quote Cache-Control = %q, want no-store", got)
	}
	var quote map[string]any
	if err := json.Unmarshal(quoteBody, &quote); err != nil || quote["offer_id"] != offer.ID || quote["offer_version"] != float64(offer.Version) || quote["amount_minor"] != float64(offer.ProposedPrice.AmountMinor) || quote["currency"] != offer.ProposedPrice.Currency {
		t.Fatalf("canonical HTTP quote does not match Offer Desk fixture: body=%s err=%v", quoteBody, err)
	}

	doors := []struct {
		name      string
		signature string
		runID     string
	}{
		{name: commerce.ServiceOrderDoorBrowser},
		{name: commerce.ServiceOrderDoorBrowserAgent},
		{name: commerce.ServiceOrderDoorMCP},
		{name: commerce.ServiceOrderDoorAgentRun, signature: "test-only signed payment for Agent Manager door", runID: "6cf3e5c6-7d0f-48e6-bce0-a2f8ae7c7451"},
	}
	orders := make(map[string]commerce.ServiceOrder, len(doors))
	for _, door := range doors {
		t.Run(door.name, func(t *testing.T) {
			key := "http-four-door:" + door.name
			jobInput := map[string]string{"sector": "battery storage", "market": "US"}
			beginBody := map[string]any{
				"offer_id": offer.ID, "offer_version": offer.Version, "door": door.name,
				"idempotency_key": key, "job_input": jobInput,
			}
			if door.runID != "" {
				beginBody["agent_manager_run_id"] = door.runID
			}
			if door.signature != "" {
				beginBody["payment_signature"] = door.signature
			}
			encoded := marshalServiceOrderJSON(t, beginBody)
			_, firstBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders", encoded, http.StatusOK)
			first := decodeServiceOrderHTTPResponse(t, firstBody)
			_, replayBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders", encoded, http.StatusOK)
			replay := decodeServiceOrderHTTPResponse(t, replayBody)
			if first.ID == "" || first.JobID == "" || first.ID != replay.ID || first.JobID != replay.JobID {
				t.Fatalf("exact begin replay changed durable order/job: first=%+v replay=%+v", first, replay)
			}
			if first.AgentManagerRunID != door.runID || replay.AgentManagerRunID != door.runID {
				t.Fatalf("order did not durably retain Agent Manager run identity: first=%q replay=%q want=%q", first.AgentManagerRunID, replay.AgentManagerRunID, door.runID)
			}
			changedBody := map[string]any{}
			if err := json.Unmarshal([]byte(encoded), &changedBody); err != nil {
				t.Fatal(err)
			}
			changedBody["job_input"] = map[string]string{"sector": "battery storage", "market": "CA"}
			serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders", marshalServiceOrderJSON(t, changedBody), http.StatusConflict)
			if door.signature != "" {
				changedSignature := map[string]any{}
				if err := json.Unmarshal([]byte(encoded), &changedSignature); err != nil {
					t.Fatal(err)
				}
				changedSignature["payment_signature"] = door.signature + ":changed"
				serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders", marshalServiceOrderJSON(t, changedSignature), http.StatusConflict)
				changedRun := map[string]any{}
				if err := json.Unmarshal([]byte(encoded), &changedRun); err != nil {
					t.Fatal(err)
				}
				changedRun["agent_manager_run_id"] = "84cde810-bd56-41e0-9e38-44154e221744"
				serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders", marshalServiceOrderJSON(t, changedRun), http.StatusConflict)
			}

			_, verifiedBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders/"+first.ID+"/verify-capture", "", http.StatusOK)
			verified := decodeServiceOrderHTTPResponse(t, verifiedBody)
			if verified.VerificationID == "" || verified.CaptureID == "" || verified.State != commerce.ServiceOrderLedgerPending {
				t.Fatalf("verify/capture did not complete before reconciliation: %+v", verified)
			}
			if door.name == commerce.ServiceOrderDoorAgentRun {
				serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders/"+first.ID+"/reconcile", "", http.StatusUnprocessableEntity)
				_, pendingBody := serviceOrderHTTPCall(t, api.Client(), http.MethodGet, api.URL+"/api/v1/service-orders/"+first.ID, "", http.StatusOK)
				pending := decodeServiceOrderHTTPResponse(t, pendingBody)
				if pending.State != commerce.ServiceOrderLedgerPending || pending.LedgerPostingID != "" {
					t.Fatalf("LPBS linked the Treasury receipt before it became readable: %+v", pending)
				}
				treasury.mu.Lock()
				treasury.receiptVisible = true
				treasury.mu.Unlock()
			}
			_, reconciledBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders/"+first.ID+"/reconcile", "", http.StatusOK)
			reconciled := decodeServiceOrderHTTPResponse(t, reconciledBody)
			if reconciled.State != commerce.ServiceOrderReconciled || reconciled.JobID != first.JobID || reconciled.OfferID != offer.ID || reconciled.OfferVersion != offer.Version || reconciled.AmountMinor != offer.ProposedPrice.AmountMinor || reconciled.Currency != offer.ProposedPrice.Currency {
				t.Fatalf("reconciled order lost the canonical offer/job binding: %+v", reconciled)
			}
			if reconciled.LedgerPostingID == "" || reconciled.LedgerReceiptID == "" {
				t.Fatalf("reconciled order has no accepted ledger posting and receipt: %+v", reconciled)
			}
			_, captureReplayBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders/"+first.ID+"/verify-capture", "", http.StatusOK)
			captureReplay := decodeServiceOrderHTTPResponse(t, captureReplayBody)
			_, ledgerReplayBody := serviceOrderHTTPCall(t, api.Client(), http.MethodPost, api.URL+"/api/v1/service-orders/"+first.ID+"/reconcile", "", http.StatusOK)
			ledgerReplay := decodeServiceOrderHTTPResponse(t, ledgerReplayBody)
			if captureReplay.CaptureID != reconciled.CaptureID || ledgerReplay.LedgerPostingID != reconciled.LedgerPostingID || ledgerReplay.LedgerReceiptID != reconciled.LedgerReceiptID {
				t.Fatalf("terminal replay changed capture or accepted ledger receipt: initial=%+v capture-replay=%+v ledger-replay=%+v", reconciled, captureReplay, ledgerReplay)
			}
			orders[door.name] = reconciled
		})
	}

	jobIDs := make(map[string]struct{}, len(doors))
	postingIDs := make(map[string]struct{}, len(doors))
	receiptIDs := make(map[string]struct{}, len(doors))
	for door, order := range orders {
		addServiceOrderJourneyID(t, jobIDs, door, order.JobID)
		addServiceOrderJourneyID(t, postingIDs, door, order.LedgerPostingID)
		addServiceOrderJourneyID(t, receiptIDs, door, order.LedgerReceiptID)
	}
	if len(orders) != len(doors) || len(jobIDs) != 4 || len(postingIDs) != 4 || len(receiptIDs) != 4 {
		t.Fatalf("four-door result is incomplete: orders=%d jobs=%d postings=%d receipts=%d", len(orders), len(jobIDs), len(postingIDs), len(receiptIDs))
	}
	var durableOrders, durableJobs, durableReconciled int
	if err := spine.db.QueryRowContext(ctx, `SELECT count(*),count(DISTINCT job_id),count(*) FILTER (WHERE state='reconciled') FROM service_orders`).Scan(&durableOrders, &durableJobs, &durableReconciled); err != nil {
		t.Fatal(err)
	}
	if durableOrders != 4 || durableJobs != 4 || durableReconciled != 4 {
		t.Fatalf("durable Postgres orders/jobs/reconciled=%d/%d/%d, want 4/4/4", durableOrders, durableJobs, durableReconciled)
	}

	postings, err := ledgerconnect.NewJournalServiceClient(http.DefaultClient, config["url"]).ListPostings(ctx, connect.NewRequest(&ledgerpb.ListPostingsRequest{BookId: config["book"], Limit: 100}))
	if err != nil {
		t.Fatal(err)
	}
	if len(postings.Msg.Postings) != 4 {
		t.Fatalf("isolated real Money Ledger test fixture has %d postings; want four", len(postings.Msg.Postings))
	}
	ledgerAuth.mu.Lock()
	ledgerTokens := append([]string(nil), ledgerAuth.authorization...)
	ledgerAuth.mu.Unlock()
	if len(ledgerTokens) != 3 {
		t.Fatalf("LPBS posted %d scoped test Money Ledger requests; want three non-x402 doors", len(ledgerTokens))
	}
	for _, token := range ledgerTokens {
		if token != "Bearer lpbs-run-only-money-ledger-write" {
			t.Fatalf("LPBS Money Ledger request used unexpected credential %q", token)
		}
	}
	postingCategories := make(map[string]string, len(postings.Msg.Postings))
	for _, posting := range postings.Msg.Postings {
		postingCategories[posting.Id] = posting.Event.Category
	}
	for door, order := range orders {
		category := "test.service_order.sandbox"
		if door == commerce.ServiceOrderDoorAgentRun {
			category = "test.service_order.treasury_x402"
		}
		if got := postingCategories[order.LedgerPostingID]; got != category {
			t.Fatalf("door %s posting %s category=%q, want %q", door, order.LedgerPostingID, got, category)
		}
	}
	if treasury.settlementExecutions != 1 || treasury.verifyCalls != 1 {
		t.Fatalf("Agent Manager fake Treasury call counts verify=%d settlement=%d, want 1 each", treasury.verifyCalls, treasury.settlementExecutions)
	}
}

type serviceOrderLedgerAuthTransport struct {
	mu            sync.Mutex
	base          http.RoundTripper
	authorization []string
}

func (t *serviceOrderLedgerAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.mu.Lock()
	t.authorization = append(t.authorization, request.Header.Get("Authorization"))
	t.mu.Unlock()
	return t.base.RoundTrip(request)
}

func addServiceOrderJourneyID(t *testing.T, ids map[string]struct{}, door, id string) {
	t.Helper()
	if id == "" {
		t.Fatalf("door %s has an empty durable identifier", door)
	}
	if _, exists := ids[id]; exists {
		t.Fatalf("door %s reused durable identifier %s", door, id)
	}
	ids[id] = struct{}{}
}

func serviceOrderHTTPCall(t *testing.T, client *http.Client, method, url, body string, wantStatus int) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if method == http.MethodPost {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var response bytes.Buffer
	if _, err := response.ReadFrom(resp.Body); err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, url, resp.StatusCode, wantStatus, response.String())
	}
	copy := *resp
	return &copy, response.Bytes()
}

func marshalServiceOrderJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func decodeServiceOrderHTTPResponse(t *testing.T, body []byte) commerce.ServiceOrder {
	t.Helper()
	var order commerce.ServiceOrder
	if err := json.Unmarshal(body, &order); err != nil {
		t.Fatalf("decode service-order HTTP response: %v body=%s", err, body)
	}
	return order
}
