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

func TestResolveDeploymentStageRejectsDirectVaultEndpointForThinClient(t *testing.T) {
	stage := NewResolveDeploymentStage()
	input := &StageInput{Config: &Config{
		DeploymentMode: DeploymentModeExternalServer,
		ScenarioName:   "secrets-manager",
		ProxyURL:       "http://127.0.0.1:8200/v1/sys/health",
	}}
	result := stage.Execute(context.Background(), input)
	if result.Status != StatusFailed {
		t.Fatalf("status = %q, want %q", result.Status, StatusFailed)
	}
	if result.Error == "" || !strings.Contains(result.Error, "cannot connect directly to a Vault endpoint") {
		t.Fatalf("error = %q, want actionable direct-Vault rejection", result.Error)
	}
	if input.ResourceDeploymentPlan != nil {
		t.Fatalf("direct Vault endpoint must not receive a resource deployment plan: %#v", input.ResourceDeploymentPlan)
	}
}

func TestResolveDeploymentStageRequiresExplicitArtifactTrustMode(t *testing.T) {
	stage := NewResolveDeploymentStage(WithResolveDeploymentScenarioRoot(t.TempDir()))
	input := &StageInput{Config: &Config{DeploymentMode: DeploymentModeBundled, ScenarioName: "demo", ResourceArtifactRoot: t.TempDir(), Platforms: []string{"linux-amd64"}}}
	result := stage.Execute(context.Background(), input)
	if result.Status != StatusFailed || !strings.Contains(result.Error, "artifact trust mode is required") {
		t.Fatalf("result = %#v, want explicit artifact trust mode rejection", result)
	}
}
