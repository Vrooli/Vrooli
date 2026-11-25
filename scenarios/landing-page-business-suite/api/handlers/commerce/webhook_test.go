package billing

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookRejectsOversizedBodyBeforeService(t *testing.T) {
	called := false
	status := 0
	deps := WebhookDependencies{
		Handle: func(context.Context, []byte, string) error { called = true; return nil },
		WriteError: func(_ http.ResponseWriter, got int, _, _ string) {
			status = got
		},
		WriteJSON: func(http.ResponseWriter, any) {},
		Log:       func(string, map[string]any) {},
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(make([]byte, maxWebhookBodyBytes+1)))
	req.Header.Set("Stripe-Signature", "sig")
	Webhook(deps).ServeHTTP(httptest.NewRecorder(), req)
	if status != http.StatusRequestEntityTooLarge || called {
		t.Fatalf("status=%d called=%t", status, called)
	}
}

func TestWebhookMapsProcessorFailureWithoutLeakingDetails(t *testing.T) {
	status, message := 0, ""
	deps := WebhookDependencies{
		Handle: func(context.Context, []byte, string) error { return errors.New("provider secret") },
		WriteError: func(_ http.ResponseWriter, got int, gotMessage, _ string) {
			status, message = got, gotMessage
		},
		WriteJSON: func(http.ResponseWriter, any) {},
		Log:       func(string, map[string]any) {},
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("{}"))
	req.Header.Set("Stripe-Signature", "sig")
	Webhook(deps).ServeHTTP(httptest.NewRecorder(), req)
	if status != http.StatusBadRequest || message != "Webhook processing failed" {
		t.Fatalf("status=%d message=%q", status, message)
	}
}

func TestWebhookPassesRequestContextToProcessor(t *testing.T) {
	type contextKey struct{}
	want := "selected-db-pool"
	var got context.Context
	deps := WebhookDependencies{
		Handle:     func(ctx context.Context, _ []byte, _ string) error { got = ctx; return nil },
		WriteError: func(http.ResponseWriter, int, string, string) { t.Fatal("unexpected webhook error") },
		WriteJSON:  func(http.ResponseWriter, any) {},
		Log:        func(string, map[string]any) {},
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBufferString("{}"))
	req = req.WithContext(context.WithValue(req.Context(), contextKey{}, want))
	req.Header.Set("Stripe-Signature", "sig")
	Webhook(deps).ServeHTTP(httptest.NewRecorder(), req)
	if got == nil || got.Value(contextKey{}) != want {
		t.Fatalf("processor context value = %v, want %q", got, want)
	}
}
