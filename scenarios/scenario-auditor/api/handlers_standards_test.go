package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	re "scenario-auditor/internal/ruleengine"
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

	rulesDir := filepath.Join(root, "scenarios", "scenario-auditor", "api", "rules", "structure")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("failed to create rules directory: %v", err)
	}

	sourceRulePath := filepath.Join("rules", "structure", "required_layout.go")
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
	beforeCount := 0
	for _, v := range violations {
		if v.FilePath == "Makefile" {
			beforeCount++
		}
	}
	if beforeCount == 0 {
		t.Fatalf("expected violations referencing Makefile, got %+v", violations)
	}

	makefilePath := filepath.Join(scenarioPath, "Makefile")
	makefileTemplate := "fmt:\n\t@$(MAKE) fmt-go\n\t@$(MAKE) fmt-ui\n\nfmt-go:\n\t@if [ -d api ] && find api -name \"*.go\" | head -1 | grep -q .; then \\\n\t\techo \"Formatting Go code...\"; \\\n\t\tif command -v gofumpt >/dev/null 2>&1; then \\\n\t\t\tcd api && gofumpt -w .; \\\n\t\telif command -v gofmt >/dev/null 2>&1; then \\\n\t\t\tcd api && gofmt -w .; \\\n\t\tfi; \\\n\t\techo \"$(GREEN)✓ Go code formatted$(RESET)\"; \\\n\tfi\n\nfmt-ui:\n\t@echo \"Formatting UI assets...\"\n\nlint:\n\t@$(MAKE) lint-go\n\t@$(MAKE) lint-ui\n\nlint-go:\n\t@if [ -d api ] && find api -name \"*.go\" | head -1 | grep -q .; then \\\n\t\techo \"Linting Go code...\"; \\\n\t\tif command -v golangci-lint >/dev/null 2>&1; then \\\n\t\t\tcd api && golangci-lint run; \\\n\t\telse \\\n\t\t\tcd api && go vet ./...; \\\n\t\tfi; \\\n\t\techo \"$(GREEN)✓ Go code linted$(RESET)\"; \\\n\tfi\n\nlint-ui:\n\t@echo \"Linting UI code...\"\n\ncheck:\n\t@$(MAKE) fmt\n\t@$(MAKE) lint\n\t@$(MAKE) test\n"
	if err := os.WriteFile(makefilePath, []byte(makefileTemplate), 0o644); err != nil {
		t.Fatalf("failed to write Makefile: %v", err)
	}
}

func TestResolveVrooliRootFromWorkingDirectory(t *testing.T) {
	t.Setenv("VROOLI_ROOT", "")
	t.Setenv("APP_ROOT", "")

	tmp := t.TempDir()
	repoRoot := filepath.Join(tmp, "repo")
	scenarioRoot := filepath.Join(repoRoot, "scenarios", "scenario-auditor")
	if err := os.MkdirAll(filepath.Join(scenarioRoot, "api", "rules"), 0o755); err != nil {
		t.Fatalf("failed to create rule directory structure: %v", err)
	}

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

	root, err := re.DiscoverRepoRoot()
	if err != nil {
		t.Fatalf("resolveVrooliRoot returned error: %v", err)
	}
	if root != repoRoot {
		t.Fatalf("expected root %s, got %s", repoRoot, root)
	}
}
