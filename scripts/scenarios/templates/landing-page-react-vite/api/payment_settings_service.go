package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// PaymentSettingsService manages Stripe configuration stored by admins.
type PaymentSettingsService struct {
	db *sql.DB
}

// StripeSettingsRecord stores persisted Stripe credentials.
type StripeSettingsRecord struct {
	PublishableKey string
	SecretKey      string
	WebhookSecret  string
	DashboardURL   string
	UpdatedAt      time.Time
}

// StripeSettingsInput captures optional fields for upserts.
type StripeSettingsInput struct {
	PublishableKey *string
	SecretKey      *string
	WebhookSecret  *string
	DashboardURL   *string
}

func NewPaymentSettingsService(db *sql.DB) *PaymentSettingsService {
	return &PaymentSettingsService{db: db}
}

// GetStripeSettings returns the latest persisted Stripe configuration.
func (s *PaymentSettingsService) GetStripeSettings(ctx context.Context) (*StripeSettingsRecord, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT publishable_key, secret_key, webhook_secret, dashboard_url, updated_at
		FROM payment_settings
		WHERE id = 1
	`)

	var record StripeSettingsRecord
	var publishable, secret, webhook, dashboard sql.NullString
	if err := row.Scan(&publishable, &secret, &webhook, &dashboard, &record.UpdatedAt); err != nil {
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
		record.DashboardURL = dashboard.String
	}

	return &record, nil
}

// SaveStripeSettings persists the provided fields and returns the resulting record.
func (s *PaymentSettingsService) SaveStripeSettings(ctx context.Context, input StripeSettingsInput) (*StripeSettingsRecord, error) {
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

	updateField := func(existing string, incoming *string) string {
		if incoming == nil {
			return existing
		}
		return *incoming
	}

	pub := normalize(input.PublishableKey)
	sec := normalize(input.SecretKey)
	webhook := normalize(input.WebhookSecret)
	dashboard := normalize(input.DashboardURL)

	if current == nil {
		current = &StripeSettingsRecord{}
	}

	nextPublishable := updateField(current.PublishableKey, pub)
	nextSecret := updateField(current.SecretKey, sec)
	nextWebhook := updateField(current.WebhookSecret, webhook)
	nextDashboard := updateField(current.DashboardURL, dashboard)

	row := s.db.QueryRowContext(ctx, `
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, dashboard_url, updated_at)
		VALUES (1, $1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE SET
			publishable_key = EXCLUDED.publishable_key,
			secret_key = EXCLUDED.secret_key,
			webhook_secret = EXCLUDED.webhook_secret,
			dashboard_url = EXCLUDED.dashboard_url,
			updated_at = NOW()
		RETURNING publishable_key, secret_key, webhook_secret, dashboard_url, updated_at
	`, nextPublishable, nextSecret, nextWebhook, nextDashboard)

	var record StripeSettingsRecord
	if err := row.Scan(&record.PublishableKey, &record.SecretKey, &record.WebhookSecret, &record.DashboardURL, &record.UpdatedAt); err != nil {
		return nil, fmt.Errorf("save stripe settings: %w", err)
	}

	return &record, nil
}
