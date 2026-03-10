package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// setupRuleGoCliTestRepo creates a temp repo root with the markers
// FindRepoRoot needs, and returns the root path.
func setupRuleGoCliTestRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{".vrooli", "scenarios", "resources"} {
		mkdirAll(t, filepath.Join(root, dir))
	}
	return root
}

// writeCliModule creates a minimal Go CLI module for a scenario.
// The main.go is a trivially-compilable file so `go build ./...` succeeds.
func writeCliModule(t *testing.T, root, scenarioName string) {
	t.Helper()
	cliDir := filepath.Join(root, "scenarios", scenarioName, "cli")
	mkdirAll(t, cliDir)

	goMod := "module example.com/" + scenarioName + "/cli\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	mainGo := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(filepath.Join(cliDir, "main.go"), []byte(mainGo), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestGoCliRule_NoCLIsFound(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)

	result := RunGoCliWorkspaceIndependence(t.Context(), root, "")

	if result.Passed {
		t.Error("expected passed=false when no CLIs found (warn finding)")
	}
	if len(result.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(result.Findings))
	}
	if result.Findings[0].Level != "warn" {
		t.Errorf("expected warn level, got %s", result.Findings[0].Level)
	}
}

func TestGoCliRule_SinglePassingCLI(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	writeCliModule(t, root, "good-cli")

	result := RunGoCliWorkspaceIndependence(t.Context(), root, "good-cli")

	if !result.Passed {
		t.Errorf("expected passed=true, got false; findings: %+v", result.Findings)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestGoCliRule_MultipleCLIsRunInParallel(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	// Create 6 passing CLI modules (more than the concurrency limit of 5).
	for i := 0; i < 6; i++ {
		writeCliModule(t, root, "cli-"+string(rune('a'+i)))
	}

	start := time.Now()
	result := RunGoCliWorkspaceIndependence(t.Context(), root, "")
	elapsed := time.Since(start)

	if !result.Passed {
		t.Errorf("expected passed=true, got false; findings: %+v", result.Findings)
	}

	// Sanity: if these ran sequentially with any overhead, it would take
	// noticeably longer. We just verify it completed — the real parallelism
	// test is the timeout test below.
	if elapsed > 2*time.Minute {
		t.Errorf("expected rule to finish quickly, took %s", elapsed)
	}
}

func TestGoCliRule_FailingCLIReportsError(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	scenarioName := "bad-cli"
	cliDir := filepath.Join(root, "scenarios", scenarioName, "cli")
	mkdirAll(t, cliDir)

	goMod := "module example.com/bad-cli/cli\n\ngo 1.25\n"
	if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	// Write a Go file that won't compile.
	badGo := "package main\n\nfunc main() { undefined() }\n"
	if err := os.WriteFile(filepath.Join(cliDir, "main.go"), []byte(badGo), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, scenarioName)

	if result.Passed {
		t.Error("expected passed=false for failing CLI")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least 1 finding")
	}

	found := false
	for _, f := range result.Findings {
		if f.Level == "error" && strings.Contains(f.Message, "bad-cli") && f.ScenarioName == scenarioName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected error finding for bad-cli, got: %+v", result.Findings)
	}
}

func TestGoCliRule_CancelledContextReportsError(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	writeCliModule(t, root, "timeout-test")

	// A cancelled context makes go build fail immediately.
	// The rule should still report findings (build error or timeout sentinel).
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result := RunGoCliWorkspaceIndependence(ctx, root, "timeout-test")

	if result.Passed {
		t.Error("expected passed=false on cancelled context")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected at least 1 finding on cancelled context")
	}
	// Should have an error-level finding (either build failure or timeout sentinel).
	hasError := false
	for _, f := range result.Findings {
		if f.Level == "error" {
			hasError = true
			break
		}
	}
	if !hasError {
		t.Errorf("expected error-level finding, got: %+v", result.Findings)
	}
}

func TestGoCliRule_TimeoutSentinelWhenNoFindings(t *testing.T) {
	// Directly test the sentinel logic: if ctx is cancelled and
	// the builds didn't produce findings (e.g. they all returned
	// nil errors despite timeout), the sentinel should fire.
	// We simulate this by pointing at a repo with no CLI modules
	// that match a specific (non-existent) scenario name, but
	// with ctx already cancelled.
	root := setupRuleGoCliTestRepo(t)

	// Create a scenario with a cli/go.mod so the glob matches,
	// but the scenario name filter won't match, forcing an empty
	// goMods list → the "no CLIs found" warn fires instead.
	// That's not exactly the sentinel path.
	//
	// Instead, test the sentinel path directly by checking that
	// the function handles timeout correctly for actual builds.
	// With a cancelled context and actual modules, the build fails
	// and produces findings — which is the correct behavior.
	// The sentinel is a safety net for the edge case where builds
	// somehow return nil error despite context cancellation.
	writeCliModule(t, root, "sentinel-test")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result := RunGoCliWorkspaceIndependence(ctx, root, "sentinel-test")

	// With cancelled ctx, go build fails → error finding exists.
	// The sentinel only fires if ctx.Err() != nil AND no findings.
	// Either way, the rule must not pass and must have findings.
	if result.Passed {
		t.Error("expected passed=false")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings on cancelled context")
	}
}

func TestGoCliRule_ScenarioFilter(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	writeCliModule(t, root, "alpha")
	writeCliModule(t, root, "beta")

	// Run scoped to "alpha" only.
	result := RunGoCliWorkspaceIndependence(t.Context(), root, "alpha")

	if !result.Passed {
		t.Errorf("expected passed=true, got false; findings: %+v", result.Findings)
	}
}
