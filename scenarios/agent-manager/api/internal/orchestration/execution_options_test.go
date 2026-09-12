package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestExecutionOptionForRunnerReportsCapabilityFactsAndClearsModelsWhenUnavailable(t *testing.T) {
	available := runner.NewMockRunner(domain.RunnerTypeCodex)
	available.SetCapabilities(runner.Capabilities{
		SupportedModels:   []string{"model-a", "model-b"},
		SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking"}, NativeObjective: true}},
		EffortMappings:    map[string]string{"high": "high"},
	})
	option := executionOptionForRunner(context.Background(), available)
	if !option.Available || option.RunnerType != string(domain.RunnerTypeCodex) || !option.NativeObjective || option.DefaultModel != "model-a" || len(option.Models) != 2 || len(option.EffortLevels) != 1 {
		t.Fatalf("available option = %#v", option)
	}

	unavailable := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	unavailable.SetAvailable(false, "resource is offline")
	unavailable.SetCapabilities(runner.Capabilities{SupportedModels: []string{"should-not-be-listed"}})
	option = executionOptionForRunner(context.Background(), unavailable)
	if option.Available || len(option.Models) != 0 || option.DefaultModel != "" || option.Message != "resource is offline" {
		t.Fatalf("unavailable option = %#v", option)
	}
}
