package main

import (
	"os"
	"path/filepath"
	"strings"
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
	h := newRepoHarness(t)
	h.UseRepoContext(t)

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
	if !strings.HasPrefix(fullPath, filepath.Join(h.Root, "logs", "scenario-auditor")) {
		t.Fatalf("artifact absolute path = %q", fullPath)
	}
}

func TestGetRootsUseRepoContractHelpers(t *testing.T) {
	h := newRepoHarness(t)
	h.UseRepoContext(t)

	ctx, err := repoContext()
	if err != nil {
		t.Fatalf("repoContext: %v", err)
	}
	if got := ctx.RepoRoot(); got != h.Root {
		t.Fatalf("RepoRoot = %q, want %q", got, h.Root)
	}
	if got := ctx.ScenarioAuditorRoot(); got != filepath.Join(h.Root, "scenarios", "scenario-auditor") {
		t.Fatalf("ScenarioAuditorRoot = %q", got)
	}
}

func TestRelativeToRepoRootUsesResolvedRepoRoot(t *testing.T) {
	h := newRepoHarness(t)
	h.UseRepoContext(t)

	ctx, err := repoContext()
	if err != nil {
		t.Fatalf("repoContext: %v", err)
	}
	got := ctx.RelativeToRepoRoot(filepath.Join(h.Root, "scenarios", "demo", "api", "main.go"))
	if got != "scenarios/demo/api/main.go" {
		t.Fatalf("RelativeToRepoRoot = %q", got)
	}
}

func TestDiscoverRuleDirsUsesContractResolvedScenarioAuditorPath(t *testing.T) {
	h := newRepoHarness(t)
	rulesDir := filepath.Join(h.Root, "scenarios", "scenario-auditor", "api", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}

	dirs, err := re.DiscoverRuleDirs(h.ContextOrBuild(t).ScenarioAuditorRoot())
	if err != nil {
		t.Fatalf("DiscoverRuleDirs: %v", err)
	}
	if len(dirs) != 1 || dirs[0] != rulesDir {
		t.Fatalf("DiscoverRuleDirs = %#v, want [%q]", dirs, rulesDir)
	}
}
