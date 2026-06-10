package phases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"test-genie/internal/business"

	reqparsing "test-genie/internal/requirements/parsing"
	reqtypes "test-genie/internal/requirements/types"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func findByCode(t *testing.T, findings []*architecturev1.ArchitectureFinding, code string) *architecturev1.ArchitectureFinding {
	t.Helper()
	for _, f := range findings {
		if f.GetCode() == code {
			return f
		}
	}
	t.Fatalf("no finding with code %q; got %v", code, findingCodes(findings))
	return nil
}

func findingCodes(findings []*architecturev1.ArchitectureFinding) []string {
	out := make([]string, 0, len(findings))
	for _, f := range findings {
		out = append(out, f.GetCode())
	}
	return out
}

func TestBusinessIssueFindingsTypeEachRule(t *testing.T) {
	scenarioDir := "/scenarios/demo"
	issues := []reqtypes.ValidationIssue{
		{Rule: "duplicate_id", RequirementID: "REQ-A-001", FilePath: "/scenarios/demo/requirements/core.json", Message: "duplicate requirement ID", Severity: reqtypes.SeverityError},
		{Rule: "cycle_detection", RequirementID: "REQ-A-002", Message: "cycle detected", Severity: reqtypes.SeverityError},
		{Rule: "orphaned_child", RequirementID: "REQ-A-003", FilePath: "/scenarios/demo/requirements/core.json", Message: "references non-existent child", Severity: reqtypes.SeverityError},
		{Rule: "invalid_reference", RequirementID: "REQ-A-004", FilePath: "/scenarios/demo/requirements/core.json", Message: "validation references non-existent file: api/x_test.go", Severity: reqtypes.SeverityWarning},
	}

	findings := businessIssueFindings("demo", scenarioDir, issues)
	if len(findings) != len(issues) {
		t.Fatalf("expected %d findings, got %d", len(issues), len(findings))
	}

	cases := []struct {
		code     string
		severity architecturev1.FindingSeverity
		location string
	}{
		{"business_duplicate_req_id:REQ-A-001", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, "requirements/core.json"},
		{"business_import_cycle:REQ-A-002", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, ""},
		{"business_orphaned_ref:REQ-A-003", architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR, "requirements/core.json"},
		{"business_validation_ref_missing:REQ-A-004", architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING, "requirements/core.json"},
	}
	for _, tc := range cases {
		f := findByCode(t, findings, tc.code)
		if f.GetSource() != architecturev1.FindingSource_FINDING_SOURCE_BUSINESS {
			t.Errorf("%s: source = %v, want BUSINESS", tc.code, f.GetSource())
		}
		if f.GetSeverity() != tc.severity {
			t.Errorf("%s: severity = %v, want %v", tc.code, f.GetSeverity(), tc.severity)
		}
		if tc.location != "" {
			if len(f.GetLocations()) != 1 || f.GetLocations()[0] != tc.location {
				t.Errorf("%s: locations = %v, want [%s]", tc.code, f.GetLocations(), tc.location)
			}
		}
		if !strings.HasPrefix(f.GetStableId(), "afid:") {
			t.Errorf("%s: stable ID not stamped: %q", tc.code, f.GetStableId())
		}
	}
}

func TestBusinessIssueFindingsUnmappedRuleStillEmits(t *testing.T) {
	issues := []reqtypes.ValidationIssue{
		{Rule: "future_rule", RequirementID: "REQ-X-001", Message: "something new", Severity: reqtypes.SeverityWarning},
	}
	findings := businessIssueFindings("demo", "/scenarios/demo", issues)
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0].GetCode() != "business_future_rule:REQ-X-001" {
		t.Fatalf("unexpected code %q", findings[0].GetCode())
	}
}

func registryIndex(reqs []reqtypes.Requirement, filePath string) *reqparsing.ModuleIndex {
	return &reqparsing.ModuleIndex{
		Modules: []*reqtypes.RequirementModule{
			{FilePath: filePath, Requirements: reqs},
		},
	}
}

func TestBusinessRegistryFindingsNoValidation(t *testing.T) {
	dir := t.TempDir()
	index := registryIndex([]reqtypes.Requirement{
		{ID: "REQ-P0-001", Title: "Critical thing", Criticality: reqtypes.CriticalityP0},
		{ID: "REQ-P1-001", Title: "Normal thing"},
		{ID: "REQ-P1-002", Title: "Validated thing", Validations: []reqtypes.Validation{{Type: "unit", Ref: "api/x_test.go"}}},
	}, filepath.Join(dir, "requirements", "core.json"))

	findings := businessRegistryFindings("demo", dir, index)

	p0 := findByCode(t, findings, "business_req_no_validation:REQ-P0-001")
	if p0.GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR {
		t.Errorf("P0 requirement without validation should be ERROR, got %v", p0.GetSeverity())
	}
	normal := findByCode(t, findings, "business_req_no_validation:REQ-P1-001")
	if normal.GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("non-P0 requirement without validation should be WARNING, got %v", normal.GetSeverity())
	}
	for _, f := range findings {
		if f.GetCode() == "business_req_no_validation:REQ-P1-002" {
			t.Error("validated requirement must not produce a no-validation finding")
		}
		if f.GetSeverity() == architecturev1.FindingSeverity_FINDING_SEVERITY_BLOCKER {
			t.Errorf("business findings must never be BLOCKER in v1 (code %s)", f.GetCode())
		}
	}
}

func TestBusinessRegistryFindingsStarterTemplate(t *testing.T) {
	dir := t.TempDir()
	index := registryIndex([]reqtypes.Requirement{
		{ID: "FOUND-001", Title: "Starter", Tags: []string{"template-starter"}},
		{ID: "FOUND-002", Title: "Starter too", Tags: []string{"template-starter"}, Criticality: reqtypes.CriticalityP0},
	}, filepath.Join(dir, "requirements", "index.json"))

	findings := businessRegistryFindings("demo", dir, index)

	starter := findByCode(t, findings, "business_starter_template")
	if starter.GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("starter template should be WARNING, got %v", starter.GetSeverity())
	}
	if len(starter.GetLocations()) != 1 || starter.GetLocations()[0] != "requirements/index.json" {
		t.Errorf("starter locations = %v", starter.GetLocations())
	}
	// Starter rows are placeholders: no per-requirement findings for them,
	// even the P0-without-validation one.
	if len(findings) != 1 {
		t.Fatalf("starter rows must be excluded from per-requirement checks; got %v", findingCodes(findings))
	}
}

func TestBusinessRegistryFindingsPRDRefUnmatched(t *testing.T) {
	dir := t.TempDir()
	prd := "# Demo\n\n## Operational Targets\n\n- [ ] OT-P0-001 | Works | It works\n- [x] OT-P1-001 | Fast | It is fast\n"
	if err := os.WriteFile(filepath.Join(dir, "PRD.md"), []byte(prd), 0o644); err != nil {
		t.Fatal(err)
	}
	index := registryIndex([]reqtypes.Requirement{
		{ID: "REQ-1", Title: "Linked", PRDRef: "OT-P0-001", Validations: []reqtypes.Validation{{Type: "unit", Ref: "x"}}},
		{ID: "REQ-2", Title: "Dangling", PRDRef: "OT-P0-999", Validations: []reqtypes.Validation{{Type: "unit", Ref: "x"}}},
		{ID: "REQ-3", Title: "Non-OT ref ignored", PRDRef: "section-3", Validations: []reqtypes.Validation{{Type: "unit", Ref: "x"}}},
	}, filepath.Join(dir, "requirements", "core.json"))

	findings := businessRegistryFindings("demo", dir, index)

	dangling := findByCode(t, findings, "business_prd_ref_unmatched:REQ-2")
	if dangling.GetSeverity() != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Errorf("unmatched prd_ref should be WARNING, got %v", dangling.GetSeverity())
	}
	for _, f := range findings {
		if f.GetCode() == "business_prd_ref_unmatched:REQ-1" {
			t.Error("matched prd_ref must not produce a finding")
		}
		if f.GetCode() == "business_prd_ref_unmatched:REQ-3" {
			t.Error("non-OT prd_ref values must be skipped")
		}
	}
}

func TestBusinessRegistryFindingsSkipsPRDCheckWithoutPRD(t *testing.T) {
	dir := t.TempDir() // no PRD.md
	index := registryIndex([]reqtypes.Requirement{
		{ID: "REQ-1", Title: "X", PRDRef: "OT-P0-001", Validations: []reqtypes.Validation{{Type: "unit", Ref: "x"}}},
	}, filepath.Join(dir, "requirements", "core.json"))

	for _, f := range businessRegistryFindings("demo", dir, index) {
		if strings.HasPrefix(f.GetCode(), "business_prd_ref_unmatched") {
			t.Errorf("prd_ref check must be skipped when PRD.md is absent, got %s", f.GetCode())
		}
	}
}

// TestBusinessFindingsStableIDDeterministic pins the afid contract: the same
// input always hashes to the same stable ID, so re-audits reconcile.
func TestBusinessFindingsStableIDDeterministic(t *testing.T) {
	issue := []reqtypes.ValidationIssue{
		{Rule: "duplicate_id", RequirementID: "REQ-A-001", FilePath: "/s/d/requirements/core.json", Message: "dup", Severity: reqtypes.SeverityError},
	}
	a := businessIssueFindings("demo", "/s/d", issue)
	b := businessIssueFindings("demo", "/s/d", issue)
	if a[0].GetStableId() == "" || a[0].GetStableId() != b[0].GetStableId() {
		t.Fatalf("stable IDs differ across identical inputs: %q vs %q", a[0].GetStableId(), b[0].GetStableId())
	}
}

func TestBusinessFindingsNilSafe(t *testing.T) {
	if got := businessFindings("demo", "/tmp", nil); got != nil {
		t.Fatalf("nil result should produce no findings, got %v", findingCodes(got))
	}
	if got := businessFindings("demo", "/tmp", &business.RunResult{}); len(got) != 0 {
		t.Fatalf("empty result should produce no findings, got %v", findingCodes(got))
	}
}
