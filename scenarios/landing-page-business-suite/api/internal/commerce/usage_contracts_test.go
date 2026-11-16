package commerce

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

func TestNormalizeUsageReportCanonicalizesBYOK(t *testing.T) {
	report, err := NormalizeUsageReport(UsageReportRequest{UserIdentity: " USER@example.COM ", LimitKey: " AI_CREDITS ", AppBundleKey: " APP ", Amount: 42, IsBYOK: true})
	if err != nil {
		t.Fatalf("NormalizeUsageReport() error = %v", err)
	}
	if report.UserIdentity != "user@example.com" || report.LimitKey != "ai_credits" || report.AppBundleKey != "app" || report.Amount != 0 {
		t.Fatalf("normalized report = %#v", report)
	}
}

func TestNormalizeUsageReportRejectsUnmeteredRequest(t *testing.T) {
	if _, err := NormalizeUsageReport(UsageReportRequest{UserIdentity: "user", LimitKey: "credits"}); err == nil {
		t.Fatal("NormalizeUsageReport() succeeded for zero non-BYOK amount")
	}
}

func TestRecordUsageRejectsCallerSuppliedAppScope(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE usage_records (user_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL, app_bundle_key TEXT, operation_id TEXT, last_operation_at TEXT, updated_at TEXT, UNIQUE(user_identity, billing_period, limit_key, app_bundle_key))`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	opID := "op-record-1"
	service := NewUsageServiceWithOptions(UsageServiceOptions{DB: db, Dialect: "sqlite"})
	err = service.RecordUsage(context.Background(), UsageReportRequest{
		UserIdentity: "User@example.com",
		LimitKey:     "AI_CREDITS",
		Amount:       17,
		AppBundleKey: "My-App",
		OperationID:  &opID,
		Metadata: map[string]string{
			"model":             "openrouter/test-model",
			"cost_micros":       "1234",
			"prompt_tokens":     "10",
			"completion_tokens": "5",
			"provider":          "openrouter",
		},
	})
	if !errors.Is(err, ErrDirectUsageScopeUnavailable) {
		t.Fatalf("RecordUsage error=%v, want fail-closed app scope refusal", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM usage_records`).Scan(&count); err != nil {
		t.Fatalf("count usage records: %v", err)
	}
	if count != 0 {
		t.Fatalf("unverified app billing scope wrote %d usage record(s)", count)
	}
}
