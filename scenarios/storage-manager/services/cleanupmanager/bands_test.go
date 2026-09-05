package cleanupmanager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyUsesOrderedInclusiveThresholds(t *testing.T) {
	for _, tc := range []struct {
		used float64
		want Band
	}{
		{used: 69.9, want: BandNormal},
		{used: 70, want: BandWarning},
		{used: 80, want: BandHigh},
		{used: 90, want: BandCritical},
	} {
		if got := Classify(tc.used, 1000, Thresholds{Warning: 70, High: 80, Critical: 90}); got != tc.want {
			t.Fatalf("Classify(%v) = %q, want %q", tc.used, got, tc.want)
		}
	}
}

func TestClassifyFailsClosedForInvalidThresholds(t *testing.T) {
	if got := Classify(99, 1000, Thresholds{Warning: 80, High: 70, Critical: 90}); got != BandNormal {
		t.Fatalf("invalid thresholds classified as %q, want normal", got)
	}
}

func TestClassifyFloorOverridesPercentage(t *testing.T) {
	if got := Classify(50, 99, Thresholds{Warning: 70, High: 80, Critical: 90, FloorBytes: 100}); got != BandCritical {
		t.Fatalf("floor classification = %q, want critical", got)
	}
}

func TestLoadThresholdsReadsRepositoryContract(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo-contract.json")
	if err := os.WriteFile(path, []byte(`{"storage":{"recovery":{"floor_bytes":10,"fast_fill_percent":5,"bands":{"warning":80,"high":90,"critical":95}}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadThresholds(path)
	if err != nil {
		t.Fatalf("LoadThresholds: %v", err)
	}
	if got.Warning != 80 || got.High != 90 || got.Critical != 95 || got.FloorBytes != 10 {
		t.Fatalf("thresholds = %#v", got)
	}
}
