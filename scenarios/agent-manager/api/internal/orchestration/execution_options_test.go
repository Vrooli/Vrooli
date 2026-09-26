package orchestration

import (
	"context"
	"reflect"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/rolepolicy"
)

func TestExecutionOptionForRunnerReportsCapabilityFactsAndClearsModelsWhenUnavailable(t *testing.T) {
	available := runner.NewMockRunner(domain.RunnerTypeCodex)
	available.SetCapabilities(runner.Capabilities{
		SupportedModels:   []string{"model-a", "model-b"},
		SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking"}, NativeObjective: true}},
		EffortMappings:    map[string]string{"high": "high"},
	})
	option := executionOptionForRunner(context.Background(), available, nil)
	if !option.Available || option.RunnerType != string(domain.RunnerTypeCodex) || !option.NativeObjective || option.DefaultModel != "model-a" || len(option.Models) != 2 || len(option.EffortLevels) != 1 {
		t.Fatalf("available option = %#v", option)
	}
	if option.DefaultModelSource != executionModelSourceLocalProbe {
		t.Fatalf("default model source = %q, want local_probe", option.DefaultModelSource)
	}

	unavailable := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	unavailable.SetAvailable(false, "resource is offline")
	unavailable.SetCapabilities(runner.Capabilities{SupportedModels: []string{"should-not-be-listed"}})
	option = executionOptionForRunner(context.Background(), unavailable, nil)
	if option.Available || len(option.Models) != 0 || option.DefaultModel != "" || option.Message != "resource is offline" {
		t.Fatalf("unavailable option = %#v", option)
	}
}

func TestExecutionOptionEffortLevelsAreDeterministic(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeClaudeCode)
	mock.SetCapabilities(runner.Capabilities{EffortMappings: map[string]string{
		"max": "max", "low": "low", "xhigh": "xhigh", "medium": "medium", "high": "high",
	}})
	option := executionOptionForRunner(context.Background(), mock, nil)
	want := []string{"low", "medium", "high", "xhigh", "max"}
	if !reflect.DeepEqual(option.EffortLevels, want) {
		t.Fatalf("effort levels = %v, want %v", option.EffortLevels, want)
	}
}

func TestExecutionOptionMergesRolePolicyAndLocalProbe(t *testing.T) {
	mock := runner.NewMockRunner(domain.RunnerTypeCodex)
	mock.SetCapabilities(runner.Capabilities{
		SupportedModels: []string{"ollama/gemma4:12b"},
		EffortMappings:  map[string]string{"high": "high"},
	})
	resolved := &rolepolicy.ResolvedCandidate{Runner: domain.RunnerTypeCodex, Model: "gpt-5.6-sol", CanonicalModel: "gpt-5.6-sol", Fallbacks: []string{"gpt-5.6-luna"}}
	option := executionOptionForRunner(context.Background(), mock, resolved)
	if option.DefaultModel != "gpt-5.6-sol" || option.DefaultModelSource != executionModelSourceRolePolicy {
		t.Fatalf("default = %q/%q, want role-policy gpt-5.6-sol", option.DefaultModel, option.DefaultModelSource)
	}
	// The role model and its fallback come first, then the locally probed model.
	if len(option.Models) != 3 {
		t.Fatalf("models = %#v, want 3 (policy + fallback + local probe)", option.Models)
	}
	if option.Models[0].Source != executionModelSourceRolePolicy || !option.Models[0].IsDefault {
		t.Fatalf("first model = %#v, want the role-policy default", option.Models[0])
	}
	if option.Models[2].ID != "ollama/gemma4:12b" || option.Models[2].Source != executionModelSourceLocalProbe {
		t.Fatalf("local probe model not merged last: %#v", option.Models[2])
	}
}
