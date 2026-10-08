package commerce

import (
	"context"
	"encoding/json"
	"time"

	"landing-page-business-suite-api/internal/opsalert"
)

type AuthDeliveryAlertEvaluator struct {
	DB        PaymentAnomalyStore
	Transport *opsalert.Transport
	Now       func() time.Time
}

func (e *AuthDeliveryAlertEvaluator) Evaluate(ctx context.Context, webhookURL string, enabled bool) error {
	if e == nil || e.DB == nil || !enabled || webhookURL == "" {
		return nil
	}
	now := time.Now().UTC()
	if e.Now != nil {
		now = e.Now().UTC()
	}
	var total, failed int
	if err := e.DB.QueryRowContext(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE delivery_status = 'failed' OR provider_status IN ('bounce','dropped') OR (provider_status = 'deferred' AND provider_status_at < NOW() - INTERVAL '15 minutes')) FROM auth_tokens WHERE created_at >= NOW() - INTERVAL '30 minutes'`).Scan(&total, &failed); err != nil {
		return err
	}
	if total >= 3 && failed*2 >= total {
		return e.dispatch(ctx, webhookURL, "sign_in_delivery_degraded", map[string]any{"total": total, "failed": failed, "at": now})
	}
	var sent int
	if err := e.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_tokens WHERE created_at >= NOW() - INTERVAL '6 hours' AND delivery_status IN ('sent','pending')`).Scan(&sent); err != nil {
		return err
	}
	if sent > 0 {
		var events int
		if err := e.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_email_events WHERE received_at >= NOW() - INTERVAL '6 hours'`).Scan(&events); err != nil {
			return err
		}
		if events == 0 {
			return e.dispatch(ctx, webhookURL, "sign_in_delivery_webhook_silent", map[string]any{"sent": sent, "at": now})
		}
	}
	return nil
}

func (e *AuthDeliveryAlertEvaluator) dispatch(ctx context.Context, url, alertType string, details map[string]any) error {
	if e.Transport == nil {
		e.Transport = opsalert.New()
	}
	body, _ := json.Marshal(map[string]any{"type": alertType, "scenario": "landing-page-business-suite", "details": details})
	result, err := e.DB.ExecContext(ctx, `INSERT INTO auth_alert_log (alert_type, window_start, details, dispatch_status, created_at) VALUES ($1,$3,$2,'pending',NOW()) ON CONFLICT (alert_type, window_start) DO NOTHING`, alertType, body, e.NowValue().Truncate(5*time.Minute))
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 { return nil }
	if err := e.Transport.Send(ctx, url, body); err != nil {
		_, _ = e.DB.ExecContext(context.WithoutCancel(ctx), `UPDATE auth_alert_log SET dispatch_status='failed', dispatch_error=$1 WHERE alert_type=$2 AND window_start=$3`, err.Error(), alertType, e.NowValue().Truncate(5*time.Minute))
		return err
	}
	_, _ = e.DB.ExecContext(context.WithoutCancel(ctx), `UPDATE auth_alert_log SET dispatch_status='sent', dispatch_error=NULL WHERE alert_type=$1 AND window_start=$2`, alertType, e.NowValue().Truncate(5*time.Minute))
	return nil
}

func (e *AuthDeliveryAlertEvaluator) NowValue() time.Time { if e.Now != nil { return e.Now().UTC() }; return time.Now().UTC() }
