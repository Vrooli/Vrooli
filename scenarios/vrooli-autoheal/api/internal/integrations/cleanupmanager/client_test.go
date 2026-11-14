package cleanupmanager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientReportPressure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != reportPressureProcedure {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		var report Report
		if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
			t.Fatalf("decode report: %v", err)
		}
		if report.Band != BandCritical || report.UsedPercent != 97.5 {
			t.Fatalf("report = %+v", report)
		}
		_ = json.NewEncoder(w).Encode(Outcome{Action: "preview", PlanID: "plan-1"})
	}))
	defer server.Close()

	outcome, err := NewClient(Config{BaseURL: server.URL}).ReportPressure(context.Background(), Report{
		SourceScenario: "vrooli-autoheal",
		Partition:      "/",
		UsedPercent:    97.5,
		Band:           BandCritical,
	})
	if err != nil {
		t.Fatalf("ReportPressure() error = %v", err)
	}
	if outcome.Action != "preview" || outcome.PlanID != "plan-1" {
		t.Fatalf("outcome = %+v", outcome)
	}
}

func TestClientReportPressureRejectsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "no", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := NewClient(Config{BaseURL: server.URL}).ReportPressure(context.Background(), Report{})
	if err == nil {
		t.Fatal("ReportPressure() error = nil, want HTTP failure")
	}
}
