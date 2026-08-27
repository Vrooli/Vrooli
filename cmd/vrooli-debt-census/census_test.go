package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root, err := os.MkdirTemp("", "debt-census-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	if err := copyTree(filepath.Join("testdata", "fixture"), root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })
	return root
}

func copyTree(source, destination string) error {
	entries, err := os.ReadDir(source)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(source, entry.Name())
		destinationPath := filepath.Join(destination, entry.Name())
		if entry.IsDir() {
			if err := os.MkdirAll(destinationPath, 0o755); err != nil {
				return err
			}
			if err := copyTree(sourcePath, destinationPath); err != nil {
				return err
			}
			continue
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destinationPath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func TestMetricOSRenameOutsideConfig(t *testing.T) {
	assertMetric(t, "os_rename_outside_config", 1)
}

func TestMetricFprintfTabRows(t *testing.T) { assertMetric(t, "fprintf_tab_rows", 1) }

func TestMetricPrivateJSONWriters(t *testing.T) { assertMetric(t, "private_json_writers", 1) }

func TestMetricBindStanzas(t *testing.T) { assertMetric(t, "bind_stanzas", 1) }

func TestMetricHandWrittenDispatchers(t *testing.T) { assertMetric(t, "hand_written_dispatchers", 1) }

func TestMetricDuplicateRunnerInterfaces(t *testing.T) {
	assertMetric(t, "duplicate_runner_interfaces", 1)
}

func TestMetricTestLookPathFakes(t *testing.T) { assertMetric(t, "test_lookpath_fakes", 1) }

func TestMetricInlineShellHeredocs(t *testing.T) { assertMetric(t, "inline_shell_heredocs", 1) }

func TestMetricStampedHandlerTests(t *testing.T) { assertMetric(t, "stamped_handler_tests", 1) }

func TestMetricBarePathLiterals(t *testing.T) { assertMetric(t, "bare_vrooli_path_literals", 2) }

func TestMetricUnnamedDurationLiterals(t *testing.T) { assertMetric(t, "unnamed_duration_literals", 1) }

func TestMetricBareOctalLiterals(t *testing.T) { assertMetric(t, "bare_octal_file_modes", 1) }

func TestMetricPrivateBinderDialects(t *testing.T) { assertMetric(t, "private_binder_dialects", 2) }

func TestMetricPrivateBinderCallSitesExcludesComments(t *testing.T) {
	assertMetric(t, "private_binder_call_sites", 2)
}

func TestMetricPrivateJSONWritersByNameExcludesNearMiss(t *testing.T) {
	assertMetric(t, "private_json_writers_by_name", 1)
}

func TestMetricRenderFormatProloguesExcludesString(t *testing.T) {
	assertMetric(t, "render_format_prologues", 1)
}

func TestMetricHandlerDirsWithoutTests(t *testing.T) {
	assertMetric(t, "handler_dirs_without_tests", 1)
}

func TestMetricHandlerDirsWithoutConformanceSuite(t *testing.T) {
	assertMetric(t, "handler_dirs_without_conformance_suite", 1)
}

func TestMetricAppeasementNumberConstantsExcludesString(t *testing.T) {
	assertMetric(t, "appeasement_number_constants", 2)
}

func TestMetricAppeasementStringConstantsExcludesComment(t *testing.T) {
	assertMetric(t, "appeasement_string_constants", 1)
}

func TestMetricValueNamedDurationConstantsExcludesPurposeName(t *testing.T) {
	assertMetric(t, "value_named_duration_constants", 1)
}

func TestMetricTargetsWithoutBudget(t *testing.T) { assertMetric(t, "targets_without_budget", 3) }

func TestMetricPhaseProvidersWithoutControlPlane(t *testing.T) {
	assertMetric(t, "phase_providers_without_control_plane", 1)
}

func TestMetricFlagFlagSetCallSitesCountsFiles(t *testing.T) {
	assertMetric(t, "flag_flagset_call_sites", 1)
}

func TestCollectReportsBudgetFailure(t *testing.T) {
	budget := testBudget(1000)
	budget["os_rename_outside_config"] = budgetEntry{Budget: 0}
	doc, failures := collect(fixtureRoot(t), budgetDocument{SchemaVersion: 1, Metrics: budget, Ratchet: false})
	if len(doc.Metrics) != len(metrics()) || len(failures) != 1 {
		t.Fatalf("got %d metrics and failures %v", len(doc.Metrics), failures)
	}
}

func TestCollectCarriesStateForEveryMetric(t *testing.T) {
	budget := testBudget(100)
	doc, failures := collect(fixtureRoot(t), budgetDocument{SchemaVersion: 1, Ratchet: true, Metrics: budget})
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		Metrics []map[string]any `json:"metrics"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, metric := range decoded.Metrics {
		if _, ok := metric["state"]; !ok {
			t.Fatalf("metric lacks state: %v", metric)
		}
	}
}

func TestBudgetFixtureLooseningFails(t *testing.T) {
	budget := readBudgetFixture(t, "budget-loosened.json")
	failures := ratchetFailures(metrics()[0], budget.Metrics["os_rename_outside_config"], 1, budget.Ratchet)
	joined := strings.Join(failures, "\n")
	if !strings.Contains(joined, "ratchet_loosened_budget") || !strings.Contains(joined, "budget=2") || !strings.Contains(joined, "baseline=1") {
		t.Fatalf("expected named loosening failure, got %v", failures)
	}
}

func TestBudgetFixtureWorseningFails(t *testing.T) {
	budget := readBudgetFixture(t, "budget-worsened.json")
	failures := ratchetFailures(metrics()[0], budget.Metrics["os_rename_outside_config"], 2, budget.Ratchet)
	joined := strings.Join(failures, "\n")
	if !strings.Contains(joined, "ratchet_worsened_debt") || !strings.Contains(joined, "value=2") || !strings.Contains(joined, "baseline=1") {
		t.Fatalf("expected named worsening failure, got %v", failures)
	}
}

func TestBudgetFixtureMissingBaselineIsSchemaError(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "budget-missing-baseline.json"))
	if err != nil {
		t.Fatal(err)
	}
	var budget budgetDocument
	err = json.Unmarshal(data, &budget)
	if err == nil || !strings.Contains(err.Error(), "requires both budget and baseline") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestBudgetFixtureBelowBothPasses(t *testing.T) {
	budget := readBudgetFixture(t, "budget-below-both.json")
	if failures := ratchetFailures(metrics()[0], budget.Metrics["os_rename_outside_config"], 1, budget.Ratchet); len(failures) != 0 {
		t.Fatalf("expected below-budget measurement to pass, got %v", failures)
	}
}

func readBudgetFixture(t *testing.T, name string) budgetDocument {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var budget budgetDocument
	if err := json.Unmarshal(data, &budget); err != nil {
		t.Fatal(err)
	}
	return budget
}

func testBudget(value int) map[string]budgetEntry {
	entries := make(map[string]budgetEntry, len(metrics()))
	for _, metric := range metrics() {
		entries[metric.BudgetKey] = budgetEntry{Budget: value, Baseline: value}
	}
	return entries
}

func assertMetric(t *testing.T, name string, want int) {
	t.Helper()
	root := fixtureRoot(t)
	for _, metric := range metrics() {
		if metric.Name != name {
			continue
		}
		got, err := metric.Measure(root)
		if err != nil || got.State != metricOK || got.Value != want {
			t.Fatalf("got %#v, err %v; want %d", got, err, want)
		}
		return
	}
	t.Fatalf("metric %q is not registered", name)
}
