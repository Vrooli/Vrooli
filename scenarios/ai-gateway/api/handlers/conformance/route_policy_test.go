package conformance_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"

	handler "ai-gateway/handlers/conformance"
)

func TestValidateScenarioFailsForUnreachableGatewayRole(t *testing.T) {
	repo := t.TempDir()
	taskRoles := filepath.Join(repo, "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(taskRoles), 0o755))
	require.NoError(t, os.WriteFile(taskRoles, []byte(`{"roles":{"summarize.faithful":{}}}`), 0o644))
	scenarioRoot := filepath.Join(repo, "scenarios", "demo")
	caller := filepath.Join(scenarioRoot, "api", "gateway.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(caller), 0o755))
	require.NoError(t, os.WriteFile(caller, []byte(`package api
import inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
var request = &inferencev1.RunRequest{Role: "rerank.llm_fallback"}
`), 0o644))

	h := handler.NewConnectHandler(handler.Deps{MaturitySpec: testMaturitySpec()})
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
		Path:     scenarioRoot,
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	found := false
	for _, finding := range response.Msg.GetAssessment().GetFindings() {
		if finding.GetCode() == "ai.gateway_role_unreachable" {
			found = true
		}
	}
	require.True(t, found, "Test Genie validation provider must surface undeclared Gateway roles as a blocking finding")
}

func TestValidateScenarioFailsForImplicitSharedGatewayRole(t *testing.T) {
	repo := t.TempDir()
	taskRoles := filepath.Join(repo, "scenarios", "ai-gateway", "config", "inference-role-catalog.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(taskRoles), 0o755))
	require.NoError(t, os.WriteFile(taskRoles, []byte(`{"roles":{"summarize.faithful":{}}}`), 0o644))
	scenarioRoot := filepath.Join(repo, "scenarios", "demo")
	caller := filepath.Join(scenarioRoot, "api", "reranker.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(caller), 0o755))
	require.NoError(t, os.WriteFile(caller, []byte(`package api
func makeReranker() { _ = aisearch.NewLLMRerankerWithRunner("", gatewayRunner) }
func gatewayRunner() { _ = aigateway.NewTextGenerator(resolve, client, "caller.source") }
`), 0o644))

	h := handler.NewConnectHandler(handler.Deps{MaturitySpec: testMaturitySpec()})
	response, err := h.ValidateScenario(context.Background(), connect.NewRequest(&scenariovalidationv1.ValidateScenarioRequest{
		Scenario: "demo",
		Path:     scenarioRoot,
	}))
	require.NoError(t, err)
	require.Equal(t, scenariovalidationv1.ValidationStatus_VALIDATION_STATUS_FAILED, response.Msg.GetStatus())
	found := false
	for _, finding := range response.Msg.GetAssessment().GetFindings() {
		if finding.GetCode() == "ai.gateway_role_implicit" {
			found = true
		}
	}
	require.True(t, found, "Test Genie validation provider must block implicit shared-library roles on Gateway adapters")
}
