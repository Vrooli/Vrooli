package billing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	lpbsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/landing_page_business_suite_v1connect"
	"google.golang.org/protobuf/proto"
	"landing-page-business-suite-api/internal/commerce"
)

// stripeSettingsConnectHandler keeps secret-bearing settings operations behind
// the generated admin-only Connect service.
// StripeSettingsRuntime is the narrow runtime seam needed by credential
// settings. API-root composition supplies StripeService; tests supply the same
// concrete service or a focused fake without coupling this handler to root.
type StripeSettingsRuntime interface {
	ConfigSnapshot() *lpbsv1.StripeConfigSnapshot
	GetSecretValue(field string) (string, bool)
	RefreshConfig(context.Context) error
}

// StripeSettingsConnectHandler keeps secret-bearing settings operations behind
// the generated administrator-only Connect service.
type StripeSettingsConnectHandler struct {
	payment *commerce.PaymentSettingsService
	stripe  StripeSettingsRuntime
	anomaly *commerce.PaymentAnomalyService
}

func NewStripeSettingsConnectHandler(payment *commerce.PaymentSettingsService, stripe StripeSettingsRuntime, anomaly *commerce.PaymentAnomalyService) StripeSettingsConnectHandler {
	return StripeSettingsConnectHandler{payment: payment, stripe: stripe, anomaly: anomaly}
}

type stripeSettingsUpdate struct {
	input                 commerce.StripeSettingsInput
	anomalyWebhookEnabled *bool
	anomalyRateLimits     *string
}

func (h StripeSettingsConnectHandler) GetStripeSettings(ctx context.Context, _ *connect.Request[lpbsv1.GetStripeSettingsRequest]) (*connect.Response[lpbsv1.GetStripeSettingsResponse], error) {
	var record *lpbsv1.StripeSettings
	if h.payment != nil {
		loaded, err := h.payment.GetStripeSettings(ctx)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("load Stripe settings: %w", err))
		}
		record = loaded
	}
	responseRecord := RedactStripeSettings(record)
	snapshot := proto.Clone(h.stripe.ConfigSnapshot()).(*lpbsv1.StripeConfigSnapshot)
	if record != nil {
		snapshot.PublishableKeySet = snapshot.PublishableKeySet || strings.TrimSpace(record.PublishableKey) != ""
		snapshot.SecretKeySet = snapshot.SecretKeySet || strings.TrimSpace(record.SecretKey) != ""
		snapshot.WebhookSecretSet = snapshot.WebhookSecretSet || strings.TrimSpace(record.WebhookSecret) != ""
	}
	return connect.NewResponse(&lpbsv1.GetStripeSettingsResponse{Settings: responseRecord, Snapshot: snapshot}), nil
}

func RedactStripeSettings(record *lpbsv1.StripeSettings) *lpbsv1.StripeSettings {
	if record == nil {
		return nil
	}
	redacted := proto.Clone(record).(*lpbsv1.StripeSettings)
	redacted.AnomalyWebhookUrlSet = strings.TrimSpace(record.AnomalyWebhookUrl) != ""
	redacted.PublishableKey, redacted.SecretKey, redacted.WebhookSecret, redacted.AnomalyWebhookUrl = "", "", "", ""
	return redacted
}

func (h StripeSettingsConnectHandler) RevealStripeSecret(ctx context.Context, request *connect.Request[lpbsv1.RevealStripeSecretRequest]) (*connect.Response[lpbsv1.RevealStripeSecretResponse], error) {
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
func (h StripeSettingsConnectHandler) UpdateStripeSettings(ctx context.Context, request *connect.Request[lpbsv1.UpdateStripeSettingsRequest]) (*connect.Response[lpbsv1.UpdateStripeSettingsResponse], error) {
	if h.payment == nil || h.stripe == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("Stripe settings service is unavailable"))
	}
	update, err := h.validatedStripeSettingsUpdate(ctx, request.Msg)
	if err != nil {
		return nil, err
	}

	record, err := h.payment.SaveStripeSettings(ctx, update.input)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("save Stripe settings: %w", err))
	}
	if err := h.refreshRuntimeConfig(ctx); err != nil {
		return nil, err
	}
	snapshot := h.stripe.ConfigSnapshot()
	if record != nil {
		snapshot.PublishableKeySet = snapshot.PublishableKeySet || strings.TrimSpace(record.PublishableKey) != ""
		snapshot.SecretKeySet = snapshot.SecretKeySet || strings.TrimSpace(record.SecretKey) != ""
		snapshot.WebhookSecretSet = snapshot.WebhookSecretSet || strings.TrimSpace(record.WebhookSecret) != ""
	}
	return connect.NewResponse(&lpbsv1.UpdateStripeSettingsResponse{Settings: RedactStripeSettings(record), Snapshot: snapshot}), nil
}

func (h StripeSettingsConnectHandler) validatedStripeSettingsUpdate(ctx context.Context, req *lpbsv1.UpdateStripeSettingsRequest) (stripeSettingsUpdate, error) {
	update, err := normalizeStripeSettingsUpdate(req)
	if err != nil {
		return stripeSettingsUpdate{}, validationError(err.Error())
	}
	if err := h.validateAnomalyWebhookEnablement(ctx, update); err != nil {
		return stripeSettingsUpdate{}, err
	}
	if update.hasNoChange() {
		return stripeSettingsUpdate{}, validationError("at least one field is required")
	}
	return update, nil
}

func normalizeStripeSettingsUpdate(req *lpbsv1.UpdateStripeSettingsRequest) (stripeSettingsUpdate, error) {
	update := stripeSettingsUpdate{
		input: commerce.StripeSettingsInput{
			PublishableKey:        normalizeStripeSetting(req.PublishableKey),
			SecretKey:             normalizeStripeSetting(req.SecretKey),
			WebhookSecret:         normalizeStripeSetting(req.WebhookSecret),
			DashboardURL:          normalizeStripeSetting(req.DashboardUrl),
			AnomalyWebhookURL:     normalizeStripeSetting(req.AnomalyWebhookUrl),
			AnomalyWebhookEnabled: req.AnomalyWebhookEnabled,
		},
		anomalyWebhookEnabled: req.AnomalyWebhookEnabled,
	}
	var err error
	if update.anomalyRateLimits, err = normalizeAnomalyRateLimits(req.AnomalyRateLimits); err != nil {
		return stripeSettingsUpdate{}, err
	}
	update.input.AnomalyRateLimits = update.anomalyRateLimits
	if err := validateStripeSettingPrefixes(update.input); err != nil {
		return stripeSettingsUpdate{}, err
	}
	if update.input.DashboardURL, err = normalizeOptionalURL(update.input.DashboardURL, false); err != nil {
		return stripeSettingsUpdate{}, err
	}
	if update.input.AnomalyWebhookURL, err = normalizeOptionalURL(update.input.AnomalyWebhookURL, true); err != nil {
		return stripeSettingsUpdate{}, err
	}
	return update, nil
}

func normalizeStripeSetting(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func validateStripeSettingPrefixes(input commerce.StripeSettingsInput) error {
	for _, requirement := range []struct {
		value   *string
		valid   func(string) bool
		message string
	}{
		{input.PublishableKey, func(value string) bool { return strings.HasPrefix(value, "pk_") }, "publishable key must start with pk_"},
		{input.SecretKey, func(value string) bool { return strings.HasPrefix(value, "sk_") || strings.HasPrefix(value, "rk_") }, "restricted key must start with sk_ or rk_"},
		{input.WebhookSecret, func(value string) bool { return strings.HasPrefix(value, "whsec_") }, "webhook secret must start with whsec_"},
	} {
		if requirement.value != nil && *requirement.value != "" && !requirement.valid(*requirement.value) {
			return errors.New(requirement.message)
		}
	}
	return nil
}

func normalizeOptionalURL(value *string, requireHTTPS bool) (*string, error) {
	if value == nil || *value == "" {
		return value, nil
	}
	normalized, err := validateHTTPURL(*value)
	if err != nil {
		if requireHTTPS {
			return nil, fmt.Errorf("anomaly_webhook_url must use a valid https:// URL")
		}
		return nil, fmt.Errorf("invalid dashboard_url format")
	}
	if requireHTTPS && !strings.HasPrefix(strings.ToLower(normalized), "https://") {
		return nil, fmt.Errorf("anomaly_webhook_url must use a valid https:// URL")
	}
	return &normalized, nil
}

func (u stripeSettingsUpdate) hasNoChange() bool {
	return noStripeSettingsChange(u.input.PublishableKey, u.input.SecretKey, u.input.WebhookSecret, u.input.DashboardURL, u.input.AnomalyWebhookURL, u.anomalyWebhookEnabled, u.anomalyRateLimits)
}

func (h StripeSettingsConnectHandler) validateAnomalyWebhookEnablement(ctx context.Context, update stripeSettingsUpdate) error {
	if update.anomalyWebhookEnabled == nil || !*update.anomalyWebhookEnabled || (update.input.AnomalyWebhookURL != nil && *update.input.AnomalyWebhookURL != "") {
		return nil
	}
	existing, err := h.payment.GetStripeSettings(ctx)
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("load Stripe settings: %w", err))
	}
	if existing == nil || strings.TrimSpace(existing.AnomalyWebhookUrl) == "" {
		return validationError("anomaly_webhook_enabled=true requires anomaly_webhook_url")
	}
	return nil
}

func (h StripeSettingsConnectHandler) refreshRuntimeConfig(ctx context.Context) error {
	if err := h.stripe.RefreshConfig(ctx); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("refresh Stripe runtime config: %w", err))
	}
	if h.anomaly != nil {
		if err := h.anomaly.RefreshConfig(ctx); err != nil {
			return connect.NewError(connect.CodeInternal, fmt.Errorf("refresh anomaly dispatch config: %w", err))
		}
	}
	return nil
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

// RegisterStripeSettingsConnectRoutes mounts every secret-bearing settings
// procedure behind the administrator boundary.
func RegisterStripeSettingsConnectRoutes(router *mux.Router, payment *commerce.PaymentSettingsService, stripe StripeSettingsRuntime, anomaly *commerce.PaymentAnomalyService, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	NewStripeSettingsConnectHandler(payment, stripe, anomaly).RegisterRoutes(router, requireAdmin)
}

// RegisterRoutes mounts this handler's complete generated procedure set.
func (h StripeSettingsConnectHandler) RegisterRoutes(router *mux.Router, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	_, service := lpbsconnect.NewStripeSettingsServiceHandler(h)
	for _, path := range []string{lpbsconnect.StripeSettingsServiceGetStripeSettingsProcedure, lpbsconnect.StripeSettingsServiceUpdateStripeSettingsProcedure, lpbsconnect.StripeSettingsServiceRevealStripeSecretProcedure} {
		router.Handle(path, requireAdmin(http.HandlerFunc(service.ServeHTTP))).Methods(http.MethodPost)
	}
}

func validateHTTPURL(raw string) (string, error) {
	normalized := strings.TrimSpace(raw)
	parsed, err := url.Parse(normalized)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", errors.New("invalid URL format")
	}
	return normalized, nil
}

var _ lpbsconnect.StripeSettingsServiceHandler = StripeSettingsConnectHandler{}
