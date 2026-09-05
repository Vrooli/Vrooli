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

// testutil Require* helpers conventionally receive *testing.T and fail it on
// an unmet expectation. Recognising that shared test seam avoids treating a
// tested HTTP response as an assertion-free test while remaining narrower than
// accepting arbitrary helper calls.
func isTestutilRequireCall(sel *ast.SelectorExpr) bool {
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == "testutil" && strings.HasPrefix(sel.Sel.Name, "Require")
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
		testFuncs   int
		skipped     []string
		noAssertion []string
		edgeCovered bool
	)

	// Parse every test file once. Pass 1 collects the names of all functions
	// (test entrypoints and local helpers) whose body contains a direct
	// assertion, so a test that asserts only through a local helper such as
	// assertEqual(t, ...) is not falsely flagged as assertion-free (B4).
	type parsedFile struct {
		path string
		file *ast.File
	}
	var files []parsedFile
	assertingFuncs := map[string]bool{}
	walkSourceFiles(ws.RootPath, func(path string) {
		if !isGoTestFile(path) {
			return
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil || f == nil {
			return
		}
		files = append(files, parsedFile{path: path, file: f})
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			if goFuncBodyAsserts(fn.Body) {
				assertingFuncs[fn.Name.Name] = true
			}
		}
	})

	// Pass 2 evaluates each test entrypoint with helper-call resolution.
	for _, pf := range files {
		for _, decl := range pf.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Recv != nil || !strings.HasPrefix(fn.Name.Name, "Test") {
				continue
			}
			// TestMain is Go's package lifecycle hook, not a test case. It owns
			// setup/teardown and invokes m.Run(), so requiring an assertion here
			// would reward meaningless checks rather than test quality.
			if fn.Name.Name == "TestMain" {
				continue
			}
			testFuncs++
			hasSkip, hasAssert := inspectGoTestBody(fn.Body, assertingFuncs)
			if hasSkip {
				skipped = append(skipped, fn.Name.Name)
			}
			if !hasAssert {
				noAssertion = append(noAssertion, fn.Name.Name)
			}
			if goFuncCoversEdge(fn) {
				edgeCovered = true
			}
		}
	}

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
	if testFuncs > 0 && !edgeCovered {
		findings = append(findings, qualityFinding(scenario, ws, codeTestMissingEdgeCases, ws.RootPath,
			"No Go test asserts on an error or boundary case; the suite appears positive-path only.",
			"no test name nor error/boundary assertion (assert.Error/require.NoError/Nil/Empty/Panics or a wantErr table field) found",
			"At least one test for negative, boundary, and error inputs alongside the happy path.",
			"positive-path only",
			"Without negative and boundary cases, the most common real-world failures go untested.",
			"Add table cases for error inputs, empty/nil values, and boundaries, and assert on them (e.g. require.Error).",
			now))
	}
	return findings
}

// inspectGoTestBody reports whether a test body contains a skip and whether it
// asserts — directly (t.Error/Fatal, assert/require) or through a local helper
// function whose own body asserts (assertingFuncs, one-level resolution).
func inspectGoTestBody(body *ast.BlockStmt, assertingFuncs map[string]bool) (hasSkip, hasAssert bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			method := fun.Sel.Name
			if method == "Skip" || method == "Skipf" || method == "SkipNow" {
				hasSkip = true
			}
			if goAssertionMethods[method] {
				hasAssert = true
			}
			if ident, ok := fun.X.(*ast.Ident); ok && goAssertionPackages[ident.Name] {
				hasAssert = true
			}
			if isTestutilRequireCall(fun) {
				hasAssert = true
			}
		case *ast.Ident:
			// A call to a local helper (assertEqual(t, …)) whose body asserts.
			if assertingFuncs[fun.Name] {
				hasAssert = true
			}
		}
		return true
	})
	return hasSkip, hasAssert
}

// goFuncBodyAsserts reports whether a function body contains a direct assertion
// (a *testing.T failure method or an assert/require package call). It does not
// resolve helper calls — it is the base case used to build the helper set.
func goFuncBodyAsserts(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if goAssertionMethods[sel.Sel.Name] {
				found = true
			}
			if id, ok := sel.X.(*ast.Ident); ok && goAssertionPackages[id.Name] {
				found = true
			}
			if isTestutilRequireCall(sel) {
				found = true
			}
		}
		return true
	})
	return found
}

// edgeAssertionMethods are assertion calls that demonstrate an error/boundary
// case is being exercised (B3 — scoped per test function via AST, not whole-file
// keyword matching).
var edgeAssertionMethods = map[string]bool{
	"Error": true, "ErrorIs": true, "ErrorAs": true, "ErrorContains": true,
	"NoError": true, "Nil": true, "NotNil": true, "Empty": true, "NotEmpty": true,
	"Panics": true, "NotPanics": true, "Zero": true,
}

var wantErrFieldRe = regexp.MustCompile(`(?i)^(want|expect|expected)(ed)?err(or)?$`)

// goFuncCoversEdge reports whether a test function exercises an error or
// boundary case: its name signals it, it asserts via an error/boundary helper,
// or it drives a table with a wantErr-style field. This is a real per-function
// signal, not the old "any error/nil keyword anywhere in the file" heuristic.
func goFuncCoversEdge(fn *ast.FuncDecl) bool {
	if nameSignalsEdge(fn.Name.Name) {
		return true
	}
	covers := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if edgeAssertionMethods[x.Sel.Name] {
				covers = true
			}
		case *ast.Ident:
			if wantErrFieldRe.MatchString(x.Name) {
				covers = true
			}
		}
		return !covers
	})
	return covers
}

// nameSignalsEdge reports whether a test name embeds an edge-case keyword
// (TestParseInvalidInput, TestEmptyList, …).
func nameSignalsEdge(name string) bool {
	lower := strings.ToLower(name)
	for _, kw := range edgeCaseKeywords {
		if strings.Contains(lower, strings.ReplaceAll(kw, " ", "")) {
			return true
		}
	}
	return false
}

var (
	tsSkipOnlyRe   = regexp.MustCompile(`\b(describe|it|test)\s*\.\s*(skip|only)\b|\b(xit|xdescribe|fit|fdescribe)\s*\(`)
	tsExpectRe     = regexp.MustCompile(`\bexpect\s*\(|\bassert\b`)
	tsRenderRe     = regexp.MustCompile(`\brender\s*\(`)
	tsSnapshotRe   = regexp.MustCompile(`\btoMatch(Inline)?Snapshot\s*\(`)
	tsTestBlockRe  = regexp.MustCompile(`\b(it|test)\s*\(`)
	tsTestTitleRe  = regexp.MustCompile("\\b(?:it|test)\\s*\\(\\s*[`'\"]([^`'\"]*)")
	tsLineComment  = regexp.MustCompile(`//[^\n]*`)
	tsBlockComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
)

func analyzeTSQuality(scenario string, ws Workspace, now string) []Finding {
	var (
		testFiles     int
		skipOnly      int
		renderOnly    []string
		noAssertion   []string
		snapshotHeavy []string
		edgeCovered   bool
	)

	walkSourceFiles(ws.RootPath, func(path string) {
		if !isTSTestFile(path) {
			return
		}
		raw := readFileString(path)
		if raw == "" {
			return
		}
		testFiles++
		// Strip comments so an `expect`/`error` mentioned only in a comment is
		// not counted as a real assertion or edge case (B4).
		src := stripTSComments(raw)
		if tsSkipOnlyRe.MatchString(src) {
			skipOnly++
		}

		// Per-test-block analysis: split on it()/test() boundaries and judge
		// each block for assertion presence and edge-case intent, rather than
		// the whole file (B3/B4).
		blocks := tsTestBlocks(src)
		base := filepath.Base(path)
		for i, b := range blocks {
			if !tsExpectRe.MatchString(b.body) {
				label := fmt.Sprintf("%s#%d", base, i+1)
				if tsRenderRe.MatchString(b.body) {
					renderOnly = append(renderOnly, label)
				} else {
					noAssertion = append(noAssertion, label)
				}
			}
			if blockSignalsEdge(b) {
				edgeCovered = true
			}
		}
		if n := len(tsSnapshotRe.FindAllString(src, -1)); n > excessiveSnapshotThreshold {
			snapshotHeavy = append(snapshotHeavy, fmt.Sprintf("%s (%d)", base, n))
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
	if testFiles > 0 && !edgeCovered {
		findings = append(findings, qualityFinding(scenario, ws, codeTestMissingEdgeCases, ws.RootPath,
			"No UI test block names or exercises an error/empty/boundary state; the suite appears positive-path only.",
			"no it()/test() title nor body references error/invalid/empty/boundary states",
			"Tests for error, empty, and boundary states alongside the happy path.",
			"positive-path only",
			"Untested error/empty states are the states users most often hit.",
			"Add tests for loading, empty, and error states (name them so the intent is clear).",
			now))
	}
	return findings
}

// tsBlock is one it()/test() block: its title and the source slice up to the
// next test block (an approximation of its body sufficient for heuristics).
type tsBlock struct {
	title string
	body  string
}

// stripTSComments removes line and block comments so heuristics do not match
// tokens that appear only in commentary. It is intentionally simple (it does
// not parse string literals); that is acceptable for these advisory signals.
func stripTSComments(src string) string {
	src = tsBlockComment.ReplaceAllString(src, " ")
	src = tsLineComment.ReplaceAllString(src, " ")
	return src
}

// tsTestBlocks splits comment-stripped source into per-test blocks by slicing
// between consecutive it()/test() occurrences.
func tsTestBlocks(src string) []tsBlock {
	idxs := tsTestBlockRe.FindAllStringIndex(src, -1)
	if len(idxs) == 0 {
		return nil
	}
	titles := map[int]string{}
	for _, m := range tsTestTitleRe.FindAllStringSubmatchIndex(src, -1) {
		// m[0] is the start of the it/test( match; group 1 (m[2]:m[3]) is the
		// title text. Pair the title to its block start position.
		if len(m) >= 4 {
			titles[m[0]] = src[m[2]:m[3]]
		}
	}
	blocks := make([]tsBlock, 0, len(idxs))
	for i, m := range idxs {
		end := len(src)
		if i+1 < len(idxs) {
			end = idxs[i+1][0]
		}
		blocks = append(blocks, tsBlock{title: titles[m[0]], body: src[m[0]:end]})
	}
	return blocks
}

// blockSignalsEdge reports whether a test block names or exercises an error or
// boundary state (its title or body references an edge keyword).
func blockSignalsEdge(b tsBlock) bool {
	if containsAny(strings.ToLower(b.title), edgeCaseKeywords) {
		return true
	}
	return containsAny(strings.ToLower(b.body), edgeCaseKeywords)
}

// analyzeRequirementTags reports, per requirement, which declared requirement
// IDs have no test reference — rather than treating one matched REQ as covering
// the whole scenario (B5). It emits a single finding enumerating the untagged
// IDs so traceability gaps are visible at requirement granularity.
func analyzeRequirementTags(scenario, scenarioRoot string, workspaces []Workspace, now string) []Finding {
	reqIDs := scenarioRequirementIDs(scenarioRoot)
	if len(reqIDs) == 0 {
		return nil
	}
	referenced := map[string]bool{}
	for _, ws := range workspaces {
		walkSourceFiles(ws.RootPath, func(path string) {
			if !isGoTestFile(path) && !isTSTestFile(path) {
				return
			}
			src := readFileString(path)
			for id := range reqIDs {
				if !referenced[id] && strings.Contains(src, id) {
					referenced[id] = true
				}
			}
		})
	}

	var untagged []string
	for id := range reqIDs {
		if !referenced[id] {
			untagged = append(untagged, id)
		}
	}
	if len(untagged) == 0 {
		return nil
	}
	sort.Strings(untagged)
	return []Finding{{
		ID:           codeTestUntaggedRequirement,
		Scenario:     scenario,
		Code:         codeTestUntaggedRequirement,
		Category:     "quality",
		Severity:     codeSeverity[codeTestUntaggedRequirement],
		FilePath:     filepath.Join(scenarioRoot, "requirements"),
		Message:      fmt.Sprintf("%d of %d declared requirement ID(s) have no test reference.", len(untagged), len(reqIDs)),
		Evidence:     "untagged requirement ids: " + strings.Join(truncateList(untagged, 15), ", "),
		Expected:     "Every declared requirement ID is referenced by at least one validating test (test name or comment).",
		Observed:     fmt.Sprintf("%d untagged requirement(s)", len(untagged)),
		WhyItMatters: "Untagged requirements break traceability: there is no proof those requirements are actually tested.",
		Remediation:  "Reference each listed REQ id in the name or a comment of the test that validates it.",
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
