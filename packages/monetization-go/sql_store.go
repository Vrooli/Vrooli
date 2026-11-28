package monetization

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// SQLDialect selects the placeholder syntax used by a durable outbox store.
// The store assumes the shared monetization_usage_outbox schema and a unique
// operation_id constraint supplied by the scenario's migration.
type SQLDialect string

const (
	SQLDialectSQLite   SQLDialect = "sqlite"
	SQLDialectPostgres SQLDialect = "postgres"
)

const (
	sqlPlaceholderFirst  = 1
	sqlPlaceholderSecond = 2
	sqlPlaceholderThird  = 3
)

// SQLExecutor is the narrow database seam needed by SQLStore. It deliberately
// excludes transactions because each state transition is one atomic SQL
// statement guarded by the operation id.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// SQLStore persists the shared outbox contract in SQLite or PostgreSQL. It
// stores the full usage event so retries preserve the same idempotency key and
// payload across process restarts.
type SQLStore struct {
	db      SQLExecutor
	dialect SQLDialect
}

// NewSQLStore constructs a durable shared outbox store. The caller owns the
// database lifecycle and must have applied the shared outbox schema first.
func NewSQLStore(db SQLExecutor, dialect SQLDialect) *SQLStore {
	return &SQLStore{db: db, dialect: dialect}
}

func (s *SQLStore) placeholder(index int) string {
	if s != nil && s.dialect == SQLDialectPostgres {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

// Append durably inserts a usage event and treats an existing operation id as
// an idempotent no-op.
func (s *SQLStore) Append(ctx context.Context, usage Usage) (bool, error) {
	if s == nil || s.db == nil {
		return false, ErrNilOutbox
	}
	payload, err := json.Marshal(usage)
	if err != nil {
		return false, fmt.Errorf("marshal monetization usage: %w", err)
	}
	query := fmt.Sprintf(`
		INSERT INTO monetization_usage_outbox (operation_id, user_identity, payload, status, next_attempt_at)
		VALUES (%s, %s, %s, 'pending', CURRENT_TIMESTAMP)
		ON CONFLICT(operation_id) DO NOTHING
	`, s.placeholder(sqlPlaceholderFirst), s.placeholder(sqlPlaceholderSecond), s.placeholder(sqlPlaceholderThird))
	result, err := s.db.ExecContext(ctx, query, usage.OperationID, usage.UserIdentity, string(payload))
	if err != nil {
		return false, fmt.Errorf("persist monetization usage: %w", err)
	}
	inserted, err := result.RowsAffected()
	return inserted > 0, err
}

// Pending returns ready undelivered events ordered by creation time.
func (s *SQLStore) Pending(ctx context.Context, limit int, now time.Time) ([]OutboxRecord, error) {
	if s == nil || s.db == nil {
		return nil, ErrNilOutbox
	}
	query := fmt.Sprintf(`
		SELECT operation_id, payload, attempts, next_attempt_at, last_error, delivered_at
		FROM monetization_usage_outbox
		WHERE status = 'pending' AND next_attempt_at <= %s
		ORDER BY created_at
		LIMIT %s
	`, s.placeholder(sqlPlaceholderFirst), s.placeholder(sqlPlaceholderSecond))
	rows, err := s.db.QueryContext(ctx, query, now.UTC(), limit)
	if err != nil {
		return nil, fmt.Errorf("query pending monetization usage: %w", err)
	}
	defer rows.Close()
	result := make([]OutboxRecord, 0, limit)
	for rows.Next() {
		var operationID, payload string
		var attempts int
		var nextAttemptValue any
		var lastError sql.NullString
		var deliveredAtValue any
		if err := rows.Scan(&operationID, &payload, &attempts, &nextAttemptValue, &lastError, &deliveredAtValue); err != nil {
			return nil, fmt.Errorf("scan pending monetization usage: %w", err)
		}
		var usage Usage
		if err := json.Unmarshal([]byte(payload), &usage); err != nil {
			return nil, fmt.Errorf("decode pending monetization usage: %w", err)
		}
		if usage.OperationID == "" {
			usage.OperationID = operationID
		}
		record := OutboxRecord{Usage: usage, Attempts: attempts, NextAttemptAt: databaseTime(nextAttemptValue, now)}
		if lastError.Valid {
			record.LastError = lastError.String
		}
		record.DeliveredAt = databaseTimeOrZero(deliveredAtValue)
		result = append(result, record)
	}
	return result, rows.Err()
}

// MarkDelivered records the durable acknowledgement after the authority has
// accepted the event. The authority's operation id remains the final dedupe.
func (s *SQLStore) MarkDelivered(ctx context.Context, operationID string, at time.Time) error {
	if s == nil || s.db == nil {
		return ErrNilOutbox
	}
	query := fmt.Sprintf(`UPDATE monetization_usage_outbox SET status = 'delivered', delivered_at = %s, updated_at = CURRENT_TIMESTAMP WHERE operation_id = %s`, s.placeholder(sqlPlaceholderFirst), s.placeholder(sqlPlaceholderSecond))
	_, err := s.db.ExecContext(ctx, query, at, operationID)
	return err
}

// MarkRetry stores the next attempt time and error so a restart retains the
// retry schedule.
func (s *SQLStore) MarkRetry(ctx context.Context, operationID string, next time.Time, reason string) error {
	if s == nil || s.db == nil {
		return ErrNilOutbox
	}
	query := fmt.Sprintf(`UPDATE monetization_usage_outbox SET attempts = attempts + 1, last_error = %s, next_attempt_at = %s, updated_at = CURRENT_TIMESTAMP WHERE operation_id = %s`, s.placeholder(sqlPlaceholderFirst), s.placeholder(sqlPlaceholderSecond), s.placeholder(sqlPlaceholderThird))
	_, err := s.db.ExecContext(ctx, query, reason, next, operationID)
	return err
}

// PendingCount counts durable undelivered records for an account surface.
func (s *SQLStore) PendingCount(ctx context.Context, userIdentity string) (int, error) {
	if s == nil || s.db == nil {
		return 0, ErrNilOutbox
	}
	userIdentity = strings.TrimSpace(userIdentity)
	if userIdentity == "" {
		return 0, nil
	}
	query := fmt.Sprintf(`SELECT COUNT(*) FROM monetization_usage_outbox WHERE status = 'pending' AND user_identity = %s`, s.placeholder(1))
	rows, err := s.db.QueryContext(ctx, query, userIdentity)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		return 0, rows.Err()
	}
	var count int
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, rows.Err()
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

var (
	_ OutboxStore    = (*SQLStore)(nil)
	_ PendingCounter = (*SQLStore)(nil)
)
