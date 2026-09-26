package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	corestorage "github.com/vrooli/api-core/storage"
)

func filesystemFixture(t *testing.T, source string, owner *corestorage.OwnerManifest) AnalyzerContext {
	t.Helper()
	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	owner.ManifestPath = filepath.Join(root, ".vrooli", "service.json")
	if err := os.MkdirAll(filepath.Dir(owner.ManifestPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(owner.ManifestPath, []byte(`{"service":{"name":"fixture"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	return AnalyzerContext{
		RepoRoot: root, Scenario: "fixture", ScenarioDir: root, APIDir: apiDir,
		Language: "go", Owner: owner,
	}
}

func filesystemFindingCodes(findings []Finding) map[string]bool {
	codes := make(map[string]bool, len(findings))
	for _, finding := range findings {
		codes[finding.Code] = true
	}
	return codes
}

func TestFilesystemContractFlagsDirectWritersAndPlatformPaths(t *testing.T) {
	ac := filesystemFixture(t, `package main
import "os"
func write() { _ = os.WriteFile("C:\\vrooli\\cache\\state.json", nil, 0600) }`, &corestorage.OwnerManifest{
		Kind: corestorage.OwnerScenario, ID: "fixture", StorageDeclared: true,
	})

	findings, err := (filesystemContract{}).Analyze(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	codes := filesystemFindingCodes(findings)
	if !codes["FILESYSTEM_DIRECT_WRITER"] || !codes["FILESYSTEM_NONPORTABLE_PATH"] {
		t.Fatalf("findings = %#v, want direct-writer and non-portable-path findings", findings)
	}
}

func TestFilesystemContractFlagsImplicitCreationMode(t *testing.T) {
	ac := filesystemFixture(t, `package main
import "os"
func write() { _, _ = os.Create("cache/state.json") }`, &corestorage.OwnerManifest{
		Kind: corestorage.OwnerScenario, ID: "fixture", StorageDeclared: true,
	})
	findings, err := (filesystemContract{}).Analyze(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	if !filesystemFindingCodes(findings)["FILESYSTEM_MODE_UNPROVEN"] {
		t.Fatalf("findings = %#v, want implicit-mode finding", findings)
	}
}

func TestFilesystemContractAcceptsReviewedWriterSeam(t *testing.T) {
	ac := filesystemFixture(t, `package main
import "github.com/vrooli/vrooli/internal/config"
func write(path string, data []byte) { _, _ = config.WriteOwnedFileAtomic(path, data, 0600) }`, &corestorage.OwnerManifest{
		Kind: corestorage.OwnerScenario, ID: "fixture", StorageDeclared: true,
	})

	findings, err := (filesystemContract{}).Analyze(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	for _, finding := range findings {
		if finding.Code == "FILESYSTEM_DIRECT_WRITER" || finding.Code == "FILESYSTEM_NONPORTABLE_PATH" {
			t.Fatalf("reviewed writer seam was flagged: %#v", findings)
		}
	}
}

func TestFilesystemContractRequiresRetentionAuthorityForRegenerableEntry(t *testing.T) {
	ac := filesystemFixture(t, `package main
func main() {}`, &corestorage.OwnerManifest{
		Kind: corestorage.OwnerScenario, ID: "fixture", StorageDeclared: true,
		StorageEntries: []corestorage.StorageEntry{{
			Name: "cache", Class: corestorage.ClassCache, Regenerable: true,
		}},
	})

	findings, err := (filesystemContract{}).Analyze(context.Background(), ac)
	if err != nil {
		t.Fatal(err)
	}
	if !filesystemFindingCodes(findings)["RETENTION_AUTHORITY_MISSING"] {
		t.Fatalf("findings = %#v, want retention-authority finding", findings)
	}
}
