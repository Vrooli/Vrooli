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

func TestPreflightSkipsTemplateLocalTools(t *testing.T) {
	input := `package main

func main() {
	runTool()
}
`
	if violation := CheckPreflight([]byte(input), "tools/temporal-model/main.go", "demo-app"); violation != nil {
		t.Fatalf("expected template-local tool main to be skipped, got %+v", violation)
	}
}
