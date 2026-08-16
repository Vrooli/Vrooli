package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
	"web-console/internal/dbx"
)

// monetizationGate is the only web-console paid-surface decision point. The
// lease remains authoritative; local UI state cannot grant voice access.
type monetizationGate struct {
	gate   *monetization.Gate
	outbox *monetization.Outbox
}

// voiceSynthesis is the Class B local-capacity enforcement chokepoint.
func (m monetizationGate) voiceSynthesis(w http.ResponseWriter, r *http.Request) {
	identity := bearerIdentityHint(r.Header.Get("Authorization"))
	if m.gate == nil || identity == "" {
		monetization.WriteError(w, http.StatusUnauthorized, monetization.ErrorUnauthorized, monetization.Decision{Reason: monetization.ReasonLeaseUnavailable})
		return
	}
	decision := m.gate.Feature(r.Context(), identity, "voice_synthesis", 2)
	if !decision.Allowed {
		monetization.WriteError(w, http.StatusForbidden, monetization.ErrorSubscriptionRequired, decision)
		return
	}
	// Class B work is local and is recorded durably before its delivery loop.
	if err := m.enqueueUsage(r.Context(), identity); err != nil {
		monetization.WriteError(w, http.StatusServiceUnavailable, monetization.ErrorAuthorityUnavailable, decision)
		return
	}
	if decision.Warning {
		w.Header().Set("X-Entitlement-Warning", monetization.ReasonPastDue)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (m monetizationGate) enqueueUsage(ctx context.Context, identity string) error {
	if m.outbox == nil {
		return monetization.ErrNilOutbox
	}
	operationID, err := monetization.NewOperationID()
	if err != nil {
		return err
	}
	return m.outbox.Enqueue(ctx, monetization.Usage{
		OperationID:  operationID,
		UserIdentity: identity,
		BundleKey:    "business_suite",
		AppKey:       "web-console",
		MeterKey:     "voice_minutes",
		Units:        1,
		OccurredAt:   time.Now().UTC(),
		Metadata:     map[string]string{"operation": "voice_synthesis"},
	})
}

type monetizationUsageReport struct {
	UserIdentity string `json:"user_identity"`
	LimitKey     string `json:"limit_key"`
	UsageAmount  int64  `json:"usage_amount"`
	Amount       int64  `json:"amount"`
	AppBundleKey string `json:"app_bundle_key"`
	OperationID  string `json:"operation_id"`
	Metadata     struct {
		Operation string `json:"operation,omitempty"`
	} `json:"metadata,omitempty"`
}

type lpbsMonetizationTransport struct {
	baseURL      string
	resolveToken func(context.Context, string) (string, error)
	client       *http.Client
}

func (t *lpbsMonetizationTransport) Report(ctx context.Context, usage monetization.Usage) error {
	if t == nil || t.resolveToken == nil || strings.TrimSpace(t.baseURL) == "" {
		return fmt.Errorf("LPBS usage transport is unavailable")
	}
	token, err := t.resolveToken(ctx, t.baseURL)
	if err != nil {
		return fmt.Errorf("resolve LPBS access: %w", err)
	}
	report := monetizationUsageReport{
		UserIdentity: usage.UserIdentity,
		LimitKey:     usage.MeterKey,
		UsageAmount:  usage.Units,
		Amount:       usage.Units,
		AppBundleKey: usage.AppKey,
		OperationID:  usage.OperationID,
	}
	report.Metadata.Operation = usage.Metadata["operation"]
	body, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal LPBS usage report: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(t.baseURL, "/")+"/api/v1/usage/report", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create LPBS usage request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	client := t.client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("send LPBS usage report: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("LPBS usage report returned status %d", response.StatusCode)
	}
	return nil
}

type sqlMonetizationOutboxStore struct{ db dbx.Handle }

func (s *sqlMonetizationOutboxStore) Append(ctx context.Context, usage monetization.Usage) (bool, error) {
	payload, err := json.Marshal(usage)
	if err != nil {
		return false, fmt.Errorf("marshal monetization usage: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO monetization_usage_outbox (operation_id, user_identity, payload)
		VALUES (?, ?, ?)
		ON CONFLICT(operation_id) DO NOTHING
	`, usage.OperationID, usage.UserIdentity, string(payload))
	if err != nil {
		return false, fmt.Errorf("persist monetization usage: %w", err)
	}
	count, err := result.RowsAffected()
	return count > 0, err
}

func (s *sqlMonetizationOutboxStore) Pending(ctx context.Context, limit int, now time.Time) ([]monetization.OutboxRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT operation_id, payload, attempts, next_attempt_at, last_error, delivered_at
		FROM monetization_usage_outbox
		WHERE status = 'pending' AND next_attempt_at <= ?
		ORDER BY created_at
		LIMIT ?
	`, now.UTC().Format("2006-01-02T15:04:05.000Z"), limit)
	if err != nil {
		return nil, fmt.Errorf("query monetization usage: %w", err)
	}
	defer rows.Close()
	result := make([]monetization.OutboxRecord, 0, limit)
	for rows.Next() {
		var operationID, payload string
		var attempts int
		var nextAttempt, deliveredAt sql.NullString
		var lastError sql.NullString
		if err := rows.Scan(&operationID, &payload, &attempts, &nextAttempt, &lastError, &deliveredAt); err != nil {
			return nil, fmt.Errorf("scan monetization usage: %w", err)
		}
		var usage monetization.Usage
		if err := json.Unmarshal([]byte(payload), &usage); err != nil {
			return nil, fmt.Errorf("decode monetization usage: %w", err)
		}
		next, err := parseMonetizationTime(nextAttempt.String)
		if err != nil {
			next = now
		}
		record := monetization.OutboxRecord{Usage: usage, Attempts: attempts, NextAttemptAt: next}
		if lastError.Valid {
			record.LastError = lastError.String
		}
		if deliveredAt.Valid {
			record.DeliveredAt, _ = parseMonetizationTime(deliveredAt.String)
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (s *sqlMonetizationOutboxStore) MarkDelivered(ctx context.Context, operationID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `UPDATE monetization_usage_outbox SET status='delivered', delivered_at=?, updated_at=CURRENT_TIMESTAMP WHERE operation_id=?`, at.UTC().Format(time.RFC3339Nano), operationID)
	return err
}

func (s *sqlMonetizationOutboxStore) MarkRetry(ctx context.Context, operationID string, next time.Time, reason string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE monetization_usage_outbox SET attempts=attempts+1, last_error=?, next_attempt_at=?, updated_at=CURRENT_TIMESTAMP WHERE operation_id=?`, reason, next.UTC().Format(time.RFC3339Nano), operationID)
	return err
}

func parseMonetizationTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.000Z", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid monetization timestamp %q", value)
}

func (s *Server) drainMonetizationOutbox() {
	if s.monetizationOutbox == nil {
		return
	}
	ctx := context.Background()
	if _, err := s.monetizationOutbox.Drain(ctx, 50); err != nil {
		log.Printf("monetization outbox startup drain: %v", err)
	}
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if _, err := s.monetizationOutbox.Drain(ctx, 50); err != nil {
			log.Printf("monetization outbox drain: %v", err)
		}
	}
}

func bearerIdentityHint(raw string) string {
	const prefix = "Bearer "
	if !strings.HasPrefix(raw, prefix) {
		return ""
	}
	parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(raw, prefix)), ".")
	if len(parts) != 3 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Email string `json:"email"`
		Sub   string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	if strings.TrimSpace(claims.Email) != "" {
		return strings.ToLower(strings.TrimSpace(claims.Email))
	}
	return strings.TrimSpace(claims.Sub)
}
