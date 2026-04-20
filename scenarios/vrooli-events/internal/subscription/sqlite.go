// DOC: docs/guides/creating-subscriptions.md
// DOC: docs/internal/INVARIANTS.md
package subscription

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/vrooli/vrooli/scenarios/vrooli-events/internal/sqlutil"
	_ "modernc.org/sqlite"
)

const subscriptionSchema = `
CREATE TABLE IF NOT EXISTS subscriptions (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  owner_scenario TEXT NOT NULL,
  event_pattern TEXT NOT NULL,
  source_filter TEXT NOT NULL DEFAULT '',
  delivery_type TEXT NOT NULL CHECK(delivery_type IN ('sse','webhook')),
  delivery_target TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now')),
  updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%f','now'))
);
CREATE INDEX IF NOT EXISTS idx_subscriptions_owner ON subscriptions(owner_scenario);
CREATE INDEX IF NOT EXISTS idx_subscriptions_pattern ON subscriptions(event_pattern);

CREATE TABLE IF NOT EXISTS subscription_health (
  subscription_id INTEGER PRIMARY KEY REFERENCES subscriptions(id) ON DELETE CASCADE,
  total_delivered INTEGER NOT NULL DEFAULT 0,
  total_failed INTEGER NOT NULL DEFAULT 0,
  consecutive_failures INTEGER NOT NULL DEFAULT 0,
  last_delivered_at TEXT NOT NULL DEFAULT '',
  last_failed_at TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active'
);
`

// SQLiteStore implements subscription.Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates subscription tables in the provided database.
func NewSQLiteStore(db *sql.DB) (*SQLiteStore, error) {
	if _, err := db.Exec(subscriptionSchema); err != nil {
		return nil, fmt.Errorf("apply subscription schema: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Create(ctx context.Context, sub Subscription) (int64, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO subscriptions (name, owner_scenario, event_pattern, source_filter, delivery_type, delivery_target, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sub.Name, sub.OwnerScenario, sub.EventPattern, sub.SourceFilter,
		string(sub.DeliveryType), sub.DeliveryTarget, sqlutil.BoolToInt(sub.Enabled))
	if err != nil {
		return 0, fmt.Errorf("insert subscription: %w", err)
	}
	id, _ := res.LastInsertId()

	// Initialize health record
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO subscription_health (subscription_id) VALUES (?)`, id)
	if err != nil {
		return 0, fmt.Errorf("init health: %w", err)
	}

	return id, nil
}

func (s *SQLiteStore) Get(ctx context.Context, id int64) (Subscription, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, owner_scenario, event_pattern, source_filter, delivery_type, delivery_target, enabled, created_at, updated_at
		 FROM subscriptions WHERE id = ?`, id)
	return scanSubscriptionFrom(row)
}

func (s *SQLiteStore) List(ctx context.Context, f ListFilters) ([]Subscription, error) {
	var clauses []string
	var args []any

	if f.Owner != "" {
		clauses = append(clauses, "owner_scenario = ?")
		args = append(args, f.Owner)
	}
	if f.Pattern != "" {
		clauses = append(clauses, "event_pattern = ?")
		args = append(args, f.Pattern)
	}
	if f.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, sqlutil.BoolToInt(*f.Enabled))
	}

	query := `SELECT id, name, owner_scenario, event_pattern, source_filter, delivery_type, delivery_target, enabled, created_at, updated_at FROM subscriptions`
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		sub, err := scanSubscriptionFrom(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

func (s *SQLiteStore) Update(ctx context.Context, sub Subscription) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE subscriptions SET
		  name=?, owner_scenario=?, event_pattern=?, source_filter=?, delivery_type=?, delivery_target=?, enabled=?,
		  updated_at=strftime('%Y-%m-%dT%H:%M:%f','now')
		 WHERE id=?`,
		sub.Name, sub.OwnerScenario, sub.EventPattern, sub.SourceFilter,
		string(sub.DeliveryType), sub.DeliveryTarget, sqlutil.BoolToInt(sub.Enabled), sub.ID)
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (s *SQLiteStore) Delete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetHealth(ctx context.Context, id int64) (Health, error) {
	var h Health
	err := s.db.QueryRowContext(ctx,
		`SELECT subscription_id, total_delivered, total_failed, consecutive_failures, last_delivered_at, last_failed_at, status
		 FROM subscription_health WHERE subscription_id = ?`, id).Scan(
		&h.SubscriptionID, &h.TotalDelivered, &h.TotalFailed,
		&h.ConsecutiveFailures, &h.LastDeliveredAt, &h.LastFailedAt, &h.Status)
	if err != nil {
		return h, err
	}
	return h, nil
}

func (s *SQLiteStore) Close() error {
	return nil // DB lifecycle managed externally
}

// subscriptionScanner abstracts *sql.Row and *sql.Rows so scan logic is shared.
type subscriptionScanner interface {
	Scan(dest ...any) error
}

// scanSubscriptionFrom scans a single subscription from any scanner (*sql.Row or *sql.Rows).
func scanSubscriptionFrom(sc subscriptionScanner) (Subscription, error) {
	var sub Subscription
	var dt, createdAt, updatedAt string
	var enabled int
	err := sc.Scan(&sub.ID, &sub.Name, &sub.OwnerScenario, &sub.EventPattern,
		&sub.SourceFilter, &dt, &sub.DeliveryTarget, &enabled, &createdAt, &updatedAt)
	if err != nil {
		return sub, err
	}
	sub.DeliveryType = DeliveryType(dt)
	sub.Enabled = enabled != 0
	sub.CreatedAt = sqlutil.ParseTime(createdAt)
	sub.UpdatedAt = sqlutil.ParseTime(updatedAt)
	return sub, nil
}
