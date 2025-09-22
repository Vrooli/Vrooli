//go:build ruletests
// +build ruletests

package config

import "testing"

func TestServiceTestStepsDocCases(t *testing.T) {
	runDocTestsViolations(t, "service_test_steps.go", ".vrooli/service.json", CheckLifecycleTestSteps)
}
