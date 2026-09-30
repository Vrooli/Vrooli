package projectroot

import (
	"path/filepath"
	"testing"

	repocontract "github.com/vrooli/repo-contract-go"
)

// repoRootForTest returns the canonical Vrooli repo root that the test binary
// is running inside. repo-contract validates the .vrooli/repo-contract.json
// marker, so we cannot just point VROOLI_ROOT at an arbitrary tmpdir; we use
// the real root and verify Resolve produces it. This mirrors the pattern in
// internal/pathutil/root_test.go.
func repoRootForTest(t *testing.T) string {
	t.Helper()
	root, err := repocontract.FindRepoRootFromCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromCWD: %v", err)
	}
	t.Setenv("VROOLI_ROOT", root)
	return root
}

func TestResolve_SingleScenarioNarrowsScope(t *testing.T) {
	repoRoot := repoRootForTest(t)

	got, err := Resolve(Options{
		AcceptanceAllow: []string{
			"scenarios/development-toolchain-validator/cli/**",
			"scenarios/development-toolchain-validator/api/**",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}

	if got.ProjectRoot != repoRoot {
		t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, repoRoot)
	}
	if got.ScopePath != "scenarios/development-toolchain-validator" {
		t.Errorf("ScopePath = %q, want %q", got.ScopePath, "scenarios/development-toolchain-validator")
	}
	if got.TargetScenario != "development-toolchain-validator" {
		t.Errorf("TargetScenario = %q, want %q", got.TargetScenario, "development-toolchain-validator")
	}
}

func TestResolve_DedupsRepeatedScenario(t *testing.T) {
	repoRootForTest(t)

	got, err := Resolve(Options{
		AcceptanceAllow: []string{
			"scenarios/web-console/api/**",
			"scenarios/web-console/ui/**",
			"scenarios/web-console/cli/**",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ScopePath != "scenarios/web-console" {
		t.Errorf("ScopePath = %q, want %q", got.ScopePath, "scenarios/web-console")
	}
}

func TestResolve_MultipleScenariosFallsBackToWideScope(t *testing.T) {
	repoRoot := repoRootForTest(t)

	// Cross-cutting items targeting multiple scenarios must resolve to a
	// monorepo-wide scope rather than erroring.
	got, err := Resolve(Options{
		AcceptanceAllow: []string{
			"scenarios/foo/cli/**",
			"scenarios/bar/api/**",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ProjectRoot != repoRoot {
		t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, repoRoot)
	}
	if got.ScopePath != "." {
		t.Errorf("ScopePath = %q, want %q for multi-scenario wide-scope fallback", got.ScopePath, ".")
	}
	if got.TargetScenario != "" {
		t.Errorf("TargetScenario should be empty when multiple scenarios match, got %q", got.TargetScenario)
	}
}

func TestResolve_EmptyAcceptanceFallsBackToWideScope(t *testing.T) {
	repoRoot := repoRootForTest(t)

	got, err := Resolve(Options{AcceptanceAllow: nil})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ProjectRoot != repoRoot {
		t.Errorf("ProjectRoot = %q, want %q", got.ProjectRoot, repoRoot)
	}
	if got.ScopePath != "." {
		t.Errorf("ScopePath = %q, want %q for wide-scope fallback", got.ScopePath, ".")
	}
	if got.TargetScenario != "" {
		t.Errorf("TargetScenario should be empty in wide-scope mode, got %q", got.TargetScenario)
	}
}

func TestResolve_NonScenarioGlobsFallBackToWideScope(t *testing.T) {
	repoRootForTest(t)

	got, err := Resolve(Options{
		AcceptanceAllow: []string{
			"packages/proto/**",
			"**/*.md",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if got.ScopePath != "." {
		t.Errorf("ScopePath = %q, want %q", got.ScopePath, ".")
	}
}

func TestResolve_MixedScenarioAndNonScenarioGlobsNarrows(t *testing.T) {
	repoRootForTest(t)

	got, err := Resolve(Options{
		AcceptanceAllow: []string{
			"scenarios/swarm-manager/api/**",
			"packages/proto/**",
		},
	})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	// pathutil.ScenariosFromGlobs filters non-scenario globs, so this is
	// treated as a single-scenario item.
	if got.ScopePath != "scenarios/swarm-manager" {
		t.Errorf("ScopePath = %q, want %q", got.ScopePath, "scenarios/swarm-manager")
	}
}

func TestResolve_ProjectRootIsAbsolute(t *testing.T) {
	repoRootForTest(t)

	got, err := Resolve(Options{AcceptanceAllow: []string{"scenarios/foo/**"}})
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	if !filepath.IsAbs(got.ProjectRoot) {
		t.Errorf("ProjectRoot must be absolute, got %q", got.ProjectRoot)
	}
}
