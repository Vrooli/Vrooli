package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	re "scenario-auditor/internal/ruleengine"
)

func TestBuildViolationSummaryAggregates(t *testing.T) {
	records := []violationRecord{
		{ID: "1", Severity: "low", RuleID: "CFG-001", Title: "Config issue", FilePath: "scenarios/foo/.vrooli/service.json", LineNumber: 10, Recommendation: "Fix config"},
		{ID: "2", Severity: "critical", RuleID: "SEC-999", Title: "Security gap", FilePath: "scenarios/foo/api/main.go", LineNumber: 42, Recommendation: "Patch vuln"},
		{ID: "3", Severity: "medium", RuleID: "SEC-999", Title: "Security gap", FilePath: "scenarios/foo/api/main.go", LineNumber: 84, Recommendation: "Patch vuln"},
	}

	summary := buildViolationSummary(records, 2)

	if summary.Total != 3 {
		t.Fatalf("expected total 3, got %d", summary.Total)
	}
	if summary.HighestSeverity != "critical" {
		t.Fatalf("expected highest severity critical, got %s", summary.HighestSeverity)
	}
	if summary.BySeverity["critical"] != 1 || summary.BySeverity["medium"] != 1 || summary.BySeverity["low"] != 1 {
		t.Fatalf("unexpected severity aggregation: %#v", summary.BySeverity)
	}
	if len(summary.ByRule) != 2 {
		t.Fatalf("expected 2 rule aggregates, got %d", len(summary.ByRule))
	}
	if summary.ByRule[0].RuleID != "SEC-999" || summary.ByRule[0].Count != 2 {
		t.Fatalf("expected SEC-999 to be top rule, got %#v", summary.ByRule[0])
	}
	if len(summary.TopViolations) != 2 {
		t.Fatalf("expected 2 top violations due to limit, got %d", len(summary.TopViolations))
	}
	if summary.TopViolations[0].Severity != "critical" {
		t.Fatalf("expected most severe violation first, got %#v", summary.TopViolations)
	}
	if len(summary.RecommendedSteps) == 0 {
		t.Fatalf("expected recommended steps to be populated")
	}
}

func TestCloneSummaryFiltersBySeverityAndLimit(t *testing.T) {
	records := []violationRecord{
		{ID: "1", Severity: "low"},
		{ID: "2", Severity: "medium"},
		{ID: "3", Severity: "high"},
	}
	summary := buildViolationSummary(records, 3)

	filtered := cloneSummary(&summary, 1, "high")
	if filtered == nil {
		t.Fatalf("expected filtered summary")
	}
	if len(filtered.TopViolations) != 1 {
		t.Fatalf("expected top violations limited to 1, got %d", len(filtered.TopViolations))
	}
	if filtered.TopViolations[0].Severity != "high" {
		t.Fatalf("expected remaining violation to be high severity, got %s", filtered.TopViolations[0].Severity)
	}
}

func TestPersistScanArtifactUsesContractResolvedRepoRoot(t *testing.T) {
	root := writeRepoContractFixture(t)
	t.Setenv("VROOLI_ROOT", root)
	chdirForTest(t, filepath.Join(root, "scenarios", "scenario-auditor", "api"))

	artifact, err := persistScanArtifact("standards", "demo", "job-123", map[string]any{"ok": true})
	if err != nil {
		t.Fatalf("persistScanArtifact: %v", err)
	}
	if !strings.HasPrefix(artifact.Path, "logs/scenario-auditor/standards/demo/") {
		t.Fatalf("artifact path = %q", artifact.Path)
	}

	fullPath, err := resolveArtifactAbsolutePath(artifact.Path)
	if err != nil {
		t.Fatalf("resolveArtifactAbsolutePath: %v", err)
	}
	if !strings.HasPrefix(fullPath, filepath.Join(root, "logs", "scenario-auditor")) {
		t.Fatalf("artifact absolute path = %q", fullPath)
	}
}

func TestGetRootsUseRepoContractHelpers(t *testing.T) {
	root := writeRepoContractFixture(t)
	t.Setenv("VROOLI_ROOT", root)
	chdirForTest(t, filepath.Join(root, "scenarios", "scenario-auditor", "api"))
	resetCachedRootsForTest()

	if got := currentVrooliRoot(); got != root {
		t.Fatalf("getVrooliRoot = %q, want %q", got, root)
	}
	if got := getScenarioRoot(); got != filepath.Join(root, "scenarios", "scenario-auditor") {
		t.Fatalf("getScenarioRoot = %q", got)
	}
}

func TestRelativeToRepoRootUsesResolvedRepoRoot(t *testing.T) {
	root := writeRepoContractFixture(t)
	t.Setenv("VROOLI_ROOT", root)
	chdirForTest(t, filepath.Join(root, "scenarios", "scenario-auditor", "api"))

	got := relativeToRepoRoot(filepath.Join(root, "scenarios", "demo", "api", "main.go"))
	if got != "scenarios/demo/api/main.go" {
		t.Fatalf("relativeToRepoRoot = %q", got)
	}
}

func TestDiscoverRuleDirsUsesContractResolvedScenarioAuditorPath(t *testing.T) {
	root := writeRepoContractFixture(t)
	rulesDir := filepath.Join(root, "scenarios", "scenario-auditor", "api", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}

	dirs, err := re.DiscoverRuleDirs(root)
	if err != nil {
		t.Fatalf("DiscoverRuleDirs: %v", err)
	}
	if len(dirs) != 1 || dirs[0] != rulesDir {
		t.Fatalf("DiscoverRuleDirs = %#v, want [%q]", dirs, rulesDir)
	}
}

func writeRepoContractFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	writeRepoContractFixtureAtRoot(t, root)
	return root
}

func writeRepoContractFixtureAtRoot(t *testing.T, root string) {
	t.Helper()

	for _, dir := range []string{".vrooli", "scenarios", "resources", "packages", "cmd", "internal"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "scenario-auditor", "api"), 0o755); err != nil {
		t.Fatalf("mkdir scenario-auditor api: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "scenarios", "demo", "api"), 0o755); err != nil {
		t.Fatalf("mkdir demo api: %v", err)
	}
	writeJSONFile(t, filepath.Join(root, "scenarios", "scenario-auditor", ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": "scenario-auditor"},
	})
	writeJSONFile(t, filepath.Join(root, "scenarios", "demo", ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": "demo"},
	})
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	contract := `{
  "$schema": "schemas/repo-contract.schema.json",
  "version": "1.0.0",
  "platform": {"mode": "cross_platform_go_native", "legacy_project_bash_supported": false},
  "root": {"markers": {"required_dirs": [".vrooli", "scenarios", "resources", "packages", "cmd", "internal"], "required_files": ["go.mod"]}},
  "layout": {"project_config_dir": ".vrooli", "scenario_dir": "scenarios", "resource_dir": "resources", "package_dir": "packages", "command_dir": "cmd", "internal_dir": "internal", "docs_dir": "docs"},
  "scenario": {"required_files": [".vrooli/service.json"], "well_known_paths": {"service": ".vrooli/service.json", "docs": "docs", "requirements": "requirements", "api": "api", "ui": "ui", "cli": "cli", "initialization": "initialization"}},
  "resource": {"manifest": "resource.json", "well_known_paths": {"docs": "docs", "initialization": "initialization"}},
  "globs": {"syntax": "doublestar", "root_relative": true, "case_sensitive": true, "allow_absolute": false, "path_format": "slash_normalized"},
  "environment": {"variables": {"repo_root": "VROOLI_ROOT", "source_root": "VROOLI_SOURCE_ROOT", "sandbox_id": "VROOLI_SANDBOX_ID", "sandbox_merged": "VROOLI_SANDBOX_MERGED", "sandbox_scope": "VROOLI_SANDBOX_SCOPE"}},
  "sandbox": {"full_repo_scopes": ["", ".", "/"], "scenario_scope_prefix": "scenarios/"},
  "profiles": {
    "fixture": {
      "description": "fixture profile",
      "parameters": ["scenario"],
      "include": ["scenarios/{scenario}"],
      "optional_include": ["go.mod"],
      "exclude": [".git/**"]
    }
  }
}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), []byte(contract), 0o644); err != nil {
		t.Fatalf("write repo-contract.json: %v", err)
	}
}

func writeJSONFile(t *testing.T, path string, payload map[string]any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func resetCachedRootsForTest() {
	scenarioRootOnce = sync.Once{}
	scenarioRootPath = ""
	vrooliRootOnce = sync.Once{}
	vrooliRootPath = ""
}

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	resetCachedRootsForTest()
}
