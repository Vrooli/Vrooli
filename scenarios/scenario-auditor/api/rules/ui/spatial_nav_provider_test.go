//go:build ruletests
// +build ruletests

package ui

import "testing"

func TestSpatialNavProviderDocCases(t *testing.T) {
	runDocTestsViolations(t, "spatial_nav_provider.go", "ui/src/main.tsx", CheckSpatialNavProvider)
}
