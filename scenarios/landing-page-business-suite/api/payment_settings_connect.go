package main

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1"
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
