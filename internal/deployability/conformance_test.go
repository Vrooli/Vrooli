package deployability

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverConformanceTargetsCoversAuthoredClaimsWithoutCountPinning(t *testing.T) {
	root := t.TempDir()
	writeConformanceFile(t, root, ".vrooli/capability-vocabulary.json", `{ "capabilities": [] }`)
	writeConformanceFile(t, root, "go.mod", "module fixture\n\ngo 1.23\n")
	writeConformanceFile(t, root, "internal/tools/honest/tool.json", `{ "platforms": ["linux"] }`)
	writeConformanceFile(t, root, "internal/safeguards/control/safeguard.json", `{ "platforms": ["linux", "windows"] }`)
	targets, err := DiscoverConformanceTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 3 {
		t.Fatalf("discovered %d targets, want one per authored claim; %+v", len(targets), targets)
	}
	for _, target := range targets {
		if target.CodeRoot != "." {
			t.Errorf("target rooted at %q, want repository module root", target.CodeRoot)
		}
	}
}

func TestCheckRepositoryNamesCompilerFailureByManifestAndOS(t *testing.T) {
	root := t.TempDir()
	writeConformanceFile(t, root, "go.mod", "module fixture\n\ngo 1.23\n")
	writeConformanceFile(t, root, "internal/safeguards/linux-only/safeguard.json", `{ "platforms": ["windows"] }`)
	writeConformanceFile(t, root, "internal/safeguards/linux-only/linux_only.go", "//go:build linux\n\npackage linuxonly\n\nvar undefinedOnWindows = 1\n")
	writeConformanceFile(t, root, "internal/safeguards/linux-only/platform.go", "package linuxonly\n\nvar _ = undefinedOnWindows\n")
	report, err := CheckRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("got %d findings, want one: %+v", len(report.Findings), report.Findings)
	}
	finding := report.Findings[0]
	if finding.ManifestPath != "internal/safeguards/linux-only/safeguard.json" || finding.OS != HostOSWindows {
		t.Fatalf("finding does not identify the overclaim: %+v", finding)
	}
	if !strings.Contains(finding.Message, "GOOS=windows") {
		t.Fatalf("finding does not include compiler target: %q", finding.Message)
	}
}

func writeConformanceFile(t *testing.T, root, name, body string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
