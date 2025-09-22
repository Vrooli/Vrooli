//go:build ruletests
// +build ruletests

package api

import "testing"

func TestSecurityHeadersDocCases(t *testing.T) {
	rule := &SecurityHeadersRule{}
	runDocTestsRuleStruct(t, "security_headers.go", "api/handlers.go", func(input string, path string) ([]Violation, error) {
		return rule.Check(input, path)
	})
}
