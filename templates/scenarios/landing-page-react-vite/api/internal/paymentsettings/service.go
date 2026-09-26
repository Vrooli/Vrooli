// Package paymentsettings manages the singleton Stripe credential record that
// admins configure at runtime. Its domain type is the shared protobuf
// StripeSettings message; the handler in handlers/payments is a thin adapter.
package paymentsettings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"
)

// Service manages Stripe configuration stored by admins.
type Service struct {
	db *sql.DB
}

// Input captures optional fields for an upsert; a nil field is preserved.
type Input struct {
	PublishableKey *string
	SecretKey      *string
	WebhookSecret  *string
	DashboardURL   *string
}

// NewService constructs the payment settings Service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// GetStripeSettings returns the latest persisted configuration, or nil when no
// row exists yet.
func (s *Service) GetStripeSettings(ctx context.Context) (*landingv1.StripeSettings, error) {
	var publishable, secret, webhook, dashboard sql.NullString
	var updatedAt time.Time
	err := s.db.QueryRowContext(ctx, `
		SELECT publishable_key, secret_key, webhook_secret, dashboard_url, updated_at
		FROM payment_settings WHERE id = 1`).Scan(&publishable, &secret, &webhook, &dashboard, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	record := &landingv1.StripeSettings{
		PublishableKey: publishable.String,
		SecretKey:      secret.String,
		WebhookSecret:  webhook.String,
	}
	if dashboard.Valid {
		record.DashboardUrl = &dashboard.String
	}
	if !updatedAt.IsZero() {
		record.UpdatedAt = timestamppb.New(updatedAt)
	}
	return record, nil
}

// SaveStripeSettings persists the provided fields (preserving unset ones) and
// returns the resulting record.
func (s *Service) SaveStripeSettings(ctx context.Context, input Input) (*landingv1.StripeSettings, error) {
	current, err := s.GetStripeSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("load current stripe settings: %w", err)
	}
	if current == nil {
		current = &landingv1.StripeSettings{}
	}

	next := func(existing string, incoming *string) string {
		if incoming == nil {
			return existing
		}
		return strings.TrimSpace(*incoming)
	}

	nextPublishable := next(current.PublishableKey, input.PublishableKey)
	nextSecret := next(current.SecretKey, input.SecretKey)
	nextWebhook := next(current.WebhookSecret, input.WebhookSecret)
	nextDashboard := next(derefString(current.DashboardUrl), input.DashboardURL)

	var publishable, secret, webhook, dashboard sql.NullString
	var updatedAt time.Time
	err = s.db.QueryRowContext(ctx, `
		INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, dashboard_url, updated_at)
		VALUES (1, $1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE SET
			publishable_key = EXCLUDED.publishable_key,
			secret_key = EXCLUDED.secret_key,
			webhook_secret = EXCLUDED.webhook_secret,
			dashboard_url = EXCLUDED.dashboard_url,
			updated_at = NOW()
		RETURNING publishable_key, secret_key, webhook_secret, dashboard_url, updated_at`,
		nextPublishable, nextSecret, nextWebhook, nextDashboard,
	).Scan(&publishable, &secret, &webhook, &dashboard, &updatedAt)
	if err != nil {
		return nil, fmt.Errorf("save stripe settings: %w", err)
	}

	record := &landingv1.StripeSettings{
		PublishableKey: publishable.String,
		SecretKey:      secret.String,
		WebhookSecret:  webhook.String,
		UpdatedAt:      timestamppb.New(updatedAt),
	}
	if dashboard.Valid {
		record.DashboardUrl = &dashboard.String
	}
	return record, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
