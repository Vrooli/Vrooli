package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vrooli/api-core/health"
	maildns "github.com/vrooli/vrooli/packages/maildns-go"
	"landing-page-business-suite-api/internal/administration"
	"landing-page-business-suite-api/internal/emaildelivery"
)

// signInDeliveryWindow is the reporting window for the 24-hour operator
// report and for health. It is a day rather than an hour because health used
// to forget a broken provider as soon as the failed attempts aged out: the
// site reported healthy again while sign-in was still impossible, and only
// degraded while a customer happened to be failing to sign in.
const signInDeliveryWindow = 24 * time.Hour

// signInDeliveryCheck makes a failing sign-in mail provider visible in health
// instead of only in logs. It is optional: the API still serves, but
// customers cannot receive codes, so operators see degraded status.
func (s *Server) signInDeliveryCheck() health.Checker {
	return health.Func("sign_in_email", func(ctx context.Context) error {
		if s.userAuthService == nil {
			return errors.New("sign-in service unavailable")
		}
		// Registry eligibility first: a deployment with no enabled, usable
		// provider cannot deliver a code whether or not anyone has tried. This
		// follows the same route decision as the worker instead of checking a
		// fixed pair of legacy configuration sources.
		if isProductionEnvironment() {
			providers, err := emaildelivery.LoadProviders(ctx, s.routedDB)
			if err != nil {
				return fmt.Errorf("read email provider registry: %w", err)
			}
			decision := emaildelivery.Router{
				Credentials: emailCredentialChecker{service: s.emailService},
				DNS:         emailDNSChecker{service: s.mailDNSService, domain: senderDomain},
				Quota:       emailQuotaChecker{db: s.routedDB},
				Circuit:     emailCircuitChecker{db: s.routedDB},
			}.Select(ctx, emaildelivery.PurposeSignIn, providers)
			if decision.Chosen == nil {
				for _, candidate := range decision.Candidates {
					if candidate.SkipReason != "" {
						return fmt.Errorf("no eligible sign-in email provider: %s (%s)", candidate.Provider.ID, candidate.SkipReason)
					}
				}
				return errors.New("no enabled sign-in email provider is registered")
			}
		}
		summary, err := s.userAuthService.DeliveryHealth(ctx, signInDeliveryWindow)
		if err != nil {
			return fmt.Errorf("read sign-in delivery outcomes: %w", err)
		}
		if summary.LastFailure != nil && (summary.LastSuccess == nil || summary.LastFailure.After(*summary.LastSuccess)) {
			return fmt.Errorf("the most recent sign-in email failed (%d of %d attempts in the last 24 hours): %s",
				summary.Failed, summary.Failed+summary.Sent, summary.LastError)
		}
		return nil
	})
}

func (s *Server) signInEmailDNSCheck() health.Checker {
	return health.Func("sign_in_email_dns", func(ctx context.Context) error {
		if s.mailDNSService == nil {
			return errors.New("email readiness unavailable")
		}
		report := s.mailDNSService.Verify(ctx, senderDomain(), s.mailDNSRequirements(ctx))
		if report.DMARC.Status == maildns.Fail {
			return fmt.Errorf("dmarc: %s", report.DMARC.Detail)
		}
		for _, check := range report.Providers {
			if check.Status == maildns.Fail {
				return fmt.Errorf("%s: %s", check.Provider, check.Detail)
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
	if rows, err := s.routedDB.QueryContext(r.Context(), `SELECT status, COUNT(*) FROM email_outbox GROUP BY status`); err == nil {
		defer rows.Close()
		outbox := map[string]int{}
		for rows.Next() {
			var status string
			var count int
			if rows.Scan(&status, &count) == nil {
				outbox[status] = count
			}
		}
		report["outbox"] = outbox
	}
	if details, err := s.emailDeliveryAdminDetails(r.Context(), r.URL.Query().Get("recipient"), r.URL.Query().Get("message_id")); err == nil {
		for key, value := range details {
			report[key] = value
		}
	}
	writeJSON(w, report)
}

// emailDeliveryAdminDetails is deliberately read-only and returns provider
// diagnostics, not secrets. It is kept behind the existing administrator/
// metrics route so message addresses never become a public API.
func (s *Server) emailDeliveryAdminDetails(ctx context.Context, recipient, messageID string) (map[string]any, error) {
	if s.routedDB == nil {
		return nil, errors.New("email database unavailable")
	}
	providers, err := emaildelivery.LoadProviders(ctx, s.routedDB)
	if err != nil {
		return nil, err
	}
	router := emaildelivery.Router{
		Credentials: emailCredentialChecker{service: s.emailService},
		DNS:         emailDNSChecker{service: s.mailDNSService, domain: senderDomain},
		Quota:       emailQuotaChecker{db: s.routedDB},
		Circuit:     emailCircuitChecker{db: s.routedDB},
	}
	decision := router.Select(ctx, emaildelivery.PurposeSignIn, providers)
	providerRows := make([]map[string]any, 0, len(decision.Candidates))
	for _, candidate := range decision.Candidates {
		provider := candidate.Provider
		credentialOK, credentialDetail := emailCredentialChecker{service: s.emailService}.Verify(ctx, provider)
		dnsOK, dnsDetail := emailDNSChecker{service: s.mailDNSService, domain: senderDomain}.Verify(ctx, provider)
		state, remedy := "ready", ""
		switch {
		case !provider.Enabled:
			state, remedy = "off", "Enable this provider in the email provider registry."
		case !credentialOK:
			state, remedy = "needs_setup", "Provision or rotate the referenced credential, then rerun provider verification."
		case !dnsOK:
			state, remedy = "needs_dns", dnsDetail
		case candidate.SkipReason != "":
			state, remedy = "skipped", candidate.SkipReason
		}
		providerRows = append(providerRows, map[string]any{
			"id": provider.ID, "transport": provider.Transport, "enabled": provider.Enabled,
			"cost_rank": provider.CostRank, "recommended": provider.Recommended,
			"credential": map[string]any{"ok": credentialOK, "detail": credentialDetail},
			"dns":        map[string]any{"ok": dnsOK, "detail": dnsDetail},
			"state":      state, "remedy": remedy, "skip_reason": candidate.SkipReason,
		})
	}
	routingRows := make([]map[string]any, 0, len(decision.Candidates))
	for _, candidate := range decision.Candidates {
		routingRows = append(routingRows, map[string]any{"provider_id": candidate.Provider.ID, "cost_rank": candidate.Provider.CostRank, "skip_reason": candidate.SkipReason, "eligible": candidate.SkipReason == ""})
	}
	result := map[string]any{"providers": providerRows, "routing": routingRows}
	if decision.Chosen != nil {
		result["chosen_provider"] = decision.Chosen.ID
	}
	if quotas, quotaErr := emailQuotaAdminRows(ctx, s.routedDB); quotaErr == nil {
		result["quotas"] = quotas
	}
	if queue, queueErr := emailQueueHealth(ctx, s.routedDB); queueErr == nil {
		result["queue_health"] = queue
	}
	history, historyErr := emailMessageHistory(ctx, s.routedDB, recipient, messageID)
	if historyErr == nil {
		result["history"] = history
	}
	return result, nil
}

func emailQuotaAdminRows(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `SELECT provider_id, window_key, used_count, ceiling, window_started_at, window_ends_at FROM email_quota_windows WHERE window_ends_at > NOW() ORDER BY provider_id, window_ends_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var provider, key string
		var used, ceiling int
		var started, ends time.Time
		if err := rows.Scan(&provider, &key, &used, &ceiling, &started, &ends); err != nil {
			return nil, err
		}
		result = append(result, map[string]any{"provider_id": provider, "window": key, "used": used, "ceiling": ceiling, "started_at": started, "ends_at": ends})
	}
	return result, rows.Err()
}

func emailQueueHealth(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}) (map[string]any, error) {
	rows, err := db.QueryContext(ctx, `SELECT priority, COUNT(*), COALESCE(EXTRACT(EPOCH FROM (NOW() - MIN(requested_at))),0)::bigint FROM email_outbox WHERE status IN ('pending','retry','unknown','claimed') GROUP BY priority ORDER BY priority DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byPriority := []map[string]any{}
	var oldest int64
	for rows.Next() {
		var priority, count int
		var age int64
		if err := rows.Scan(&priority, &count, &age); err != nil {
			return nil, err
		}
		if age > oldest {
			oldest = age
		}
		byPriority = append(byPriority, map[string]any{"priority": priority, "depth": count, "oldest_age_seconds": age})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	var median sql.NullFloat64
	if err := db.QueryRowContext(ctx, `SELECT percentile_cont(0.5) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM (accepted_at - requested_at))) FROM email_outbox WHERE accepted_at IS NOT NULL AND requested_at >= NOW() - INTERVAL '24 hours'`).Scan(&median); err != nil {
		return nil, err
	}
	var medianValue any
	if median.Valid {
		medianValue = median.Float64
	}
	return map[string]any{"by_priority": byPriority, "oldest_wait_seconds": oldest, "median_acceptance_seconds_24h": medianValue}, nil
}

func emailMessageHistory(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, recipient, messageID string) ([]map[string]any, error) {
	rows, err := db.QueryContext(ctx, `SELECT id::text, purpose, recipient, requested_at, provider_id, accepted_at, status, last_error, provider_message_id FROM email_outbox WHERE ($1 = '' OR recipient ILIKE '%' || $1 || '%') AND ($2 = '' OR id::text = $2) ORDER BY requested_at DESC LIMIT 50`, recipient, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var id, purpose, to, status string
		var provider, lastError, providerMessage sql.NullString
		var requested time.Time
		var accepted sql.NullTime
		if err := rows.Scan(&id, &purpose, &to, &requested, &provider, &accepted, &status, &lastError, &providerMessage); err != nil {
			return nil, err
		}
		item := map[string]any{"id": id, "purpose": purpose, "recipient": to, "requested_at": requested, "provider_id": provider.String, "status": status, "last_error": lastError.String, "provider_message_id": providerMessage.String}
		if accepted.Valid {
			item["accepted_at"] = accepted.Time
			item["acceptance_seconds"] = accepted.Time.Sub(requested).Seconds()
		}
		item["attempts"] = emailAttemptHistory(ctx, db, id)
		result = append(result, item)
	}
	return result, rows.Err()
}

func emailAttemptHistory(ctx context.Context, db interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, outboxID string) []map[string]any {
	rows, err := db.QueryContext(ctx, `SELECT attempt_number, provider_id, outcome, diagnostic_code, provider_message_id, started_at, finished_at FROM email_attempts WHERE outbox_id=$1 ORDER BY attempt_number`, outboxID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var number int
		var provider, outcome string
		var diagnostic, message sql.NullString
		var started time.Time
		var finished sql.NullTime
		if rows.Scan(&number, &provider, &outcome, &diagnostic, &message, &started, &finished) != nil {
			continue
		}
		item := map[string]any{"attempt": number, "provider_id": provider, "outcome": outcome, "diagnostic": diagnostic.String, "provider_message_id": message.String, "started_at": started}
		if finished.Valid {
			item["finished_at"] = finished.Time
		}
		result = append(result, item)
	}
	return result
}

func (s *Server) emailReadinessReport(w http.ResponseWriter, r *http.Request) {
	if s.mailDNSService == nil {
		http.Error(w, "email readiness unavailable", http.StatusServiceUnavailable)
		return
	}
	report := s.mailDNSService.Verify(r.Context(), senderDomain(), s.mailDNSRequirements(r.Context()))
	writeJSON(w, report)
}

func senderDomain() string {
	parts := strings.Split(strings.TrimSpace(strings.ToLower(resolveConfig("EMAIL_FROM_ADDRESS"))), "@")
	if len(parts) != 2 {
		return ""
	}
	return strings.TrimSuffix(parts[1], ".")
}

func (s *Server) mailDNSRequirements(ctx context.Context) []maildns.ProviderRequirement {
	if s == nil || s.routedDB == nil {
		return nil
	}
	providers, err := emaildelivery.LoadProviders(ctx, s.routedDB)
	if err != nil {
		return nil
	}
	requirements := make([]maildns.ProviderRequirement, 0, len(providers))
	for _, provider := range providers {
		if provider.Enabled {
			requirements = append(requirements, providerMailDNSRequirements(provider)...)
		}
	}
	return requirements
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
