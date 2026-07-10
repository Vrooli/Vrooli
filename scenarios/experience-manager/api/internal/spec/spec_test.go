package spec

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestReportCarriesParserContractFields(t *testing.T) {
	report := Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []Finding{{
			Code:       "experience.schema_invalid",
			Severity:   "SEVERITY_ERROR",
			Message:    "invalid",
			Locations:  []string{"experience/index.json"},
			Suggestion: "repair schema",
		}},
		DegradedReason: "design contract absent",
	}

	if report.Scenario != "demo" || report.TargetPath == "" {
		t.Fatalf("report identity not preserved: %+v", report)
	}
	if got := report.Findings[0].Code; got != "experience.schema_invalid" {
		t.Fatalf("finding code = %q", got)
	}
	if report.DegradedReason == "" {
		t.Fatal("degraded reason should be preserved")
	}
}

func TestParseScenarioFixturesContractGreen(t *testing.T) { // [REQ:EXPERIEN-P0-001]
	root := repoRoot(t)
	for _, scenario := range []string{"experience-manager", "business-health", "web-console", "react-component-library"} {
		t.Run(scenario, func(t *testing.T) {
			report, err := ParseScenario(filepath.Join(root, "scenarios", scenario))
			if err != nil {
				t.Fatalf("ParseScenario: %v", err)
			}
			if len(report.Findings) > 0 {
				t.Fatalf("expected contract-green fixture, got findings: %+v", report.Findings)
			}
			if report.Spec == nil || report.Spec.Index.Scenario != scenario {
				t.Fatalf("parsed spec identity mismatch: %+v", report.Spec)
			}
		})
	}
}

func TestScenarioExperienceDocumentsValidateAgainstJSONSchema(t *testing.T) { // [REQ:EXPERIEN-P0-001]
	root := repoRoot(t)
	schemaBytes, err := os.ReadFile(filepath.Join(root, ".vrooli", "schemas", "scenario-experience-spec.schema.json"))
	if err != nil {
		t.Fatalf("read experience schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("scenario-experience-spec/v1", bytes.NewReader(schemaBytes)); err != nil {
		t.Fatalf("add schema resource: %v", err)
	}
	schema, err := compiler.Compile("scenario-experience-spec/v1")
	if err != nil {
		t.Fatalf("compile experience schema: %v", err)
	}

	for _, scenario := range []string{"experience-manager", "business-health", "web-console", "react-component-library"} {
		experienceRoot := filepath.Join(root, "scenarios", scenario, "experience")
		err := filepath.WalkDir(experienceRoot, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".json") {
				return nil
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var doc any
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("%s is not JSON: %v", path, err)
			}
			if err := schema.Validate(doc); err != nil {
				t.Fatalf("%s fails scenario-experience-spec.schema.json: %v", path, err)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s experience docs: %v", scenario, err)
		}
	}
}

func TestParseScenarioPreservesSpikeExtensions(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "business-health"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	matrix := report.Spec.Pages["matrix"]
	if _, ok := matrix.Extensions["x-spike"]; !ok {
		t.Fatalf("matrix x-spike extension was not preserved: %#v", matrix.Extensions)
	}
}

func TestParseScenarioPreservesClaimExtensions(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "web-console"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	workspace := report.Spec.Pages["workspace"]
	for _, claim := range workspace.Claims {
		if _, ok := claim.Extensions["x-display-mode"]; !ok {
			continue
		}
		var mode string
		if err := json.Unmarshal(claim.Extensions["x-display-mode"], &mode); err != nil {
			t.Fatalf("x-display-mode is not a string: %v", err)
		}
		if mode != "tabs" {
			t.Fatalf("x-display-mode = %q, want tabs", mode)
		}
		return
	}
	t.Fatalf("workspace claim-level x-display-mode extension was not preserved")
}

func TestParseScenarioComputesDepths(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "experience-manager"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if got := report.PageDepths["studio"]; got != 4 {
		t.Fatalf("studio depth = L%d, want L4", got)
	}
	if got := report.PageDepths["findings"]; got < 3 {
		t.Fatalf("findings depth = L%d, want at least L3", got)
	}
}

func TestParseScenarioParsesComponentDocuments(t *testing.T) {
	report, err := ParseScenario(filepath.Join(repoRoot(t), "scenarios", "react-component-library"))
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if len(report.Findings) > 0 {
		t.Fatalf("expected component pilot to parse cleanly, got findings: %+v", report.Findings)
	}
	button, ok := report.Spec.Components["button"]
	if !ok {
		t.Fatalf("button component not parsed: %+v", report.Spec.Components)
	}
	if button.Kind != kindComponent || button.Component.ExamplesRef == "" {
		t.Fatalf("button component identity not preserved: %+v", button)
	}
	if got := report.ComponentDepths["button"]; got != 3 {
		t.Fatalf("button component depth = L%d, want L3", got)
	}
}

func TestParseScenarioFindsContractViolations(t *testing.T) { // [REQ:EXPERIEN-P0-001]
	root := t.TempDir()
	scenario := filepath.Join(root, "demo")
	mustWrite(t, filepath.Join(scenario, "PRD.md"), "## Operational Targets\n- [ ] OT-P0-001 | Demo | Demo target\n")
	mustWrite(t, filepath.Join(scenario, "experience", "index.json"), `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [{"id":"home","path":"pages/home.json","status":"active"}],
  "journeys": [{"id":"journey","path":"journeys/journey.json","status":"active"}],
  "components": [{"id":"button","path":"components/button.json","status":"active"}]
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "pages", "home.json"), `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"A sufficiently long purpose.","prd_refs":["OT-P9-999"]},
  "states": [{"id":"default"}],
  "elements": [{"id":"known","role":"button"}],
  "claims": [
    {"id":"bad-custom","type":"custom","statement":"This custom claim is invalid as machine.","tier":"machine","elements":["known"]},
    {"id":"missing-ref","type":"element-present","statement":"This points at a missing element.","tier":"machine","elements":["ghost"],"states":["ghost-state"]}
  ],
  "bindings": {"elements": {"ghost-binding": {"testid":"ghost"}}}
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "journeys", "journey.json"), `{
  "kind": "experience-journey",
  "schemaVersion": "1.0.0",
  "journey": {"id":"journey","title":"Journey","purpose":"A sufficiently long journey purpose."},
  "steps": [{"page":"home","state":"missing","intent":"A sufficiently long step intent."}]
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "components", "button.json"), `{
  "kind": "experience-component",
  "schemaVersion": "1.1.0",
  "component": {"id":"button","title":"Button","purpose":"A sufficiently long component purpose.","examplesRef":"../../library/components/Button/versions/1.2.0/examples.json"},
  "states": [{"id":"default","example":"ghost-example"}],
  "elements": [{"id":"known","role":"button"}],
  "claims": [
    {"id":"component-missing-ref","type":"element-present","statement":"This points at missing component refs.","tier":"machine","elements":["ghost"],"states":["ghost-state"]}
  ],
  "bindings": {"elements": {"known": {"testid":"button"}}}
}`)

	report, err := ParseScenario(scenario)
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	for _, code := range []string{CodePRDRefUnmatched, CodeTierViolation, CodeRefUnresolved, CodeBindingOrphan} {
		if !hasCode(report.Findings, code) {
			t.Fatalf("missing %s in findings: %+v", code, report.Findings)
		}
	}
}

func TestIndexParityFindingHasGoldenAssertion(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "demo")
	mustWrite(t, filepath.Join(scenario, "experience", "index.json"), `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "pages": [],
  "journeys": []
}`)
	mustWrite(t, filepath.Join(scenario, "experience", "pages", "home.json"), `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {"id":"home","title":"Home","routes":["/"],"purpose":"Home page."},
  "states": [{"id":"default"}],
  "elements": [],
  "claims": [],
  "bindings": {"elements": {}}
}`)

	report, err := ParseScenario(scenario)
	if err != nil {
		t.Fatalf("ParseScenario: %v", err)
	}
	if len(report.Findings) != 1 {
		t.Fatalf("findings = %d, want 1: %+v", len(report.Findings), report.Findings)
	}
	finding := report.Findings[0]
	if finding.Code != CodeIndexParity || finding.Severity != SeverityError {
		t.Fatalf("finding = %+v, want index_parity error", finding)
	}
	if !strings.Contains(finding.Message, "not listed in index") {
		t.Fatalf("message = %q", finding.Message)
	}
}

func TestDoctrineRegistryTableMatchesAllFindingCodes(t *testing.T) {
	doc, err := os.ReadFile(filepath.Join(repoRoot(t), "docs", "reference", "experience-alignment.md"))
	if err != nil {
		t.Fatalf("read doctrine: %v", err)
	}
	re := regexp.MustCompile("`(experience\\.[a-z_]+)`")
	got := map[string]bool{}
	inRegistry := false
	for _, line := range strings.Split(string(doc), "\n") {
		if strings.HasPrefix(line, "## The invariants") {
			inRegistry = true
			continue
		}
		if inRegistry && strings.HasPrefix(line, "## ") {
			break
		}
		if !inRegistry || !strings.HasPrefix(line, "| `experience.") {
			continue
		}
		match := re.FindStringSubmatch(line)
		if len(match) == 2 {
			got[match[1]] = true
		}
	}
	assertCodeSet(t, "doctrine registry", keys(got), AllFindingCodes)
}

func TestFindingDocsMatchAllFindingCodes(t *testing.T) {
	dir := filepath.Join(repoRoot(t), "scenarios", "experience-manager", "docs", "findings")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read finding docs: %v", err)
	}
	var got []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		code := strings.TrimSuffix(entry.Name(), ".md")
		if strings.HasPrefix(code, "experience.") {
			got = append(got, code)
		}
	}
	assertCodeSet(t, "finding docs", got, AllFindingCodes)
}

func TestAllFindingCodesAreUnique(t *testing.T) {
	assertCodeSet(t, "all finding codes", AllFindingCodes, AllFindingCodes)
}

func TestProductionExperienceFindingLiteralsAreRegistered(t *testing.T) {
	root := filepath.Join(repoRoot(t), "scenarios", "experience-manager", "api", "internal")
	re := regexp.MustCompile(`"((?:experience)\.[a-z_]+)"`)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, match := range re.FindAllStringSubmatch(string(data), -1) {
			if !IsFindingCode(match[1]) {
				t.Fatalf("%s contains unregistered experience finding literal %q", path, match[1])
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk production sources: %v", err)
	}
}

func TestReportAddRejectsUnregisteredFindingCode(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Report.add accepted unregistered finding code")
		}
	}()
	var report Report
	report.add("experience.not_registered", SeverityWarning, "bad", "experience/index.json", "register the code")
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func assertCodeSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("%s codes mismatch\n got: %v\nwant: %v", label, got, want)
	}
	for i := 1; i < len(got); i++ {
		if got[i] == got[i-1] {
			t.Fatalf("%s contains duplicate code %q", label, got[i])
		}
	}
}

func keys(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for key := range in {
		out = append(out, key)
	}
	return out
}
