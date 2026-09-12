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

func TestResolveSpawnPolicyFallsBackToClaudeInteractiveTracking(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"protected", "tracking", "off"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"codec_pipe", "interactive"}},
	}
	resolution, err := ResolveSpawnPolicy(policy, runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{ExecutionMode: "interactive", SandboxModes: []string{"tracking", "off"}, NativeObjective: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "interactive" || resolution.SandboxMode != "tracking" || !resolution.NativeObjective {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyZeroCapabilitiesUsesCodecPipeDefault(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"off"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}},
	}
	resolution, err := ResolveSpawnPolicy(policy, runner.Capabilities{})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ExecutionMode != "codec_pipe" || resolution.SandboxMode != "off" {
		t.Fatalf("resolution=%+v", resolution)
	}
}

func TestResolveSpawnPolicyZeroCapabilitiesStillFailsRequiredMode(t *testing.T) {
	policy := &domain.SpawnPolicy{
		AxisOrder:     []string{"sandboxMode", "executionMode"},
		SandboxMode:   domain.PreferenceAxis{Prefer: []string{"tracking"}},
		ExecutionMode: domain.PreferenceAxis{Prefer: []string{"interactive"}},
		Require:       []domain.SpawnCombination{{ExecutionMode: "interactive", SandboxMode: "tracking"}},
	}
	if _, err := ResolveSpawnPolicy(policy, runner.Capabilities{}); err == nil {
		t.Fatal("expected required spawn mode to fail closed")
	}
}
