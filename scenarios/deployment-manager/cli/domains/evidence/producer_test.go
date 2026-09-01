package evidence

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadReportFactUsesReportTimestampAndFreshness(t *testing.T) {
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	path := filepath.Join(t.TempDir(), "deployment-report.json")
	if err := os.WriteFile(path, []byte(`{"generated_at":"2026-09-01T11:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	fact, err := readReportFact(path, now, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if fact.GetName() != "deployment_report_fresh" || fact.GetValue() != 1 {
		t.Fatalf("fresh fact = %s/%v", fact.GetName(), fact.GetValue())
	}
	if got := fact.GetObservedAt().AsTime(); !got.Equal(time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC)) {
		t.Fatalf("observed_at = %s", got)
	}
	if fact.GetDimension() != "producer:deployment-manager" || fact.GetStaleAfterDays() != 1 {
		t.Fatalf("provenance = %s/%d", fact.GetDimension(), fact.GetStaleAfterDays())
	}

	if err := os.WriteFile(path, []byte(`{"generated_at":"2026-08-30T11:00:00Z"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fact, err = readReportFact(path, now, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if fact.GetValue() != 0 {
		t.Fatalf("stale fact value = %v, want 0", fact.GetValue())
	}
}

func TestReadReportFactRejectsMalformedTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deployment-report.json")
	if err := os.WriteFile(path, []byte(`{"generated_at":"not-a-time"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readReportFact(path, time.Now(), time.Hour); err == nil {
		t.Fatal("malformed report timestamp was accepted")
	}
}
