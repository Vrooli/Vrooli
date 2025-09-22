package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildRuleBucketsRespectsDisabledStates(t *testing.T) {
	originalStore := ruleStateStore
	defer func() { ruleStateStore = originalStore }()

	ruleStateStore = &RuleStateStore{states: make(map[string]bool)}

	rules := map[string]RuleInfo{
		"enabled_rule": {
			ID:      "enabled_rule",
			Targets: []string{"api"},
			Enabled: true,
		},
		"disabled_rule": {
			ID:      "disabled_rule",
			Targets: []string{"api"},
			Enabled: true,
		},
		"metadata_disabled": {
			ID:      "metadata_disabled",
			Targets: []string{"api"},
			Enabled: false,
		},
	}

	if err := ruleStateStore.SetState("disabled_rule", false); err != nil {
		t.Fatalf("SetState returned error: %v", err)
	}

	buckets, active := buildRuleBuckets(rules, nil)

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

	_, targetedActive := buildRuleBuckets(rules, []string{"disabled_rule"})
	if len(targetedActive) != 0 {
		t.Fatalf("expected no active targeted rules when the requested rule is disabled, got %d", len(targetedActive))
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

func TestPerformStandardsCheckRunsStructureRules(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "project")
	scenariosDir := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenariosDir, 0o755); err != nil {
		t.Fatalf("failed to create scenarios directory: %v", err)
	}

	rulesDir := filepath.Join(root, "scenarios", "scenario-auditor", "rules", "structure")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("failed to create rules directory: %v", err)
	}

	sourceRulePath := filepath.Join("..", "rules", "structure", "required_layout.go")
	ruleContents, err := os.ReadFile(sourceRulePath)
	if err != nil {
		t.Fatalf("failed to load structure rule source: %v", err)
	}

	rulePath := filepath.Join(rulesDir, "required_layout.go")
	if err := os.WriteFile(rulePath, ruleContents, 0o644); err != nil {
		t.Fatalf("failed to write structure rule: %v", err)
	}

	scenarioPath := filepath.Join(scenariosDir, "demo")
	if err := os.MkdirAll(scenarioPath, 0o755); err != nil {
		t.Fatalf("failed to create scenario directory: %v", err)
	}

	apiPath := filepath.Join(scenarioPath, "api")
	if err := os.MkdirAll(apiPath, 0o755); err != nil {
		t.Fatalf("failed to create api directory: %v", err)
	}

	cliPath := filepath.Join(scenarioPath, "cli")
	if err := os.MkdirAll(cliPath, 0o755); err != nil {
		t.Fatalf("failed to create cli directory: %v", err)
	}

	testPhasesPath := filepath.Join(scenarioPath, "test", "phases")
	if err := os.MkdirAll(testPhasesPath, 0o755); err != nil {
		t.Fatalf("failed to create test phases directory: %v", err)
	}

	// Create all required files except the Makefile to trigger a single violation.
	requiredFiles := map[string]string{
		".vrooli/service.json":             "{}\n",
		"api/main.go":                      "package main\nfunc main() {}\n",
		"cli/install.sh":                   "#!/usr/bin/env bash\n",
		"cli/demo":                         "#!/usr/bin/env bash\n",
		"test/run-tests.sh":                "#!/usr/bin/env bash\n",
		"test/phases/test-unit.sh":         "#!/usr/bin/env bash\n",
		"test/phases/test-integration.sh":  "#!/usr/bin/env bash\n",
		"test/phases/test-structure.sh":    "#!/usr/bin/env bash\n",
		"test/phases/test-dependencies.sh": "#!/usr/bin/env bash\n",
		"test/phases/test-business.sh":     "#!/usr/bin/env bash\n",
		"test/phases/test-performance.sh":  "#!/usr/bin/env bash\n",
		"PRD.md":                           "# Product Requirements\n",
		"README.md":                        "# README\n",
	}

	for rel, contents := range requiredFiles {
		abs := filepath.Join(scenarioPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("failed to create directory for %s: %v", rel, err)
		}
		if err := os.WriteFile(abs, []byte(contents), 0o755); err != nil {
			t.Fatalf("failed to write %s: %v", rel, err)
		}
	}

	t.Setenv("VROOLI_ROOT", root)

	violations, _, err := performStandardsCheck(context.Background(), scenarioPath, "", nil, nil)
	if err != nil {
		t.Fatalf("performStandardsCheck returned error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation for missing Makefile, got %d", len(violations))
	}
	if violations[0].FilePath != "Makefile" {
		t.Fatalf("expected violation to reference Makefile, got %s", violations[0].FilePath)
	}

	makefilePath := filepath.Join(scenarioPath, "Makefile")
	if err := os.WriteFile(makefilePath, []byte("# demo makefile\n"), 0o644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}

	violations, _, err = performStandardsCheck(context.Background(), scenarioPath, "", nil, nil)
	if err != nil {
		t.Fatalf("performStandardsCheck returned error after adding Makefile: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected no violations after adding Makefile, got %d", len(violations))
	}
}
