package commerce

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
)

var _ ServiceOrderDB = (*database.RoutedDB)(nil)

type serviceOrderContractProvider struct{}

func (serviceOrderContractProvider) Authorize(context.Context, ServiceOrderAuthorizationRequest) (ServiceOrderAuthorizationReceipt, error) {
	return ServiceOrderAuthorizationReceipt{}, nil
}

func (serviceOrderContractProvider) Capture(context.Context, ServiceOrderCaptureRequest) (ServiceOrderCaptureReceipt, error) {
	return ServiceOrderCaptureReceipt{}, nil
}

func (serviceOrderContractProvider) Void(context.Context, ServiceOrderVoidRequest) (ServiceOrderVoidReceipt, error) {
	return ServiceOrderVoidReceipt{}, nil
}

type serviceOrderContractVerifier struct{}

func (serviceOrderContractVerifier) Verify(context.Context, ServiceOrder) (ServiceOrderVerification, error) {
	return ServiceOrderVerification{}, nil
}

type serviceOrderContractReceiptLookup struct{}

func (serviceOrderContractReceiptLookup) ReadAccepted(context.Context, string) (TreasuryX402Receipt, error) {
	return TreasuryX402Receipt{}, nil
}

type serviceOrderContractOfferSource struct{ quote ServiceOrderQuote }

func (s serviceOrderContractOfferSource) ResolveServiceOrderQuote(context.Context, string, int) (ServiceOrderQuote, error) {
	return s.quote, nil
}

func TestServiceOrderDoorContractNamesAllFourPS2Doors(t *testing.T) {
	for _, door := range []string{
		ServiceOrderDoorBrowser,
		ServiceOrderDoorBrowserAgent,
		ServiceOrderDoorMCP,
		ServiceOrderDoorAgentRun,
	} {
		if !validServiceOrderDoor(door) {
			t.Errorf("expected PS2 door %q to be accepted", door)
		}
	}
	for _, door := range []string{"", "plugin", "x402", "mpp_x402", "unknown"} {
		if validServiceOrderDoor(door) {
			t.Errorf("unexpected door %q was accepted", door)
		}
	}
}

func TestServiceOrderRequestRejectsUnknownDoorBeforePersistence(t *testing.T) {
	request := BeginServiceOrderRequest{
		Quote:          ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD"},
		Door:           "caller-chosen-channel",
		IdempotencyKey: "door-validation-test",
		JobInput:       json.RawMessage(`{"sector":"battery storage"}`),
	}
	if err := validateServiceOrderRequest(request); err == nil {
		t.Fatal("unknown door was accepted")
	}
}

func TestCompanyMarketScanRequestEnforcesItsPublishedInputLimit(t *testing.T) {
	base := BeginServiceOrderRequest{
		Quote: ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD"},
		Door:  ServiceOrderDoorBrowser, IdempotencyKey: "bounded-input-test",
		JobInput: json.RawMessage(`{"sector":"Battery storage"}`),
	}
	if err := validateServiceOrderRequest(base); err != nil {
		t.Fatalf("valid sector request rejected: %v", err)
	}
	for name, input := range map[string]string{
		"missing field": `{"company":"Acme"}`,
		"extra field":   `{"sector":"Acme","notes":"private"}`,
		"blank field":   `{"sector":"  "}`,
		"over limit":    `{"sector":"` + strings.Repeat("x", 501) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			request := base
			request.JobInput = json.RawMessage(input)
			if err := validateServiceOrderRequest(request); err == nil {
				t.Fatalf("invalid job input %s was accepted", name)
			}
		})
	}
}

func TestAgentManagerX402ServiceOrderRequiresRunIdentityAndKeepsItDoorScoped(t *testing.T) {
	base := BeginServiceOrderRequest{
		Quote: ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD", X402PriceID: "company-market-scan-v1"},
		Door:  ServiceOrderDoorAgentRun, IdempotencyKey: "run-identity-validation", PaymentSignature: "synthetic-test-signature",
		JobInput: json.RawMessage(`{"sector":"battery storage"}`),
	}
	if err := validateServiceOrderRequest(base); err == nil {
		t.Fatal("Agent Manager x402 order without run identity was accepted")
	}
	base.AgentManagerRunID = "not-a-uuid"
	if err := validateServiceOrderRequest(base); err == nil {
		t.Fatal("Agent Manager x402 order with malformed run identity was accepted")
	}
	base.AgentManagerRunID = "c2df5c38-daf1-4fd3-a2f9-279904b6e82f"
	if err := validateServiceOrderRequest(base); err != nil {
		t.Fatalf("valid run-bound Agent Manager x402 order rejected: %v", err)
	}
	base.Door = ServiceOrderDoorBrowser
	if err := validateServiceOrderRequest(base); err == nil {
		t.Fatal("browser order accepted Agent Manager run identity")
	}
}

func TestServiceOrderAcceptsTreasuryReceiptOnlyComposition(t *testing.T) {
	service, err := NewServiceOrderSandbox(ServiceOrderSandboxOptions{
		DB:               &sql.DB{},
		Provider:         serviceOrderContractProvider{},
		Verifier:         serviceOrderContractVerifier{},
		TreasuryReceipts: serviceOrderContractReceiptLookup{},
	})
	if err != nil {
		t.Fatalf("Treasury receipt-only service order composition rejected: %v", err)
	}
	if service == nil {
		t.Fatal("Treasury receipt-only composition returned nil service")
	}
}

func TestServiceOrderReceiptOnlyCompositionRejectsNonX402Door(t *testing.T) {
	service, err := NewServiceOrderSandbox(ServiceOrderSandboxOptions{
		DB:               &sql.DB{},
		Provider:         serviceOrderContractProvider{},
		Verifier:         serviceOrderContractVerifier{},
		TreasuryReceipts: serviceOrderContractReceiptLookup{},
	})
	if err != nil {
		t.Fatalf("Treasury receipt-only service order composition rejected: %v", err)
	}
	_, err = service.Begin(context.Background(), BeginServiceOrderRequest{
		Quote: ServiceOrderQuote{OfferID: "company-market-scan", OfferVersion: 1, AmountMinor: 4900, Currency: "USD"},
		Door:  ServiceOrderDoorBrowser, IdempotencyKey: "must-require-ledger-writer",
		JobInput: json.RawMessage(`{"sector":"Example"}`),
	})
	if err == nil || err.Error() != "non-x402 orders require an isolated Money Ledger writer" {
		t.Fatalf("Begin() error = %v; want missing Money Ledger writer refusal", err)
	}
}

func TestServiceOrderBeginFromOfferRejectsSourceIdentityMismatchBeforePersistence(t *testing.T) {
	service, err := NewServiceOrderSandbox(ServiceOrderSandboxOptions{
		DB:               &sql.DB{},
		Provider:         serviceOrderContractProvider{},
		Verifier:         serviceOrderContractVerifier{},
		TreasuryReceipts: serviceOrderContractReceiptLookup{},
	})
	if err != nil {
		t.Fatalf("NewServiceOrderSandbox() error = %v", err)
	}
	_, err = service.BeginFromOffer(context.Background(), serviceOrderContractOfferSource{quote: ServiceOrderQuote{
		OfferID: "other-offer", OfferVersion: 7, AmountMinor: 1, Currency: "USD", X402PriceID: "other-price",
	}}, ServiceOrderOfferRequest{
		OfferID: "company-market-scan", OfferVersion: 1, Door: ServiceOrderDoorAgentRun,
		IdempotencyKey: "source-identity-mismatch", JobInput: json.RawMessage(`{"sector":"Example"}`),
		PaymentSignature: "opaque-test-signature",
	})
	if err == nil || err.Error() != "service offer source returned a different offer identity or version" {
		t.Fatalf("BeginFromOffer() error = %v, want canonical identity mismatch", err)
	}
}
