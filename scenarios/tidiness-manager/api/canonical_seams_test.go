package main

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
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
	if findings := seamFindings("sample", []Seam{seam}, hits); len(findings) != 0 {
		t.Fatalf("budgeted hit must not gate: %#v", findings)
	}
	seam.Budget = 0
	hits, err = ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if findings := seamFindings("sample", []Seam{seam}, hits); len(findings) != 1 || findings[0].RuleID != "BYPASSED_SEAM" {
		t.Fatalf("expected one gating finding, got %#v", findings)
	}
}

func TestSemanticNamingSplitsObservedIdentifiers(t *testing.T) {
	tests := map[string][]string{
		"handlerParameterA":   {"handler", "Parameter", "A"},
		"mndMainNumberValue2": {"mnd", "Main", "Number", "Value", "2"},
		"bytesPerKiB":         {"bytes", "Per", "Ki", "B"},
		"desktopMinPercent":   {"desktop", "Min", "Percent"},
		"configPath":          {"config", "Path"},
		"handlerDedicated":    {"handler", "Dedicated"},
	}
	for identifier, want := range tests {
		if got := splitIdentifierWords(identifier); !reflect.DeepEqual(got, want) {
			t.Errorf("splitIdentifierWords(%q) = %#v, want %#v", identifier, got, want)
		}
	}
}

func TestSemanticNamingRejectsGenericConstantAndAcceptsDomainName(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "internal/sample/constants.go", `package sample
const handlerParameterZ = 7
const retryAttemptLimit = 7
const bytesPerKiB = 1024
const desktopMinPercent = 25
const configPath = "config.json"
const handlerDedicated = "dedicated"
`)
	seam := Seam{
		ID: "semantic", Canonical: "domain vocabulary", Why: "readability", Remediation: "name the meaning",
		Bypass: SeamBypass{Kind: "semantic-naming", Pattern: `.*`, DeclKind: "const", GenericWords: []string{"handler", "parameter", "value", "number", "const", "literal", "item", "data", "mnd", "main"}, MinDomainWords: 1},
		Scope:  SeamScope{Include: []string{"internal/**"}}, Severity: "high",
	}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Symbol != "handlerParameterZ" {
		t.Fatalf("semantic naming hits = %#v", hits)
	}
}

func TestSuppressionBreadthPricesFileLevelDirectiveByDeclarations(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "internal/sample/broad.go", `//nolint:goconst // fixture literals intentionally repeat
package sample
const first = "same"
const second = "same"
var third = "same"
type fourth string
func fifth() {}
`)
	seam := Seam{
		ID: "breadth", Canonical: "narrow suppression", Why: "scope", Remediation: "narrow it",
		Bypass: SeamBypass{Kind: "suppression-breadth", Pattern: `^//nolint:goconst`, Unit: "declarations"},
		Scope:  SeamScope{Include: []string{"internal/**"}}, Severity: "high",
	}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 5 {
		t.Fatalf("file-level suppression breadth = %d, want 5: %#v", len(hits), hits)
	}

	writeSeamTestFile(t, root, "internal/sample/broad.go", `package sample
//nolint:goconst // fixture literals intentionally repeat
const first = "same"
const second = "same"
var third = "same"
type fourth string
func fifth() {}
`)
	hits, err = ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("declaration-level suppression breadth = %d, want 1: %#v", len(hits), hits)
	}
}

func TestReplacedCallFindsPrivateNestedSwapLoopOnly(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "internal/sample/sorts.go", `package sample
func privateSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
func ExportedSort(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
func privateSortWithCall(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0; j-- {
			observe(values)
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
func observe([]string) {}
`)
	seam := Seam{
		ID: "sort", Canonical: "slices.Sort", Why: "performance", Remediation: "use slices.Sort",
		Bypass: SeamBypass{Kind: "replaced-call", Pattern: `.*`, CanonicalCall: "slices.Sort", ShapeKind: "nested_swap_loop"},
		Scope:  SeamScope{Include: []string{"internal/**"}}, Severity: "critical",
	}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Symbol != "privateSort" || hits[0].Severity != "critical" {
		t.Fatalf("replaced-call hits = %#v", hits)
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

func TestScanSeamsMatchesPairedDeclarationPathsAndExcludesAliases(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "internal/app/project/types.go", "package projectapp\ntype Shared struct{}\ntype AliasOnly struct{}\n")
	writeSeamTestFile(t, root, "internal/cli/projectcli/types.go", "package projectcli\ntype Shared struct{}\ntype AliasOnly = projectapp.AliasOnly\n")
	writeSeamTestFile(t, root, "internal/app/resource/types.go", "package resourceapp\ntype AppOnly struct{}\n")
	seam := Seam{
		ID: "app-cli", Canonical: "internal/app/<domain>", Why: "one contract", Remediation: "use an alias",
		Bypass: SeamBypass{
			Kind: "declaration", Pattern: `^[A-Z]`, ExcludeAliases: true,
			PairedPathPatterns: []string{`^internal/app/([^/]+)/`, `^internal/cli/([a-z0-9]+)(?:cli|handlers)/`},
		},
		Scope: SeamScope{Include: []string{"internal/**"}}, Severity: "high",
	}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Symbol != "Shared" || hits[0].Path != "internal/cli/projectcli/types.go" {
		t.Fatalf("paired declaration hits = %#v, want the CLI-side Shared definition", hits)
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
		{ID: "decoder", Canonical: "scenario.LoadServiceManifest", Why: "one decoder", Remediation: "use scenario", Bypass: SeamBypass{Kind: "shape", ShapeKind: "json_nesting", Pattern: `^Decoder$`, OuterKey: "dependencies", InnerKey: "resources"}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
		{ID: "json", Canonical: "cliout.WriteJSONValue", Why: "one renderer", Remediation: "use cliout", Bypass: SeamBypass{Kind: "shape", ShapeKind: "constructs_type", Pattern: `^dynamic$`, ConstructedType: "structpb.Value"}, Scope: SeamScope{Include: []string{"src/**"}}, Severity: "high"},
	}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 3 {
		t.Fatalf("shape hits = %#v, want duration, decoder, and dynamic JSON", hits)
	}
}

func TestScanSeamsExcludesConfiguredSymbols(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "one/one_test.go", "package one\nfunc TestConformance() {}\nfunc TestGenerated() {}\n")
	writeSeamTestFile(t, root, "two/two_test.go", "package two\nfunc TestConformance() {}\nfunc TestGenerated() {}\n")
	seam := Seam{
		ID: "tests", Canonical: "focused tests", Why: "specific names", Remediation: "rename",
		Bypass: SeamBypass{Kind: "declaration", Pattern: `^Test[A-Za-z]+$`, DeclKind: "func", RepeatedAcrossPackages: 2, ExcludeSymbols: []string{"TestConformance"}},
		Scope:  SeamScope{Include: []string{"**/*_test.go"}}, Severity: "low",
	}
	hits, err := ScanSeams(root, []Seam{seam})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 || hits[0].Symbol != "TestGenerated" || hits[1].Symbol != "TestGenerated" {
		t.Fatalf("configured symbol exclusions produced hits %#v", hits)
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

func TestScanSeamsReportsForbiddenExecutable(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, "kept/source.go", "package kept\n")
	writeSeamTestFile(t, root, "artifact", "\x7fELFpayload")
	writeSeamTestFile(t, root, ".vrooli/build/allowed", "\x7fELFpayload")
	if err := os.Chmod(filepath.Join(root, "artifact"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, ".vrooli/build/allowed"), 0o755); err != nil {
		t.Fatal(err)
	}
	seams := []Seam{{
		ID: "executables", Canonical: "build output", Why: "generated", Remediation: "remove it",
		Bypass: SeamBypass{Kind: "absence", Pattern: "^$", RequireFor: "**", ForbidKind: "executable"},
		Scope:  SeamScope{Include: []string{"**"}, Exclude: []string{".vrooli/build/**"}}, Severity: "high",
	}}
	hits, err := ScanSeams(root, seams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].Path != "artifact" {
		t.Fatalf("executable hits = %#v", hits)
	}
	findings := seamFindings("sample", seams, hits)
	if len(findings) != 1 || findings[0].RuleID != "BYPASSED_SEAM" {
		t.Fatalf("executable budget findings = %#v", findings)
	}
}

func TestSeamFindingsReportSlackAndNeverMatchesWithoutHits(t *testing.T) {
	seam := Seam{ID: "never", Canonical: "canonical", Why: "coverage", Remediation: "repair", Severity: "high", Budget: 5, Reserve: 1, ReserveReason: "one reviewed exception"}
	findings := seamFindings("sample", []Seam{seam}, nil)
	if len(findings) != 2 {
		t.Fatalf("zero-hit findings = %#v", findings)
	}
	byRule := map[string]TidinessFinding{}
	for _, finding := range findings {
		byRule[finding.RuleID] = finding
	}
	if byRule["SEAM_BUDGET_SLACK"].Evidence["slack"] != 4 {
		t.Fatalf("slack finding = %#v", byRule["SEAM_BUDGET_SLACK"])
	}
	if byRule["SEAM_RULE_NEVER_MATCHES"].Severity != "critical" {
		t.Fatalf("never-matches finding = %#v", byRule["SEAM_RULE_NEVER_MATCHES"])
	}
}

func TestSeamFindingsReportSlackAboveObservedPlusReserve(t *testing.T) {
	seam := Seam{ID: "loose", Canonical: "canonical", Why: "coverage", Remediation: "repair", Severity: "high", Budget: 4, Reserve: 1, ReserveReason: "one reviewed exception"}
	hits := []SeamHit{{SeamID: seam.ID}, {SeamID: seam.ID}}
	findings := seamFindings("sample", []Seam{seam}, hits)
	if len(findings) != 1 || findings[0].RuleID != "SEAM_BUDGET_SLACK" {
		t.Fatalf("slack findings = %#v", findings)
	}
	if findings[0].Evidence["observed"] != 2 || findings[0].Evidence["reserve"] != 1 || findings[0].Evidence["slack"] != 1 {
		t.Fatalf("slack evidence = %#v", findings[0].Evidence)
	}
}

func TestLoadSeamsRejectsUnjustifiedNonZeroAllowance(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, `{"schemaVersion":"1.0.0","seams":[{"id":"slack","canonical":"canonical","why":"coverage","remediation":"repair","bypass":{"kind":"call","pattern":"^x$"},"scope":{"include":["**"],"exclude":[]},"severity":"high","budget":3}]}`)
	if _, err := LoadSeams(root); err == nil || !strings.Contains(err.Error(), "reserve_reason") {
		t.Fatalf("unjustified budget error = %v", err)
	}
}

func TestLoadSeamsRequiresParameterizedShapeFields(t *testing.T) {
	tests := []struct {
		name      string
		shapeKind string
		want      string
	}{
		{name: "JSON nesting", shapeKind: "json_nesting", want: "outerKey and innerKey"},
		{name: "constructed type", shapeKind: "constructs_type", want: "constructedType"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			config := `{"schemaVersion":"1.0.0","seams":[{"id":"shape","canonical":"canonical","why":"coverage","remediation":"repair","bypass":{"kind":"shape","pattern":".*","shapeKind":"` + test.shapeKind + `"},"scope":{"include":["**"],"exclude":[]},"severity":"high","budget":0}]}`
			writeSeamTestFile(t, root, repositorySeamsPath, config)
			if _, err := LoadSeams(root); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("parameter validation error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCanonicalSeamDocAgreesWithSupportedKinds(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join("..", "docs", "reference", "canonical-seams.md"))
	if err != nil {
		t.Fatal(err)
	}
	if got := documentedKinds(string(doc), "Accepted bypass kinds:"); !reflect.DeepEqual(got, supportedBypassKinds) {
		t.Fatalf("documented bypass kinds = %#v, supported = %#v", got, supportedBypassKinds)
	}
	if got := documentedKinds(string(doc), "Accepted shape kinds:"); !reflect.DeepEqual(got, supportedShapeKinds) {
		t.Fatalf("documented shape kinds = %#v, supported = %#v", got, supportedShapeKinds)
	}
}

func documentedKinds(document, prefix string) []string {
	for _, line := range strings.Split(document, "\n") {
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		matches := regexp.MustCompile("`([^`]+)`").FindAllStringSubmatch(line, -1)
		kinds := make([]string, 0, len(matches))
		for _, match := range matches {
			kinds = append(kinds, match[1])
		}
		return kinds
	}
	return nil
}

func TestSeamResolutionBaselineOnly(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(baseSeamJSON("base", "baseline")))
	target := filepath.Join(root, "internal")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	seams, resolution, err := ResolveSeams(target, SeamTargetControlPlane)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) != 1 || seams[0].Canonical != "baseline" || len(resolution.Files) != 1 || resolution.ScanRoot != root {
		t.Fatalf("baseline resolution: seams=%#v resolution=%#v", seams, resolution)
	}
}

func TestSeamResolutionRebasesOnlyTargetScopedBaselineRules(t *testing.T) {
	root := t.TempDir()
	control := baseSeamJSON("control", "control")
	scenario := strings.ReplaceAll(baseSeamJSON("scenario", "scenario"), `"internal/**"`, `"scenarios/demo/**"`)
	writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(control+","+scenario))
	target := filepath.Join(root, "scenarios", "demo")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	seams, resolution, err := ResolveSeams(target, SeamTargetScenario)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) != 1 || seams[0].ID != "scenario" || !reflect.DeepEqual(seams[0].Scope.Include, []string{"**"}) || resolution.ScanRoot != target {
		t.Fatalf("scenario baseline resolution: seams=%#v resolution=%#v", seams, resolution)
	}
}

func TestSeamOverlayAddsAndReplaces(t *testing.T) {
	tests := []struct {
		name       string
		overlay    string
		wantCount  int
		wantSecond string
	}{
		{name: "adds", overlay: baseSeamJSON("added", "overlay"), wantCount: 2, wantSecond: "overlay"},
		{name: "replaces", overlay: baseSeamJSON("base", "replacement"), wantCount: 1, wantSecond: "replacement"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(baseSeamJSON("base", "baseline")))
			target := filepath.Join(root, "internal")
			writeSeamTestFile(t, target, targetSeamsPath, seamConfigJSON(test.overlay))
			seams, resolution, err := ResolveSeams(target, SeamTargetControlPlane)
			if err != nil {
				t.Fatal(err)
			}
			if len(seams) != test.wantCount || seams[len(seams)-1].Canonical != test.wantSecond || len(resolution.Files) != 2 {
				t.Fatalf("overlay resolution: seams=%#v resolution=%#v", seams, resolution)
			}
		})
	}
}

func TestSeamOverlayDisablesBaseline(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(baseSeamJSON("base", "baseline")))
	target := filepath.Join(root, "internal")
	writeSeamTestFile(t, target, targetSeamsPath, seamConfigJSON(`{"id":"base","disabled":true,"reserve_reason":"not applicable to this target"}`))
	seams, resolution, err := ResolveSeams(target, SeamTargetControlPlane)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) != 0 || len(resolution.Files) != 2 {
		t.Fatalf("disabled resolution: seams=%#v resolution=%#v", seams, resolution)
	}
}

func TestSeamOverlayRejectsDisableWithoutReason(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(baseSeamJSON("base", "baseline")))
	target := filepath.Join(root, "internal")
	writeSeamTestFile(t, target, targetSeamsPath, seamConfigJSON(`{"id":"base","disabled":true}`))
	if _, _, err := ResolveSeams(target, SeamTargetControlPlane); err == nil || !strings.Contains(err.Error(), "reserve_reason") {
		t.Fatalf("disable guard error = %v", err)
	}
}

func TestSeamOverlayEndToEnd(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, seamConfigJSON(""))
	target := filepath.Join(root, "internal")
	writeSeamTestFile(t, root, "internal/sample.go", "package internal\nfunc sample() { bypass() }\n")
	overlayPath := filepath.Join(target, filepath.FromSlash(targetSeamsPath))
	writeSeamTestFile(t, target, targetSeamsPath, seamConfigJSON(baseSeamJSON("overlay", "canonical")))
	seams, resolution, err := ResolveSeams(target, SeamTargetControlPlane)
	if err != nil {
		t.Fatal(err)
	}
	hits, err := ScanSeams(resolution.ScanRoot, seams)
	if err != nil || len(hits) != 1 || len(resolution.Files) != 2 {
		t.Fatalf("overlay scan: hits=%#v resolution=%#v err=%v", hits, resolution, err)
	}
	if err := os.Remove(overlayPath); err != nil {
		t.Fatal(err)
	}
	seams, resolution, err = ResolveSeams(target, SeamTargetControlPlane)
	if err != nil {
		t.Fatal(err)
	}
	hits, err = ScanSeams(resolution.ScanRoot, seams)
	if err != nil || len(hits) != 0 || len(resolution.Files) != 1 {
		t.Fatalf("scan after overlay removal: hits=%#v resolution=%#v err=%v", hits, resolution, err)
	}
}

func seamConfigJSON(seams string) string {
	return `{"schemaVersion":"1.0.0","seams":[` + seams + `]}`
}

func baseSeamJSON(id, canonical string) string {
	return `{"id":"` + id + `","canonical":"` + canonical + `","why":"reason","remediation":"repair","bypass":{"kind":"call","pattern":"^bypass$"},"scope":{"include":["internal/**"],"exclude":[]},"severity":"high","budget":0}`
}

func TestRepositoryExecutableSeamMatchesDeclaredBudget(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", ".."))
	seams, err := LoadSeams(root)
	if err != nil {
		t.Fatal(err)
	}
	var executableSeams []Seam
	for _, seam := range seams {
		if seam.ID == "committed-executable-artifacts" {
			executableSeams = append(executableSeams, seam)
		}
	}
	if len(executableSeams) != 1 {
		t.Fatalf("executable seam count = %d", len(executableSeams))
	}
	hits, err := ScanSeams(root, executableSeams)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != executableSeams[0].Budget {
		t.Fatalf("executable observation = %d, budget = %d", len(hits), executableSeams[0].Budget)
	}
	retiredExecutable := strings.Join([]string{"vrooli", "debt", "census"}, "-")
	for _, hit := range hits {
		if hit.Path == retiredExecutable {
			t.Fatal("retired census executable remains in observation")
		}
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
	writeSeamTestFile(t, root, repositorySeamsPath, `{"schemaVersion":"1.0.0","unknown":true,"seams":[]}`)
	if _, err := LoadSeams(root); err == nil {
		t.Fatal("expected unknown field to fail strict decoding")
	}
}

// [REQ:TM-LS-009]
func TestLoadSeamsRejectsTrailingJSON(t *testing.T) {
	root := t.TempDir()
	writeSeamTestFile(t, root, repositorySeamsPath, `{"schemaVersion":"1.0.0","seams":[]} {}`)
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
	if len(seams) != 38 {
		t.Fatalf("expected thirty-eight declared seams, got %d", len(seams))
	}
	if _, err := ScanSeams(root, seams); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryScenarioSeamResolution(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	targetRoot := filepath.Join(repositoryRoot, "scenarios", "tidiness-manager")
	seams, resolution, err := ResolveSeams(targetRoot, SeamTargetScenario)
	if err != nil {
		t.Fatal(err)
	}
	if len(seams) == 0 || len(resolution.Files) != 1 || resolution.ScanRoot != targetRoot {
		t.Fatalf("repository scenario resolution: seams=%d resolution=%#v", len(seams), resolution)
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
