package settings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
)

func TestGetRedactsStoredSecretsWithoutMutatingSource(t *testing.T) {
	record := &lpbsv1.StripeSettings{PublishableKey: "pk_test_source", SecretKey: "sk_test_source", WebhookSecret: "whsec_source", AnomalyWebhookUrl: "https://example.test/hook"}
	var response *lpbsv1.GetStripeSettingsResponse
	handler := Get(ReadDependencies{
		Load:      func(context.Context) (*lpbsv1.StripeSettings, error) { return record, nil },
		Snapshot:  func() *lpbsv1.StripeConfigSnapshot { return &lpbsv1.StripeConfigSnapshot{} },
		WriteJSON: func(_ http.ResponseWriter, payload any) { response = payload.(*lpbsv1.GetStripeSettingsResponse) },
	})
	handler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/admin/settings/stripe", nil))
	if response == nil || response.Settings.GetSecretKey() != "" || response.Settings.GetWebhookSecret() != "" || response.Settings.GetPublishableKey() != "" || response.Settings.GetAnomalyWebhookUrl() != "" || !response.Settings.GetAnomalyWebhookUrlSet() {
		t.Fatalf("response settings were not redacted: %+v", response.GetSettings())
	}
	if record.GetSecretKey() != "sk_test_source" || record.GetAnomalyWebhookUrl() != "https://example.test/hook" {
		t.Fatalf("source record was mutated: %+v", record)
	}
}
