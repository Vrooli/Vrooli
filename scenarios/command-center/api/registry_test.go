package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAuthoredMetricsHaveExplicitTypedBindings(t *testing.T) {
	reg, err := LoadRegistry("../config/outcome-registry.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(reg.Metrics) != 37 {
		t.Fatalf("metrics = %d, want current authored set", len(reg.Metrics))
	}
	for _, metric := range reg.Metrics {
		binding := metric.Source
		if binding.IntegrationID == "" || binding.FeatureID == "" || binding.ContractVersion == "" || binding.Selector == "" || binding.ExpectedUnit == "" || binding.SourceTimePolicy != "producer_required" {
			t.Errorf("metric %q has incomplete typed binding: %+v", metric.ID, binding)
		}
		if binding.IntegrationID == "none" && metric.Coverage != CoverageMissing {
			t.Errorf("no-source metric %q has coverage %s", metric.ID, metric.Coverage)
		}
		if strings.TrimSpace(metric.Source.FeatureID) != strings.TrimSpace(metric.Source.Selector) {
			t.Errorf("metric %q feature/selector mismatch", metric.ID)
		}
	}
}

func TestLoadRegistry_ValidFile(t *testing.T) {
	reg, err := LoadRegistry(filepath.Join("..", "config", "outcome-registry.json"))
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	if reg.SchemaVersion == "" {
		t.Fatalf("expected non-empty schema version")
	}
	for _, id := range []string{"mission-control", "hive", "forge", "ledger", "broadcast", "panorama"} {
		entries := reg.Dashboard(id)
		if len(entries) == 0 {
			t.Errorf("dashboard %q has no metrics", id)
		}
	}
}

func TestOutcomeRegistrySelectorsAreDeclared(t *testing.T) {
	reg, err := LoadRegistry(filepath.Join("..", "config", "outcome-registry.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, metric := range reg.Metrics {
		if metric.Coverage == CoverageMissing {
			continue
		}
		if _, ok := selectors[metric.Source.Selector]; !ok {
			t.Errorf("metric %q names undeclared selector %q", metric.ID, metric.Source.Selector)
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

func TestLoadRegistry_UnknownOriginRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	payload := `{"schemaVersion":"2.0.0","origins":{"local":{"mode":"discovery","environment":"local","display":"Local"}},"metrics":[{"id":"x","label":"x","unit":"count","source":{"integrationId":"x","featureId":"x","selector":"x","expectedUnit":"count","contractVersion":"legacy.v1","sourceTimePolicy":"producer_required","ttlSeconds":60,"origin":"missing"},"coverage":"NOW"}],"rooms":[],"tombstones":[]}`
	if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRegistry(path); err == nil || !strings.Contains(err.Error(), "unknown origin") {
		t.Fatalf("expected unknown origin error, got %v", err)
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
	if reg.SchemaVersion != "2.0.0" || len(reg.Metrics) != 37 {
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
