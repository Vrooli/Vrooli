package orchestration

import (
	"strings"
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
	caps.SpawnCapabilities[0].SandboxModes = []string{"protected"}
	if got := nativeObjectiveFor(run, caps); got != "" {
		t.Fatalf("nativeObjectiveFor() in protected mode = %q", got)
	}
}

func TestNativeObjectiveForDelegatesLongContractsWithoutTruncatingThem(t *testing.T) {
	run := &domain.Run{
		RunMode: domain.RunModeSandboxed,
		ResolvedConfig: &domain.RunConfig{
			Until:         strings.Repeat("engine-owned contract ", 500),
			SandboxConfig: &domain.SandboxConfig{Mode: domain.SandboxModeTracking},
		},
	}
	caps := runner.Capabilities{SpawnCapabilities: []runner.SpawnCapability{{
		ExecutionMode: "interactive", SandboxModes: []string{"tracking"}, NativeObjective: true,
	}}}

	got := nativeObjectiveFor(run, caps)
	if got != boundedNativeObjectiveText {
		t.Fatalf("nativeObjectiveFor() long contract = %q, want bounded delegation", got)
	}
	if len([]rune(got)) > maxNativeObjectiveCharacters {
		t.Fatalf("bounded native objective has %d characters, want <= %d", len([]rune(got)), maxNativeObjectiveCharacters)
	}
}
