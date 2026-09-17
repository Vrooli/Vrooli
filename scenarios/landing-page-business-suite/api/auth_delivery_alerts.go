package main

import (
	"context"
	"time"

	"landing-page-business-suite-api/internal/commerce"
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
				}
			}
		}
	}()
	return cancel
}
