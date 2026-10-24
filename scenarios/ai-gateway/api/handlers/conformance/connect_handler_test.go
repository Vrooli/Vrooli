package conformance_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	handler "ai-gateway/handlers/conformance"
)

func TestScanScenarioReturnsScannerFindings(t *testing.T) { // [REQ:AIGW-CONFORMANCE-INVENTORY]
	root := t.TempDir()
	path := filepath.Join(root, "api.go")
	require.NoError(t, os.WriteFile(path, []byte(`package api
const url = "https://api.openrouter.ai/v1/chat/completions"
`), 0o644))

	h := handler.NewConnectHandler(handler.Deps{})
	resp, err := h.ScanScenario(context.Background(), connect.NewRequest(&conformancev1.ScanScenarioRequest{
		Scenario: "fixture",
		Path:     root,
	}))
	require.NoError(t, err)
	require.Equal(t, "fixture", resp.Msg.GetScenario())
	require.Equal(t, "blocked-needs-investigation", resp.Msg.GetMaturityLevel())
	require.Len(t, resp.Msg.GetFindings(), 2)
	var found bool
	for _, finding := range resp.Msg.GetFindings() {
		found = found || finding.GetRuleId() == "ai.direct_openrouter_http"
	}
	require.True(t, found)
}

func TestValidateScenarioReturnsSharedAssessment(t *testing.T) { // [REQ:AIGW-CONFORMANCE-MATURITY]
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "api.go"), []byte(`package api
const url = "https://api.openrouter.ai/v1/chat/completions"
`), 0o644))

	h := handler.NewConnectHandler(handler.Deps{MaturitySpec: testMaturitySpec()})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "fixture",
		Path:     root,
	}))
	require.NoError(t, err)
	require.Equal(t, "fixture", resp.Msg.GetScenario())
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, resp.Msg.GetStatus())
	require.Equal(t, "ai-gateway", resp.Msg.GetAssessment().GetProvider())
	require.Equal(t, "ai-conformance", resp.Msg.GetAssessment().GetPhase())
	require.NotNil(t, resp.Msg.GetNativeDetail())
	require.NotNil(t, resp.Msg.GetMetrics())
}

func TestFixRPCsReturnGuidanceWithoutApplying(t *testing.T) {
	h := handler.NewConnectHandler(handler.Deps{MaturitySpec: testMaturitySpec()})

	preview, err := h.PreviewFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "fixture"}))
	require.NoError(t, err)
	require.False(t, preview.Msg.GetApplied())
	require.NotEmpty(t, preview.Msg.GetMessages())

	applied, err := h.ApplyFix(context.Background(), connect.NewRequest(&scenariovalidationv1.FixRequest{Scenario: "fixture"}))
	require.NoError(t, err)
	require.False(t, applied.Msg.GetApplied())
	require.Empty(t, applied.Msg.GetCandidates())
}

func testMaturitySpec() *assessment.Spec {
	return &assessment.Spec{
		Provider: "ai-gateway",
		Phase:    "ai-conformance",
		Version:  "2.0.0",
		Levels: []assessment.Level{
			{ID: "L0", Name: "Inventory"},
			{ID: "L1", Name: "Boundary hygiene"},
			{ID: "L2", Name: "Gateway adoption"},
		},
		Capabilities: []assessment.CapabilitySpec{{
			ID:    "ai_boundary",
			Label: "AI Boundary",
			Levels: []assessment.Level{
				{ID: "L0", Name: "Unknown"},
				{ID: "L1", Name: "Unsafe"},
				{ID: "L2", Name: "Clean"},
			},
		}},
		Findings: map[string]assessment.FindingMapping{
			"ai.direct_openrouter_http": {
				CapabilityID:     "ai_boundary",
				LocalLevelImpact: "L1",
				GlobalImpact:     assessment.ImpactSafetyBlocker,
				Dimension:        "security",
				SeverityDefault:  "SEVERITY_ERROR",
				CleanRequirement: "required",
				FixClass:         assessment.FixClassManual,
				FixReason:        "Migration needs scenario-specific API contract judgment.",
			},
			"ai.gateway_not_adopted": {
				CapabilityID:     "ai_boundary",
				LocalLevelImpact: "L2",
				GlobalImpact:     assessment.ImpactAdvisory,
				Dimension:        "operational-targets",
				SeverityDefault:  "SEVERITY_INFO",
				CleanRequirement: "advisory",
				FixClass:         assessment.FixClassManual,
				FixReason:        "Gateway migration requires scenario-owner review.",
			},
		},
		Fallback: assessment.FallbackPolicy{
			CapabilityID:     "ai_boundary",
			LocalLevelImpact: "L1",
			GlobalImpact:     assessment.ImpactAdvisory,
			Dimension:        "operational-targets",
			SeverityDefault:  "SEVERITY_INFO",
			CleanRequirement: "advisory",
		},
	}
}
