package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestStorageEntryConformanceFlagsOwnedPhantom(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "fixture")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"service":{"name":"fixture"},"storage":{"entries":{"metrics":{"rung":"owned","path":"/never-created/metrics","kind":"dir","class":"state","regenerable":true,"budget":{"max_bytes":"2GiB"}}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "fixture", ManifestPath: manifest, StorageEntries: []corestorage.StorageEntry{{Name: "metrics", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "/never-created/metrics"}, Kind: "dir", Class: corestorage.ClassState, Regenerable: true, Budget: &corestorage.BudgetDeclaration{MaxBytes: "2GiB"}}}}
	got, err := (storageEntryConformance{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Scenario: "fixture", ScenarioDir: scenarioDir, APIDir: filepath.Join(scenarioDir, "api"), Language: "go", Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range got {
		if finding.Code == "STORAGE_ENTRY_NO_WRITER" {
			return
		}
	}
	t.Fatalf("findings = %#v, want STORAGE_ENTRY_NO_WRITER", got)
}

func TestStorageEntryConformanceFlagsNonBindingCeiling(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "fixture")
	manifest := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(scenarioDir, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "data", "records.bin"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"service":{"name":"fixture"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "fixture", ManifestPath: manifest, StorageEntries: []corestorage.StorageEntry{{Name: "records", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "data"}, Kind: "dir", Class: corestorage.ClassData, Budget: &corestorage.BudgetDeclaration{MaxBytes: "110B", Rationale: "measured and governed on 2026-08-05"}}}}
	got, err := (storageEntryConformance{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Scenario: "fixture", Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range got {
		if finding.Code == "STORAGE_CEILING_NON_BINDING" {
			if finding.Severity != SeverityError {
				t.Fatalf("finding severity = %v, want SeverityError", finding.Severity)
			}
			return
		}
	}
	t.Fatalf("findings = %#v, want STORAGE_CEILING_NON_BINDING", got)
}

func TestStorageEntryConformanceAcceptsFrameworkOwnedEntry(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "fixture")
	if err := os.MkdirAll(filepath.Join(scenarioDir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"service":{"name":"fixture"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "fixture", ManifestPath: manifest, StorageEntries: []corestorage.StorageEntry{{Name: "data", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "data"}, Kind: "dir", Class: corestorage.ClassData}}}
	got, err := (storageEntryConformance{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Scenario: "fixture", ScenarioDir: scenarioDir, APIDir: filepath.Join(scenarioDir, "api"), Language: "go", Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range got {
		if finding.Code == "STORAGE_ENTRY_NO_WRITER" {
			t.Fatalf("framework-owned entry was treated as unwritten: %#v", finding)
		}
	}
}

func TestStorageEntryConformanceAcceptsDirectWriterAndSQLiteSidecars(t *testing.T) {
	root := t.TempDir()
	scenarioDir := filepath.Join(root, "scenarios", "fixture")
	apiDir := filepath.Join(scenarioDir, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(`package main
import ("os"; "path/filepath")
func main() { _ = os.MkdirAll(filepath.Dir("data/fixture.db"), 0755); _, _ = os.Create("fixture.db") }`), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(scenarioDir, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"service":{"name":"fixture"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "fixture", ManifestPath: manifest, StorageEntries: []corestorage.StorageEntry{
		{Name: "db", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "data/fixture.db"}, Kind: "file", Class: corestorage.ClassData, Format: "sqlite"},
		{Name: "db_wal", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "data/fixture.db-wal"}, Kind: "file", Class: corestorage.ClassState},
		{Name: "db_shm", Rung: corestorage.RungOwned, Path: corestorage.PortablePath{Value: "data/fixture.db-shm"}, Kind: "file", Class: corestorage.ClassState},
	}}
	got, err := (storageEntryConformance{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Scenario: "fixture", ScenarioDir: scenarioDir, APIDir: apiDir, Language: "go", Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range got {
		if finding.Code == "STORAGE_ENTRY_NO_WRITER" || finding.Code == "STORAGE_SQLITE_SIDECAR_UNDECLARED" {
			t.Fatalf("unexpected conformance finding: %#v", finding)
		}
	}
}

func TestCustomTestRunsRequiresLinkedOwnerProviderForGovernedStatus(t *testing.T) {
	manifest := []byte(`{"storage":{"entries":{"run_artifacts":{"kind":"dir","class":"test_runs","regenerable":true,"reclaim":{"pruner":"custom"},"budget":{"max_age":"30d","max_bytes":"20GiB"}}},"cleanup_providers":[{"id":"test-genie-run-retention","safety_tier":"safe_with_owner","default_mode":"disabled","default_approval":"owner"}]}}`)
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "test-genie", StorageEntries: []corestorage.StorageEntry{{
		Name: "run_artifacts", Kind: "dir", Class: corestorage.ClassTestRuns, Regenerable: true,
		Reclaim: &corestorage.ReclaimDeclaration{Pruner: "custom"},
		Budget:  &corestorage.BudgetDeclaration{MaxAge: "30d", MaxBytes: "20GiB"},
	}}}
	ctx := AnalyzerContext{Owner: owner, RepoRoot: t.TempDir()}

	findings := retentionConflicts(ctx, manifest)
	var found bool
	for _, finding := range findings {
		if finding.Code == "STORAGE_CUSTOM_RECLAIM_UNLINKED" && strings.Contains(finding.Message, "run_artifacts") {
			found = true
		}
	}
	if !found {
		t.Fatalf("unlinked custom test_runs findings = %#v, want explicit owner-provider link finding", findings)
	}
	accountability := accountabilityFindings(ctx, findings)
	var blocked bool
	for _, finding := range accountability {
		if finding.Code == "STORAGE_ACCOUNTABILITY_UNRECONCILED" {
			blocked = true
		}
	}
	if !blocked {
		t.Fatalf("unlinked custom owner was reported governed: accountability findings=%#v", accountability)
	}

	linked := []byte(strings.Replace(string(manifest), `"default_approval":"owner"`, `"default_approval":"owner","storage_entries":["run_artifacts"]`, 1))
	for _, finding := range retentionConflicts(ctx, linked) {
		if finding.Code == "STORAGE_CUSTOM_RECLAIM_UNLINKED" {
			t.Fatalf("linked owner provider still reported unlinked: %#v", finding)
		}
	}
}
