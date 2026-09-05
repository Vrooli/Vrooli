// Package measures owns the fixed persistence catalog used by the scenario's
// analytical measures. Query selection is deliberately closed: a caller can
// name a reviewed measure, but can never supply a table or SQL fragment.
package measures

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Counter is the narrow database seam used by SQLRepository. *sql.DB satisfies
// it; focused tests can provide a recording fake without a database driver.
type Counter interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Repository provides durable aggregate counts for named measures.
type Repository interface {
	CountCreated(context.Context, string, time.Time, time.Time) (int64, error)
	QueryFor(string) (string, bool)
}

// SQLRepository executes only queries from catalog. It intentionally has no
// method that accepts arbitrary SQL or a table name.
type SQLRepository struct {
	db Counter
}

func NewSQLRepository(db Counter) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) CountCreated(ctx context.Context, measure string, from, to time.Time) (int64, error) {
	query, ok := r.QueryFor(measure)
	if !ok {
		return 0, fmt.Errorf("unknown measure %q", measure)
	}
	var result int64
	if err := r.db.QueryRowContext(ctx, query, from, to).Scan(&result); err != nil {
		return 0, err
	}
	return result, nil
}

func (r *SQLRepository) QueryFor(measure string) (string, bool) {
	query, ok := createdCountQueries[measure]
	return query, ok
}

var createdCountQueries = map[string]string{
	"subscriptions.created":             "SELECT COUNT(*) FROM subscriptions WHERE created_at >= $1 AND created_at < $2",
	"credit_transactions.created":       "SELECT COUNT(*) FROM credit_transactions WHERE created_at >= $1 AND created_at < $2",
	"checkout_sessions.created":         "SELECT COUNT(*) FROM checkout_sessions WHERE created_at >= $1 AND created_at < $2",
	"bundle_products.created":           "SELECT COUNT(*) FROM bundle_products WHERE created_at >= $1 AND created_at < $2",
	"bundle_prices.created":             "SELECT COUNT(*) FROM bundle_prices WHERE created_at >= $1 AND created_at < $2",
	"subscription_schedules.created":    "SELECT COUNT(*) FROM subscription_schedules WHERE created_at >= $1 AND created_at < $2",
	"intro_coupon_usage.created":        "SELECT COUNT(*) FROM intro_coupon_usage WHERE created_at >= $1 AND created_at < $2",
	"payment_anomaly_log.created":       "SELECT COUNT(*) FROM payment_anomaly_log WHERE created_at >= $1 AND created_at < $2",
	"users.created":                     "SELECT COUNT(*) FROM users WHERE created_at >= $1 AND created_at < $2",
	"user_sessions.created":             "SELECT COUNT(*) FROM user_sessions WHERE created_at >= $1 AND created_at < $2",
	"auth_tokens.created":               "SELECT COUNT(*) FROM auth_tokens WHERE created_at >= $1 AND created_at < $2", // #nosec G101 -- fixed database table identifier, never a credential
	"api_keys.created":                  "SELECT COUNT(*) FROM api_keys WHERE created_at >= $1 AND created_at < $2",
	"credit_reservations.created":       "SELECT COUNT(*) FROM credit_reservations WHERE created_at >= $1 AND created_at < $2",
	"subscription_tier_limits.created":  "SELECT COUNT(*) FROM subscription_tier_limits WHERE created_at >= $1 AND created_at < $2",
	"usage_records.created":             "SELECT COUNT(*) FROM usage_records WHERE created_at >= $1 AND created_at < $2",
	"admin_sessions.created":            "SELECT COUNT(*) FROM admin_sessions WHERE created_at >= $1 AND created_at < $2",
	"admin_users.created":               "SELECT COUNT(*) FROM admin_users WHERE created_at >= $1 AND created_at < $2",
	"assets.created":                    "SELECT COUNT(*) FROM assets WHERE created_at >= $1 AND created_at < $2",
	"download_apps.created":             "SELECT COUNT(*) FROM download_apps WHERE created_at >= $1 AND created_at < $2",
	"download_artifacts.created":        "SELECT COUNT(*) FROM download_artifacts WHERE created_at >= $1 AND created_at < $2",
	"download_assets.created":           "SELECT COUNT(*) FROM download_assets WHERE created_at >= $1 AND created_at < $2",
	"download_storage_settings.created": "SELECT COUNT(*) FROM download_storage_settings WHERE created_at >= $1 AND created_at < $2",
	"feedback_requests.created":         "SELECT COUNT(*) FROM feedback_requests WHERE created_at >= $1 AND created_at < $2",
	"metrics_events.created":            "SELECT COUNT(*) FROM metrics_events WHERE created_at >= $1 AND created_at < $2",
	"remote_profiles.created":           "SELECT COUNT(*) FROM remote_profiles WHERE created_at >= $1 AND created_at < $2",
	"waitlist_emails.created":           "SELECT COUNT(*) FROM waitlist_emails WHERE created_at >= $1 AND created_at < $2",
}
