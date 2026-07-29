package commerce

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	landing_page_business_suite_v1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PaymentSettingsStore is the context-aware persistence contract for Stripe
// and payment-anomaly configuration.
//
// seam: PaymentSettingsStore keeps sensitive payment configuration independent
// of a concrete pool and preserves request-scoped test isolation.
type PaymentSettingsStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// PaymentSettingsService manages Stripe configuration stored by admins.
type PaymentSettingsService struct {
	db PaymentSettingsStore
}

// StripeSettingsInput captures optional fields for upserts.
type StripeSettingsInput struct {
	PublishableKey        *string
	SecretKey             *string
	WebhookSecret         *string
	DashboardURL          *string
	AnomalyWebhookURL     *string
	AnomalyWebhookEnabled *bool
	AnomalyRateLimits     *string
}

func NewPaymentSettingsService(db PaymentSettingsStore) *PaymentSettingsService {
	return &PaymentSettingsService{db: db}
}

// GetStripeSettings returns the latest persisted Stripe configuration.
func (s *PaymentSettingsService) GetStripeSettings(ctx context.Context) (*landing_page_business_suite_v1.StripeSettings, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT publishable_key, secret_key, webhook_secret, dashboard_url,
			anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at
		FROM payment_settings
		WHERE id = 1
	`)

	record := &landing_page_business_suite_v1.StripeSettings{}
	var publishable, secret, webhook, dashboard, anomalyURL, anomalyLimits sql.NullString
	var anomalyEnabled sql.NullBool
	var updatedAt time.Time
	if err := row.Scan(&publishable, &secret, &webhook, &dashboard, &anomalyURL, &anomalyEnabled, &anomalyLimits, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if publishable.Valid {
		record.PublishableKey = publishable.String
	}
	if secret.Valid {
		record.SecretKey = secret.String
	}
	if webhook.Valid {
		record.WebhookSecret = webhook.String
	}
	if dashboard.Valid {
		record.DashboardUrl = proto.String(dashboard.String)
	}
	if anomalyURL.Valid {
		record.AnomalyWebhookUrl = anomalyURL.String
	}
	if anomalyEnabled.Valid {
		record.AnomalyWebhookEnabled = anomalyEnabled.Bool
	}
	if anomalyLimits.Valid {
		record.AnomalyRateLimits = anomalyLimits.String
	}

	if !updatedAt.IsZero() {
		record.UpdatedAt = timestamppb.New(updatedAt)
	}

	return record, nil
}

// SaveStripeSettings persists the provided fields and returns the resulting record.
func (s *PaymentSettingsService) SaveStripeSettings(ctx context.Context, input StripeSettingsInput) (*landing_page_business_suite_v1.StripeSettings, error) {
	current, err := s.GetStripeSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load current stripe settings: %w", err)
	}

	normalize := func(value *string) *string {
		if value == nil {
			return nil
		}
		trimmed := strings.TrimSpace(*value)
		return &trimmed
	}

	updateStringField := func(existing string, incoming *string) string {
		if incoming == nil {
			return existing
		}
		return *incoming
	}

	updateOptionalField := func(existing *string, incoming *string) *string {
		if incoming == nil {
			return existing
		}
		return proto.String(*incoming)
	}

	pub := normalize(input.PublishableKey)
	sec := normalize(input.SecretKey)
	webhook := normalize(input.WebhookSecret)
	dashboard := normalize(input.DashboardURL)
	anomalyURL := normalize(input.AnomalyWebhookURL)
	anomalyLimits := normalize(input.AnomalyRateLimits)

	if current == nil {
		current = &landing_page_business_suite_v1.StripeSettings{}
	}

	nextPublishable := updateStringField(current.PublishableKey, pub)
	nextSecret := updateStringField(current.SecretKey, sec)
	nextWebhook := updateStringField(current.WebhookSecret, webhook)
	nextDashboard := updateOptionalField(current.DashboardUrl, dashboard)
	nextAnomalyURL := updateStringField(current.AnomalyWebhookUrl, anomalyURL)
	nextAnomalyEnabled := current.AnomalyWebhookEnabled
	if input.AnomalyWebhookEnabled != nil {
		nextAnomalyEnabled = *input.AnomalyWebhookEnabled
	}
	nextAnomalyLimits := updateStringField(current.AnomalyRateLimits, anomalyLimits)

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO payment_settings (
			id, publishable_key, secret_key, webhook_secret, dashboard_url,
			anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at
		)
		VALUES (1, $1, $2, $3, $4, $5, $6, $7::jsonb, NOW())
		ON CONFLICT (id) DO UPDATE SET
			publishable_key = EXCLUDED.publishable_key,
			secret_key = EXCLUDED.secret_key,
			webhook_secret = EXCLUDED.webhook_secret,
			dashboard_url = EXCLUDED.dashboard_url,
			anomaly_webhook_url = EXCLUDED.anomaly_webhook_url,
			anomaly_webhook_enabled = EXCLUDED.anomaly_webhook_enabled,
			anomaly_rate_limits = EXCLUDED.anomaly_rate_limits,
			updated_at = NOW()
		RETURNING publishable_key, secret_key, webhook_secret, dashboard_url,
			anomaly_webhook_url, anomaly_webhook_enabled, anomaly_rate_limits, updated_at
	`, nextPublishable, nextSecret, nextWebhook, nextDashboard,
		nextAnomalyURL, nextAnomalyEnabled, jsonOrEmptyObject(nextAnomalyLimits))

	record := &landing_page_business_suite_v1.StripeSettings{}
	var anomalyURLOut, anomalyLimitsOut sql.NullString
	var anomalyEnabledOut sql.NullBool
	var updatedAt time.Time
	if err := row.Scan(
		&record.PublishableKey, &record.SecretKey, &record.WebhookSecret, &record.DashboardUrl,
		&anomalyURLOut, &anomalyEnabledOut, &anomalyLimitsOut, &updatedAt,
	); err != nil {
		return nil, fmt.Errorf("save stripe settings: %w", err)
	}

	if anomalyURLOut.Valid {
		record.AnomalyWebhookUrl = anomalyURLOut.String
	}
	if anomalyEnabledOut.Valid {
		record.AnomalyWebhookEnabled = anomalyEnabledOut.Bool
	}
	if anomalyLimitsOut.Valid {
		record.AnomalyRateLimits = anomalyLimitsOut.String
	}
	record.UpdatedAt = timestamppb.New(updatedAt)

	return record, nil
}

// jsonOrEmptyObject returns a valid JSON object string, falling back to "{}"
// when the input is empty or whitespace. Used to coerce nullable text into
// jsonb column input.
func jsonOrEmptyObject(s string) string {
	if strings.TrimSpace(s) == "" {
		return "{}"
	}
	return s
}
