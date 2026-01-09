//go:build ruletests
// +build ruletests

package api

import "testing"

func TestPreflightDocCases(t *testing.T) {
	runDocTestsRuleStruct(t, "preflight.go", "api/main.go", func(input string, path string) ([]Violation, error) {
		v := CheckPreflight([]byte(input), path, "demo-app")
		if v == nil {
			return nil, nil
		}
		return []Violation{*v}, nil
	})
}
