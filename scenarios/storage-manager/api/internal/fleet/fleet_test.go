package fleet

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type fakeClassifier struct {
	entries map[string]ScenarioEntry
	fail    map[string]bool
}

func (f fakeClassifier) Classify(_ context.Context, scenario string) (ScenarioEntry, error) {
	if f.fail[scenario] {
		return ScenarioEntry{}, errors.New("boom")
	}
	return f.entries[scenario], nil
}

type fakeEnum struct{ list []string }

func (f fakeEnum) List(context.Context) ([]string, error) { return f.list, nil }

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestScanRollup(t *testing.T) {
	cls := fakeClassifier{
		entries: map[string]ScenarioEntry{
			"good":  {Scenario: "good", Engines: []string{"sqlite"}, StorageStage: "greenfield", IsolationReady: true, NamespaceAdopted: true, HasBackupTarget: true, FindingCount: 0},
			"bad":   {Scenario: "bad", Engines: []string{"postgres"}, StorageStage: "production", IsolationReady: false, IsolationReason: "seams unwired", HasBackupTarget: false, FindingCount: 3, ErrorCount: 1, AutofixableCount: 1},
			"cache": {Scenario: "cache", Engines: []string{"redis"}, StorageStage: "pilot", IsolationReady: true, HasBackupTarget: false, FindingCount: 0},
		},
		fail: map[string]bool{"broken": true},
	}
	svc := NewService(cls, fakeEnum{}, nil, fixedClock{t: time.Unix(1000, 0).UTC()})

	res, err := svc.Scan(context.Background(), []string{"good", "bad", "cache", "broken"})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.ScenarioCount != 3 {
		t.Fatalf("scenario_count: got %d want 3", res.ScenarioCount)
	}
	if len(res.Errors) != 1 || res.Errors[0].Scenario != "broken" {
		t.Fatalf("expected 1 error for broken, got %+v", res.Errors)
	}
	if res.IsolationUnreadyCount != 1 {
		t.Fatalf("isolation_unready: got %d want 1", res.IsolationUnreadyCount)
	}
	// "bad" is data-persisting (postgres) without a backup target → counted.
	// "cache" is redis-only (ephemeral) → NOT counted.
	if res.NoBackupCount != 1 {
		t.Fatalf("no_backup: got %d want 1 (postgres counted, redis not)", res.NoBackupCount)
	}
	if res.FindingCount != 3 {
		t.Fatalf("finding_count: got %d want 3", res.FindingCount)
	}
	if res.ScannedAt.IsZero() {
		t.Fatal("expected scanned_at stamped from clock")
	}
	// Distributions present.
	if len(res.EngineDistribution) != 3 || len(res.StageDistribution) != 3 {
		t.Fatalf("distributions: engines=%v stages=%v", res.EngineDistribution, res.StageDistribution)
	}

	unready := res.IsolationUnready()
	if len(unready) != 1 || unready[0].Scenario != "bad" {
		t.Fatalf("IsolationUnready: %+v", unready)
	}
	pg := res.ByEngine("postgres")
	if len(pg) != 1 || pg[0].Scenario != "bad" {
		t.Fatalf("ByEngine(postgres): %+v", pg)
	}
}

func TestScanUsesEnumeratorWhenNoTargets(t *testing.T) {
	cls := fakeClassifier{entries: map[string]ScenarioEntry{"a": {Scenario: "a"}, "b": {Scenario: "b"}}}
	svc := NewService(cls, fakeEnum{list: []string{"b", "a"}}, nil, nil)
	res, err := svc.Scan(context.Background(), nil)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if res.ScenarioCount != 2 {
		t.Fatalf("want 2 scenarios, got %d", res.ScenarioCount)
	}
	// Sorted.
	if res.Entries[0].Scenario != "a" || res.Entries[1].Scenario != "b" {
		t.Fatalf("expected sorted entries, got %+v", res.Entries)
	}
}

func TestScanUnwiredService(t *testing.T) {
	var svc *Service
	if _, err := svc.Scan(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error from nil service")
	}
}

func TestDataDirBudgetCheckerMeasuresScenarioAndRuntimeDirs(t *testing.T) {
	scenarioDir := t.TempDir()
	homeDir := t.TempDir()
	writeFile(t, filepath.Join(scenarioDir, ".vrooli", "service.json"), `{"storage_health":{"data_dir_budget_bytes":100}}`)
	writeFile(t, filepath.Join(scenarioDir, "data", "local.bin"), stringsOfLen(80))
	writeFile(t, filepath.Join(homeDir, ".vrooli", "data", "vrooli", "alpha", "runtime.bin"), stringsOfLen(70))

	got, err := (DataDirBudgetChecker{HomeDir: homeDir}).Check(context.Background(), "alpha", scenarioDir)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if got.Bytes != 150 {
		t.Fatalf("bytes = %d, want 150", got.Bytes)
	}
	if got.BudgetBytes != 100 {
		t.Fatalf("budget = %d, want override 100", got.BudgetBytes)
	}
	if !got.OverBudget || got.Severity != "warning" {
		t.Fatalf("over/severity = %v/%q, want warning", got.OverBudget, got.Severity)
	}
	if len(got.Paths) != 2 {
		t.Fatalf("paths = %v, want local and runtime dirs", got.Paths)
	}
}

func TestDataDirSeverityScalesWithOvershoot(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{101, "warning"},
		{200, "serious"},
		{400, "critical"},
	}
	for _, tc := range cases {
		if got := dataDirSeverity(tc.bytes, 100); got != tc.want {
			t.Fatalf("dataDirSeverity(%d, 100) = %q, want %q", tc.bytes, got, tc.want)
		}
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func stringsOfLen(n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = 'x'
	}
	return string(buf)
}
