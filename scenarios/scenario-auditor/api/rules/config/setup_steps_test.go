//go:build ruletests
// +build ruletests

package config

import "testing"

func TestSetupStepsDocCases(t *testing.T) {
	runDocTestsViolations(t, "setup_steps.go", "scenarios/example/.vrooli/service.json", CheckSetupStepsConfiguration)
}
