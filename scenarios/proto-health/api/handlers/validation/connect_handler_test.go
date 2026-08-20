package validation

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"proto-health/internal/protosurface"
	internal "proto-health/internal/validation"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

type fakeValidator struct {
	report internal.Report
	path   string
}

func (f fakeValidator) ValidateScenario(ctx context.Context, scenario string) (internal.Report, error) {
	if f.path != "" && internal.ScenarioPathFrom(ctx) != f.path {
		return internal.Report{}, fmt.Errorf("scenario path = %q, want %q", internal.ScenarioPathFrom(ctx), f.path)
	}
	return f.report, nil
}

func (f fakeValidator) DescribeScenarioProtos(context.Context, string) (protosurface.Surface, error) {
	return protosurface.Surface{}, nil
}

func (f fakeValidator) DescribeScenariosProtos(context.Context, []string, int32, string) ([]internal.SurfaceResult, error) {
	return nil, nil
}

func protoHealthSpec(t *testing.T) *assessment.Spec {
	t.Helper()
	spec, err := assessment.LoadSpecFromScenario("../../..")
	require.NoError(t, err)
	return spec
}

func TestValidateScenarioAttachesMetrics(t *testing.T) {
	handler := NewConnectHandler(Deps{
		Validator:    fakeValidator{report: internal.Report{Scenario: "demo", Passed: true}},
		MaturitySpec: protoHealthSpec(t),
	})

	resp, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{Scenario: "demo"}))
	require.NoError(t, err)

	m := resp.Msg.GetMetrics()
	require.NotNil(t, m, "metrics must be attached to the response")
	require.GreaterOrEqual(t, m.GetWallClockMs(), int64(0))
	require.NotNil(t, m.GetEnvironment())
	require.Equal(t, runtime.GOOS, m.GetEnvironment().GetOs())
	require.Equal(t, runtime.GOARCH, m.GetEnvironment().GetArch())
	require.Equal(t, int32(runtime.NumCPU()), m.GetEnvironment().GetNumCpu())
}

func TestValidateScenarioForwardsExplicitScenarioPath(t *testing.T) {
	const path = "/tmp/template-workspace/scenarios/demo"
	handler := NewConnectHandler(Deps{
		Validator:    fakeValidator{report: internal.Report{Scenario: "demo", Passed: true}, path: path},
		MaturitySpec: protoHealthSpec(t),
	})

	_, err := handler.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
		Path:     path,
	}))
	require.NoError(t, err)
}

func TestBuildMaturityAssessmentMapsProtoFindings(t *testing.T) {
	spec := &assessment.Spec{
		Provider: "proto-health",
		Phase:    "proto",
		Version:  "1.0.0",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No proto surface"},
			{ID: "L1", Name: "Organized proto packages"},
			{ID: "L2", Name: "Generated artifacts synchronized"},
		},
		Findings: map[string]assessment.FindingMapping{
			internal.CodeGenOutOfSync: {
				LocalLevelImpact:    "L2",
				GlobalImpact:        assessment.ImpactFoundationBlocker,
				Dimension:           "proto-health",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"proto-contract-audit"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     assessment.ImpactUnknown,
			Dimension:        "proto-health",
			SeverityDefault:  "SEVERITY_WARNING",
		},
	}
	require.NoError(t, assessment.ValidateSpec(*spec))

	got, err := buildMaturityAssessment(internal.Report{
		Scenario: "demo",
		Findings: []internal.Finding{{
			Code:       internal.CodeGenOutOfSync,
			Severity:   internal.SeverityError,
			Location:   "packages/proto/gen",
			Message:    "generated proto artifacts are out of sync",
			Suggestion: "run cd packages/proto && GOFLAGS=-mod=mod go run ./cmd/protogen generate --scenario demo",
		}},
	}, spec)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "proto-health", got.GetProvider())
	require.Equal(t, "proto", got.GetPhase())
	require.Equal(t, "L1", got.GetLocal().GetCurrentLevel())
	require.Equal(t, "L2", got.GetLocal().GetNextLevel())
	require.Equal(t, []string{internal.CodeGenOutOfSync}, got.GetLocal().GetBlockingFindingCodes())
	require.EqualValues(t, 1, got.GetFindingsByGlobalImpact()["foundation_blocker"])
	require.Equal(t, []string{"proto-contract-audit"}, got.GetRecommendedSkillIds())
	require.Equal(t, "proto-health", got.GetFindings()[0].GetMaturity().GetDimension())
}
