package commerce

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/servicetoken"
)

func TestTreasuryX402HTTPClientBindsVerifyAndSettleToOneLPBSJob(t *testing.T) {
	const (
		priceID     = "company-market-scan-v1"
		resourceID  = "company-market-scan:v1"
		orderID     = "order-17"
		jobID       = "job-17"
		admissionID = "admission-17"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer order-scoped-token" {
			t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
		}
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case treasuryX402VerifyPath:
			if body["price_id"] != priceID || body["resource_id"] != resourceID || body["order_id"] != orderID || body["job_id"] != jobID || body["payment_signature"] != "signed-payment" {
				t.Fatalf("verify binding = %#v", body)
			}
			_, _ = w.Write([]byte(`{"admission_id":"admission-17","status":"authorized","mode":"local-fork","payer":"0xabc","network":"eip155:8453","price_id":"company-market-scan-v1","resource_id":"company-market-scan:v1","order_id":"order-17","job_id":"job-17"}`))
		case "/api/v1/x402/authorizations/admission-17/settle":
			if _, hasAdmissionBody := body["admission_id"]; hasAdmissionBody || body["resource_id"] != resourceID || body["order_id"] != orderID || body["job_id"] != jobID {
				t.Fatalf("settle binding = %#v", body)
			}
			_, _ = w.Write([]byte(`{"admission_id":"admission-17","price_id":"company-market-scan-v1","status":"settled","payer":"0xabc","mode":"local-fork","transaction_id":"0xtx17","network":"eip155:8453","job_id":"job-17","resource_id":"company-market-scan:v1","order_id":"order-17","posting_id":"posting-17","receipt_id":"ledger-receipt-17"}`))
		case "/api/v1/x402/authorizations/admission-17/cancel":
			if _, hasAdmissionBody := body["admission_id"]; hasAdmissionBody || body["resource_id"] != resourceID || body["order_id"] != orderID || body["job_id"] != jobID {
				t.Fatalf("cancel binding = %#v", body)
			}
			_, _ = w.Write([]byte(`{"admission_id":"admission-17","price_id":"company-market-scan-v1","status":"cancelled","payer":"0xabc","network":"eip155:8453","mode":"local-fork","job_id":"job-17","resource_id":"company-market-scan:v1","order_id":"order-17"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	client, err := NewTreasuryX402HTTPClient(server.URL, fixedTreasuryReceiptToken("order-scoped-token"), server.Client())
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := client.VerifyForResource(context.Background(), TreasuryX402VerifyRequest{PriceID: priceID, ResourceID: resourceID, OrderID: orderID, JobID: jobID, PaymentSignature: "signed-payment"})
	if err != nil {
		t.Fatal(err)
	}
	if authorization.AdmissionID != admissionID {
		t.Fatalf("authorization = %+v", authorization)
	}
	settlement, err := client.SettleAuthorization(context.Background(), TreasuryX402SettleRequest{AdmissionID: admissionID, PriceID: priceID, Payer: "0xabc", Network: "eip155:8453", ResourceID: resourceID, OrderID: orderID, JobID: jobID})
	if err != nil {
		t.Fatal(err)
	}
	if settlement.Transaction != "0xtx17" || settlement.PostingID != "posting-17" || settlement.ReceiptID != "ledger-receipt-17" {
		t.Fatalf("settlement = %+v", settlement)
	}
	cancellation, err := client.CancelAuthorization(context.Background(), TreasuryX402CancelRequest{AdmissionID: admissionID, PriceID: priceID, Payer: "0xabc", Network: "eip155:8453", ResourceID: resourceID, OrderID: orderID, JobID: jobID})
	if err != nil || cancellation.Status != "cancelled" {
		t.Fatalf("cancellation = %+v, %v", cancellation, err)
	}
}

func TestTreasuryX402HTTPClientKeepsNonterminalSettlementStatesPending(t *testing.T) {
	for _, status := range []string{"unknown", "settling", "recovering"} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"admission_id":"admission-1","price_id":"price-1","status":"` + status + `","payer":"payer-1","mode":"local-fork","transaction_id":"","network":"eip155:8453","job_id":"job-1","resource_id":"resource-1","order_id":"order-1"}`))
			}))
			defer server.Close()
			client, err := NewTreasuryX402HTTPClient(server.URL, fixedTreasuryReceiptToken("token"), nil)
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.SettleAuthorization(context.Background(), TreasuryX402SettleRequest{AdmissionID: "admission-1", PriceID: "price-1", Payer: "payer-1", Network: "eip155:8453", ResourceID: "resource-1", OrderID: "order-1", JobID: "job-1"})
			if !errors.Is(err, ErrTreasuryX402SettlementPending) {
				t.Fatalf("%s settlement error = %v", status, err)
			}
		})
	}
}

func TestTreasuryX402HTTPClientRejectsSettlementBindingMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"admission_id":"admission-1","price_id":"price-1","status":"settled","payer":"payer-1","mode":"local-fork","transaction_id":"0xtx","network":"eip155:8453","job_id":"other-job","resource_id":"resource-1","order_id":"order-1"}`))
	}))
	defer server.Close()
	client, err := NewTreasuryX402HTTPClient(server.URL, fixedTreasuryReceiptToken("token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SettleAuthorization(context.Background(), TreasuryX402SettleRequest{AdmissionID: "admission-1", PriceID: "price-1", Payer: "payer-1", Network: "eip155:8453", ResourceID: "resource-1", OrderID: "order-1", JobID: "job-1"})
	if err == nil || !strings.Contains(err.Error(), "another admission, job, resource or order") {
		t.Fatalf("mismatched settlement binding error = %v", err)
	}
}

func TestTreasuryX402HTTPClientRejectsSettlementPriceMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"admission_id":"admission-1","price_id":"different-price","status":"settled","payer":"payer-1","mode":"local-fork","transaction_id":"0xtx","network":"eip155:8453","job_id":"job-1","resource_id":"resource-1","order_id":"order-1"}`))
	}))
	defer server.Close()
	client, err := NewTreasuryX402HTTPClient(server.URL, fixedTreasuryReceiptToken("token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.SettleAuthorization(context.Background(), TreasuryX402SettleRequest{AdmissionID: "admission-1", PriceID: "company-market-scan-v1", Payer: "payer-1", Network: "eip155:8453", ResourceID: "resource-1", OrderID: "order-1", JobID: "job-1"})
	if err == nil || !strings.Contains(err.Error(), "another admission, job, resource or order") {
		t.Fatalf("mismatched settlement price error = %v", err)
	}
}

func TestTreasuryX402HTTPClientRejectsProductionModeAndNonLoopbackOrigin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"admission_id":"admission-1","status":"authorized","mode":"production","payer":"payer","network":"eip155:8453","price_id":"price","resource_id":"resource","order_id":"order","job_id":"job"}`))
	}))
	defer server.Close()
	client, err := NewTreasuryX402HTTPClient(server.URL, fixedTreasuryReceiptToken("token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.VerifyForResource(context.Background(), TreasuryX402VerifyRequest{PriceID: "price", ResourceID: "resource", OrderID: "order", JobID: "job", PaymentSignature: "signature"})
	if err == nil || !strings.Contains(err.Error(), "differently bound") {
		t.Fatalf("production mode verification error = %v", err)
	}
	if _, err := NewTreasuryX402HTTPClient("https://treasury.example", fixedTreasuryReceiptToken("token"), nil); err == nil {
		t.Fatal("non-loopback Treasury origin was accepted by the test-mode order client")
	}
}

func TestTreasuryX402HTTPClientRefusesRedirectsAndBadBindings(t *testing.T) {
	var destinationCalls int
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		destinationCalls++
		w.WriteHeader(http.StatusOK)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/settle") {
			w.Header().Set("Location", destination.URL)
			w.WriteHeader(http.StatusFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"admission_id":"another","status":"authorized","mode":"local-fork","payer":"payer","network":"eip155:8453","price_id":"price","resource_id":"resource","order_id":"other-order","job_id":"job"}`))
	}))
	defer source.Close()
	client, err := NewTreasuryX402HTTPClient(source.URL, fixedTreasuryReceiptToken("token"), source.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.VerifyForResource(context.Background(), TreasuryX402VerifyRequest{PriceID: "price", ResourceID: "resource", OrderID: "order", JobID: "job", PaymentSignature: "signature"})
	if err == nil || !strings.Contains(err.Error(), "differently bound") {
		t.Fatalf("mismatched verification error = %v", err)
	}
	_, err = client.SettleAuthorization(context.Background(), TreasuryX402SettleRequest{AdmissionID: "admission", PriceID: "price", Payer: "payer", Network: "eip155:8453", ResourceID: "resource", OrderID: "order", JobID: "job"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("redirect settlement error = %v", err)
	}
	if destinationCalls != 0 {
		t.Fatal("service token followed a redirect")
	}
}

func TestTreasuryX402OrderTokensRequireExactPrincipalAndScopes(t *testing.T) {
	tokens := NewTreasuryX402OrderTokens(func(context.Context) (string, error) { return "issued-credential", nil }, "http://127.0.0.1:15000", nil,
		servicetoken.WithExchanger(func(_ context.Context, credential string) (servicetoken.Exchanged, error) {
			if credential != "issued-credential" {
				t.Fatalf("exchanged credential = %q", credential)
			}
			return servicetoken.Exchanged{AccessToken: "order-token", ExpiresAt: time.Now().Add(time.Minute), Subject: TreasuryX402OrderPrincipal, Audience: TreasuryX402OrderAudience, Scopes: []string{TreasuryX402SettleScope, TreasuryX402AuthorizeScope, TreasuryX402CancelScope}}, nil
		}),
	)
	if token, err := tokens.Token(context.Background()); err != nil || token != "order-token" {
		t.Fatalf("valid order token = %q, %v", token, err)
	}
	wrong := NewTreasuryX402OrderTokens(func(context.Context) (string, error) { return "credential", nil }, "http://127.0.0.1:15000", nil,
		servicetoken.WithExchanger(func(context.Context, string) (servicetoken.Exchanged, error) {
			return servicetoken.Exchanged{AccessToken: "too-broad", ExpiresAt: time.Now().Add(time.Minute), Subject: TreasuryX402OrderPrincipal, Audience: TreasuryX402OrderAudience, Scopes: []string{TreasuryX402SettleScope, TreasuryX402AuthorizeScope, TreasuryX402CancelScope, "treasury:wallet-admin"}}, nil
		}),
	)
	if token, err := wrong.Token(context.Background()); token != "" || !errors.Is(err, servicetoken.ErrIdentityMismatch) {
		t.Fatalf("unexpected broad token=%q err=%v", token, err)
	}
}

type recordingTreasuryX402Client struct {
	verifyRequest            TreasuryX402VerifyRequest
	settleRequest            TreasuryX402SettleRequest
	cancelRequest            TreasuryX402CancelRequest
	verifyCalls, settleCalls int
	cancelCalls              int
}

func (c *recordingTreasuryX402Client) VerifyForResource(_ context.Context, request TreasuryX402VerifyRequest) (TreasuryX402Authorization, error) {
	c.verifyCalls++
	c.verifyRequest = request
	return TreasuryX402Authorization{AdmissionID: "admission:" + request.OrderID, Status: "authorized", Mode: "local-fork", Payer: "payer", Network: "eip155:8453", PriceID: request.PriceID, ResourceID: request.ResourceID, OrderID: request.OrderID, JobID: request.JobID}, nil
}

func (c *recordingTreasuryX402Client) SettleAuthorization(_ context.Context, request TreasuryX402SettleRequest) (TreasuryX402Settlement, error) {
	c.settleCalls++
	c.settleRequest = request
	return TreasuryX402Settlement{AdmissionID: request.AdmissionID, PriceID: request.PriceID, Status: "settled", Payer: request.Payer, Mode: "local-fork", Transaction: "tx:" + request.OrderID, Network: request.Network, JobID: request.JobID, ResourceID: request.ResourceID, OrderID: request.OrderID}, nil
}

func (c *recordingTreasuryX402Client) CancelAuthorization(_ context.Context, request TreasuryX402CancelRequest) (TreasuryX402Cancellation, error) {
	c.cancelCalls++
	c.cancelRequest = request
	return TreasuryX402Cancellation{AdmissionID: request.AdmissionID, PriceID: request.PriceID, Status: "cancelled", Payer: request.Payer, Network: request.Network, Mode: "local-fork", JobID: request.JobID, ResourceID: request.ResourceID, OrderID: request.OrderID}, nil
}

func TestTreasuryX402ServiceOrderProviderDefersSettlementUntilCapture(t *testing.T) {
	client := &recordingTreasuryX402Client{}
	provider, err := NewTreasuryX402ServiceOrderProvider(client)
	if err != nil {
		t.Fatal(err)
	}
	authorization, err := provider.Authorize(context.Background(), ServiceOrderAuthorizationRequest{Door: ServiceOrderDoorAgentRun, OrderID: "order", JobID: "job", PriceID: "price", ResourceID: "resource", PaymentSignature: "sig"})
	if err != nil || !authorization.Approved || authorization.AuthorizationID != "admission:order" || client.settleCalls != 0 {
		t.Fatalf("authorize receipt=%+v err=%v settle calls=%d", authorization, err, client.settleCalls)
	}
	capture, err := provider.Capture(context.Background(), ServiceOrderCaptureRequest{Door: ServiceOrderDoorAgentRun, OrderID: "order", JobID: "job", AuthorizationID: authorization.AuthorizationID, ResourceID: "resource", PriceID: "price", AmountMinor: 4900, Currency: "USD"})
	if err != nil || !capture.Captured || capture.TreasuryExternalID != "treasury-x402-inflow:tx:order" || client.settleCalls != 1 {
		t.Fatalf("capture receipt=%+v err=%v settle calls=%d", capture, err, client.settleCalls)
	}
	if client.verifyRequest.OrderID != client.settleRequest.OrderID || client.verifyRequest.JobID != client.settleRequest.JobID || client.verifyRequest.ResourceID != client.settleRequest.ResourceID || client.verifyRequest.PriceID != client.settleRequest.PriceID {
		t.Fatalf("resource/order/job binding changed across flow: verify=%+v settle=%+v", client.verifyRequest, client.settleRequest)
	}
	failedAuthorization, err := provider.Authorize(context.Background(), ServiceOrderAuthorizationRequest{Door: ServiceOrderDoorAgentRun, OrderID: "order-failed", JobID: "job-failed", PriceID: "price", ResourceID: "resource", PaymentSignature: "failed-sig"})
	if err != nil {
		t.Fatal(err)
	}
	void, err := provider.Void(context.Background(), ServiceOrderVoidRequest{Door: ServiceOrderDoorAgentRun, OrderID: "order-failed", JobID: "job-failed", ResourceID: "resource", PriceID: "price", AuthorizationID: failedAuthorization.AuthorizationID})
	if err != nil || !void.Voided || client.cancelCalls != 1 || client.settleCalls != 1 {
		t.Fatalf("verification-failure cancellation=%+v err=%v cancel calls=%d settle calls=%d", void, err, client.cancelCalls, client.settleCalls)
	}
	if client.cancelRequest.AdmissionID != failedAuthorization.AuthorizationID || client.cancelRequest.OrderID != "order-failed" || client.cancelRequest.JobID != "job-failed" || client.cancelRequest.ResourceID != "resource" {
		t.Fatalf("cancellation binding changed: %+v", client.cancelRequest)
	}
}
