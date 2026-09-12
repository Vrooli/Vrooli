package validation

import (
	"context"
	"testing"

	"audio-tools/internal/soak"
	"audio-tools/internal/stt/session"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
)

func TestValidateScenarioDoesNotExecuteWhenExecutionIsNotRequested(t *testing.T) {
	runs := 0
	h := &handler{
		deps: Deps{RunSoak: func(context.Context, soak.Options, *session.Registry) (soak.Result, error) {
			runs++
			return soak.Result{}, nil
		}},
		spec: testSpec(),
	}

	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "audio-tools",
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_SKIPPED, response.Msg.GetStatus())
	require.Zero(t, runs)
}

func TestValidateScenarioFailsClosedWhenQualificationInputsAreMissing(t *testing.T) {
	runs := 0
	h := &handler{
		deps: Deps{
			ScenarioDir: t.TempDir(),
			RunSoak: func(context.Context, soak.Options, *session.Registry) (soak.Result, error) {
				runs++
				return soak.Result{}, nil
			},
		},
		spec: testSpec(),
	}

	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario:         "audio-tools",
		IncludeExecution: true,
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	require.Contains(t, response.Msg.GetAssessment().GetFindings()[0].GetMessage(), "VROOLI_AUDIO_SOAK_REPLAY=1")
	require.Zero(t, runs)
}

func testSpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "audio-tools",
		Phase:    "soak",
		Version:  "1.0.0",
		Capabilities: []assessment.CapabilitySpec{{
			ID: "qualification", Label: "Qualification",
			Levels: []assessment.Level{
				{ID: "L0", Name: "Unavailable", Description: "Unavailable", EntryCriteria: []string{}, ExitCriteria: []string{"artifact"}},
				{ID: "L1", Name: "Qualified", Description: "Qualified", EntryCriteria: []string{"artifact"}, ExitCriteria: []string{}},
			},
		}},
		Findings: map[string]assessment.FindingMapping{
			"SOAK_CONFIGURATION_MISSING": {
				CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactFoundationBlocker,
				Dimension: "tests", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required",
				FixClass: assessment.FixClassManual, FixReason: "test input is deployment-specific",
			},
			"SOAK_QUALIFICATION_FAILED": {
				CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactSafetyBlocker,
				Dimension: "performance", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required",
				FixClass: assessment.FixClassManual, FixReason: "artifact requires investigation",
			},
		},
		Fallback: assessment.FallbackPolicy{CapabilityID: "qualification", LocalLevelImpact: "L0", GlobalImpact: assessment.ImpactUnknown, Dimension: "tests", SeverityDefault: "SEVERITY_ERROR", CleanRequirement: "required"},
	}
}
