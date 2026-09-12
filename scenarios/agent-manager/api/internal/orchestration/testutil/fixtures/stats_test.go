package fixtures

import "testing"

func TestNewStatsSnapshotDefaults(t *testing.T) {
	snapshot := NewStatsSnapshot()

	if snapshot.StatusCounts == nil {
		t.Fatal("expected status counts")
	}
	if snapshot.StatusCounts.Total != 27 {
		t.Fatalf("total runs = %d, want 27", snapshot.StatusCounts.Total)
	}
	if snapshot.SuccessRate != 0.85 {
		t.Fatalf("success rate = %f, want 0.85", snapshot.SuccessRate)
	}
	if len(snapshot.RunnerBreakdown) != 2 {
		t.Fatalf("runner breakdown len = %d, want 2", len(snapshot.RunnerBreakdown))
	}
	if len(snapshot.TimeSeries) != 2 {
		t.Fatalf("time series len = %d, want 2", len(snapshot.TimeSeries))
	}
}
