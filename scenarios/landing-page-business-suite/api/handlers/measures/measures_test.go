package measures

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"
	_ "github.com/mattn/go-sqlite3"
	measurelib "github.com/vrooli/measures-go"

	lpbsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-business-suite/v1/measures"
	measuresv1 "github.com/vrooli/vrooli/packages/proto/gen/go/measures/v1"
)

func TestSubscriptionMeasureRegistryAndConnectShareTheSameAggregate(t *testing.T) {
	db := newMeasureTestDB(t)
	fixedNow := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	insertCreatedAt(t, db, "subscriptions", time.Date(2026, time.July, 14, 9, 0, 0, 0, time.UTC))
	insertCreatedAt(t, db, "subscriptions", time.Date(2026, time.July, 12, 9, 0, 0, 0, time.UTC))

	registry, err := NewRegistry(db, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	result, err := registry.Execute(context.Background(), measurelib.MeasureRequest{
		Measure: MeasureSubscriptionsCreated,
		Params:  map[string]string{"window": string(measurelib.TokenThisWeek)},
	})
	if err != nil {
		t.Fatalf("registry Execute() error = %v", err)
	}
	if result.Value != "1" {
		t.Fatalf("registry count = %q, want 1", result.Value)
	}
	if result.Provenance.ExecutedQuery == "" || result.Provenance.ComputedAt != fixedNow {
		t.Fatalf("registry provenance = %+v, want query and fixed timestamp", result.Provenance)
	}

	handler := NewHandler(db, func() time.Time { return fixedNow })
	response, err := handler.CountSubscriptionsCreated(context.Background(), connect.NewRequest(&lpbsv1.CountSubscriptionsCreatedRequest{
		Window: tokenWindow(measuresv1.TimeWindowToken_TIME_WINDOW_TOKEN_THIS_WEEK),
	}))
	if err != nil {
		t.Fatalf("Connect CountSubscriptionsCreated() error = %v", err)
	}
	if response.Msg.GetCount() != 1 {
		t.Fatalf("Connect count = %d, want 1", response.Msg.GetCount())
	}
}

func TestCreditAndCheckoutMeasuresUseFixedDomainQueries(t *testing.T) {
	db := newMeasureTestDB(t)
	fixedNow := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	for _, table := range []string{"credit_transactions", "checkout_sessions"} {
		insertCreatedAt(t, db, table, time.Date(2026, time.July, 13, 1, 0, 0, 0, time.UTC))
		insertCreatedAt(t, db, table, time.Date(2026, time.July, 14, 1, 0, 0, 0, time.UTC))
	}

	handler := NewHandler(db, func() time.Time { return fixedNow })
	credits, err := handler.CountCreditTransactionsCreated(context.Background(), connect.NewRequest(&lpbsv1.CountCreditTransactionsCreatedRequest{}))
	if err != nil {
		t.Fatalf("CountCreditTransactionsCreated() error = %v", err)
	}
	checkout, err := handler.CountCheckoutSessionsCreated(context.Background(), connect.NewRequest(&lpbsv1.CountCheckoutSessionsCreatedRequest{}))
	if err != nil {
		t.Fatalf("CountCheckoutSessionsCreated() error = %v", err)
	}
	if credits.Msg.GetCount() != 2 || checkout.Msg.GetCount() != 2 {
		t.Fatalf("counts = credits:%d checkout:%d, want 2:2", credits.Msg.GetCount(), checkout.Msg.GetCount())
	}
}

func TestFinancialCatalogAndAuditMeasuresUseTheirAuthoritativeTables(t *testing.T) {
	db := newMeasureTestDB(t)
	fixedNow := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	for _, table := range []string{"bundle_products", "bundle_prices", "subscription_schedules", "intro_coupon_usage", "payment_anomaly_log"} {
		insertCreatedAt(t, db, table, time.Date(2026, time.July, 14, 1, 0, 0, 0, time.UTC))
	}

	registry, err := NewRegistry(db, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	for _, measure := range []string{
		MeasureBundleProductsCreated,
		MeasureBundlePricesCreated,
		MeasureSubscriptionSchedulesCreated,
		MeasureIntroCouponUsageCreated,
		MeasurePaymentAnomaliesCreated,
	} {
		result, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: measure, Params: map[string]string{"window": string(measurelib.TokenThisWeek)}})
		if err != nil {
			t.Fatalf("registry Execute(%s) error = %v", measure, err)
		}
		if result.Value != "1" || result.Provenance.ExecutedQuery == "" {
			t.Errorf("%s result = %+v, want count 1 with provenance", measure, result)
		}
	}
}

func TestCustomerAndUsageMeasuresUseTheirAuthoritativeTables(t *testing.T) {
	db := newMeasureTestDB(t)
	fixedNow := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	measures := map[string]string{
		MeasureUsersCreated: "users", MeasureUserSessionsCreated: "user_sessions", MeasureAuthTokensCreated: "auth_tokens",
		MeasureProviderKeysCreated: "api_keys", MeasureCreditReservationsCreated: "credit_reservations",
		MeasureSubscriptionTierLimitsCreated: "subscription_tier_limits", MeasureUsageRecordsCreated: "usage_records",
	}
	for _, table := range measures {
		insertCreatedAt(t, db, table, time.Date(2026, time.July, 14, 1, 0, 0, 0, time.UTC))
	}
	registry, err := NewRegistry(db, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	for measure := range measures {
		result, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: measure, Params: map[string]string{"window": string(measurelib.TokenThisWeek)}})
		if err != nil || result.Value != "1" {
			t.Errorf("Execute(%s) = %+v, %v; want count 1", measure, result, err)
		}
	}
}

func TestOperationsAndContentMeasuresUseTheirAuthoritativeTables(t *testing.T) {
	db := newMeasureTestDB(t)
	fixedNow := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	measures := map[string]string{
		MeasureAdminSessionsCreated: "admin_sessions", MeasureAdminUsersCreated: "admin_users",
		MeasureAssetsCreated: "assets", MeasureDownloadAppsCreated: "download_apps",
		MeasureDownloadArtifactsCreated: "download_artifacts", MeasureDownloadAssetsCreated: "download_assets",
		MeasureDownloadStorageSettingsCreated: "download_storage_settings", MeasureFeedbackRequestsCreated: "feedback_requests",
		MeasureMetricsEventsCreated: "metrics_events", MeasureRemoteProfilesCreated: "remote_profiles",
		MeasureWaitlistEmailsCreated: "waitlist_emails",
	}
	for _, table := range measures {
		insertCreatedAt(t, db, table, time.Date(2026, time.July, 14, 1, 0, 0, 0, time.UTC))
	}
	registry, err := NewRegistry(db, func() time.Time { return fixedNow })
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}
	for measure := range measures {
		result, err := registry.Execute(context.Background(), measurelib.MeasureRequest{Measure: measure, Params: map[string]string{"window": string(measurelib.TokenThisWeek)}})
		if err != nil || result.Value != "1" {
			t.Errorf("Execute(%s) = %+v, %v; want count 1", measure, result, err)
		}
	}
}

func newMeasureTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, table := range []string{
		"subscriptions", "credit_transactions", "checkout_sessions", "bundle_products", "bundle_prices",
		"subscription_schedules", "intro_coupon_usage", "payment_anomaly_log",
		"users", "user_sessions", "auth_tokens", "api_keys", "credit_reservations", "subscription_tier_limits", "usage_records",
		"admin_sessions", "admin_users", "assets", "download_apps", "download_artifacts", "download_assets",
		"download_storage_settings", "feedback_requests", "metrics_events", "remote_profiles", "waitlist_emails",
	} {
		if _, err := db.Exec("CREATE TABLE " + table + " (created_at DATETIME NOT NULL)"); err != nil {
			t.Fatalf("create %s: %v", table, err)
		}
	}
	return db
}

func insertCreatedAt(t *testing.T, db *sql.DB, table string, createdAt time.Time) {
	t.Helper()
	if _, err := db.Exec("INSERT INTO "+table+" (created_at) VALUES (?)", createdAt); err != nil {
		t.Fatalf("insert %s: %v", table, err)
	}
}

func tokenWindow(token measuresv1.TimeWindowToken) *measuresv1.TimeWindow {
	return &measuresv1.TimeWindow{Window: &measuresv1.TimeWindow_Token{Token: token}}
}
