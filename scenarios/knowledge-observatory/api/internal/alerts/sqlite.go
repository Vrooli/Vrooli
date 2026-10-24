package alerts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/sqlitetime"
)

// SQLite implements Repository against SQLite.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ Repository = (*SQLite)(nil)

// NewSQLite returns a Repository backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

func (s *SQLite) Insert(ctx context.Context, a Alert) (string, error) {
	if s == nil || s.DB == nil {
		return "", fmt.Errorf("alerts repository not configured")
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO alerts
  (id, level, collection_name, metric_name, threshold_value, actual_value,
   message, acknowledged, acknowledged_at, created_at)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING
`, a.ID, strings.TrimSpace(a.Level), strings.TrimSpace(a.CollectionName),
		strings.TrimSpace(a.MetricName), a.ThresholdValue, a.ActualValue,
		strings.TrimSpace(a.Message), a.Acknowledged, sqlitetime.FormatPtr(a.AcknowledgedAt))
	if err != nil {
		return "", fmt.Errorf("insert alert: %w", err)
	}
	return a.ID, nil
}

func (s *SQLite) Get(ctx context.Context, id string) (Alert, bool, error) {
	if s == nil || s.DB == nil {
		return Alert{}, false, fmt.Errorf("alerts repository not configured")
	}
	var (
		a    Alert
		ackd sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, level, COALESCE(collection_name, ''), COALESCE(metric_name, ''),
       threshold_value, actual_value, COALESCE(message, ''), acknowledged,
       acknowledged_at, created_at
FROM alerts
WHERE id = ?
`, strings.TrimSpace(id)).Scan(&a.ID, &a.Level, &a.CollectionName, &a.MetricName,
		&a.ThresholdValue, &a.ActualValue, &a.Message, &a.Acknowledged, &ackd, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return Alert{}, false, nil
	}
	if err != nil {
		return Alert{}, false, fmt.Errorf("get alert: %w", err)
	}
	if ackd.Valid {
		t := ackd.Time.UTC()
		a.AcknowledgedAt = &t
	}
	return a, true, nil
}

func (s *SQLite) ListUnacknowledged(ctx context.Context, limit int) ([]Alert, error) {
	if s == nil || s.DB == nil {
		return nil, fmt.Errorf("alerts repository not configured")
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
SELECT id, level, COALESCE(collection_name, ''), COALESCE(metric_name, ''),
       threshold_value, actual_value, COALESCE(message, ''), acknowledged,
       acknowledged_at, created_at
FROM alerts
WHERE acknowledged = 0
ORDER BY created_at DESC, id DESC
LIMIT ?
`, limit)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var out []Alert
	for rows.Next() {
		var (
			a    Alert
			ackd sql.NullTime
		)
		if err := rows.Scan(&a.ID, &a.Level, &a.CollectionName, &a.MetricName,
			&a.ThresholdValue, &a.ActualValue, &a.Message, &a.Acknowledged, &ackd, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		if ackd.Valid {
			t := ackd.Time.UTC()
			a.AcknowledgedAt = &t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *SQLite) Acknowledge(ctx context.Context, id string, at time.Time) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("alerts repository not configured")
	}
	_, err := s.DB.ExecContext(ctx,
		`UPDATE alerts SET acknowledged = 1, acknowledged_at = ? WHERE id = ?`,
		sqlitetime.Format(at), strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("acknowledge alert: %w", err)
	}
	return nil
}
