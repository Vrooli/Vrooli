package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"landing-page-business-suite-api/internal/commerce"
	"landing-page-business-suite-api/internal/emaildelivery"
	"landing-page-business-suite-api/internal/opsalert"
)

func (s *Server) startAuthDeliveryAlerts() func() {
	evaluator := &commerce.AuthDeliveryAlertEvaluator{DB: s.db, Transport: opsalert.New()}
	ctx, cancel := context.WithCancel(context.Background())
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				var url string
				var enabled bool
				if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(anomaly_webhook_url,''), COALESCE(anomaly_webhook_enabled,FALSE) FROM payment_settings WHERE id = 1`).Scan(&url, &enabled); err == nil {
					_ = evaluator.Evaluate(ctx, url, enabled)
					if enabled && url != "" {
						_ = s.evaluateEmailDeliveryAlerts(ctx, url)
					}
				}
			}
		}
	}()
	return cancel
}

// evaluateEmailDeliveryAlerts turns durable delivery state into actionable,
// six-hour-bucketed operator alerts. It never includes credentials, tokens, or
// message bodies in the alert payload.
func (s *Server) evaluateEmailDeliveryAlerts(ctx context.Context, webhookURL string) error {
	if s == nil || s.db == nil || s.routedDB == nil || webhookURL == "" {
		return nil
	}
	transport := opsalert.New()
	var alerts []struct {
		typ     string
		details map[string]any
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM email_outbox WHERE priority >= 80 AND status = 'retry' AND last_error = 'no eligible provider' AND updated_at >= NOW() - INTERVAL '30 minutes'`).Scan(&count); err == nil && count > 0 {
		alerts = append(alerts, struct {
			typ     string
			details map[string]any
		}{"email_no_eligible_route", map[string]any{"messages": count}})
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM email_outbox WHERE purpose = 'signin' AND status IN ('pending','retry','unknown','expired') AND expires_at IS NOT NULL AND expires_at <= NOW() AND updated_at >= NOW() - INTERVAL '6 hours'`).Scan(&count); err == nil && count > 0 {
		alerts = append(alerts, struct {
			typ     string
			details map[string]any
		}{"email_signin_older_than_token", map[string]any{"messages": count}})
	}
	rows, err := s.routedDB.QueryContext(ctx, `SELECT provider_id, window_key, used_count, ceiling FROM email_quota_windows WHERE ceiling > 0 AND used_count::numeric / ceiling >= 0.8 AND window_ends_at > NOW()`)
	if err == nil {
		for rows.Next() {
			var provider, window string
			var used, ceiling int
			if rows.Scan(&provider, &window, &used, &ceiling) == nil {
				alerts = append(alerts, struct {
					typ     string
					details map[string]any
				}{"email_provider_quota_high", map[string]any{"provider": provider, "window": window, "used": used, "ceiling": ceiling}})
			}
		}
		rows.Close()
	}
	if s.mailDNSService != nil {
		if providers, loadErr := emaildelivery.LoadProviders(ctx, s.routedDB); loadErr == nil {
			for _, provider := range providers {
				requirements := providerMailDNSRequirements(provider)
				if len(requirements) == 0 {
					continue
				}
				report := s.mailDNSService.Verify(ctx, senderDomain(), requirements)
				for _, verdict := range report.Providers {
					if verdict.Status == maildns.Fail && provider.Enabled {
						alerts = append(alerts, struct {
							typ     string
							details map[string]any
						}{"email_provider_dns_unauthorized", map[string]any{"provider": provider.ID, "detail": verdict.Detail, "record": verdict.Record}})
					}
				}
			}
		}
	}
	for _, alert := range alerts {
		if err := s.dispatchEmailDeliveryAlert(ctx, transport, webhookURL, alert.typ, alert.details); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) dispatchEmailDeliveryAlert(ctx context.Context, transport *opsalert.Transport, webhookURL, alertType string, details map[string]any) error {
	body, err := json.Marshal(map[string]any{"type": alertType, "scenario": "landing-page-business-suite", "details": details})
	if err != nil {
		return err
	}
	window := time.Now().UTC().Truncate(providerAlertWindow)
	inserted, err := s.db.ExecContext(ctx, `INSERT INTO auth_alert_log (alert_type, window_start, details, dispatch_status, created_at) VALUES ($1,$2,$3,'pending',NOW()) ON CONFLICT (alert_type, window_start) DO NOTHING`, alertType, window, body)
	if err != nil {
		return err
	}
	if rows, _ := inserted.RowsAffected(); rows == 0 {
		return nil
	}
	if transport == nil {
		transport = opsalert.New()
	}
	if err := transport.Send(ctx, webhookURL, body); err != nil {
		_, _ = s.db.ExecContext(context.WithoutCancel(ctx), `UPDATE auth_alert_log SET dispatch_status='failed', dispatch_error=$1 WHERE alert_type=$2 AND window_start=$3`, err.Error(), alertType, window)
		return fmt.Errorf("send %s alert: %w", alertType, err)
	}
	_, _ = s.db.ExecContext(context.WithoutCancel(ctx), `UPDATE auth_alert_log SET dispatch_status='sent', dispatch_error=NULL WHERE alert_type=$1 AND window_start=$2`, alertType, window)
	return nil
}
