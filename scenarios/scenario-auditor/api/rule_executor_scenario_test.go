//go:build rule_runtime_tests
// +build rule_runtime_tests

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuleExecutorPassesScenario(t *testing.T) {
	tempDir := t.TempDir()
	ruleSource := []byte(`package scenario

type Violation struct {
    Message string
}

func CheckScenario(content []byte, path string, scenario string) []Violation {
    if scenario == "" {
        return nil
    }
    return []Violation{
        {
            Message: scenario,
        },
    }
}
`)

	rulePath := filepath.Join(tempDir, "scenario_rule.go")
	if err := os.WriteFile(rulePath, ruleSource, 0o600); err != nil {
		t.Fatalf("failed to write rule file: %v", err)
	}

	info := RuleInfo{
		ID:       "scenario-pass",
		Category: "config",
		FilePath: rulePath,
	}

	exec, status := compileGoRule(&info)
	if !status.Valid {
		t.Fatalf("expected rule to compile, got error: %s", status.Error)
	}
	info.executor = exec

	violations, err := info.Check("sample", "test.json", "demo-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}
	if violations[0].Message != "demo-scenario" {
		t.Fatalf("expected violation message to include scenario, got %q", violations[0].Message)
	}
}
