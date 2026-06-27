package installgateway

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mkSurface(t *testing.T, repoRoot, scenario, surface string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(repoRoot, "scenarios", scenario, surface)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveBuildsArgvPerEcosystem(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{"pnpm-lock.yaml": "", "package.json": "{}"})
	mkSurface(t, repoRoot, "demo", "api", map[string]string{"go.mod": "module demo\n"})
	mkSurface(t, repoRoot, "demo", "cli", map[string]string{"requirements.txt": ""})

	cases := []struct {
		surface, ecosystem, pkg, version string
		wantManager                      string
		wantArgv                         string
	}{
		{"ui", "npm", "react-hook-form", "7.0.0", "pnpm", "pnpm add react-hook-form@7.0.0"},
		{"ui", "npm", "zod", "", "pnpm", "pnpm add zod"},
		{"api", "go", "github.com/foo/bar", "v1.2.3", "go", "go get github.com/foo/bar@v1.2.3"},
		{"cli", "pip", "requests", "2.31.0", "pip", "pip install requests==2.31.0"},
	}
	for _, tc := range cases {
		r, err := Resolve(repoRoot, "demo", tc.surface, tc.ecosystem, tc.pkg, tc.version)
		if err != nil {
			t.Fatalf("%s/%s: %v", tc.ecosystem, tc.pkg, err)
		}
		if r.PackageManager != tc.wantManager {
			t.Errorf("%s/%s manager = %q, want %q", tc.ecosystem, tc.pkg, r.PackageManager, tc.wantManager)
		}
		if r.Command() != tc.wantArgv {
			t.Errorf("%s/%s argv = %q, want %q", tc.ecosystem, tc.pkg, r.Command(), tc.wantArgv)
		}
	}
}

func TestResolveAddsPnpmWorkspaceRootForSurfaceWorkspace(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{
		"package.json":        "{}",
		"pnpm-lock.yaml":      "",
		"pnpm-workspace.yaml": "packages:\n  - .\n",
	})

	r, err := Resolve(repoRoot, "demo", "ui", "npm", "helmet", "^6.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.Command() != "pnpm add --workspace-root helmet@^6.1.0" {
		t.Fatalf("command = %q", r.Command())
	}
}

func TestExecInstallerRunsInSurfaceRoot(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	surfaceRoot := filepath.Join(dir, "surface")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(surfaceRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	cwdFile := filepath.Join(dir, "cwd.txt")
	fakeTool := filepath.Join(binDir, "fake-tool")
	if err := os.WriteFile(fakeTool, []byte("#!/bin/sh\npwd > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	_, err := (ExecInstaller{}).Install(context.Background(), Resolution{
		SurfaceRoot: surfaceRoot,
		Argv:        []string{"fake-tool", cwdFile},
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(cwdFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(got)) != surfaceRoot {
		t.Fatalf("installer cwd = %q, want %q", strings.TrimSpace(string(got)), surfaceRoot)
	}
}

func TestResolveRejectsBadSurfaceAndEcosystem(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{"pnpm-lock.yaml": ""})

	if _, err := Resolve(repoRoot, "demo", "worker", "npm", "x", ""); err == nil {
		t.Fatal("expected error for non-ui/api/cli surface")
	}
	if _, err := Resolve(repoRoot, "demo", "api", "npm", "x", ""); err == nil || !strings.Contains(err.Error(), "surface directory not found") {
		t.Fatalf("expected missing-surface error, got %v", err)
	}
	if _, err := Resolve(repoRoot, "demo", "ui", "cargo", "x", ""); err == nil {
		t.Fatal("expected error for unsupported ecosystem")
	}
}
