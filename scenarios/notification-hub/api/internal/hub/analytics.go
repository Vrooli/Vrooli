package hub

import (
	"context"
	"database/sql"
	"fmt"
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
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifications WHERE created_at >= ? AND created_at <= ?`, fromText, toText).Scan(&total); err != nil {
		return Analytics{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT channel, COUNT(*), SUM(CASE WHEN outcome='delivered' THEN 1 ELSE 0 END), SUM(CASE WHEN outcome='failed' THEN 1 ELSE 0 END) FROM delivery_attempts WHERE created_at >= ? AND created_at <= ? GROUP BY channel ORDER BY channel`, fromText, toText)
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
		if err := s.db.QueryRowContext(ctx, `SELECT AVG((julianday(r.delivered_at)-julianday(a.created_at))*86400000.0) FROM delivery_attempts a JOIN receipts r ON r.notification_id=a.notification_id AND r.channel=a.channel WHERE a.channel=? AND a.created_at >= ? AND a.created_at <= ?`, channels[i].Channel, fromText, toText).Scan(&avg); err != nil {
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
