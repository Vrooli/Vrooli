package orchestration

import (
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
)

func TestNativeObjectiveForUsesDeclaredInteractiveCapability(t *testing.T) {
	run := &domain.Run{
		RunMode: domain.RunModeSandboxed,
		ResolvedConfig: &domain.RunConfig{
			Until:         "all phases complete",
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
		},
	}
	caps := runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{
		ExecutionMode: "interactive", SandboxModes: []string{"tracking"}, NativeObjective: true,
	}}}
	if got := nativeObjectiveFor(run, caps); got != "all phases complete" {
		t.Fatalf("nativeObjectiveFor() = %q", got)
	}
	caps.SpawnCapabilities[0].NativeObjective = false
	if got := nativeObjectiveFor(run, caps); got != "" {
		t.Fatalf("nativeObjectiveFor() without declaration = %q", got)
	}
}
