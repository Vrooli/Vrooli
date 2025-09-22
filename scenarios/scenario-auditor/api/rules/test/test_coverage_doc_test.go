//go:build ruletests
// +build ruletests

package testrules

import "testing"

func TestTestCoverageDocCases(t *testing.T) {
	runDocTestsViolations(t, "test_coverage.go", "api/user.go", CheckTestFileCoverage)
}
