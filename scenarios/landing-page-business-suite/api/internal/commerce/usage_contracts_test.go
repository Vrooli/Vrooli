package commerce

import "testing"

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
