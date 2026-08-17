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

func mkResourceSurface(t *testing.T, repoRoot, resource, surface string, files map[string]string) string {
	t.Helper()
	dir := filepath.Join(repoRoot, "resources", resource, surface)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestResolveSupportsResourceSurface(t *testing.T) {
	repoRoot := t.TempDir()
	resourceRoot := mkResourceSurface(t, repoRoot, "doc-parse", "cli", map[string]string{
		"go.mod": "module resource-doc-parse/cli\n",
	})

	r, err := Resolve(repoRoot, "doc-parse", "cli", "go", "github.com/tetratelabs/wazero", "v1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.SurfaceRoot != resourceRoot {
		t.Fatalf("resource surface root = %q, want %q", r.SurfaceRoot, resourceRoot)
	}
	if r.ManifestPath != filepath.Join(resourceRoot, "go.mod") {
		t.Fatalf("resource manifest = %q, want %q", r.ManifestPath, filepath.Join(resourceRoot, "go.mod"))
	}
	if got, want := r.Command(), "go get github.com/tetratelabs/wazero@v1.9.0"; got != want {
		t.Fatalf("resource install command = %q, want %q", got, want)
	}
}

func TestResolvePrefersScenarioSurfaceWhenNamesOverlap(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "doc-parse", "cli", map[string]string{"go.mod": "module scenario-doc-parse/cli\n"})
	resourceRoot := mkResourceSurface(t, repoRoot, "doc-parse", "cli", map[string]string{"go.mod": "module resource-doc-parse/cli\n"})

	r, err := Resolve(repoRoot, "doc-parse", "cli", "go", "github.com/tetratelabs/wazero", "v1.9.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.SurfaceRoot == resourceRoot {
		t.Fatal("scenario surface was shadowed by resource surface")
	}
}

func TestResolveBuildsArgvPerEcosystem(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{"pnpm-lock.yaml": "", "package.json": "{}"})
	mkSurface(t, repoRoot, "demo", "api", map[string]string{"go.mod": "module demo\n"})
	mkSurface(t, repoRoot, "demo", "cli", map[string]string{"requirements.txt": ""})
	mkSurface(t, repoRoot, "demo", "tools/mermaid-lint", map[string]string{"pnpm-lock.yaml": "", "package.json": "{}"})
	mkSurface(t, repoRoot, "demo", "platforms/electron", map[string]string{"package-lock.json": "", "package.json": "{}"})

	cases := []struct {
		surface, ecosystem, pkg, version string
		wantManager                      string
		wantArgv                         string
	}{
		{"ui", "npm", "react-hook-form", "7.0.0", "pnpm", "pnpm add --ignore-scripts react-hook-form@7.0.0"},
		{"ui", "npm", "zod", "", "pnpm", "pnpm add --ignore-scripts zod"},
		{"api", "go", "github.com/foo/bar", "v1.2.3", "go", "go get github.com/foo/bar@v1.2.3"},
		{"cli", "pip", "requests", "2.31.0", "pip", "pip install requests==2.31.0"},
		{"tools/mermaid-lint", "npm", "mermaid", "11.13.0", "pnpm", "pnpm add --ignore-scripts mermaid@11.13.0"},
		{"platforms/electron", "npm", "brace-expansion", "^5.0.8", "npm", "npm install --ignore-scripts brace-expansion@^5.0.8"},
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

func TestResolveSupportsSharedPackageToolsSurface(t *testing.T) {
	repoRoot := t.TempDir()
	packageRoot := filepath.Join(repoRoot, "packages", "api-base")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"name":"@vrooli/api-base"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "pnpm-lock.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := Resolve(repoRoot, "api-base", "tools/api-base", "npm", "react", "^18.3.1")
	if err != nil {
		t.Fatal(err)
	}
	if r.SurfaceRoot != packageRoot || r.ManifestPath != filepath.Join(packageRoot, "package.json") {
		t.Fatalf("package resolution = %+v, want root %s", r, packageRoot)
	}
}

func TestResolveSharedPackageWorkspaceDoesNotDisableWorkspaceMode(t *testing.T) {
	repoRoot := t.TempDir()
	packageRoot := filepath.Join(repoRoot, "packages", "audio-capture-browser")
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"name":"@vrooli/audio-capture-browser"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "pnpm-lock.yaml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "pnpm-workspace.yaml"), []byte("packages:\n  - .\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Resolve(repoRoot, "audio-tools", "tools/audio-capture-browser", "npm", "@types/react", "^18.2.47")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := r.Command(), "pnpm add --ignore-scripts --workspace-root @types/react@^18.2.47"; got != want {
		t.Fatalf("command = %q, want %q", got, want)
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
	if r.Command() != "pnpm add --ignore-scripts --workspace-root helmet@^6.1.0" {
		t.Fatalf("command = %q", r.Command())
	}
}

func TestResolveNpmOverrideAndPersistOverride(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{
		"package.json":        `{"name":"demo"}`,
		"pnpm-lock.yaml":      "",
		"pnpm-workspace.yaml": "packages:\n  - .\n",
	})

	r, err := ResolveNpmOverride(repoRoot, "demo", "ui")
	if err != nil {
		t.Fatal(err)
	}
	if r.Command() != "pnpm install --ignore-scripts --workspace-root" {
		t.Fatalf("command = %q", r.Command())
	}
	if err := SetNpmOverride(r.ManifestPath, "minimatch", "^10.2.6"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(r.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"minimatch": "^10.2.6"`) {
		t.Fatalf("override was not recorded: %s", data)
	}
}

func TestResolveNpmOverrideRejectsNonPnpmSurface(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "platforms/electron", map[string]string{"package.json": "{}", "package-lock.json": ""})
	if _, err := ResolveNpmOverride(repoRoot, "demo", "platforms/electron"); err == nil {
		t.Fatal("expected npm override to reject an npm-lock surface")
	}
}

func TestResolvePreservesExistingJSDevDependencyClassification(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{
		"package.json":   `{"devDependencies":{"vite":"^5.4.0"}}`,
		"pnpm-lock.yaml": "",
	})

	r, err := Resolve(repoRoot, "demo", "ui", "npm", "vite", "6.4.3")
	if err != nil {
		t.Fatal(err)
	}
	if r.Command() != "pnpm add --ignore-scripts -D vite@6.4.3" {
		t.Fatalf("command = %q, want pnpm dev-dependency upgrade", r.Command())
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
		t.Fatal("expected error for non-installable surface")
	}
	if _, err := Resolve(repoRoot, "demo", "tools", "npm", "x", ""); err == nil {
		t.Fatal("expected tools root to be rejected; a tools package is required")
	}
	if _, err := Resolve(repoRoot, "demo", "platforms", "npm", "x", ""); err == nil {
		t.Fatal("expected platforms root to be rejected; a platform package is required")
	}
	if _, err := Resolve(repoRoot, "demo", "tools/../api", "npm", "x", ""); err == nil {
		t.Fatal("expected tools traversal to be rejected")
	}
	if _, err := Resolve(repoRoot, "demo", "api", "npm", "x", ""); err == nil || !strings.Contains(err.Error(), "surface directory not found") {
		t.Fatalf("expected missing-surface error, got %v", err)
	}
	if _, err := Resolve(repoRoot, "demo", "ui", "cargo", "x", ""); err == nil || !strings.Contains(err.Error(), "Cargo.toml") {
		t.Fatalf("cargo without manifest should be rejected, got %v", err)
	}
}

func TestResolveNeverGuessesManagerFromLanguageAlone(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{"package.json": "{}"})
	for _, ecosystem := range []string{"npm", "pip", "cargo", "go"} {
		t.Run(ecosystem, func(t *testing.T) {
			if _, err := Resolve(repoRoot, "demo", "ui", ecosystem, "safe-package", ""); err == nil {
				t.Fatalf("%s request was accepted without manager evidence", ecosystem)
			}
		})
	}
}

func TestResolveRejectsScenarioAndPackageFlagInjection(t *testing.T) {
	repoRoot := t.TempDir()
	mkSurface(t, repoRoot, "demo", "ui", map[string]string{"pnpm-lock.yaml": ""})
	for _, tc := range []struct {
		name, scenario, packageName, version string
	}{
		{"scenario traversal", "../outside", "safe", ""},
		{"package flag", "demo", "--ignore-scripts", ""},
		{"version flag", "demo", "safe", "--registry=https://evil"},
		{"shell syntax", "demo", "safe;curl", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Resolve(repoRoot, tc.scenario, "ui", "npm", tc.packageName, tc.version); err == nil {
				t.Fatal("unsafe install request was accepted")
			}
		})
	}
}

func TestAdapterRegistryMakesUnsupportedNativeManagersExplicit(t *testing.T) {
	for _, ecosystem := range []string{"npm", "pnpm", "yarn", "bun", "python", "go", "rust", "c", "cpp"} {
		adapter, err := AdapterFor(ecosystem)
		if err != nil {
			t.Fatalf("%s: %v", ecosystem, err)
		}
		if (ecosystem == "c" || ecosystem == "cpp") && adapter.MutationSupported() {
			t.Fatalf("%s must not claim a universal mutation adapter", ecosystem)
		}
	}
}

func TestSafeInstallProfileAndProtectedException(t *testing.T) {
	profile := SafeProfileFor("pnpm", []string{"pnpm", "add", "--ignore-scripts", "pkg"})
	if !profile.ScriptsDisabled || profile.LifecycleMode != "scripts-disabled-by-default" || profile.Governance != "scenario-dependency-analyzer" {
		t.Fatalf("unexpected install profile: %+v", profile)
	}
	if err := ValidateProtectedBuildException(ProtectedBuildException{Owner: "release", Reason: "native build", Command: "pnpm rebuild", PolicyMode: "guarded"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateProtectedBuildException(ProtectedBuildException{Owner: "release", Command: "pnpm rebuild", PolicyMode: "guarded"}); err == nil {
		t.Fatal("missing exception reason should be rejected")
	}
}

func TestFrozenReproductionArgsAreScriptSafe(t *testing.T) {
	tests := map[string][]string{
		"npm":    {"npm", "ci", "--ignore-scripts"},
		"pnpm":   {"pnpm", "install", "--frozen-lockfile", "--ignore-scripts"},
		"yarn":   {"yarn", "install", "--immutable", "--ignore-scripts"},
		"bun":    {"bun", "install", "--frozen-lockfile", "--ignore-scripts"},
		"cargo":  {"cargo", "fetch", "--locked"},
		"go":     {"go", "mod", "download"},
		"poetry": {"poetry", "install", "--sync", "--no-root"},
	}
	for manager, want := range tests {
		got, err := FrozenReproductionArgs(manager)
		if err != nil {
			t.Fatalf("%s: %v", manager, err)
		}
		if strings.Join(got, " ") != strings.Join(want, " ") {
			t.Errorf("%s = %v, want %v", manager, got, want)
		}
		profile := SafeProfileFor(manager, got)
		if !profile.FrozenLockfile || !profile.ScriptsDisabled && manager != "go" && manager != "cargo" && manager != "poetry" {
			t.Errorf("%s profile = %+v", manager, profile)
		}
	}
}
