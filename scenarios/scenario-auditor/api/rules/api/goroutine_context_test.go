//go:build ruletests
// +build ruletests

package api

import "testing"

func TestGoroutineContextDocCases(t *testing.T) {
	runDocTestsViolations(t, "goroutine_context.go", "api/main.go", CheckGoroutineContext)
}
