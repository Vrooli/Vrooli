package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

func TestStorageCorrespondenceRejectsRepoRelativeScenarioPath(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json")
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "demo", ManifestPath: manifest, StorageEntries: []corestorage.StorageEntry{{Name: "data", Class: corestorage.ClassData, Path: corestorage.PortablePath{Value: "data"}}}}
	findings, err := (&storageCorrespondence{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Platform: corestorage.PlatformLinux, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "STORAGE_PATH_DIVERGENT" || findings[0].Severity != SeverityError {
		t.Fatalf("findings = %#v, want one hard divergence", findings)
	}
}

func TestStorageCoverageRejectsBytesOutsideDeclaredClassSubpath(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "runtime")
	t.Setenv("VROOLI_DATA_ROOT", dataRoot)
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "demo", ManifestPath: filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), StorageEntries: []corestorage.StorageEntry{{Name: "records", Class: corestorage.ClassData, Subpath: "records"}}}
	declared := filepath.Join(dataRoot, "vrooli", "demo", "records", "ok")
	outside := filepath.Join(dataRoot, "vrooli", "demo", "unlisted", "payload")
	for path, value := range map[string]string{declared: "ok", outside: "unlisted"} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	findings, err := (&storageCoverage{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Platform: corestorage.PlatformLinux, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "STORAGE_PATH_UNCOVERED" || findings[0].Severity != SeverityError {
		t.Fatalf("findings = %#v, want one hard coverage finding", findings)
	}
}

func TestStorageCoverageAuditsEmptyOwnerAcrossSiblingClassRoots(t *testing.T) {
	root := t.TempDir()
	dataRoot := filepath.Join(root, "runtime")
	t.Setenv("VROOLI_DATA_ROOT", dataRoot)
	owner := &corestorage.OwnerManifest{Kind: corestorage.OwnerScenario, ID: "empty-owner", ManifestPath: filepath.Join(root, "scenarios", "empty-owner", ".vrooli", "service.json")}
	path := filepath.Join(dataRoot, "vrooli", "empty-owner", "data", "orphan.bin")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("unlisted"), 0o644); err != nil {
		t.Fatal(err)
	}
	findings, err := (&storageCoverage{}).Analyze(context.Background(), AnalyzerContext{RepoRoot: root, Platform: corestorage.PlatformLinux, Owner: owner})
	if err != nil {
		t.Fatal(err)
	}
	if len(findings) != 1 || findings[0].Code != "STORAGE_PATH_UNCOVERED" {
		t.Fatalf("findings = %#v, want an uncovered finding for an empty owner", findings)
	}
}
