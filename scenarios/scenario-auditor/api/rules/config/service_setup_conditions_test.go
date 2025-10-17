//go:build ruletests
// +build ruletests

package config

import "testing"

func TestServiceSetupConditionsDocCases(t *testing.T) {
	runDocTestsViolations(t, "service_setup_conditions.go", ".vrooli/service.json", CheckServiceSetupConditions)
}
