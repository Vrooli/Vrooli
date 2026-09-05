package offerfacts

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadFactsUsesLatestRecordPerScenario(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	body := `[{"scenario_name":"web-console","updated_at":"2026-09-01T10:00:00Z"},{"scenario_name":"web-console","updated_at":"2026-09-01T12:00:00Z"},{"scenario_name":"audio-tools","updated_at":"2026-09-01T11:00:00Z"}]`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	facts, err := ReadFacts(path, time.Now(), 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) != 3 {
		t.Fatalf("facts = %d, want 3 including the web-console trigger alias", len(facts))
	}
	for _, fact := range facts {
		if fact.GetDimension() != "producer:scenario-to-desktop" || fact.GetValue() != 1 || fact.GetStaleAfterDays() != 30 {
			t.Fatalf("fact provenance = %#v", fact)
		}
		if fact.GetName() == "shipped_on_ramp.desktop.web-console" && !fact.GetObservedAt().AsTime().Equal(time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)) {
			t.Fatalf("web-console timestamp = %s", fact.GetObservedAt().AsTime())
		}
	}
}

func TestReadFactsRejectsInvalidRecordTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "records.json")
	if err := os.WriteFile(path, []byte(`[{"scenario_name":"web-console","updated_at":"invalid"}]`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadFacts(path, time.Now(), 30); err == nil {
		t.Fatal("invalid timestamp was accepted")
	}
}
