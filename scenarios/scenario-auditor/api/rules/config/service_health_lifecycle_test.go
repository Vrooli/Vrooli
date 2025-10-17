//go:build ruletests
// +build ruletests

package config

import "testing"

func TestServiceHealthLifecycleDocCases(t *testing.T) {
	runDocTestsViolations(t, "service_health_lifecycle.go", ".vrooli/service.json", CheckServiceHealthLifecycle)
}
