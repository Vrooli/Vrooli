package vps

import (
	"context"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/apphealth"
)

func withReadinessAppHealth(t *testing.T, report apphealth.Report) {
	t.Helper()
	original := appHealthFetchForReadiness
	appHealthFetchForReadiness = func(context.Context, string) apphealth.Report { return report }
	t.Cleanup(func() { appHealthFetchForReadiness = original })
}

func TestAppDependencyWarningNamesFailingDependencies(t *testing.T) {
	withReadinessAppHealth(t, apphealth.Parse("https://example.com/health", []byte(
		`{"status":"degraded","dependencies":{"sign_in_email":{"connected":false,"error":"smtp: 535"}}}`), time.Now()))

	got := appDependencyWarning(context.Background(), "example.com", "/api/v1/health", []string{"local", "https"})

	if !strings.Contains(got, "sign_in_email") {
		t.Errorf("warning = %q, want the failing dependency named", got)
	}
	if !strings.Contains(got, "smtp: 535") {
		t.Errorf("warning = %q, want the application's own reason", got)
	}
}

func TestAppDependencyWarningSilentWhenEveryDependencyPasses(t *testing.T) {
	withReadinessAppHealth(t, apphealth.Parse("https://example.com/health", []byte(
		`{"status":"healthy","dependencies":{"database":{"connected":true}}}`), time.Now()))

	if got := appDependencyWarning(context.Background(), "example.com", "/api/v1/health", []string{"https"}); got != "" {
		t.Errorf("warning = %q, want none", got)
	}
}

// The readiness checks already proved the application answers, so an
// unreadable body is worth saying out loud rather than passing silently.
func TestAppDependencyWarningReportsUnreadableBody(t *testing.T) {
	withReadinessAppHealth(t, apphealth.Parse("https://example.com/health", []byte("<html>"), time.Now()))

	got := appDependencyWarning(context.Background(), "example.com", "/api/v1/health", []string{"public"})

	if !strings.Contains(got, "not observed") {
		t.Errorf("warning = %q, want an explicit not-observed warning", got)
	}
}

// A local-only readiness step never reached the public endpoint, so it must
// not claim anything about the application's dependencies.
func TestAppDependencyWarningSkippedWithoutAPublicCheck(t *testing.T) {
	withReadinessAppHealth(t, apphealth.Parse("https://example.com/health", []byte(
		`{"status":"degraded","dependencies":{"sign_in_email":{"connected":false}}}`), time.Now()))

	if got := appDependencyWarning(context.Background(), "example.com", "/api/v1/health", []string{"local"}); got != "" {
		t.Errorf("warning = %q, want none for a local-only readiness check", got)
	}
	if got := appDependencyWarning(context.Background(), "", "/api/v1/health", []string{"https"}); got != "" {
		t.Errorf("warning = %q, want none without a domain", got)
	}
}
