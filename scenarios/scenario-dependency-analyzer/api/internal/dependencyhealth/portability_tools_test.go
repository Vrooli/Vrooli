package dependencyhealth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeToolManifest(t *testing.T, repoRoot, name, body string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "internal", "tools", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tool.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The repository's own manifests are held to the rule by the control plane
// (internal/tools TestToolManifestsDeclareAcquisitionRouteForEveryPlatform).
// This test proves the provider reads the repository's internal/tools
// directory and surfaces the shared rule's verdict.
func TestValidateToolAcquisitionCoverageReadsRepositoryToolDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	writeToolManifest(t, repoRoot, "covered", `{"name": "covered", "defaultPackage": "covered"}`)
	if err := validateToolAcquisitionCoverage(repoRoot); err != nil {
		t.Fatalf("covered tool rejected: %v", err)
	}

	writeToolManifest(t, repoRoot, "gap", `{"name": "gap", "platform_status": {"linux": {"status": "unqualified"}, "macos": {"status": "unsupported"}, "windows": {"status": "unsupported"}}}`)
	err := validateToolAcquisitionCoverage(repoRoot)
	if err == nil || !strings.Contains(err.Error(), `tool "gap" claims linux`) {
		t.Fatalf("uncovered tool not reported: %v", err)
	}
}
