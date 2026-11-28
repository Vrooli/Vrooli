package serviceorders

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"landing-page-business-suite-api/internal/commerce"
)

type runtimeStub struct {
	beginRequest                              commerce.ServiceOrderOfferRequest
	beginCalls                                int
	getID, verifyID, reconcileID              string
	getErr, beginErr, verifyErr, reconcileErr error
	order                                     *commerce.ServiceOrder
}

func (s *runtimeStub) BeginFromOffer(_ context.Context, _ commerce.ServiceOrderOfferSource, r commerce.ServiceOrderOfferRequest) (*commerce.ServiceOrder, error) {
	s.beginCalls++
	s.beginRequest = r
	if s.beginErr != nil {
		return nil, s.beginErr
	}
	return &commerce.ServiceOrder{ID: "order-1", JobID: "job-1", OfferID: r.OfferID, OfferVersion: r.OfferVersion, AmountMinor: 4900, Currency: "USD", Door: r.Door, Mode: "sandbox", State: commerce.ServiceOrderAuthorized}, nil
}

func (s *runtimeStub) GetOrder(_ context.Context, id string) (*commerce.ServiceOrder, error) {
	s.getID = id
	if s.order != nil {
		return s.order, s.getErr
	}
	return &commerce.ServiceOrder{ID: id}, s.getErr
}

type resultReaderStub struct{ result DeliveryResult }

func (s resultReaderStub) ReadResult(context.Context, commerce.ServiceOrder) (DeliveryResult, error) {
	return s.result, nil
}

func (s *runtimeStub) VerifyAndCapture(_ context.Context, id string) (*commerce.ServiceOrder, error) {
	s.verifyID = id
	return &commerce.ServiceOrder{ID: id, State: commerce.ServiceOrderCaptured}, s.verifyErr
}

func (s *runtimeStub) Reconcile(_ context.Context, id string) (*commerce.ServiceOrder, error) {
	s.reconcileID = id
	return &commerce.ServiceOrder{ID: id, State: commerce.ServiceOrderReconciled}, s.reconcileErr
}

type offerStub struct{}

func (offerStub) ResolveServiceOrderQuote(context.Context, string, int) (commerce.ServiceOrderQuote, error) {
	return commerce.ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD", X402PriceID: "company-market-scan-v1"}, nil
}

func newTestHandler(t *testing.T, rt Runtime) *Handler {
	t.Helper()
	h, err := NewHandler(Dependencies{Runtime: rt, Offers: offerStub{}, TestMode: true})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func request(t *testing.T, h http.Handler, method, path, body string, remote string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = remote
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestNewHandlerRequiresTestModeAndCanonicalOfferSource(t *testing.T) {
	rt := &runtimeStub{}
	if _, err := NewHandler(Dependencies{Runtime: rt, Offers: offerStub{}}); err == nil {
		t.Fatal("production-mode handler was constructed")
	}
	if _, err := NewHandler(Dependencies{Runtime: rt, TestMode: true}); err == nil {
		t.Fatal("handler without canonical offer source was constructed")
	}
}

func TestBeginUsesOfferIdentityAndPassesNoCallerPrice(t *testing.T) {
	rt := &runtimeStub{}
	h := newTestHandler(t, rt)
	body := `{"offer_id":"company-market-scan","offer_version":1,"door":"mcp","idempotency_key":"mcp-job-1","job_input":{"company":"Acme"},"payment_signature":"test-signature"}`
	rec := request(t, h, http.MethodPost, "/api/v1/service-orders", body, "127.0.0.1:45000")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if rt.beginCalls != 1 || rt.beginRequest.OfferID != "company-market-scan" || rt.beginRequest.OfferVersion != 1 || rt.beginRequest.Door != "mcp" || rt.beginRequest.IdempotencyKey != "mcp-job-1" {
		t.Fatalf("unexpected request passed to runtime: %+v", rt.beginRequest)
	}
	if got := string(rt.beginRequest.JobInput); got != `{"company":"Acme"}` {
		t.Fatalf("job_input = %s", got)
	}
	var response map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["amount_minor"] != float64(4900) || response["mode"] != "sandbox" || response["job_id"] != "job-1" {
		t.Fatalf("canonical price or durable identity missing: %v", response)
	}
}

func TestOfferProjectionUsesCanonicalOfferSourceAndDoesNotCache(t *testing.T) {
	h := newTestHandler(t, &runtimeStub{})
	rec := request(t, h, http.MethodGet, "/api/v1/service-orders/offer?offer_id=company-market-scan&offer_version=1", "", "127.0.0.1:45000")
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", rec.Header().Get("Cache-Control"))
	}
	var got offerResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got != (offerResponse{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD"}) {
		t.Fatalf("offer = %+v", got)
	}
}

func TestOfferProjectionRejectsAmbiguousQueryAndNonLoopback(t *testing.T) {
	h := newTestHandler(t, &runtimeStub{})
	for _, tc := range []struct {
		path, remote string
		want         int
	}{
		{"?offer_id=company-market-scan&offer_version=1&amount_minor=1", "127.0.0.1:45000", http.StatusBadRequest},
		{"?offer_id=company-market-scan&offer_version=0", "127.0.0.1:45000", http.StatusBadRequest},
		{"?offer_id=company-market-scan&offer_version=1", "192.0.2.10:45000", http.StatusForbidden},
	} {
		rec := request(t, h, http.MethodGet, "/api/v1/service-orders/offer"+tc.path, "", tc.remote)
		if rec.Code != tc.want {
			t.Errorf("query %q remote %s: status %d want %d", tc.path, tc.remote, rec.Code, tc.want)
		}
	}
}

func TestBeginRejectsPriceFieldsUnknownFieldsAndOversizedJSON(t *testing.T) {
	for _, body := range []string{
		`{"offer_id":"company-market-scan","offer_version":1,"door":"browser","idempotency_key":"k","amount_minor":1}`,
		`{"offer_id":"company-market-scan","offer_version":1,"door":"browser","idempotency_key":"k","unexpected":true}`,
		`{"offer_id":"company-market-scan","offer_version":1,"door":"browser","idempotency_key":"k"} {}`,
		`{"offer_id":"company-market-scan","offer_version":1,"door":"browser","idempotency_key":"k","job_input":"` + strings.Repeat("x", maxRequestBytes) + `"}`,
	} {
		rt := &runtimeStub{}
		rec := request(t, newTestHandler(t, rt), http.MethodPost, "/api/v1/service-orders", body, "127.0.0.1:45000")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("body was not rejected: status=%d body=%s", rec.Code, rec.Body.String())
		}
		if rt.beginCalls != 0 {
			t.Errorf("invalid body reached runtime")
		}
	}
}

func TestRoutesReadVerifyCaptureAndReconcileStableOrderID(t *testing.T) {
	rt := &runtimeStub{}
	router := mux.NewRouter()
	newTestHandler(t, rt).RegisterRoutes(router)
	for _, tc := range []struct {
		method, path string
		state        string
	}{
		{http.MethodGet, "/api/v1/service-orders/order-abc", "get"},
		{http.MethodPost, "/api/v1/service-orders/order-abc/verify-capture", "verify"},
		{http.MethodPost, "/api/v1/service-orders/order-abc/reconcile", "reconcile"},
	} {
		rec := request(t, router, tc.method, tc.path, "", "127.0.0.1:45000")
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d: %s", tc.path, rec.Code, rec.Body.String())
		}
	}
	if rt.getID != "order-abc" || rt.verifyID != "order-abc" || rt.reconcileID != "order-abc" {
		t.Fatalf("order identity changed: %+v", rt)
	}
}

func TestResultIsOnlyAvailableAfterReconciliationAndIsReturnedAsMarkdown(t *testing.T) {
	order := &commerce.ServiceOrder{ID: "order-1", JobID: "job-1", OfferID: "company-market-scan", OfferVersion: 1, State: commerce.ServiceOrderReconciled}
	rt := &runtimeStub{order: order}
	h, err := NewHandler(Dependencies{Runtime: rt, Offers: offerStub{}, Results: resultReaderStub{result: DeliveryResult{Format: "markdown", Body: []byte("# Order result\n")}}, TestMode: true})
	if err != nil {
		t.Fatal(err)
	}
	rec := request(t, h, http.MethodGet, "/api/v1/service-orders/order-1/result", "", "127.0.0.1:45000")
	if rec.Code != http.StatusOK || rec.Body.String() != "# Order result\n" || !strings.HasPrefix(rec.Header().Get("Content-Type"), "text/markdown") {
		t.Fatalf("result response = %d %q %v", rec.Code, rec.Body.String(), rec.Header())
	}
	if !strings.Contains(rec.Header().Get("Content-Disposition"), "company-market-scan-job-1.md") || rec.Header().Get("X-Content-SHA256") == "" {
		t.Fatalf("result response lacks a stable filename and digest: %v", rec.Header())
	}

	order.State = commerce.ServiceOrderLedgerPending
	rec = request(t, h, http.MethodGet, "/api/v1/service-orders/order-1/result", "", "127.0.0.1:45000")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "only after reconciliation") {
		t.Fatalf("unreconciled result response = %d %s", rec.Code, rec.Body.String())
	}
}

func TestTransportEnforcesLoopbackAndEmptyActionBodies(t *testing.T) {
	rt := &runtimeStub{}
	h := newTestHandler(t, rt)
	if rec := request(t, h, http.MethodPost, "/api/v1/service-orders", `{}`, "192.0.2.10:45000"); rec.Code != http.StatusForbidden {
		t.Fatalf("non-loopback begin status = %d", rec.Code)
	}
	if rec := request(t, h, http.MethodPost, "/api/v1/service-orders/order-1/verify-capture", `{"capture":true}`, "127.0.0.1:45000"); rec.Code != http.StatusBadRequest {
		t.Fatalf("action body status = %d", rec.Code)
	}
	if rt.beginCalls != 0 || rt.verifyID != "" {
		t.Fatalf("unsafe request reached runtime: %+v", rt)
	}
}

func TestServiceErrorsMapConflictAndNotFound(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want int
	}{{commerce.ErrServiceOrderIdempotencyConflict, http.StatusConflict}, {sql.ErrNoRows, http.StatusNotFound}} {
		rt := &runtimeStub{beginErr: tc.err}
		rec := request(t, newTestHandler(t, rt), http.MethodPost, "/api/v1/service-orders", `{"offer_id":"x","offer_version":1,"door":"browser","idempotency_key":"k"}`, "127.0.0.1:45000")
		if rec.Code != tc.want {
			t.Errorf("error %v status=%d want %d", tc.err, rec.Code, tc.want)
		}
	}
}
