package commerce

import (
	"context"
	"database/sql"
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

func TestRecordUsagePersistsProviderMetadataEvent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	for _, statement := range []string{
		`CREATE TABLE usage_records (user_identity TEXT NOT NULL, billing_period TEXT NOT NULL, limit_key TEXT NOT NULL, usage_amount INTEGER NOT NULL, app_bundle_key TEXT, operation_id TEXT, last_operation_at TEXT, updated_at TEXT, UNIQUE(user_identity, billing_period, limit_key, app_bundle_key))`,
		`CREATE TABLE usage_events (operation_id TEXT UNIQUE, user_identity TEXT NOT NULL, app_bundle_key TEXT NOT NULL, model TEXT NOT NULL, credits INTEGER NOT NULL, cost_micros INTEGER NOT NULL, prompt_tokens INTEGER NOT NULL, completion_tokens INTEGER NOT NULL, provider TEXT NOT NULL)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create test schema: %v", err)
		}
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
	if err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}

	var user, app, model, provider string
	var credits, cost, prompt, completion int64
	if err := db.QueryRow(`SELECT user_identity, app_bundle_key, model, credits, cost_micros, prompt_tokens, completion_tokens, provider FROM usage_events WHERE operation_id=?`, opID).
		Scan(&user, &app, &model, &credits, &cost, &prompt, &completion, &provider); err != nil {
		t.Fatalf("read usage event: %v", err)
	}
	if user != "user@example.com" || app != "my-app" || model != "openrouter/test-model" || provider != "openrouter" {
		t.Fatalf("event identity metadata = user=%q app=%q model=%q provider=%q", user, app, model, provider)
	}
	if credits != 17 || cost != 1234 || prompt != 10 || completion != 5 {
		t.Fatalf("event accounting = credits=%d cost=%d prompt=%d completion=%d", credits, cost, prompt, completion)
	}
}
