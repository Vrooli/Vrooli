//go:build ruletests
// +build ruletests

package api

import "testing"

func TestResourceCleanupDocCases(t *testing.T) {
	runDocTestsViolations(t, "resource_cleanup.go", "api/handlers.go", func(content []byte, path string) []Violation {
		t.Logf("cases run")
		t.Logf("input=%q", string(content))
		v := CheckResourceCleanup(content, path)
		t.Logf("case path=%s violations=%d", path, len(v))
		return v
	})
}
