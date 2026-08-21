package orchestration

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestResolveSpawnPolicyPrefersDeclaredFeasibleCombination(t *testing.T) {
	resolution, err := ResolveSpawnPolicy(&domain.SpawnPolicy{AxisOrder: []string{"sandboxMode", "executionMode"}, SandboxMode: domain.PreferenceAxis{Prefer: []string{"protected", "tracking", "off"}}, ExecutionMode: domain.PreferenceAxis{Prefer: []string{"codec_pipe", "interactive"}}}, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "codec_pipe", SandboxModes: []string{"tracking"}}, {ExecutionMode: "interactive", SandboxModes: []string{"tracking"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "tracking" {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyNamesUnsatisfiableRequirement(t *testing.T) {
	_, err := ResolveSpawnPolicy(&domain.SpawnPolicy{AxisOrder: []string{"sandboxMode", "executionMode"}, SandboxMode: domain.PreferenceAxis{Prefer: []string{"protected"}}, ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}}, Require: []domain.SpawnCombination{{ExecutionMode: "interactive", SandboxMode: "protected"}}}, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking"}}}})
	if err == nil {
		t.Fatal("expected named unsatisfiable requirement")
	}
}
