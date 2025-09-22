//go:build ruletests
// +build ruletests

package config

import "testing"

func TestDevelopLifecycleDocCases(t *testing.T) {
	runDocTestsViolations(t, "develop_steps.go", ".vrooli/service.json", CheckDevelopLifecycleSteps)
}
