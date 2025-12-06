package main

import (
	"encoding/json"
	"net/http"
	"strings"

	lprvv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func handleGetStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := paymentService.GetStripeSettings(r.Context())
		if err != nil {
			http.Error(w, "Failed to load Stripe settings", http.StatusInternalServerError)
			return
		}

		snapshot := stripeService.ConfigSnapshot()
		resp := lprvv1.GetStripeSettingsResponse{
			Snapshot: snapshot,
		}

		if record != nil {
			resp.Settings = &lprvv1.StripeSettings{
				PublishableKey: record.PublishableKey,
				SecretKey:      record.SecretKey,
				WebhookSecret:  record.WebhookSecret,
				DashboardUrl:   record.DashboardURL,
				UpdatedAt:      timestamppb.New(record.UpdatedAt),
			}
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

		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		req := lprvv1.UpdateStripeSettingsRequest{
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

		current, err := paymentService.GetStripeSettings(r.Context())
		if err != nil {
			http.Error(w, "Failed to read current Stripe settings", http.StatusInternalServerError)
			return
		}

		needsPublishable := (current == nil || strings.TrimSpace(current.PublishableKey) == "")
		needsSecret := (current == nil || strings.TrimSpace(current.SecretKey) == "")

		if needsPublishable && (req.PublishableKey == nil || *req.PublishableKey == "") {
			http.Error(w, "publishable_key is required", http.StatusBadRequest)
			return
		}
		if needsSecret && (req.SecretKey == nil || *req.SecretKey == "") {
			http.Error(w, "secret_key is required", http.StatusBadRequest)
			return
		}

		record, err := paymentService.SaveStripeSettings(r.Context(), StripeSettingsInput{
			PublishableKey: req.PublishableKey,
			SecretKey:      req.SecretKey,
			WebhookSecret:  req.WebhookSecret,
			DashboardURL:   req.DashboardUrl,
		})
		if err != nil {
			http.Error(w, "Failed to save Stripe settings", http.StatusInternalServerError)
			return
		}

		if err := stripeService.RefreshConfig(r.Context()); err != nil {
			http.Error(w, "Failed to refresh Stripe runtime config", http.StatusInternalServerError)
			return
		}

		snapshot := stripeService.ConfigSnapshot()
		resp := lprvv1.UpdateStripeSettingsResponse{
			Snapshot: snapshot,
		}

		if record != nil {
			resp.Settings = &lprvv1.StripeSettings{
				PublishableKey: record.PublishableKey,
				SecretKey:      record.SecretKey,
				WebhookSecret:  record.WebhookSecret,
				DashboardUrl:   record.DashboardURL,
			}
			if !record.UpdatedAt.IsZero() {
				resp.Settings.UpdatedAt = timestamppb.New(record.UpdatedAt)
			}
		}

		writeJSON(w, resp)
	}
}
