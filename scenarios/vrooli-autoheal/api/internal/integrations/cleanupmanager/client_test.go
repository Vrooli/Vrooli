package cleanupmanager

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestClientResolvesStorageManagerAfterRestart(t *testing.T) {
	t.Setenv("VROOLI_CLEANUP_MANAGER_API_URL", "")
	var firstCalls, secondCalls atomic.Int32
	makeServer := func(calls *atomic.Int32) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != reportPressureProcedure {
				t.Errorf("path = %q, want %q", r.URL.Path, reportPressureProcedure)
			}
			calls.Add(1)
			_ = json.NewEncoder(w).Encode(Outcome{Action: "preview"})
		}))
	}
	first, second := makeServer(&firstCalls), makeServer(&secondCalls)
	defer first.Close()
	defer second.Close()
	addresses := []string{first.URL, second.URL}
	var resolves atomic.Int32
	client := NewClient(Config{Resolve: func(context.Context) (string, error) {
		return addresses[int(resolves.Add(1))-1], nil
	}})
	for range 2 {
		if _, err := client.ReportPressure(context.Background(), Report{Band: BandWarning}); err != nil {
			t.Fatalf("ReportPressure() error = %v", err)
		}
	}
	if resolves.Load() != 2 || firstCalls.Load() != 1 || secondCalls.Load() != 1 {
		t.Fatalf("resolutions/first requests/second requests = %d/%d/%d, want 2/1/1", resolves.Load(), firstCalls.Load(), secondCalls.Load())
	}
}

func TestClientReportsDiscoveryFailureWithoutCallingPeer(t *testing.T) {
	t.Setenv("VROOLI_CLEANUP_MANAGER_API_URL", "")
	client := NewClient(Config{Resolve: func(context.Context) (string, error) {
		return "", errors.New("storage-manager unavailable")
	}})
	if _, err := client.ReportPressure(context.Background(), Report{}); err == nil {
		t.Fatal("ReportPressure() error = nil, want discovery failure")
	}
}
