package main

import (
	"net/http"
	"strings"

	landing_page_react_vite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

func handleGetStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := paymentService.GetStripeSettings(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to load Stripe settings", ApiErrorTypeServerError)
			return
		}
		hasPublishable := record != nil && strings.TrimSpace(record.PublishableKey) != ""
		hasSecret := record != nil && strings.TrimSpace(record.SecretKey) != ""
		hasWebhook := record != nil && strings.TrimSpace(record.WebhookSecret) != ""
		// Redact secrets before sending to the client.
		if record != nil {
			record.PublishableKey = ""
			record.SecretKey = ""
			record.WebhookSecret = ""
		}

		snapshot := stripeService.ConfigSnapshot()
		// Ensure flags reflect DB state even if runtime config was initialized before admin saves.
		if hasPublishable {
			snapshot.PublishableKeySet = true
		}
		if hasSecret {
			snapshot.SecretKeySet = true
		}
		if hasWebhook {
			snapshot.WebhookSecretSet = true
		}

		resp := &landing_page_react_vite_v1.GetStripeSettingsResponse{
			Snapshot: snapshot,
			Settings: record,
		}

		writeJSON(w, resp)
	}
}

func handleUpdateStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PublishableKey *string `json:"publishable_key"`
			SecretKey      *string `json:"secret_key"`
			WebhookSecret  *string `json:"webhook_secret"`
			DashboardURL   *string `json:"dashboard_url"`
		}

		if !decodeJSONBody(w, r, &body) {
			return
		}

		req := landing_page_react_vite_v1.UpdateStripeSettingsRequest{
			PublishableKey: body.PublishableKey,
			SecretKey:      body.SecretKey,
			WebhookSecret:  body.WebhookSecret,
			DashboardUrl:   body.DashboardURL,
		}

		normalize := func(value *string) *string {
			if value == nil {
				return nil
			}
			trimmed := strings.TrimSpace(*value)
			return &trimmed
		}

		req.PublishableKey = normalize(req.PublishableKey)
		req.SecretKey = normalize(req.SecretKey)
		req.WebhookSecret = normalize(req.WebhookSecret)
		req.DashboardUrl = normalize(req.DashboardUrl)

		if (req.PublishableKey == nil || *req.PublishableKey == "") &&
			(req.SecretKey == nil || *req.SecretKey == "") &&
			(req.WebhookSecret == nil || *req.WebhookSecret == "") &&
			(req.DashboardUrl == nil || *req.DashboardUrl == "") {
			writeJSONError(w, http.StatusBadRequest, "At least one field is required", ApiErrorTypeValidation)
			return
		}

		record, err := paymentService.SaveStripeSettings(r.Context(), StripeSettingsInput{
			PublishableKey: req.PublishableKey,
			SecretKey:      req.SecretKey,
			WebhookSecret:  req.WebhookSecret,
			DashboardURL:   req.DashboardUrl,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to save Stripe settings", ApiErrorTypeServerError)
			return
		}
		hasPublishable := record != nil && strings.TrimSpace(record.PublishableKey) != ""
		hasSecret := record != nil && strings.TrimSpace(record.SecretKey) != ""
		hasWebhook := record != nil && strings.TrimSpace(record.WebhookSecret) != ""
		// Redact secrets before responding.
		if record != nil {
			record.PublishableKey = ""
			record.SecretKey = ""
			record.WebhookSecret = ""
		}

		if err := stripeService.RefreshConfig(r.Context()); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to refresh Stripe runtime config", ApiErrorTypeServerError)
			return
		}

		snapshot := stripeService.ConfigSnapshot()
		if hasPublishable {
			snapshot.PublishableKeySet = true
		}
		if hasSecret {
			snapshot.SecretKeySet = true
		}
		if hasWebhook {
			snapshot.WebhookSecretSet = true
		}
		resp := &landing_page_react_vite_v1.UpdateStripeSettingsResponse{
			Snapshot: snapshot,
			Settings: record,
		}

		writeJSON(w, resp)
	}
}
