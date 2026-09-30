package measures

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCreatedCountCatalogContainsEverySupportedMeasure(t *testing.T) {
	wantTables := map[string]string{
		"subscriptions.created": "subscriptions", "credit_transactions.created": "credit_transactions", "checkout_sessions.created": "checkout_sessions",
		"bundle_products.created": "bundle_products", "bundle_prices.created": "bundle_prices", "subscription_schedules.created": "subscription_schedules",
		"intro_coupon_usage.created": "intro_coupon_usage", "payment_anomaly_log.created": "payment_anomaly_log", "users.created": "users",
		"user_sessions.created": "user_sessions", "auth_tokens.created": "auth_tokens", "api_keys.created": "api_keys",
		"credit_reservations.created": "credit_reservations", "subscription_tier_limits.created": "subscription_tier_limits", "usage_records.created": "usage_records",
		"admin_sessions.created": "admin_sessions", "admin_users.created": "admin_users", "assets.created": "assets",
		"download_apps.created": "download_apps", "download_artifacts.created": "download_artifacts", "download_assets.created": "download_assets",
		"download_storage_settings.created": "download_storage_settings", "feedback_requests.created": "feedback_requests", "metrics_events.created": "metrics_events",
		"remote_profiles.created": "remote_profiles", "waitlist_emails.created": "waitlist_emails",
	}
	if len(createdCountQueries) != len(wantTables) {
		t.Fatalf("catalog size = %d, want %d", len(createdCountQueries), len(wantTables))
	}
	for measure, table := range wantTables {
		query, ok := createdCountQueries[measure]
		if !ok {
			t.Errorf("catalog missing %q", measure)
			continue
		}
		if !strings.Contains(query, "FROM "+table+" WHERE created_at >= $1 AND created_at < $2") {
			t.Errorf("catalog query for %q = %q, want fixed %s created-at aggregate", measure, query, table)
		}
	}
}

func TestSQLRepositoryRejectsUnknownMeasureBeforeDatabaseAccess(t *testing.T) {
	repository := NewSQLRepository(nil)
	_, err := repository.CountCreated(context.Background(), "unreviewed.created", time.Time{}, time.Time{})
	if err == nil || !strings.Contains(err.Error(), "unknown measure") {
		t.Fatalf("CountCreated() error = %v, want unknown-measure error", err)
	}
}
