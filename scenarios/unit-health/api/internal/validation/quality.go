package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// excessiveSnapshotThreshold is the per-file snapshot count above which a UI
// test file is flagged as snapshot-heavy.
const excessiveSnapshotThreshold = 5

// edgeCaseKeywords signal that a test exercises negative/boundary behavior. A
// workspace whose tests never mention any of these is flagged as positive-path
// only.
var edgeCaseKeywords = []string{
	"error", "invalid", "fail", "empty", "nil", "negative",
	"boundary", "missing", "not found", "notfound", "zero", "overflow",
	"timeout", "cancel", "edge", "panic", "malformed",
}

// goAssertionMethods are *testing.T/B failure methods that count as assertions.
var goAssertionMethods = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true,
	"Fail": true, "FailNow": true,
}

// goAssertionPackages are common assertion helper packages.
var goAssertionPackages = map[string]bool{
	"assert": true, "require": true, "is": true, "qt": true, "should": true,
}

// analyzeQuality runs the static test-quality checks: skipped/only tests,
// assertion-free tests, render-only and snapshot-heavy UI tests, edge-case
// coverage, and requirement tagging.
func analyzeQuality(scenario, scenarioRoot string, workspaces []Workspace, now string) []Finding {
	var findings []Finding
	for _, ws := range workspaces {
		switch ws.Language {
		case "go":
			findings = append(findings, analyzeGoQuality(scenario, ws, now)...)
		case "typescript":
			findings = append(findings, analyzeTSQuality(scenario, ws, now)...)
		}
	}
	findings = append(findings, analyzeRequirementTags(scenario, scenarioRoot, workspaces, now)...)
	return findings
}

func qualityFinding(scenario string, ws Workspace, code, file, message, evidence, expected, observed, why, remediation, now string) Finding {
	return Finding{
		ID:           code + "-" + ws.ID,
		Scenario:     scenario,
		WorkspaceID:  ws.ID,
		Language:     ws.Language,
		Code:         code,
		Category:     "quality",
		Severity:     codeSeverity[code],
		FilePath:     file,
		Message:      message,
		Evidence:     evidence,
		Expected:     expected,
		Observed:     observed,
		WhyItMatters: why,
		Remediation:  remediation,
		CreatedAt:    now,
	}
}

func analyzeGoQuality(scenario string, ws Workspace, now string) []Finding {
	var (
		testFuncs    int
		skipped      []string
		noAssertion  []string
		edgeKeywords bool
	)

	walkSourceFiles(ws.RootPath, func(path string) {
		if !isGoTestFile(path) {
			return
		}
		src := readFileString(path)
		if src == "" {
			return
		}
		if containsAny(strings.ToLower(src), edgeCaseKeywords) {
			edgeKeywords = true
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil || f == nil {
			return
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			// A receiver method named Test* is not a test entrypoint.
			if fn.Recv != nil {
				continue
			}
			testFuncs++
			hasSkip, hasAssert := inspectGoTestBody(fn.Body)
			if hasSkip {
				skipped = append(skipped, fn.Name.Name)
			}
			if !hasAssert {
				noAssertion = append(noAssertion, fn.Name.Name)
			}
		}
	})

	var findings []Finding
	if len(skipped) > 0 {
		sort.Strings(skipped)
		findings = append(findings, qualityFinding(scenario, ws, codeTestSkippedOrOnly, ws.RootPath,
			fmt.Sprintf("%d Go test(s) are skipped at runtime.", len(skipped)),
			"skipped: "+strings.Join(truncateList(skipped, 10), ", "),
			"Tests run rather than t.Skip unconditionally.",
			fmt.Sprintf("%d skipped test(s)", len(skipped)),
			"Skipped tests silently stop protecting the behavior they name.",
			"Remove the skip or gate it on a documented, temporary condition.",
			now))
	}
	if len(noAssertion) > 0 {
		sort.Strings(noAssertion)
		findings = append(findings, qualityFinding(scenario, ws, codeTestNoAssertion, ws.RootPath,
			fmt.Sprintf("%d Go test(s) make no observable assertion.", len(noAssertion)),
			"no-assertion: "+strings.Join(truncateList(noAssertion, 10), ", "),
			"Each test asserts an expectation (t.Error/Fatal, require/assert, or an explicit compare).",
			fmt.Sprintf("%d assertion-free test(s)", len(noAssertion)),
			"A test with no assertion only proves the code does not panic, not that it is correct.",
			"Add explicit assertions on the observable result of each test.",
			now))
	}
	if testFuncs > 0 && !edgeKeywords {
		findings = append(findings, qualityFinding(scenario, ws, codeTestMissingEdgeCases, ws.RootPath,
			"Go tests appear to cover only the positive path.",
			"no test names or bodies mention error/invalid/empty/boundary cases",
			"Tests for negative, boundary, and error inputs alongside the happy path.",
			"positive-path only",
			"Without negative and boundary cases, the most common real-world failures go untested.",
			"Add table cases for error inputs, empty/nil values, and boundaries.",
			now))
	}
	return findings
}

// inspectGoTestBody reports whether a test body contains a skip and whether it
// contains any assertion.
func inspectGoTestBody(body *ast.BlockStmt) (hasSkip, hasAssert bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		method := sel.Sel.Name
		if method == "Skip" || method == "Skipf" || method == "SkipNow" {
			hasSkip = true
		}
		if goAssertionMethods[method] {
			hasAssert = true
		}
		if ident, ok := sel.X.(*ast.Ident); ok && goAssertionPackages[ident.Name] {
			hasAssert = true
		}
		return true
	})
	return hasSkip, hasAssert
}

var (
	tsSkipOnlyRe  = regexp.MustCompile(`\b(describe|it|test)\s*\.\s*(skip|only)\b|\b(xit|xdescribe|fit|fdescribe)\b`)
	tsExpectRe    = regexp.MustCompile(`\bexpect\s*\(|\bassert\b`)
	tsRenderRe    = regexp.MustCompile(`\brender\s*\(`)
	tsSnapshotRe  = regexp.MustCompile(`\btoMatch(Inline)?Snapshot\s*\(`)
	tsTestBlockRe = regexp.MustCompile(`\b(it|test)\s*\(`)
)

func analyzeTSQuality(scenario string, ws Workspace, now string) []Finding {
	var (
		testFiles     int
		skipOnly      int
		renderOnly    []string
		noAssertion   []string
		snapshotHeavy []string
		edgeKeywords  bool
	)

	walkSourceFiles(ws.RootPath, func(path string) {
		if !isTSTestFile(path) {
			return
		}
		src := readFileString(path)
		if src == "" {
			return
		}
		testFiles++
		if containsAny(strings.ToLower(src), edgeCaseKeywords) {
			edgeKeywords = true
		}
		if tsSkipOnlyRe.MatchString(src) {
			skipOnly++
		}
		hasExpect := tsExpectRe.MatchString(src)
		hasTestBlock := tsTestBlockRe.MatchString(src)
		hasRender := tsRenderRe.MatchString(src)
		if hasTestBlock && !hasExpect {
			if hasRender {
				renderOnly = append(renderOnly, filepath.Base(path))
			} else {
				noAssertion = append(noAssertion, filepath.Base(path))
			}
		}
		if n := len(tsSnapshotRe.FindAllString(src, -1)); n > excessiveSnapshotThreshold {
			snapshotHeavy = append(snapshotHeavy, fmt.Sprintf("%s (%d)", filepath.Base(path), n))
		}
	})

	var findings []Finding
	if skipOnly > 0 {
		findings = append(findings, qualityFinding(scenario, ws, codeTestSkippedOrOnly, ws.RootPath,
			fmt.Sprintf("%d UI test file(s) use .skip/.only or x-prefixed blocks.", skipOnly),
			fmt.Sprintf("%d file(s) with describe/it.skip|only or xit/xdescribe", skipOnly),
			"No focused (.only) or skipped (.skip/xit) tests committed.",
			fmt.Sprintf("%d file(s)", skipOnly),
			".only silently disables every other test in the run; .skip stops protecting behavior.",
			"Remove .only/.skip and x-prefixed blocks before committing.",
			now))
	}
	if len(renderOnly) > 0 {
		sort.Strings(renderOnly)
		findings = append(findings, qualityFinding(scenario, ws, codeTestRenderOnly, ws.RootPath,
			fmt.Sprintf("%d UI test file(s) render a component but assert nothing.", len(renderOnly)),
			"render-only: "+strings.Join(truncateList(renderOnly, 10), ", "),
			"Each render is followed by expectations on the rendered output or behavior.",
			fmt.Sprintf("%d render-only file(s)", len(renderOnly)),
			"A render-only test only proves the component does not throw on mount, not that it works.",
			"Assert on visible text, roles, or interactions after rendering.",
			now))
	}
	if len(noAssertion) > 0 {
		sort.Strings(noAssertion)
		findings = append(findings, qualityFinding(scenario, ws, codeTestNoAssertion, ws.RootPath,
			fmt.Sprintf("%d UI test file(s) define tests with no expect().", len(noAssertion)),
			"no-assertion: "+strings.Join(truncateList(noAssertion, 10), ", "),
			"Each test calls expect() (or an assertion) on an observable result.",
			fmt.Sprintf("%d assertion-free file(s)", len(noAssertion)),
			"A test with no expect() passes regardless of behavior.",
			"Add expect() assertions to each test.",
			now))
	}
	if len(snapshotHeavy) > 0 {
		sort.Strings(snapshotHeavy)
		findings = append(findings, qualityFinding(scenario, ws, codeTestExcessiveSnapshots, ws.RootPath,
			fmt.Sprintf("%d UI test file(s) rely heavily on snapshots.", len(snapshotHeavy)),
			"snapshot-heavy: "+strings.Join(truncateList(snapshotHeavy, 10), ", "),
			fmt.Sprintf("Targeted assertions rather than more than %d snapshots per file.", excessiveSnapshotThreshold),
			"snapshot-heavy files",
			"Large snapshot suites are brittle and rubber-stamped on update, hiding real regressions.",
			"Replace broad snapshots with targeted assertions on the meaningful output.",
			now))
	}
	if testFiles > 0 && !edgeKeywords {
		findings = append(findings, qualityFinding(scenario, ws, codeTestMissingEdgeCases, ws.RootPath,
			"UI tests appear to cover only the positive path.",
			"no test references error/invalid/empty/boundary states",
			"Tests for error, empty, and boundary states alongside the happy path.",
			"positive-path only",
			"Untested error/empty states are the states users most often hit.",
			"Add tests for loading, empty, and error states.",
			now))
	}
	return findings
}

// analyzeRequirementTags emits one TEST_UNTAGGED_REQUIREMENT finding when the
// scenario declares requirement IDs but no test file references any of them.
func analyzeRequirementTags(scenario, scenarioRoot string, workspaces []Workspace, now string) []Finding {
	reqIDs := scenarioRequirementIDs(scenarioRoot)
	if len(reqIDs) == 0 {
		return nil
	}
	referenced := false
	for _, ws := range workspaces {
		walkSourceFiles(ws.RootPath, func(path string) {
			if referenced {
				return
			}
			if !isGoTestFile(path) && !isTSTestFile(path) {
				return
			}
			src := readFileString(path)
			for id := range reqIDs {
				if strings.Contains(src, id) {
					referenced = true
					return
				}
			}
		})
		if referenced {
			break
		}
	}
	if referenced {
		return nil
	}
	ids := make([]string, 0, len(reqIDs))
	for id := range reqIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return []Finding{{
		ID:           codeTestUntaggedRequirement,
		Scenario:     scenario,
		Code:         codeTestUntaggedRequirement,
		Category:     "quality",
		Severity:     codeSeverity[codeTestUntaggedRequirement],
		FilePath:     filepath.Join(scenarioRoot, "requirements"),
		Message:      fmt.Sprintf("The scenario declares %d requirement ID(s) but no test references any of them.", len(ids)),
		Evidence:     "requirement ids: " + strings.Join(truncateList(ids, 10), ", "),
		Expected:     "Tests reference the requirement IDs they validate (e.g. in a test name or comment).",
		Observed:     "no requirement-tagged tests",
		WhyItMatters: "Untagged requirements break traceability: there is no proof a requirement is actually tested.",
		Remediation:  "Reference the relevant REQ id in the validating test's name or a comment.",
		CreatedAt:    now,
	}}
}

var requirementIDRe = regexp.MustCompile(`\bREQ-[A-Za-z0-9-]+\b`)

// scenarioRequirementIDs extracts requirement IDs declared under requirements/.
func scenarioRequirementIDs(scenarioRoot string) map[string]struct{} {
	ids := map[string]struct{}{}
	reqDir := filepath.Join(scenarioRoot, "requirements")
	if scenarioRoot == "" {
		return ids
	}
	walkSourceFiles(reqDir, func(path string) {
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" && ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return
		}
		for _, m := range requirementIDRe.FindAllString(readFileString(path), -1) {
			ids[m] = struct{}{}
		}
	})
	return ids
}

func containsAny(haystack string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

func truncateList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	out := append([]string{}, items[:max]...)
	return append(out, fmt.Sprintf("… (+%d more)", len(items)-max))
}
