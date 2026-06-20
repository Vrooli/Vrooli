package validation

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"structure-health/internal/profile"
	internalvalidation "structure-health/internal/validation"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
)

// fakeDescriber returns canned code facts so handler tests do not depend on a
// running Code Facts service or a real scenario on disk.
type fakeDescriber struct{}

func (fakeDescriber) Describe(_ context.Context, scenario, _ string) (profile.Facts, error) {
	return profile.Facts{
		Scenario: scenario,
		RootPath: "/tmp/" + scenario,
		Surfaces: []profile.Surface{
			{ID: "api", Kind: "api", Language: "go"},
			{ID: "ui", Kind: "ui", Framework: "react-vite"},
		},
	}, nil
}

func testSpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "structure-health",
		Phase:    "structure",
		Version:  "1.0.0",
		Levels: []assessment.Level{
			{ID: "L0", Name: "base"},
			{ID: "L1", Name: "skeleton"},
			{ID: "L2", Name: "wiring"},
			{ID: "L3", Name: "serving"},
			{ID: "L4", Name: "profile"},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L4",
			GlobalImpact:     assessment.ImpactEvolvabilityGap,
			Dimension:        "structure",
			SeverityDefault:  "SEVERITY_WARNING",
			CleanRequirement: "advisory",
		},
	}
}

func newTestHandler() *Handler {
	svc := internalvalidation.New()
	spec := testSpec()
	svc.Spec = spec
	svc.Facts = fakeDescriber{}
	return NewHandlerWithDeps(Deps{Service: svc, MaturitySpec: spec})
}

// [REQ:SH-BOUND-001] [REQ:SH-BOUND-002]
func TestNativeValidateScenarioStub(t *testing.T) {
	h := newTestHandler()
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	msg := resp.Msg
	if msg.GetStatus() == "" {
		t.Fatal("status must be set")
	}
	if msg.GetScenario() != "demo" {
		t.Fatalf("scenario = %q, want demo", msg.GetScenario())
	}
	if msg.GetProfile() == nil || msg.GetProfile().GetId() == "" {
		t.Fatalf("profile must be set: %+v", msg.GetProfile())
	}
	if msg.GetAssessment() == nil {
		t.Fatal("assessment must be present")
	}
}

func TestNativeValidateScenarioRequiresTarget(t *testing.T) {
	h := newTestHandler()
	_, err := h.ValidateScenario(context.Background(), connect.NewRequest(&validationv1.ValidateScenarioRequest{}))
	if err == nil {
		t.Fatal("expected error when scenario and path are empty")
	}
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

// [REQ:SH-BOUND-001] [REQ:SH-FIX-004]
func TestSharedValidateScenarioStub(t *testing.T) {
	shared := NewSharedHandler(newTestHandler())
	resp, err := shared.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("shared ValidateScenario: %v", err)
	}
	msg := resp.Msg
	if msg.GetStatus() == scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_UNSPECIFIED {
		t.Fatalf("status must be a concrete verdict, got %v", msg.GetStatus())
	}
	if msg.GetAssessment() == nil {
		t.Fatal("assessment must be present in shared response")
	}
	if msg.GetNativeDetail() == nil {
		t.Fatal("native_detail must be packed in shared response")
	}
}

// rootedDescriber points validation at a real on-disk scenario root so the
// auto-fix handlers can read + edit its service.json.
type rootedDescriber struct{ root string }

func (d rootedDescriber) Describe(_ context.Context, scenario, _ string) (profile.Facts, error) {
	return profile.Facts{Scenario: scenario, RootPath: d.root}, nil
}

func newFixHandler(root string) *Handler {
	svc := internalvalidation.New()
	spec := testSpec()
	svc.Spec = spec
	svc.Facts = rootedDescriber{root: root}
	return NewHandlerWithDeps(Deps{Service: svc, MaturitySpec: spec})
}

func writeFixScenario(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	body := `{
  "service": {
    "name": "wrong-name"
  },
  "ports": {
    "api": {
      "env_var": "API_PORT",
      "range": "15000-19999"
    }
  },
  "lifecycle": {
    "setup": {
      "steps": []
    }
  }
}
`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return root
}

// [REQ:SH-FIX-001]
func TestPreviewFixConfigIsDryRun(t *testing.T) {
	root := writeFixScenario(t)
	path := filepath.Join(root, ".vrooli", "service.json")
	before, _ := os.ReadFile(path)

	h := newFixHandler(root)
	resp, err := h.PreviewFixConfig(context.Background(), connect.NewRequest(&validationv1.FixConfigRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("PreviewFixConfig: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatal("preview must report applied=false")
	}
	if len(resp.Msg.GetCandidates()) == 0 {
		t.Fatal("expected candidates")
	}
	for _, c := range resp.Msg.GetCandidates() {
		if c.GetApplied() {
			t.Fatalf("preview candidate %s must not be applied", c.GetRuleId())
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("preview must not modify disk")
	}
}

// [REQ:SH-FIX-001]
func TestApplyFixConfigWrites(t *testing.T) {
	root := writeFixScenario(t)
	h := newFixHandler(root)
	resp, err := h.ApplyFixConfig(context.Background(), connect.NewRequest(&validationv1.FixConfigRequest{Scenario: "demo"}))
	if err != nil {
		t.Fatalf("ApplyFixConfig: %v", err)
	}
	if !resp.Msg.GetApplied() {
		t.Fatal("apply must report applied=true")
	}
	raw, _ := os.ReadFile(filepath.Join(root, ".vrooli", "service.json"))
	if !strings.Contains(string(raw), `"name": "`+filepath.Base(root)+`"`) {
		t.Fatalf("name not fixed:\n%s", raw)
	}
}

func TestFixConfigRequiresScenario(t *testing.T) {
	h := newFixHandler(t.TempDir())
	if _, err := h.PreviewFixConfig(context.Background(), connect.NewRequest(&validationv1.FixConfigRequest{})); err == nil {
		t.Fatal("expected invalid-argument error when scenario+path are empty")
	}
}
