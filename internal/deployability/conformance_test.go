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
	writeRepoContract(t, root)
	writeConformanceFile(t, root, ".vrooli/capability-vocabulary.json", `{ "capabilities": [] }`)
	writeConformanceFile(t, root, "go.mod", "module fixture\n\ngo 1.23\n")
	writeConformanceFile(t, root, "internal/tools/honest/tool.json", `{ "platforms": ["linux"] }`)
	writeConformanceFile(t, root, "internal/safeguards/control/safeguard.json", `{ "platforms": ["linux", "windows"] }`)
	targets, err := DiscoverConformanceTargets(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("discovered %d targets, want one per non-Linux platform for the module; %+v", len(targets), targets)
	}
	for _, target := range targets {
		if target.CodeRoot != "." {
			t.Errorf("target rooted at %q, want repository module root", target.CodeRoot)
		}
	}
}

func TestCheckRepositoryNamesCompilerFailureByManifestAndOS(t *testing.T) {
	root := t.TempDir()
	writeRepoContract(t, root)
	writeConformanceFile(t, root, "go.mod", "module fixture\n\ngo 1.23\n")
	writeConformanceFile(t, root, "internal/safeguards/linux-only/safeguard.json", `{ "platforms": ["windows"] }`)
	writeConformanceFile(t, root, "internal/safeguards/linux-only/linux_only.go", "//go:build linux\n\npackage linuxonly\n\nvar undefinedOnWindows = 1\n")
	writeConformanceFile(t, root, "internal/safeguards/linux-only/platform.go", "package linuxonly\n\nvar _ = undefinedOnWindows\n")
	report, err := CheckRepository(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Findings) != 2 {
		t.Fatalf("got %d findings, want both non-Linux compiler failures: %+v", len(report.Findings), report.Findings)
	}
	for _, finding := range report.Findings {
		if finding.ManifestPath != "go.mod" {
			t.Fatalf("finding does not identify the module: %+v", finding)
		}
		if finding.OS != HostOSWindows && finding.OS != HostOSMacOS {
			t.Fatalf("unexpected compiler target: %+v", finding)
		}
		if !strings.Contains(finding.Message, "GOOS=") {
			t.Fatalf("finding does not include compiler target: %q", finding.Message)
		}
	}
}

func writeRepoContract(t *testing.T, root string) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, readErr := os.ReadFile(filepath.Join(wd, ".vrooli", "repo-contract.json"))
		if readErr == nil {
			writeConformanceFile(t, root, ".vrooli/repo-contract.json", string(data))
			return
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			t.Fatalf("locate repository contract: %v", readErr)
		}
		wd = parent
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
