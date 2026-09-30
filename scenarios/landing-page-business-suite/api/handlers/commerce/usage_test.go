package billing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func usageTestDependencies(t *testing.T, user string) (UsageDependencies, *int, *string) {
	t.Helper()
	status := 0
	message := ""
	return UsageDependencies{
		UserEmail: func(context.Context) string { return user },
		WriteError: func(_ http.ResponseWriter, gotStatus int, gotMessage, _ string) {
			status, message = gotStatus, gotMessage
		},
		LogError: func(string, map[string]any) {},
	}, &status, &message
}

func TestReportUsageRequiresVerifiedIdentityBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "")
	req := httptest.NewRequest(http.MethodPost, "/usage/report", nil)
	ReportUsage(nil, deps).ServeHTTP(httptest.NewRecorder(), req)
	if *status != http.StatusUnauthorized || *message != "Authentication required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestReportUsageRejectsMalformedJSONBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "customer@example.com")
	req := httptest.NewRequest(http.MethodPost, "/usage/report", nil)
	ReportUsage(nil, deps).ServeHTTP(httptest.NewRecorder(), req)
	if *status != http.StatusBadRequest || *message != "Invalid request body" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestUsageSummaryRequiresAuthenticatedIdentity(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "")
	GetUsageSummary(nil, nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/usage/summary", nil))
	if *status != http.StatusUnauthorized || *message != "Authentication required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}

func TestCheckLimitValidatesLimitKeyBeforeService(t *testing.T) {
	deps, status, message := usageTestDependencies(t, "customer@example.com")
	CheckLimit(nil, deps).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/usage/check", nil))
	if *status != http.StatusBadRequest || *message != "limit_key is required" {
		t.Fatalf("status=%d message=%q", *status, *message)
	}
}
