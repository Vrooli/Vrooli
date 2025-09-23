//go:build ruletests
// +build ruletests

package structure

import "testing"

func TestTestCoverageDocCases(t *testing.T) {
	runDocTests(t, "test_coverage.go", "scenarios/demo", func(input, path, scenario string) ([]Violation, error) {
		return CheckTestFileCoverage([]byte(input), path, scenario)
	})
}
