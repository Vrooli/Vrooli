package validation

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/maturity-go/autofix"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	internalvalidation "api-health/internal/validation"
)

type stubValidator struct {
	report           internalvalidation.Report
	includeExecution bool
	scenario         string
	path             string
}

func (s *stubValidator) ValidateScenario(_ context.Context, scenario, path string, includeExecution bool) (internalvalidation.Report, error) {
	s.includeExecution = includeExecution
	s.scenario = scenario
	s.path = path
	return s.report, nil
}

type stubFixers struct {
	candidates []autofix.Candidate
}

func (s stubFixers) Preview(string, []string) ([]autofix.Candidate, error) {
	return s.candidates, nil
}

func (s stubFixers) Apply(string, []string) ([]autofix.Candidate, error) {
	out := append([]autofix.Candidate(nil), s.candidates...)
	for i := range out {
		out[i].Applied = true
	}
	return out, nil
}

func TestValidateScenarioBuildsSharedResponse(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(findRepoRoot(t), "scenarios", "api-health"))
	require.NoError(t, err)
	validator := &stubValidator{report: internalvalidation.Report{
		Scenario: "demo",
		Target: internalvalidation.Target{
			Scenario:   "demo",
			RootPath:   "/tmp/demo",
			Resolution: internalvalidation.ResolutionResolved,
			APIKind:    internalvalidation.APIKindGo,
			HasAPIDir:  true,
		},
		Passed: true,
	}}
	h := NewConnectHandler(Deps{
		Validator:    validator,
		MaturitySpec: spec,
	})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo", IncludeExecution: true}))
	require.NoError(t, err)
	require.Equal(t, "demo", resp.Msg.GetScenario())
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_PASSED, resp.Msg.GetStatus())
	require.NotNil(t, resp.Msg.GetAssessment())
	require.Equal(t, "api-health", resp.Msg.GetAssessment().GetProvider())
	require.Equal(t, "api", resp.Msg.GetAssessment().GetPhase())
	require.NotNil(t, resp.Msg.GetMetrics())
	require.NotNil(t, resp.Msg.GetNativeDetail())

	var detail structpb.Struct
	require.NoError(t, resp.Msg.GetNativeDetail().UnmarshalTo(&detail))
	require.Equal(t, "demo", detail.Fields["scenario"].GetStringValue())
	require.True(t, validator.includeExecution)
}

func TestValidateScenarioMapsTargetFinding(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(findRepoRoot(t), "scenarios", "api-health"))
	require.NoError(t, err)
	h := NewConnectHandler(Deps{
		Validator: &stubValidator{report: internalvalidation.Report{
			Scenario: "missing",
			Target:   internalvalidation.Target{Scenario: "missing", RootPath: "/missing", Resolution: internalvalidation.ResolutionMissing},
			Findings: []internalvalidation.Finding{{
				Severity: internalvalidation.SeverityError,
				Code:     internalvalidation.CodeTargetUnresolved,
				Title:    "Target unresolved",
				Location: "/missing",
				Message:  "missing",
			}},
			Summary: internalvalidation.Summary{Errors: 1},
		}},
		MaturitySpec: spec,
	})

	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "missing"}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, resp.Msg.GetStatus())
	require.Len(t, resp.Msg.GetAssessment().GetFindings(), 1)
	require.Equal(t, internalvalidation.CodeTargetUnresolved, resp.Msg.GetAssessment().GetFindings()[0].GetCode())
}

func TestValidateTargetPreservesControlPlaneRoot(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(findRepoRoot(t), "scenarios", "api-health"))
	require.NoError(t, err)
	validator := &stubValidator{report: internalvalidation.Report{
		Scenario: "control-plane",
		Target:   internalvalidation.Target{Scenario: "control-plane", RootPath: "/repo", Resolution: internalvalidation.ResolutionResolved},
		Passed:   true,
	}}
	h := NewConnectHandler(Deps{Validator: validator, MaturitySpec: spec})
	target := &commonv1.ValidationTarget{
		Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_CONTROL_PLANE,
		Id:   "control-plane",
		Root: "repo",
	}
	resp, err := h.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Target: target, Path: "/repo", IncludeExecution: true,
	}))
	require.NoError(t, err)
	require.Equal(t, target, resp.Msg.GetTarget())
	require.Equal(t, "/repo", validator.path)
	require.Equal(t, "control-plane", validator.scenario)
	require.True(t, validator.includeExecution)
}

func TestPreviewAndApplyFixMapCandidates(t *testing.T) {
	validator := &stubValidator{report: internalvalidation.Report{
		Scenario: "demo",
		Target:   internalvalidation.Target{Scenario: "demo", RootPath: "/tmp/demo", Resolution: internalvalidation.ResolutionResolved},
	}}
	h := NewConnectHandler(Deps{
		Validator: validator,
		Fixers: stubFixers{candidates: []autofix.Candidate{{
			RuleID:      internalvalidation.CodeRawStatusCode,
			FilePath:    "/tmp/demo/api/main.go",
			Description: "replace raw status",
			Before:      "w.WriteHeader(404)",
			After:       "w.WriteHeader(http.StatusNotFound)",
		}}},
	})

	preview, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.False(t, preview.Msg.GetApplied())
	require.Len(t, preview.Msg.GetCandidates(), 1)
	require.False(t, preview.Msg.GetCandidates()[0].GetApplied())
	require.Equal(t, internalvalidation.CodeRawStatusCode, preview.Msg.GetCandidates()[0].GetRuleId())

	applied, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "demo"}))
	require.NoError(t, err)
	require.True(t, applied.Msg.GetApplied())
	require.True(t, applied.Msg.GetCandidates()[0].GetApplied())
}

func TestPreviewFixReportsNoCandidates(t *testing.T) {
	h := NewConnectHandler(Deps{
		Validator: &stubValidator{report: internalvalidation.Report{
			Scenario: "clean",
			Target:   internalvalidation.Target{Scenario: "clean", RootPath: "/tmp/clean", Resolution: internalvalidation.ResolutionResolved},
		}},
		Fixers: stubFixers{},
	})

	resp, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "clean"}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetCandidates())
	require.Contains(t, resp.Msg.GetMessages()[0], "no deterministic API Health fixes")
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs(".")
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(abs, ".vrooli", "repo-contract.json")); err == nil {
			return abs
		}
		parent := filepath.Dir(abs)
		require.NotEqual(t, abs, parent, "repo root not found")
		abs = parent
	}
}
