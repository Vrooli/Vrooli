//go:build ruletests
// +build ruletests

package api

import "testing"

func TestHealthCheckDocCases(t *testing.T) {
	runDocTestsViolations(t, "health_check.go", "api/main.go", CheckHealthCheckImplementation)
}
