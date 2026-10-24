package hub

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type AnalyticsChannel struct {
	Channel              string
	Delivered, Failed    int64
	Attempts             int64
	FailureRate          float64
	AverageLatencyMillis float64
}

type Analytics struct {
	Since, Until       string
	Channels           []AnalyticsChannel
	TotalNotifications int64
}

func (s *Service) Analytics(ctx context.Context, since, until string) (Analytics, error) {
	return s.analytics(ctx, since, until, "", false)
}

// AnalyticsForRecipient returns counts and delivery aggregates only for the
// configured recipient. An empty recipient produces an empty, dated report.
func (s *Service) AnalyticsForRecipient(ctx context.Context, since, until, recipient string) (Analytics, error) {
	return s.analytics(ctx, since, until, recipient, true)
}

func (s *Service) analytics(ctx context.Context, since, until, recipient string, scoped bool) (Analytics, error) {
	now := s.clock.Now().UTC()
	from, err := parseAnalyticsTime(since, now.Add(-24*time.Hour))
	if err != nil {
		return Analytics{}, fmt.Errorf("invalid since: %w", err)
	}
	to, err := parseAnalyticsTime(until, now)
	if err != nil {
		return Analytics{}, fmt.Errorf("invalid until: %w", err)
	}
	fromText, toText := from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano)
	if scoped {
		recipient = strings.TrimSpace(recipient)
		if recipient == "" {
			return Analytics{Since: fromText, Until: toText}, nil
		}
	}
	var total int64
	totalQuery := `SELECT COUNT(*) FROM notifications WHERE created_at >= ? AND created_at <= ?`
	totalArgs := []any{fromText, toText}
	if scoped {
		totalQuery = `SELECT COUNT(*) FROM notifications WHERE requested_by = ? AND created_at >= ? AND created_at <= ?`
		totalArgs = []any{recipient, fromText, toText}
	}
	if err := s.db.QueryRowContext(ctx, totalQuery, totalArgs...).Scan(&total); err != nil {
		return Analytics{}, err
	}
	attemptsQuery := `SELECT channel, COUNT(*), SUM(CASE WHEN outcome='delivered' THEN 1 ELSE 0 END), SUM(CASE WHEN outcome='failed' THEN 1 ELSE 0 END) FROM delivery_attempts WHERE created_at >= ? AND created_at <= ? GROUP BY channel ORDER BY channel`
	attemptsArgs := []any{fromText, toText}
	if scoped {
		attemptsQuery = `SELECT a.channel, COUNT(*), SUM(CASE WHEN a.outcome='delivered' THEN 1 ELSE 0 END), SUM(CASE WHEN a.outcome='failed' THEN 1 ELSE 0 END) FROM delivery_attempts a JOIN notifications n ON n.id = a.notification_id WHERE n.requested_by = ? AND a.created_at >= ? AND a.created_at <= ? GROUP BY a.channel ORDER BY a.channel`
		attemptsArgs = []any{recipient, fromText, toText}
	}
	rows, err := s.db.QueryContext(ctx, attemptsQuery, attemptsArgs...)
	if err != nil {
		return Analytics{}, err
	}
	defer rows.Close()
	var channels []AnalyticsChannel
	for rows.Next() {
		var item AnalyticsChannel
		if err := rows.Scan(&item.Channel, &item.Attempts, &item.Delivered, &item.Failed); err != nil {
			return Analytics{}, err
		}
		if item.Attempts > 0 {
			item.FailureRate = float64(item.Failed) / float64(item.Attempts)
		}
		channels = append(channels, item)
	}
	if err := rows.Err(); err != nil {
		return Analytics{}, err
	}
	// Materialize the grouped rows before issuing the per-channel latency
	// queries. RoutedDB intentionally uses one SQLite connection in tests and
	// production, so querying while the aggregate cursor is open would block.
	for i := range channels {
		var avg sql.NullFloat64
		latencyQuery := `SELECT AVG((julianday(r.delivered_at)-julianday(a.created_at))*86400000.0) FROM delivery_attempts a JOIN receipts r ON r.notification_id=a.notification_id AND r.channel=a.channel WHERE a.channel=? AND a.created_at >= ? AND a.created_at <= ?`
		latencyArgs := []any{channels[i].Channel, fromText, toText}
		if scoped {
			latencyQuery = `SELECT AVG((julianday(r.delivered_at)-julianday(a.created_at))*86400000.0) FROM delivery_attempts a JOIN notifications n ON n.id=a.notification_id JOIN receipts r ON r.notification_id=a.notification_id AND r.channel=a.channel WHERE n.requested_by=? AND a.channel=? AND a.created_at >= ? AND a.created_at <= ?`
			latencyArgs = []any{recipient, channels[i].Channel, fromText, toText}
		}
		if err := s.db.QueryRowContext(ctx, latencyQuery, latencyArgs...).Scan(&avg); err != nil {
			return Analytics{}, err
		}
		if avg.Valid {
			channels[i].AverageLatencyMillis = avg.Float64
		}
	}
	return Analytics{Since: fromText, Until: toText, Channels: channels, TotalNotifications: total}, nil
}

func parseAnalyticsTime(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	return time.Parse(time.RFC3339, value)
}
