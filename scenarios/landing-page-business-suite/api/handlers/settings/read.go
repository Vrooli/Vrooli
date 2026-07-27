// Package settings owns transport policy for Stripe configuration. In
// particular, ordinary reads are redacted and explicit reveal is opt-in.
package settings

import (
	"context"
	"net/http"
	"strings"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/proto"
)

type ReadDependencies struct {
	Load       func(context.Context) (*lpbsv1.StripeSettings, error)
	Snapshot   func() *lpbsv1.StripeConfigSnapshot
	Secret     func(string) (string, bool)
	WriteJSON  func(http.ResponseWriter, any)
	WriteError func(http.ResponseWriter, int, string, string)
}

func Get(deps ReadDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := deps.Load(r.Context())
		if err != nil {
			deps.WriteError(w, http.StatusInternalServerError, "Failed to load Stripe settings", "server_error")
			return
		}
		var responseRecord *lpbsv1.StripeSettings
		hasPublishable, hasSecret, hasWebhook, hasAnomalyURL := false, false, false, false
		if record != nil {
			hasPublishable = strings.TrimSpace(record.PublishableKey) != ""
			hasSecret = strings.TrimSpace(record.SecretKey) != ""
			hasWebhook = strings.TrimSpace(record.WebhookSecret) != ""
			hasAnomalyURL = strings.TrimSpace(record.AnomalyWebhookUrl) != ""
			responseRecord = proto.Clone(record).(*lpbsv1.StripeSettings)
			responseRecord.PublishableKey, responseRecord.SecretKey, responseRecord.WebhookSecret, responseRecord.AnomalyWebhookUrl = "", "", "", ""
			responseRecord.AnomalyWebhookUrlSet = hasAnomalyURL
		}
		snapshot := proto.Clone(deps.Snapshot()).(*lpbsv1.StripeConfigSnapshot)
		if hasPublishable {
			snapshot.PublishableKeySet = true
		}
		if hasSecret {
			snapshot.SecretKeySet = true
		}
		if hasWebhook {
			snapshot.WebhookSecretSet = true
		}
		deps.WriteJSON(w, &lpbsv1.GetStripeSettingsResponse{Settings: responseRecord, Snapshot: snapshot})
	}
}

func Reveal(deps ReadDependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		field := r.URL.Query().Get("field")
		if field == "" {
			deps.WriteError(w, http.StatusBadRequest, "Missing 'field' query parameter", "validation")
			return
		}
		if field != "secret_key" && field != "webhook_secret" && field != "publishable_key" && field != "anomaly_webhook_url" {
			deps.WriteError(w, http.StatusBadRequest, "Invalid field. Allowed: secret_key, webhook_secret, publishable_key, anomaly_webhook_url", "validation")
			return
		}
		if field == "anomaly_webhook_url" {
			record, err := deps.Load(r.Context())
			if err != nil {
				deps.WriteError(w, http.StatusInternalServerError, "Failed to load Stripe settings", "server_error")
				return
			}
			if record == nil || strings.TrimSpace(record.AnomalyWebhookUrl) == "" {
				deps.WriteError(w, http.StatusNotFound, "No value set for this field", "not_found")
				return
			}
			deps.WriteJSON(w, map[string]string{"field": field, "value": record.AnomalyWebhookUrl})
			return
		}
		value, ok := deps.Secret(field)
		if !ok {
			deps.WriteError(w, http.StatusNotFound, "No value set for this field", "not_found")
			return
		}
		deps.WriteJSON(w, map[string]string{"field": field, "value": value})
	}
}
