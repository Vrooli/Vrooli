package credits

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	monetization "github.com/vrooli/vrooli/packages/monetization-go"
)

// sqlUsageOutboxStore adapts BAS's existing durable table to the shared
// monetization outbox contract. The unique operation_id constraint remains
// the local idempotency boundary; LPBS applies the same key upstream.
type sqlUsageOutboxStore struct {
	db sqlExecutor
}

func (s *sqlUsageOutboxStore) Append(ctx context.Context, usage monetization.Usage) (bool, error) {
	payload, err := json.Marshal(usage)
	if err != nil {
		return false, fmt.Errorf("marshal shared usage outbox payload: %w", err)
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO monetization_usage_outbox (operation_id, user_identity, payload, status, next_attempt_at)
		VALUES (?, ?, ?, 'pending', CURRENT_TIMESTAMP)
		ON CONFLICT(operation_id) DO NOTHING
	`, usage.OperationID, usage.UserIdentity, string(payload))
	if err != nil {
		return false, fmt.Errorf("persist shared usage outbox: %w", err)
	}
	inserted, err := result.RowsAffected()
	return inserted > 0, err
}

func (s *sqlUsageOutboxStore) Pending(ctx context.Context, limit int, now time.Time) ([]monetization.OutboxRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT operation_id, payload, attempts, next_attempt_at, last_error, delivered_at
		FROM monetization_usage_outbox
		WHERE status = 'pending' AND next_attempt_at <= ?
		ORDER BY created_at
		LIMIT ?
	`, now.UTC().Format("2006-01-02 15:04:05"), limit)
	if err != nil {
		return nil, fmt.Errorf("query shared usage outbox: %w", err)
	}
	defer rows.Close()
	result := make([]monetization.OutboxRecord, 0, limit)
	for rows.Next() {
		var operationID, payload string
		var attempts int
		var nextAttemptValue any
		var lastError sql.NullString
		var deliveredAtValue any
		if err := rows.Scan(&operationID, &payload, &attempts, &nextAttemptValue, &lastError, &deliveredAtValue); err != nil {
			return nil, fmt.Errorf("scan shared usage outbox: %w", err)
		}
		nextAttempt := databaseTime(nextAttemptValue, now)
		usage, err := decodeUsagePayload(payload, operationID)
		if err != nil {
			return nil, err
		}
		record := monetization.OutboxRecord{Usage: usage, Attempts: attempts, NextAttemptAt: nextAttempt}
		if lastError.Valid {
			record.LastError = lastError.String
		}
		if deliveredAt := databaseTimeOrZero(deliveredAtValue); !deliveredAt.IsZero() {
			record.DeliveredAt = deliveredAt
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func databaseTime(value any, fallback time.Time) time.Time {
	if parsed := databaseTimeOrZero(value); !parsed.IsZero() {
		return parsed
	}
	return fallback
}

func databaseTimeOrZero(value any) time.Time {
	switch typed := value.(type) {
	case time.Time:
		return typed
	case string:
		for _, layout := range []string{time.RFC3339Nano, "2006-01-02 15:04:05.999999999-07:00", "2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05"} {
			if parsed, err := time.Parse(layout, typed); err == nil {
				return parsed
			}
		}
	case []byte:
		return databaseTimeOrZero(string(typed))
	}
	return time.Time{}
}

func (s *sqlUsageOutboxStore) MarkDelivered(ctx context.Context, operationID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE monetization_usage_outbox
		SET status = 'delivered', delivered_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE operation_id = ?
	`, at, operationID)
	return err
}

func (s *sqlUsageOutboxStore) MarkRetry(ctx context.Context, operationID string, next time.Time, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE monetization_usage_outbox
		SET attempts = attempts + 1, last_error = ?, next_attempt_at = ?, updated_at = CURRENT_TIMESTAMP
		WHERE operation_id = ?
	`, reason, next, operationID)
	return err
}

type lpbsUsageTransport struct {
	service *Service
}

func (t *lpbsUsageTransport) Report(ctx context.Context, usage monetization.Usage) error {
	if t == nil || t.service == nil {
		return fmt.Errorf("LPBS usage transport is unavailable")
	}
	accessToken, err := t.service.resolveLPBSAccess(ctx)
	if err != nil {
		return err
	}
	return t.service.sendLPBSReport(ctx, lpbsReportFromUsage(usage), accessToken)
}

func usageFromLPBSReport(report LPBSUsageReport) monetization.Usage {
	metadata := map[string]string{
		"operation":     report.Metadata.Operation,
		"model":         report.Metadata.Model,
		"prompt_tokens": strconv.Itoa(report.Metadata.PromptTokens),
		"is_byok":       strconv.FormatBool(report.Metadata.IsBYOK),
	}
	return monetization.Usage{
		OperationID:  report.OperationID,
		UserIdentity: report.UserIdentity,
		BundleKey:    "business_suite",
		AppKey:       report.AppBundleKey,
		MeterKey:     report.LimitKey,
		Units:        report.UsageAmount,
		OccurredAt:   time.Now().UTC(),
		Metadata:     metadata,
	}
}

func lpbsReportFromUsage(usage monetization.Usage) LPBSUsageReport {
	metadata := usage.Metadata
	promptTokens, _ := strconv.Atoi(metadata["prompt_tokens"])
	byok, _ := strconv.ParseBool(metadata["is_byok"])
	return LPBSUsageReport{
		UserIdentity: usage.UserIdentity,
		LimitKey:     usage.MeterKey,
		UsageAmount:  usage.Units,
		Amount:       usage.Units,
		AppBundleKey: usage.AppKey,
		OperationID:  usage.OperationID,
		Metadata: LPBSUsageReportMetadata{
			Operation:    metadata["operation"],
			Model:        metadata["model"],
			PromptTokens: promptTokens,
			IsBYOK:       byok,
		},
	}
}

func decodeUsagePayload(payload, operationID string) (monetization.Usage, error) {
	var usage monetization.Usage
	if err := json.Unmarshal([]byte(payload), &usage); err == nil &&
		strings.TrimSpace(usage.OperationID) != "" && strings.TrimSpace(usage.AppKey) != "" {
		return usage, nil
	}
	var legacy LPBSUsageReport
	if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
		return monetization.Usage{}, fmt.Errorf("decode shared usage outbox payload: %w", err)
	}
	if legacy.OperationID == "" {
		legacy.OperationID = operationID
	}
	return usageFromLPBSReport(legacy), nil
}
