package adapters

import (
	"os"
	"path/filepath"
	"testing"

	"unit-health/internal/discovery"
)

func TestResolveWorkspacePlansPolyglotSurfacesThroughAdapters(t *testing.T) {
	goResolution, goDiagnostics := ResolveWorkspace(WorkspaceInput{Language: "go", Surface: discovery.Surface{ID: "api", RootPath: t.TempDir()}})
	if goResolution.CanonicalFramework != "go test" || !goResolution.CoverageRequested || len(goDiagnostics) != 0 {
		t.Fatalf("go resolution=%+v diagnostics=%+v", goResolution, goDiagnostics)
	}

	pythonResolution, pythonDiagnostics := ResolveWorkspace(WorkspaceInput{Language: "python", Surface: discovery.Surface{ID: "worker", RootPath: t.TempDir()}})
	if pythonResolution.Framework != "pytest" || pythonResolution.Status != "degraded" || len(pythonDiagnostics) != 1 || pythonDiagnostics[0].Code != diagnosticUnsupported {
		t.Fatalf("python resolution=%+v diagnostics=%+v", pythonResolution, pythonDiagnostics)
	}
}

func TestResolveWorkspaceUsesMypyWhenConfigured(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[tool.mypy]\ncheck_untyped_defs = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, _ := ResolveWorkspace(WorkspaceInput{Language: "python", Surface: discovery.Surface{ID: "api", RootPath: root}})
	if resolution.Framework != "mypy" || resolution.CanonicalFramework != "mypy" {
		t.Fatalf("python typecheck resolution=%+v", resolution)
	}
}

func TestResolveWorkspaceAddsTypecheckPlanForTypeCheckScript(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run","test:coverage":"vitest run --coverage","type-check":"tsc --noEmit"},"devDependencies":{"vitest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language: "typescript", Surface: discovery.Surface{ID: "ui", RootPath: root, PackageManager: "pnpm"},
		ResolveExecutable: func([]string) (string, bool) { return filepath.Join(root, "pnpm"), true },
	})
	if len(diagnostics) != 0 || resolution.TypecheckCommand != "pnpm run type-check" || resolution.TypecheckExecutable == "" || len(resolution.TypecheckArgs) != 2 {
		t.Fatalf("resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}

func TestResolveWorkspaceJestIsExplicitlyDegraded(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"jest"},"devDependencies":{"jest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "typescript", Surface: discovery.Surface{ID: "ui", RootPath: root, PackageManager: "pnpm"}})
	if resolution.Framework != "jest" || resolution.Status != "degraded" || resolution.TestCommand != "pnpm test" {
		t.Fatalf("resolution=%+v", resolution)
	}
	if len(diagnostics) == 0 || diagnostics[0].Code != diagnosticNoncanonical {
		t.Fatalf("diagnostics=%+v", diagnostics)
	}
}

func TestResolveWorkspaceGenericNodeJestIsNotGivenReactFindings(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"jest"},"devDependencies":{"jest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "node", Surface: discovery.Surface{ID: "worker", RootPath: root, PackageManager: "npm"}})
	if resolution.Framework != "jest" || resolution.CanonicalFramework != "jest" || resolution.Status != "ready" || resolution.TestCommand != "npm test" {
		t.Fatalf("resolution=%+v", resolution)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("generic Node/Jest diagnostics=%+v", diagnostics)
	}
}

func TestResolveWorkspaceBatsUsesResolvedExecutable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "smoke.bats"), []byte("@test 'ok' { true; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language:          "bash",
		Surface:           discovery.Surface{ID: "cli", RootPath: root},
		ResolveExecutable: func([]string) (string, bool) { return filepath.Join(root, "bats with spaces"), true },
	})
	if resolution.Status != "ready" || resolution.TestExecutable == "" || len(diagnostics) != 0 {
		t.Fatalf("resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}

func TestResolveWorkspaceUsesResolvedLaunchersForPolyglotAdapters(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run","test:coverage":"vitest run --coverage"},"devDependencies":{"vitest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolve := func(candidates []string) (string, bool) {
		if len(candidates) == 1 && candidates[0] == "pnpm" {
			return filepath.Join(root, "tools", "pnpm with spaces"), true
		}
		return "", false
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language: "typescript", Surface: discovery.Surface{ID: "ui", RootPath: root, PackageManager: "pnpm"}, ResolveExecutable: resolve,
	})
	if resolution.Status != "ready" || resolution.TestExecutable == "" || len(diagnostics) != 0 {
		t.Fatalf("node resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}

	python, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language: "python", Surface: discovery.Surface{ID: "worker", RootPath: root}, ResolveExecutable: func(candidates []string) (string, bool) {
			if len(candidates) > 0 && candidates[0] == "python3" {
				return filepath.Join(root, "python with spaces"), true
			}
			return "", false
		},
	})
	if python.TestExecutable == "" || len(diagnostics) != 1 || diagnostics[0].Code != diagnosticUnsupported {
		t.Fatalf("python resolution=%+v diagnostics=%+v", python, diagnostics)
	}
}

func TestResolveWorkspaceNormalizesVersionedPackageManagerForExecution(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"scripts":{"test":"vitest run","test:coverage":"vitest run --coverage"},"devDependencies":{"vitest":"1"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolve := func(candidates []string) (string, bool) {
		if len(candidates) == 1 && candidates[0] == "pnpm" {
			return filepath.Join(root, "pnpm"), true
		}
		return "", false
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language:          "typescript",
		Surface:           discovery.Surface{ID: "ui", RootPath: root, PackageManager: "pnpm@9.12.1"},
		ResolveExecutable: resolve,
	})
	if resolution.Status != "ready" || resolution.TestExecutable == "" || len(diagnostics) != 0 {
		t.Fatalf("versioned package manager was not normalized: resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}

func TestResolveWorkspaceReportsMissingPolyglotLauncher(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\nname='demo'\nversion='0.1.0'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{
		Language: "rust", Surface: discovery.Surface{ID: "cli", RootPath: root}, ResolveExecutable: func([]string) (string, bool) { return "", false },
	})
	if resolution.Status != "degraded" || len(diagnostics) != 1 || diagnostics[0].Code != diagnosticDependency {
		t.Fatalf("resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}

func TestResolvePythonCoverageOnlyWhenPytestCovIsDeclared(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "pyproject.toml"), []byte("[project.optional-dependencies]\ntest = ['pytest-cov']\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "python", Surface: discovery.Surface{ID: "worker", RootPath: root}})
	if !resolution.CoverageRequested || len(diagnostics) != 1 || diagnostics[0].Code != diagnosticUnsupported {
		t.Fatalf("resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}

func TestResolveRustCoverageWhenCargoLLVMcovIsResolvable(t *testing.T) {
	root := t.TempDir()
	resolve := func(candidates []string) (string, bool) {
		if len(candidates) == 1 && candidates[0] == "cargo" {
			return filepath.Join(root, "cargo"), true
		}
		if len(candidates) == 1 && candidates[0] == "cargo-llvm-cov" {
			return filepath.Join(root, "cargo-llvm-cov"), true
		}
		return "", false
	}
	resolution, diagnostics := ResolveWorkspace(WorkspaceInput{Language: "rust", Surface: discovery.Surface{ID: "cli", RootPath: root}, ResolveExecutable: resolve})
	if resolution.Status != "ready" || !resolution.CoverageRequested || resolution.CoverageExecutable == "" || len(diagnostics) != 0 {
		t.Fatalf("resolution=%+v diagnostics=%+v", resolution, diagnostics)
	}
}
