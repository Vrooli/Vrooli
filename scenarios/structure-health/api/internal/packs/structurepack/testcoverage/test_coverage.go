package testcoverage

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	auditrules "structure-health/internal/packs/auditrules"
)

/*
Rule: Test Package Coverage
Description: Ensure every Go package (directory) with source files has at least one *_test.go file
Reason: Guarantees core logic is exercised by automated tests, preventing untested regressions
Category: structure
Severity: medium
Standard: testing-standards-v1
Targets: structure

Note: main.go files are always skipped - entry points are better validated via integration/e2e tests.
Other exempt files: doc.go, generated code (.pb.go, _gen.go), vendor/, testdata/, mocks/.

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
  <expected-message>Missing test file in directory api/</expected-message>
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
  <expected-message>Missing test file in directory api/lib/util/</expected-message>
</test-case>

<test-case id="multiple-sources-one-test" should-fail="false" path="scenarios/demo">
  <description>Multiple Go sources covered by single test file</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/a.go",
    "api/b.go",
    "api/a_test.go"
  ]
}
  </input>
</test-case>

<test-case id="cli-main-skipped" should-fail="false" path="scenarios/roi-fit-analysis" scenario="roi-fit-analysis">
  <description>CLI entrypoint main.go is always skipped - entry points are validated via integration/e2e tests</description>
  <input language="json">
{
  "scenario": "roi-fit-analysis",
  "files": [
    "cli/main.go"
  ]
}
  </input>
</test-case>

<test-case id="constants-only-skip" should-fail="false" path="scenarios/demo" scenario="demo">
  <description>Pure constants files remain exempt from coverage requirements</description>
  <input language="json">
{
  "scenario": "demo",
  "files": [
    "api/internal/server/metadata/constants.go"
  ]
}
  </input>
</test-case>
*/

type testCoveragePayload struct {
	Scenario string   `json:"scenario"`
	Files    []string `json:"files"`
}

// CheckTestFileCoverage validates scenario structure for missing Go test files.
func CheckTestFileCoverage(content []byte, scenarioPath string, scenario string) ([]auditrules.Violation, error) {
	var payload testCoveragePayload
	if err := json.Unmarshal(content, &payload); err != nil {
		return []auditrules.Violation{structureError("structure", fmt.Sprintf("Invalid structure payload: %v", err))}, nil
	}

	scenarioName := strings.TrimSpace(payload.Scenario)
	if scenarioName == "" {
		scenarioName = strings.TrimSpace(scenario)
	}
	if scenarioName == "" {
		scenarioName = "(unknown scenario)"
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

	dirMap := make(map[string][]string)
	for _, file := range files {
		dir := filepath.Dir(file)
		if dir == "." {
			dir = ""
		}
		dirMap[dir] = append(dirMap[dir], file)
	}

	dirs := make([]string, 0, len(dirMap))
	for dir := range dirMap {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var violations []auditrules.Violation
	for _, dir := range dirs {
		filesInDir := dirMap[dir]
		hasSource := false
		hasTest := false
		var sourceFiles []string
		for _, file := range filesInDir {
			// Check if this is a test file first
			if strings.HasSuffix(file, "_test.go") {
				hasTest = true
				continue
			}

			// Then check if we should skip this file (generated, vendor, etc.)
			if shouldSkipForCoverage(file, scenarioPath) {
				continue
			}

			// It's a regular source file
			hasSource = true
			sourceFiles = append(sourceFiles, file)
		}
		if hasSource && !hasTest {
			dirDisplay := dir
			if dirDisplay == "" {
				dirDisplay = "."
			} else if !strings.HasSuffix(dirDisplay, "/") {
				dirDisplay = dirDisplay + "/"
			}

			violationPath := dir
			if violationPath == "" || violationPath == "." {
				violationPath = scenarioName
			} else if scenarioName != "(unknown scenario)" {
				violationPath = fmt.Sprintf("%s/%s", scenarioName, strings.TrimPrefix(violationPath, "./"))
			}
			violationPath = strings.TrimSuffix(violationPath, "//")
			violations = append(violations, auditrules.Violation{
				Type:           "test_coverage",
				Severity:       "medium",
				Title:          "Missing Test File",
				Message:        fmt.Sprintf("Missing test file in directory %s", dirDisplay),
				Description:    fmt.Sprintf("Scenario %s is missing a test file in directory %s for files: %s", scenarioName, dirDisplay, summarizeSources(sourceFiles, 3)),
				FilePath:       violationPath,
				Recommendation: fmt.Sprintf("Create at least one *_test.go file in %s to cover the source files", dirDisplay),
				Standard:       "testing-standards-v1",
			})
		}
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

func shouldSkipForCoverage(path, scenarioRoot string) bool {
	if !strings.HasSuffix(path, ".go") {
		return true
	}

	lowered := strings.ToLower(path)
	if isTopLevelTestHarness(lowered) {
		return true
	}
	if strings.Contains(lowered, "/testkit/") {
		return true
	}
	if strings.Contains(lowered, "/vendor/") || strings.Contains(lowered, "vendor/") {
		return true
	}
	if strings.Contains(lowered, "/testdata/") || strings.HasPrefix(lowered, "testdata/") {
		return true
	}
	if hasPathSegment(lowered, "mock", "mocks") {
		return true
	}
	if hasPathSegment(lowered, "generated") {
		return true
	}

	absPath, hasFile := absoluteScenarioPath(scenarioRoot, path)
	base := filepath.Base(lowered)

	switch base {
	case "doc.go":
		return true
	case "main.go":
		// Entry points are better validated via integration/e2e tests
		return true
	case "types.go", "constants.go", "interfaces.go":
		if hasFile {
			return !fileHasExecutableLogic(absPath)
		}
		return false
	}

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

	if hasFile && !fileHasExecutableLogic(absPath) {
		return true
	}

	return false
}

func isTopLevelTestHarness(path string) bool {
	return strings.HasPrefix(path, "test/") || strings.HasPrefix(path, "tests/")
}

func summarizeSources(files []string, limit int) string {
	if limit <= 0 || len(files) <= limit {
		return strings.Join(files, ", ")
	}
	clamped := limit
	if clamped > len(files) {
		clamped = len(files)
	}
	prefix := strings.Join(files[:clamped], ", ")
	return fmt.Sprintf("%s, ... (+%d more)", prefix, len(files)-clamped)
}

func hasPathSegment(path string, names ...string) bool {
	if len(names) == 0 {
		return false
	}
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return false
	}
	nameSet := make(map[string]struct{}, len(names))
	for _, name := range names {
		nameSet[name] = struct{}{}
	}
	// Ignore the final segment (file name) when determining directory segments.
	for i := 0; i < len(segments)-1; i++ {
		segment := segments[i]
		if segment == "" {
			continue
		}
		if _, ok := nameSet[segment]; ok {
			return true
		}
	}
	return false
}

func absoluteScenarioPath(root, rel string) (string, bool) {
	if strings.TrimSpace(root) == "" {
		return "", false
	}
	relNative := filepath.FromSlash(rel)
	abs := relNative
	if !filepath.IsAbs(relNative) {
		abs = filepath.Join(root, relNative)
	}
	candidates := []string{filepath.Clean(abs)}
	if !filepath.IsAbs(candidates[0]) {
		prefixes := []string{"..", "../..", "../../..", "../../../.."}
		for _, prefix := range prefixes {
			candidate := filepath.Join(prefix, root, relNative)
			candidate = filepath.Clean(candidate)
			if candidate == candidates[0] {
				continue
			}
			candidates = append(candidates, candidate)
		}
		trimmedRoot := strings.TrimPrefix(root, "scenarios/")
		if trimmedRoot != root {
			for _, prefix := range prefixes {
				candidate := filepath.Join(prefix, trimmedRoot, relNative)
				candidate = filepath.Clean(candidate)
				candidates = append(candidates, candidate)
			}
		}
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate, true
		}
	}
	return candidates[0], false
}

func fileHasExecutableLogic(path string) bool {
	src, err := os.ReadFile(path)
	if err != nil {
		return !os.IsNotExist(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.SkipObjectResolution)
	if err != nil {
		return true
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Body != nil && len(d.Body.List) > 0 {
				return true
			}
		case *ast.GenDecl:
			switch d.Tok {
			case token.VAR:
				return true
			case token.CONST:
				continue
			case token.TYPE:
				if typeDeclHasBehaviour(d) {
					return true
				}
				continue
			default:
				return true
			}
		default:
			return true
		}
	}
	return false
}

func typeDeclHasBehaviour(decl *ast.GenDecl) bool {
	for _, spec := range decl.Specs {
		typeSpec, ok := spec.(*ast.TypeSpec)
		if !ok {
			return true
		}
		switch typeSpec.Type.(type) {
		case *ast.StructType, *ast.InterfaceType, *ast.FuncType, *ast.MapType, *ast.ChanType:
			return true
		}
	}
	return false
}

func structureError(path, message string) auditrules.Violation {
	return auditrules.Violation{
		Severity:       "medium",
		Title:          "Invalid scenario structure payload",
		Description:    message,
		FilePath:       path,
		Recommendation: "Ensure scenario structure data is available before running this rule",
	}
}
