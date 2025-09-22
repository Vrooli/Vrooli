//go:build ruletests
// +build ruletests

package config

import "testing"

func TestEnvValidationDocCases(t *testing.T) {
	runDocTestsViolationsErr(t, "env_validation.go", "api/main.go", func(content []byte, path string) ([]Violation, error) {
		rule := &EnvValidationRule{}
		return rule.Check(string(content), path)
	})
}
