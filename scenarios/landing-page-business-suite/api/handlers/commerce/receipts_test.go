package billing

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/commerce"
)

func TestRegisterReceiptUsesAuthenticatedIdentity(t *testing.T) {
	var got commerce.Receipt
	handler := RegisterReceipt(ReceiptDependencies{
		Validators: commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string {
			return "authenticated@example.com"
		},
		Register: func(_ context.Context, _ commerce.ReceiptValidators, receipt commerce.Receipt) (*commerce.EntitlementPayload, error) {
			got = receipt
			return &commerce.EntitlementPayload{Status: "active", PlanTier: "pro"}, nil
		},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": message, "kind": kind})
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"apple","token":"signed","user_identity":"attacker@example.com"}`))
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got.UserIdentity != "authenticated@example.com" {
		t.Fatalf("receipt identity = %q, want authenticated identity", got.UserIdentity)
	}
}

func TestRegisterReceiptMapsReplayToConflict(t *testing.T) {
	handler := RegisterReceipt(ReceiptDependencies{
		Validators: commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string {
			return "authenticated@example.com"
		},
		Register: func(context.Context, commerce.ReceiptValidators, commerce.Receipt) (*commerce.EntitlementPayload, error) {
			return nil, commerce.ErrReceiptReplay
		},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": message, "kind": kind})
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"google","token":"purchase"}`))
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusConflict)
	}
}

func TestRegisterReceiptMapsUnexpectedValidatorFailureToUnavailable(t *testing.T) {
	handler := RegisterReceipt(ReceiptDependencies{
		Validators: commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string {
			return "authenticated@example.com"
		},
		Register: func(context.Context, commerce.ReceiptValidators, commerce.Receipt) (*commerce.EntitlementPayload, error) {
			return nil, errors.New("provider unavailable")
		},
		WriteError: func(w http.ResponseWriter, status int, message, kind string) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": message, "kind": kind})
		},
	})

	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"apple","token":"signed"}`))
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
