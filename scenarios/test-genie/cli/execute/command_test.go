package execute

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	execTypes "test-genie/cli/internal/execute"
)

func TestResolveWorkspacePathsUsesSandboxHostAndLogicalRoots(t *testing.T) {
	hostMerged := t.TempDir()
	scenarioPath := filepath.Join(hostMerged, "scenarios", "demo")
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VROOLI_SANDBOX_MERGED_HOST", hostMerged)
	t.Setenv("VROOLI_SANDBOX_REPO_ROOT", "/canonical/Vrooli")
	parsed := Args{Scenario: "demo"}
	got, err := resolveWorkspacePaths(&parsed)
	if err != nil {
		t.Fatal(err)
	}
	if got != scenarioPath || parsed.LogicalRepoRoot != "/canonical/Vrooli" || parsed.LogicalScenarioRelPath != "scenarios/demo" {
		t.Fatalf("resolved path contract disagrees: path=%q parsed=%+v", got, parsed)
	}
}

func TestResolveWorkspacePathsNamesUnreachablePhysicalRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing", "scenarios", "demo")
	t.Setenv("VROOLI_SANDBOX_MERGED_HOST", filepath.Dir(filepath.Dir(missing)))
	parsed := Args{Scenario: "demo"}
	_, err := resolveWorkspacePaths(&parsed)
	if err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("error = %v, want unreachable path %q", err, missing)
	}
}

func TestPlannedPhaseNamesPreservesServerOrder(t *testing.T) {
	preview := execTypes.PlanPreview{
		Phases: []execTypes.PlanPhase{
			{Name: "structure"},
			{Name: "unit"},
			{Name: "integration"},
		},
	}

	got := plannedPhaseNames(preview)
	want := []string{"structure", "unit", "integration"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected ordered planned phases %v, got %v", want, got)
	}
}

func TestParseArgsAcceptsAbsoluteScenarioPath(t *testing.T) {
	scenarioPath := filepath.Join(t.TempDir(), "scenarios", "demo")
	parsed, err := ParseArgs([]string{"demo", "--scenario-path", scenarioPath, "--preset", "comprehensive"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if parsed.ScenarioPath != scenarioPath {
		t.Fatalf("ScenarioPath = %q, want %q", parsed.ScenarioPath, scenarioPath)
	}
	if parsed.Preset != "comprehensive" {
		t.Fatalf("Preset = %q", parsed.Preset)
	}
}

func TestParseArgsRejectsRelativeScenarioPath(t *testing.T) {
	if _, err := ParseArgs([]string{"demo", "--scenario-path", "scenarios/demo"}); err == nil {
		t.Fatal("expected relative --scenario-path to fail")
	}
}

func TestParseArgsAcceptsLogicalPlacement(t *testing.T) {
	scenarioPath := filepath.Join(t.TempDir(), "scenarios", "demo")
	repoRoot := t.TempDir()
	parsed, err := ParseArgs([]string{
		"demo",
		"--scenario-path", scenarioPath,
		"--logical-repo-root", repoRoot,
		"--logical-scenario-relpath", "scenarios/demo",
	})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if parsed.LogicalRepoRoot != repoRoot {
		t.Fatalf("LogicalRepoRoot = %q, want %q", parsed.LogicalRepoRoot, repoRoot)
	}
	if parsed.LogicalScenarioRelPath != "scenarios/demo" {
		t.Fatalf("LogicalScenarioRelPath = %q", parsed.LogicalScenarioRelPath)
	}
}

func TestParseArgsRejectsInvalidLogicalPlacement(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "missing relative placement",
			args: []string{"demo", "--logical-repo-root", t.TempDir()},
		},
		{
			name: "relative repo root",
			args: []string{"demo", "--logical-repo-root", "repo", "--logical-scenario-relpath", "scenarios/demo"},
		},
		{
			name: "absolute scenario relpath",
			args: []string{"demo", "--logical-repo-root", t.TempDir(), "--logical-scenario-relpath", filepath.Join(t.TempDir(), "scenarios/demo")},
		},
		{
			name: "escaping scenario relpath",
			args: []string{"demo", "--logical-repo-root", t.TempDir(), "--logical-scenario-relpath", "../demo"},
		},
		{
			name: "scenario name mismatch",
			args: []string{"demo", "--logical-repo-root", t.TempDir(), "--logical-scenario-relpath", "scenarios/other"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseArgs(tt.args); err == nil {
				t.Fatalf("expected ParseArgs(%v) to fail", tt.args)
			}
		})
	}
}

func TestExecutionResultErrorFailsUnsuccessfulJSONResult(t *testing.T) {
	if err := executionResultError(Response{Success: true}); err != nil {
		t.Fatalf("executionResultError(success) error = %v", err)
	}
	if err := executionResultError(Response{Success: false}); err == nil {
		t.Fatal("expected unsuccessful execution result to fail")
	}
}

func TestExtractErrorMessageIncludesStructuredDetails(t *testing.T) {
	got := extractErrorMessage([]byte(`{
		"success": false,
		"error": "suite execution failed",
		"errors": ["start target scenario demo: exit status 2"]
	}`))

	for _, want := range []string{"suite execution failed", "start target scenario demo: exit status 2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("extractErrorMessage() = %q, want %q", got, want)
		}
	}
}
