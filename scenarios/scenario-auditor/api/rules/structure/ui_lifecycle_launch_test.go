//go:build ruletests
// +build ruletests

package structure

import (
	"testing"

	rules "scenario-auditor/rules"
)

func TestUISharedLifecycleLaunchDocCases(t *testing.T) {
	runDocTests(t, "ui_lifecycle_launch.go", "scenario", func(input string, path string, scenario string) ([]rules.Violation, error) {
		return CheckUISharedLifecycleLaunch([]byte(input), path, scenario)
	})
}
