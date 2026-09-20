package forecasts

import (
	"testing"
	"time"
)

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
