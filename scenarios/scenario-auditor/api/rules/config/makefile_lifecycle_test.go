//go:build ruletests
// +build ruletests

package config

import "testing"

func TestMakefileLifecycleDocCases(t *testing.T) {
	runDocTestsCustom(t, "makefile_lifecycle.go", "Makefile", func(input string, path string) ([]MakefileLifecycleViolation, error) {
		return CheckMakefileLifecycle(input, path)
	}, func(v MakefileLifecycleViolation) string {
		return v.Message
	})
}
