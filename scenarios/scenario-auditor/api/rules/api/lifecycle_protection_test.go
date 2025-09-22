//go:build ruletests
// +build ruletests

package api

import "testing"

func TestLifecycleProtectionDocCases(t *testing.T) {
	runDocTestsRuleStruct(t, "lifecycle_protection.go", "api/main.go", func(input string, path string) ([]Violation, error) {
		v := CheckLifecycleProtection([]byte(input), path, "demo-app")
		if v == nil {
			return nil, nil
		}
		return []Violation{*v}, nil
	})
}
