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
		UserID: func(context.Context) string { return "user-1" },
		ResolveBusinessAccount: func(_ context.Context, userID, identity, requested string) (string, error) {
			if userID != "user-1" || identity != "authenticated@example.com" || requested != "account-1" {
				t.Fatalf("account resolver got userID=%q identity=%q requested=%q", userID, identity, requested)
			}
			return "account-1", nil
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

	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"apple","token":"signed","user_identity":"attacker@example.com","business_account_id":"account-1"}`))
	response := httptest.NewRecorder()
	handler(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got.UserIdentity != "authenticated@example.com" {
		t.Fatalf("receipt identity = %q, want authenticated identity", got.UserIdentity)
	}
	if got.BusinessAccountID != "account-1" {
		t.Fatalf("receipt business account = %q, want resolved account", got.BusinessAccountID)
	}
}

func TestRegisterReceiptMapsReplayToConflict(t *testing.T) {
	handler := RegisterReceipt(ReceiptDependencies{
		Validators: commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string {
			return "authenticated@example.com"
		},
		UserID:                 func(context.Context) string { return "user-1" },
		ResolveBusinessAccount: func(context.Context, string, string, string) (string, error) { return "account-1", nil },
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
		UserID:                 func(context.Context) string { return "user-1" },
		ResolveBusinessAccount: func(context.Context, string, string, string) (string, error) { return "account-1", nil },
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

func TestRegisterReceiptRejectsUnownedBusinessAccountBeforeRegistration(t *testing.T) {
	registered := false
	handler := RegisterReceipt(ReceiptDependencies{
		Validators:   commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string { return "authenticated@example.com" },
		UserID:       func(context.Context) string { return "user-1" },
		ResolveBusinessAccount: func(context.Context, string, string, string) (string, error) {
			return "", errors.New("not a member")
		},
		Register: func(context.Context, commerce.ReceiptValidators, commerce.Receipt) (*commerce.EntitlementPayload, error) {
			registered = true
			return nil, nil
		},
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"apple","token":"signed","business_account_id":"victim"}`))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusForbidden || registered {
		t.Fatalf("status=%d registered=%t, want forbidden before registration", response.Code, registered)
	}
}

func TestRegisterReceiptRequiresAuthenticatedBusinessUserIDBeforeResolution(t *testing.T) {
	resolved, registered := false, false
	handler := RegisterReceipt(ReceiptDependencies{
		Validators:   commerce.ReceiptValidators{},
		UserIdentity: func(context.Context) string { return "authenticated@example.com" },
		UserID:       func(context.Context) string { return "  " },
		ResolveBusinessAccount: func(context.Context, string, string, string) (string, error) {
			resolved = true
			return "account-1", nil
		},
		Register: func(context.Context, commerce.ReceiptValidators, commerce.Receipt) (*commerce.EntitlementPayload, error) {
			registered = true
			return nil, nil
		},
		WriteError: func(w http.ResponseWriter, status int, _, _ string) { w.WriteHeader(status) },
	})
	request := httptest.NewRequest(http.MethodPost, "/api/v1/subscriptions/receipts", strings.NewReader(`{"source":"apple","token":"signed"}`))
	response := httptest.NewRecorder()
	handler(response, request)
	if response.Code != http.StatusUnauthorized || resolved || registered {
		t.Fatalf("status=%d resolved=%t registered=%t, want unauthorized before resolution", response.Code, resolved, registered)
	}
}
