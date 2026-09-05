package integrationstatus

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestScenarioCheckerReportsConfiguredAvailability(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	status, err := (ScenarioChecker{Scenario: "plan-manager", Required: true, DegradedBehavior: "plan work is parked", ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Now: func() time.Time { return now }, FreshFor: time.Minute}).Check(context.Background())
	if err != nil || !status.Configured || status.Availability != Available || !status.FreshUntil.Equal(now.Add(time.Minute)) {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}

func TestScenarioCheckerReportsUnconfiguredWithoutTransportError(t *testing.T) {
	status, err := (ScenarioChecker{Scenario: "missing", DegradedBehavior: "work is parked", ResolveURL: func(context.Context, string) (string, error) { return "", nil }}).Check(context.Background())
	if err != nil || status.Availability != Unconfigured || status.Configured {
		t.Fatalf("status=%#v err=%v", status, err)
	}
}
