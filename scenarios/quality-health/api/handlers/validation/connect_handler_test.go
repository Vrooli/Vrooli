package validation

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"connectrpc.com/connect"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/surfaces"

	"github.com/vrooli/maturity-go/assessment"
	autofixcore "github.com/vrooli/maturity-go/autofix"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

// stubAuditor satisfies the Auditor interface for handler tests without spinning
// up CodeFacts or real filesystem discovery.
type stubAuditor struct {
	previewCandidates []autofixcore.Candidate
	applyCandidates   []autofixcore.Candidate
}

func (s *stubAuditor) Audit(_ context.Context, req internalaudit.Request) (internalaudit.Response, error) {
	return internalaudit.Response{
		RunID:  "qh-stub",
		Status: "passed",
		Inventory: surfaces.Inventory{
			Scenario:   req.Scenario,
			TargetKind: "scenario",
		},
		Maturity: internalaudit.Maturity{Rung: 3, Label: "L3"},
	}, nil
}

func (s *stubAuditor) PreviewFix(_ context.Context, scenario, _ string, _ []string) (surfaces.Inventory, []autofixcore.Candidate, error) {
	return surfaces.Inventory{Scenario: scenario}, s.previewCandidates, nil
}

func (s *stubAuditor) ApplyFix(_ context.Context, scenario, _ string, _ []string) (surfaces.Inventory, []autofixcore.Candidate, error) {
	return surfaces.Inventory{Scenario: scenario}, s.applyCandidates, nil
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	h := NewConnectHandler(Deps{
		Auditor:      &stubAuditor{},
		MaturitySpec: testMaturitySpec(t),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "quality-health"}))
	if err != nil {
		t.Fatalf("ValidateScenario: %v", err)
	}
	m := resp.Msg.GetMetrics()
	if m == nil {
		t.Fatal("metrics must be attached to the response")
	}
	if m.GetWallClockMs() < 0 {
		t.Fatalf("wall clock must be non-negative, got %d", m.GetWallClockMs())
	}
	env := m.GetEnvironment()
	if env == nil {
		t.Fatal("metrics environment must be populated with the stdlib baseline")
	}
	if env.GetOs() != runtime.GOOS {
		t.Fatalf("env os = %q, want %q", env.GetOs(), runtime.GOOS)
	}
	if env.GetArch() != runtime.GOARCH {
		t.Fatalf("env arch = %q, want %q", env.GetArch(), runtime.GOARCH)
	}
	if env.GetNumCpu() != int32(runtime.NumCPU()) {
		t.Fatalf("env num_cpu = %d, want %d", env.GetNumCpu(), runtime.NumCPU())
	}
}

func TestPreviewFixMapsCandidates(t *testing.T) {
	h := NewConnectHandler(Deps{Auditor: &stubAuditor{previewCandidates: []autofixcore.Candidate{
		{RuleID: "TSCONFIG_STRICT", FilePath: "ui/tsconfig.json", Description: "enable strict", Before: "a", After: "b"},
	}}})
	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "quality-health"}))
	if err != nil {
		t.Fatalf("PreviewFix: %v", err)
	}
	if resp.Msg.GetApplied() {
		t.Fatalf("preview applied = true, want false")
	}
	if len(resp.Msg.GetCandidates()) != 1 || resp.Msg.GetCandidates()[0].GetRuleId() != "TSCONFIG_STRICT" {
		t.Fatalf("unexpected candidates: %+v", resp.Msg.GetCandidates())
	}
}

func TestApplyFixEmptyStampsMessage(t *testing.T) {
	h := NewConnectHandler(Deps{Auditor: &stubAuditor{}})
	resp, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "quality-health"}))
	if err != nil {
		t.Fatalf("ApplyFix: %v", err)
	}
	if !resp.Msg.GetApplied() {
		t.Fatalf("apply applied = false, want true")
	}
	if len(resp.Msg.GetMessages()) == 0 {
		t.Fatalf("expected empty-set message")
	}
}

func TestFixRequiresTarget(t *testing.T) {
	h := NewConnectHandler(Deps{Auditor: &stubAuditor{}})
	if _, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{})); err == nil {
		t.Fatalf("expected error for missing scenario/path")
	}
}

func testMaturitySpec(t *testing.T) *assessment.Spec {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("load descriptor maturity: %v", err)
	}
	return spec
}
