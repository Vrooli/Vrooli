package pipeline

import (
	"context"
	"strings"
	"testing"
)

func TestResolveDeploymentStageLeavesResourcesServerSideForThinClient(t *testing.T) {
	stage := NewResolveDeploymentStage()
	input := &StageInput{Config: &Config{
		DeploymentMode: DeploymentModeExternalServer,
		ScenarioName:   "vault-consuming-app",
		ProxyURL:       "https://server.example/apps/vault-consuming-app/proxy/",
	}}
	result := stage.Execute(context.Background(), input)
	if result.Status != StatusSkipped {
		t.Fatalf("status = %q, want skipped", result.Status)
	}
	if input.ResourceDeploymentPlan != nil {
		t.Fatalf("thin client must not receive a local resource deployment plan: %#v", input.ResourceDeploymentPlan)
	}
	if len(result.Logs) == 0 || !strings.Contains(strings.Join(result.Logs, "\n"), "proxy") {
		t.Fatalf("logs = %#v, want thin-client proxy explanation", result.Logs)
	}
}
