package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
)

func TestResolveWorkspaceCoversMissingRunnersAndFallbackLanguages(t *testing.T) {
	missing := t.TempDir()
	if resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "typescript", Surface: discovery.Surface{RootPath: missing}}); resolution.Status != "degraded" || len(diagnostics) == 0 {
		t.Fatal("missing package manifest was not degraded")
	}
	for _, tc := range []struct {
		language string
		file     string
	}{{"bash", ""}, {"powershell", ""}, {"rust", "Cargo.toml"}, {"go", "go.mod"}} {
		root := t.TempDir()
		if tc.file != "" {
			if err := os.WriteFile(filepath.Join(root, tc.file), []byte(""), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: tc.language, Surface: discovery.Surface{RootPath: root}, ResolveExecutable: func([]string) (string, bool) { return "", false }})
		if resolution.Status == "ready" || len(diagnostics) == 0 {
			t.Fatalf("%s fallback = %+v, %v", tc.language, resolution, diagnostics)
		}
	}
	if resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "cobol", Surface: discovery.Surface{RootPath: missing}}); resolution.Status != "unsupported" || len(diagnostics) != 1 {
		t.Fatalf("unsupported = %+v, %v", resolution, diagnostics)
	}
}

func TestResolveWorkspaceCoversNodeProjectionBranchesAndTestFileDiscovery(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run"},"devDependencies":{"vitest":"1"},"packageManager":"npm@10"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "vite.config.ts"), []byte("coverage: true"), 0o600); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "typescript", Surface: discovery.Surface{RootPath: root, PackageManager: "pnpm"}, ResolveExecutable: func([]string) (string, bool) { return "/bin/pnpm", true }})
	if resolution.Status != "ready" || len(diagnostics) == 0 {
		t.Fatalf("node projection = %+v, %v", resolution, diagnostics)
	}
	powershell := t.TempDir()
	os.MkdirAll(filepath.Join(powershell, "node_modules"), 0o755)
	os.WriteFile(filepath.Join(powershell, "node_modules", "ignored.Tests.ps1"), []byte(""), 0o600)
	os.WriteFile(filepath.Join(powershell, "real.Tests.ps1"), []byte(""), 0o600)
	if got := firstTestFile(powershell, ".Tests.ps1"); got == "" || !filepath.IsAbs(got) {
		t.Fatalf("first test file = %q", got)
	}
	if !hasFilesWithExt(powershell, ".Tests.ps1") || ignoredDir("vendor") == false || ignoredDir("src") {
		t.Fatal("test file helpers misclassified paths")
	}
}
