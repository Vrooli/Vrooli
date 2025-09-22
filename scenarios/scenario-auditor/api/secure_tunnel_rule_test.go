package main

import (
	"path/filepath"
	"testing"
)

func TestSecureTunnelRuleCases(t *testing.T) {
	rule := loadSecureTunnelRule(t)
	runner := NewTestRunner()

	results, err := runner.RunAllTests(rule.ID, rule)
	if err != nil {
		t.Fatalf("RunAllTests returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("expected embedded test cases, got zero")
	}

	for _, result := range results {
		shouldFail := result.TestCase.ShouldFail
		if shouldFail && !result.Passed {
			continue
		}
		if !shouldFail && result.Passed {
			continue
		}
		t.Fatalf("test %s: expected pass=%v, got %v (violations=%d, error=%s)", result.TestCase.ID, !shouldFail, result.Passed, len(result.ActualViolations), result.Error)
	}
}

func loadSecureTunnelRule(t *testing.T) RuleInfo {
	t.Helper()
	path := filepath.Join("..", "rules", "ui", "secure_tunnel.go")
	info := RuleInfo{
		ID:       "secure_tunnel",
		FilePath: path,
		Category: "ui",
	}

	exec, status := compileGoRule(&info)
	if !status.Valid {
		t.Fatalf("expected rule to compile, got error: %s", status.Error)
	}
	info.executor = exec
	return info
}
