package validation

import (
	"path/filepath"
	"testing"

	"unit-health/internal/adapters"
	"unit-health/internal/executor"
	"unit-health/internal/runhistory"
	"unit-health/internal/testquality"
)

const fixedNowStr = "2026-06-16T12:00:00Z"

func findingByCode(findings []Finding, code string) (Finding, bool) {
	for _, f := range findings {
		if f.Code == code {
			return f, true
		}
	}
	return Finding{}, false
}

// --- Coverage analyzer ---------------------------------------------------

// [REQ:UH-ANALYZE-001]
func TestAnalyzeCoverageGoProfile(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "api")
	// 4 statements total, 3 covered (count>0) => 75%.
	writeFile(t, filepath.Join(wsDir, "coverage.out"), "mode: atomic\n"+
		"mod/a.go:1.1,3.2 2 1\n"+
		"mod/a.go:4.1,5.2 1 0\n"+
		"mod/b.go:1.1,2.2 1 5\n")
	profile := reactViteUnitPolicyProfile()
	api := profile.PolicyClasses["go_service"]
	api.Coverage.MinimumPercent = 80
	profile.PolicyClasses["go_service"] = api
	writeUnitPolicyProfile(t, root, profile)

	ws := Workspace{ID: "api", Language: "go", RootPath: wsDir, CoverageCommand: "go test -cover ./..."}
	targets, findings := analyzeCoverage("demo", root, []Workspace{ws}, fixedNowStr)

	if len(targets) != 2 {
		t.Fatalf("expected 2 coverage targets, got %d: %+v", len(targets), targets)
	}
	// Below 80% threshold => LOW_COVERAGE.
	if _, ok := findingByCode(findings, codeLowCoverage); !ok {
		t.Errorf("expected LOW_COVERAGE, got %v", codes(findings))
	}
}

func TestAnalyzeCoverageAbsentArtifact(t *testing.T) {
	root := t.TempDir()
	ws := Workspace{ID: "api", Language: "go", RootPath: filepath.Join(root, "api"), CoverageCommand: "go test -cover ./..."}
	_, findings := analyzeCoverage("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeCoverageAbsent); !ok {
		t.Errorf("expected COVERAGE_ABSENT, got %v", codes(findings))
	}
}

func TestAnalyzeCoverageVitestSummary(t *testing.T) {
	root := t.TempDir()
	wsDir := filepath.Join(root, "ui")
	writeFile(t, filepath.Join(wsDir, "coverage", "coverage-summary.json"),
		`{"total":{"lines":{"total":10,"covered":9}},"src/App.tsx":{"lines":{"total":10,"covered":9}}}`)
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: wsDir, CoverageCommand: "pnpm test:coverage"}
	targets, findings := analyzeCoverage("demo", root, []Workspace{ws}, fixedNowStr)
	if len(targets) != 1 || targets[0].CoveragePercent != 90 {
		t.Fatalf("expected one 90%% target, got %+v", targets)
	}
	// The default threshold is 50%, so a 90% target is clean.
	if _, ok := findingByCode(findings, codeLowCoverage); ok {
		t.Errorf("did not expect LOW_COVERAGE without a threshold")
	}
}

func TestParseLCOV(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "lcov.info")
	writeFile(t, path, "SF:src/a.ts\nLF:4\nLH:2\nend_of_record\n")
	cov, ok := adapters.ReadCoverage(root, []adapters.Artifact{{Kind: "lcov", Path: "lcov.info"}})
	if !ok || cov["src/a.ts"].Total != 4 || cov["src/a.ts"].Covered != 2 {
		t.Fatalf("lcov parse = %+v ok=%v", cov, ok)
	}
}

// --- Architecture analyzer ----------------------------------------------

func TestAnalyzeArchitectureProductionImportsHelper(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "service.go"), "package demo\n\nimport _ \"demo/internal/testutil\"\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "util.go"), "package testutil\n")

	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	f, ok := findingByCode(findings, codeTestHelperFromProd)
	if !ok {
		t.Fatalf("expected TEST_HELPER_FROM_PRODUCTION, got %v", codes(findings))
	}
	if f.Severity != "error" {
		t.Errorf("helper-from-prod severity = %q, want error", f.Severity)
	}
}

func TestAnalyzeArchitectureAllowsGeneratedTemporalReplayBridge(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "modeltest", "model.go"), "package modeltest\n")
	writeFile(t, filepath.Join(root, "internal", "orders", "flow", "generated", "replay.go"), "package generated\n\nimport _ \"github.com/vrooli/vrooli/packages/proto/modeltest\"\n")

	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestHelperFromProd); ok {
		t.Fatalf("generated temporal replay must not be reported as production test-helper use: %v", codes(findings))
	}
}

func TestAnalyzeArchitectureMissingSeam(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "svc.go"), "package demo\n\nimport \"time\"\n\nfunc N() { _ = time.Now() }\n")
	writeFile(t, filepath.Join(root, "svc_test.go"), "package demo\n\nimport \"testing\"\n\nfunc TestN(t *testing.T) { N() }\n")

	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeMissingInjectableSeam); !ok {
		t.Errorf("expected MISSING_INJECTABLE_SEAM, got %v", codes(findings))
	}
}

func TestAnalyzeArchitectureTestUtilMissing(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	for _, n := range []string{"a", "b", "c"} {
		writeFile(t, filepath.Join(root, n+"_test.go"), "package demo\n\nimport \"testing\"\n\nfunc Test"+n+"(t *testing.T) { t.Fatal(\"x\") }\n")
	}
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); !ok {
		t.Errorf("expected TEST_UTIL_MISSING, got %v", codes(findings))
	}
}

// [REQ:UH-ANALYZE-002]
func TestAnalyzeArchitectureSharedTestUtilRootWithConsumers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "helper.go"), "package testutil\n\nfunc Name() string { return \"shared\" }\n")
	for _, n := range []string{"a", "b", "c"} {
		writeFile(t, filepath.Join(root, n+"_test.go"), "package demo\n\nimport \"testing\"\n\nfunc Test"+n+"(t *testing.T) { t.Helper() }\n")
	}

	findings := analyzeArchitecture("demo", []Workspace{{ID: "cli", Language: "go", RootPath: root}}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUtilMissing); ok {
		t.Fatalf("shared testutil root with consumers was reported missing: %v", codes(findings))
	}
}

func TestAnalyzeArchitectureGoProjectionDriftMissingImportBan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "testutil.go"), "package testutil\n")
	writeFile(t, filepath.Join(root, "a_test.go"), "package demo\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) { t.Fatal(\"x\") }\n")

	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	f, ok := findingByCode(findings, codeUnitProjectionDrift)
	if !ok {
		t.Fatalf("expected UNIT_POLICY_PROJECTION_DRIFT, got %v", codes(findings))
	}
	if f.Observed != "missing production import-ban test" {
		t.Fatalf("unexpected projection finding: %+v", f)
	}
}

func TestAnalyzeArchitectureGoProjectionCleanWithImportBan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "testutil.go"), "package testutil\n")
	writeFile(t, filepath.Join(root, "internal", "testutil", "no_prod_import_test.go"), "package testutil_test\n\nimport \"testing\"\n\nfunc TestNoProductionImports(t *testing.T) {}\n")
	writeFile(t, filepath.Join(root, "a_test.go"), "package demo\n\nimport \"testing\"\n\nfunc TestA(t *testing.T) { t.Fatal(\"x\") }\n")

	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeArchitecture("demo", []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeUnitProjectionDrift); ok {
		t.Fatalf("canonical Go projection should be clean, got %+v", findings)
	}
}

// --- Quality analyzer ---------------------------------------------------

func TestLegacyGoQualityDoesNotCompeteWithScopedAdapter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), `package demo

import "testing"

func TestSkipped(t *testing.T) { t.Skip("later") }

func TestNoAssert(t *testing.T) { _ = 1 + 1 }

func TestErrorPath(t *testing.T) {
	if false {
		t.Fatal("invalid input should fail")
	}
}
`)
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestSkippedOrOnly); ok {
		t.Errorf("static declaration invented runtime skip evidence: %v", codes(findings))
	}
}

func TestAnalyzeQualityTSDoesNotInventNativeEvidenceFromSourceNames(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "App.test.tsx"), `
import { render } from "@testing-library/react";
describe.only("App", () => {
  it("renders", () => { render(<App/>); });
});
`)
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestSkippedOrOnly); ok {
		t.Errorf("source heuristic bypassed calibrated native lint: %v", codes(findings))
	}
}

func TestAnalyzeQualityTSDoesNotTreatFitTextAsFocusedTest(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "src", "App.test.tsx"), `
it("reports a rejection", () => {
  const message = "Not a fit";
  expect(message).toContain("fit");
});
`)
	ws := Workspace{ID: "ui", Language: "typescript", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestSkippedOrOnly); ok {
		t.Errorf("ordinary fit text must not be treated as a focused test, got %v", codes(findings))
	}
}

func TestLegacyQualityDoesNotInventRegistryFromMarkdown(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "requirements", "core.md"), "# Core\n\n- REQ-CORE-001: must validate\n")
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), "package demo\n\nimport \"testing\"\n\nfunc TestX(t *testing.T){ t.Fatal(\"invalid\") }\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUntaggedRequirement); ok {
		t.Errorf("legacy quality invented registry obligations: %v", codes(findings))
	}
}

func TestAnalyzeQualityTaggedRequirementIsClean(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "requirements", "core.md"), "- REQ-CORE-001: must validate\n")
	writeFile(t, filepath.Join(root, "go.mod"), "module demo\n\ngo 1.25\n")
	writeFile(t, filepath.Join(root, "x_test.go"), "package demo\n\nimport \"testing\"\n\n// covers REQ-CORE-001\nfunc TestX(t *testing.T){ t.Fatal(\"invalid\") }\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	findings := analyzeQuality("demo", root, []Workspace{ws}, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestUntaggedRequirement); ok {
		t.Errorf("did not expect TEST_UNTAGGED_REQUIREMENT when a test references the REQ id")
	}
}

// --- Diagnostics analyzer -----------------------------------------------

func TestRollupFindingsMapsTypedViolationsPerWorkspace(t *testing.T) {
	workspaces := []Workspace{{ID: "ui", Language: "typescript", Framework: "vite", RootPath: "/ui"}, {ID: "api", Language: "go", Framework: "go", RootPath: "/api"}}
	results := []testquality.Result{
		{RuleID: "focused-test", Target: testquality.Target{Workspace: "ui", File: "App.test.tsx", TestID: "App"}, Status: testquality.Violation},
		{RuleID: "skip-declaration", Target: testquality.Target{Workspace: "api", File: "handler_test.go", TestID: "TestPlaceholder"}, Status: testquality.Violation},
		{RuleID: "requirement-link", Target: testquality.Target{Workspace: "api", File: "handler_test.go", TestID: "TestUnlinked"}, Status: testquality.Violation},
		{RuleID: "focused-test", Target: testquality.Target{Workspace: "ui", File: "other.test.tsx", TestID: "Unknown"}, Status: testquality.Unknown},
	}

	findings := rollupFindings("demo", workspaces, results, fixedNowStr)
	if len(findings) != 3 {
		t.Fatalf("rollups=%d, want three bounded workspace/rule findings: %+v", len(findings), findings)
	}
	if finding, ok := findingByCode(findings, codeTestSkippedOrOnly); !ok || finding.WorkspaceID != "ui" || finding.Evidence != "violating tests: App.test.tsx#App (focused-test)" {
		t.Fatalf("focused-test rollup=%+v", finding)
	}
	if finding, ok := findingByCode(findings, codeTestUntaggedRequirement); !ok || finding.WorkspaceID != "api" || finding.Evidence != "violating tests: handler_test.go#TestUnlinked (requirement-link)" {
		t.Fatalf("requirement-link rollup=%+v", finding)
	}
}

func TestRollupFindingsDoesNotPromoteUnknownOrClean(t *testing.T) {
	workspaces := []Workspace{{ID: "ui", Language: "typescript", RootPath: "/ui"}}
	results := []testquality.Result{
		{RuleID: "focused-test", Target: testquality.Target{Workspace: "ui", File: "App.test.tsx", TestID: "App"}, Status: testquality.Unknown},
		{RuleID: "skip-declaration", Target: testquality.Target{Workspace: "ui", File: "other.test.tsx", TestID: "Other"}, Status: testquality.CheckedClean},
	}
	if findings := rollupFindings("demo", workspaces, results, fixedNowStr); len(findings) != 0 {
		t.Fatalf("unknown/clean results produced rollups: %+v", findings)
	}
}

func TestAnalyzeDiagnosticsRuntimePressureAndHang(t *testing.T) {
	plan := ExecutionPlan{Commands: []PlannedCommand{
		{WorkspaceID: "api", Command: "go test ./...", WorkingDirectory: "/x/api", TimeoutSeconds: 100},
		{WorkspaceID: "ui", Command: "pnpm test", WorkingDirectory: "/x/ui", TimeoutSeconds: 100},
	}}
	results := []CommandResult{
		{Name: "go", Command: "go test ./...", WorkingDirectory: "/x/api", Status: "passed", DurationMS: 90000, TimeoutSeconds: 100},
		{Name: "ui", Command: "pnpm test", WorkingDirectory: "/x/ui", Status: "timeout", FailureClass: "timeout_hang", StderrExcerpt: "stuck", TimeoutSeconds: 100},
	}
	// No history: near-timeout is a runtime diagnostic, not a TEST_RUNTIME_GROWTH
	// finding (growth needs cross-run history — see TestRuntimeGrowthFromHistory).
	diags, findings := analyzeDiagnostics("demo", nil, plan, results, nil, fixedNowStr)

	var sawRuntime, sawHang bool
	for _, d := range diags {
		switch d.Kind {
		case "runtime":
			sawRuntime = true
		case "hang":
			sawHang = true
		}
	}
	if !sawRuntime || !sawHang {
		t.Fatalf("expected runtime+hang diagnostics, got %+v", diags)
	}
	if _, ok := findingByCode(findings, codeTestRuntimeGrowth); ok {
		t.Errorf("near-timeout must NOT fire TEST_RUNTIME_GROWTH without history, got %v", codes(findings))
	}
}

func TestAnalyzeDiagnosticsFlakeMarkers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "x_test.go"), "package demo\n\nimport \"testing\"\n\n// this test is flaky under load\nfunc TestX(t *testing.T){}\n")
	ws := Workspace{ID: "api", Language: "go", RootPath: root}
	// Static "flaky" source markers are now a weak supplementary diagnostic
	// only (info), not a TEST_FLAKE_SUSPECTED finding — that requires cross-run
	// variance (see TestFlakeFromCrossRunVariance).
	diags, findings := analyzeDiagnostics("demo", []Workspace{ws}, ExecutionPlan{}, nil, nil, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestFlakeSuspected); ok {
		t.Errorf("static markers must NOT fire TEST_FLAKE_SUSPECTED, got %v", codes(findings))
	}
	var sawFlake bool
	for _, d := range diags {
		if d.Kind == "flake" {
			sawFlake = true
		}
	}
	if !sawFlake {
		t.Errorf("expected a supplementary flake diagnostic, got %+v", diags)
	}
}

func TestRuntimeGrowthFromHistory(t *testing.T) {
	identity := reliabilityTestIdentity(t, "source", "toolchain")
	plan := ExecutionPlan{Commands: []PlannedCommand{
		{WorkspaceID: "api", Command: "go test ./...", WorkingDirectory: "/x/api", TimeoutSeconds: 600},
	}}
	results := []CommandResult{
		{Command: "go test ./...", WorkingDirectory: "/x/api", Status: "passed", DurationMS: 9000, TimeoutSeconds: 600},
	}
	// Three prior passing runs around 3s; current run is 9s = 3× baseline.
	hist := []runhistory.CommandSample{
		{WorkspaceID: "api", Command: "go test ./...", DurationMS: 3000, Status: "passed"},
		{WorkspaceID: "api", Command: "go test ./...", DurationMS: 2900, Status: "passed"},
		{WorkspaceID: "api", Command: "go test ./...", DurationMS: 3100, Status: "passed"},
	}
	results[0].ComparisonIdentity = identity
	for i := range hist {
		hist[i].Identity = identity
	}
	_, findings := analyzeDiagnostics("demo", nil, plan, results, hist, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestRuntimeGrowth); !ok {
		t.Errorf("expected TEST_RUNTIME_GROWTH from 3× baseline growth, got %v", codes(findings))
	}
}

func TestFlakeFromCrossRunVariance(t *testing.T) {
	identity := reliabilityTestIdentity(t, "source", "toolchain")
	plan := ExecutionPlan{Commands: []PlannedCommand{
		{WorkspaceID: "api", Command: "go test ./...", WorkingDirectory: "/x/api", TimeoutSeconds: 600},
	}}
	// Current run failed; a prior run passed → flip-flop.
	results := []CommandResult{
		{Command: "go test ./...", WorkingDirectory: "/x/api", Status: "failed", TimeoutSeconds: 600},
	}
	hist := []runhistory.CommandSample{
		{WorkspaceID: "api", Command: "go test ./...", DurationMS: 3000, Status: "passed"},
	}
	results[0].ComparisonIdentity = identity
	results[0].FailureClass = executor.ClassTestFailure
	hist[0].Identity = identity
	_, findings := analyzeDiagnostics("demo", nil, plan, results, hist, fixedNowStr)
	if _, ok := findingByCode(findings, codeTestFlakeSuspected); !ok {
		t.Errorf("expected TEST_FLAKE_SUSPECTED from pass/fail flip-flop, got %v", codes(findings))
	}
}
