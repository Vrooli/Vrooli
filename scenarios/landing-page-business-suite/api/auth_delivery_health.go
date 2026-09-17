package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/vrooli/api-core/health"
	"landing-page-business-suite-api/internal/administration"
)

const signInDeliveryWindow = time.Hour

// signInDeliveryCheck makes a failing sign-in mail provider visible in health
// instead of only in logs. It is optional: the API still serves, but
// customers cannot receive codes, so operators see degraded status.
func (s *Server) signInDeliveryCheck() health.Checker {
	return health.Func("sign_in_email", func(ctx context.Context) error {
		if s.userAuthService == nil {
			return errors.New("sign-in service unavailable")
		}
		summary, err := s.userAuthService.DeliveryHealth(ctx, signInDeliveryWindow)
		if err != nil {
			return fmt.Errorf("read sign-in delivery outcomes: %w", err)
		}
		if summary.LastFailure != nil && (summary.LastSuccess == nil || summary.LastFailure.After(*summary.LastSuccess)) {
			return fmt.Errorf("%d of the last %d sign-in emails failed in the past hour; latest: %s", summary.Failed, summary.Failed+summary.Sent, summary.LastError)
		}
		return nil
	})
}

func (s *Server) signInEmailDNSCheck() health.Checker {
	return health.Func("sign_in_email_dns", func(ctx context.Context) error {
		if s.emailReadinessService == nil {
			return errors.New("email readiness unavailable")
		}
		// DNS deliverability is a deployed-sending gate: a non-production host
		// serves on a loopback origin and sends no real mail, so the From domain
		// can never align with the public origin or publish DKIM records here.
		// Evaluating it locally would permanently degrade a healthy development
		// instance and trigger needless recovery churn. The admin
		// email-readiness report still runs the full evaluation on demand.
		if !isProductionEnvironment() {
			return nil
		}
		var lastEvent sql.NullTime
		_ = s.db.QueryRowContext(ctx, `SELECT MAX(received_at) FROM auth_email_events`).Scan(&lastEvent)
		report := s.emailReadinessService.Evaluate(ctx, resolveConfig("EMAIL_FROM_ADDRESS"), resolveConfig("PUBLIC_BASE_URL"), resolveSecret("SENDGRID_API_KEY") != "", resolveConfig("EMAIL_SMTP_DKIM_SELECTOR"), resolveSecret("SENDGRID_WEBHOOK_PUBLIC_KEY") != "", lastEvent.Time)
		for _, check := range report.Checks {
			if check.Status == "fail" {
				return fmt.Errorf("%s: %s", check.Name, check.Detail)
			}
		}
		return nil
	})
}

// signInDeliveryReport exposes recent outcomes to administrators.
func (s *Server) signInDeliveryReport(w http.ResponseWriter, r *http.Request) {
	summary, err := s.userAuthService.DeliveryHealth(r.Context(), 24*time.Hour)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Unable to read sign-in delivery outcomes.", ApiErrorTypeServerError)
		return
	}
	report := map[string]any{"window_hours": 24, "delivery": summary}
	var metrics struct {
		Sent, Failed, Delivered, Bounced, Deferred, Dropped int
		LastError                                           sql.NullString
		LastWebhookEvent                                    sql.NullTime
	}
	if err := s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FILTER (WHERE delivery_status='sent'), COUNT(*) FILTER (WHERE delivery_status='failed'), COUNT(*) FILTER (WHERE provider_status='delivered'), COUNT(*) FILTER (WHERE provider_status='bounce'), COUNT(*) FILTER (WHERE provider_status='deferred'), COUNT(*) FILTER (WHERE provider_status='dropped'), (SELECT delivery_error FROM auth_tokens WHERE token_type='magic_link' AND delivery_status='failed' ORDER BY created_at DESC LIMIT 1), (SELECT MAX(received_at) FROM auth_email_events) FROM auth_tokens WHERE token_type='magic_link' AND created_at > NOW() - INTERVAL '24 hours'`).Scan(&metrics.Sent, &metrics.Failed, &metrics.Delivered, &metrics.Bounced, &metrics.Deferred, &metrics.Dropped, &metrics.LastError, &metrics.LastWebhookEvent); err == nil {
		report["delivery_24h"] = map[string]any{"sent": metrics.Sent, "failed": metrics.Failed, "delivered": metrics.Delivered, "bounced": metrics.Bounced, "deferred": metrics.Deferred, "dropped": metrics.Dropped, "last_error": metrics.LastError.String, "last_webhook_event": metrics.LastWebhookEvent.Time}
	}
	writeJSON(w, report)
}

func (s *Server) emailReadinessReport(w http.ResponseWriter, r *http.Request) {
	if s.emailReadinessService == nil {
		http.Error(w, "email readiness unavailable", http.StatusServiceUnavailable)
		return
	}
	var lastEvent sql.NullTime
	_ = s.db.QueryRowContext(r.Context(), `SELECT MAX(received_at) FROM auth_email_events`).Scan(&lastEvent)
	report := s.emailReadinessService.Evaluate(r.Context(), resolveConfig("EMAIL_FROM_ADDRESS"), resolveConfig("PUBLIC_BASE_URL"), resolveSecret("SENDGRID_API_KEY") != "", resolveConfig("EMAIL_SMTP_DKIM_SELECTOR"), resolveSecret("SENDGRID_WEBHOOK_PUBLIC_KEY") != "", lastEvent.Time)
	writeJSON(w, report)
}

func (s *Server) signInDeliveryProbe(w http.ResponseWriter, r *http.Request) {
	var request struct {
		To string `json:"to"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&request); err != nil || !administration.LooksLikeEmail(request.To) {
		writeJSONError(w, http.StatusBadRequest, "A valid destination email is required.", ApiErrorTypeValidation)
		return
	}
	started, err := s.userAuthService.RequestSignIn(r.Context(), administration.SignInRequest{Email: request.To, IPAddress: getClientIP(r), UserAgent: r.UserAgent(), Context: json.RawMessage(`{"purpose":"probe"}`)})
	if err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "Unable to send delivery probe.", ApiErrorTypeServerError)
		return
	}
	writeJSON(w, map[string]any{"request_id": started.RequestID, "expires_at": started.ExpiresAt.UTC().Format(time.RFC3339)})
}
