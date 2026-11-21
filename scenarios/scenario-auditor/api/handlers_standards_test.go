package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"scenario-auditor/internal/repocontext"
	rulespkg "scenario-auditor/rules"
)

func TestBuildRuleBucketsRespectsDisabledStates(t *testing.T) {
	originalStore := ruleStateStore
	defer func() { ruleStateStore = originalStore }()

	ruleStateStore = &RuleStateStore{states: make(map[string]bool)}

	rules := map[string]RuleInfo{
		"enabled_rule": {
			Rule: rulespkg.Rule{
				ID:       "enabled_rule",
				Category: "api",
				Enabled:  true,
			},
			Targets: []string{"api"},
		},
		"disabled_rule": {
			Rule: rulespkg.Rule{
				ID:       "disabled_rule",
				Category: "api",
				Enabled:  true,
			},
			Targets: []string{"api"},
		},
		"metadata_disabled": {
			Rule: rulespkg.Rule{
				ID:       "metadata_disabled",
				Category: "api",
				Enabled:  false,
			},
			Targets: []string{"api"},
		},
	}

	if err := ruleStateStore.SetState("disabled_rule", false); err != nil {
		t.Fatalf("SetState returned error: %v", err)
	}

	buckets, active, disabledRequested, missingRequested := buildRuleBuckets(rules, nil, false)
	if len(disabledRequested) != 0 || len(missingRequested) != 0 {
		t.Fatalf("expected no disabled/missing results for broad scan, got disabled=%v missing=%v", disabledRequested, missingRequested)
	}

	if _, ok := active["disabled_rule"]; ok {
		t.Fatalf("expected disabled_rule to be filtered out when disabled via state store")
	}

	if _, ok := active["metadata_disabled"]; ok {
		t.Fatalf("expected metadata_disabled to be filtered out due to metadata flag")
	}

	apiBucket := buckets["api"]
	if len(apiBucket) != 1 {
		t.Fatalf("expected exactly one rule in api bucket, got %d", len(apiBucket))
	}
	if apiBucket[0].ID != "enabled_rule" {
		t.Fatalf("expected enabled_rule to remain active, got %s", apiBucket[0].ID)
	}

	_, targetedActive, targetedDisabled, targetedMissing := buildRuleBuckets(rules, []string{"disabled_rule"}, false)
	if len(targetedMissing) != 0 {
		t.Fatalf("did not expect missing rules, got %v", targetedMissing)
	}
	if len(targetedDisabled) != 1 || targetedDisabled[0] != "disabled_rule" {
		t.Fatalf("expected disabled_rule to be reported as disabled, got %v", targetedDisabled)
	}
	if len(targetedActive) != 0 {
		t.Fatalf("expected no active targeted rules when the requested rule is disabled, got %d", len(targetedActive))
	}

	_, forcedActive, forcedDisabled, forcedMissing := buildRuleBuckets(rules, []string{"disabled_rule"}, true)
	if len(forcedMissing) != 0 || len(forcedDisabled) != 0 {
		t.Fatalf("did not expect disabled or missing results when forcing, got disabled=%v missing=%v", forcedDisabled, forcedMissing)
	}
	if len(forcedActive) != 1 {
		t.Fatalf("expected forced run to include disabled_rule, got %d active rules", len(forcedActive))
	}
}

func TestClassifyFileTargetsMakefile(t *testing.T) {
	root := filepath.Join("/tmp", "project", "scenarios", "demo")
	fullPath := filepath.Join(root, "Makefile")
	scenario, relative, targets := classifyFileTargets(fullPath)

	if scenario != "demo" {
		t.Fatalf("expected scenario demo, got %s", scenario)
	}
	if relative != "Makefile" {
		t.Fatalf("expected relative path Makefile, got %s", relative)
	}
	if len(targets) != 1 || targets[0] != targetMakefile {
		t.Fatalf("expected targets [%s], got %v", targetMakefile, targets)
	}
}

func TestClassifyFileTargetsPRD(t *testing.T) {
	root := filepath.Join("/tmp", "project", "scenarios", "demo")
	fullPath := filepath.Join(root, "PRD.md")
	scenario, relative, targets := classifyFileTargets(fullPath)

	if scenario != "demo" {
		t.Fatalf("expected scenario demo, got %s", scenario)
	}
	if relative != "PRD.md" {
		t.Fatalf("expected relative path PRD.md, got %s", relative)
	}
	if len(targets) != 1 || targets[0] != targetDocumentation {
		t.Fatalf("expected targets [%s], got %v", targetDocumentation, targets)
	}
}

func TestGetStandardsViolationsSummaryHandler(t *testing.T) {
	scenario := "summary-target"
	violations := []StandardsViolation{
		{ID: "V-1", ScenarioName: scenario, Severity: "critical", Title: "Critical issue", FilePath: "api/main.go", LineNumber: 10, Recommendation: "Patch immediately"},
		{ID: "V-2", ScenarioName: scenario, Severity: "medium", Title: "Medium issue", FilePath: "ui/src/App.tsx", LineNumber: 42},
		{ID: "V-3", ScenarioName: scenario, Severity: "low", Title: "Low issue", FilePath: "README.md", LineNumber: 5},
	}
	standardsStore.StoreViolations(scenario, violations)

	req := httptest.NewRequest(http.MethodGet, "/standards/violations/summary?scenario="+scenario+"&limit=2&min_severity=medium", nil)
	recorder := httptest.NewRecorder()

	getStandardsViolationsSummaryHandler(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", recorder.Code)
	}

	var payload struct {
		Summary *ViolationSummary `json:"summary"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload.Summary == nil {
		t.Fatal("expected summary payload")
	}
	if payload.Summary.Total != len(violations) {
		t.Fatalf("expected total %d, got %d", len(violations), payload.Summary.Total)
	}
	if payload.Summary.HighestSeverity != "critical" {
		t.Fatalf("expected highest severity critical, got %s", payload.Summary.HighestSeverity)
	}
	if len(payload.Summary.TopViolations) == 0 {
		t.Fatalf("expected top violations slice to be populated")
	}
	for _, v := range payload.Summary.TopViolations {
		if normalizeSeverity(v.Severity) == "low" {
			t.Fatalf("expected min_severity filter to exclude low severity entries, got %#v", v)
		}
	}
}

func TestClassifyFileTargetsReadme(t *testing.T) {
	root := filepath.Join("/tmp", "project", "scenarios", "demo")
	fullPath := filepath.Join(root, "README.md")
	scenario, relative, targets := classifyFileTargets(fullPath)

	if scenario != "demo" {
		t.Fatalf("expected scenario demo, got %s", scenario)
	}
	if relative != "README.md" {
		t.Fatalf("expected relative path README.md, got %s", relative)
	}
	if len(targets) != 1 || targets[0] != targetDocumentation {
		t.Fatalf("expected targets [%s], got %v", targetDocumentation, targets)
	}
}

func TestClassifyFileTargetsDocsMarkdown(t *testing.T) {
	root := filepath.Join("/tmp", "project", "scenarios", "demo")
	fullPath := filepath.Join(root, "docs", "overview.md")
	scenario, relative, targets := classifyFileTargets(fullPath)

	if scenario != "demo" {
		t.Fatalf("expected scenario demo, got %s", scenario)
	}
	expectedRel := filepath.ToSlash(filepath.Join("docs", "overview.md"))
	if relative != expectedRel {
		t.Fatalf("expected relative path %s, got %s", expectedRel, relative)
	}
	if len(targets) != 1 || targets[0] != targetDocumentation {
		t.Fatalf("expected targets [%s], got %v", targetDocumentation, targets)
	}
}

func TestResolveVrooliRootFromWorkingDirectory(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("VROOLI_SOURCE_ROOT", "")
	t.Setenv("APP_ROOT", "")

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	h := newRepoHarness(t)
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}
	contractData, err := os.ReadFile(filepath.Join(h.Root, ".vrooli", "repo-contract.json"))
	if err != nil {
		t.Fatalf("failed to read fixture contract: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, ".vrooli"), 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".vrooli", "repo-contract.json"), contractData, 0o644); err != nil {
		t.Fatalf("failed to write repo contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module fixture\n\ngo 1.24.0\n"), 0o644); err != nil {
		t.Fatalf("failed to write go.mod: %v", err)
	}
	for _, dir := range []string{"scenarios", "resources", "packages", "cmd", "internal", "templates"} {
		if err := os.MkdirAll(filepath.Join(repoRoot, dir), 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
	}
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "scenario-auditor")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "api", "rules"), 0o755); err != nil {
		t.Fatalf("failed to create rule directory structure: %v", err)
	}
	writeJSONFile(t, filepath.Join(scenarioRoot, ".vrooli", "service.json"), map[string]any{
		"service": map[string]any{"name": "scenario-auditor"},
	})

	workingDir := filepath.Join(scenarioRoot, "api")
	if err := os.MkdirAll(workingDir, 0o755); err != nil {
		t.Fatalf("failed to create working directory: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to capture working directory: %v", err)
	}

	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("failed to change working directory: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	clearRepoContext()
	ctx, err := repocontext.FromEnvOrCWD()
	if err != nil {
		t.Fatalf("resolveVrooliRoot returned error: %v", err)
	}
	if ctx.RepoRoot() != repoRoot {
		t.Fatalf("expected root %s, got %s", repoRoot, ctx.RepoRoot())
	}
}

func TestExternalViolationOutsideMappedPhysicalTargetIsDropped(t *testing.T) {
	physicalScenario := filepath.Join(t.TempDir(), "scenarios", "demo")
	outsideScenario := filepath.Join(t.TempDir(), "scenarios", "demo", "PRD.md")
	target := standardsScanTarget{Name: "demo", Path: physicalScenario}
	violation := StandardsViolation{FilePath: outsideScenario}

	if !shouldDropExternalViolationForTarget(target, violation) {
		t.Fatal("expected absolute external violation outside the physical target to be dropped")
	}

	inside := filepath.Join(physicalScenario, "PRD.md")
	if shouldDropExternalViolationForTarget(target, StandardsViolation{FilePath: inside}) {
		t.Fatal("did not expect physical target violation to be dropped")
	}
	if got := stableExternalViolationPath(target, inside); got != "PRD.md" {
		t.Fatalf("stableExternalViolationPath = %q, want PRD.md", got)
	}
}
