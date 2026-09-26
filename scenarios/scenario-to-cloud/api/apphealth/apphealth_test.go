package apphealth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// lpbsDegradedBody is the live vrooli.com response shape: HTTP 200, an
// application-level "degraded" status, and one failing dependency. A deploy
// that reads only the status code cannot tell this from healthy.
const lpbsDegradedBody = `{"status":"degraded","service":"landing-page-business-suite-api","readiness":true,
"dependencies":{"database":{"connected":true,"latency_ms":0.9},
"sign_in_email":{"connected":false,"latency_ms":1.7,"error":"1 of the last 1 sign-in emails failed in the past hour; latest: smtp: 535 \"Authentication failed\""}}}`

func TestParseReportsFailingDependencyAsWarning(t *testing.T) {
	report := Parse("https://example.com/health", []byte(lpbsDegradedBody), time.Now())

	if report.Unavailable {
		t.Fatalf("a parseable body must not be unavailable: %s", report.Reason)
	}
	if report.Status != "degraded" {
		t.Errorf("status = %q, want degraded", report.Status)
	}
	if len(report.Dependencies) != 2 {
		t.Fatalf("dependencies = %d, want 2: %+v", len(report.Dependencies), report.Dependencies)
	}
	// Sorted by name: database first, sign_in_email second.
	if report.Dependencies[0].Name != "database" || report.Dependencies[0].Status != DependencyPass {
		t.Errorf("first dependency = %+v, want database pass", report.Dependencies[0])
	}
	warnings := report.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("warnings = %d, want 1: %+v", len(warnings), warnings)
	}
	if warnings[0].Name != "sign_in_email" {
		t.Errorf("warning name = %q, want sign_in_email", warnings[0].Name)
	}
	if warnings[0].Detail == "" {
		t.Error("a failing dependency must carry the application's own reason")
	}
}

func TestParseHealthyBodyHasNoWarnings(t *testing.T) {
	body := `{"status":"healthy","dependencies":{"database":{"connected":true},"sign_in_email":{"connected":true}}}`

	report := Parse("https://example.com/health", []byte(body), time.Now())

	if len(report.Warnings()) != 0 {
		t.Errorf("warnings = %+v, want none", report.Warnings())
	}
	if got, want := report.Summary(), "2 of 2 application dependencies pass"; got != want {
		t.Errorf("summary = %q, want %q", got, want)
	}
}

func TestParseAcceptsFlatCheckList(t *testing.T) {
	body := `{"checks":[{"name":"spf","status":"pass"},{"name":"dmarc","status":"warn","detail":"monitoring mode"}]}`

	report := Parse("https://example.com/readiness", []byte(body), time.Now())

	if report.Unavailable {
		t.Fatalf("a check list must parse: %s", report.Reason)
	}
	warnings := report.Warnings()
	if len(warnings) != 1 || warnings[0].Name != "dmarc" {
		t.Fatalf("warnings = %+v, want dmarc only", warnings)
	}
	if warnings[0].Detail != "monitoring mode" {
		t.Errorf("detail = %q, want the reported detail", warnings[0].Detail)
	}
}

func TestParseUnreadableBodyIsUnavailableNotPassing(t *testing.T) {
	for name, body := range map[string]string{
		"empty":       "",
		"not json":    "<html>maintenance</html>",
		"no evidence": `{"service":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			report := Parse("https://example.com/health", []byte(body), time.Now())

			if !report.Unavailable {
				t.Fatalf("body %q must be unavailable, got %+v", body, report)
			}
			if len(report.Warnings()) != 0 {
				t.Error("an unobserved report must not invent warnings")
			}
			if report.Reason == "" {
				t.Error("unavailable must explain itself")
			}
		})
	}
}

func TestParseTreatsUnknownStatusWordAsWarning(t *testing.T) {
	body := `{"status":"healthy","dependencies":{"queue":{"status":"who knows"}}}`

	report := Parse("https://example.com/health", []byte(body), time.Now())

	if len(report.Warnings()) != 1 {
		t.Errorf("an unreadable dependency status must warn, got %+v", report.Dependencies)
	}
}

func TestFetchReadsBodyOfA200Response(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(lpbsDegradedBody))
	}))
	defer server.Close()

	report := Fetch(context.Background(), server.Client(), server.URL+"/health", 5*time.Second)

	if report.Unavailable {
		t.Fatalf("a 200 with a readable body must be observed: %s", report.Reason)
	}
	if len(report.Warnings()) != 1 {
		t.Errorf("warnings = %+v, want the failing mail dependency", report.Warnings())
	}
}

func TestFetchUnreachableEndpointIsUnavailable(t *testing.T) {
	report := Fetch(context.Background(), &http.Client{Timeout: time.Second}, "http://127.0.0.1:1/health", time.Second)

	if !report.Unavailable {
		t.Fatalf("an unreachable endpoint must be unavailable, got %+v", report)
	}
}

func TestFetchNonSuccessStatusKeepsEvidenceButRecordsReason(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(lpbsDegradedBody))
	}))
	defer server.Close()

	report := Fetch(context.Background(), server.Client(), server.URL+"/health", 5*time.Second)

	if len(report.Warnings()) != 1 {
		t.Errorf("disclosed dependencies must be kept, got %+v", report.Dependencies)
	}
	if report.Reason == "" {
		t.Error("a non-2xx health response must be recorded as the reason")
	}
}

func TestFetchWithoutURLIsUnavailable(t *testing.T) {
	report := Fetch(context.Background(), nil, "  ", time.Second)

	if !report.Unavailable || report.Reason == "" {
		t.Fatalf("a missing URL must be an explained unavailable, got %+v", report)
	}
}
