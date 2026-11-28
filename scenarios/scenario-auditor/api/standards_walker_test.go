package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	rulespkg "scenario-auditor/rules"
)

func TestWalkStandardsSourceFilesSharesFilteringAndMetadata(t *testing.T) {
	scenarioRoot := filepath.Join(t.TempDir(), "scenarios", "demo")
	for _, path := range []string{
		filepath.Join(scenarioRoot, "api", "main.go"),
		filepath.Join(scenarioRoot, "ui", "App.tsx"),
		filepath.Join(scenarioRoot, ".git", "ignored.go"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte("package sample\n"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	var visited []standardsSourceFile
	err := walkStandardsSourceFiles(context.Background(), scenarioRoot, false, true, func(file standardsSourceFile) error {
		visited = append(visited, file)
		return nil
	})
	if err != nil {
		t.Fatalf("walk source files: %v", err)
	}
	if len(visited) != 2 {
		t.Fatalf("visited %d files, want 2: %+v", len(visited), visited)
	}
	want := map[string][]string{
		"api/main.go": {"api", "main_go"},
		"ui/App.tsx":  {"ui"},
	}
	for _, file := range visited {
		if file.scenarioName != "demo" {
			t.Errorf("scenario for %s = %q, want demo", file.path, file.scenarioName)
		}
		if got := file.targets; !reflect.DeepEqual(got, want[file.scenarioRelative]) {
			t.Errorf("targets for %s = %v, want %v", file.scenarioRelative, got, want[file.scenarioRelative])
		}
		if file.content != "package sample\n" {
			t.Errorf("content for %s = %q", file.scenarioRelative, file.content)
		}
		delete(want, file.scenarioRelative)
	}
	if len(want) != 0 {
		t.Errorf("missing eligible files: %v", want)
	}
}

func TestWalkStandardsSourceFilesKeepsCallerWalkErrorPolicy(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), "missing")
	visit := func(standardsSourceFile) error { return nil }

	if err := walkStandardsSourceFiles(context.Background(), missingRoot, true, false, visit); err != nil {
		t.Fatalf("ignored walk error = %v, want nil", err)
	}
	err := walkStandardsSourceFiles(context.Background(), missingRoot, false, true, visit)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("propagated walk error = %v, want not-exist", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	err = walkStandardsSourceFiles(canceled, t.TempDir(), true, false, visit)
	if !errors.Is(err, errStandardsScanCancelled) {
		t.Fatalf("canceled walk error = %v, want %v", err, errStandardsScanCancelled)
	}
}

func TestEvaluateRuleOnScenarioKeepsPerRuleTargetsAndDisabledRuleBehavior(t *testing.T) {
	h := newRepoHarness(t)
	h.UseRepoContext(t)
	scenarioRoot := h.WriteScenario(t, "demo")
	for path, content := range map[string]string{
		filepath.Join(scenarioRoot, "api", "main.go"): "package sample\n",
		filepath.Join(scenarioRoot, "ui", "App.tsx"):  "export const App = () => null;\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create parent for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	var calls []struct{ path, scenario string }
	rule := RuleInfo{Rule: rulespkg.Rule{
		ID:      "fixture_disabled_rule",
		Name:    "Fixture disabled rule",
		Enabled: false,
	}, Targets: []string{targetAPI}}.WithExecutor(ruleExecutorFunc(func(_ string, path, scenario string) ([]rulespkg.Violation, error) {
		calls = append(calls, struct{ path, scenario string }{path: path, scenario: scenario})
		return []rulespkg.Violation{{Message: "fixture finding"}}, nil
	}), RuleImplementationStatus{Valid: true})

	violations, filesScanned, targets, err := evaluateRuleOnScenario(rule, "demo")
	if err != nil {
		t.Fatalf("evaluate rule: %v", err)
	}
	if filesScanned != 1 {
		t.Fatalf("files scanned = %d, want 1 matching API file", filesScanned)
	}
	if !reflect.DeepEqual(targets, []string{targetAPI}) {
		t.Fatalf("targets = %v, want [%s]", targets, targetAPI)
	}
	if len(calls) != 1 || calls[0].path != filepath.Join(scenarioRoot, "api", "main.go") || calls[0].scenario != "demo" {
		t.Fatalf("rule calls = %+v, want API path and caller scenario", calls)
	}
	if len(violations) != 1 || violations[0].FilePath != "api/main.go" {
		t.Fatalf("violations = %+v, want one relative API finding", violations)
	}
}

type ruleExecutorFunc func(content, pathHint, scenario string) ([]rulespkg.Violation, error)

func (f ruleExecutorFunc) Execute(content, pathHint, scenario string) ([]rulespkg.Violation, error) {
	return f(content, pathHint, scenario)
}
