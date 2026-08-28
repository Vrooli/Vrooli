package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

// [REQ:TM-LS-009]
func TestScanSeamsMatchesCallsAndLiteralsWithoutTextFalsePositives(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/bypass.go", `package sample
import "os"
func bypass() { _ = os.Rename("a", "b"); _ = "VROOLI_REPO_ROOT" }
`)
	writeSeamTestFile(t, root, "src/clean.go", `package sample
// os.Rename("a", "b") and VROOLI_REPO_ROOT are documentation only.
func clean() {}
`)
	writeSeamTestFile(t, root, "src/aliased.go", `package sample
import filesystem "os"
func aliased() { _ = filesystem.Rename("a", "b") }
`)
	seams := []Seam{
		{ID: "owned-write", Canonical: "config.WriteOwnedFileAtomic", Why: "ownership", Remediation: "use the owned writer", Bypass: SeamBypass{Kind: "call", Pattern: `^os\.Rename$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "repo-root", Canonical: "buildinfo.ResolveSourceRoot", Why: "one root", Remediation: "use buildinfo", Bypass: SeamBypass{Kind: "literal", Pattern: `^VROOLI_REPO_ROOT$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("expected direct call, aliased call, and literal hits only, got %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsSkipsCanonicalDeclarationFile(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "config/write.go", `package config
import "os"
func WriteOwnedFileAtomic() { _ = os.Rename("a", "b") }
`)
	seam := Seam{ID: "owned-write", Canonical: "config.WriteOwnedFileAtomic", Why: "ownership", Remediation: "use the owned writer", Bypass: SeamBypass{Kind: "call", Pattern: `^os\.Rename$`}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("canonical declaration must not report its implementation detail: %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsMatchesNumericAndQualifiedDurationLiterals(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/literals.go", `package sample
import tm "time"
var mode = 0o644
var timeout = 30 * tm.Second
const namedTimeout = 45 * tm.Second
`)
	seams := []Seam{
		{ID: "mode", Canonical: "tuning.PermFile", Why: "named modes", Remediation: "use tuning", Bypass: SeamBypass{Kind: "literal", Pattern: `^0[oO][0-7]+$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "duration", Canonical: "tuning.Duration", Why: "named durations", Remediation: "use tuning", Bypass: SeamBypass{Kind: "literal", Pattern: `^time\.(Second|Minute|Hour|Millisecond|Microsecond|Nanosecond):[0-9]+$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected numeric and qualified-duration hits, got %#v", hits)
	}
}

// [REQ:TM-LS-009]
func TestScanSeamsHonorsExcludeAndBudget(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/one.go", "package sample\nfunc one(){ bypass() }\n")
	writeSeamTestFile(t, root, "generated/two.go", "package sample\nfunc two(){ bypass() }\n")
	seam := Seam{ID: "call", Canonical: "canonical", Why: "reason", Remediation: "fix", Bypass: SeamBypass{Kind: "call", Pattern: `^bypass$`}, Scope: SeamScope{Include: []string{"**"}, Exclude: []string{"generated/**"}}, Severity: "high", Budget: 1}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected one non-excluded hit, got %#v", hits)
	}
	if findings := seamFindings("sample", hits); len(findings) != 0 {
		t.Fatalf("budgeted hit must not gate: %#v", findings)
	}
	seam.Budget = 0
	hits, err = ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if findings := seamFindings("sample", hits); len(findings) != 1 || findings[0].RuleID != "BYPASSED_SEAM" {
		t.Fatalf("expected one gating finding, got %#v", findings)
	}
}

func TestScanSeamsMatchesDeclarationKindsAndPackageRepetition(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "one/decl.go", `package one
type Shared struct{}
type Contract interface { Run() }
const SharedValue = 1
var SharedVar = 2
func bindOne() {}
func (Shared) bindMethod() {}
`)
	writeSeamTestFile(t, root, "two/decl.go", `package two
type Shared struct{}
const SharedValue = 1
func bindTwo() {}
`)
	seams := []Seam{
		{ID: "funcs", Canonical: "canonical.Func", Why: "one", Remediation: "use one", Bypass: SeamBypass{Kind: "declaration", Pattern: `^bind`, DeclKind: "func", RepeatedAcrossPackages: 2}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"},
		{ID: "methods", Canonical: "canonical.Method", Why: "one", Remediation: "use one", Bypass: SeamBypass{Kind: "declaration", Pattern: `^bind`, DeclKind: "method"}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"},
		{ID: "interfaces", Canonical: "canonical.Interface", Why: "one", Remediation: "use one", Bypass: SeamBypass{Kind: "declaration", Pattern: `^Contract$`, DeclKind: "interface"}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"},
		{ID: "consts", Canonical: "canonical.Const", Why: "one", Remediation: "use one", Bypass: SeamBypass{Kind: "declaration", Pattern: `^SharedValue$`, DeclKind: "const", RepeatedAcrossPackages: 2}, Scope: SeamScope{Include: []string{"**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, hit := range hits {
		counts[hit.SeamID]++
	}
	if counts["funcs"] != 0 {
		t.Fatalf("func repetition unexpectedly matched distinct names: %#v", counts)
	}
	if counts["methods"] != 1 || counts["interfaces"] != 1 || counts["consts"] != 2 {
		t.Fatalf("declaration matches = %#v, want method=1 interface=1 const=2", counts)
	}
}

func TestScanSeamsMatchesSwitchOnArgvShape(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/dispatch.go", `package sample
func dispatch(args []string) {
 switch args[0] { case "run": }
}
`)
	seam := Seam{ID: "dispatcher", Canonical: "commandtree", Why: "typed dispatch", Remediation: "use commandtree", Bypass: SeamBypass{Kind: "shape", ShapeKind: "switch_on_argv", Pattern: `^switch_on_argv$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Symbol != "switch_on_argv" {
		t.Fatalf("shape hits = %#v", hits)
	}
}

func TestScanSeamsGroupsEquivalentInterfaceShapes(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/interfaces.go", `package sample
type First interface { Run(ctx context.Context) error }
type Second interface { Run(other context.Context) error }
`)
	seam := Seam{ID: "interfaces", Canonical: "runner", Why: "one contract", Remediation: "share contract", Bypass: SeamBypass{Kind: "shape", ShapeKind: "interface_method_set", Pattern: "^(First|Second)$", MinMembers: 2}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("interface shape hits = %#v", hits)
	}
}

func TestScanSeamsMatchesContextDurationAndManifestDecoderShapes(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "src/shapes.go", `package sample
import "context"
import "google.golang.org/protobuf/types/known/structpb"
type Decoder struct { Dependencies struct { Resources []string `+"`json:\"resources\"`"+` } `+"`json:\"dependencies\"`"+` }
func run(ctx context.Context) { context.WithTimeout(ctx, 30) }
func dynamic() { _, _ = structpb.NewValue("x") }
`)
	seams := []Seam{
		{ID: "duration", Canonical: "tuning.Timeout", Why: "named timing", Remediation: "use tuning", Bypass: SeamBypass{Kind: "shape", ShapeKind: "context_duration_literal", Pattern: `^context\.WithTimeout$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "decoder", Canonical: "scenario.LoadServiceManifest", Why: "one decoder", Remediation: "use scenario", Bypass: SeamBypass{Kind: "shape", ShapeKind: "service_manifest_decoder", Pattern: `^Decoder$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "json", Canonical: "cliout.WriteJSONValue", Why: "one renderer", Remediation: "use cliout", Bypass: SeamBypass{Kind: "shape", ShapeKind: "dynamic_json_writer", Pattern: `^dynamic$`}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("shape hits = %#v, want duration, decoder, and dynamic JSON", hits)
	}
}

func TestScanSeamsReportsMissingAbsenceTargetAndJSONPointer(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "trigger.go", "package sample\nfunc trigger() {}\n")
	writeSeamTestFile(t, root, "present.json", `{"dependencies":{"scenarios":{}}}`)
	seams := []Seam{
		{ID: "missing-file", Canonical: "file", Why: "required", Remediation: "add it", Bypass: SeamBypass{Kind: "absence", Pattern: "^$", RequireFor: "trigger.go", RequirePresent: "missing.json"}, Scope: SeamScope{Include: []string{"**"}}, Severity: "critical"},
		{ID: "missing-pointer", Canonical: "pointer", Why: "required", Remediation: "add it", Bypass: SeamBypass{Kind: "absence", Pattern: "^$", RequireFor: "trigger.go", RequirePresent: "present.json#/dependencies/scenarios/code-facts"}, Scope: SeamScope{Include: []string{"**"}}, Severity: "critical"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("absence hits = %#v", hits)
	}
	seams[0].Bypass.RequirePresent = "present.json"
	hits, err = ScanSeams(root, seams[:1])
	if err != nil || len(hits) != 0 {
		t.Fatalf("satisfied absence = hits %v err %v", hits, err)
	}
}

func TestScanSeamsRequiresJSONWriterFunctionInContractTest(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "internal/app/credentials/command.go", `package credentials
func newJSONCommand() { cliout.WriteJSONValue(nil, nil) }
`)
	seam := Seam{ID: "json-contract", Canonical: "JSON contract", Why: "stable output", Remediation: "assert it", Bypass: SeamBypass{Kind: "absence", Pattern: "^$", RequireFor: "internal/app/credentials/*.go", RequirePresent: "__json_contract_assertion__"}, Scope: SeamScope{Include: []string{"internal/app/credentials/**"}}, Severity: "critical"}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil || len(hits) != 1 {
		t.Fatalf("missing contract assertion: hits=%v err=%v", hits, err)
	}
	writeSeamTestFile(t, root, "internal/app/credentials/command_test.go", `package credentials
func TestContract() { _ = "newJSONCommand" }
`)
	hits, err = ScanSeams(root, []Seam{seam})
	if err != nil || len(hits) != 0 {
		t.Fatalf("satisfied contract assertion: hits=%v err=%v", hits, err)
	}
}

func TestBrokerFactMatchesQualifiedPackageIdentity(t *testing.T) {
	candidate := compiledSeam{
		seam:    Seam{Bypass: SeamBypass{Kind: "call"}},
		pattern: regexp.MustCompile(`^logx\.FormatJSON$`),
	}
	fact := &factsv1.GenericFact{
		Family:     factsv1.FactFamily_FACT_FAMILY_REFERENCES,
		Subject:    "FormatJSON",
		Attributes: map[string]string{"package_id": "package:github.com/vrooli/vrooli/internal/logx"},
	}
	if !brokerFactMatches(candidate, fact) {
		t.Fatal("broker did not match package-qualified reference")
	}
}

// [REQ:TM-LS-009]
func TestLoadSeamsRejectsUnknownFields(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, canonicalSeamsPath, `{"schemaVersion":"1.0.0","unknown":true,"seams":[]}`)
	if _, err := LoadSeams(root); err == nil {
		t.Fatal("expected unknown field to fail strict decoding")
	}
}

// [REQ:TM-LS-009]
func TestLoadSeamsRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, canonicalSeamsPath, `{"schemaVersion":"1.0.0","seams":[]} {}`)
	if _, err := LoadSeams(root); err == nil {
		t.Fatal("expected trailing JSON value to fail strict decoding")
	}
}

// [REQ:TM-LS-009]
func TestRepositoryCanonicalSeamsLoadAndScan(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	if got := seamTreeRoot(filepath.Join(root, "internal")); got != root {
		t.Fatalf("control-plane seam root = %q, want repository root %q", got, root)
	}
	seams, err := LoadSeams(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) != 36 {
		t.Fatalf("expected thirty-six declared seams, got %d", len(seams))
	}
	if _, err := ScanSeams(root, seams); err != nil {
		t.Fatal(err)
	}
}

func writeSeamTestFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
