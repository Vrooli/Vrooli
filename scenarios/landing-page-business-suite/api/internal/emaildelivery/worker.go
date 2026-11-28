package emaildelivery

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type OutcomeKind string

const (
	OutcomeAccepted  OutcomeKind = "accepted"
	OutcomeTemporary OutcomeKind = "temporary"
	OutcomePermanent OutcomeKind = "permanent"
	OutcomeUnknown   OutcomeKind = "unknown"
)

type Outcome struct {
	Kind                      OutcomeKind
	ProviderMessageID, Detail string
}
type Adapter interface {
	Send(context.Context, Provider, Message) Outcome
}

type Worker struct {
	DB interface {
		BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		ExecContext(context.Context, string, ...any) (sql.Result, error)
	}
	Router   Router
	Adapters map[string]Adapter
	// AdapterFor resolves transports for providers added to the registry
	// without requiring a code change for every provider ID.
	AdapterFor func(Provider) Adapter
	Now        func() time.Time
}

// Run drains the durable outbox until the owning process is stopped. The
// worker deliberately has no hidden goroutine or global singleton: callers own
// lifecycle, cancellation, and error reporting.
func (w Worker) Run(ctx context.Context, interval time.Duration) error {
	if interval <= 0 {
		interval = time.Second
	}
	for {
		processed, err := w.ProcessOnce(ctx)
		if err != nil {
			return err
		}
		if processed {
			continue
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// ProcessOnce claims and records the attempt in a short transaction, commits,
// performs the network call, and settles afterward. Unknown outcomes remain
// bound to the selected provider and are never submitted to another provider.
func (w Worker) ProcessOnce(ctx context.Context) (bool, error) {
	now := time.Now()
	if w.Now != nil {
		now = w.Now()
	}
	tx, err := w.DB.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE email_outbox SET status='expired', settled_at=NOW(), updated_at=NOW(), last_error='message expired' WHERE status IN ('pending','retry') AND expires_at IS NOT NULL AND expires_at <= $1`, now); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE email_outbox o SET status='suppressed', settled_at=NOW(), updated_at=NOW(), last_error='recipient is suppressed' WHERE o.status IN ('pending','retry') AND EXISTS (SELECT 1 FROM email_suppressions s WHERE s.recipient = o.recipient)`); err != nil {
		_ = tx.Rollback()
		return false, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, purpose, recipient, sender, subject, text_body, html_body, template_ref, template_data, idempotency_key, priority, attempts, provider_id, expires_at FROM email_outbox WHERE (status IN ('pending','retry') AND available_at <= $1) OR (status='unknown' AND lease_expires_at <= $1) ORDER BY priority DESC, requested_at FOR UPDATE SKIP LOCKED LIMIT 1`, now)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if !rows.Next() {
		rows.Close()
		_ = tx.Rollback()
		return false, nil
	}
	var id string
	var purpose Purpose
	var message Message
	var templateData []byte
	var idempotencyKey string
	var providerID sql.NullString
	var attempts int
	var expires sql.NullTime
	if err := rows.Scan(&id, &purpose, &message.Recipient, &message.Sender, &message.Subject, &message.TextBody, &message.HTMLBody, &message.TemplateRef, &templateData, &idempotencyKey, &message.Priority, &attempts, &providerID, &expires); err != nil {
		rows.Close()
		_ = tx.Rollback()
		return false, err
	}
	rows.Close()
	message.Purpose, message.DedupeKey = purpose, id
	message.TemplateData = templateData
	message.IdempotencyKey = idempotencyKey
	if expires.Valid {
		message.ExpiresAt = &expires.Time
	}
	providers, err := LoadProviders(ctx, w.DB)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	decision := RouteDecision{}
	if providerID.Valid && providerID.String != "" {
		for _, provider := range providers {
			if provider.ID == providerID.String {
				selected := provider
				decision.Chosen = &selected
				break
			}
		}
		if decision.Chosen == nil {
			_ = tx.Rollback()
			return false, fmt.Errorf("unknown email attempt provider %q is no longer registered", providerID.String)
		}
	} else {
		decision = w.Router.Select(ctx, purpose, providers)
	}
	if decision.Chosen == nil {
		_, err = tx.ExecContext(ctx, `UPDATE email_outbox SET status='retry', last_error='no eligible provider', available_at=$2, updated_at=NOW() WHERE id=$1`, id, now.Add(retryDelay(attempts)))
		if err == nil {
			err = tx.Commit()
		} else {
			_ = tx.Rollback()
		}
		return true, err
	}
	reserved, err := reserveQuota(ctx, tx, *decision.Chosen, now)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if !reserved {
		_ = tx.Rollback()
		_, err = w.DB.ExecContext(ctx, `UPDATE email_outbox SET status='retry', last_error='quota window is exhausted', available_at=$2, updated_at=NOW() WHERE id=$1`, id, now.Add(retryDelay(attempts)))
		return true, err
	}
	adapter := w.Adapters[decision.Chosen.ID]
	if adapter == nil && w.AdapterFor != nil {
		adapter = w.AdapterFor(*decision.Chosen)
	}
	if adapter == nil {
		_ = tx.Rollback()
		return false, fmt.Errorf("no adapter for provider %s", decision.Chosen.ID)
	}
	_, err = tx.ExecContext(ctx, `UPDATE email_outbox SET status='claimed', provider_id=$2, attempts=attempts+1, lease_expires_at=$3, updated_at=NOW() WHERE id=$1`, id, decision.Chosen.ID, now.Add(5*time.Minute))
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO email_attempts (outbox_id, provider_id, attempt_number, idempotency_key, outcome, started_at) SELECT $1, $2, attempts, $3 || ':' || attempts, 'in_flight', NOW() FROM email_outbox WHERE id=$1`, id, decision.Chosen.ID, id)
	if err != nil {
		_ = tx.Rollback()
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}
	outcome := adapter.Send(ctx, *decision.Chosen, message)
	var settle string
	var args []any
	switch outcome.Kind {
	case OutcomeAccepted:
		settle, args = `UPDATE email_outbox SET status='accepted', provider_message_id=$2, accepted_at=NOW(), settled_at=NOW(), last_error=NULL, updated_at=NOW() WHERE id=$1`, []any{id, outcome.ProviderMessageID}
	case OutcomeUnknown:
		settle, args = `UPDATE email_outbox SET status='unknown', last_error=$2, lease_expires_at=$3, updated_at=NOW() WHERE id=$1`, []any{id, outcome.Detail, now.Add(5 * time.Minute)}
	case OutcomePermanent:
		settle, args = `UPDATE email_outbox SET status='failed', last_error=$2, settled_at=NOW(), updated_at=NOW() WHERE id=$1`, []any{id, outcome.Detail}
	default:
		settle, args = `UPDATE email_outbox SET status='retry', last_error=$2, available_at=$3, updated_at=NOW() WHERE id=$1`, []any{id, outcome.Detail, now.Add(retryDelay(attempts))}
	}
	if _, err = w.DB.ExecContext(ctx, settle, args...); err != nil {
		return true, err
	}
	if err = recordCircuitOutcome(ctx, w.DB, decision.Chosen.ID, outcome.Kind); err != nil {
		return true, err
	}
	if outcome.Kind == OutcomeAccepted || outcome.Kind == OutcomePermanent {
		deliveryStatus := "sent"
		if outcome.Kind == OutcomePermanent {
			deliveryStatus = "failed"
		}
		if _, err = w.DB.ExecContext(ctx, `UPDATE auth_tokens t SET delivery_status=$2, provider=$3, provider_message_id=$4, delivery_error=$5 WHERE t.id = (SELECT (template_data->>'auth_token_id')::uuid FROM email_outbox WHERE id=$1 AND template_ref='auth.sign-in')`, id, deliveryStatus, decision.Chosen.ID, outcome.ProviderMessageID, outcome.Detail); err != nil {
			return true, err
		}
	}
	_, err = w.DB.ExecContext(ctx, `UPDATE email_attempts SET outcome=$2, diagnostic_code=$3, provider_message_id=$4, finished_at=NOW() WHERE outbox_id=$1 AND outcome='in_flight'`, id, string(outcome.Kind), outcome.Detail, outcome.ProviderMessageID)
	return true, err
}

func reserveQuota(ctx context.Context, tx *sql.Tx, provider Provider, now time.Time) (bool, error) {
	periods := make([]string, 0, len(provider.PublishedLimits))
	for period := range provider.PublishedLimits {
		periods = append(periods, period)
	}
	sort.Strings(periods)
	for _, period := range periods {
		ceiling := provider.PublishedLimits[period]
		if ceiling <= 0 {
			continue
		}
		start, end, err := quotaWindow(period, now.UTC())
		if err != nil {
			return false, err
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO email_quota_windows (provider_id, window_key, window_started_at, window_ends_at, used_count, ceiling) VALUES ($1,$2,$3,$4,0,$5) ON CONFLICT (provider_id, window_key) DO NOTHING`, provider.ID, period, start, end, ceiling); err != nil {
			return false, err
		}
		var used int
		if err := tx.QueryRowContext(ctx, `UPDATE email_quota_windows SET used_count=used_count+1 WHERE provider_id=$1 AND window_key=$2 AND used_count < ceiling RETURNING used_count`, provider.ID, period).Scan(&used); err != nil {
			if err == sql.ErrNoRows {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}

func quotaWindow(period string, now time.Time) (time.Time, time.Time, error) {
	switch period {
	case "day":
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		return start, start.Add(24 * time.Hour), nil
	case "month":
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("unsupported quota period %q", period)
	}
}

func recordCircuitOutcome(ctx context.Context, db interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, providerID string, outcome OutcomeKind) error {
	if outcome == OutcomeAccepted {
		_, err := db.ExecContext(ctx, `INSERT INTO email_provider_circuits (provider_id, consecutive_failures, open_until, updated_at) VALUES ($1,0,NULL,NOW()) ON CONFLICT (provider_id) DO UPDATE SET consecutive_failures=0, open_until=NULL, updated_at=NOW()`, providerID)
		return err
	}
	if outcome != OutcomeTemporary && outcome != OutcomeUnknown {
		return nil
	}
	_, err := db.ExecContext(ctx, `INSERT INTO email_provider_circuits (provider_id, consecutive_failures, open_until, updated_at) VALUES ($1,1,NULL,NOW()) ON CONFLICT (provider_id) DO UPDATE SET consecutive_failures=email_provider_circuits.consecutive_failures+1, open_until=CASE WHEN email_provider_circuits.consecutive_failures+1 >= 3 THEN NOW()+INTERVAL '5 minutes' ELSE email_provider_circuits.open_until END, updated_at=NOW()`, providerID)
	return err
}

func retryDelay(attempts int) time.Duration {
	if attempts < 1 {
		attempts = 1
	}
	delays := []time.Duration{
		5 * time.Second,
		30 * time.Second,
		2 * time.Minute,
		10 * time.Minute,
		30 * time.Minute,
	}
	if attempts > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempts-1]
}
