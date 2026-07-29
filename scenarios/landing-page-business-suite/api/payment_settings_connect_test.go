package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
)

func newStripeSettingsConnectHandler(t *testing.T) (stripeSettingsConnectHandler, *PaymentSettingsService) {
	t.Helper()
	db := setupTestDB(t)
	t.Cleanup(func() { db.Close() })
	resetStripeTestData(t, db)
	payment := NewPaymentSettingsService(db)
	return stripeSettingsConnectHandler{payment: payment, stripe: NewStripeServiceWithSettings(db, NewPlanService(db), payment), anomaly: NewPaymentAnomalyService(context.Background(), db, context.Background())}, payment
}

func TestStripeSettingsConnectUpdateRedactsSecretsAndRefreshesRuntime(t *testing.T) {
	handler, _ := newStripeSettingsConnectHandler(t)
	response, err := handler.UpdateStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.UpdateStripeSettingsRequest{
		PublishableKey: protoString("pk_live_connect"), SecretKey: protoString("sk_live_connect"), WebhookSecret: protoString("whsec_live_connect"), DashboardUrl: protoString("https://dashboard.stripe.com/test"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Msg.GetSettings(); got.GetPublishableKey() != "" || got.GetSecretKey() != "" || got.GetWebhookSecret() != "" {
		t.Fatalf("update response leaked credentials: %+v", got)
	}
	if !response.Msg.GetSnapshot().GetPublishableKeySet() || !response.Msg.GetSnapshot().GetSecretKeySet() || !response.Msg.GetSnapshot().GetWebhookSecretSet() {
		t.Fatalf("expected refreshed credential indicators: %+v", response.Msg.GetSnapshot())
	}
}

func TestStripeSettingsConnectRejectsInvalidUpdate(t *testing.T) {
	handler, _ := newStripeSettingsConnectHandler(t)
	_, err := handler.UpdateStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.UpdateStripeSettingsRequest{AnomalyWebhookEnabled: protoBool(true)}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument || !strings.Contains(err.Error(), "requires anomaly_webhook_url") {
		t.Fatalf("expected missing webhook validation error, got %v", err)
	}
	_, err = handler.UpdateStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.UpdateStripeSettingsRequest{AnomalyRateLimits: protoString(`{"checkout":{"burst":-1,"refill_seconds":10}}`)}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid rate limit error, got %v", err)
	}
}

func TestStripeSettingsConnectGetAndRevealKeepSecretsSeparated(t *testing.T) {
	handler, _ := newStripeSettingsConnectHandler(t)
	_, err := handler.UpdateStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.UpdateStripeSettingsRequest{SecretKey: protoString("sk_live_connect")}))
	if err != nil {
		t.Fatal(err)
	}
	response, err := handler.GetStripeSettings(context.Background(), connect.NewRequest(&lpbsv1.GetStripeSettingsRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if response.Msg.GetSettings().GetSecretKey() != "" {
		t.Fatal("ordinary settings read leaked secret key")
	}
	revealed, err := handler.RevealStripeSecret(context.Background(), connect.NewRequest(&lpbsv1.RevealStripeSecretRequest{Field: "secret_key"}))
	if err != nil || revealed.Msg.GetValue() != "sk_live_connect" {
		t.Fatalf("expected explicit reveal to return configured key, got response=%v error=%v", revealed, err)
	}
}

func TestStripeSettingsConnectRoutesRequireAdminAndExposeOnlyProcedures(t *testing.T) {
	handler, _ := newStripeSettingsConnectHandler(t)
	router := mux.NewRouter()
	required := false
	registerStripeSettingsConnectRoutes(router, handler.payment, handler.stripe, handler.anomaly, func(next http.HandlerFunc) http.HandlerFunc {
		return func(writer http.ResponseWriter, request *http.Request) {
			required = true
			next(writer, request)
		}
	})
	request := httptest.NewRequest(http.MethodPost, lpbsconnect.StripeSettingsServiceGetStripeSettingsProcedure, bytes.NewBufferString("{}"))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if !required || recorder.Code != http.StatusOK {
		t.Fatalf("admin-protected procedure was not mounted correctly: required=%t status=%d body=%s", required, recorder.Code, recorder.Body.String())
	}
	legacy := httptest.NewRecorder()
	router.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/stripe", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy settings route is still mounted: %d", legacy.Code)
	}
}

func protoString(value string) *string { return &value }
func protoBool(value bool) *bool       { return &value }
