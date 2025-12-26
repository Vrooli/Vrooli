//go:build ruletests
// +build ruletests

package api

import "testing"

func TestServerRunDocCases(t *testing.T) {
	runDocTestsViolations(t, "server_run.go", "api/main.go", CheckServerRun)
}
