package conformance_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/maturity-go/assessment"
	conformancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/conformance"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"google.golang.org/protobuf/types/known/structpb"

	handler "ai-gateway/handlers/conformance"
	internalconformance "ai-gateway/internal/conformance"
)

type modelEvalReadinessSourceFunc func(context.Context, internalconformance.ModelEvalRoleDenominators) (internalconformance.ModelEvalReadiness, error)

func (f modelEvalReadinessSourceFunc) Readiness(ctx context.Context, roles internalconformance.ModelEvalRoleDenominators) (internalconformance.ModelEvalReadiness, error) {
	return f(ctx, roles)
}

func aiGatewayScenarioRoot(t *testing.T) string {
	t.Helper()
	for _, variable := range []string{"VROOLI_SCENARIO_DIR", "TEST_GENIE_SCENARIO_DIR"} {
		if root := strings.TrimSpace(os.Getenv(variable)); root != "" && filepath.IsAbs(root) {
			return root
		}
	}
	for _, variable := range []string{"VROOLI_ROOT", "VROOLI_SOURCE_ROOT"} {
		if root := strings.TrimSpace(os.Getenv(variable)); root != "" && filepath.IsAbs(root) {
			candidate := filepath.Join(root, "scenarios", "ai-gateway")
			if _, err := os.Stat(filepath.Join(candidate, ".vrooli", "test-genie.json")); err == nil {
				return candidate
			}
		}
	}
	_, sourceFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	for candidate := filepath.Dir(sourceFile); candidate != filepath.Dir(candidate); candidate = filepath.Dir(candidate) {
		paths, err := filepath.Glob(filepath.Join(candidate, ".vrooli", "test-genie*.json"))
		require.NoError(t, err)
		if len(paths) > 0 {
			return candidate
		}
	}
	t.Fatal("could not locate AI Gateway scenario root from test source")
	return ""
}

func TestModelEvalPhaseReportsMissingEvidenceAndNeverExecutesInference(t *testing.T) {
	scenarioDir := aiGatewayScenarioRoot(t)
	policyPath := filepath.Join(scenarioDir, "../../resources/ollama/model-policy.json")
	taskRoleCatalogPath := filepath.Join(scenarioDir, "config/inference-role-catalog.json")
	readiness := internalconformance.ModelEvalReadiness{
		RegisteredSuites:        35,
		RequiredRetrievalSuites: 35,
		MissingDenominatorIDs:   []string{"knowledge-observatory.docs.starter"},
		RoleCoverage: []internalconformance.RoleCoverage{
			{Role: "chat.default", Cases: 2, Required: 20, Deficient: true},
			{Role: "chat.small", Cases: 0, Required: 20, Deficient: true},
		},
		TaskRoleCoverage: []internalconformance.RoleCoverage{
			{Role: "classify.fast", Cases: 0, Required: 20, Deficient: true},
		},
	}
	h := handler.NewConnectHandler(handler.Deps{
		ScenarioDir: scenarioDir, ModelPolicyPath: policyPath, TaskRoleCatalogPath: taskRoleCatalogPath,
		ModelEvalSource: modelEvalReadinessSourceFunc(func(_ context.Context, roles internalconformance.ModelEvalRoleDenominators) (internalconformance.ModelEvalReadiness, error) {
			require.Len(t, roles.PolicyRoles, 11)
			require.Len(t, roles.TaskRoles, 19)
			return readiness, nil
		}),
	})
	resp, err := h.ValidateTarget(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateTargetRequest{
		Phase:  "model-eval",
		Target: &commonv1.ValidationTarget{Id: "ai-gateway", Kind: commonv1.ValidationTargetKind_VALIDATION_TARGET_KIND_SCENARIO},
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, resp.Msg.GetStatus())
	require.Equal(t, "model_eval_evidence_incomplete", resp.Msg.GetFailureClassification())
	var native structpb.Struct
	require.NoError(t, resp.Msg.GetNativeDetail().UnmarshalTo(&native))
	require.Equal(t, false, native.GetFields()["live_inference_executed"].GetBoolValue())
	require.Equal(t, "blocked_accepted_suite_evidence_incomplete", native.GetFields()["execution_state"].GetStringValue())
	readinessValue := native.GetFields()["readiness"].GetStructValue()
	require.Equal(t, "knowledge-observatory.docs.starter", readinessValue.GetFields()["missing_denominator_suite_ids"].GetListValue().GetValues()[0].GetStringValue())
	messages := make([]string, 0, len(resp.Msg.GetAssessment().GetFindings()))
	for _, finding := range resp.Msg.GetAssessment().GetFindings() {
		messages = append(messages, finding.GetMessage())
	}
	require.Contains(t, strings.Join(messages, "\n"), "chat.default=2/20")
	require.Contains(t, strings.Join(messages, "\n"), "classify.fast=0/20")
}

func TestModelEvalPhaseIdentitySurvivesLegacyScenarioTransport(t *testing.T) {
	scenarioDir := aiGatewayScenarioRoot(t)
	policyPath := filepath.Join(scenarioDir, "../../resources/ollama/model-policy.json")
	taskRoleCatalogPath := filepath.Join(scenarioDir, "config/inference-role-catalog.json")
	h := handler.NewConnectHandler(handler.Deps{
		ScenarioDir: scenarioDir, ModelPolicyPath: policyPath, TaskRoleCatalogPath: taskRoleCatalogPath,
		ModelEvalSource: modelEvalReadinessSourceFunc(func(_ context.Context, roles internalconformance.ModelEvalRoleDenominators) (internalconformance.ModelEvalReadiness, error) {
			return internalconformance.ModelEvalReadiness{
				RegisteredSuites: 35, RequiredRetrievalSuites: 35,
				MissingDenominatorIDs: []string{"suite.missing"},
				RoleCoverage:          []internalconformance.RoleCoverage{{Role: roles.PolicyRoles[0], Cases: 0, Required: 20, Deficient: true}},
				TaskRoleCoverage:      []internalconformance.RoleCoverage{{Role: roles.TaskRoles[0], Cases: 0, Required: 20, Deficient: true}},
			}, nil
		}),
	})
	resp, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "ai-gateway", Phase: "model-eval",
	}))
	require.NoError(t, err)
	require.Equal(t, "model-eval", resp.Msg.GetAssessment().GetPhase())
	require.Equal(t, "ai-gateway", resp.Msg.GetAssessment().GetProvider())
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, resp.Msg.GetStatus())
}

func TestModelEvalDescriptorHasPhaseSpecificProviderIdentity(t *testing.T) {
	scenarioDir := aiGatewayScenarioRoot(t)
	describer, err := assessment.LoadDescriber(scenarioDir)
	require.NoError(t, err)
	response, err := describer.DescribeProvider(context.Background(), connect.NewRequest(&scenariovalidationv1.DescribeProviderRequest{Phase: "model-eval"}))
	require.NoError(t, err)
	require.Equal(t, "ai-gateway", response.Msg.GetProvider())
	require.Equal(t, "model-eval", response.Msg.GetPhase())
	require.Equal(t, "1.0.0", response.Msg.GetSpecVersion())
}

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
