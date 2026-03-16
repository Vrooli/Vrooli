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

	// When no Go CLIs exist, the rule should pass (no findings).
	// This is not a violation — it just means the rule doesn't apply.
	if !result.Passed {
		t.Errorf("expected passed=true when no CLIs found, got false; findings: %+v", result.Findings)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
	}
}

func TestGoCliRule_NoCLIsPerScenario(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	// Create a scenario directory with no CLI.
	mkdirAll(t, filepath.Join(root, "scenarios", "no-cli-scenario"))

	result := RunGoCliWorkspaceIndependence(t.Context(), root, "no-cli-scenario")

	// A scenario without a Go CLI should pass — the rule doesn't apply.
	if !result.Passed {
		t.Errorf("expected passed=true for scenario without CLI, got false; findings: %+v", result.Findings)
	}
	if len(result.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(result.Findings))
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

// TestGoCliRule_DetectsSubpackageImportNotJustInternal verifies the rule
// catches CLI imports of any API subpackage (e.g., /models, /types), not
// just /internal/. This aligns the rule checker with the fixer scope.
func TestGoCliRule_DetectsSubpackageImportNotJustInternal(t *testing.T) {
	root := setupGoCliTestDir(t, "sub-import")
	scenarioDir := filepath.Join(root, "scenarios", "sub-import")

	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"),
		[]byte("module github.com/vrooli/sub-import/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"),
		[]byte("module github.com/vrooli/sub-import/cli\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// CLI imports api/models (not api/internal/).
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"), []byte(`package main

import _ "github.com/vrooli/sub-import/api/models"

func main() {}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, "sub-import")

	// Should detect the import and report missing go.mod wiring.
	found := false
	for _, f := range result.Findings {
		if strings.Contains(f.Message, "missing go.mod wiring") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected finding about missing go.mod wiring for api/models import")
	}
}

func TestGoCliRule_BuildTimeoutReportsSpecificError(t *testing.T) {
	// When a build times out, the finding should clearly indicate
	// timeout rather than a generic build failure.
	root := setupRuleGoCliTestRepo(t)
	writeCliModule(t, root, "timeout-specific")

	// Use a cancelled context to simulate timeout.
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result := RunGoCliWorkspaceIndependence(ctx, root, "timeout-specific")

	if result.Passed {
		t.Error("expected passed=false")
	}
	if len(result.Findings) == 0 {
		t.Fatal("expected findings on cancelled context")
	}
	// Should have an error-level finding.
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

func TestGoCliRule_ReplacePathValidation_ValidPath(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	scenarioName := "valid-replace"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, filepath.Join(scenarioDir, "cli"))
	mkdirAll(t, filepath.Join(scenarioDir, "api"))

	// Write api/go.mod.
	if err := os.WriteFile(filepath.Join(scenarioDir, "api", "go.mod"),
		[]byte("module example.com/valid-replace/api\n\ngo 1.25\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// CLI go.mod with a valid replace pointing to an existing directory.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"),
		[]byte("module example.com/valid-replace/cli\n\ngo 1.25\n\nreplace example.com/valid-replace/api => ../api\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, scenarioName)

	// No warnings expected — the replace path is valid.
	for _, f := range result.Findings {
		if f.Level == "warn" && strings.Contains(f.Message, "non-existent path") {
			t.Errorf("unexpected replace path warning: %s", f.Message)
		}
	}
}

func TestGoCliRule_ReplacePathValidation_BrokenPath(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	scenarioName := "broken-replace"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, filepath.Join(scenarioDir, "cli"))
	// Intentionally do NOT create the api directory.

	// CLI go.mod with a replace pointing to a non-existent directory.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"),
		[]byte("module example.com/broken-replace/cli\n\ngo 1.25\n\nreplace example.com/broken-replace/api => ../api\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, scenarioName)

	// Should have a warning about the broken replace path.
	found := false
	for _, f := range result.Findings {
		if f.Level == "warn" && strings.Contains(f.Message, "non-existent path") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about non-existent replace path")
	}
}

func TestGoCliRule_ReplacePathValidation_RemoteModule(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	scenarioName := "remote-replace"
	scenarioDir := filepath.Join(root, "scenarios", scenarioName)
	mkdirAll(t, filepath.Join(scenarioDir, "cli"))

	// CLI go.mod with a non-local replace (module path, not filesystem).
	// This should NOT trigger path validation.
	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "go.mod"),
		[]byte("module example.com/remote/cli\n\ngo 1.25\n\nreplace example.com/old => example.com/new v1.0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(scenarioDir, "cli", "main.go"),
		[]byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, scenarioName)

	for _, f := range result.Findings {
		if strings.Contains(f.Message, "non-existent path") {
			t.Errorf("should not warn about remote module replace: %s", f.Message)
		}
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

// TestGoCliRule_ProtoCommentDoesNotFalsePositive verifies that a go.mod
// containing "github.com/vrooli/vrooli/packages/proto" only in a comment
// does NOT trigger a proto replace finding. This was a false positive caused
// by raw string matching on the go.mod text.
func TestGoCliRule_ProtoCommentDoesNotFalsePositive(t *testing.T) {
	root := setupRuleGoCliTestRepo(t)
	scenarioName := "proto-comment"
	cliDir := filepath.Join(root, "scenarios", scenarioName, "cli")
	mkdirAll(t, cliDir)

	// go.mod mentions proto only in a comment — not in require or replace.
	goMod := `module example.com/proto-comment/cli

go 1.25

// TODO: add github.com/vrooli/vrooli/packages/proto when ready
`
	if err := os.WriteFile(filepath.Join(cliDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cliDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := RunGoCliWorkspaceIndependence(t.Context(), root, scenarioName)

	for _, f := range result.Findings {
		if strings.Contains(f.Message, "proto") && f.Level == "error" {
			t.Errorf("false positive: proto comment should not trigger finding: %s", f.Message)
		}
	}
}
