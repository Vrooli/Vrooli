package commerce

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const paymentReconciliationIncidentType = "payment_reconciliation"

var (
	incidentCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_]{0,63}$`)
	incidentKeyPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9:_-]{0,127}$`)
)

type PaymentReconciliationIncidentService struct {
	db         paymentReconciliationDB
	dispatcher *AnomalyAlertDispatcher
	webhookURL func() string
	scenario   string
	logError   func(string, map[string]interface{})
}

func NewPaymentReconciliationIncidentService(db paymentReconciliationDB, dispatcher *AnomalyAlertDispatcher, webhookURL func() string, runtime DispatcherRuntime) (*PaymentReconciliationIncidentService, error) {
	if db == nil || dispatcher == nil || webhookURL == nil {
		return nil, errors.New("payment reconciliation incident dependencies are incomplete")
	}
	return &PaymentReconciliationIncidentService{db: db, dispatcher: dispatcher, webhookURL: webhookURL, scenario: runtime.ScenarioName, logError: runtime.LogError}, nil
}

// Open records one durable incident identity. Repeated observations do not
// create duplicate open alerts; a later reopening increments the generation.
func (s *PaymentReconciliationIncidentService) Open(ctx context.Context, key, severity string, details map[string]interface{}) (bool, error) {
	key = strings.TrimSpace(key)
	if s == nil || s.db == nil || !incidentKeyPattern.MatchString(key) {
		return false, errors.New("payment reconciliation incident key is required")
	}
	details, err := safePaymentIncidentDetails(details)
	if err != nil {
		return false, err
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return false, fmt.Errorf("marshal payment reconciliation incident: %w", err)
	}
	if severity == "" {
		severity = "error"
	}
	if severity != "info" && severity != "warn" && severity != "error" && severity != "critical" {
		return false, errors.New("payment reconciliation incident severity is invalid")
	}
	result, err := s.db.ExecContext(ctx, `
		WITH opened AS (
			INSERT INTO payment_reconciliation_incidents(incident_key,incident_type,severity,status,generation,details,opened_at,updated_at)
			VALUES($1,$2,$3,'open',1,$4::jsonb,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
			ON CONFLICT(incident_key) DO UPDATE SET
				incident_type=EXCLUDED.incident_type,severity=EXCLUDED.severity,status='open',
				generation=CASE WHEN payment_reconciliation_incidents.status='resolved' THEN payment_reconciliation_incidents.generation+1 ELSE payment_reconciliation_incidents.generation END,
				details=EXCLUDED.details,opened_at=CASE WHEN payment_reconciliation_incidents.status='resolved' THEN CURRENT_TIMESTAMP ELSE payment_reconciliation_incidents.opened_at END,
				resolved_at=NULL,updated_at=CURRENT_TIMESTAMP
			RETURNING incident_key,generation,details
		)
		INSERT INTO payment_reconciliation_incident_events(incident_key,generation,transition,incident_type,severity,details,dispatch_status,dispatch_after,created_at)
		SELECT incident_key,generation,'opened',$2,$3,details,'pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM opened
		ON CONFLICT(incident_key,generation,transition) DO NOTHING
	`, key, paymentReconciliationIncidentType, severity, string(raw))
	if err != nil {
		return false, fmt.Errorf("open payment reconciliation incident: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect opened payment reconciliation incident: %w", err)
	}
	return rows == 1, nil
}

// Resolve records one stable resolution event for the currently open
// generation. Duplicate recovery observations are idempotent.
func (s *PaymentReconciliationIncidentService) Resolve(ctx context.Context, key string, details map[string]interface{}) (bool, error) {
	key = strings.TrimSpace(key)
	if s == nil || s.db == nil || !incidentKeyPattern.MatchString(key) {
		return false, errors.New("payment reconciliation incident key is required")
	}
	details, err := safePaymentIncidentDetails(details)
	if err != nil {
		return false, err
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return false, fmt.Errorf("marshal payment reconciliation resolution: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		WITH resolved AS (
			UPDATE payment_reconciliation_incidents SET status='resolved',resolved_at=CURRENT_TIMESTAMP,updated_at=CURRENT_TIMESTAMP
			WHERE incident_key=$1 AND status='open' RETURNING incident_key,generation,incident_type,severity
		)
		INSERT INTO payment_reconciliation_incident_events(incident_key,generation,transition,incident_type,severity,details,dispatch_status,dispatch_after,created_at)
		SELECT incident_key,generation,'resolved',incident_type,severity,$2::jsonb,'pending',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP FROM resolved
		ON CONFLICT(incident_key,generation,transition) DO NOTHING
	`, key, string(raw))
	if err != nil {
		return false, fmt.Errorf("resolve payment reconciliation incident: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("inspect resolved payment reconciliation incident: %w", err)
	}
	return rows == 1, nil
}

func safePaymentIncidentDetails(details map[string]interface{}) (map[string]interface{}, error) {
	if details == nil {
		details = map[string]interface{}{"code": "unspecified"}
	}
	code, _ := details["code"].(string)
	code = strings.TrimSpace(code)
	if !incidentCodePattern.MatchString(code) {
		return nil, errors.New("payment reconciliation incident requires a safe machine-readable code")
	}
	return map[string]interface{}{"code": code}, nil
}

type paymentReconciliationIncidentEvent struct {
	ID         int64
	Key        string
	Generation int64
	Transition string
	Incident   string
	Severity   string
	Details    map[string]interface{}
	CreatedAt  time.Time
	ClaimToken string
}

func (s *PaymentReconciliationIncidentService) DispatchPending(ctx context.Context, limit int) error {
	if s == nil || s.db == nil {
		return errors.New("payment reconciliation incident service is unavailable")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		url := strings.TrimSpace(s.webhookURL())
		if url == "" {
			return nil
		}
		event, err := s.claimNext(ctx)
		if errors.Is(err, errNoPendingReconciliationIncident) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := s.send(ctx, url, event); err != nil {
			return err
		}
	}
	return nil
}

var errNoPendingReconciliationIncident = errors.New("no pending reconciliation incident event")

func (s *PaymentReconciliationIncidentService) claimNext(ctx context.Context) (paymentReconciliationIncidentEvent, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return paymentReconciliationIncidentEvent{}, fmt.Errorf("create reconciliation incident delivery claim: %w", err)
	}
	event := paymentReconciliationIncidentEvent{ClaimToken: hex.EncodeToString(token[:])}
	var raw []byte
	err := s.db.QueryRowContext(ctx, `
		WITH candidate AS (
			SELECT event.id FROM payment_reconciliation_incident_events AS event
			WHERE (event.dispatch_status IN ('pending','retryable_failed') OR (event.dispatch_status='dispatching' AND event.dispatch_after <= CURRENT_TIMESTAMP)) AND event.dispatch_after <= CURRENT_TIMESTAMP
			  AND NOT EXISTS (
				SELECT 1 FROM payment_reconciliation_incident_events AS earlier
				WHERE earlier.incident_key=event.incident_key AND earlier.id < event.id AND earlier.dispatch_status <> 'sent'
			  )
			ORDER BY event.id FOR UPDATE SKIP LOCKED LIMIT 1
		)
		UPDATE payment_reconciliation_incident_events AS event
		SET dispatch_status='dispatching',claim_token=$1,dispatch_attempts=dispatch_attempts+1,dispatch_after=CURRENT_TIMESTAMP + INTERVAL '5 minutes'
		FROM candidate WHERE event.id=candidate.id
		RETURNING event.id,event.incident_key,event.generation,event.transition,event.incident_type,event.severity,event.details,event.created_at
	`, event.ClaimToken).Scan(&event.ID, &event.Key, &event.Generation, &event.Transition, &event.Incident, &event.Severity, &raw, &event.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return paymentReconciliationIncidentEvent{}, errNoPendingReconciliationIncident
		}
		return paymentReconciliationIncidentEvent{}, fmt.Errorf("claim payment reconciliation incident event: %w", err)
	}
	if err := json.Unmarshal(raw, &event.Details); err != nil {
		return paymentReconciliationIncidentEvent{}, fmt.Errorf("decode payment reconciliation incident event: %w", err)
	}
	return event, nil
}

func (s *PaymentReconciliationIncidentService) send(ctx context.Context, webhookURL string, event paymentReconciliationIncidentEvent) error {
	safeDetails, err := safePaymentIncidentDetails(event.Details)
	if err != nil {
		return s.finishDelivery(ctx, event, false, "alert_payload_invalid")
	}
	payload, err := json.Marshal(struct {
		ID         string                 `json:"id"`
		Type       string                 `json:"type"`
		Severity   string                 `json:"severity"`
		Transition string                 `json:"transition"`
		Generation int64                  `json:"generation"`
		Details    map[string]interface{} `json:"details,omitempty"`
		CreatedAt  string                 `json:"created_at"`
		Scenario   string                 `json:"scenario"`
	}{ID: event.Key, Type: event.Incident, Severity: event.Severity, Transition: event.Transition, Generation: event.Generation, Details: safeDetails, CreatedAt: event.CreatedAt.UTC().Format(time.RFC3339), Scenario: s.scenario})
	if err != nil {
		return s.finishDelivery(ctx, event, false, "alert_payload_invalid")
	}
	s.dispatcher.mu.Lock()
	maxAttempts := s.dispatcher.maxAttempts
	backoffs := append([]time.Duration(nil), s.dispatcher.backoffs...)
	s.dispatcher.mu.Unlock()
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	var lastErr string
	for attempt := 0; attempt < maxAttempts; attempt++ {
		code, message, retryable := s.dispatcher.sendOnce(ctx, webhookURL, payload)
		if code >= 200 && code < 300 {
			return s.finishDelivery(ctx, event, true, "")
		}
		lastErr = "delivery_failed"
		if code >= 400 && code <= 599 {
			lastErr = fmt.Sprintf("http_status_%d", code)
		} else if strings.TrimSpace(message) != "" {
			lastErr = "transport_error"
		}
		if !retryable || attempt+1 >= maxAttempts {
			break
		}
		backoff := time.Duration(1<<attempt) * time.Second
		if attempt < len(backoffs) {
			backoff = backoffs[attempt]
		}
		select {
		case <-ctx.Done():
			return s.finishDelivery(context.WithoutCancel(ctx), event, false, ctx.Err().Error())
		case <-time.After(backoff):
		}
	}
	return s.finishDelivery(context.WithoutCancel(ctx), event, false, lastErr)
}

func (s *PaymentReconciliationIncidentService) finishDelivery(ctx context.Context, event paymentReconciliationIncidentEvent, sent bool, message string) error {
	if len(message) > 512 {
		message = message[:512]
	}
	status := "retryable_failed"
	if sent {
		status = "sent"
		message = ""
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE payment_reconciliation_incident_events
		SET dispatch_status=$3,dispatch_error=NULLIF($4,''),dispatch_after=CURRENT_TIMESTAMP + CASE WHEN $3='sent' THEN INTERVAL '0 seconds' ELSE INTERVAL '30 seconds' END,claim_token=''
		WHERE id=$1 AND claim_token=$2 AND dispatch_status='dispatching'
	`, event.ID, event.ClaimToken, status, message)
	if err != nil {
		return fmt.Errorf("finish payment reconciliation alert delivery: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect payment reconciliation alert delivery: %w", err)
	}
	if rows != 1 {
		return errors.New("payment reconciliation alert delivery claim was lost")
	}
	if !sent && s.logError != nil {
		s.logError("payment_reconciliation_alert_delivery_failed", map[string]interface{}{"event_id": event.ID, "error": message})
	}
	return nil
}

func (s *PaymentReconciliationIncidentService) Run(ctx context.Context, interval time.Duration) error {
	if interval < time.Second {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := s.DispatchPending(ctx, 20); err != nil && !errors.Is(err, context.Canceled) && s.logError != nil {
			s.logError("payment_reconciliation_alert_worker_failed", map[string]interface{}{"error": err.Error()})
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
