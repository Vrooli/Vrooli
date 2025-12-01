package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

type stripeSettingsResponse struct {
	PublishableKeyPreview string `json:"publishable_key_preview,omitempty"`
	PublishableKeySet     bool   `json:"publishable_key_set"`
	SecretKeySet          bool   `json:"secret_key_set"`
	WebhookSecretSet      bool   `json:"webhook_secret_set"`
	DashboardURL          string `json:"dashboard_url,omitempty"`
	UpdatedAt             string `json:"updated_at,omitempty"`
	Source                string `json:"source"`
}

type stripeSettingsRequest struct {
	PublishableKey *string `json:"publishable_key"`
	SecretKey      *string `json:"secret_key"`
	WebhookSecret  *string `json:"webhook_secret"`
	DashboardURL   *string `json:"dashboard_url"`
}

func handleGetStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		record, err := paymentService.GetStripeSettings(r.Context())
		if err != nil {
			http.Error(w, "Failed to load Stripe settings", http.StatusInternalServerError)
			return
		}

		snapshot := stripeService.ConfigSnapshot()
		resp := stripeSettingsResponse{
			PublishableKeyPreview: snapshot.PublishableKeyPreview,
			PublishableKeySet:     snapshot.PublishableKeySet,
			SecretKeySet:          snapshot.SecretKeySet,
			WebhookSecretSet:      snapshot.WebhookSecretSet,
			Source:                snapshot.Source,
		}

		if record != nil {
			resp.DashboardURL = record.DashboardURL
			if !record.UpdatedAt.IsZero() {
				resp.UpdatedAt = record.UpdatedAt.UTC().Format(time.RFC3339)
			}
		}

		writeJSON(w, resp)
	}
}

func handleUpdateStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req stripeSettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
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
		req.DashboardURL = normalize(req.DashboardURL)

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
			DashboardURL:   req.DashboardURL,
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
		resp := stripeSettingsResponse{
			PublishableKeyPreview: snapshot.PublishableKeyPreview,
			PublishableKeySet:     snapshot.PublishableKeySet,
			SecretKeySet:          snapshot.SecretKeySet,
			WebhookSecretSet:      snapshot.WebhookSecretSet,
			Source:                snapshot.Source,
		}

		if record != nil && !record.UpdatedAt.IsZero() {
			resp.DashboardURL = record.DashboardURL
			resp.UpdatedAt = record.UpdatedAt.UTC().Format(time.RFC3339)
		}

		writeJSON(w, resp)
	}
}
