package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRegistry_ValidFile(t *testing.T) {
	reg, err := LoadRegistry(filepath.Join("..", "config", "gap-registry.json"))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if reg.Version == "" {
		t.Fatalf("expected non-empty version")
	}
	for _, id := range []string{"mission-control", "hive", "forge", "ledger", "broadcast", "panorama"} {
		entries := reg.Dashboard(id)
		if len(entries) == 0 {
			t.Errorf("dashboard %q has no metrics", id)
		}
	}
}

func TestLoadRegistry_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadRegistry(path); err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

func TestLoadRegistry_UnknownField_Rejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	payload := `{
		"version": "1.0.0",
		"dashboards": {"m": [{"id":"x","label":"x","dataSource":"live","upstreamSource":"swarm","bogus":"nope"}]}
	}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadRegistry(path); err == nil {
		t.Fatal("expected decode error for unknown field, got nil")
	}
}

func TestLoadRegistry_MissingVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	payload := `{"dashboards": {"m": [{"id":"x","label":"x","dataSource":"live","upstreamSource":"swarm"}]}}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := LoadRegistry(path); err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestGapsByDashboard_FiltersLive(t *testing.T) {
	reg := &Registry{
		Version: "1.0.0",
		Dashboards: map[string][]MetricEntry{
			"d": {
				{ID: "a", DataSource: StatusLive},
				{ID: "b", DataSource: StatusGap},
				{ID: "c", DataSource: StatusPartial},
			},
			"empty": {
				{ID: "x", DataSource: StatusLive},
			},
		},
	}
	gaps := reg.GapsByDashboard()
	if len(gaps) != 1 {
		t.Fatalf("expected 1 dashboard with gaps, got %d", len(gaps))
	}
	if len(gaps["d"]) != 2 {
		t.Fatalf("expected 2 gap entries, got %d", len(gaps["d"]))
	}
	if _, ok := gaps["empty"]; ok {
		t.Error("dashboard with all live entries should not appear")
	}
}

func TestOutcomeRegistryCarriesIndependentReadingAxesAndSamples(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if reg.SchemaVersion != "2.0.0" || len(reg.Metrics) != 38 {
		t.Fatalf("schema=%s metrics=%d", reg.SchemaVersion, len(reg.Metrics))
	}
	for _, m := range reg.Metrics {
		if !validCoverage(m.Coverage) || !validEmpirical(m.Empirical) {
			t.Fatalf("invalid axes for %s", m.ID)
		}
		if m.Coverage != CoverageNow && (m.Sample == nil || m.Sample.Basis == "") {
			t.Fatalf("%s has no authored sample basis", m.ID)
		}
	}
}
