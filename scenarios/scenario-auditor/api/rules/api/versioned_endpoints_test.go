//go:build ruletests
// +build ruletests

package api

import "testing"

func TestVersionedEndpointsDocCases(t *testing.T) {
	runDocTestsViolations(t, "versioned_endpoints.go", "api/handlers.go", CheckVersionedEndpoints)
}
