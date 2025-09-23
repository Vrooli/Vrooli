package structure

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	rules "scenario-auditor/rules"
)

/*
Rule: Test File Coverage
Description: Ensure every Go source file in the scenario has a companion *_test.go file
Reason: Guarantees core logic is exercised by automated tests, preventing untested regressions
Category: structure
Severity: medium
Standard: testing-standards-v1
Targets: structure

<test-case id="missing-test-file" should-fail="true" path="scenarios/demo">
  <description>Go source without matching test file</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/user_service.go"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>user_service_test.go</expected-message>
</test-case>

<test-case id="all-tested" should-fail="false" path="scenarios/demo">
  <description>Go source that already has a matching test file</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/user_service.go",
    "api/user_service_test.go",
    "cli/main.go",
    "cli/main_test.go"
  ]
}
  </input>
</test-case>

<test-case id="ignores-generated" should-fail="false" path="scenarios/demo">
  <description>Protocol buffer and mock outputs are exempt from coverage requirements</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/generated.pb.go",
    "api/mocks/user_mock.go"
  ]
}
  </input>
</test-case>

<test-case id="ignores-vendor" should-fail="false" path="scenarios/demo">
  <description>Vendor and testdata directories are excluded</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "vendor/github.com/pkg/lib/lib.go",
    "testdata/sample/app.go"
  ]
}
  </input>
</test-case>

<test-case id="nested-go-file" should-fail="true" path="scenarios/demo">
  <description>Deeply nested Go source must still have a companion test</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/lib/util/file.go"
  ]
}
  </input>
  <expected-violations>1</expected-violations>
  <expected-message>api/lib/util/file_test.go</expected-message>
</test-case>
*/

type testCoveragePayload struct {
	Scenario string   `json:"scenario"`
	Files    []string `json:"files"`
}

// CheckTestFileCoverage validates scenario structure for missing Go test files.
func CheckTestFileCoverage(content []byte, scenarioPath string, scenario string) ([]rules.Violation, error) {
	var payload testCoveragePayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return []rules.Violation{structureError("structure", fmt.Sprintf("Invalid structure payload: %v", err))}, nil
	}

	scenarioName := strings.TrimSpace(payload.Scenario)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(scenario)
	}

	if len(payload.Files) == 0 {
		return nil, nil
	}

	files := make([]string, 0, len(payload.Files))
	for _, f := range payload.Files {
		normalized := normalizePath(f)
		if normalized != "" {
			files = append(files, normalized)
		}
	}

	if len(files) == 0 {
		return nil, nil
	}

	sort.Strings(files)
	fileSet := make(map[string]struct{}, len(files))
	for _, f := range files {
		fileSet[f] = struct{}{}
	}

	var violations []rules.Violation
	for _, file := range files {
		if shouldSkipForCoverage(file) {
			continue
		}

		expectedTest := coverageMate(file)
		if _, ok := fileSet[expectedTest]; ok {
			continue
		}

		violations = append(violations, rules.Violation{
			Type:           "test_coverage",
			Severity:       "medium",
			Title:          "Missing Test File",
			Message:        fmt.Sprintf("Missing test file %s", filepath.ToSlash(expectedTest)),
			Description:    fmt.Sprintf("Missing test file %s for %s", filepath.ToSlash(expectedTest), filepath.ToSlash(file)),
			FilePath:       filepath.ToSlash(file),
			Recommendation: fmt.Sprintf("Create %s to cover %s", filepath.Base(expectedTest), filepath.Base(file)),
			Standard:       "testing-standards-v1",
		})
	}

	return violations, nil
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	trimmed = strings.TrimPrefix(trimmed, "./")
	return filepath.ToSlash(trimmed)
}

func shouldSkipForCoverage(path string) bool {
	if !strings.HasSuffix(path, ".go") {
		return true
	}
	if strings.HasSuffix(path, "_test.go") {
		return true
	}

	lowered := strings.ToLower(path)
	if strings.Contains(lowered, "/vendor/") || strings.Contains(lowered, "vendor/") {
		return true
	}
	if strings.Contains(lowered, "/testdata/") || strings.HasPrefix(lowered, "testdata/") {
		return true
	}
	if strings.Contains(lowered, "/mock/") || strings.Contains(lowered, "/mocks/") {
		return true
	}
	if strings.Contains(lowered, "/generated/") {
		return true
	}

	base := filepath.Base(lowered)
	switch {
	case strings.HasSuffix(base, ".pb.go"),
		strings.HasSuffix(base, "_pb.go"),
		strings.HasSuffix(base, "_pb2.go"),
		strings.HasSuffix(base, "_gen.go"),
		strings.HasSuffix(base, "_generated.go"),
		strings.Contains(base, "_gen.go"),
		strings.HasSuffix(base, "_mock.go"),
		strings.HasPrefix(base, "mock_"):
		return true
	}

	return false
}

func coverageMate(path string) string {
	withoutExt := strings.TrimSuffix(path, ".go")
	return withoutExt + "_test.go"
}

func structureError(path, message string) rules.Violation {
	return rules.Violation{
		Severity:       "medium",
		Title:          "Invalid scenario structure payload",
		Description:    message,
		FilePath:       path,
		Recommendation: "Ensure scenario structure data is available before running this rule",
	}
}
