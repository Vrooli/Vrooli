package executor

import "testing"

func TestClassifyFailure(t *testing.T) {
	cases := []struct {
		name           string
		stdout, stderr string
		want           string
	}{
		{"go test failure", "--- FAIL: TestX", "", ClassTestFailure},
		{"command not found", "", "bash: pnpm: command not found", ClassMissingDependency},
		{"node module missing", "Error: Cannot find module 'vitest'", "", ClassMissingDependency},
		{"pnpm no lockfile", "", "ERR_PNPM_NO_LOCKFILE  Cannot install", ClassMissingDependency},
		{"frozen lockfile drift", "", "ERR_PNPM_OUTDATED_LOCKFILE frozen-lockfile", ClassMissingDependency},
		{"go missing tool", "go: no such tool covdata", "", ClassMisconfiguration},
		{"no go test files", "no test files", "", ClassMisconfiguration},
		// Go's unquoted "cannot find module" stays a misconfiguration, not a node
		// dependency miss — the quote after "module" is what distinguishes node.
		{"go cannot find module", "go: cannot find module providing package x", "", ClassMisconfiguration},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := classifyFailure(c.stdout, c.stderr); got != c.want {
				t.Errorf("classifyFailure(%q,%q) = %q, want %q", c.stdout, c.stderr, got, c.want)
			}
		})
	}
}
