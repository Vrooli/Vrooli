package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"experience-manager/internal/spec"
	contractv1 "github.com/vrooli/vrooli/packages/proto/gen/go/experience-manager/v1/contract"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeEngine struct {
	report spec.Report
	err    error
}

func (e fakeEngine) ValidateScenario(context.Context, string, string) (spec.Report, error) {
	return e.report, e.err
}

func TestSharedValidateScenarioUsesParserReport(t *testing.T) { // [REQ:EXPERIEN-P0-002]
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
	}}})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" {
		t.Fatalf("scenario = %q", resp.Msg.GetScenario())
	}
	if resp.Msg.GetAssessment() == nil {
		t.Fatal("expected maturity assessment")
	}
}

func TestSharedValidateScenarioHonorsExperienceGate(t *testing.T) { // [REQ:EXPERIEN-P0-002]
	report := spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []spec.Finding{{
			Code:      spec.CodeSchemaInvalid,
			Severity:  spec.SeverityError,
			Message:   "bad schema",
			Locations: []string{"experience/index.json"},
		}},
	}
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: report}})

	t.Setenv("EXPERIENCE_ALIGNMENT_GATE", "")
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("required error status = %s, want FAILED", got)
	}

	t.Setenv("EXPERIENCE_ALIGNMENT_GATE", "strict")
	resp, err = h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
	}))
	if err != nil {
		t.Fatalf("ValidateScenario with legacy gate env: %v", err)
	}
	if got := resp.Msg.GetStatus(); got != scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED {
		t.Fatalf("legacy gate env must not rewrite validation truth; status = %s, want FAILED", got)
	}
}

func TestNativeValidateScenarioReturnsStatusFromFindings(t *testing.T) {
	h := NewConnectHandler(Deps{Engine: fakeEngine{report: spec.Report{
		Scenario:   "demo",
		TargetPath: "/tmp/demo",
		Findings: []spec.Finding{{
			Code:      spec.CodeSchemaInvalid,
			Severity:  spec.SeverityError,
			Message:   "bad schema",
			Locations: []string{"experience/index.json"},
		}},
	}}})
	resp, err := h.validateNative(context.Background(), "demo", "")
	if err != nil {
		t.Fatalf("validateNative: %v", err)
	}
	if resp.GetStatus() != "FAILED" {
		t.Fatalf("status = %q, want FAILED", resp.GetStatus())
	}
	if resp.GetReport().GetFindings()[0].GetCode() != spec.CodeSchemaInvalid {
		t.Fatalf("finding = %+v", resp.GetReport().GetFindings()[0])
	}
}

func TestFixRPCsPreviewAndApplyLiveRegistry(t *testing.T) { // [REQ:EXPERIEN-P1-003]
	root := t.TempDir()
	h := NewConnectHandler(Deps{})
	for name, call := range map[string]func() (*connect.Response[scenariovalidationv1.FixResponse], error){
		"preview": func() (*connect.Response[scenariovalidationv1.FixResponse], error) {
			return h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
				Scenario: "demo",
				Path:     root,
				RuleIds:  []string{"experience-fix.case_scaffold"},
			}))
		},
		"apply": func() (*connect.Response[scenariovalidationv1.FixResponse], error) {
			return h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{
				Scenario: "demo",
				Path:     root,
				RuleIds:  []string{"experience-fix.case_scaffold"},
			}))
		},
	} {
		resp, err := call()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if resp.Msg.GetScenario() != "demo" {
			t.Fatalf("%s scenario = %q", name, resp.Msg.GetScenario())
		}
	}
}

func TestScaffoldCasesUsesCaseScaffoldFixer(t *testing.T) { // [REQ:EXPERIEN-P1-002]
	root := fixtureScenarioRoot(t)
	h := NewConnectHandler(Deps{})

	preview, err := h.scaffoldCases(&contractv1.ScaffoldCasesRequest{
		Scenario: "demo",
		Path:     root,
		DryRun:   true,
	})
	if err != nil {
		t.Fatalf("preview scaffold: %v", err)
	}
	if preview.GetApplied() || len(preview.GetDiffs()) != 2 {
		t.Fatalf("preview = applied %v diffs %d, want dry-run with case + registry", preview.GetApplied(), len(preview.GetDiffs()))
	}
	casePath := filepath.Join(root, "bas", "cases", "experience-spec", "home.json")
	if _, err := os.Stat(casePath); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote case file")
	}

	applied, err := h.scaffoldCases(&contractv1.ScaffoldCasesRequest{
		Scenario: "demo",
		Path:     root,
	})
	if err != nil {
		t.Fatalf("apply scaffold: %v", err)
	}
	if !applied.GetApplied() || len(applied.GetDiffs()) != 2 {
		t.Fatalf("apply = applied %v diffs %d, want case + registry", applied.GetApplied(), len(applied.GetDiffs()))
	}
	data, err := os.ReadFile(casePath)
	if err != nil {
		t.Fatalf("read scaffolded case: %v", err)
	}
	if !strings.Contains(string(data), `"spec_entry_id": "home"`) {
		t.Fatalf("scaffolded case missing spec linkage:\n%s", data)
	}
}

func fixtureScenarioRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"PRD.md": `# Fixture

## Operational Targets
- [ ] OT-P0-005 | UI page | Build the page.
`,
		"experience/index.json": `{
  "kind": "experience-index",
  "contract": {"kind": "scenario-experience", "schema": "scenario-experience-spec/v1"},
  "schemaVersion": "1.0.0",
  "scenario": "demo",
  "description": "Fixture experience contract.",
  "pages": [{"id": "home", "path": "pages/home.json", "title": "Home", "status": "active"}],
  "journeys": []
}`,
		"experience/pages/home.json": `{
  "kind": "experience-page",
  "schemaVersion": "1.0.0",
  "page": {
    "id": "home",
    "title": "Home",
    "routes": ["/"],
    "purpose": "Home page proves scaffold generation.",
    "prd_refs": ["OT-P0-005"]
  },
  "priorities": [{"statement": "Primary action first.", "notes": ""}],
  "states": [{"id": "default", "description": "Default state."}],
  "elements": [{"id": "primary", "role": "button", "name": "Primary", "description": "Primary action."}],
  "claims": [{
    "id": "primary-visible",
    "type": "element-present",
    "statement": "Primary action is visible.",
    "tier": "machine",
    "elements": ["primary"],
    "states": ["default"]
  }],
  "bindings": {"elements": {"primary": {"testid": "primary-action"}}},
  "sketch": {"regions": [{"id": "main", "elements": ["primary"]}]}
}`,
		"bas/registry.json": `{"scenario":"demo","metadata":{"execution_mode":"observer"},"playbooks":[]}`,
	}
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
