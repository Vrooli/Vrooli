package main

import (
	"encoding/json"
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
		hasAnomalyURL := record != nil && strings.TrimSpace(record.AnomalyWebhookUrl) != ""
		// Redact secrets before sending to the client.
		if record != nil {
			record.PublishableKey = ""
			record.SecretKey = ""
			record.WebhookSecret = ""
			// Replace the anomaly URL with a set-indicator flag; the unredacted
			// value is available via handleRevealStripeSecret.
			record.AnomalyWebhookUrl = ""
			record.AnomalyWebhookUrlSet = hasAnomalyURL
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

// handleRevealStripeSecret returns the unredacted value of a specific Stripe secret field.
// Requires admin authentication. Accepts a 'field' query parameter with values:
// - secret_key
// - webhook_secret
// - publishable_key
// Returns the value from the merged config (env vars + database).
func handleRevealStripeSecret(stripeService *StripeService, paymentService *PaymentSettingsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		field := r.URL.Query().Get("field")
		if field == "" {
			writeJSONError(w, http.StatusBadRequest, "Missing 'field' query parameter", ApiErrorTypeValidation)
			return
		}

		allowedFields := map[string]bool{
			"secret_key":          true,
			"webhook_secret":      true,
			"publishable_key":     true,
			"anomaly_webhook_url": true,
		}
		if !allowedFields[field] {
			writeJSONError(w, http.StatusBadRequest, "Invalid field. Allowed: secret_key, webhook_secret, publishable_key, anomaly_webhook_url", ApiErrorTypeValidation)
			return
		}

		// Anomaly webhook URL is stored in payment_settings, not the Stripe runtime
		// config, so read it directly from the settings record.
		if field == "anomaly_webhook_url" {
			if paymentService == nil {
				writeJSONError(w, http.StatusNotFound, "No value set for this field", ApiErrorTypeNotFound)
				return
			}
			record, err := paymentService.GetStripeSettings(r.Context())
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to load Stripe settings", ApiErrorTypeServerError)
				return
			}
			if record == nil || strings.TrimSpace(record.AnomalyWebhookUrl) == "" {
				writeJSONError(w, http.StatusNotFound, "No value set for this field", ApiErrorTypeNotFound)
				return
			}
			writeJSON(w, map[string]string{"field": field, "value": record.AnomalyWebhookUrl})
			return
		}

		value, hasValue := stripeService.GetSecretValue(field)
		if !hasValue {
			writeJSONError(w, http.StatusNotFound, "No value set for this field", ApiErrorTypeNotFound)
			return
		}

		resp := map[string]string{
			"field": field,
			"value": value,
		}
		writeJSON(w, resp)
	}
}

func handleUpdateStripeSettings(paymentService *PaymentSettingsService, stripeService *StripeService, anomalyService *PaymentAnomalyService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			PublishableKey        *string          `json:"publishable_key"`
			SecretKey             *string          `json:"secret_key"`
			WebhookSecret         *string          `json:"webhook_secret"`
			DashboardURL          *string          `json:"dashboard_url"`
			AnomalyWebhookURL     *string          `json:"anomaly_webhook_url"`
			AnomalyWebhookEnabled *bool            `json:"anomaly_webhook_enabled"`
			AnomalyRateLimits     *json.RawMessage `json:"anomaly_rate_limits"`
		}

		if !decodeJSONBody(w, r, &body) {
			return
		}

		req := landing_page_react_vite_v1.UpdateStripeSettingsRequest{
			PublishableKey:        body.PublishableKey,
			SecretKey:             body.SecretKey,
			WebhookSecret:         body.WebhookSecret,
			DashboardUrl:          body.DashboardURL,
			AnomalyWebhookUrl:     body.AnomalyWebhookURL,
			AnomalyWebhookEnabled: body.AnomalyWebhookEnabled,
		}

		// Accept rate limits as either a JSON object (inline) or a JSON string
		// (pre-serialised). Normalise to a compact JSON object string for storage.
		var rateLimitsStr *string
		if body.AnomalyRateLimits != nil {
			raw := strings.TrimSpace(string(*body.AnomalyRateLimits))
			if raw == "" || raw == "null" {
				empty := ""
				rateLimitsStr = &empty
			} else {
				// Validate shape: must be an object mapping type -> {burst, refill_seconds}.
				var asObject map[string]struct {
					Burst         int `json:"burst"`
					RefillSeconds int `json:"refill_seconds"`
				}
				if err := json.Unmarshal(*body.AnomalyRateLimits, &asObject); err != nil {
					// Try parsing as a JSON string that itself contains an object.
					var asString string
					if err2 := json.Unmarshal(*body.AnomalyRateLimits, &asString); err2 == nil {
						if err3 := json.Unmarshal([]byte(asString), &asObject); err3 != nil {
							writeJSONError(w, http.StatusBadRequest, "anomaly_rate_limits must be a JSON object mapping types to {burst, refill_seconds}", ApiErrorTypeValidation)
							return
						}
					} else {
						writeJSONError(w, http.StatusBadRequest, "anomaly_rate_limits must be a JSON object mapping types to {burst, refill_seconds}", ApiErrorTypeValidation)
						return
					}
				}
				for key, override := range asObject {
					if strings.TrimSpace(key) == "" || override.Burst < 0 || override.RefillSeconds < 0 {
						writeJSONError(w, http.StatusBadRequest, "anomaly_rate_limits entries require non-empty type, non-negative burst and refill_seconds", ApiErrorTypeValidation)
						return
					}
				}
				normalised, err := json.Marshal(asObject)
				if err != nil {
					writeJSONError(w, http.StatusBadRequest, "anomaly_rate_limits could not be serialised", ApiErrorTypeValidation)
					return
				}
				s := string(normalised)
				rateLimitsStr = &s
				req.AnomalyRateLimits = &s
			}
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
		req.AnomalyWebhookUrl = normalize(req.AnomalyWebhookUrl)

		if req.PublishableKey != nil && *req.PublishableKey != "" {
			if !strings.HasPrefix(*req.PublishableKey, "pk_") {
				writeJSONError(w, http.StatusBadRequest, "Publishable key must start with pk_", ApiErrorTypeValidation)
				return
			}
		}

		if req.SecretKey != nil && *req.SecretKey != "" {
			if !strings.HasPrefix(*req.SecretKey, "sk_") && !strings.HasPrefix(*req.SecretKey, "rk_") {
				writeJSONError(w, http.StatusBadRequest, "Restricted key must start with sk_ or rk_", ApiErrorTypeValidation)
				return
			}
		}

		if req.WebhookSecret != nil && *req.WebhookSecret != "" {
			if !strings.HasPrefix(*req.WebhookSecret, "whsec_") {
				writeJSONError(w, http.StatusBadRequest, "Webhook secret must start with whsec_", ApiErrorTypeValidation)
				return
			}
		}

		if req.DashboardUrl != nil && *req.DashboardUrl != "" {
			normalizedURL, err := ValidateURL(*req.DashboardUrl)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "Invalid dashboard_url format", ApiErrorTypeValidation)
				return
			}
			req.DashboardUrl = &normalizedURL
		}

		if req.AnomalyWebhookUrl != nil && *req.AnomalyWebhookUrl != "" {
			normalizedURL, err := ValidateURL(*req.AnomalyWebhookUrl)
			if err != nil {
				writeJSONError(w, http.StatusBadRequest, "Invalid anomaly_webhook_url format", ApiErrorTypeValidation)
				return
			}
			if !strings.HasPrefix(strings.ToLower(normalizedURL), "https://") {
				writeJSONError(w, http.StatusBadRequest, "anomaly_webhook_url must use https://", ApiErrorTypeValidation)
				return
			}
			req.AnomalyWebhookUrl = &normalizedURL
		}

		// Enforce: enabling dispatch requires a URL (from the request or the saved record).
		if req.AnomalyWebhookEnabled != nil && *req.AnomalyWebhookEnabled {
			haveURLInReq := req.AnomalyWebhookUrl != nil && *req.AnomalyWebhookUrl != ""
			if !haveURLInReq {
				existing, _ := paymentService.GetStripeSettings(r.Context())
				if existing == nil || strings.TrimSpace(existing.AnomalyWebhookUrl) == "" {
					writeJSONError(w, http.StatusBadRequest, "anomaly_webhook_enabled=true requires anomaly_webhook_url", ApiErrorTypeValidation)
					return
				}
			}
		}

		if (req.PublishableKey == nil || *req.PublishableKey == "") &&
			(req.SecretKey == nil || *req.SecretKey == "") &&
			(req.WebhookSecret == nil || *req.WebhookSecret == "") &&
			(req.DashboardUrl == nil || *req.DashboardUrl == "") &&
			req.AnomalyWebhookUrl == nil &&
			req.AnomalyWebhookEnabled == nil &&
			rateLimitsStr == nil {
			writeJSONError(w, http.StatusBadRequest, "At least one field is required", ApiErrorTypeValidation)
			return
		}

		record, err := paymentService.SaveStripeSettings(r.Context(), StripeSettingsInput{
			PublishableKey:        req.PublishableKey,
			SecretKey:             req.SecretKey,
			WebhookSecret:         req.WebhookSecret,
			DashboardURL:          req.DashboardUrl,
			AnomalyWebhookURL:     req.AnomalyWebhookUrl,
			AnomalyWebhookEnabled: req.AnomalyWebhookEnabled,
			AnomalyRateLimits:     rateLimitsStr,
		})
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to save Stripe settings", ApiErrorTypeServerError)
			return
		}
		hasPublishable := record != nil && strings.TrimSpace(record.PublishableKey) != ""
		hasSecret := record != nil && strings.TrimSpace(record.SecretKey) != ""
		hasWebhook := record != nil && strings.TrimSpace(record.WebhookSecret) != ""
		hasAnomalyURL := record != nil && strings.TrimSpace(record.AnomalyWebhookUrl) != ""
		// Redact secrets before responding.
		if record != nil {
			record.PublishableKey = ""
			record.SecretKey = ""
			record.WebhookSecret = ""
			record.AnomalyWebhookUrl = ""
			record.AnomalyWebhookUrlSet = hasAnomalyURL
		}

		if err := stripeService.RefreshConfig(r.Context()); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to refresh Stripe runtime config", ApiErrorTypeServerError)
			return
		}
		if anomalyService != nil {
			if err := anomalyService.RefreshConfig(r.Context()); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to refresh anomaly dispatch config", ApiErrorTypeServerError)
				return
			}
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
