package forecasts

import (
	"strings"
	"testing"
	"time"
)

func TestNewServiceWithOptionsUsesInjectedIDGenerator(t *testing.T) {
	service, ok := NewServiceWithOptions(nil, nil, ServiceOptions{IDGenerator: func() string { return "forecast-test-id" }}).(*service)
	if !ok || service.id() != "forecast-test-id" {
		t.Fatalf("service did not retain injected ID generator: %#v", service)
	}
}

func TestBuildKeepsCentralAndCautiousScenariosExplainable(t *testing.T) {
	x, err := Build(Snapshot{StartDate: "2026-10-01", Timezone: "UTC", HorizonDays: 10, KnownWork: 600}, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if x.CentralFinish != "2026-10-02" || x.CautiousFinish != "2026-10-02" {
		t.Fatalf("unexpected finishes: %#v", x)
	}
	if x.RiskState != RiskOnTrack || x.ResultState != StateFeasible {
		t.Fatalf("unexpected state: %#v", x)
	}
	if x.InputFingerprint == "" || x.Explanation == "" {
		t.Fatal("forecast must carry fingerprint and explanation")
	}
}

func TestBuildDoesNotInventAHealthyForecastForUnknownWork(t *testing.T) {
	x, err := Build(Snapshot{StartDate: "2026-10-01", Timezone: "UTC", HorizonDays: 10, KnownWork: 0, UnresolvedWork: 2}, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if x.ResultState != StateNoKnownWork || x.RiskState != RiskUnknown || x.CentralFinish != "" {
		t.Fatalf("unexpected unknown state: %#v", x)
	}
}

func TestBuildReportsUnknownTimingWhenHorizonHasZeroUsableCapacity(t *testing.T) {
	x, err := Build(Snapshot{StartDate: "2026-11-01", Timezone: "UTC", HorizonDays: 2, KnownWork: 600, OccupiedMinutes: 840, Commitments: []CommitmentInput{{ID: "c-1", Result: "Deliver report", PromisedBoundary: "2026-11-02"}}}, time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if x.AvailableMinutes != 0 || x.ResultState != StateBeyondHorizon || x.RiskState == RiskOnTrack {
		t.Fatalf("zero-capacity horizon must be beyond horizon without an on-track claim: %#v", x)
	}
	if x.CentralFinish != "" || x.CautiousFinish != "" {
		t.Fatalf("zero-capacity horizon must not invent finish dates: %#v", x)
	}
	if x.Explanation == "" || !strings.Contains(x.Explanation, "no usable capacity") || !strings.Contains(x.Explanation, "unknown") {
		t.Fatalf("expected an honest zero-capacity explanation: %q", x.Explanation)
	}
	if len(x.CommitmentOutlooks) != 1 || x.CommitmentOutlooks[0].RiskState != RiskUnknown || x.CommitmentOutlooks[0].ForecastFinish != "" || x.CommitmentOutlooks[0].Explanation == "" {
		t.Fatalf("commitment outlook must preserve unknown timing: %#v", x.CommitmentOutlooks)
	}
}

func TestBuildComparesPromiseBoundaryWithBothScenarios(t *testing.T) {
	x, err := Build(Snapshot{StartDate: "2026-10-01", Timezone: "UTC", HorizonDays: 10, KnownWork: 700, Commitments: []CommitmentInput{{ID: "c-1", Result: "Send brief", PromisedBoundary: "2026-10-01"}, {ID: "c-2", Result: "Publish brief", PromisedBoundary: "2026-10-02"}}}, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(x.CommitmentOutlooks) != 2 {
		t.Fatalf("expected two commitment outlooks, got %d", len(x.CommitmentOutlooks))
	}
	if x.CommitmentOutlooks[0].RiskState != "at_risk" || x.CommitmentOutlooks[1].RiskState != RiskElevated {
		t.Fatalf("unexpected commitment risks: %#v", x.CommitmentOutlooks)
	}
}
