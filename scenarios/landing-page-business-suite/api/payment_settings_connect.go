package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/proto"
)

// stripeSettingsConnectHandler keeps secret-bearing settings operations behind
// the generated admin-only Connect service.
type stripeSettingsConnectHandler struct {
	payment *PaymentSettingsService
	stripe  *StripeService
	anomaly *PaymentAnomalyService
}

func (h stripeSettingsConnectHandler) GetStripeSettings(ctx context.Context, _ *connect.Request[lpbsv1.GetStripeSettingsRequest]) (*connect.Response[lpbsv1.GetStripeSettingsResponse], error) {
	var record *lpbsv1.StripeSettings
	if h.payment != nil {
		loaded, err := h.payment.GetStripeSettings(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Stripe settings: %w", err))
		}
		record = loaded
	}
	responseRecord := redactStripeSettings(record)
	snapshot := proto.Clone(h.stripe.ConfigSnapshot()).(*lpbsv1.StripeConfigSnapshot)
	if record != nil {
		snapshot.PublishableKeySet = snapshot.PublishableKeySet || strings.TrimSpace(record.PublishableKey) != ""
		snapshot.SecretKeySet = snapshot.SecretKeySet || strings.TrimSpace(record.SecretKey) != ""
		snapshot.WebhookSecretSet = snapshot.WebhookSecretSet || strings.TrimSpace(record.WebhookSecret) != ""
	}
	return connect.NewResponse(&lpbsv1.GetStripeSettingsResponse{Settings: responseRecord, Snapshot: snapshot}), nil
}

func redactStripeSettings(record *lpbsv1.StripeSettings) *lpbsv1.StripeSettings {
	if record == nil {
		return nil
	}
	redacted := proto.Clone(record).(*lpbsv1.StripeSettings)
	redacted.AnomalyWebhookUrlSet = strings.TrimSpace(record.AnomalyWebhookUrl) != ""
	redacted.PublishableKey, redacted.SecretKey, redacted.WebhookSecret, redacted.AnomalyWebhookUrl = "", "", "", ""
	return redacted
}

func (h stripeSettingsConnectHandler) RevealStripeSecret(ctx context.Context, request *connect.Request[lpbsv1.RevealStripeSecretRequest]) (*connect.Response[lpbsv1.RevealStripeSecretResponse], error) {
	field := request.Msg.GetField()
	if field != "secret_key" && field != "webhook_secret" && field != "publishable_key" && field != "anomaly_webhook_url" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid Stripe settings field"))
	}
	if field == "anomaly_webhook_url" {
		if h.payment == nil {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no value set for %s", field))
		}
		record, err := h.payment.GetStripeSettings(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Stripe settings: %w", err))
		}
		if record == nil || strings.TrimSpace(record.AnomalyWebhookUrl) == "" {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no value set for %s", field))
		}
		return connect.NewResponse(&lpbsv1.RevealStripeSecretResponse{Field: field, Value: record.AnomalyWebhookUrl}), nil
	}
	value, ok := h.stripe.GetSecretValue(field)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no value set for %s", field))
	}
	return connect.NewResponse(&lpbsv1.RevealStripeSecretResponse{Field: field, Value: value}), nil
}

// UpdateStripeSettings validates, persists, and activates a partial Stripe
// configuration update. It deliberately keeps validation next to the typed
// transport so all callers receive the same contract and errors.
func (h stripeSettingsConnectHandler) UpdateStripeSettings(ctx context.Context, request *connect.Request[lpbsv1.UpdateStripeSettingsRequest]) (*connect.Response[lpbsv1.UpdateStripeSettingsResponse], error) {
	if h.payment == nil || h.stripe == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("Stripe settings service is unavailable"))
	}
	req := request.Msg
	normalize := func(value *string) *string {
		if value == nil {
			return nil
		}
		trimmed := strings.TrimSpace(*value)
		return &trimmed
	}
	pub := normalize(req.PublishableKey)
	secret := normalize(req.SecretKey)
	webhook := normalize(req.WebhookSecret)
	dashboard := normalize(req.DashboardUrl)
	anomalyURL := normalize(req.AnomalyWebhookUrl)
	rateLimits, err := normalizeAnomalyRateLimits(req.AnomalyRateLimits)
	if err != nil {
		return nil, validationError(err.Error())
	}

	if pub != nil && *pub != "" && !strings.HasPrefix(*pub, "pk_") {
		return nil, validationError("publishable key must start with pk_")
	}
	if secret != nil && *secret != "" && !strings.HasPrefix(*secret, "sk_") && !strings.HasPrefix(*secret, "rk_") {
		return nil, validationError("restricted key must start with sk_ or rk_")
	}
	if webhook != nil && *webhook != "" && !strings.HasPrefix(*webhook, "whsec_") {
		return nil, validationError("webhook secret must start with whsec_")
	}
	if dashboard != nil && *dashboard != "" {
		normalized, err := ValidateURL(*dashboard)
		if err != nil {
			return nil, validationError("invalid dashboard_url format")
		}
		dashboard = &normalized
	}
	if anomalyURL != nil && *anomalyURL != "" {
		normalized, err := ValidateURL(*anomalyURL)
		if err != nil || !strings.HasPrefix(strings.ToLower(normalized), "https://") {
			return nil, validationError("anomaly_webhook_url must use a valid https:// URL")
		}
		anomalyURL = &normalized
	}
	if req.AnomalyWebhookEnabled != nil && *req.AnomalyWebhookEnabled && (anomalyURL == nil || *anomalyURL == "") {
		existing, err := h.payment.GetStripeSettings(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Stripe settings: %w", err))
		}
		if existing == nil || strings.TrimSpace(existing.AnomalyWebhookUrl) == "" {
			return nil, validationError("anomaly_webhook_enabled=true requires anomaly_webhook_url")
		}
	}
	if noStripeSettingsChange(pub, secret, webhook, dashboard, anomalyURL, req.AnomalyWebhookEnabled, rateLimits) {
		return nil, validationError("at least one field is required")
	}

	record, err := h.payment.SaveStripeSettings(ctx, StripeSettingsInput{PublishableKey: pub, SecretKey: secret, WebhookSecret: webhook, DashboardURL: dashboard, AnomalyWebhookURL: anomalyURL, AnomalyWebhookEnabled: req.AnomalyWebhookEnabled, AnomalyRateLimits: rateLimits})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save Stripe settings: %w", err))
	}
	if err := h.stripe.RefreshConfig(ctx); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refresh Stripe runtime config: %w", err))
	}
	if h.anomaly != nil {
		if err := h.anomaly.RefreshConfig(ctx); err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("refresh anomaly dispatch config: %w", err))
		}
	}
	snapshot := h.stripe.ConfigSnapshot()
	if record != nil {
		snapshot.PublishableKeySet = snapshot.PublishableKeySet || strings.TrimSpace(record.PublishableKey) != ""
		snapshot.SecretKeySet = snapshot.SecretKeySet || strings.TrimSpace(record.SecretKey) != ""
		snapshot.WebhookSecretSet = snapshot.WebhookSecretSet || strings.TrimSpace(record.WebhookSecret) != ""
	}
	return connect.NewResponse(&lpbsv1.UpdateStripeSettingsResponse{Settings: redactStripeSettings(record), Snapshot: snapshot}), nil
}

func normalizeAnomalyRateLimits(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return &raw, nil
	}
	var limits map[string]struct {
		Burst         int `json:"burst"`
		RefillSeconds int `json:"refill_seconds"`
	}
	if err := json.Unmarshal([]byte(raw), &limits); err != nil {
		return nil, fmt.Errorf("anomaly_rate_limits must be a JSON object mapping types to {burst, refill_seconds}")
	}
	for key, override := range limits {
		if strings.TrimSpace(key) == "" || override.Burst < 0 || override.RefillSeconds < 0 {
			return nil, fmt.Errorf("anomaly_rate_limits entries require non-empty type, non-negative burst and refill_seconds")
		}
	}
	normalized, err := json.Marshal(limits)
	if err != nil {
		return nil, fmt.Errorf("anomaly_rate_limits could not be serialised")
	}
	result := string(normalized)
	return &result, nil
}

func noStripeSettingsChange(pub, secret, webhook, dashboard, anomalyURL *string, anomalyEnabled *bool, rateLimits *string) bool {
	for _, value := range []*string{pub, secret, webhook, dashboard, anomalyURL} {
		if value != nil && *value != "" {
			return false
		}
	}
	return anomalyEnabled == nil && rateLimits == nil
}

func validationError(message string) error {
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("%s", message))
}

func registerStripeSettingsConnectRoutes(router *mux.Router, payment *PaymentSettingsService, stripe *StripeService, anomaly *PaymentAnomalyService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewStripeSettingsServiceHandler(stripeSettingsConnectHandler{payment: payment, stripe: stripe, anomaly: anomaly})
	for _, path := range []string{lpbsconnect.StripeSettingsServiceGetStripeSettingsProcedure, lpbsconnect.StripeSettingsServiceUpdateStripeSettingsProcedure, lpbsconnect.StripeSettingsServiceRevealStripeSecretProcedure} {
		router.Handle(path, requireAdmin(http.HandlerFunc(service.ServeHTTP))).Methods(http.MethodPost)
	}
}
